package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// mergeWriteFile writes content to a .srt file named name inside dir, failing
// the test on error.
func mergeWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

// TestIntegrationMergeFillGaps builds a base track with a hole between its two
// subtitles and a donor track whose extra line lands inside that hole, then
// asserts the fill-gaps merge writes a renumbered output containing the donor
// line.
func TestIntegrationMergeFillGaps(t *testing.T) {
	dir := chdirTemp(t)

	// Base has a gap from 02,000 to 10,000.
	baseContent := "1\n00:00:01,000 --> 00:00:02,000\nBase one\n\n" +
		"2\n00:00:10,000 --> 00:00:12,000\nBase two\n"
	basePath := mergeWriteFile(t, dir, "base.srt", baseContent)

	// Donor's middle line (05,000 -> 06,000) falls in the base gap; the other
	// two donor lines overlap the base subtitles and must be skipped.
	donorContent := "1\n00:00:01,000 --> 00:00:02,000\nDonor overlap one\n\n" +
		"2\n00:00:05,000 --> 00:00:06,000\nDonor gap filler\n\n" +
		"3\n00:00:10,000 --> 00:00:12,000\nDonor overlap two\n"
	donorPath := mergeWriteFile(t, dir, "donor.srt", donorContent)

	outPath := filepath.Join(dir, "merged.srt")

	out := runCaptured(t, func() error {
		return HandleMerge(basePath, donorPath, outPath, "fill-gaps", 0, 0, false, false, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Merge completed successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Added: 1") {
		t.Errorf("stdout missing 'Added: 1': %q", out.stdout)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	fileContent := string(got)

	// The donor gap-filler must be present in the merged output.
	if !strings.Contains(fileContent, "Donor gap filler") {
		t.Errorf("merged file missing donor gap line: %q", fileContent)
	}
	// Both base lines survive.
	if !strings.Contains(fileContent, "Base one") || !strings.Contains(fileContent, "Base two") {
		t.Errorf("merged file missing base lines: %q", fileContent)
	}
	// Three subtitles, renumbered 1..3 in start-time order. The gap filler
	// sorts between the two base lines, so it is index 2.
	if !strings.Contains(fileContent, "2\n00:00:05,000 --> 00:00:06,000\nDonor gap filler") {
		t.Errorf("merged file not renumbered as expected: %q", fileContent)
	}
	if !strings.Contains(fileContent, "3\n00:00:10,000 --> 00:00:12,000\nBase two") {
		t.Errorf("merged file last index not 3: %q", fileContent)
	}
}
