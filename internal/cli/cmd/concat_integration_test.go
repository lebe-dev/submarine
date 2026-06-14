package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// runConcat invokes HandleConcat (text output, not dry-run) capturing its
// stdout/stderr.
func runConcat(t *testing.T, parts []string, outFile string, gapMs int64) capturedRun {
	t.Helper()
	return runCaptured(t, func() error {
		return HandleConcat(parts, outFile, gapMs, false, cli.OutputFormatText)
	})
}

// concatLoadSubs loads and parses all subtitles from a file for assertions.
func concatLoadSubs(t *testing.T, file string) []subtitle.Subtitle {
	t.Helper()
	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		t.Fatalf("failed to resolve %s: %v", file, err)
	}
	service := subtitle.NewSubRipService(resolved.BaseDir)
	subs, err := service.GetAll(resolved.Filename)
	if err != nil {
		t.Fatalf("failed to load %s: %v", file, err)
	}
	return subs
}

// TestIntegrationConcatTwoParts joins two parts and verifies the output total
// count equals the sum of the parts and that part2's timecodes are shifted past
// the end of part1.
func TestIntegrationConcatTwoParts(t *testing.T) {
	dir := chdirTemp(t)

	part1 := filepath.Join(dir, "part1.srt")
	part2 := filepath.Join(dir, "part2.srt")
	outFile := filepath.Join(dir, "out.srt")

	// part1 has two subtitles, ending at 00:00:10,000.
	part1Content := "1\n00:00:01,000 --> 00:00:05,000\nP1 First\n\n2\n00:00:06,000 --> 00:00:10,000\nP1 Second\n"
	if err := os.WriteFile(part1, []byte(part1Content), 0o644); err != nil {
		t.Fatal(err)
	}

	// part2 starts again at 00:00:01,000 (will be shifted forward on concat).
	part2Content := "1\n00:00:01,000 --> 00:00:03,000\nP2 First\n"
	if err := os.WriteFile(part2, []byte(part2Content), 0o644); err != nil {
		t.Fatal(err)
	}

	const gapMs = int64(1000)

	out := runConcat(t, []string{part1, part2}, outFile, gapMs)
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Subtitles concatenated successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 3") {
		t.Errorf("stdout missing 'Total subtitles: 3': %q", out.stdout)
	}

	merged := concatLoadSubs(t, outFile)

	// Total count must equal the sum of both parts (2 + 1 = 3).
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged subtitles, got %d", len(merged))
	}

	// Renumbered sequentially from 1.
	for i, sub := range merged {
		if got := sub.Index.Value(); got != uint32(i+1) {
			t.Errorf("subtitle %d has index %d, want %d", i, got, i+1)
		}
	}

	// part1's last subtitle ends at 00:00:10,000. part2's only subtitle (index 3
	// after renumbering) must be shifted past that end.
	part1LastEnd := merged[1].EndTime.Value()
	part2FirstStart := merged[2].StartTime.Value()
	if part2FirstStart <= part1LastEnd {
		t.Errorf(
			"expected part2 start (%s) to be shifted past part1 end (%s)",
			subtitle.FormatTimestamp(part2FirstStart),
			subtitle.FormatTimestamp(part1LastEnd),
		)
	}
}

// TestIntegrationConcatRequiresTwoFiles verifies that a single input file is
// rejected.
func TestIntegrationConcatRequiresTwoFiles(t *testing.T) {
	dir := chdirTemp(t)

	part1 := filepath.Join(dir, "part1.srt")
	outFile := filepath.Join(dir, "out.srt")

	if err := os.WriteFile(part1, []byte("1\n00:00:01,000 --> 00:00:02,000\nOnly\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runConcat(t, []string{part1}, outFile, 0)
	if out.success {
		t.Fatal("expected failure for single input file")
	}
	if !strings.Contains(out.stderr, "concat needs at least two input files") {
		t.Errorf("stderr missing 'concat needs at least two input files': %q", out.stderr)
	}
}

// TestIntegrationConcatDryRun verifies dry-run computes the total but writes no
// output file.
func TestIntegrationConcatDryRun(t *testing.T) {
	dir := chdirTemp(t)

	part1 := filepath.Join(dir, "part1.srt")
	part2 := filepath.Join(dir, "part2.srt")
	outFile := filepath.Join(dir, "out.srt")

	if err := os.WriteFile(part1, []byte("1\n00:00:01,000 --> 00:00:02,000\nA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part2, []byte("1\n00:00:01,000 --> 00:00:02,000\nB\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCaptured(t, func() error {
		return HandleConcat([]string{part1, part2}, outFile, 500, true, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Dry-run") {
		t.Errorf("stdout missing 'Dry-run': %q", out.stdout)
	}
	if !strings.Contains(out.stdout, "Total subtitles: 2") {
		t.Errorf("stdout missing 'Total subtitles: 2': %q", out.stdout)
	}

	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		t.Errorf("dry-run should not have written output file: %v", err)
	}
}
