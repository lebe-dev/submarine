package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// diffWriteFile writes content to a .srt file under dir, failing the test on
// error.
func diffWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

// diffEnvelope mirrors the JSON success envelope produced by cli.OutputSuccess
// for a DiffResultDto payload.
type diffEnvelope struct {
	Ok   bool `json:"ok"`
	Data struct {
		AFile       string `json:"a_file"`
		BFile       string `json:"b_file"`
		By          string `json:"by"`
		ToleranceMs int64  `json:"tolerance_ms"`
		OnlyInA     []struct {
			Index uint32 `json:"index"`
			Text  string `json:"text"`
		} `json:"only_in_a"`
		OnlyInB []struct {
			Index uint32 `json:"index"`
			Text  string `json:"text"`
		} `json:"only_in_b"`
		CommonCount int `json:"common_count"`
	} `json:"data"`
}

// TestIntegrationDiffExtraLineInB verifies that when file B has one extra
// subtitle line that A does not, that line shows up under only-in-B and the
// common count reflects the shared subtitles.
func TestIntegrationDiffExtraLineInB(t *testing.T) {
	dir := chdirTemp(t)

	// A has 2 subtitles; B has the same 2 plus one extra line.
	contentA := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n" +
		"2\n00:00:03,000 --> 00:00:04,000\nSecond\n"
	contentB := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n" +
		"2\n00:00:03,000 --> 00:00:04,000\nSecond\n\n" +
		"3\n00:00:05,000 --> 00:00:06,000\nThird extra line\n"

	fileA := diffWriteFile(t, dir, "a.srt", contentA)
	fileB := diffWriteFile(t, dir, "b.srt", contentB)

	out := runCaptured(t, func() error {
		return HandleDiff(fileA, fileB, "text", 0, cli.OutputFormatJson)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	var env diffEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.stdout)), &env); err != nil {
		t.Fatalf("failed to parse JSON output %q: %v", out.stdout, err)
	}

	if !env.Ok {
		t.Fatalf("expected ok=true, got %q", out.stdout)
	}
	if env.Data.CommonCount != 2 {
		t.Errorf("expected common_count=2, got %d (%q)", env.Data.CommonCount, out.stdout)
	}
	if len(env.Data.OnlyInA) != 0 {
		t.Errorf("expected no only_in_a entries, got %d (%q)", len(env.Data.OnlyInA), out.stdout)
	}
	if len(env.Data.OnlyInB) != 1 {
		t.Fatalf("expected exactly 1 only_in_b entry, got %d (%q)", len(env.Data.OnlyInB), out.stdout)
	}
	if env.Data.OnlyInB[0].Text != "Third extra line" {
		t.Errorf("expected only_in_b text 'Third extra line', got %q", env.Data.OnlyInB[0].Text)
	}
	if env.Data.By != "text" {
		t.Errorf("expected by='text', got %q", env.Data.By)
	}
}

// TestIntegrationDiffTextOutput verifies the unified-diff-style text summary
// includes the extra B-side line prefixed with '+'.
func TestIntegrationDiffTextOutput(t *testing.T) {
	dir := chdirTemp(t)

	contentA := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	contentB := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n" +
		"2\n00:00:03,000 --> 00:00:04,000\nExtra B line\n"

	fileA := diffWriteFile(t, dir, "a.srt", contentA)
	fileB := diffWriteFile(t, dir, "b.srt", contentB)

	out := runCaptured(t, func() error {
		return HandleDiff(fileA, fileB, "text", 0, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Common:    1") {
		t.Errorf("stdout missing 'Common:    1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Only in B: 1") {
		t.Errorf("stdout missing 'Only in B: 1': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "+ [2]") {
		t.Errorf("stdout missing '+ [2]' line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Extra B line") {
		t.Errorf("stdout missing 'Extra B line': %q", out.stdout)
	}
}

// TestIntegrationDiffInvalidByMode verifies an invalid --by value yields an
// invalid_by_mode error.
func TestIntegrationDiffInvalidByMode(t *testing.T) {
	dir := chdirTemp(t)

	content := "1\n00:00:01,000 --> 00:00:02,000\nFirst\n"
	fileA := diffWriteFile(t, dir, "a.srt", content)
	fileB := diffWriteFile(t, dir, "b.srt", content)

	out := runCaptured(t, func() error {
		return HandleDiff(fileA, fileB, "bogus", 0, cli.OutputFormatText)
	})
	if out.success {
		t.Fatal("expected failure for invalid by mode")
	}
	if !strings.Contains(out.stderr, "invalid diff mode") {
		t.Errorf("stderr missing 'invalid diff mode': %q", out.stderr)
	}
}
