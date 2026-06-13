package doctor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// SubRipDoctorService is the SubRip (.srt) doctor service implementation.
// Was Rust struct SubRipDoctorService.
type SubRipDoctorService struct {
	baseDir string
}

// parserState is the parser state machine for line-by-line analysis.
// Was Rust enum ParserState.
type parserState int

const (
	expectingTimestamp parserState = iota
	expectingText
	betweenBlocks
)

// NewSubRipDoctorService creates a new SubRip doctor service.
//
// baseDir is the base directory for resolving relative file paths.
// Was SubRipDoctorService::new.
func NewSubRipDoctorService(baseDir string) *SubRipDoctorService {
	return &SubRipDoctorService{baseDir: baseDir}
}

// validateFilename validates a filename for path traversal attacks.
// Was SubRipDoctorService::validate_filename.
func (s *SubRipDoctorService) validateFilename(filename string) error {
	if filename == "" {
		return NewInvalidPath("filename cannot be empty")
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return NewInvalidPath(fmt.Sprintf("invalid filename (path traversal detected): %s", filename))
	}

	return nil
}

// getFilePath gets the full path to the file.
// Was SubRipDoctorService::get_file_path.
func (s *SubRipDoctorService) getFilePath(filename string) (string, error) {
	if err := s.validateFilename(filename); err != nil {
		return "", err
	}

	path := filepath.Join(s.baseDir, filename)

	if !pathExists(path) {
		return "", NewFileNotFound(path)
	}

	return path, nil
}

// parseTimestamp parses a timestamp in HH:MM:SS,mmm format.
// Was SubRipDoctorService::parse_timestamp. On error it returns the same
// message strings as the Rust impl.
func (s *SubRipDoctorService) parseTimestamp(timestampStr string) (uint64, error) {
	parts := splitOnSeparators(timestampStr, ':', ',')
	if len(parts) != 4 {
		return 0, fmt.Errorf("expected format HH:MM:SS,mmm, got %s", timestampStr)
	}

	hours, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid hours: %s", parts[0])
	}
	minutes, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %s", parts[1])
	}
	seconds, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %s", parts[2])
	}
	millis, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid milliseconds: %s", parts[3])
	}

	if minutes >= 60 {
		return 0, fmt.Errorf("minutes must be < 60, got %d", minutes)
	}
	if seconds >= 60 {
		return 0, fmt.Errorf("seconds must be < 60, got %d", seconds)
	}
	if millis >= 1000 {
		return 0, fmt.Errorf("milliseconds must be < 1000, got %d", millis)
	}

	totalMs := hours*3600000 + minutes*60000 + seconds*1000 + millis
	return totalMs, nil
}

// Diagnose diagnoses a subtitle file for issues.
// Was SubRipDoctorService::diagnose.
func (s *SubRipDoctorService) Diagnose(filename string) (DiagnosticReport, error) {
	slog.Info(fmt.Sprintf("diagnostics for file '%s'", filename))

	filePath, err := s.getFilePath(filename)
	if err != nil {
		return DiagnosticReport{}, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return DiagnosticReport{}, NewIoError(err)
	}

	lines := splitLines(string(content))
	totalLines := len(lines)

	slog.Debug(fmt.Sprintf("file contain lines: %d", totalLines))

	var issues []ValidationIssue
	state := betweenBlocks
	currentBlock := 0
	emptyLineCount := 0
	textCollected := false
	consecutiveEmptyStart := 0

	for lineNum, line := range lines {
		lineNumber := lineNum + 1 // 1-based
		isEmpty := rustTrim(line) == ""

		switch state {
		case betweenBlocks:
			if isEmpty {
				if emptyLineCount == 0 {
					consecutiveEmptyStart = lineNumber
				}
				emptyLineCount++
			} else {
				if emptyLineCount > 1 {
					ctx := fmt.Sprintf("lines %d-%d", consecutiveEmptyStart, consecutiveEmptyStart+emptyLineCount-1)
					issues = append(issues, ValidationIssue{
						LineNumber:  consecutiveEmptyStart,
						BlockNumber: nil,
						IssueType: IssueType{
							Kind:  MultipleEmptyLines,
							Count: emptyLineCount,
						},
						Severity: Warning,
						Context:  &ctx,
					})
				}
				emptyLineCount = 0
				currentBlock++
				textCollected = false

				blockNum := currentBlock
				idx, parseErr := parseU32(rustTrim(line))
				if parseErr != nil {
					issues = append(issues, ValidationIssue{
						LineNumber:  lineNumber,
						BlockNumber: &blockNum,
						IssueType: IssueType{
							Kind:   InvalidIndex,
							Value:  line,
							Reason: "not a valid number",
						},
						Severity: Error,
						Context:  nil,
					})
				} else if idx < 1 {
					issues = append(issues, ValidationIssue{
						LineNumber:  lineNumber,
						BlockNumber: &blockNum,
						IssueType: IssueType{
							Kind:   InvalidIndex,
							Value:  line,
							Reason: "must be >= 1",
						},
						Severity: Error,
						Context:  nil,
					})
				}
				state = expectingTimestamp
			}
		case expectingTimestamp:
			if isEmpty {
				blockNum := currentBlock
				ctx := "expected timestamp line"
				issues = append(issues, ValidationIssue{
					LineNumber:  lineNumber,
					BlockNumber: &blockNum,
					IssueType: IssueType{
						Kind: EmptyLineInBlock,
					},
					Severity: Error,
					Context:  &ctx,
				})
				state = betweenBlocks
			} else {
				if !strings.Contains(line, " --> ") {
					blockNum := currentBlock
					issues = append(issues, ValidationIssue{
						LineNumber:  lineNumber,
						BlockNumber: &blockNum,
						IssueType: IssueType{
							Kind:   MalformedTimestamp,
							Value:  line,
							Reason: "missing ' --> ' separator",
						},
						Severity: Error,
						Context:  nil,
					})
				} else {
					parts := strings.Split(line, " --> ")
					if len(parts) == 2 {
						startMs, startErr := s.parseTimestamp(parts[0])
						endMs, endErr := s.parseTimestamp(parts[1])
						switch {
						case startErr == nil && endErr == nil:
							if endMs <= startMs {
								blockNum := currentBlock
								issues = append(issues, ValidationIssue{
									LineNumber:  lineNumber,
									BlockNumber: &blockNum,
									IssueType: IssueType{
										Kind:  InvalidTimestamp,
										Start: parts[0],
										End:   parts[1],
									},
									Severity: Error,
									Context:  nil,
								})
							}
						case startErr != nil:
							blockNum := currentBlock
							issues = append(issues, ValidationIssue{
								LineNumber:  lineNumber,
								BlockNumber: &blockNum,
								IssueType: IssueType{
									Kind:   MalformedTimestamp,
									Value:  parts[0],
									Reason: startErr.Error(),
								},
								Severity: Error,
								Context:  nil,
							})
						default:
							blockNum := currentBlock
							issues = append(issues, ValidationIssue{
								LineNumber:  lineNumber,
								BlockNumber: &blockNum,
								IssueType: IssueType{
									Kind:   MalformedTimestamp,
									Value:  parts[1],
									Reason: endErr.Error(),
								},
								Severity: Error,
								Context:  nil,
							})
						}
					}
				}
				state = expectingText
			}
		case expectingText:
			if isEmpty {
				if !textCollected {
					blockNum := currentBlock
					ctx := "no text before empty line"
					issues = append(issues, ValidationIssue{
						LineNumber:  lineNumber,
						BlockNumber: &blockNum,
						IssueType: IssueType{
							Kind: EmptyText,
						},
						Severity: Error,
						Context:  &ctx,
					})
				}
				state = betweenBlocks
				emptyLineCount = 0
			} else {
				textCollected = true
			}
		}
	}

	if state == betweenBlocks && emptyLineCount > 1 {
		ctx := "trailing empty lines"
		issues = append(issues, ValidationIssue{
			LineNumber:  consecutiveEmptyStart,
			BlockNumber: nil,
			IssueType: IssueType{
				Kind:  MultipleEmptyLines,
				Count: emptyLineCount,
			},
			Severity: Warning,
			Context:  &ctx,
		})
	}

	slog.Debug(fmt.Sprintf("detected problematic lines: %d", len(issues)))

	subtitleService := subtitle.NewSubRipService(s.baseDir)
	_, getAllErr := subtitleService.GetAll(filename)
	isParsable := getAllErr == nil

	errorCount := 0
	warningCount := 0
	for _, i := range issues {
		if i.Severity == Error {
			errorCount++
		}
		if i.Severity == Warning {
			warningCount++
		}
	}
	slog.Info(fmt.Sprintf("diagnostics completed: found %d errors, %d warnings", errorCount, warningCount))

	return DiagnosticReport{
		FilePath:    filePath,
		TotalLines:  totalLines,
		TotalBlocks: currentBlock,
		Issues:      issues,
		IsParsable:  isParsable,
	}, nil
}

// Fix fixes issues in a subtitle file.
// Was SubRipDoctorService::fix.
func (s *SubRipDoctorService) Fix(filename string) (FixReport, error) {
	slog.Info(fmt.Sprintf("starting file fix: %s", filename))

	filePath, err := s.getFilePath(filename)
	if err != nil {
		return FixReport{}, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return FixReport{}, NewIoError(err)
	}

	lines := splitLines(string(content))

	var outputLines []string
	state := betweenBlocks
	var currentBlockLines []string
	emptyLineCount := 0
	issuesFixed := 0
	var unfixableIssues []ValidationIssue

	for _, line := range lines {
		isEmpty := rustTrim(line) == ""

		switch state {
		case betweenBlocks:
			if isEmpty {
				emptyLineCount++
			} else {
				if len(outputLines) != 0 {
					if emptyLineCount > 1 {
						issuesFixed++
					}
					outputLines = append(outputLines, "")
				}
				emptyLineCount = 0
				currentBlockLines = currentBlockLines[:0]
				currentBlockLines = append(currentBlockLines, line)
				state = expectingTimestamp
			}
		case expectingTimestamp:
			if isEmpty {
				issuesFixed++
			} else {
				currentBlockLines = append(currentBlockLines, line)
				state = expectingText
			}
		case expectingText:
			if isEmpty {
				// End of block - write collected lines
				outputLines = append(outputLines, currentBlockLines...)
				state = betweenBlocks
				emptyLineCount = 0
			} else {
				currentBlockLines = append(currentBlockLines, line)
			}
		}
	}

	if len(currentBlockLines) != 0 {
		outputLines = append(outputLines, currentBlockLines...)
	}

	fixedContent := strings.Join(outputLines, "\n")
	if err := os.WriteFile(filePath, []byte(fixedContent), 0o644); err != nil {
		return FixReport{}, NewIoError(err)
	}

	slog.Debug("fixed file written")

	subtitleService := subtitle.NewSubRipService(s.baseDir)
	_, getAllErr := subtitleService.GetAll(filename)
	validationSuccess := getAllErr == nil

	if validationSuccess {
		slog.Info("validation successful: file can be parsed")
	} else {
		slog.Error("validation failed: file still contains errors")
	}

	slog.Info(fmt.Sprintf("fix completed: %d issues fixed", issuesFixed))

	return FixReport{
		OriginalPath:      filePath,
		FixedPath:         filePath,
		IssuesFixed:       issuesFixed,
		IssuesUnfixable:   len(unfixableIssues),
		UnfixableIssues:   unfixableIssues,
		ValidationSuccess: validationSuccess,
	}, nil
}

// splitOnSeparators splits s on any of the given separator runes, mirroring
// Rust's `str::split(&[char])`. Consecutive separators yield empty fields and
// leading/trailing separators yield empty fields too.
func splitOnSeparators(s string, seps ...rune) []string {
	isSep := func(r rune) bool {
		for _, sep := range seps {
			if r == sep {
				return true
			}
		}
		return false
	}

	var parts []string
	start := 0
	for i, r := range s {
		if isSep(r) {
			parts = append(parts, s[start:i])
			start = i + len(string(r))
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseU32 parses a base-10 u32, matching Rust's `str::parse::<u32>()`.
func parseU32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// splitLines mirrors Rust's str::lines(): splits on '\n', strips an optional
// trailing '\r' from each line, and yields no trailing empty element for a
// terminating '\n'. An empty string yields no lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}

// rustTrim strips leading and trailing Unicode whitespace, mirroring Rust's
// str::trim().
func rustTrim(s string) string {
	return strings.TrimFunc(s, isRustWhitespace)
}

// pathExists reports whether the path exists (mirrors Path::exists()).
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isRustWhitespace reports whether r is whitespace per Rust's char::is_whitespace
// (Unicode White_Space).
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
