package subtitle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SubRipService is the file-based implementation of Service for SubRip (.srt)
// format. Was Rust struct SubRipService.
type SubRipService struct {
	baseDir string
}

// NewSubRipService creates a SubRipService with the specified base directory.
// Was SubRipService::new.
func NewSubRipService(baseDir string) *SubRipService {
	return &SubRipService{baseDir: baseDir}
}

// validateFilename validates a filename to prevent path traversal attacks.
// Was SubRipService::validate_filename.
func validateFilename(filename string) error {
	if filename == "" {
		return NewInvalidPath("filename cannot be empty")
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return NewInvalidPath(fmt.Sprintf("invalid filename (path traversal detected): %s", filename))
	}

	return nil
}

// buildFilePath builds the full path to the subtitle file.
// Was SubRipService::build_file_path.
func (s *SubRipService) buildFilePath(filename string) string {
	return filepath.Join(s.baseDir, filename)
}

// isBlockBoundary reports whether a blank line at the given position is a block
// boundary: the next non-blank line is a valid subtitle index (u32 >= 1).
// Was SubRipService::is_block_boundary.
func isBlockBoundary(lines []string, currentPos int) bool {
	nextPos := currentPos + 1

	for nextPos < len(lines) && strings.TrimSpace(lines[nextPos]) == "" {
		nextPos++
	}

	if nextPos >= len(lines) {
		return true
	}

	nextLine := strings.TrimSpace(lines[nextPos])
	index, err := parseU32(nextLine)
	if err != nil {
		return false
	}
	return index >= 1
}

// parseU32 parses a base-10 u32, matching Rust's `str::parse::<u32>()`.
func parseU32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// createSubtitle is a helper to create a subtitle with validation, mirroring
// SubRipService::create_subtitle (including the anyhow context messages).
func createSubtitle(index uint32, startTime, endTime uint64, textLines []string, lineNum int) (Subtitle, error) {
	text := strings.Join(textLines, "\n")

	subtitleIndex, err := NewSubtitleIndex(index)
	if err != nil {
		return Subtitle{}, fmt.Errorf("line %d: invalid index value %d: %w", lineNum, index, err)
	}

	subtitleStart, err := NewSubtitleTimestamp(time.Duration(int64(startTime)) * time.Millisecond)
	if err != nil {
		return Subtitle{}, fmt.Errorf("line %d: invalid start timestamp: %w", lineNum, err)
	}

	subtitleEnd, err := NewSubtitleTimestamp(time.Duration(int64(endTime)) * time.Millisecond)
	if err != nil {
		return Subtitle{}, fmt.Errorf("line %d: invalid end timestamp: %w", lineNum, err)
	}

	subtitleText, err := NewSubtitleText(text)
	if err != nil {
		return Subtitle{}, fmt.Errorf("line %d: invalid subtitle text (empty or whitespace-only): %w", lineNum, err)
	}

	sub, err := NewSubtitle(subtitleIndex, subtitleStart, subtitleEnd, subtitleText)
	if err != nil {
		return Subtitle{}, fmt.Errorf("line %d: failed to create subtitle: %w", lineNum, err)
	}
	return sub, nil
}

// parserState mirrors the inner Rust enum ParserState.
type parserState int

const (
	stateExpectingIndex parserState = iota
	stateExpectingTimestamp
	stateReadingText
)

// parseSrtFile parses SRT file content into a slice of subtitles.
// Was SubRipService::parse_srt_file.
func parseSrtFile(content string) ([]Subtitle, error) {
	var subtitles []Subtitle
	lines := splitLines(content)
	state := stateExpectingIndex

	var currentIndex uint32
	var currentStart uint64
	var currentEnd uint64
	var textLines []string
	var lineNum int

	for i, line := range lines {
		lineNum = i + 1

		switch state {
		case stateExpectingIndex:
			if strings.TrimSpace(line) == "" {
				continue
			}

			idx, err := parseU32(strings.TrimSpace(line))
			if err != nil {
				return nil, fmt.Errorf("line %d: expected subtitle index, found '%s': %w", lineNum, strings.TrimSpace(line), err)
			}
			currentIndex = idx

			state = stateExpectingTimestamp

		case stateExpectingTimestamp:
			parts := strings.Split(line, " --> ")
			if len(parts) != 2 {
				return nil, fmt.Errorf(
					"line %d: invalid timestamp format '%s' (expected 'HH:MM:SS,mmm --> HH:MM:SS,mmm')",
					lineNum, line,
				)
			}

			startDur, err := ParseTimestamp(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("line %d: failed to parse start timestamp '%s': %w", lineNum, parts[0], err)
			}
			currentStart = uint64(int64(startDur / time.Millisecond))

			endDur, err := ParseTimestamp(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("line %d: failed to parse end timestamp '%s': %w", lineNum, parts[1], err)
			}
			currentEnd = uint64(int64(endDur / time.Millisecond))

			textLines = textLines[:0]
			state = stateReadingText

		case stateReadingText:
			// check if this blank line is a block boundary
			if strings.TrimSpace(line) == "" && isBlockBoundary(lines, i) {
				// finalize current subtitle
				subtitle, err := createSubtitle(currentIndex, currentStart, currentEnd, textLines, lineNum)
				if err != nil {
					return nil, err
				}
				subtitles = append(subtitles, subtitle)
				state = stateExpectingIndex
			} else {
				// add line to text (including blank lines within text)
				textLines = append(textLines, line)
			}
		}
	}

	// handle last subtitle if file doesn't end with blank line
	if state == stateReadingText {
		if len(textLines) == 0 {
			return nil, fmt.Errorf("line %d: incomplete subtitle (missing text)", len(lines))
		}
		subtitle, err := createSubtitle(currentIndex, currentStart, currentEnd, textLines, len(lines))
		if err != nil {
			return nil, err
		}
		subtitles = append(subtitles, subtitle)
	}

	return subtitles, nil
}

// serializeToSrt serializes subtitles to SRT format.
// Was SubRipService::serialize_to_srt.
func serializeToSrt(subtitles []Subtitle) string {
	parts := make([]string, 0, len(subtitles))
	for _, s := range subtitles {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, "\n\n")
}

// GetByID implements Service.
func (s *SubRipService) GetByID(filename string, id uint32) (*Subtitle, error) {
	if err := validateFilename(filename); err != nil {
		return nil, err
	}

	filePath := s.buildFilePath(filename)

	if !pathExists(filePath) {
		return nil, NewFileNotFound(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, NewIoError(err)
	}

	subtitles, err := parseSrtFile(string(content))
	if err != nil {
		return nil, NewParseError(err)
	}

	for i := range subtitles {
		if subtitles[i].Index.Value() == id {
			found := subtitles[i]
			return &found, nil
		}
	}

	return nil, nil
}

// GetAll implements Service.
func (s *SubRipService) GetAll(filename string) ([]Subtitle, error) {
	if err := validateFilename(filename); err != nil {
		return nil, err
	}

	filePath := s.buildFilePath(filename)

	if !pathExists(filePath) {
		return nil, NewFileNotFound(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, NewIoError(err)
	}

	subtitles, err := parseSrtFile(string(content))
	if err != nil {
		return nil, NewParseError(err)
	}
	return subtitles, nil
}

// Set implements Service.
func (s *SubRipService) Set(filename string, id uint32, update SubtitleUpdate) (UpdateReport, error) {
	if !update.HasUpdates() {
		return UpdateReport{}, NewNoFieldsToUpdate()
	}

	if err := validateFilename(filename); err != nil {
		return UpdateReport{}, err
	}

	filePath := s.buildFilePath(filename)
	if !pathExists(filePath) {
		return UpdateReport{}, NewFileNotFound(filePath)
	}

	subtitles, err := s.GetAll(filename)
	if err != nil {
		return UpdateReport{}, err
	}

	subtitlePos := -1
	for i := range subtitles {
		if subtitles[i].Index.Value() == id {
			subtitlePos = i
			break
		}
	}
	if subtitlePos < 0 {
		return UpdateReport{}, NewSubtitleNotFound(id)
	}

	updatedSubtitle, err := update.ApplyTo(subtitles[subtitlePos])
	if err != nil {
		return UpdateReport{}, NewParseError(err)
	}

	subtitles[subtitlePos] = updatedSubtitle

	content := serializeToSrt(subtitles)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return UpdateReport{}, NewWriteFailed(err.Error())
	}

	var fieldsUpdated []string
	if update.StartTime != nil {
		fieldsUpdated = append(fieldsUpdated, "start_time")
	}
	if update.EndTime != nil {
		fieldsUpdated = append(fieldsUpdated, "end_time")
	}
	if update.Text != nil {
		fieldsUpdated = append(fieldsUpdated, "text")
	}

	return UpdateReport{
		FilePath:      filePath,
		SubtitleIndex: id,
		FieldsUpdated: fieldsUpdated,
	}, nil
}

// Add implements Service.
func (s *SubRipService) Add(filename string, startTime, endTime SubtitleTimestamp, text SubtitleText) (AddReport, error) {
	slog.Info(fmt.Sprintf("adding subtitle '%s' to file: %s", text.Value(), filename))
	slog.Debug(fmt.Sprintf("subtitle timestamps: %s --> %s",
		FormatTimestamp(startTime.Value()), FormatTimestamp(endTime.Value())))
	slog.Debug(fmt.Sprintf("subtitle text length: %d", len(text.Value())))

	slog.Debug("validating filename")
	if err := validateFilename(filename); err != nil {
		return AddReport{}, err
	}

	filePath := s.buildFilePath(filename)
	slog.Debug(fmt.Sprintf("checking file existence: %q", filePath))

	var subtitles []Subtitle
	if pathExists(filePath) {
		slog.Debug("reading existing subtitles")
		subs, err := s.GetAll(filename)
		if err != nil {
			return AddReport{}, err
		}
		slog.Debug(fmt.Sprintf("found %d existing subtitles", len(subs)))
		subtitles = subs
	} else {
		slog.Info(fmt.Sprintf("file does not exist, will create new file: %q", filePath))
		subtitles = nil
	}

	var maxIndex uint32
	for i := range subtitles {
		if v := subtitles[i].Index.Value(); v > maxIndex {
			maxIndex = v
		}
	}
	newIndex := maxIndex + 1
	slog.Debug(fmt.Sprintf("calculated new index: %d", newIndex))

	if len(subtitles) > 0 {
		slog.Debug("validating timestamp against existing subtitles")
		maxEndTime := subtitles[0].EndTime.Value()
		for i := range subtitles {
			if v := subtitles[i].EndTime.Value(); v > maxEndTime {
				maxEndTime = v
			}
		}
		slog.Debug(fmt.Sprintf("last subtitle ends at: %s", FormatTimestamp(maxEndTime)))

		if startTime.Value() < maxEndTime {
			slog.Debug("timestamp conflict detected")
			return AddReport{}, NewTimestampConflict(
				FormatTimestamp(maxEndTime),
				FormatTimestamp(startTime.Value()),
			)
		}
		slog.Debug("timestamp validation passed")
	}

	slog.Debug(fmt.Sprintf("creating new subtitle with index %d", newIndex))
	subtitleIndex, err := NewSubtitleIndex(newIndex)
	if err != nil {
		return AddReport{}, NewParseError(fmt.Errorf("invalid index: %s", err))
	}

	newSubtitle, err := NewSubtitle(subtitleIndex, startTime, endTime, text)
	if err != nil {
		return AddReport{}, NewParseError(err)
	}

	subtitles = append(subtitles, newSubtitle)
	slog.Debug(fmt.Sprintf("added subtitle to collection, total count: %d", len(subtitles)))

	slog.Debug("serializing subtitles to srt format")
	content := serializeToSrt(subtitles)

	slog.Debug(fmt.Sprintf("writing file: %q", filePath))
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return AddReport{}, NewWriteFailed(err.Error())
	}

	slog.Info(fmt.Sprintf("subtitle added successfully with index %d", newIndex))
	return AddReport{
		FilePath:       filePath,
		NewIndex:       newIndex,
		TotalSubtitles: len(subtitles),
	}, nil
}

// WriteAll implements Service.
func (s *SubRipService) WriteAll(filename string, subtitles []Subtitle) error {
	slog.Debug(fmt.Sprintf("writing %d subtitles to %s", len(subtitles), filename))

	if err := validateFilename(filename); err != nil {
		return err
	}
	filePath := s.buildFilePath(filename)

	var content strings.Builder
	for i := range subtitles {
		content.WriteString(subtitles[i].String())
		content.WriteString("\n\n")
	}

	slog.Debug(fmt.Sprintf("writing file: %q", filePath))
	if err := os.WriteFile(filePath, []byte(trimEnd(content.String())), 0o644); err != nil {
		return NewWriteFailed(fmt.Sprintf("failed to write file: %s", err))
	}

	slog.Info(fmt.Sprintf("wrote %d subtitles to %s", len(subtitles), filename))
	return nil
}

// pathExists reports whether the path exists (mirrors Path::exists()).
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// trimEnd strips trailing Unicode whitespace, mirroring Rust's str::trim_end().
func trimEnd(s string) string {
	return strings.TrimRightFunc(s, func(r rune) bool {
		return isRustWhitespace(r)
	})
}

// isRustWhitespace reports whether r is whitespace per Rust's char::is_whitespace
// (Unicode White_Space). Go's unicode.IsSpace matches the same set for the
// characters that appear in subtitle files.
func isRustWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	switch r {
	case 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006,
		0x2007, 0x2008, 0x2009, 0x200A, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
		return true
	}
	return false
}
