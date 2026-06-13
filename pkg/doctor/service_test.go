package doctor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// createTestService creates a test service backed by t.TempDir().
func createTestService(t *testing.T) (*SubRipDoctorService, string) {
	t.Helper()
	dir := t.TempDir()
	return NewSubRipDoctorService(dir), dir
}

// writeTestFile writes a test file into dir.
func writeTestFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func doctorErrKind(err error) (DoctorErrorKind, bool) {
	var de *DoctorError
	if errors.As(err, &de) {
		return de.Kind, true
	}
	return 0, false
}

// --- parse_timestamp ---

func TestParseTimestampValid(t *testing.T) {
	svc := NewSubRipDoctorService("")

	cases := []struct {
		in   string
		want uint64
	}{
		{"00:00:00,000", 0},
		{"00:00:01,000", 1000},
		{"00:00:01,436", 1436},
		{"01:02:03,004", 1*3600000 + 2*60000 + 3*1000 + 4},
		{"23:59:59,999", 23*3600000 + 59*60000 + 59*1000 + 999},
	}

	for _, c := range cases {
		got, err := svc.parseTimestamp(c.in)
		if err != nil {
			t.Fatalf("parseTimestamp(%q) unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseTimestamp(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseTimestampWrongPartCount(t *testing.T) {
	svc := NewSubRipDoctorService("")
	_, err := svc.parseTimestamp("00:00:01")
	if err == nil {
		t.Fatal("expected error")
	}
	want := "expected format HH:MM:SS,mmm, got 00:00:01"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestParseTimestampInvalidFields(t *testing.T) {
	svc := NewSubRipDoctorService("")

	cases := []struct {
		in   string
		want string
	}{
		{"xx:00:01,000", "invalid hours: xx"},
		{"00:yy:01,000", "invalid minutes: yy"},
		{"00:00:zz,000", "invalid seconds: zz"},
		{"00:00:01,www", "invalid milliseconds: www"},
		{"00:60:01,000", "minutes must be < 60, got 60"},
		{"00:00:60,000", "seconds must be < 60, got 60"},
		{"00:00:01,1000", "milliseconds must be < 1000, got 1000"},
	}

	for _, c := range cases {
		_, err := svc.parseTimestamp(c.in)
		if err == nil {
			t.Fatalf("parseTimestamp(%q) expected error", c.in)
		}
		if err.Error() != c.want {
			t.Errorf("parseTimestamp(%q) = %q, want %q", c.in, err.Error(), c.want)
		}
	}
}

// --- validate_filename / get_file_path ---

func TestDiagnoseEmptyFilename(t *testing.T) {
	svc, _ := createTestService(t)
	_, err := svc.Diagnose("")
	if err == nil {
		t.Fatal("expected error")
	}
	kind, ok := doctorErrKind(err)
	if !ok || kind != ErrInvalidPath {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
	if err.Error() != "invalid file path: filename cannot be empty" {
		t.Errorf("unexpected message: %q", err.Error())
	}
}

func TestDiagnosePathTraversal(t *testing.T) {
	svc, _ := createTestService(t)
	for _, name := range []string{"../etc/passwd", "sub/dir.srt", "a\\b.srt"} {
		_, err := svc.Diagnose(name)
		if err == nil {
			t.Fatalf("expected error for %q", name)
		}
		kind, ok := doctorErrKind(err)
		if !ok || kind != ErrInvalidPath {
			t.Fatalf("expected ErrInvalidPath for %q, got %v", name, err)
		}
	}
}

func TestDiagnoseFileNotFound(t *testing.T) {
	svc, _ := createTestService(t)
	_, err := svc.Diagnose("missing.srt")
	if err == nil {
		t.Fatal("expected error")
	}
	kind, ok := doctorErrKind(err)
	if !ok || kind != ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

// --- diagnose: clean file ---

func TestDiagnoseValidFile(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:01,436 --> 00:00:03,481\nHello, Harry.\n\n2\n00:00:03,481 --> 00:00:05,135\nGoodbye.\n"
	writeTestFile(t, dir, "valid.srt", content)

	report, err := svc.Diagnose("valid.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if report.HasIssues() {
		t.Errorf("expected no issues, got %d: %v", len(report.Issues), report.Issues)
	}
	if report.TotalBlocks != 2 {
		t.Errorf("expected 2 blocks, got %d", report.TotalBlocks)
	}
	if !report.IsParsable {
		t.Error("expected parsable")
	}
	if report.ErrorCount() != 0 || report.WarningCount() != 0 {
		t.Errorf("expected 0 errors/0 warnings, got %d/%d", report.ErrorCount(), report.WarningCount())
	}
}

// --- diagnose: multiple empty lines (warning) ---

func TestDiagnoseMultipleEmptyLines(t *testing.T) {
	svc, dir := createTestService(t)
	// Three blank lines between block 1 and block 2.
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst.\n\n\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond.\n"
	writeTestFile(t, dir, "warn.srt", content)

	report, err := svc.Diagnose("warn.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if report.WarningCount() != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", report.WarningCount(), report.Issues)
	}
	var found *ValidationIssue
	for i := range report.Issues {
		if report.Issues[i].IssueType.Kind == MultipleEmptyLines {
			found = &report.Issues[i]
		}
	}
	if found == nil {
		t.Fatal("expected a MultipleEmptyLines issue")
	}
	if found.Severity != Warning {
		t.Errorf("expected Warning severity")
	}
	// The first of the three blanks is consumed as the end-of-text separator
	// in the ExpectingText state; the remaining two blanks are counted between
	// blocks, starting at line 5.
	if found.IssueType.Count != 2 {
		t.Errorf("expected count 2, got %d", found.IssueType.Count)
	}
	if found.LineNumber != 5 {
		t.Errorf("expected line 5 (first counted blank), got %d", found.LineNumber)
	}
	if found.BlockNumber != nil {
		t.Errorf("expected nil block number")
	}
	if found.Context == nil || *found.Context != "lines 5-6" {
		t.Errorf("unexpected context: %v", deref(found.Context))
	}
}

func TestDiagnoseTrailingEmptyLines(t *testing.T) {
	svc, dir := createTestService(t)
	// Three trailing blank lines after the last block. The first blank ends the
	// text block (ExpectingText -> betweenBlocks); the remaining two are counted
	// as trailing empties and, being > 1, trigger the trailing warning.
	content := "1\n00:00:01,000 --> 00:00:02,000\nOnly.\n\n\n\n"
	writeTestFile(t, dir, "trail.srt", content)

	report, err := svc.Diagnose("trail.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if report.WarningCount() != 1 {
		t.Fatalf("expected 1 trailing warning, got %d: %v", report.WarningCount(), report.Issues)
	}
	issue := report.Issues[len(report.Issues)-1]
	if issue.Context == nil || *issue.Context != "trailing empty lines" {
		t.Errorf("unexpected context: %v", deref(issue.Context))
	}
	if issue.IssueType.Count != 2 {
		t.Errorf("expected count 2, got %d", issue.IssueType.Count)
	}
}

// --- diagnose: invalid index ---

func TestDiagnoseInvalidIndexNotNumber(t *testing.T) {
	svc, dir := createTestService(t)
	content := "abc\n00:00:01,000 --> 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "badidx.srt", content)

	report, err := svc.Diagnose("badidx.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if report.ErrorCount() < 1 {
		t.Fatalf("expected at least 1 error, got %d", report.ErrorCount())
	}
	issue := report.Issues[0]
	if issue.IssueType.Kind != InvalidIndex {
		t.Fatalf("expected InvalidIndex, got %v", issue.IssueType.Kind)
	}
	if issue.IssueType.Reason != "not a valid number" {
		t.Errorf("unexpected reason: %q", issue.IssueType.Reason)
	}
	if issue.IssueType.Value != "abc" {
		t.Errorf("unexpected value: %q", issue.IssueType.Value)
	}
	if issue.BlockNumber == nil || *issue.BlockNumber != 1 {
		t.Errorf("expected block 1")
	}
	if issue.Severity != Error {
		t.Errorf("expected Error severity")
	}
}

func TestDiagnoseInvalidIndexZero(t *testing.T) {
	svc, dir := createTestService(t)
	content := "0\n00:00:01,000 --> 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "zeroidx.srt", content)

	report, err := svc.Diagnose("zeroidx.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := report.Issues[0]
	if issue.IssueType.Kind != InvalidIndex {
		t.Fatalf("expected InvalidIndex, got %v", issue.IssueType.Kind)
	}
	if issue.IssueType.Reason != "must be >= 1" {
		t.Errorf("unexpected reason: %q", issue.IssueType.Reason)
	}
}

// --- diagnose: malformed timestamp ---

func TestDiagnoseMissingSeparator(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:01,000 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "sep.srt", content)

	report, err := svc.Diagnose("sep.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := findKind(report.Issues, MalformedTimestamp)
	if issue == nil {
		t.Fatal("expected MalformedTimestamp")
	}
	if issue.IssueType.Reason != "missing ' --> ' separator" {
		t.Errorf("unexpected reason: %q", issue.IssueType.Reason)
	}
}

func TestDiagnoseMalformedStartTimestamp(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n99:99:99,999 --> 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "badstart.srt", content)

	report, err := svc.Diagnose("badstart.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := findKind(report.Issues, MalformedTimestamp)
	if issue == nil {
		t.Fatal("expected MalformedTimestamp")
	}
	if issue.IssueType.Value != "99:99:99,999" {
		t.Errorf("expected value of start part, got %q", issue.IssueType.Value)
	}
	if issue.IssueType.Reason != "minutes must be < 60, got 99" {
		t.Errorf("unexpected reason: %q", issue.IssueType.Reason)
	}
}

func TestDiagnoseMalformedEndTimestamp(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:99,000\nText.\n"
	writeTestFile(t, dir, "badend.srt", content)

	report, err := svc.Diagnose("badend.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := findKind(report.Issues, MalformedTimestamp)
	if issue == nil {
		t.Fatal("expected MalformedTimestamp")
	}
	if issue.IssueType.Value != "00:00:99,000" {
		t.Errorf("expected value of end part, got %q", issue.IssueType.Value)
	}
	if issue.IssueType.Reason != "seconds must be < 60, got 99" {
		t.Errorf("unexpected reason: %q", issue.IssueType.Reason)
	}
}

// --- diagnose: invalid timestamp (end <= start) ---

func TestDiagnoseInvalidTimestampOrder(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:05,000 --> 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "order.srt", content)

	report, err := svc.Diagnose("order.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := findKind(report.Issues, InvalidTimestamp)
	if issue == nil {
		t.Fatal("expected InvalidTimestamp")
	}
	if issue.IssueType.Start != "00:00:05,000" || issue.IssueType.End != "00:00:02,000" {
		t.Errorf("unexpected start/end: %q / %q", issue.IssueType.Start, issue.IssueType.End)
	}
	want := "Invalid timestamp: end (00:00:02,000) <= start (00:00:05,000)"
	if issue.IssueType.String() != want {
		t.Errorf("display = %q, want %q", issue.IssueType.String(), want)
	}
}

func TestDiagnoseEqualTimestamps(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:02,000 --> 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "equal.srt", content)

	report, err := svc.Diagnose("equal.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	if findKind(report.Issues, InvalidTimestamp) == nil {
		t.Fatal("expected InvalidTimestamp for equal start/end")
	}
}

// --- diagnose: empty line where timestamp expected ---

func TestDiagnoseEmptyLineInBlock(t *testing.T) {
	svc, dir := createTestService(t)
	// Blank line right after the index, where a timestamp is expected.
	content := "1\n\n00:00:01,000 --> 00:00:02,000\nText.\n"
	writeTestFile(t, dir, "emptyblock.srt", content)

	report, err := svc.Diagnose("emptyblock.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := findKind(report.Issues, EmptyLineInBlock)
	if issue == nil {
		t.Fatal("expected EmptyLineInBlock")
	}
	if issue.Severity != Error {
		t.Errorf("expected Error severity")
	}
	if issue.Context == nil || *issue.Context != "expected timestamp line" {
		t.Errorf("unexpected context: %v", issue.Context)
	}
	if issue.IssueType.String() != "Empty line inside block" {
		t.Errorf("unexpected display: %q", issue.IssueType.String())
	}
}

// --- diagnose: empty text ---

func TestDiagnoseEmptyText(t *testing.T) {
	svc, dir := createTestService(t)
	// Index, timestamp, then an empty line before any text.
	content := "1\n00:00:01,000 --> 00:00:02,000\n\n2\n00:00:03,000 --> 00:00:04,000\nText.\n"
	writeTestFile(t, dir, "emptytext.srt", content)

	report, err := svc.Diagnose("emptytext.srt")
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	issue := findKind(report.Issues, EmptyText)
	if issue == nil {
		t.Fatal("expected EmptyText")
	}
	if issue.Context == nil || *issue.Context != "no text before empty line" {
		t.Errorf("unexpected context: %v", issue.Context)
	}
}

// --- fix ---

func TestFixRemovesEmptyLineInBlock(t *testing.T) {
	svc, dir := createTestService(t)
	// Empty line between index and timestamp should be removed.
	content := "1\n\n00:00:01,000 --> 00:00:02,000\nHello.\n"
	writeTestFile(t, dir, "fix1.srt", content)

	report, err := svc.Fix("fix1.srt")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if report.IssuesFixed != 1 {
		t.Errorf("expected 1 issue fixed, got %d", report.IssuesFixed)
	}
	if !report.ValidationSuccess {
		t.Errorf("expected validation success")
	}
	got := readFile(t, dir, "fix1.srt")
	want := "1\n00:00:01,000 --> 00:00:02,000\nHello."
	if got != want {
		t.Errorf("fixed content = %q, want %q", got, want)
	}
}

func TestFixNormalizesMultipleEmptyLines(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst.\n\n\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond.\n"
	writeTestFile(t, dir, "fix2.srt", content)

	report, err := svc.Fix("fix2.srt")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if report.IssuesFixed != 1 {
		t.Errorf("expected 1 issue fixed, got %d", report.IssuesFixed)
	}
	got := readFile(t, dir, "fix2.srt")
	want := "1\n00:00:01,000 --> 00:00:02,000\nFirst.\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond."
	if got != want {
		t.Errorf("fixed content = %q, want %q", got, want)
	}
	if !report.ValidationSuccess {
		t.Errorf("expected validation success")
	}
}

func TestFixCleanFileNoChange(t *testing.T) {
	svc, dir := createTestService(t)
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst.\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond.\n"
	writeTestFile(t, dir, "fix3.srt", content)

	report, err := svc.Fix("fix3.srt")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if report.IssuesFixed != 0 {
		t.Errorf("expected 0 issues fixed, got %d", report.IssuesFixed)
	}
	got := readFile(t, dir, "fix3.srt")
	// The terminating newline is dropped because lines() consumes it and the
	// output is re-joined with "\n".
	want := "1\n00:00:01,000 --> 00:00:02,000\nFirst.\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond."
	if got != want {
		t.Errorf("fixed content = %q, want %q", got, want)
	}
}

func TestFixReportPaths(t *testing.T) {
	svc, dir := createTestService(t)
	writeTestFile(t, dir, "paths.srt", "1\n00:00:01,000 --> 00:00:02,000\nHi.\n")

	report, err := svc.Fix("paths.srt")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	expected := filepath.Join(dir, "paths.srt")
	if report.OriginalPath != expected || report.FixedPath != expected {
		t.Errorf("paths = %q / %q, want %q", report.OriginalPath, report.FixedPath, expected)
	}
	if report.IssuesUnfixable != 0 {
		t.Errorf("expected 0 unfixable, got %d", report.IssuesUnfixable)
	}
	if len(report.UnfixableIssues) != 0 {
		t.Errorf("expected empty unfixable issues")
	}
}

func TestFixFileNotFound(t *testing.T) {
	svc, _ := createTestService(t)
	_, err := svc.Fix("nope.srt")
	if err == nil {
		t.Fatal("expected error")
	}
	kind, ok := doctorErrKind(err)
	if !ok || kind != ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

// --- DiagnosticReport helpers ---

func TestDiagnosticReportCounts(t *testing.T) {
	blk := 1
	report := DiagnosticReport{
		Issues: []ValidationIssue{
			{Severity: Error, BlockNumber: &blk, IssueType: IssueType{Kind: EmptyText}},
			{Severity: Warning, IssueType: IssueType{Kind: MultipleEmptyLines, Count: 2}},
			{Severity: Error, IssueType: IssueType{Kind: EmptyText}},
		},
	}
	if report.ErrorCount() != 2 {
		t.Errorf("ErrorCount = %d, want 2", report.ErrorCount())
	}
	if report.WarningCount() != 1 {
		t.Errorf("WarningCount = %d, want 1", report.WarningCount())
	}
	if !report.HasIssues() {
		t.Error("HasIssues = false, want true")
	}
	if !report.HasErrors() {
		t.Error("HasErrors = false, want true")
	}

	empty := DiagnosticReport{}
	if empty.HasIssues() {
		t.Error("empty.HasIssues = true, want false")
	}
	if empty.HasErrors() {
		t.Error("empty.HasErrors = true, want false")
	}
}

// --- model Display strings ---

func TestIssueTypeDisplay(t *testing.T) {
	cases := []struct {
		it   IssueType
		want string
	}{
		{IssueType{Kind: EmptyLineInBlock}, "Empty line inside block"},
		{IssueType{Kind: MultipleEmptyLines, Count: 3}, "Multiple empty lines (3 consecutive)"},
		{IssueType{Kind: InvalidTimestamp, Start: "00:00:05,000", End: "00:00:02,000"}, "Invalid timestamp: end (00:00:02,000) <= start (00:00:05,000)"},
		{IssueType{Kind: MissingField, Field: "index"}, "Missing field: index"},
		{IssueType{Kind: InsufficientLines, Found: 2}, "Insufficient lines (found 2, need at least 3)"},
		{IssueType{Kind: InvalidIndex, Value: "abc", Reason: "not a valid number"}, "Invalid index 'abc': not a valid number"},
		{IssueType{Kind: MalformedTimestamp, Value: "xx", Reason: "invalid hours: xx"}, "Malformed timestamp 'xx': invalid hours: xx"},
		{IssueType{Kind: EmptyText}, "Empty or whitespace-only text"},
	}
	for _, c := range cases {
		if got := c.it.String(); got != c.want {
			t.Errorf("IssueType{%d}.String() = %q, want %q", c.it.Kind, got, c.want)
		}
	}
}

func TestValidationIssueDisplay(t *testing.T) {
	blk := 2
	withBlock := ValidationIssue{
		LineNumber:  10,
		BlockNumber: &blk,
		IssueType:   IssueType{Kind: EmptyText},
		Severity:    Error,
	}
	wantWithBlock := "❌ Line 10, Block 2: Empty or whitespace-only text"
	if got := withBlock.String(); got != wantWithBlock {
		t.Errorf("with block: got %q, want %q", got, wantWithBlock)
	}

	noBlock := ValidationIssue{
		LineNumber: 4,
		IssueType:  IssueType{Kind: MultipleEmptyLines, Count: 3},
		Severity:   Warning,
	}
	wantNoBlock := "⚠️ Line 4: Multiple empty lines (3 consecutive)"
	if got := noBlock.String(); got != wantNoBlock {
		t.Errorf("no block: got %q, want %q", got, wantNoBlock)
	}
}

func TestDoctorErrorMessages(t *testing.T) {
	cases := []struct {
		err  *DoctorError
		want string
	}{
		{NewFileNotFound("/a/b.srt"), "file not found: /a/b.srt"},
		{NewInvalidPath("filename cannot be empty"), "invalid file path: filename cannot be empty"},
		{NewBackupFailed("disk full"), "backup creation failed: disk full"},
		{NewValidationFailed("still broken"), "failed to validate fixed file: still broken"},
	}
	for _, c := range cases {
		if c.err.Error() != c.want {
			t.Errorf("got %q, want %q", c.err.Error(), c.want)
		}
	}

	ioErr := NewIoError(errors.New("boom"))
	if ioErr.Error() != "i/o error: boom" {
		t.Errorf("io error message: %q", ioErr.Error())
	}
	if !errors.Is(ioErr, ioErr.Wrapped) {
		t.Error("expected IoError to unwrap to its source")
	}
}

// --- helpers ---

func findKind(issues []ValidationIssue, kind IssueTypeKind) *ValidationIssue {
	for i := range issues {
		if issues[i].IssueType.Kind == kind {
			return &issues[i]
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(b)
}
