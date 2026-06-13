package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lebe-dev/submarine/pkg/doctor"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// -- Path resolution --

// ResolvedPath holds a resolved file path with the components needed by command
// handlers. Port of the Rust `struct ResolvedPath`.
type ResolvedPath struct {
	// FullPath is the full canonical (or resolved) path to the file.
	FullPath string
	// BaseDir is the parent directory of the file.
	BaseDir string
	// Filename is the file name component.
	Filename string
}

// ResolveExistingPath resolves a file path that must already exist, with path
// traversal protection. Port of the Rust `resolve_existing_path`.
func ResolveExistingPath(file string) (*ResolvedPath, error) {
	filePath := file
	slog.Debug(fmt.Sprintf("resolving existing path: %q", filePath))

	if !filepath.IsAbs(filePath) {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %s", err)
		}

		resolved := filepath.Join(currentDir, filePath)
		normalized, err := canonicalize(resolved)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve file path: %s", err)
		}

		canonicalCurrentDir, err := canonicalize(currentDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve current directory: %s", err)
		}

		if !pathStartsWith(normalized, canonicalCurrentDir) {
			slog.Error(fmt.Sprintf("path traversal attempt detected: %q", filePath))
			return nil, fmt.Errorf("invalid file path: path traversal not allowed")
		}
	}

	canonicalPath, err := canonicalize(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %s", err)
	}
	slog.Debug(fmt.Sprintf("canonical path: %q", canonicalPath))

	baseDir := filepath.Dir(canonicalPath)
	if baseDir == canonicalPath {
		return nil, fmt.Errorf("invalid file path")
	}
	slog.Debug(fmt.Sprintf("base directory: %q", baseDir))

	filename := filepath.Base(canonicalPath)
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return nil, fmt.Errorf("invalid file name")
	}
	slog.Debug(fmt.Sprintf("filename: %s", filename))

	return &ResolvedPath{
		FullPath: canonicalPath,
		BaseDir:  baseDir,
		Filename: filename,
	}, nil
}

// ResolveNewPath resolves a file path that may not exist yet, with path
// traversal protection. Port of the Rust `resolve_new_path`.
func ResolveNewPath(file string) (*ResolvedPath, error) {
	filePath := file
	slog.Debug(fmt.Sprintf("resolving new path: %q", filePath))

	var resolvedPath string
	if !filepath.IsAbs(filePath) {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %s", err)
		}
		resolvedPath = filepath.Join(currentDir, filePath)
	} else {
		resolvedPath = filePath
	}

	if hasParentDirComponent(resolvedPath) {
		slog.Error(fmt.Sprintf("path traversal attempt detected: %q", filePath))
		return nil, fmt.Errorf("invalid file path: path traversal not allowed")
	}

	baseDir := filepath.Dir(resolvedPath)
	if baseDir == resolvedPath {
		return nil, fmt.Errorf("invalid file path")
	}
	slog.Debug(fmt.Sprintf("base directory: %q", baseDir))

	filename := filepath.Base(resolvedPath)
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return nil, fmt.Errorf("invalid file name")
	}
	slog.Debug(fmt.Sprintf("filename: %s", filename))

	return &ResolvedPath{
		FullPath: resolvedPath,
		BaseDir:  baseDir,
		Filename: filename,
	}, nil
}

// canonicalize mirrors Rust's `Path::canonicalize`: resolve symlinks and make
// absolute, erroring if the path does not exist.
func canonicalize(p string) (string, error) {
	if _, err := os.Lstat(p); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// pathStartsWith mirrors Rust's `Path::starts_with`: component-wise prefix.
func pathStartsWith(p, prefix string) bool {
	rel, err := filepath.Rel(prefix, p)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// hasParentDirComponent mirrors checking for a `Component::ParentDir` ("..").
func hasParentDirComponent(p string) bool {
	for _, c := range strings.Split(filepath.ToSlash(p), "/") {
		if c == ".." {
			return true
		}
	}
	return false
}

// -- Error formatting --

// CliError is a structured error with code, message, and optional hint. Port of
// the Rust `struct CliError`.
type CliError struct {
	Code    string
	Message string
	Hint    *string
}

// strPtr returns a pointer to s (helper for optional hints).
func strPtr(s string) *string { return &s }

// The Go subtitle.SubtitleError / doctor.DoctorError carry only the formatted
// Display string (Msg), not the raw variant payloads. The Rust
// `format_subtitle_error` / `format_doctor_error` re-format using those raw
// payloads, so we recover them by stripping/parsing the known Display
// templates (which are fixed and deterministic).

// FormatSubtitleError converts a SubtitleError into a structured CliError. Port
// of the Rust `format_subtitle_error`.
func FormatSubtitleError(e *subtitle.SubtitleError, file string) CliError {
	switch e.Kind {
	case subtitle.ErrFileNotFound:
		path := strings.TrimPrefix(e.Msg, "Subtitle file not found: ")
		return CliError{
			Code:    "file_not_found",
			Message: fmt.Sprintf("File not found: %s", path),
		}
	case subtitle.ErrInvalidPath:
		msg := strings.TrimPrefix(e.Msg, "Invalid file path: ")
		return CliError{
			Code:    "invalid_path",
			Message: fmt.Sprintf("Invalid file path: %s", msg),
		}
	case subtitle.ErrParse:
		src := strings.TrimPrefix(e.Msg, "Failed to parse subtitle file: ")
		return CliError{
			Code:    "parse_error",
			Message: fmt.Sprintf("Failed to parse subtitle file: %s", src),
			Hint:    strPtr(fmt.Sprintf("Try running 'sm doctor --fix %s' first", file)),
		}
	case subtitle.ErrIO:
		src := strings.TrimPrefix(e.Msg, "I/O error reading subtitle file: ")
		return CliError{
			Code:    "io_error",
			Message: fmt.Sprintf("I/O error: %s", src),
		}
	case subtitle.ErrSubtitleNotFound:
		// Display == CLI message: "Subtitle with index {idx} not found in file"
		return CliError{
			Code:    "subtitle_not_found",
			Message: e.Msg,
		}
	case subtitle.ErrNoFieldsToUpdate:
		return CliError{
			Code:    "no_fields_to_update",
			Message: "At least one of --start, --end, or --text must be specified",
		}
	case subtitle.ErrBackupFailed:
		msg := strings.TrimPrefix(e.Msg, "Failed to create backup: ")
		return CliError{
			Code:    "backup_failed",
			Message: fmt.Sprintf("Failed to create backup: %s", msg),
		}
	case subtitle.ErrWriteFailed:
		msg := strings.TrimPrefix(e.Msg, "Failed to write updated file: ")
		return CliError{
			Code:    "write_failed",
			Message: fmt.Sprintf("Failed to write updated file: %s", msg),
		}
	case subtitle.ErrTimestampConflict:
		newStart, lastEnd := parseTimestampConflict(e.Msg)
		return CliError{
			Code: "timestamp_conflict",
			Message: fmt.Sprintf(
				"Timestamp conflict: last subtitle ends at %s, new starts at %s",
				lastEnd, newStart,
			),
			Hint: strPtr("New subtitle must start at or after the last subtitle ends"),
		}
	case subtitle.ErrCSVParse:
		line, message := parseLineMessage(e.Msg, "CSV parsing error at line ")
		return CliError{
			Code:    "csv_parse_error",
			Message: fmt.Sprintf("CSV parsing failed at line %s: %s", line, message),
			Hint:    strPtr(fmt.Sprintf("Check the CSV file format at line %s", line)),
		}
	case subtitle.ErrInvalidCSVHeader:
		delimiter, actual := parseInvalidCsvHeader(e.Msg)
		return CliError{
			Code: "invalid_csv_header",
			Message: fmt.Sprintf(
				"Invalid CSV header. Expected: start_time%send_time%stext, got: %s",
				delimiter, delimiter, actual,
			),
		}
	case subtitle.ErrTimestampOverlap:
		line, newStart, existingEnd := parseTimestampOverlap(e.Msg)
		return CliError{
			Code: "timestamp_overlap",
			Message: fmt.Sprintf(
				"Timestamp overlap at line %s: existing ends at %s, new starts at %s",
				line, existingEnd, newStart,
			),
		}
	case subtitle.ErrAnchoredParse:
		line, message := parseLineMessage(e.Msg, "Anchored format parsing error at line ")
		return CliError{
			Code:    "anchored_parse_error",
			Message: fmt.Sprintf("Anchored format parsing failed at line %s: %s", line, message),
			Hint:    strPtr("Ensure format is [INDEX] TEXT with proper structure"),
		}
	case subtitle.ErrReferenceIndexNotFound:
		// Display == CLI message: "Reference file does not contain subtitle with index {index}"
		return CliError{
			Code:    "reference_index_not_found",
			Message: e.Msg,
			Hint:    strPtr("Check that the reference file contains all required indices"),
		}
	case subtitle.ErrInvalidAnchoredFormat:
		// Display == CLI message: "Invalid anchored format at line {line}: {message}"
		return CliError{
			Code:    "invalid_anchored_format",
			Message: e.Msg,
		}
	}
	return CliError{Code: "io_error", Message: e.Msg}
}

// parseTimestampConflict recovers (new_start, last_end) from the Display:
// "Timestamp conflict: new subtitle (starts at {new_start}) must start at or
// after the last subtitle ends (at {last_end})".
func parseTimestampConflict(msg string) (newStart, lastEnd string) {
	const startMarker = "(starts at "
	const midMarker = ") must start at or after the last subtitle ends (at "
	const endMarker = ")"
	i := strings.Index(msg, startMarker)
	if i < 0 {
		return "", ""
	}
	rest := msg[i+len(startMarker):]
	j := strings.Index(rest, midMarker)
	if j < 0 {
		return "", ""
	}
	newStart = rest[:j]
	rest = rest[j+len(midMarker):]
	lastEnd = strings.TrimSuffix(rest, endMarker)
	return newStart, lastEnd
}

// parseTimestampOverlap recovers (line, new_start, existing_end) from the Display:
// "Timestamp overlap detected: subtitle at CSV line {line} (starts at {new_start})
// overlaps with existing subtitle (ends at {existing_end})".
func parseTimestampOverlap(msg string) (line, newStart, existingEnd string) {
	const lineMarker = "subtitle at CSV line "
	const startMarker = " (starts at "
	const midMarker = ") overlaps with existing subtitle (ends at "
	const endMarker = ")"
	i := strings.Index(msg, lineMarker)
	if i < 0 {
		return "", "", ""
	}
	rest := msg[i+len(lineMarker):]
	j := strings.Index(rest, startMarker)
	if j < 0 {
		return "", "", ""
	}
	line = rest[:j]
	rest = rest[j+len(startMarker):]
	k := strings.Index(rest, midMarker)
	if k < 0 {
		return line, "", ""
	}
	newStart = rest[:k]
	rest = rest[k+len(midMarker):]
	existingEnd = strings.TrimSuffix(rest, endMarker)
	return line, newStart, existingEnd
}

// parseLineMessage recovers (line, message) from a Display of the form
// "<prefix>{line}: {message}".
func parseLineMessage(msg, prefix string) (line, message string) {
	rest := strings.TrimPrefix(msg, prefix)
	i := strings.Index(rest, ": ")
	if i < 0 {
		return rest, ""
	}
	return rest[:i], rest[i+len(": "):]
}

// parseInvalidCsvHeader recovers (delimiter, actual) from the Display:
// "Invalid CSV header: expected 'start_time{d}end_time{d}text', got '{actual}'".
func parseInvalidCsvHeader(msg string) (delimiter, actual string) {
	const exPrefix = "Invalid CSV header: expected 'start_time"
	const exMid = "end_time"
	i := strings.Index(msg, exPrefix)
	if i < 0 {
		return "", ""
	}
	rest := msg[i+len(exPrefix):]
	j := strings.Index(rest, exMid)
	if j < 0 {
		return "", ""
	}
	delimiter = rest[:j]
	// actual is between "got '" and the trailing "'".
	const gotMarker = "got '"
	g := strings.Index(rest, gotMarker)
	if g < 0 {
		return delimiter, ""
	}
	actual = rest[g+len(gotMarker):]
	actual = strings.TrimSuffix(actual, "'")
	return delimiter, actual
}

// FormatDoctorError converts a DoctorError into a structured CliError. Port of
// the Rust `format_doctor_error`.
func FormatDoctorError(e *doctor.DoctorError) CliError {
	switch e.Kind {
	case doctor.ErrFileNotFound:
		path := strings.TrimPrefix(e.Msg, "file not found: ")
		return CliError{
			Code:    "file_not_found",
			Message: fmt.Sprintf("File not found: %s", path),
		}
	case doctor.ErrInvalidPath:
		msg := strings.TrimPrefix(e.Msg, "invalid file path: ")
		return CliError{
			Code:    "invalid_path",
			Message: fmt.Sprintf("Invalid file path: %s", msg),
		}
	case doctor.ErrIO:
		src := strings.TrimPrefix(e.Msg, "i/o error: ")
		return CliError{
			Code:    "io_error",
			Message: fmt.Sprintf("I/O error: %s", src),
		}
	case doctor.ErrBackupFailed:
		msg := strings.TrimPrefix(e.Msg, "backup creation failed: ")
		return CliError{
			Code:    "backup_failed",
			Message: fmt.Sprintf("Backup creation failed: %s", msg),
		}
	case doctor.ErrValidationFailed:
		msg := strings.TrimPrefix(e.Msg, "failed to validate fixed file: ")
		return CliError{
			Code:    "validation_failed",
			Message: fmt.Sprintf("Validation failed: %s", msg),
		}
	}
	return CliError{Code: "io_error", Message: e.Msg}
}

// -- Range parsing --

// ParseRange parses a range string in format "START-END" into a (start, end)
// pair. Port of the Rust `parse_range`.
func ParseRange(rng string) (uint32, uint32, error) {
	parts := strings.Split(rng, "-")

	if len(parts) != 2 {
		return 0, 0, fmt.Errorf(
			"Invalid range format '%s'. Expected format: START-END (e.g., 1-50)",
			rng,
		)
	}

	start, err := parseU32(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf(
			"Invalid start index '%s'. Must be a positive number",
			parts[0],
		)
	}

	end, err := parseU32(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf(
			"Invalid end index '%s'. Must be a positive number",
			parts[1],
		)
	}

	if start < 1 {
		return 0, 0, fmt.Errorf("Start index must be >= 1, got %d", start)
	}

	if end < 1 {
		return 0, 0, fmt.Errorf("End index must be >= 1, got %d", end)
	}

	if start > end {
		return 0, 0, fmt.Errorf(
			"Start index must be <= end index (got %d > %d)",
			start, end,
		)
	}

	return start, end, nil
}

// parseU32 mirrors Rust's `str::parse::<u32>()` (rejects whitespace, signs,
// and out-of-range values; only decimal digits accepted).
func parseU32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// ValidateRangeBoundaries validates that the requested range is within the
// available subtitle indices. Port of the Rust `validate_range_boundaries`.
func ValidateRangeBoundaries(start, end uint32, subtitles []subtitle.Subtitle) error {
	if len(subtitles) == 0 {
		return fmt.Errorf("File contains no subtitles")
	}

	minIndex := subtitles[0].Index.Value()
	maxIndex := subtitles[0].Index.Value()
	for _, s := range subtitles {
		idx := s.Index.Value()
		if idx < minIndex {
			minIndex = idx
		}
		if idx > maxIndex {
			maxIndex = idx
		}
	}

	slog.Debug(fmt.Sprintf(
		"file contains indices %d-%d, requested range %d-%d",
		minIndex, maxIndex, start, end,
	))

	if start > maxIndex {
		return fmt.Errorf(
			"Start index %d is beyond the last subtitle (index %d)",
			start, maxIndex,
		)
	}

	if end < minIndex {
		return fmt.Errorf(
			"End index %d is before the first subtitle (index %d)",
			end, minIndex,
		)
	}

	return nil
}

// -- Input validation --

// RejectControlChars rejects control characters in user-provided text input.
// Allows \n, \r, \t but rejects all other ASCII control characters (< 0x20).
// Port of the Rust `reject_control_chars`.
func RejectControlChars(input, field string) error {
	// Rust uses chars().enumerate(), so the position is the char index, not the
	// byte offset.
	i := 0
	for _, ch := range input {
		if isASCIIControl(ch) && ch != '\n' && ch != '\r' && ch != '\t' {
			return fmt.Errorf(
				"Invalid character in %s: control character (0x%02x) at position %d",
				field, uint32(ch), i,
			)
		}
		i++
	}
	return nil
}

// isASCIIControl mirrors Rust's `char::is_ascii_control`.
func isASCIIControl(ch rune) bool {
	return ch < 0x20 || ch == 0x7f
}

// RejectPercentEncoding rejects percent-encoded sequences in filenames to
// prevent double-encoding. Port of the Rust `reject_percent_encoding`.
func RejectPercentEncoding(input, field string) error {
	bytes := []byte(input)
	for i := 0; i < len(bytes); i++ {
		if bytes[i] == '%' && i+2 < len(bytes) {
			hex1 := bytes[i+1]
			hex2 := bytes[i+2]
			if isASCIIHexDigit(hex1) && isASCIIHexDigit(hex2) {
				return fmt.Errorf(
					"Invalid character in %s: percent-encoded sequence '%%%c%c' detected. Use plain filenames",
					field, hex1, hex2,
				)
			}
		}
	}
	return nil
}

// isASCIIHexDigit mirrors Rust's `u8::is_ascii_hexdigit`.
func isASCIIHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}
