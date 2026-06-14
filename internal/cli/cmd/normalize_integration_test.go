package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// normalizeOutOfOrderContent is an SRT file whose entries are out of start-time
// order and have a gap in their indices (1, then 5). After a sort+renumber pass
// the entries must be ordered by start time and renumbered contiguously 1..N.
const normalizeOutOfOrderContent = "5\n00:00:05,000 --> 00:00:06,000\nSecond\n\n1\n00:00:01,000 --> 00:00:02,000\nFirst\n"

// runNormalize invokes HandleNormalize with text output, capturing
// stdout/stderr.
func runNormalize(t *testing.T, file string, sortFlag, renumber, fixOverlaps, dryRun bool) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return HandleNormalize(file, sortFlag, renumber, fixOverlaps, dryRun, cli.OutputFormatText)
	})
}

// TestIntegrationNormalizeSortsAndRenumbers writes an out-of-order file with a
// gap in indices, normalizes it (sort + renumber), and asserts the file is
// sorted by start time and renumbered 1..N.
func TestIntegrationNormalizeSortsAndRenumbers(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	if err := os.WriteFile(testFile, []byte(normalizeOutOfOrderContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runNormalize(t, testFile, true, true, false, false)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitles normalized successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)

	// Sorted by start time: "First" (00:00:01) must appear before "Second"
	// (00:00:05).
	firstIdx := strings.Index(fileContent, "First")
	secondIdx := strings.Index(fileContent, "Second")
	if firstIdx < 0 || secondIdx < 0 {
		t.Fatalf("file missing expected text: %q", fileContent)
	}
	if firstIdx > secondIdx {
		t.Errorf("file not sorted by start time: %q", fileContent)
	}

	// Renumbered 1..N: indices must be 1 and 2 (no more gap to index 5).
	if strings.Contains(fileContent, "5\n") {
		t.Errorf("file still contains original gap index 5: %q", fileContent)
	}
	if !strings.HasPrefix(fileContent, "1\n00:00:01,000 --> 00:00:02,000\nFirst") {
		t.Errorf("file not renumbered/sorted as expected, got: %q", fileContent)
	}
	if !strings.Contains(fileContent, "2\n00:00:05,000 --> 00:00:06,000\nSecond") {
		t.Errorf("file missing renumbered second entry: %q", fileContent)
	}
}

// TestIntegrationNormalizeDryRunLeavesFileUntouched asserts that a dry-run
// previews the changes but never writes to the file.
func TestIntegrationNormalizeDryRunLeavesFileUntouched(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "test.srt")

	if err := os.WriteFile(testFile, []byte(normalizeOutOfOrderContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runNormalize(t, testFile, true, true, false, true)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Dry-run") {
		t.Errorf("stdout missing dry-run line: %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != normalizeOutOfOrderContent {
		t.Errorf("dry-run modified the file; got %q want %q", string(got), normalizeOutOfOrderContent)
	}
}
