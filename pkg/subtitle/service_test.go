package subtitle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestService creates a test service backed by t.TempDir().
func createTestService(t *testing.T) (*SubRipService, string) {
	t.Helper()
	dir := t.TempDir()
	return NewSubRipService(dir), dir
}

// writeTestFile writes a test file into dir.
func writeTestFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func svcMakeTestSubtitle(t *testing.T, index uint32, startMs, endMs int64, text string) Subtitle {
	t.Helper()
	idx, _ := NewSubtitleIndex(index)
	start, _ := NewSubtitleTimestamp(ms(startMs))
	end, _ := NewSubtitleTimestamp(ms(endMs))
	txt, _ := NewSubtitleText(text)
	sub, err := NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("NewSubtitle: %v", err)
	}
	return sub
}

func errKind(err error) (ErrorKind, bool) {
	var se *SubtitleError
	if errors.As(err, &se) {
		return se.Kind, true
	}
	return 0, false
}

func TestGetSubtitleByIDFound(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nFirst subtitle\n\n" +
		"2\n00:00:03,500 --> 00:00:05,500\nSecond subtitle"
	writeTestFile(t, dir, "test.srt", content)

	result, err := service.GetByID("test.srt", 2)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if result == nil {
		t.Fatalf("expected some result")
	}
	if result.Index.Value() != 2 {
		t.Errorf("expected index 2, got %d", result.Index.Value())
	}
	if result.Text.Value() != "Second subtitle" {
		t.Errorf("expected 'Second subtitle', got %q", result.Text.Value())
	}
}

func TestGetSubtitleByIDNotFound(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nFirst subtitle"
	writeTestFile(t, dir, "test.srt", content)

	result, err := service.GetByID("test.srt", 99)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result")
	}
}

func TestFileNotFound(t *testing.T) {
	service, _ := createTestService(t)

	_, err := service.GetByID("nonexistent.srt", 1)
	if kind, ok := errKind(err); !ok || kind != ErrFileNotFound {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestParseError(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\nINVALID FORMAT\nBroken"
	writeTestFile(t, dir, "broken.srt", content)

	_, err := service.GetByID("broken.srt", 1)
	if kind, ok := errKind(err); !ok || kind != ErrParse {
		t.Errorf("expected ErrParse, got %v", err)
	}
}

func TestInvalidPathTraversal(t *testing.T) {
	service, _ := createTestService(t)

	_, err := service.GetByID("../etc/passwd", 1)
	if kind, ok := errKind(err); !ok || kind != ErrInvalidPath {
		t.Errorf("expected ErrInvalidPath, got %v", err)
	}
}

func TestInvalidPathAbsolute(t *testing.T) {
	service, _ := createTestService(t)

	_, err := service.GetByID("/etc/passwd", 1)
	if kind, ok := errKind(err); !ok || kind != ErrInvalidPath {
		t.Errorf("expected ErrInvalidPath, got %v", err)
	}
}

func TestInvalidPathEmpty(t *testing.T) {
	service, _ := createTestService(t)

	_, err := service.GetByID("", 1)
	if kind, ok := errKind(err); !ok || kind != ErrInvalidPath {
		t.Errorf("expected ErrInvalidPath, got %v", err)
	}
}

func TestEmptyFile(t *testing.T) {
	service, dir := createTestService(t)
	writeTestFile(t, dir, "empty.srt", "")

	result, err := service.GetByID("empty.srt", 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result")
	}
}

func TestComplexSubtitleWithHTML(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>"
	writeTestFile(t, dir, "complex.srt", content)

	result, err := service.GetByID("complex.srt", 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if result == nil {
		t.Fatalf("expected some result")
	}
	if !result.HasHTMLTags() {
		t.Errorf("expected HTML tags")
	}
	if result.LineCount() != 2 {
		t.Errorf("expected line count 2, got %d", result.LineCount())
	}
}

func TestSubtitleWithGapInIndices(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n" +
		"5\n00:00:03,000 --> 00:00:04,000\nFifth"
	writeTestFile(t, dir, "gaps.srt", content)

	result, err := service.GetByID("gaps.srt", 5)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if result == nil {
		t.Errorf("expected some result for index 5")
	}

	resultMissing, err := service.GetByID("gaps.srt", 3)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if resultMissing != nil {
		t.Errorf("expected nil result for index 3")
	}
}

// Tests for parse_srt_file method

func TestParseSrtSimple(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000\nFirst subtitle\n\n" +
		"2\n00:00:03,500 --> 00:00:05,500\nSecond subtitle\n\n" +
		"3\n00:00:06,000 --> 00:00:08,000\nThird subtitle"

	result, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	if result[0].Index.Value() != 1 || result[0].Text.Value() != "First subtitle" {
		t.Errorf("unexpected result[0]: %+v", result[0])
	}
	if result[1].Index.Value() != 2 || result[1].Text.Value() != "Second subtitle" {
		t.Errorf("unexpected result[1]: %+v", result[1])
	}
	if result[2].Index.Value() != 3 || result[2].Text.Value() != "Third subtitle" {
		t.Errorf("unexpected result[2]: %+v", result[2])
	}
}

func TestParseSrtMultilineText(t *testing.T) {
	content := "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>"

	result, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Index.Value() != 1 {
		t.Errorf("expected index 1")
	}
	if result[0].Text.Value() != "<i>Previously on\n\"Resident Alien\"...</i>" {
		t.Errorf("unexpected text: %q", result[0].Text.Value())
	}
	if result[0].LineCount() != 2 {
		t.Errorf("expected line count 2, got %d", result[0].LineCount())
	}
}

func TestParseSrtWithHTML(t *testing.T) {
	content := "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>\n\n" +
		"2\n00:00:03,481 --> 00:00:05,135\nHello, Harry."

	result, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if !result[0].HasHTMLTags() {
		t.Errorf("expected result[0] HTML tags")
	}
	if result[1].HasHTMLTags() {
		t.Errorf("expected result[1] no HTML tags")
	}
}

func TestParseSrtGapsInIndices(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n" +
		"5\n00:00:03,000 --> 00:00:04,000\nFifth"

	result, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].Index.Value() != 1 {
		t.Errorf("expected index 1")
	}
	if result[1].Index.Value() != 5 {
		t.Errorf("expected index 5")
	}
}

func TestParseSrtEmptyFile(t *testing.T) {
	result, err := parseSrtFile("")
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestParseSrtSingleSubtitle(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000\nSingle subtitle"

	result, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Index.Value() != 1 || result[0].Text.Value() != "Single subtitle" {
		t.Errorf("unexpected result[0]: %+v", result[0])
	}
}

func TestParseSrtInvalidIndex(t *testing.T) {
	content := "NOT_A_NUMBER\n00:00:01,000 --> 00:00:03,000\nText"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "expected subtitle index") && !strings.Contains(msg, "invalid digit") {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestParseSrtInvalidTimestampFormat(t *testing.T) {
	content := "1\nINVALID_TIMESTAMP\nText"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid timestamp format") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestParseSrtMissingArrow(t *testing.T) {
	content := "1\n00:00:01,000 00:00:03,000\nText"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid timestamp format") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestParseSrtEmptyText(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000\n\n\n2\n00:00:04,000 --> 00:00:05,000\nValid text"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "invalid subtitle text") &&
		!strings.Contains(msg, "empty") &&
		!strings.Contains(msg, "insufficient") {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestParseSrtIndexZero(t *testing.T) {
	content := "0\n00:00:01,000 --> 00:00:03,000\nText"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid index value") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestParseSrtEndBeforeStart(t *testing.T) {
	content := "1\n00:00:05,000 --> 00:00:03,000\nText"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "End time must be after start time") &&
		!strings.Contains(msg, "failed to create subtitle") {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestParseSrtWhitespaceHandling(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000\n  Trimmed text  \n\n\n2\n00:00:04,000 --> 00:00:06,000\nSecond"

	result, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].Text.Value() != "Trimmed text" {
		t.Errorf("expected 'Trimmed text', got %q", result[0].Text.Value())
	}
}

func TestParseSrtInsufficientLines(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000"

	_, err := parseSrtFile(content)
	if err == nil {
		t.Fatalf("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "incomplete subtitle") && !strings.Contains(msg, "missing text") {
		t.Errorf("unexpected error message: %q", msg)
	}
}

// Tests for blank lines within subtitle text

func TestParseSrtWithBlankLineInText(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst line\n\nSecond line"

	subtitles, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subtitles) != 1 {
		t.Fatalf("expected 1, got %d", len(subtitles))
	}
	if subtitles[0].Text.Value() != "First line\n\nSecond line" {
		t.Errorf("unexpected text: %q", subtitles[0].Text.Value())
	}
}

func TestParseSrtMultipleBlankLinesInText(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nLine 1\n\n\nLine 2"

	subtitles, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subtitles) != 1 {
		t.Fatalf("expected 1, got %d", len(subtitles))
	}
	if subtitles[0].Text.Value() != "Line 1\n\n\nLine 2" {
		t.Errorf("unexpected text: %q", subtitles[0].Text.Value())
	}
}

func TestParseSrtFileEndsWithoutBlankLine(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nText without trailing blank"

	subtitles, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subtitles) != 1 {
		t.Fatalf("expected 1, got %d", len(subtitles))
	}
	if subtitles[0].Text.Value() != "Text without trailing blank" {
		t.Errorf("unexpected text: %q", subtitles[0].Text.Value())
	}
}

func TestParseSrtWithLeadingBlankLines(t *testing.T) {
	content := "\n\n1\n00:00:01,000 --> 00:00:02,000\nText"

	subtitles, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subtitles) != 1 {
		t.Fatalf("expected 1, got %d", len(subtitles))
	}
	if subtitles[0].Text.Value() != "Text" {
		t.Errorf("unexpected text: %q", subtitles[0].Text.Value())
	}
}

func TestParseSrtMultipleSubtitlesWithBlankText(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nText A\n\nMore A\n\n2\n00:00:03,000 --> 00:00:04,000\nText B"

	subs, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2, got %d", len(subs))
	}
	if subs[0].Text.Value() != "Text A\n\nMore A" {
		t.Errorf("unexpected subs[0]: %q", subs[0].Text.Value())
	}
	if subs[1].Text.Value() != "Text B" {
		t.Errorf("unexpected subs[1]: %q", subs[1].Text.Value())
	}
}

func TestParseSrtMultipleBlankLinesBetweenBlocks(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond"

	subs, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2, got %d", len(subs))
	}
	if subs[0].Text.Value() != "First" {
		t.Errorf("unexpected subs[0]: %q", subs[0].Text.Value())
	}
	if subs[1].Text.Value() != "Second" {
		t.Errorf("unexpected subs[1]: %q", subs[1].Text.Value())
	}
}

func TestParseSrtComplexMultilineWithBlanks(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000\nLine 1\n\nLine 2\nLine 3\n\nLine 4\n\n2\n00:00:04,000 --> 00:00:06,000\nSimple"

	subs, err := parseSrtFile(content)
	if err != nil {
		t.Fatalf("parseSrtFile: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2, got %d", len(subs))
	}
	if subs[0].Text.Value() != "Line 1\n\nLine 2\nLine 3\n\nLine 4" {
		t.Errorf("unexpected subs[0]: %q", subs[0].Text.Value())
	}
	if subs[1].Text.Value() != "Simple" {
		t.Errorf("unexpected subs[1]: %q", subs[1].Text.Value())
	}
}

// Tests for set functionality

func TestSerializeToSrt(t *testing.T) {
	subtitle1 := svcMakeTestSubtitle(t, 1, 1000, 2000, "First")
	subtitle2 := svcMakeTestSubtitle(t, 2, 3000, 4000, "Second")

	content := serializeToSrt([]Subtitle{subtitle1, subtitle2})

	expected := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond"
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

func TestSetSubtitleText(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nOriginal text"
	writeTestFile(t, dir, "test.srt", content)

	newText, _ := NewSubtitleText("Updated text")
	update := SubtitleUpdate{Text: &newText}

	report, err := service.Set("test.srt", 1, update)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if report.SubtitleIndex != 1 {
		t.Errorf("expected index 1, got %d", report.SubtitleIndex)
	}
	if len(report.FieldsUpdated) != 1 || report.FieldsUpdated[0] != "text" {
		t.Errorf("expected [text], got %v", report.FieldsUpdated)
	}

	subtitle, err := service.GetByID("test.srt", 1)
	if err != nil || subtitle == nil {
		t.Fatalf("GetByID: %v / %v", subtitle, err)
	}
	if subtitle.Text.Value() != "Updated text" {
		t.Errorf("expected 'Updated text', got %q", subtitle.Text.Value())
	}
	if subtitle.StartTime.Millis() != 1000 {
		t.Errorf("expected start 1000, got %d", subtitle.StartTime.Millis())
	}
	if subtitle.EndTime.Millis() != 3000 {
		t.Errorf("expected end 3000, got %d", subtitle.EndTime.Millis())
	}
}

func TestSetSubtitleStartTime(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nOriginal text"
	writeTestFile(t, dir, "test.srt", content)

	newStart, _ := NewSubtitleTimestamp(ms(2000))
	update := SubtitleUpdate{StartTime: &newStart}

	report, err := service.Set("test.srt", 1, update)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if report.SubtitleIndex != 1 {
		t.Errorf("expected index 1, got %d", report.SubtitleIndex)
	}
	if len(report.FieldsUpdated) != 1 || report.FieldsUpdated[0] != "start_time" {
		t.Errorf("expected [start_time], got %v", report.FieldsUpdated)
	}

	subtitle, err := service.GetByID("test.srt", 1)
	if err != nil || subtitle == nil {
		t.Fatalf("GetByID: %v / %v", subtitle, err)
	}
	if subtitle.StartTime.Millis() != 2000 {
		t.Errorf("expected start 2000, got %d", subtitle.StartTime.Millis())
	}
	if subtitle.Text.Value() != "Original text" {
		t.Errorf("expected 'Original text', got %q", subtitle.Text.Value())
	}
}

func TestSetSubtitleAllFields(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nOriginal text"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(2000))
	end, _ := NewSubtitleTimestamp(ms(4000))
	txt, _ := NewSubtitleText("New text")
	update := SubtitleUpdate{StartTime: &start, EndTime: &end, Text: &txt}

	report, err := service.Set("test.srt", 1, update)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(report.FieldsUpdated) != 3 {
		t.Errorf("expected 3 fields, got %v", report.FieldsUpdated)
	}

	subtitle, err := service.GetByID("test.srt", 1)
	if err != nil || subtitle == nil {
		t.Fatalf("GetByID: %v / %v", subtitle, err)
	}
	if subtitle.StartTime.Millis() != 2000 {
		t.Errorf("expected start 2000, got %d", subtitle.StartTime.Millis())
	}
	if subtitle.EndTime.Millis() != 4000 {
		t.Errorf("expected end 4000, got %d", subtitle.EndTime.Millis())
	}
	if subtitle.Text.Value() != "New text" {
		t.Errorf("expected 'New text', got %q", subtitle.Text.Value())
	}
}

func TestSetSubtitleNotFound(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nText"
	writeTestFile(t, dir, "test.srt", content)

	txt, _ := NewSubtitleText("New")
	update := SubtitleUpdate{Text: &txt}

	_, err := service.Set("test.srt", 99, update)
	var se *SubtitleError
	if !errors.As(err, &se) || se.Kind != ErrSubtitleNotFound {
		t.Errorf("expected ErrSubtitleNotFound, got %v", err)
	}
}

func TestSetSubtitleNoFields(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nText"
	writeTestFile(t, dir, "test.srt", content)

	update := SubtitleUpdate{}

	_, err := service.Set("test.srt", 1, update)
	if kind, ok := errKind(err); !ok || kind != ErrNoFieldsToUpdate {
		t.Errorf("expected ErrNoFieldsToUpdate, got %v", err)
	}
}

func TestSetSubtitleInvalidTimeOrder(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nText"
	writeTestFile(t, dir, "test.srt", content)

	end, _ := NewSubtitleTimestamp(ms(500))
	update := SubtitleUpdate{EndTime: &end}

	if _, err := service.Set("test.srt", 1, update); err == nil {
		t.Errorf("expected error")
	}
}

func TestSetSubtitleBackupCreated(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nText"
	writeTestFile(t, dir, "test.srt", content)

	txt, _ := NewSubtitleText("New")
	update := SubtitleUpdate{Text: &txt}

	report, err := service.Set("test.srt", 1, update)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if report.SubtitleIndex != 1 {
		t.Errorf("expected index 1, got %d", report.SubtitleIndex)
	}
	if len(report.FieldsUpdated) != 1 || report.FieldsUpdated[0] != "text" {
		t.Errorf("expected [text], got %v", report.FieldsUpdated)
	}
}

func TestSetSubtitlePreservesOtherSubtitles(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond\n\n3\n00:00:05,000 --> 00:00:06,000\nThird"
	writeTestFile(t, dir, "test.srt", content)

	txt, _ := NewSubtitleText("MODIFIED")
	update := SubtitleUpdate{Text: &txt}

	if _, err := service.Set("test.srt", 2, update); err != nil {
		t.Fatalf("Set: %v", err)
	}

	all, err := service.GetAll("test.srt")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}
	if all[0].Text.Value() != "First" {
		t.Errorf("expected 'First', got %q", all[0].Text.Value())
	}
	if all[1].Text.Value() != "MODIFIED" {
		t.Errorf("expected 'MODIFIED', got %q", all[1].Text.Value())
	}
	if all[2].Text.Value() != "Third" {
		t.Errorf("expected 'Third', got %q", all[2].Text.Value())
	}
}

func TestSetSubtitleMultilineText(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:03,000\nOriginal"
	writeTestFile(t, dir, "test.srt", content)

	txt, _ := NewSubtitleText("Line 1\nLine 2\nLine 3")
	update := SubtitleUpdate{Text: &txt}

	if _, err := service.Set("test.srt", 1, update); err != nil {
		t.Fatalf("Set: %v", err)
	}

	subtitle, err := service.GetByID("test.srt", 1)
	if err != nil || subtitle == nil {
		t.Fatalf("GetByID: %v / %v", subtitle, err)
	}
	if subtitle.LineCount() != 3 {
		t.Errorf("expected line count 3, got %d", subtitle.LineCount())
	}
	if subtitle.Text.Value() != "Line 1\nLine 2\nLine 3" {
		t.Errorf("unexpected text: %q", subtitle.Text.Value())
	}
}

// Tests for add functionality

func TestAddToEmptyFile(t *testing.T) {
	service, dir := createTestService(t)
	writeTestFile(t, dir, "empty.srt", "")

	start, _ := NewSubtitleTimestamp(ms(1000))
	end, _ := NewSubtitleTimestamp(ms(2000))
	text, _ := NewSubtitleText("First subtitle")

	report, err := service.Add("empty.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 1 {
		t.Errorf("expected new index 1, got %d", report.NewIndex)
	}
	if report.TotalSubtitles != 1 {
		t.Errorf("expected total 1, got %d", report.TotalSubtitles)
	}

	subtitles, err := service.GetAll("empty.srt")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(subtitles) != 1 {
		t.Fatalf("expected 1, got %d", len(subtitles))
	}
	if subtitles[0].Index.Value() != 1 {
		t.Errorf("expected index 1")
	}
	if subtitles[0].Text.Value() != "First subtitle" {
		t.Errorf("expected 'First subtitle', got %q", subtitles[0].Text.Value())
	}
}

func TestAddToExistingFile(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(5000))
	end, _ := NewSubtitleTimestamp(ms(6000))
	text, _ := NewSubtitleText("Third")

	report, err := service.Add("test.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 3 {
		t.Errorf("expected new index 3, got %d", report.NewIndex)
	}
	if report.TotalSubtitles != 3 {
		t.Errorf("expected total 3, got %d", report.TotalSubtitles)
	}

	subtitles, err := service.GetAll("test.srt")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(subtitles) != 3 {
		t.Fatalf("expected 3, got %d", len(subtitles))
	}
	if subtitles[2].Index.Value() != 3 {
		t.Errorf("expected index 3")
	}
	if subtitles[2].Text.Value() != "Third" {
		t.Errorf("expected 'Third', got %q", subtitles[2].Text.Value())
	}
}

func TestAddWithGapInIndices(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n5\n00:00:03,000 --> 00:00:04,000\nFifth"
	writeTestFile(t, dir, "gaps.srt", content)

	start, _ := NewSubtitleTimestamp(ms(5000))
	end, _ := NewSubtitleTimestamp(ms(6000))
	text, _ := NewSubtitleText("Sixth")

	report, err := service.Add("gaps.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 6 {
		t.Errorf("expected new index 6, got %d", report.NewIndex)
	}

	subtitles, err := service.GetAll("gaps.srt")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(subtitles) != 3 {
		t.Fatalf("expected 3, got %d", len(subtitles))
	}
	if subtitles[2].Index.Value() != 6 {
		t.Errorf("expected index 6")
	}
}

func TestAddInvalidTimestamps(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(5000))
	end, _ := NewSubtitleTimestamp(ms(3000))
	text, _ := NewSubtitleText("Bad")

	if _, err := service.Add("test.srt", start, end, text); err == nil {
		t.Errorf("expected error")
	}
}

func TestAddMultilineText(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(3000))
	end, _ := NewSubtitleTimestamp(ms(5000))
	text, _ := NewSubtitleText("Line 1\nLine 2\nLine 3")

	report, err := service.Add("test.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 2 {
		t.Errorf("expected new index 2, got %d", report.NewIndex)
	}

	subtitle, err := service.GetByID("test.srt", 2)
	if err != nil || subtitle == nil {
		t.Fatalf("GetByID: %v / %v", subtitle, err)
	}
	if subtitle.LineCount() != 3 {
		t.Errorf("expected line count 3, got %d", subtitle.LineCount())
	}
	if subtitle.Text.Value() != "Line 1\nLine 2\nLine 3" {
		t.Errorf("unexpected text: %q", subtitle.Text.Value())
	}
}

func TestAddBackupCreated(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(3000))
	end, _ := NewSubtitleTimestamp(ms(5000))
	text, _ := NewSubtitleText("Second")

	report, err := service.Add("test.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 2 {
		t.Errorf("expected new index 2, got %d", report.NewIndex)
	}
	if report.TotalSubtitles != 2 {
		t.Errorf("expected total 2, got %d", report.TotalSubtitles)
	}
}

func TestAddCreatesNewFile(t *testing.T) {
	service, dir := createTestService(t)

	start, _ := NewSubtitleTimestamp(ms(1000))
	end, _ := NewSubtitleTimestamp(ms(2000))
	text, _ := NewSubtitleText("First subtitle")

	filePath := filepath.Join(dir, "new_file.srt")
	if pathExists(filePath) {
		t.Fatalf("file should not exist yet")
	}

	report, err := service.Add("new_file.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 1 {
		t.Errorf("expected new index 1, got %d", report.NewIndex)
	}
	if report.TotalSubtitles != 1 {
		t.Errorf("expected total 1, got %d", report.TotalSubtitles)
	}

	if !pathExists(filePath) {
		t.Fatalf("file should exist now")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "1\n") {
		t.Errorf("expected content to contain '1\\n'")
	}
	if !strings.Contains(content, "00:00:01,000 --> 00:00:02,000") {
		t.Errorf("expected content to contain timestamp")
	}
	if !strings.Contains(content, "First subtitle") {
		t.Errorf("expected content to contain 'First subtitle'")
	}
}

func TestAddMalformedFile(t *testing.T) {
	service, dir := createTestService(t)
	writeTestFile(t, dir, "bad.srt", "INVALID CONTENT\nNOT SRT FORMAT")

	start, _ := NewSubtitleTimestamp(ms(1000))
	end, _ := NewSubtitleTimestamp(ms(2000))
	text, _ := NewSubtitleText("Text")

	_, err := service.Add("bad.srt", start, end, text)
	if kind, ok := errKind(err); !ok || kind != ErrParse {
		t.Errorf("expected ErrParse, got %v", err)
	}
}

func TestAddTimestampBeforeLast(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(6000)) // 00:00:06,000
	end, _ := NewSubtitleTimestamp(ms(8000))
	text, _ := NewSubtitleText("Too early")

	_, err := service.Add("test.srt", start, end, text)
	var se *SubtitleError
	if !errors.As(err, &se) || se.Kind != ErrTimestampConflict {
		t.Fatalf("expected ErrTimestampConflict, got %v", err)
	}
	expected := "Timestamp conflict: new subtitle (starts at 00:00:06,000) must start at or after the last subtitle ends (at 00:00:07,000)"
	if se.Error() != expected {
		t.Errorf("expected %q, got %q", expected, se.Error())
	}
}

func TestAddTimestampOverlapping(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:10,000 --> 00:00:15,000\nFirst"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(14000))
	end, _ := NewSubtitleTimestamp(ms(16000))
	text, _ := NewSubtitleText("Overlapping")

	_, err := service.Add("test.srt", start, end, text)
	if kind, ok := errKind(err); !ok || kind != ErrTimestampConflict {
		t.Errorf("expected ErrTimestampConflict, got %v", err)
	}
}

func TestAddTimestampExactlyAfter(t *testing.T) {
	service, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond"
	writeTestFile(t, dir, "test.srt", content)

	start, _ := NewSubtitleTimestamp(ms(7000)) // 00:00:07,000
	end, _ := NewSubtitleTimestamp(ms(9000))
	text, _ := NewSubtitleText("Exactly after")

	report, err := service.Add("test.srt", start, end, text)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if report.NewIndex != 3 {
		t.Errorf("expected new index 3, got %d", report.NewIndex)
	}
	if report.TotalSubtitles != 3 {
		t.Errorf("expected total 3, got %d", report.TotalSubtitles)
	}
}
