package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// gapsWriteFile writes content to <dir>/<name> and returns the full path,
// failing the test on error. Prefixed with the command name to avoid colliding
// with helpers in other files of package cmd.
func gapsWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

// gapsContent has a 4-second gap between subtitle 1 (ends 00:00:02,000) and
// subtitle 2 (starts 00:00:06,000).
const gapsContent = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:06,000 --> 00:00:07,000\nSecond\n"

// TestIntegrationGapsReportsLargeGap verifies that a gap between two subtitles
// is reported with the right after_index and duration.
func TestIntegrationGapsReportsLargeGap(t *testing.T) {
	dir := chdirTemp(t)
	testFile := gapsWriteFile(t, dir, "test.srt", gapsContent)

	out := runCaptured(t, func() error {
		return HandleGaps(testFile, 1000, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "after index 1:") {
		t.Errorf("stdout missing 'after index 1:': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "00:00:02,000 --> 00:00:06,000") {
		t.Errorf("stdout missing gap boundaries: %q", out.stdout)
	}
	// 4-second gap -> "4.000s" via FormatDurationReadable.
	if !strings.Contains(out.stdout, "duration 4.000s") {
		t.Errorf("stdout missing 'duration 4.000s': %q", out.stdout)
	}
}

// TestIntegrationGapsHugeMinGapReportsNone verifies that a min-gap larger than
// any gap reports no gaps.
func TestIntegrationGapsHugeMinGapReportsNone(t *testing.T) {
	dir := chdirTemp(t)
	testFile := gapsWriteFile(t, dir, "test.srt", gapsContent)

	out := runCaptured(t, func() error {
		return HandleGaps(testFile, 60000, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "No gaps found") {
		t.Errorf("stdout missing 'No gaps found': %q", out.stdout)
	}
	if strings.Contains(out.stdout, "after index") {
		t.Errorf("stdout should not report any gap: %q", out.stdout)
	}
}
