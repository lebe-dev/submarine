// Package subtitle contains the core domain models and services for the
// submarine subtitle translation toolkit. It is a 1-to-1 port of the Rust
// crate `lib::subtitle`.
package subtitle

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrorKind discriminates the variants of SubtitleError. It mirrors the
// variants of the Rust `enum SubtitleError`.
type ErrorKind int

const (
	// ErrFileNotFound -> SubtitleError::FileNotFound
	ErrFileNotFound ErrorKind = iota
	// ErrIO -> SubtitleError::IoError
	ErrIO
	// ErrParse -> SubtitleError::ParseError
	ErrParse
	// ErrInvalidPath -> SubtitleError::InvalidPath
	ErrInvalidPath
	// ErrSubtitleNotFound -> SubtitleError::SubtitleNotFound
	ErrSubtitleNotFound
	// ErrNoFieldsToUpdate -> SubtitleError::NoFieldsToUpdate
	ErrNoFieldsToUpdate
	// ErrBackupFailed -> SubtitleError::BackupFailed
	ErrBackupFailed
	// ErrWriteFailed -> SubtitleError::WriteFailed
	ErrWriteFailed
	// ErrTimestampConflict -> SubtitleError::TimestampConflict
	ErrTimestampConflict
	// ErrCSVParse -> SubtitleError::CsvParseError
	ErrCSVParse
	// ErrInvalidCSVHeader -> SubtitleError::InvalidCsvHeader
	ErrInvalidCSVHeader
	// ErrTimestampOverlap -> SubtitleError::TimestampOverlap
	ErrTimestampOverlap
	// ErrAnchoredParse -> SubtitleError::AnchoredParseError
	ErrAnchoredParse
	// ErrReferenceIndexNotFound -> SubtitleError::ReferenceIndexNotFound
	ErrReferenceIndexNotFound
	// ErrInvalidAnchoredFormat -> SubtitleError::InvalidAnchoredFormat
	ErrInvalidAnchoredFormat
)

// SubtitleError mirrors Rust enum SubtitleError. Msg holds the formatted
// message (identical text to the Rust #[error(...)] attribute). Carry only the
// fields a variant needs; zero-value the rest.
type SubtitleError struct {
	Kind    ErrorKind
	Msg     string
	Wrapped error // for IoError / ParseError sources
}

// Error returns the formatted error message (matching the Rust #[error(...)]).
func (e *SubtitleError) Error() string { return e.Msg }

// Unwrap returns the wrapped source error (IoError / ParseError).
func (e *SubtitleError) Unwrap() error { return e.Wrapped }

// NewFileNotFound -> SubtitleError::FileNotFound(p)
func NewFileNotFound(p string) *SubtitleError {
	return &SubtitleError{Kind: ErrFileNotFound, Msg: fmt.Sprintf("Subtitle file not found: %s", p)}
}

// NewIoError -> SubtitleError::IoError(err)
func NewIoError(err error) *SubtitleError {
	return &SubtitleError{Kind: ErrIO, Msg: fmt.Sprintf("I/O error reading subtitle file: %s", err), Wrapped: err}
}

// NewParseError -> SubtitleError::ParseError(err)
func NewParseError(err error) *SubtitleError {
	return &SubtitleError{Kind: ErrParse, Msg: fmt.Sprintf("Failed to parse subtitle file: %s", err), Wrapped: err}
}

// NewInvalidPath -> SubtitleError::InvalidPath(p)
func NewInvalidPath(p string) *SubtitleError {
	return &SubtitleError{Kind: ErrInvalidPath, Msg: fmt.Sprintf("Invalid file path: %s", p)}
}

// NewSubtitleNotFound -> SubtitleError::SubtitleNotFound(idx)
func NewSubtitleNotFound(idx uint32) *SubtitleError {
	return &SubtitleError{Kind: ErrSubtitleNotFound, Msg: fmt.Sprintf("Subtitle with index %d not found in file", idx)}
}

// NewNoFieldsToUpdate -> SubtitleError::NoFieldsToUpdate
func NewNoFieldsToUpdate() *SubtitleError {
	return &SubtitleError{Kind: ErrNoFieldsToUpdate, Msg: "No fields specified for update (at least one of --start, --end, or --text required)"}
}

// NewBackupFailed -> SubtitleError::BackupFailed(s)
func NewBackupFailed(s string) *SubtitleError {
	return &SubtitleError{Kind: ErrBackupFailed, Msg: fmt.Sprintf("Failed to create backup: %s", s)}
}

// NewWriteFailed -> SubtitleError::WriteFailed(s)
func NewWriteFailed(s string) *SubtitleError {
	return &SubtitleError{Kind: ErrWriteFailed, Msg: fmt.Sprintf("Failed to write updated file: %s", s)}
}

// NewTimestampConflict -> SubtitleError::TimestampConflict { last_end, new_start }
func NewTimestampConflict(lastEnd, newStart string) *SubtitleError {
	return &SubtitleError{
		Kind: ErrTimestampConflict,
		Msg: fmt.Sprintf(
			"Timestamp conflict: new subtitle (starts at %s) must start at or after the last subtitle ends (at %s)",
			newStart, lastEnd,
		),
	}
}

// NewCsvParseError -> SubtitleError::CsvParseError { line, message }
func NewCsvParseError(line int, message string) *SubtitleError {
	return &SubtitleError{Kind: ErrCSVParse, Msg: fmt.Sprintf("CSV parsing error at line %d: %s", line, message)}
}

// NewInvalidCsvHeader -> SubtitleError::InvalidCsvHeader(delimiter, got)
func NewInvalidCsvHeader(delimiter, got string) *SubtitleError {
	return &SubtitleError{
		Kind: ErrInvalidCSVHeader,
		Msg:  fmt.Sprintf("Invalid CSV header: expected 'start_time%send_time%stext', got '%s'", delimiter, delimiter, got),
	}
}

// NewTimestampOverlap -> SubtitleError::TimestampOverlap { line, existing_end, new_start }
func NewTimestampOverlap(line int, existingEnd, newStart string) *SubtitleError {
	return &SubtitleError{
		Kind: ErrTimestampOverlap,
		Msg: fmt.Sprintf(
			"Timestamp overlap detected: subtitle at CSV line %d (starts at %s) overlaps with existing subtitle (ends at %s)",
			line, newStart, existingEnd,
		),
	}
}

// NewAnchoredParseError -> SubtitleError::AnchoredParseError { line, message }
func NewAnchoredParseError(line int, message string) *SubtitleError {
	return &SubtitleError{Kind: ErrAnchoredParse, Msg: fmt.Sprintf("Anchored format parsing error at line %d: %s", line, message)}
}

// NewReferenceIndexNotFound -> SubtitleError::ReferenceIndexNotFound { index }
func NewReferenceIndexNotFound(index uint32) *SubtitleError {
	return &SubtitleError{Kind: ErrReferenceIndexNotFound, Msg: fmt.Sprintf("Reference file does not contain subtitle with index %d", index)}
}

// NewInvalidAnchoredFormat -> SubtitleError::InvalidAnchoredFormat { line, message }
func NewInvalidAnchoredFormat(line int, message string) *SubtitleError {
	return &SubtitleError{Kind: ErrInvalidAnchoredFormat, Msg: fmt.Sprintf("Invalid anchored format at line %d: %s", line, message)}
}

// --- validated newtypes (was nutype) ---

// SubtitleIndex is a validated subtitle index (>= 1). Was nutype SubtitleIndex(u32).
type SubtitleIndex struct{ v uint32 }

// NewSubtitleIndex validates that v >= 1 (nutype: greater_or_equal = 1).
func NewSubtitleIndex(v uint32) (SubtitleIndex, error) {
	if v < 1 {
		return SubtitleIndex{}, fmt.Errorf("SubtitleIndex: value (%d) is too small. The value must be greater or equal to 1", v)
	}
	return SubtitleIndex{v: v}, nil
}

// Value returns the underlying index value.
func (i SubtitleIndex) Value() uint32 { return i.v }

// String formats the index as "%d" (Rust Display).
func (i SubtitleIndex) String() string { return strconv.FormatUint(uint64(i.v), 10) }

// SubtitleTimestamp is a validated timestamp (>= 0). Was nutype
// SubtitleTimestamp(Duration). Stored at millisecond resolution.
type SubtitleTimestamp struct{ v time.Duration }

// NewSubtitleTimestamp validates that d >= 0 (nutype: predicate d >= Duration::zero()).
func NewSubtitleTimestamp(d time.Duration) (SubtitleTimestamp, error) {
	if d < 0 {
		return SubtitleTimestamp{}, fmt.Errorf("SubtitleTimestamp: failed the predicate test")
	}
	return SubtitleTimestamp{v: d}, nil
}

// Value returns the underlying duration.
func (t SubtitleTimestamp) Value() time.Duration { return t.v }

// Millis returns the duration in whole milliseconds (chrono num_milliseconds).
func (t SubtitleTimestamp) Millis() int64 { return int64(t.v / time.Millisecond) }

// SubtitleText is sanitized (trimmed) and validated (non-empty) subtitle text.
// Was nutype SubtitleText(String) with sanitize(trim), validate(not_empty).
type SubtitleText struct{ v string }

// NewSubtitleText sanitizes (trims) then validates (non-empty) the text.
func NewSubtitleText(s string) (SubtitleText, error) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return SubtitleText{}, fmt.Errorf("SubtitleText: value is empty")
	}
	return SubtitleText{v: trimmed}, nil
}

// Value returns the sanitized text.
func (t SubtitleText) Value() string { return t.v }

// String returns the sanitized text (Rust Display).
func (t SubtitleText) String() string { return t.v }

// --- main entity ---

// Subtitle is the main domain entity. Was Rust struct Subtitle.
type Subtitle struct {
	Index     SubtitleIndex
	StartTime SubtitleTimestamp
	EndTime   SubtitleTimestamp
	Text      SubtitleText
}

// NewSubtitle creates a Subtitle with cross-field validation (end > start).
// Was Subtitle::new.
func NewSubtitle(index SubtitleIndex, start, end SubtitleTimestamp, text SubtitleText) (Subtitle, error) {
	if end.v <= start.v {
		return Subtitle{}, fmt.Errorf(
			"end time must be after start time (start: %dms, end: %dms)",
			start.Millis(), end.Millis(),
		)
	}
	return Subtitle{
		Index:     index,
		StartTime: start,
		EndTime:   end,
		Text:      text,
	}, nil
}

// ParseTimestamp parses SRT timestamp format "HH:MM:SS,mmm" to a Duration.
// Was Subtitle::parse_timestamp.
func ParseTimestamp(s string) (time.Duration, error) {
	// Split by comma to separate seconds from milliseconds
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, fmt.Errorf("Invalid timestamp format '%s': expected HH:MM:SS,mmm", s)
	}

	// Parse time part (HH:MM:SS)
	timeParts := strings.Split(parts[0], ":")
	if len(timeParts) != 3 {
		return 0, fmt.Errorf("Invalid time format '%s': expected HH:MM:SS", parts[0])
	}

	hours, err := parseI64(timeParts[0])
	if err != nil {
		return 0, fmt.Errorf("Failed to parse hours from '%s': %w", s, err)
	}
	minutes, err := parseI64(timeParts[1])
	if err != nil {
		return 0, fmt.Errorf("Failed to parse minutes from '%s': %w", s, err)
	}
	seconds, err := parseI64(timeParts[2])
	if err != nil {
		return 0, fmt.Errorf("Failed to parse seconds from '%s': %w", s, err)
	}
	milliseconds, err := parseI64(parts[1])
	if err != nil {
		return 0, fmt.Errorf("Failed to parse milliseconds from '%s': %w", s, err)
	}

	// Validate ranges
	if minutes >= 60 {
		return 0, fmt.Errorf("Minutes must be < 60, got %d in '%s'", minutes, s)
	}
	if seconds >= 60 {
		return 0, fmt.Errorf("Seconds must be < 60, got %d in '%s'", seconds, s)
	}
	if milliseconds >= 1000 {
		return 0, fmt.Errorf("Milliseconds must be < 1000, got %d in '%s'", milliseconds, s)
	}

	// Calculate total duration
	totalMs := hours*3_600_000 + minutes*60_000 + seconds*1_000 + milliseconds

	return time.Duration(totalMs) * time.Millisecond, nil
}

// parseI64 parses a base-10 i64, matching Rust's `str::parse::<i64>()` which
// rejects leading/trailing whitespace and non-digit characters.
func parseI64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// FormatTimestamp formats a Duration to SRT timestamp format "HH:MM:SS,mmm".
// Was Subtitle::format_timestamp.
func FormatTimestamp(d time.Duration) string {
	totalMs := int64(d / time.Millisecond)

	hours := totalMs / 3_600_000
	minutes := (totalMs % 3_600_000) / 60_000
	seconds := (totalMs % 60_000) / 1_000
	milliseconds := totalMs % 1_000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

// Duration returns end - start. Was Subtitle::duration.
func (s Subtitle) Duration() time.Duration {
	return s.EndTime.v - s.StartTime.v
}

// HasHTMLTags reports whether the text contains both '<' and '>'.
// Was Subtitle::has_html_tags.
func (s Subtitle) HasHTMLTags() bool {
	return strings.ContainsRune(s.Text.v, '<') && strings.ContainsRune(s.Text.v, '>')
}

// TextWithoutHTML returns the text with HTML tags stripped (simple strip, not
// HTML parsing). Was Subtitle::text_without_html.
func (s Subtitle) TextWithoutHTML() string {
	var result strings.Builder
	inTag := false

	for _, ch := range s.Text.v {
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
		case !inTag:
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// LineCount returns the number of lines in the subtitle text.
// Was Subtitle::line_count (Rust str::lines().count()).
func (s Subtitle) LineCount() int {
	return len(splitLines(s.Text.v))
}

// String renders the subtitle as `index\nstart --> end\ntext` (no trailing
// newline). Was Display for Subtitle.
func (s Subtitle) String() string {
	var b strings.Builder
	// Line 1: Index
	b.WriteString(s.Index.String())
	b.WriteString("\n")
	// Line 2: Timestamp range
	b.WriteString(FormatTimestamp(s.StartTime.v))
	b.WriteString(" --> ")
	b.WriteString(FormatTimestamp(s.EndTime.v))
	b.WriteString("\n")
	// Line 3+: Text content (may be multi-line)
	b.WriteString(s.Text.v)
	return b.String()
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

// SubtitleUpdate represents a partial update to a subtitle. Was Rust
// struct SubtitleUpdate.
type SubtitleUpdate struct {
	StartTime *SubtitleTimestamp
	EndTime   *SubtitleTimestamp
	Text      *SubtitleText
}

// HasUpdates reports whether at least one field is specified.
// Was SubtitleUpdate::has_updates.
func (u SubtitleUpdate) HasUpdates() bool {
	return u.StartTime != nil || u.EndTime != nil || u.Text != nil
}

// ApplyTo applies this update to an existing subtitle, creating a new one with
// cross-field validation. Was SubtitleUpdate::apply_to.
func (u SubtitleUpdate) ApplyTo(s Subtitle) (Subtitle, error) {
	newStart := s.StartTime
	if u.StartTime != nil {
		newStart = *u.StartTime
	}
	newEnd := s.EndTime
	if u.EndTime != nil {
		newEnd = *u.EndTime
	}
	newText := s.Text
	if u.Text != nil {
		newText = *u.Text
	}

	// Use NewSubtitle to get cross-field validation (end > start)
	return NewSubtitle(s.Index, newStart, newEnd, newText)
}

// UpdateReport is the report from a successful update operation.
type UpdateReport struct {
	FilePath      string
	SubtitleIndex uint32
	FieldsUpdated []string
}

// AddReport is the report from a successful add operation.
type AddReport struct {
	FilePath       string
	NewIndex       uint32
	TotalSubtitles int
}

// ImportReport is the report from a successful import operation.
type ImportReport struct {
	FilePath       string
	BackupPath     string
	ImportedCount  int
	TotalSubtitles int
	StartIndex     uint32
	EndIndex       uint32
}
