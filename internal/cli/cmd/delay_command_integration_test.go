package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// delayStrPtr returns a pointer to s (helper local to the delay integration
// tests). Prefixed to avoid colliding with other helpers in package cmd.
func delayStrPtr(s string) *string { return &s }

// runDelay invokes HandleDelay (text output, not dry-run) capturing its
// stdout/stderr, mirroring how the Rust integration tests shell out to
// `sm delay <file> <offset>`.
func runDelay(t *testing.T, file, offset string, rng, fromTimestamp *string, dryRun bool) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return HandleDelay(file, offset, rng, fromTimestamp, dryRun, cli.OutputFormatText)
	})
}

// delayWriteFixture writes a small three-subtitle SRT file into the current
// (temp) working directory and returns its absolute path.
func delayWriteFixture(t *testing.T, dir, name string) string {
	t.Helper()
	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n" +
		"2\n00:00:05,000 --> 00:00:06,000\nSecond\n\n" +
		"3\n00:00:10,000 --> 00:00:11,000\nThird\n"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestIntegrationDelayWholeFile shifts every subtitle when no scope selector is
// given (the original behavior is preserved).
func TestIntegrationDelayWholeFile(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runDelay(t, testFile, "+1000", nil, nil, false)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Time offset applied successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Subtitles adjusted: 3") {
		t.Errorf("stdout missing 'Subtitles adjusted: 3': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	// All three subtitles shifted by +1000ms.
	if !strings.Contains(fileContent, "00:00:02,000 --> 00:00:03,000") {
		t.Errorf("first subtitle not shifted: %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:06,000 --> 00:00:07,000") {
		t.Errorf("second subtitle not shifted: %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:11,000 --> 00:00:12,000") {
		t.Errorf("third subtitle not shifted: %q", fileContent)
	}
}

// TestIntegrationDelayWithRange shifts only subtitles inside the inclusive
// index range, leaving the rest unchanged.
func TestIntegrationDelayWithRange(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runDelay(t, testFile, "+1000", delayStrPtr("2-3"), nil, false)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Range: 2-3") {
		t.Errorf("stdout missing 'Range: 2-3': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Subtitles adjusted: 2") {
		t.Errorf("stdout missing 'Subtitles adjusted: 2': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	// Subtitle 1 untouched.
	if !strings.Contains(fileContent, "00:00:01,000 --> 00:00:02,000") {
		t.Errorf("first subtitle should be unchanged: %q", fileContent)
	}
	// Subtitles 2 and 3 shifted by +1000ms.
	if !strings.Contains(fileContent, "00:00:06,000 --> 00:00:07,000") {
		t.Errorf("second subtitle not shifted: %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:11,000 --> 00:00:12,000") {
		t.Errorf("third subtitle not shifted: %q", fileContent)
	}
}

// TestIntegrationDelayWithFromTimestamp shifts only subtitles whose start time
// is at or after the given timestamp.
func TestIntegrationDelayWithFromTimestamp(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runDelay(t, testFile, "+1000", nil, delayStrPtr("00:00:05,000"), false)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "From timestamp: 00:00:05,000") {
		t.Errorf("stdout missing 'From timestamp: 00:00:05,000': %q", out.stdout)
	}
	// Subtitles 2 (starts at 5s) and 3 (starts at 10s) are at or after 5s.
	if !strings.Contains(out.stdout, "Subtitles adjusted: 2") {
		t.Errorf("stdout missing 'Subtitles adjusted: 2': %q", out.stdout)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	fileContent := string(got)
	// Subtitle 1 (starts at 1s) untouched.
	if !strings.Contains(fileContent, "00:00:01,000 --> 00:00:02,000") {
		t.Errorf("first subtitle should be unchanged: %q", fileContent)
	}
	// Subtitle 2 shifted (boundary: start == from).
	if !strings.Contains(fileContent, "00:00:06,000 --> 00:00:07,000") {
		t.Errorf("second subtitle not shifted: %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:11,000 --> 00:00:12,000") {
		t.Errorf("third subtitle not shifted: %q", fileContent)
	}
}

// TestIntegrationDelayRangeAndFromTimestampConflict rejects supplying both
// selectors at once.
func TestIntegrationDelayRangeAndFromTimestampConflict(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runDelay(t, testFile, "+1000", delayStrPtr("1-2"), delayStrPtr("00:00:05,000"), false)
	if out.success {
		t.Fatal("expected failure when both --range and --from-timestamp are given")
	}
	if !strings.Contains(out.stderr, "use either --range or --from-timestamp, not both") {
		t.Errorf("stderr missing conflict message: %q", out.stderr)
	}

	// File must remain untouched on the validation error.
	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "00:00:01,000 --> 00:00:02,000") {
		t.Errorf("file should be unchanged on conflict: %q", string(got))
	}
}

// TestIntegrationDelayDryRunRange previews a range-limited shift without
// writing the file.
func TestIntegrationDelayDryRunRange(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runDelay(t, testFile, "+1000", delayStrPtr("1-1"), nil, true)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Dry-run: time offset would be applied") {
		t.Errorf("stdout missing dry-run line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Range: 1-1") {
		t.Errorf("stdout missing 'Range: 1-1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Subtitles affected: 1") {
		t.Errorf("stdout missing 'Subtitles affected: 1': %q", out.stdout)
	}

	// Dry-run must not modify the file.
	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "00:00:01,000 --> 00:00:02,000") {
		t.Errorf("file should be unchanged after dry-run: %q", string(got))
	}
}

// TestIntegrationDelayJSONRange asserts the JSON envelope carries the range
// fields and the adjusted count.
func TestIntegrationDelayJSONRange(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runCaptured(t, func() error {
		return HandleDelay(testFile, "+500", delayStrPtr("2-3"), nil, false, cli.OutputFormatJson)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "\"ok\":true") {
		t.Errorf("json missing ok:true: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "\"subtitles_adjusted\":2") {
		t.Errorf("json missing subtitles_adjusted:2: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "\"range_start\":2") {
		t.Errorf("json missing range_start:2: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "\"range_end\":3") {
		t.Errorf("json missing range_end:3: %q", out.stdout)
	}
}

// TestIntegrationDelayInvalidRange surfaces a range parse error.
func TestIntegrationDelayInvalidRange(t *testing.T) {
	dir := chdirTemp(t)
	testFile := delayWriteFixture(t, dir, "test.srt")

	out := runDelay(t, testFile, "+1000", delayStrPtr("bad"), nil, false)
	if out.success {
		t.Fatal("expected failure on invalid range")
	}
	if !strings.Contains(out.stderr, "Invalid range format") {
		t.Errorf("stderr missing 'Invalid range format': %q", out.stderr)
	}
}
