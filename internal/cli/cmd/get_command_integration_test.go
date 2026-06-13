package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// runGet invokes HandleGet (text output) capturing its stdout/stderr, mirroring
// the Rust `run_get_command` helper that shells out to `sm get <file> <index>`.
func runGet(t *testing.T, file, index string) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return cmd_HandleGet(file, index)
	})
}

// cmd_HandleGet adapts HandleGet to the text-output default used by the binary.
func cmd_HandleGet(file, index string) error {
	return HandleGet(file, index, cli.OutputFormatText)
}

// ========== Positive Test Cases ==========

// TestIntegrationGetExistingSubtitleSimple ports `test_get_existing_subtitle_simple`.
func TestIntegrationGetExistingSubtitleSimple(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "1")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "1\n") {
		t.Errorf("stdout missing '1\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:01,000 --> 00:00:03,000") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "First subtitle") {
		t.Errorf("stdout missing text: %q", out.stdout)
	}

	out = runGet(t, file, "2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "2\n") {
		t.Errorf("stdout missing '2\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:03,500 --> 00:00:05,500") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Second subtitle") {
		t.Errorf("stdout missing text: %q", out.stdout)
	}

	out = runGet(t, file, "3")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "3\n") {
		t.Errorf("stdout missing '3\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:06,000 --> 00:00:08,000") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Third subtitle") {
		t.Errorf("stdout missing text: %q", out.stdout)
	}
}

// TestIntegrationGetSubtitleWithHtml ports `test_get_subtitle_with_html`.
func TestIntegrationGetSubtitleWithHtml(t *testing.T) {
	file := testDataPath(t, "valid/complex.srt")

	out := runGet(t, file, "1")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "1\n") {
		t.Errorf("stdout missing '1\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:01,436 --> 00:00:03,481") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "<i>Previously on") {
		t.Errorf("stdout missing html open: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "\"Resident Alien\"...</i>") {
		t.Errorf("stdout missing html close: %q", out.stdout)
	}
}

// TestIntegrationGetSubtitleWithIndexGap ports `test_get_subtitle_with_index_gap`.
func TestIntegrationGetSubtitleWithIndexGap(t *testing.T) {
	file := testDataPath(t, "valid/complex.srt")

	out := runGet(t, file, "5")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "5\n") {
		t.Errorf("stdout missing '5\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:07,790 --> 00:00:09,357") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "<i>You should be") {
		t.Errorf("stdout missing text: %q", out.stdout)
	}
}

// TestIntegrationGetSubtitleFromRealFile ports `test_get_subtitle_from_real_file`.
func TestIntegrationGetSubtitleFromRealFile(t *testing.T) {
	file := testDataPath(t, "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt")

	out := runGet(t, file, "1")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "1\n") {
		t.Errorf("stdout missing '1\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:01,436 --> 00:00:03,481") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}

	out = runGet(t, file, "2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "2\n") {
		t.Errorf("stdout missing '2\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:03,481 --> 00:00:05,135") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
}

// TestIntegrationGetEdgeCaseIndices ports `test_get_edge_case_indices`.
func TestIntegrationGetEdgeCaseIndices(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "1")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "First subtitle") {
		t.Errorf("stdout missing 'First subtitle': %q", out.stdout)
	}

	out = runGet(t, file, "3")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Third subtitle") {
		t.Errorf("stdout missing 'Third subtitle': %q", out.stdout)
	}
}

// ========== Negative Test Cases ==========

// TestIntegrationGetNonexistentIndex ports `test_get_nonexistent_index`.
func TestIntegrationGetNonexistentIndex(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "99999")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "not found") {
		t.Errorf("stderr missing 'not found': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "99999") {
		t.Errorf("stderr missing '99999': %q", out.stderr)
	}
}

// TestIntegrationGetFileNotFound ports `test_get_file_not_found`.
func TestIntegrationGetFileNotFound(t *testing.T) {
	out := runGet(t, "nonexistent.srt", "1")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") && !strings.Contains(out.stderr, "Error") {
		t.Errorf("stderr missing error marker: %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "not found") &&
		!strings.Contains(out.stderr, "No such file") &&
		!strings.Contains(out.stderr, "resolve file path") {
		t.Errorf("stderr missing file-not-found marker: %q", out.stderr)
	}
}

// TestIntegrationGetMalformedFile ports `test_get_malformed_file`.
func TestIntegrationGetMalformedFile(t *testing.T) {
	file := testDataPath(t, "invalid/malformed.srt")

	out := runGet(t, file, "1")
	if out.success {
		t.Fatal("expected failure")
	}
	stderr := strings.ToLower(out.stderr)
	if !strings.Contains(stderr, "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
	if !strings.Contains(stderr, "parse") && !strings.Contains(stderr, "invalid") {
		t.Errorf("stderr missing 'parse'/'invalid': %q", out.stderr)
	}
}

// TestIntegrationGetEmptyFile ports `test_get_empty_file`.
func TestIntegrationGetEmptyFile(t *testing.T) {
	file := testDataPath(t, "invalid/empty.srt")

	out := runGet(t, file, "1")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "not found") {
		t.Errorf("stderr missing 'not found': %q", out.stderr)
	}
}

// TestIntegrationGetPathTraversal ports `test_get_path_traversal`.
func TestIntegrationGetPathTraversal(t *testing.T) {
	out := runGet(t, "../../../etc/passwd", "1")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") && !strings.Contains(out.stderr, "Error") {
		t.Errorf("stderr missing error marker: %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "Invalid") &&
		!strings.Contains(out.stderr, "path") &&
		!strings.Contains(out.stderr, "resolve file path") {
		t.Errorf("stderr missing path marker: %q", out.stderr)
	}
}

// TestIntegrationGetIndexZero ports `test_get_index_zero`.
func TestIntegrationGetIndexZero(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "0")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "not found") {
		t.Errorf("stderr missing 'not found': %q", out.stderr)
	}
}

// TestIntegrationGetWithAbsolutePath ports `test_get_with_absolute_path`.
func TestIntegrationGetWithAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.srt")

	content := "1\n00:00:01,000 --> 00:00:03,000\nTest subtitle\n"
	if err := os.WriteFile(tempFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	out := runGet(t, tempFile, "1")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Test subtitle") {
		t.Errorf("stdout missing 'Test subtitle': %q", out.stdout)
	}
}

// ========== Tests for get with range syntax ==========

// TestIntegrationGetWithRangeAllExist ports `test_get_with_range_all_exist`.
func TestIntegrationGetWithRangeAllExist(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "1-3")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "1\n") {
		t.Errorf("stdout missing '1\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "First subtitle") {
		t.Errorf("stdout missing 'First subtitle': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "2\n") {
		t.Errorf("stdout missing '2\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Second subtitle") {
		t.Errorf("stdout missing 'Second subtitle': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "3\n") {
		t.Errorf("stdout missing '3\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Third subtitle") {
		t.Errorf("stdout missing 'Third subtitle': %q", out.stdout)
	}
}

// TestIntegrationGetWithRangeSingle ports `test_get_with_range_single`.
func TestIntegrationGetWithRangeSingle(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "2-2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "2\n") {
		t.Errorf("stdout missing '2\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Second subtitle") {
		t.Errorf("stdout missing 'Second subtitle': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "First subtitle") {
		t.Errorf("stdout should not contain 'First subtitle': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "Third subtitle") {
		t.Errorf("stdout should not contain 'Third subtitle': %q", out.stdout)
	}
}

// TestIntegrationGetWithRangeInvalidFormat ports `test_get_with_range_invalid_format`.
func TestIntegrationGetWithRangeInvalidFormat(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "5-2")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
}

// TestIntegrationGetWithInvalidIndex ports `test_get_with_invalid_index`.
func TestIntegrationGetWithInvalidIndex(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "abc")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "Invalid index") {
		t.Errorf("stderr missing 'Invalid index': %q", out.stderr)
	}
}

// ========== Additional Range Test Cases ==========

// TestIntegrationGetRangePartialOverlap ports `test_get_range_partial_overlap`.
func TestIntegrationGetRangePartialOverlap(t *testing.T) {
	file := testDataPath(t, "valid/complex.srt")

	out := runGet(t, file, "2-4")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "2\n") {
		t.Errorf("stdout missing '2\\n': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:03,481 --> 00:00:05,135") {
		t.Errorf("stdout missing timestamps: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Hello, Harry.") {
		t.Errorf("stdout missing 'Hello, Harry.': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "Previously on") {
		t.Errorf("stdout should not contain 'Previously on': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "You should be") {
		t.Errorf("stdout should not contain 'You should be': %q", out.stdout)
	}
}

// TestIntegrationGetRangeNoSubtitles ports `test_get_range_no_subtitles`.
func TestIntegrationGetRangeNoSubtitles(t *testing.T) {
	file := testDataPath(t, "valid/complex.srt")

	out := runGet(t, file, "3-4")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "No subtitles found in range 3-4") {
		t.Errorf("stdout missing range message: %q", out.stdout)
	}
}

// TestIntegrationGetRangeSubset ports `test_get_range_subset`.
func TestIntegrationGetRangeSubset(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "1-2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "First subtitle") {
		t.Errorf("stdout missing 'First subtitle': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Second subtitle") {
		t.Errorf("stdout missing 'Second subtitle': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "Third subtitle") {
		t.Errorf("stdout should not contain 'Third subtitle': %q", out.stdout)
	}
}

// TestIntegrationGetRangeWithHtmlTags ports `test_get_range_with_html_tags`.
func TestIntegrationGetRangeWithHtmlTags(t *testing.T) {
	file := testDataPath(t, "valid/complex.srt")

	out := runGet(t, file, "1-2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "<i>Previously on") {
		t.Errorf("stdout missing html open: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "\"Resident Alien\"...</i>") {
		t.Errorf("stdout missing html close: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Hello, Harry.") {
		t.Errorf("stdout missing 'Hello, Harry.': %q", out.stdout)
	}
}

// TestIntegrationGetRangeMultilineText ports `test_get_range_multiline_text`.
func TestIntegrationGetRangeMultilineText(t *testing.T) {
	file := testDataPath(t, "valid/complex.srt")

	out := runGet(t, file, "1-1")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Previously on") {
		t.Errorf("stdout missing 'Previously on': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Resident Alien") {
		t.Errorf("stdout missing 'Resident Alien': %q", out.stdout)
	}
}

// TestIntegrationGetRangeBeyondEnd ports `test_get_range_beyond_end`.
func TestIntegrationGetRangeBeyondEnd(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "2-100")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Second subtitle") {
		t.Errorf("stdout missing 'Second subtitle': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Third subtitle") {
		t.Errorf("stdout missing 'Third subtitle': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "First subtitle") {
		t.Errorf("stdout should not contain 'First subtitle': %q", out.stdout)
	}
}

// TestIntegrationGetRangeStartZero ports `test_get_range_start_zero`.
func TestIntegrationGetRangeStartZero(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "0-5")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "Start index must be >= 1") {
		t.Errorf("stderr missing 'Start index must be >= 1': %q", out.stderr)
	}
}

// TestIntegrationGetRangeEndZero ports `test_get_range_end_zero`.
func TestIntegrationGetRangeEndZero(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "1-0")
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "error") {
		t.Errorf("stderr missing 'error': %q", out.stderr)
	}
	if !strings.Contains(out.stderr, "End index must be >= 1") &&
		!strings.Contains(out.stderr, "Start index must be <=") {
		t.Errorf("stderr missing end/range validation: %q", out.stderr)
	}
}

// TestIntegrationGetRangeEmptyFile ports `test_get_range_empty_file`.
func TestIntegrationGetRangeEmptyFile(t *testing.T) {
	file := testDataPath(t, "invalid/empty.srt")

	out := runGet(t, file, "1-5")
	if !out.success {
		t.Fatalf("expected success (empty file is valid), stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "No subtitles found in range 1-5") {
		t.Errorf("stdout missing range message: %q", out.stdout)
	}
}

// TestIntegrationGetRangeOutputFormat ports `test_get_range_output_format`.
func TestIntegrationGetRangeOutputFormat(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "1-2")
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	lines := strings.Split(strings.TrimSuffix(out.stdout, "\n"), "\n")

	idx1Pos := indexOfLine(lines, "1")
	if idx1Pos < 0 {
		t.Fatalf("subtitle 1 line not found in: %q", out.stdout)
	}
	if !strings.Contains(lines[idx1Pos+1], "00:00:01,000 --> 00:00:03,000") {
		t.Errorf("line after subtitle 1 missing timestamps: %q", lines[idx1Pos+1])
	}
	if !strings.Contains(lines[idx1Pos+2], "First subtitle") {
		t.Errorf("line after timestamp missing 'First subtitle': %q", lines[idx1Pos+2])
	}
	if lines[idx1Pos+3] != "" {
		t.Errorf("expected blank line after subtitle 1, got %q", lines[idx1Pos+3])
	}

	idx2Pos := indexOfLine(lines, "2")
	if idx2Pos < 0 {
		t.Fatalf("subtitle 2 line not found in: %q", out.stdout)
	}
	if !strings.Contains(lines[idx2Pos+1], "00:00:03,500 --> 00:00:05,500") {
		t.Errorf("line after subtitle 2 missing timestamps: %q", lines[idx2Pos+1])
	}
	if !strings.Contains(lines[idx2Pos+2], "Second subtitle") {
		t.Errorf("line after timestamp missing 'Second subtitle': %q", lines[idx2Pos+2])
	}
}

// indexOfLine returns the index of the first line exactly equal to target, or -1.
func indexOfLine(lines []string, target string) int {
	for i, l := range lines {
		if l == target {
			return i
		}
	}
	return -1
}

// TestIntegrationGetRangeInvalidFormatVariations ports
// `test_get_range_invalid_format_variations`.
func TestIntegrationGetRangeInvalidFormatVariations(t *testing.T) {
	file := testDataPath(t, "valid/simple.srt")

	out := runGet(t, file, "123-")
	if out.success {
		t.Fatal("expected failure for '123-'")
	}
	if !strings.Contains(out.stderr, "Invalid") {
		t.Errorf("stderr missing 'Invalid': %q", out.stderr)
	}

	out = runGet(t, file, "abc-123")
	if out.success {
		t.Fatal("expected failure for 'abc-123'")
	}
	if !strings.Contains(out.stderr, "Invalid") {
		t.Errorf("stderr missing 'Invalid': %q", out.stderr)
	}

	out = runGet(t, file, "123-abc")
	if out.success {
		t.Fatal("expected failure for '123-abc'")
	}
	if !strings.Contains(out.stderr, "Invalid") {
		t.Errorf("stderr missing 'Invalid': %q", out.stderr)
	}
}
