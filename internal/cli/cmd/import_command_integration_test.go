package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// runImport invokes HandleImport with CSV format and --force, capturing
// stdout/stderr. Mirrors the Rust `run_import_command` helper which shells out
// to `sm import --format csv --delimiter <d> --force <srt> <csv>`.
func runImport(t *testing.T, srtFile, csvFile, delimiter string) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return HandleImport(
			srtFile,
			csvFile,
			cli.ImportFormatCsv,
			nil,       // reference
			delimiter, // delimiter
			false,     // dryRun
			true,      // force
			cli.OutputFormatText,
		)
	})
}

// runImportDryRun invokes HandleImport with CSV format and --dry-run (no
// --force), capturing stdout/stderr. Mirrors the Rust dry-run invocations.
func runImportDryRun(t *testing.T, srtFile, csvFile, delimiter string) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return HandleImport(
			srtFile,
			csvFile,
			cli.ImportFormatCsv,
			nil,       // reference
			delimiter, // delimiter
			true,      // dryRun
			false,     // force
			cli.OutputFormatText,
		)
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func createTempCsv(t *testing.T, dir, filename, content string) string {
	t.Helper()
	p := filepath.Join(dir, filename)
	writeFile(t, p, content)
	return p
}

func createEmptySrt(t *testing.T, dir, filename string) string {
	t.Helper()
	p := filepath.Join(dir, filename)
	writeFile(t, p, "")
	return p
}

func createSrtWithContent(t *testing.T, dir, filename, content string) string {
	t.Helper()
	p := filepath.Join(dir, filename)
	writeFile(t, p, content)
	return p
}

// ========== Mandatory Tests (User Requirements) ==========

// TestIntegrationImportCreatesNewSrtFile ports `test_import_creates_new_srt_file`.
func TestIntegrationImportCreatesNewSrtFile(t *testing.T) {
	dir := chdirTemp(t)

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000|First subtitle\n00:00:04,000|00:00:06,000|Second subtitle"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)
	srtFile := createEmptySrt(t, dir, "new_file.srt")

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("Command should succeed. stderr: %s stdout: %s", out.stderr, out.stdout)
	}

	got, err := os.ReadFile(srtFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)
	if !strings.Contains(content, "1\n") {
		t.Errorf("file missing '1\\n': %q", content)
	}
	if !strings.Contains(content, "00:00:01,000 --> 00:00:03,000") {
		t.Errorf("file missing first timestamps: %q", content)
	}
	if !strings.Contains(content, "First subtitle") {
		t.Errorf("file missing 'First subtitle': %q", content)
	}
	if !strings.Contains(content, "2\n") {
		t.Errorf("file missing '2\\n': %q", content)
	}
	if !strings.Contains(content, "00:00:04,000 --> 00:00:06,000") {
		t.Errorf("file missing second timestamps: %q", content)
	}
	if !strings.Contains(content, "Second subtitle") {
		t.Errorf("file missing 'Second subtitle': %q", content)
	}

	if !strings.Contains(out.stdout, "Imported: 2 subtitles") {
		t.Errorf("stdout missing 'Imported: 2 subtitles': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "✓ Subtitles imported successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
}

// TestIntegrationMultipleImportsAccumulateData ports `test_multiple_imports_accumulate_data`.
func TestIntegrationMultipleImportsAccumulateData(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "accumulated.srt")

	csv1 := "start_time|end_time|text\n00:00:01,000|00:00:02,000|First\n00:00:03,000|00:00:04,000|Second"
	csv1File := createTempCsv(t, dir, "import1.csv", csv1)

	out1 := runImport(t, srtFile, csv1File, "|")
	if !out1.success {
		t.Fatalf("import 1 failed: %s", out1.stderr)
	}
	if !strings.Contains(out1.stdout, "Imported: 2 subtitles") {
		t.Errorf("out1 missing 'Imported: 2 subtitles': %q", out1.stdout)
	}
	if !strings.Contains(out1.stdout, "Index range: 1-2") {
		t.Errorf("out1 missing 'Index range: 1-2': %q", out1.stdout)
	}
	if !strings.Contains(out1.stdout, "Total subtitles: 2") {
		t.Errorf("out1 missing 'Total subtitles: 2': %q", out1.stdout)
	}

	content, _ := os.ReadFile(srtFile)
	if !strings.Contains(string(content), "First") || !strings.Contains(string(content), "Second") {
		t.Errorf("file missing first import content: %q", string(content))
	}

	csv2 := "start_time|end_time|text\n00:00:05,000|00:00:06,000|Third\n00:00:07,000|00:00:08,000|Fourth\n00:00:09,000|00:00:10,000|Fifth"
	csv2File := createTempCsv(t, dir, "import2.csv", csv2)

	out2 := runImport(t, srtFile, csv2File, "|")
	if !out2.success {
		t.Fatalf("import 2 failed: %s", out2.stderr)
	}
	if !strings.Contains(out2.stdout, "Imported: 3 subtitles") {
		t.Errorf("out2 missing 'Imported: 3 subtitles': %q", out2.stdout)
	}
	if !strings.Contains(out2.stdout, "Index range: 3-5") {
		t.Errorf("out2 missing 'Index range: 3-5': %q", out2.stdout)
	}
	if !strings.Contains(out2.stdout, "Total subtitles: 5") {
		t.Errorf("out2 missing 'Total subtitles: 5': %q", out2.stdout)
	}

	content, _ = os.ReadFile(srtFile)
	for _, want := range []string{"First", "Second", "Third", "Fourth", "Fifth"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("file missing %q: %q", want, string(content))
		}
	}

	csv3 := "start_time|end_time|text\n00:00:11,000|00:00:13,000|Sixth"
	csv3File := createTempCsv(t, dir, "import3.csv", csv3)

	out3 := runImport(t, srtFile, csv3File, "|")
	if !out3.success {
		t.Fatalf("import 3 failed: %s", out3.stderr)
	}
	if !strings.Contains(out3.stdout, "Imported: 1 subtitles") {
		t.Errorf("out3 missing 'Imported: 1 subtitles': %q", out3.stdout)
	}
	if !strings.Contains(out3.stdout, "Index range: 6-6") {
		t.Errorf("out3 missing 'Index range: 6-6': %q", out3.stdout)
	}
	if !strings.Contains(out3.stdout, "Total subtitles: 6") {
		t.Errorf("out3 missing 'Total subtitles: 6': %q", out3.stdout)
	}

	content, _ = os.ReadFile(srtFile)
	for _, want := range []string{"First", "Second", "Third", "Fourth", "Fifth", "Sixth"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("final file missing %q: %q", want, string(content))
		}
	}
}

// ========== Basic Functionality Tests ==========

// TestIntegrationImportWithPipeDelimiter ports `test_import_with_pipe_delimiter`.
func TestIntegrationImportWithPipeDelimiter(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvFile := testDataPath(t, "sample-import.csv")

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Imported: 4 subtitles") {
		t.Errorf("stdout missing 'Imported: 4 subtitles': %q", out.stdout)
	}

	content, _ := os.ReadFile(srtFile)
	c := string(content)
	if !strings.Contains(c, "<i>Previously on") {
		t.Errorf("file missing html open: %q", c)
	}
	if !strings.Contains(c, "\"Resident Alien\"...</i>") {
		t.Errorf("file missing html close: %q", c)
	}
	if !strings.Contains(c, "Hello, Harry.") {
		t.Errorf("file missing 'Hello, Harry.': %q", c)
	}
	if !strings.Contains(c, "My name is Joseph.") {
		t.Errorf("file missing 'My name is Joseph.': %q", c)
	}
}

// TestIntegrationImportWithSemicolonDelimiter ports `test_import_with_semicolon_delimiter`.
func TestIntegrationImportWithSemicolonDelimiter(t *testing.T) {
	dir := chdirTemp(t)

	csvContent := "start_time;end_time;text\n00:00:01,000;00:00:03,000;First subtitle\n00:00:04,000;00:00:06,000;Second subtitle"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)
	srtFile := createEmptySrt(t, dir, "test.srt")

	out := runImport(t, srtFile, csvFile, ";")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Imported: 2 subtitles") {
		t.Errorf("stdout missing 'Imported: 2 subtitles': %q", out.stdout)
	}
}

// TestIntegrationImportToExistingFileWithContent ports `test_import_to_existing_file_with_content`.
func TestIntegrationImportToExistingFileWithContent(t *testing.T) {
	dir := chdirTemp(t)

	existing := "1\n00:00:01,000 --> 00:00:02,000\nExisting first\n\n2\n00:00:03,000 --> 00:00:04,000\nExisting second\n"
	srtFile := createSrtWithContent(t, dir, "existing.srt", existing)

	csvContent := "start_time|end_time|text\n00:00:05,000|00:00:06,000|New third\n00:00:07,000|00:00:08,000|New fourth"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Imported: 2 subtitles") {
		t.Errorf("stdout missing 'Imported: 2 subtitles': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Index range: 3-4") {
		t.Errorf("stdout missing 'Index range: 3-4': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 4") {
		t.Errorf("stdout missing 'Total subtitles: 4': %q", out.stdout)
	}

	content, _ := os.ReadFile(srtFile)
	for _, want := range []string{"Existing first", "Existing second", "New third", "New fourth"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("file missing %q: %q", want, string(content))
		}
	}
}

// ========== Delimiter Validation Tests ==========

// TestIntegrationImportInvalidDelimiterEmpty ports `test_import_invalid_delimiter_empty`.
func TestIntegrationImportInvalidDelimiterEmpty(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")
	csvFile := createTempCsv(t, dir, "test.csv", "header\ndata")

	out := runImport(t, srtFile, csvFile, "")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "must be a single character") {
		t.Errorf("stderr missing 'must be a single character': %q", out.stderr)
	}
}

// TestIntegrationImportInvalidDelimiterMultipleChars ports
// `test_import_invalid_delimiter_multiple_chars`.
func TestIntegrationImportInvalidDelimiterMultipleChars(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")
	csvFile := createTempCsv(t, dir, "test.csv", "header\ndata")

	out := runImport(t, srtFile, csvFile, "||")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "must be a single character") {
		t.Errorf("stderr missing 'must be a single character': %q", out.stderr)
	}
}

// ========== CSV Format Validation Tests ==========

// TestIntegrationImportCsvInvalidHeader ports `test_import_csv_invalid_header`.
func TestIntegrationImportCsvInvalidHeader(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "wrong|header|format\n00:00:01,000|00:00:03,000|Text"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Invalid CSV header") {
		t.Errorf("stderr missing 'Invalid CSV header': %q", out.stderr)
	}
}

// TestIntegrationImportCsvWrongColumnCount ports `test_import_csv_wrong_column_count`.
func TestIntegrationImportCsvWrongColumnCount(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000" // Missing text column
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(strings.ToLower(out.stderr), "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
}

// TestIntegrationImportCsvEmptyFile ports `test_import_csv_empty_file`.
func TestIntegrationImportCsvEmptyFile(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "start_time|end_time|text" // Only header, no data
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "contains no data rows") {
		t.Errorf("stderr missing 'contains no data rows': %q", out.stderr)
	}
}

// ========== Timestamp Validation Tests ==========

// TestIntegrationImportInvalidTimestampFormat ports `test_import_invalid_timestamp_format`.
func TestIntegrationImportInvalidTimestampFormat(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "start_time|end_time|text\n99:99:99,999|00:00:03,000|Invalid timestamp"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "timestamp") && !strings.Contains(out.stderr, "line") {
		t.Errorf("stderr missing 'timestamp'/'line': %q", out.stderr)
	}
}

// TestIntegrationImportEndBeforeStart ports `test_import_end_before_start`.
func TestIntegrationImportEndBeforeStart(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "start_time|end_time|text\n00:00:05,000|00:00:03,000|End before start"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(strings.ToLower(out.stderr), "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
}

// TestIntegrationImportTimestampConflictWithExisting ports
// `test_import_timestamp_conflict_with_existing`.
func TestIntegrationImportTimestampConflictWithExisting(t *testing.T) {
	dir := chdirTemp(t)

	existing := "1\n00:00:05,000 --> 00:00:10,000\nExisting subtitle\n"
	srtFile := createSrtWithContent(t, dir, "test.srt", existing)

	csvContent := "start_time|end_time|text\n00:00:08,000|00:00:12,000|Overlapping subtitle"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Timestamp conflict") {
		t.Errorf("stderr missing 'Timestamp conflict': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "00:00:10,000") { // Last ends at
		t.Errorf("stderr missing '00:00:10,000': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "00:00:08,000") { // New starts at
		t.Errorf("stderr missing '00:00:08,000': %q", out.stderr)
	}
}

// TestIntegrationImportTimestampExactlyAfterLast ports
// `test_import_timestamp_exactly_after_last`.
func TestIntegrationImportTimestampExactlyAfterLast(t *testing.T) {
	dir := chdirTemp(t)

	existing := "1\n00:00:05,000 --> 00:00:10,000\nExisting subtitle\n"
	srtFile := createSrtWithContent(t, dir, "test.srt", existing)

	csvContent := "start_time|end_time|text\n00:00:10,000|00:00:12,000|Exactly after"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitles imported successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
}

// ========== Backup Tests ==========

// TestIntegrationImportCreatesBackup ports `test_import_creates_backup`.
func TestIntegrationImportCreatesBackup(t *testing.T) {
	dir := chdirTemp(t)

	original := "1\n00:00:01,000 --> 00:00:02,000\nOriginal content\n"
	srtFile := createSrtWithContent(t, dir, "test.srt", original)

	csvContent := "start_time|end_time|text\n00:00:03,000|00:00:04,000|New content"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Backup:") {
		t.Errorf("stdout missing 'Backup:': %q", out.stdout)
	}

	var backupLine string
	for _, line := range strings.Split(out.stdout, "\n") {
		if strings.Contains(line, "Backup:") {
			backupLine = line
			break
		}
	}
	if backupLine == "" {
		t.Fatalf("Backup line not found in: %q", out.stdout)
	}

	backupPath := strings.TrimSpace(strings.ReplaceAll(backupLine, "Backup:", ""))
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("Backup file should exist at: %s (%v)", backupPath, err)
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(backupContent), "Original content") {
		t.Errorf("backup missing 'Original content': %q", string(backupContent))
	}
	if strings.Contains(string(backupContent), "New content") {
		t.Errorf("backup should not contain 'New content': %q", string(backupContent))
	}
}

// TestIntegrationImportToEmptySrtNoBackupError ports
// `test_import_to_empty_srt_no_backup_error`.
func TestIntegrationImportToEmptySrtNoBackupError(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "empty.srt")

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000|First subtitle"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("Import to empty file should succeed. stderr: %s", out.stderr)
	}
}

// ========== Path Traversal Tests ==========

// TestIntegrationImportPathTraversalSrtRejected ports
// `test_import_path_traversal_srt_rejected`.
func TestIntegrationImportPathTraversalSrtRejected(t *testing.T) {
	dir := chdirTemp(t)
	csvFile := createTempCsv(t, dir, "test.csv", "start_time|end_time|text")

	out := runImport(t, "../../../etc/passwd", csvFile, "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "path traversal") && !strings.Contains(out.stderr, "failed to resolve") {
		t.Errorf("Should reject path traversal attempt. stderr was: %s", out.stderr)
	}
}

// TestIntegrationImportPathTraversalCsvRejected ports
// `test_import_path_traversal_csv_rejected`.
func TestIntegrationImportPathTraversalCsvRejected(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	out := runImport(t, srtFile, "../../../etc/passwd", "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "path traversal") && !strings.Contains(out.stderr, "failed to resolve") {
		t.Errorf("Should reject path traversal attempt. stderr was: %s", out.stderr)
	}
}

// ========== File Handling Tests ==========

// TestIntegrationImportCsvFileNotFound ports `test_import_csv_file_not_found`.
func TestIntegrationImportCsvFileNotFound(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	out := runImport(t, srtFile, "/tmp/nonexistent_csv_12345.csv", "|")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "failed to resolve file path") {
		t.Errorf("stderr missing 'failed to resolve file path': %q", out.stderr)
	}
}

// TestIntegrationImportMultilineTextConversion ports
// `test_import_multiline_text_conversion`.
func TestIntegrationImportMultilineTextConversion(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	// CSV with \n literal that should be converted to actual newlines.
	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000|Line one\\nLine two\\nLine three"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	content, _ := os.ReadFile(srtFile)
	if !strings.Contains(string(content), "Line one\nLine two\nLine three") {
		t.Errorf("file missing converted newlines: %q", string(content))
	}
}

// TestIntegrationImportPreservesHtmlTags ports `test_import_preserves_html_tags`.
func TestIntegrationImportPreservesHtmlTags(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000|<i>Italic text</i>\n00:00:04,000|00:00:06,000|<b>Bold text</b>"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	content, _ := os.ReadFile(srtFile)
	if !strings.Contains(string(content), "<i>Italic text</i>") {
		t.Errorf("file missing italic tag: %q", string(content))
	}
	if !strings.Contains(string(content), "<b>Bold text</b>") {
		t.Errorf("file missing bold tag: %q", string(content))
	}
}

// ========== Output and Summary Tests ==========

// TestIntegrationImportOutputShowsSummary ports `test_import_output_shows_summary`.
func TestIntegrationImportOutputShowsSummary(t *testing.T) {
	dir := chdirTemp(t)
	srtFile := createEmptySrt(t, dir, "test.srt")

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:02,000|First\n00:00:03,000|00:00:04,000|Second\n00:00:05,000|00:00:06,000|Third"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitles imported successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Imported: 3 subtitles") {
		t.Errorf("stdout missing 'Imported: 3 subtitles': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Index range: 1-3") {
		t.Errorf("stdout missing 'Index range: 1-3': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 3") {
		t.Errorf("stdout missing 'Total subtitles: 3': %q", out.stdout)
	}
}

// TestIntegrationImportIndexRangeWithExistingGaps ports
// `test_import_index_range_with_existing_gaps`.
func TestIntegrationImportIndexRangeWithExistingGaps(t *testing.T) {
	dir := chdirTemp(t)

	existing := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond\n\n5\n00:00:05,000 --> 00:00:06,000\nFifth\n"
	srtFile := createSrtWithContent(t, dir, "test.srt", existing)

	csvContent := "start_time|end_time|text\n00:00:07,000|00:00:08,000|Sixth\n00:00:09,000|00:00:10,000|Seventh"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Index range: 6-7") {
		t.Errorf("stdout missing 'Index range: 6-7': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Imported: 2 subtitles") {
		t.Errorf("stdout missing 'Imported: 2 subtitles': %q", out.stdout)
	}

	content, _ := os.ReadFile(srtFile)
	c := string(content)
	if !strings.Contains(c, "6\n") {
		t.Errorf("file missing '6\\n': %q", c)
	}
	if !strings.Contains(c, "Sixth") {
		t.Errorf("file missing 'Sixth': %q", c)
	}
	if !strings.Contains(c, "7\n") {
		t.Errorf("file missing '7\\n': %q", c)
	}
	if !strings.Contains(c, "Seventh") {
		t.Errorf("file missing 'Seventh': %q", c)
	}
}

// ========== Dry-run and Force Flag Tests ==========

// TestIntegrationImportDryRunMode ports `test_import_dry_run_mode`.
func TestIntegrationImportDryRunMode(t *testing.T) {
	dir := chdirTemp(t)

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000|First subtitle\n00:00:04,000|00:00:06,000|Second subtitle"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)
	srtFile := createEmptySrt(t, dir, "test.srt")

	initial, _ := os.ReadFile(srtFile)

	out := runImportDryRun(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("Command should succeed. stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Preview of 2 subtitle(s) to be imported") {
		t.Errorf("stdout missing preview header: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Dry-run mode: no subtitles were imported") {
		t.Errorf("stdout missing dry-run line: %q", out.stdout)
	}

	final, _ := os.ReadFile(srtFile)
	if string(initial) != string(final) {
		t.Errorf("File should not be modified in dry-run mode")
	}
}

// TestIntegrationImportWithForceFlag ports `test_import_with_force_flag`.
func TestIntegrationImportWithForceFlag(t *testing.T) {
	dir := chdirTemp(t)

	csvContent := "start_time|end_time|text\n00:00:01,000|00:00:03,000|Test subtitle"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)
	srtFile := createEmptySrt(t, dir, "test.srt")

	out := runImport(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("Command should succeed. stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitles imported successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Imported: 1 subtitles") {
		t.Errorf("stdout missing 'Imported: 1 subtitles': %q", out.stdout)
	}

	content, _ := os.ReadFile(srtFile)
	if !strings.Contains(string(content), "Test subtitle") {
		t.Errorf("file missing 'Test subtitle': %q", string(content))
	}
}

// TestIntegrationImportDryRunShowsPreviewDetails ports
// `test_import_dry_run_shows_preview_details`.
func TestIntegrationImportDryRunShowsPreviewDetails(t *testing.T) {
	dir := chdirTemp(t)

	csvContent := "start_time|end_time|text\n00:00:01,436|00:00:03,481|<i>Previously on...</i>\n00:00:05,000|00:00:07,000|Hello, world!"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)
	srtFile := createEmptySrt(t, dir, "test.srt")

	out := runImportDryRun(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("Command should succeed. stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "00:00:01,436 --> 00:00:03,481") {
		t.Errorf("stdout missing first timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:05,000 --> 00:00:07,000") {
		t.Errorf("stdout missing second timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Previously on") {
		t.Errorf("stdout missing 'Previously on': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Hello, world!") {
		t.Errorf("stdout missing 'Hello, world!': %q", out.stdout)
	}
}

// TestIntegrationImportDryRunWithManySubtitlesShowsLimitedPreview ports
// `test_import_dry_run_with_many_subtitles_shows_limited_preview`.
func TestIntegrationImportDryRunWithManySubtitlesShowsLimitedPreview(t *testing.T) {
	dir := chdirTemp(t)

	csvContent := "start_time|end_time|text\n" +
		"00:00:01,000|00:00:02,000|Subtitle 1\n" +
		"00:00:03,000|00:00:04,000|Subtitle 2\n" +
		"00:00:05,000|00:00:06,000|Subtitle 3\n" +
		"00:00:07,000|00:00:08,000|Subtitle 4\n" +
		"00:00:09,000|00:00:10,000|Subtitle 5\n" +
		"00:00:11,000|00:00:12,000|Subtitle 6\n" +
		"00:00:13,000|00:00:14,000|Subtitle 7"
	csvFile := createTempCsv(t, dir, "test.csv", csvContent)
	srtFile := createEmptySrt(t, dir, "test.srt")

	out := runImportDryRun(t, srtFile, csvFile, "|")
	if !out.success {
		t.Fatalf("Command should succeed. stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Preview of 7 subtitle(s) to be imported") {
		t.Errorf("stdout missing preview header: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "... and 2 more subtitle(s)") {
		t.Errorf("stdout missing '... and 2 more subtitle(s)': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Subtitle 1") {
		t.Errorf("stdout missing 'Subtitle 1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Subtitle 5") {
		t.Errorf("stdout missing 'Subtitle 5': %q", out.stdout)
	}
	if strings.Count(out.stdout, "Subtitle 6") != 0 {
		t.Errorf("Subtitle 6 should not appear in preview details: %q", out.stdout)
	}
	if strings.Count(out.stdout, "Subtitle 7") != 0 {
		t.Errorf("Subtitle 7 should not appear in preview details: %q", out.stdout)
	}
}
