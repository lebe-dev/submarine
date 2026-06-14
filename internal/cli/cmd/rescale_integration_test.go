package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/internal/cli"
)

// rescaleFactorPtr returns a pointer to f (helper for the optional --factor arg).
func rescaleFactorPtr(f float64) *float64 { return &f }

// TestIntegrationRescaleFactorHalves rescales a file with --factor 0.5 to a
// separate output path and asserts the output timecodes are halved.
func TestIntegrationRescaleFactorHalves(t *testing.T) {
	dir := chdirTemp(t)
	inFile := filepath.Join(dir, "in.srt")
	outFile := filepath.Join(dir, "out.srt")

	// Timecodes chosen so that halving yields exact whole milliseconds.
	content := "1\n00:00:10,000 --> 00:00:20,000\nFirst\n\n" +
		"2\n00:00:30,000 --> 00:00:40,000\nSecond\n"
	if err := os.WriteFile(inFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCaptured(t, func() error {
		return HandleRescale(inFile, outFile, rescaleFactorPtr(0.5), nil, nil, nil, false, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "✓ Rescale applied successfully") {
		t.Errorf("stdout missing success line: %q", out.stdout)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
	fileContent := string(got)

	// Halved: 10s->5s, 20s->10s, 30s->15s, 40s->20s.
	if !strings.Contains(fileContent, "00:00:05,000 --> 00:00:10,000") {
		t.Errorf("output missing halved timecodes for subtitle 1: %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:15,000 --> 00:00:20,000") {
		t.Errorf("output missing halved timecodes for subtitle 2: %q", fileContent)
	}
	if !strings.Contains(fileContent, "First") || !strings.Contains(fileContent, "Second") {
		t.Errorf("output missing text: %q", fileContent)
	}
}

// TestIntegrationRescaleNoModeFails asserts that providing no rescale mode
// produces an invalid_args error.
func TestIntegrationRescaleNoModeFails(t *testing.T) {
	dir := chdirTemp(t)
	inFile := filepath.Join(dir, "in.srt")
	outFile := filepath.Join(dir, "out.srt")

	content := "1\n00:00:10,000 --> 00:00:20,000\nFirst\n"
	if err := os.WriteFile(inFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCaptured(t, func() error {
		return HandleRescale(inFile, outFile, nil, nil, nil, nil, false, cli.OutputFormatText)
	})
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "specify one of --factor") {
		t.Errorf("stderr missing invalid_args message: %q", out.stderr)
	}
	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		t.Errorf("output file should not have been created")
	}
}

// TestIntegrationRescaleDryRunWritesNothing asserts that --dry-run previews
// without writing the output file.
func TestIntegrationRescaleDryRunWritesNothing(t *testing.T) {
	dir := chdirTemp(t)
	inFile := filepath.Join(dir, "in.srt")
	outFile := filepath.Join(dir, "out.srt")

	content := "1\n00:00:10,000 --> 00:00:20,000\nFirst\n"
	if err := os.WriteFile(inFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCaptured(t, func() error {
		return HandleRescale(inFile, outFile, rescaleFactorPtr(0.5), nil, nil, nil, true, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}
	if !strings.Contains(out.stdout, "Dry-run") {
		t.Errorf("stdout missing dry-run notice: %q", out.stdout)
	}
	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		t.Errorf("output file should not be written in dry-run mode")
	}
}

// TestIntegrationRescaleAnchorTwoPoints rescales using two anchor points and
// asserts the output is written.
func TestIntegrationRescaleAnchorTwoPoints(t *testing.T) {
	dir := chdirTemp(t)
	inFile := filepath.Join(dir, "in.srt")
	outFile := filepath.Join(dir, "out.srt")

	// Starts at 10s (idx 1) and 30s (idx 2). Anchor: 10s->5s, 30s->15s => factor 0.5.
	content := "1\n00:00:10,000 --> 00:00:20,000\nFirst\n\n" +
		"2\n00:00:30,000 --> 00:00:40,000\nSecond\n"
	if err := os.WriteFile(inFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	anchors := []string{"1=00:00:05,000", "2=00:00:15,000"}
	out := runCaptured(t, func() error {
		return HandleRescale(inFile, outFile, nil, nil, nil, anchors, false, cli.OutputFormatText)
	})
	if !out.success {
		t.Fatalf("expected success, stderr: %s", out.stderr)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
	fileContent := string(got)
	if !strings.Contains(fileContent, "00:00:05,000 --> 00:00:10,000") {
		t.Errorf("output missing anchored timecodes for subtitle 1: %q", fileContent)
	}
	if !strings.Contains(fileContent, "00:00:15,000 --> 00:00:20,000") {
		t.Errorf("output missing anchored timecodes for subtitle 2: %q", fileContent)
	}
}

// TestIntegrationRescaleBadAnchorFormat asserts an invalid anchor string
// produces an invalid_anchor error.
func TestIntegrationRescaleBadAnchorFormat(t *testing.T) {
	dir := chdirTemp(t)
	inFile := filepath.Join(dir, "in.srt")
	outFile := filepath.Join(dir, "out.srt")

	content := "1\n00:00:10,000 --> 00:00:20,000\nFirst\n\n" +
		"2\n00:00:30,000 --> 00:00:40,000\nSecond\n"
	if err := os.WriteFile(inFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	anchors := []string{"1@bad", "2=00:00:15,000"}
	out := runCaptured(t, func() error {
		return HandleRescale(inFile, outFile, nil, nil, nil, anchors, false, cli.OutputFormatText)
	})
	if out.success {
		t.Fatal("expected failure")
	}
	if !strings.Contains(out.stderr, "invalid anchor format") {
		t.Errorf("stderr missing invalid anchor message: %q", out.stderr)
	}
}
