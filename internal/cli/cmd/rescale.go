package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/rescale"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// rescaleStrPtr returns a pointer to s (helper for optional hints).
func rescaleStrPtr(s string) *string { return &s }

// rescaleAnchor holds a parsed "IDX=HH:MM:SS,mmm" anchor: the targeted subtitle
// index and the new timestamp it should map to.
type rescaleAnchor struct {
	index uint32
	tNew  time.Duration
}

// rescaleParseAnchor parses an anchor string of the form "IDX=HH:MM:SS,mmm".
// Returns ok=false when the format is invalid (missing '=' separator, bad
// index, or bad timestamp).
func rescaleParseAnchor(anchor string) (rescaleAnchor, bool) {
	parts := strings.SplitN(anchor, "=", 2)
	if len(parts) != 2 {
		return rescaleAnchor{}, false
	}

	idx, err := rescaleParseIndex(strings.TrimSpace(parts[0]))
	if err != nil {
		return rescaleAnchor{}, false
	}

	tNew, err := subtitle.ParseTimestamp(strings.TrimSpace(parts[1]))
	if err != nil {
		return rescaleAnchor{}, false
	}

	return rescaleAnchor{index: idx, tNew: tNew}, true
}

// rescaleParseIndex parses a decimal subtitle index (u32).
func rescaleParseIndex(s string) (uint32, error) {
	if s == "" {
		return 0, fmt.Errorf("empty index")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid index %q", s)
		}
	}
	var v uint64
	for _, ch := range s {
		v = v*10 + uint64(ch-'0')
		if v > math.MaxUint32 {
			return 0, fmt.Errorf("index out of range %q", s)
		}
	}
	return uint32(v), nil
}

// rescaleFindStart returns the current StartTime of the subtitle with the given
// index, or ok=false when no subtitle has that index.
func rescaleFindStart(subs []subtitle.Subtitle, index uint32) (time.Duration, bool) {
	for i := range subs {
		if subs[i].Index.Value() == index {
			return subs[i].StartTime.Value(), true
		}
	}
	return 0, false
}

// HandleRescale rescales subtitle timecodes via a linear transform t' = a*t + b.
// Exactly one mode must be provided: --factor, --from-fps/--to-fps, or two
// --anchor values.
func HandleRescale(file, outFile string, factor *float64, fromFps, toFps *float64, anchors []string, dryRun bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("rescaling file: %s", file))

	// Validate that exactly one mode is provided.
	modeCount := 0
	if factor != nil {
		modeCount++
	}
	if fromFps != nil && toFps != nil {
		modeCount++
	}
	if len(anchors) == 2 {
		modeCount++
	}

	if modeCount != 1 {
		cli.OutputError(
			format,
			"invalid_args",
			"specify one of --factor, --from-fps/--to-fps, or two --anchor",
			rescaleStrPtr("Provide exactly one rescale mode"),
		)
		return errors.New("")
	}

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}

	service := subtitle.NewSubRipService(resolved.BaseDir)

	subtitles, err := service.GetAll(resolved.Filename)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}
	slog.Info(fmt.Sprintf("loaded %d subtitles", len(subtitles)))

	// Resolve the transform coefficients (a, b) and the mode label.
	var a, b float64
	var modeStr string

	switch {
	case factor != nil:
		a = *factor
		b = 0
		modeStr = "factor"

	case fromFps != nil && toFps != nil:
		var fpsErr error
		a, fpsErr = rescale.FactorFromFps(*fromFps, *toFps)
		if fpsErr != nil {
			cli.OutputError(format, "rescale_error", fpsErr.Error(), nil)
			return errors.New("")
		}
		b = 0
		modeStr = "fps"

	default: // len(anchors) == 2
		anchor1, ok := rescaleParseAnchor(anchors[0])
		if !ok {
			cli.OutputError(
				format,
				"invalid_anchor",
				fmt.Sprintf("invalid anchor format: %s (expected IDX=HH:MM:SS,mmm)", anchors[0]),
				nil,
			)
			return errors.New("")
		}
		anchor2, ok := rescaleParseAnchor(anchors[1])
		if !ok {
			cli.OutputError(
				format,
				"invalid_anchor",
				fmt.Sprintf("invalid anchor format: %s (expected IDX=HH:MM:SS,mmm)", anchors[1]),
				nil,
			)
			return errors.New("")
		}

		t1Old, found := rescaleFindStart(subtitles, anchor1.index)
		if !found {
			cli.OutputError(
				format,
				"invalid_anchor",
				fmt.Sprintf("anchor index %d not found in file", anchor1.index),
				nil,
			)
			return errors.New("")
		}
		t2Old, found := rescaleFindStart(subtitles, anchor2.index)
		if !found {
			cli.OutputError(
				format,
				"invalid_anchor",
				fmt.Sprintf("anchor index %d not found in file", anchor2.index),
				nil,
			)
			return errors.New("")
		}

		var anchorErr error
		a, b, anchorErr = rescale.ComputeAnchorTransform(t1Old, anchor1.tNew, t2Old, anchor2.tNew)
		if anchorErr != nil {
			cli.OutputError(format, "rescale_error", anchorErr.Error(), nil)
			return errors.New("")
		}
		modeStr = "anchor"
	}

	slog.Info(fmt.Sprintf("rescale transform: a=%v b=%v mode=%s", a, b, modeStr))

	out, err := rescale.Rescale(subtitles, a, b)
	if err != nil {
		cli.OutputError(format, "rescale_error", err.Error(), nil)
		return errors.New("")
	}

	if dryRun {
		slog.Info("dry-run mode, previewing rescale")

		dto := cli.RescaleResultDto{
			File:       file,
			Mode:       modeStr,
			Factor:     cli.Percentage(a),
			OffsetMs:   int64(math.Round(b)),
			TotalCount: len(out),
			Output:     outFile,
			BackupPath: "N/A (dry-run)",
			DryRun:     true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: rescale would be applied")
			fmt.Println()
			fmt.Printf("Mode: %s\n", modeStr)
			fmt.Printf("Factor: %v\n", a)
			fmt.Printf("Offset: %d ms\n", int64(math.Round(b)))
			fmt.Printf("Subtitles affected: %d\n", len(out))
			fmt.Printf("Output: %s\n", outFile)
			fmt.Println()
			rescaleShowSample(subtitles, out)
		})

		return nil
	}

	outResolved, err := cli.ResolveNewPath(outFile)
	if err != nil {
		return err
	}

	// Back up the output file only if it already exists.
	backupService := backup.NewSubRipBackupService()
	backupResult, backupErr := backupService.CreateBackup(outResolved.FullPath)

	var backupPath string
	switch {
	case backupErr != nil:
		slog.Error(fmt.Sprintf("failed to create backup: %s", backupErr))
		cli.OutputError(
			format,
			"backup_failed",
			fmt.Sprintf("Failed to create backup: %s", backupErr),
			nil,
		)
		return errors.New("")
	case backupResult != nil:
		slog.Debug(fmt.Sprintf("backup created: %s", *backupResult))
		backupPath = *backupResult
	default:
		slog.Debug("file does not exist, skipping backup")
		backupPath = "N/A (new file)"
	}

	outService := subtitle.NewSubRipService(outResolved.BaseDir)
	if err := outService.WriteAll(outResolved.Filename, out); err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, outFile)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}

	slog.Info(fmt.Sprintf("successfully rescaled %d subtitles", len(out)))

	dto := cli.RescaleResultDto{
		File:       file,
		Mode:       modeStr,
		Factor:     cli.Percentage(a),
		OffsetMs:   int64(math.Round(b)),
		TotalCount: len(out),
		Output:     outFile,
		BackupPath: backupPath,
		DryRun:     false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Rescale applied successfully")
		fmt.Println()
		fmt.Printf("Mode: %s\n", modeStr)
		fmt.Printf("Factor: %v\n", a)
		fmt.Printf("Offset: %d ms\n", int64(math.Round(b)))
		fmt.Printf("Subtitles rescaled: %d\n", len(out))
		fmt.Printf("Output: %s\n", outFile)
		fmt.Printf("Backup: %s\n", backupPath)
	})

	return nil
}

// rescaleShowSample prints the first few subtitles before and after rescaling.
func rescaleShowSample(before, after []subtitle.Subtitle) {
	sampleCount := min(3, len(before))
	if sampleCount == 0 {
		return
	}

	fmt.Printf("Sample (first %d subtitles):\n", sampleCount)
	for i := 0; i < sampleCount; i++ {
		fmt.Printf(
			"  [%d] %s --> %s  =>  %s --> %s\n",
			before[i].Index.Value(),
			subtitle.FormatTimestamp(before[i].StartTime.Value()),
			subtitle.FormatTimestamp(before[i].EndTime.Value()),
			subtitle.FormatTimestamp(after[i].StartTime.Value()),
			subtitle.FormatTimestamp(after[i].EndTime.Value()),
		)
	}
}
