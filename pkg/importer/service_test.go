package importer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// writeTempFile writes content to a fresh temp file and returns its path. It
// mirrors the Rust tests' use of NamedTempFile + writeln!.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

// subtitleErrKind extracts the *subtitle.SubtitleError Kind from err, failing
// the test if err is not such an error.
func subtitleErrKind(t *testing.T, err error) subtitle.ErrorKind {
	t.Helper()
	var se *subtitle.SubtitleError
	if !errors.As(err, &se) {
		t.Fatalf("expected *subtitle.SubtitleError, got %T: %v", err, err)
	}
	return se.Kind
}

func TestParseCsvValid(t *testing.T) {
	path := writeTempFile(t, "start_time|end_time|text\n00:00:01,000|00:00:02,000|First subtitle\n00:00:03,000|00:00:04,000|Second subtitle\n")

	service := NewCsvImportService()
	rows, err := service.ParseCsvFile(path, '|')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].LineNumber != 2 {
		t.Errorf("rows[0].LineNumber = %d, want 2", rows[0].LineNumber)
	}
	if rows[0].StartTime != "00:00:01,000" {
		t.Errorf("rows[0].StartTime = %q, want %q", rows[0].StartTime, "00:00:01,000")
	}
	if rows[0].EndTime != "00:00:02,000" {
		t.Errorf("rows[0].EndTime = %q, want %q", rows[0].EndTime, "00:00:02,000")
	}
	if rows[0].Text != "First subtitle" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "First subtitle")
	}
	if rows[1].LineNumber != 3 {
		t.Errorf("rows[1].LineNumber = %d, want 3", rows[1].LineNumber)
	}
	if rows[1].Text != "Second subtitle" {
		t.Errorf("rows[1].Text = %q, want %q", rows[1].Text, "Second subtitle")
	}
}

func TestParseCsvMultilineText(t *testing.T) {
	path := writeTempFile(t, "start_time|end_time|text\n00:00:01,000|00:00:02,000|Line 1\\nLine 2\n")

	service := NewCsvImportService()
	rows, err := service.ParseCsvFile(path, '|')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].Text != "Line 1\nLine 2" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "Line 1\nLine 2")
	}
}

func TestParseCsvInvalidHeader(t *testing.T) {
	path := writeTempFile(t, "wrong|header|format\n")

	service := NewCsvImportService()
	_, err := service.ParseCsvFile(path, '|')
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if kind := subtitleErrKind(t, err); kind != subtitle.ErrInvalidCSVHeader {
		t.Errorf("error kind = %v, want ErrInvalidCSVHeader", kind)
	}
}

func TestParseCsvWrongColumnCount(t *testing.T) {
	path := writeTempFile(t, "start_time|end_time|text\n00:00:01,000|00:00:02,000\n") // Missing text column

	service := NewCsvImportService()
	_, err := service.ParseCsvFile(path, '|')
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var se *subtitle.SubtitleError
	if !errors.As(err, &se) {
		t.Fatalf("expected *subtitle.SubtitleError, got %T: %v", err, err)
	}
	if se.Kind != subtitle.ErrCSVParse {
		t.Errorf("error kind = %v, want ErrCSVParse", se.Kind)
	}
	// Rust asserts CsvParseError { line: 2, .. }; the formatted message embeds
	// "at line 2".
	if want := "CSV parsing error at line 2:"; !strings.Contains(se.Msg, want) {
		t.Errorf("error message = %q, want it to contain %q", se.Msg, want)
	}
}

func TestParseCsvCustomDelimiter(t *testing.T) {
	path := writeTempFile(t, "start_time;end_time;text\n00:00:01,000;00:00:02,000;First\n")

	service := NewCsvImportService()
	rows, err := service.ParseCsvFile(path, ';')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Text != "First" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "First")
	}
}

func TestParseCsvTrimWhitespace(t *testing.T) {
	path := writeTempFile(t, "start_time|end_time|text\n  00:00:01,000  |  00:00:02,000  |Text\n")

	service := NewCsvImportService()
	rows, err := service.ParseCsvFile(path, '|')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].StartTime != "00:00:01,000" {
		t.Errorf("rows[0].StartTime = %q, want %q", rows[0].StartTime, "00:00:01,000")
	}
	if rows[0].EndTime != "00:00:02,000" {
		t.Errorf("rows[0].EndTime = %q, want %q", rows[0].EndTime, "00:00:02,000")
	}
}

func TestParseCsvHtmlTags(t *testing.T) {
	path := writeTempFile(t, "start_time|end_time|text\n00:00:01,000|00:00:02,000|<i>Italic</i>\n")

	service := NewCsvImportService()
	rows, err := service.ParseCsvFile(path, '|')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].Text != "<i>Italic</i>" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "<i>Italic</i>")
	}
}

// ========== Anchored Format Tests ==========

func TestParseAnchoredSingleLine(t *testing.T) {
	path := writeTempFile(t, "[1] First subtitle\n[2] Second subtitle\n")

	service := NewAnchoredImportService()
	rows, err := service.ParseAnchoredFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Index != 1 {
		t.Errorf("rows[0].Index = %d, want 1", rows[0].Index)
	}
	if rows[0].Text != "First subtitle" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "First subtitle")
	}
	if rows[0].LineNumber != 1 {
		t.Errorf("rows[0].LineNumber = %d, want 1", rows[0].LineNumber)
	}
	if rows[1].Index != 2 {
		t.Errorf("rows[1].Index = %d, want 2", rows[1].Index)
	}
	if rows[1].Text != "Second subtitle" {
		t.Errorf("rows[1].Text = %q, want %q", rows[1].Text, "Second subtitle")
	}
	if rows[1].LineNumber != 2 {
		t.Errorf("rows[1].LineNumber = %d, want 2", rows[1].LineNumber)
	}
}

func TestParseAnchoredMultiline(t *testing.T) {
	path := writeTempFile(t, "[1] Line 1 of sub 1\nLine 2 of sub 1\nLine 3 of sub 1\n[2] Single line\n")

	service := NewAnchoredImportService()
	rows, err := service.ParseAnchoredFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Text != "Line 1 of sub 1\nLine 2 of sub 1\nLine 3 of sub 1" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "Line 1 of sub 1\nLine 2 of sub 1\nLine 3 of sub 1")
	}
	if rows[1].Text != "Single line" {
		t.Errorf("rows[1].Text = %q, want %q", rows[1].Text, "Single line")
	}
}

func TestParseAnchoredWithHtml(t *testing.T) {
	path := writeTempFile(t, "[1] <i>Italic text</i>\n[2] <b>Bold text</b>\n")

	service := NewAnchoredImportService()
	rows, err := service.ParseAnchoredFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rows[0].Text != "<i>Italic text</i>" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "<i>Italic text</i>")
	}
	if rows[1].Text != "<b>Bold text</b>" {
		t.Errorf("rows[1].Text = %q, want %q", rows[1].Text, "<b>Bold text</b>")
	}
}

func TestParseAnchoredInvalidNoBracket(t *testing.T) {
	path := writeTempFile(t, "Text without bracket\n")

	service := NewAnchoredImportService()
	_, err := service.ParseAnchoredFile(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var se *subtitle.SubtitleError
	if !errors.As(err, &se) {
		t.Fatalf("expected *subtitle.SubtitleError, got %T: %v", err, err)
	}
	if se.Kind != subtitle.ErrAnchoredParse {
		t.Errorf("error kind = %v, want ErrAnchoredParse", se.Kind)
	}
	if want := "at line 1:"; !strings.Contains(se.Msg, want) {
		t.Errorf("error message = %q, want it to contain %q", se.Msg, want)
	}
}

func TestParseAnchoredEmptyText(t *testing.T) {
	path := writeTempFile(t, "[1]\n[2] Next\n")

	service := NewAnchoredImportService()
	_, err := service.ParseAnchoredFile(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var se *subtitle.SubtitleError
	if !errors.As(err, &se) {
		t.Fatalf("expected *subtitle.SubtitleError, got %T: %v", err, err)
	}
	if se.Kind != subtitle.ErrAnchoredParse {
		t.Errorf("error kind = %v, want ErrAnchoredParse", se.Kind)
	}
	if want := "at line 1:"; !strings.Contains(se.Msg, want) {
		t.Errorf("error message = %q, want it to contain %q", se.Msg, want)
	}
}

func TestParseAnchoredEmptyFile(t *testing.T) {
	path := writeTempFile(t, "")

	service := NewAnchoredImportService()
	_, err := service.ParseAnchoredFile(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var se *subtitle.SubtitleError
	if !errors.As(err, &se) {
		t.Fatalf("expected *subtitle.SubtitleError, got %T: %v", err, err)
	}
	if se.Kind != subtitle.ErrAnchoredParse {
		t.Errorf("error kind = %v, want ErrAnchoredParse", se.Kind)
	}
	if want := "at line 0:"; !strings.Contains(se.Msg, want) {
		t.Errorf("error message = %q, want it to contain %q", se.Msg, want)
	}
}

func TestParseAnchoredMixedIndices(t *testing.T) {
	path := writeTempFile(t, "[5] Fifth subtitle\n[10] Tenth subtitle\n[1] First subtitle\n")

	service := NewAnchoredImportService()
	rows, err := service.ParseAnchoredFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Index != 5 {
		t.Errorf("rows[0].Index = %d, want 5", rows[0].Index)
	}
	if rows[1].Index != 10 {
		t.Errorf("rows[1].Index = %d, want 10", rows[1].Index)
	}
	if rows[2].Index != 1 {
		t.Errorf("rows[2].Index = %d, want 1", rows[2].Index)
	}
}

func TestParseAnchoredWithBlankLines(t *testing.T) {
	path := writeTempFile(t, "[1] First\n\n[2] Second\n")

	service := NewAnchoredImportService()
	rows, err := service.ParseAnchoredFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Blank line becomes part of the first subtitle's text
	if rows[0].Text != "First" {
		t.Errorf("rows[0].Text = %q, want %q", rows[0].Text, "First")
	}
	if rows[1].Text != "Second" {
		t.Errorf("rows[1].Text = %q, want %q", rows[1].Text, "Second")
	}
}
