package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// detectOffsetFileA is an SRT with several long, unique dialogue lines (each at
// least minAnchorRunes runes long), so DetectOffset can use them as anchors.
const detectOffsetFileA = "1\n" +
	"00:00:01,000 --> 00:00:03,000\n" +
	"The quick brown fox jumps over the lazy dog\n\n" +
	"2\n" +
	"00:00:05,000 --> 00:00:07,000\n" +
	"Twas brillig and the slithy toves did gyre\n\n" +
	"3\n" +
	"00:00:09,000 --> 00:00:11,000\n" +
	"All happy families resemble one another\n\n" +
	"4\n" +
	"00:00:13,000 --> 00:00:15,000\n" +
	"It was the best of times it was the worst\n\n" +
	"5\n" +
	"00:00:17,000 --> 00:00:19,000\n" +
	"Call me Ishmael some years ago never mind\n\n" +
	"6\n" +
	"00:00:21,000 --> 00:00:23,000\n" +
	"In the beginning the universe was created\n"

// detectOffsetWriteShifted writes detectOffsetFileA shifted later by a constant
// number of milliseconds, keeping the same dialogue text so the anchors match.
func detectOffsetWriteShifted(t *testing.T, path string, shiftMs int) {
	t.Helper()

	lines := []struct {
		startMs, endMs int
		text           string
	}{
		{1000, 3000, "The quick brown fox jumps over the lazy dog"},
		{5000, 7000, "Twas brillig and the slithy toves did gyre"},
		{9000, 11000, "All happy families resemble one another"},
		{13000, 15000, "It was the best of times it was the worst"},
		{17000, 19000, "Call me Ishmael some years ago never mind"},
		{21000, 23000, "In the beginning the universe was created"},
	}

	var b strings.Builder
	for i, ln := range lines {
		startTs := subtitle.FormatTimestamp(time.Duration(ln.startMs+shiftMs) * time.Millisecond)
		endTs := subtitle.FormatTimestamp(time.Duration(ln.endMs+shiftMs) * time.Millisecond)
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("\n")
		b.WriteString(startTs)
		b.WriteString(" --> ")
		b.WriteString(endTs)
		b.WriteString("\n")
		b.WriteString(ln.text)
		b.WriteString("\n\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestIntegrationDetectOffsetConstantShiftText verifies that, for a file B that
// is a constant shift of file A, the text output reports the (non-zero) median
// offset and reports the same video.
func TestIntegrationDetectOffsetConstantShiftText(t *testing.T) {
	dir := chdirTemp(t)
	fileA := filepath.Join(dir, "a.srt")
	fileB := filepath.Join(dir, "b.srt")

	if err := os.WriteFile(fileA, []byte(detectOffsetFileA), 0o644); err != nil {
		t.Fatal(err)
	}
	detectOffsetWriteShifted(t, fileB, 2500)

	out := runCaptured(t, func() error {
		return HandleDetectOffset(fileA, fileB, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	if !strings.Contains(out.stdout, "Median offset (ms): 2500") {
		t.Errorf("stdout missing 'Median offset (ms): 2500': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Anchor matches:") {
		t.Errorf("stdout missing 'Anchor matches:': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Stddev (ms):") {
		t.Errorf("stdout missing 'Stddev (ms):': %q", out.stdout)
	}
	// 6 matched anchors >= minSameVideoAnchors, so no "may not describe the same
	// video" note should appear.
	if strings.Contains(out.stdout, "may not describe the same video") {
		t.Errorf("unexpected same-video warning: %q", out.stdout)
	}
}

// TestIntegrationDetectOffsetConstantShiftJson verifies the JSON envelope is a
// success envelope and contains the median_offset_ms field.
func TestIntegrationDetectOffsetConstantShiftJson(t *testing.T) {
	dir := chdirTemp(t)
	fileA := filepath.Join(dir, "a.srt")
	fileB := filepath.Join(dir, "b.srt")

	if err := os.WriteFile(fileA, []byte(detectOffsetFileA), 0o644); err != nil {
		t.Fatal(err)
	}
	detectOffsetWriteShifted(t, fileB, 2500)

	out := runCaptured(t, func() error {
		return HandleDetectOffset(fileA, fileB, cli.OutputFormatJson)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	if !strings.Contains(out.stdout, `"ok":true`) {
		t.Errorf("stdout missing success envelope: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "median_offset_ms") {
		t.Errorf("stdout missing 'median_offset_ms': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, `"median_offset_ms":2500`) {
		t.Errorf("stdout missing non-zero median 2500: %q", out.stdout)
	}
}

// TestIntegrationDetectOffsetEmptyFile verifies that an empty file produces an
// empty_file error.
func TestIntegrationDetectOffsetEmptyFile(t *testing.T) {
	dir := chdirTemp(t)
	fileA := filepath.Join(dir, "a.srt")
	fileB := filepath.Join(dir, "empty.srt")

	if err := os.WriteFile(fileA, []byte(detectOffsetFileA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCaptured(t, func() error {
		return HandleDetectOffset(fileA, fileB, cli.OutputFormatText)
	})
	if out.success {
		t.Fatal("expected failure for empty file")
	}
	if !strings.Contains(out.stderr, "File is empty") {
		t.Errorf("stderr missing 'File is empty': %q", out.stderr)
	}
}
