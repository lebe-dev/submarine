package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// runAdd invokes HandleAdd (text output, not dry-run) capturing its
// stdout/stderr, mirroring the Rust `run_add_command` helper that shells out to
// `sm add <file> <timestamps> <text>`.
func runAdd(t *testing.T, file, timestamps, text string) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return HandleAdd(file, timestamps, text, false, cli.OutputFormatText)
	})
}

// chdirTemp changes the working directory to a fresh temp dir for the duration
// of the test, restoring it on cleanup. The add handler creates backups in a
// `backups/` directory relative to the working directory; isolating cwd keeps
// those backups inside the temp tree (and mirrors the Rust binary running from
// the project root with its own backups/ dir).
func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	// EvalSymlinks so absolute paths under the temp dir match canonicalized
	// resolution (macOS /var -> /private/var).
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return dir
	}
	return resolved
}

// TestIntegrationAddToExistingFile ports `test_add_to_existing_file`.
func TestIntegrationAddToExistingFile(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:03,000-00:00:05,000", "Second subtitle")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitle added successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "New index: 2") {
		t.Errorf("stdout missing 'New index: 2': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 2") {
		t.Errorf("stdout missing 'Total subtitles: 2': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	if !strings.Contains(fileContent, "2\n") {
		t.Errorf("file missing '2\\n': %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:03,000 --> 00:00:05,000") {
		t.Errorf("file missing timestamps: %q", fileContent)
	}
	if !strings.Contains(fileContent, "Second subtitle") {
		t.Errorf("file missing 'Second subtitle': %q", fileContent)
	}
}

// TestIntegrationAddToEmptyFile ports `test_add_to_empty_file`.
func TestIntegrationAddToEmptyFile(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "empty.srt")

	if err := os.WriteFile(testFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:01,000-00:00:03,000", "First subtitle")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "New index: 1") {
		t.Errorf("stdout missing 'New index: 1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 1") {
		t.Errorf("stdout missing 'Total subtitles: 1': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	if !strings.Contains(fileContent, "1\n") {
		t.Errorf("file missing '1\\n': %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:01,000 --> 00:00:03,000") {
		t.Errorf("file missing timestamps: %q", fileContent)
	}
	if !strings.Contains(fileContent, "First subtitle") {
		t.Errorf("file missing 'First subtitle': %q", fileContent)
	}
}

// TestIntegrationAddWithMultilineText ports `test_add_with_multiline_text`.
func TestIntegrationAddWithMultilineText(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:03,000-00:00:05,000", "Line 1\nLine 2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Line 1\nLine 2") {
		t.Errorf("file missing multiline text: %q", string(got))
	}
}

// TestIntegrationAddWithHtmlTags ports `test_add_with_html_tags`.
func TestIntegrationAddWithHtmlTags(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:03,000-00:00:05,000", "<i>Italic text</i>")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "<i>Italic text</i>") {
		t.Errorf("file missing html tags: %q", string(got))
	}
}

// TestIntegrationAddCreatesNewFile ports `test_add_creates_new_file`.
func TestIntegrationAddCreatesNewFile(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "new_file.srt")

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatalf("expected file not to exist: %v", err)
	}

	out := runAdd(t, testFile, "00:00:01,000-00:00:03,000", "First subtitle in new file")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitle added successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "New index: 1") {
		t.Errorf("stdout missing 'New index: 1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 1") {
		t.Errorf("stdout missing 'Total subtitles: 1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "N/A (new file)") {
		t.Errorf("stdout missing 'N/A (new file)': %q", out.stdout)
	}

	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("expected file to be created: %v", err)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	if !strings.Contains(fileContent, "1\n") {
		t.Errorf("file missing '1\\n': %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:01,000 --> 00:00:03,000") {
		t.Errorf("file missing timestamps: %q", fileContent)
	}
	if !strings.Contains(fileContent, "First subtitle in new file") {
		t.Errorf("file missing text: %q", fileContent)
	}
}

// TestIntegrationAddInvalidTimestampFormat ports `test_add_invalid_timestamp_format`.
func TestIntegrationAddInvalidTimestampFormat(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Missing hyphen separator.
	out := runAdd(t, testFile, "00:00:01,000 00:00:03,000", "Text")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Invalid timestamp format") {
		t.Errorf("stderr missing 'Invalid timestamp format': %q", out.stderr)
	}
}

// TestIntegrationAddEndBeforeStart ports `test_add_end_before_start`.
func TestIntegrationAddEndBeforeStart(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:05,000-00:00:03,000", "Text")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
}

// TestIntegrationAddBackupCreated ports `test_add_backup_created`.
func TestIntegrationAddBackupCreated(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:03,000-00:00:05,000", "Second")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
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
	if !strings.Contains(string(backupContent), "First") {
		t.Errorf("backup missing 'First': %q", string(backupContent))
	}
	if strings.Contains(string(backupContent), "Second") {
		t.Errorf("backup should not contain 'Second': %q", string(backupContent))
	}
}

// TestIntegrationAddWithGapInIndices ports `test_add_with_gap_in_indices`.
func TestIntegrationAddWithGapInIndices(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n5\n00:00:03,000 --> 00:00:04,000\nFifth\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:05,000-00:00:06,000", "Sixth")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "New index: 6") { // max(1,5) + 1 = 6
		t.Errorf("stdout missing 'New index: 6': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 3") {
		t.Errorf("stdout missing 'Total subtitles: 3': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	if !strings.Contains(fileContent, "6\n") {
		t.Errorf("file missing '6\\n': %q", fileContent)
	}
	if !strings.Contains(fileContent, "Sixth") {
		t.Errorf("file missing 'Sixth': %q", fileContent)
	}
}

// TestIntegrationAddMalformedFile ports `test_add_malformed_file`.
func TestIntegrationAddMalformedFile(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "bad.srt")

	if err := os.WriteFile(testFile, []byte("INVALID CONTENT\nNOT SRT FORMAT"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:01,000-00:00:03,000", "Text")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Failed to parse subtitle file") {
		t.Errorf("stderr missing 'Failed to parse subtitle file': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "sm doctor --fix") {
		t.Errorf("stderr missing 'sm doctor --fix' hint: %q", out.stderr)
	}
}

// TestIntegrationAddTimestampBeforeLastSubtitle ports
// `test_add_timestamp_before_last_subtitle`.
func TestIntegrationAddTimestampBeforeLastSubtitle(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:06,000-00:00:08,000", "Too early")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Timestamp conflict") {
		t.Errorf("stderr missing 'Timestamp conflict': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "00:00:07,000") { // last ends at
		t.Errorf("stderr missing '00:00:07,000': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "00:00:06,000") { // new starts at
		t.Errorf("stderr missing '00:00:06,000': %q", out.stderr)
	}
}

// TestIntegrationAddTimestampExactlyAfterLast ports
// `test_add_timestamp_exactly_after_last`.
func TestIntegrationAddTimestampExactlyAfterLast(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:07,000-00:00:09,000", "Exactly after")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitle added successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "New index: 3") {
		t.Errorf("stdout missing 'New index: 3': %q", out.stdout)
	}
}

// TestIntegrationAddTimestampOverlapping ports `test_add_timestamp_overlapping`.
func TestIntegrationAddTimestampOverlapping(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	content := "1\n00:00:10,000 --> 00:00:15,000\nFirst\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runAdd(t, testFile, "00:00:14,000-00:00:16,000", "Overlapping")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Timestamp conflict") {
		t.Errorf("stderr missing 'Timestamp conflict': %q", out.stderr)
	}
}
