package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// dedupeDuplicateContent is two subtitles with identical text at near-identical
// timecodes (overlapping within a small tolerance) followed by a distinct one.
// Deduplicating with a generous tolerance collapses the first two into one,
// leaving 2 final subtitles (Removed == 1).
const dedupeDuplicateContent = "1\n00:00:01,000 --> 00:00:03,000\nHello world\n\n" +
	"2\n00:00:01,050 --> 00:00:03,100\nHello world\n\n" +
	"3\n00:00:05,000 --> 00:00:07,000\nGoodbye\n"

// TestIntegrationDedupeRemovesDuplicate verifies that a duplicated line at
// near-identical timecodes is collapsed: Removed == 1 and the file shrinks.
func TestIntegrationDedupeRemovesDuplicate(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "dupes.srt")

	if err := os.WriteFile(testFile, []byte(dedupeDuplicateContent), 0o644); err != nil {
		t.Fatal(err)
	}
	origSize := len(dedupeDuplicateContent)

	out := runCaptured(t, func() error {
		return HandleDedupe(testFile, 200, false, false, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Duplicates removed successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Removed: 1") {
		t.Errorf("stdout missing 'Removed: 1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Final subtitles: 2") {
		t.Errorf("stdout missing 'Final subtitles: 2': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) >= origSize {
		t.Errorf("expected file to shrink: orig=%d, new=%d", origSize, len(got))
	}
	fileContent := string(got)
	if !strings.Contains(fileContent, "Hello world") {
		t.Errorf("file missing 'Hello world': %q", fileContent)
	}
	if !strings.Contains(fileContent, "Goodbye") {
		t.Errorf("file missing 'Goodbye': %q", fileContent)
	}
	// The renumbered, deduped output has exactly two blocks (indices 1 and 2).
	if strings.Contains(fileContent, "3\n00:00:05,000") {
		t.Errorf("expected renumbering to drop index 3, got: %q", fileContent)
	}
}

// TestIntegrationDedupeDryRunLeavesFileUntouched verifies that --dry-run reports
// the would-be result but never modifies the file.
func TestIntegrationDedupeDryRunLeavesFileUntouched(t *testing.T) {
	dir := chdirTemp(t)
	testFile := filepath.Join(dir, "dupes.srt")

	if err := os.WriteFile(testFile, []byte(dedupeDuplicateContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCaptured(t, func() error {
		return HandleDedupe(testFile, 200, false, true, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Dry-run: duplicates would be removed") {
		t.Errorf("stdout missing dry-run line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Removed: 1") {
		t.Errorf("stdout missing 'Removed: 1': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != dedupeDuplicateContent {
		t.Errorf("dry-run modified file; got: %q", string(got))
	}
}
