package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/offset"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleDetectOffset loads two subtitle files and reports the time offset (and
// possible fps drift) between them, computed from their shared anchor lines.
//
// It is a read-only command: it never mutates the input files, so it performs
// no backup and has no dry-run mode.
func HandleDetectOffset(fileA, fileB string, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("detecting offset between files: %s and %s", fileA, fileB))

	subsA, err := detectOffsetLoadFile(fileA, format)
	if err != nil {
		return err
	}

	subsB, err := detectOffsetLoadFile(fileB, format)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("loaded %d and %d subtitles", len(subsA), len(subsB)))

	if len(subsA) == 0 {
		cli.OutputError(format, "empty_file", fmt.Sprintf("File is empty: %s", fileA), nil)
		return errors.New("")
	}

	if len(subsB) == 0 {
		cli.OutputError(format, "empty_file", fmt.Sprintf("File is empty: %s", fileB), nil)
		return errors.New("")
	}

	rep := offset.DetectOffset(subsA, subsB)

	dto := cli.DetectOffsetDto{
		AnchorMatches:  rep.AnchorMatches,
		MedianOffsetMs: rep.MedianOffsetMs,
		StddevMs:       rep.StddevMs,
		DriftDetected:  rep.DriftDetected,
		SameVideo:      rep.SameVideo,
	}

	cli.OutputSuccess(format, dto, func() {
		detectOffsetDisplayReport(&rep, fileA, fileB)
	})

	slog.Info(fmt.Sprintf(
		"offset detection completed: %d anchor matches, median %d ms",
		rep.AnchorMatches, rep.MedianOffsetMs,
	))

	return nil
}

// detectOffsetLoadFile loads a subtitle file, emitting a structured CLI error
// and returning an empty (anyhow-style) error on failure.
func detectOffsetLoadFile(file string, format cli.OutputFormat) ([]subtitle.Subtitle, error) {
	slog.Debug(fmt.Sprintf("loading subtitle file: %s", file))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return nil, err
	}

	service := subtitle.NewSubRipService(resolved.BaseDir)

	subtitles, err := service.GetAll(resolved.Filename)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return nil, errors.New("")
		}
		return nil, err
	}

	slog.Info(fmt.Sprintf("successfully loaded %d subtitle(s) from %s", len(subtitles), resolved.Filename))
	return subtitles, nil
}

// detectOffsetDisplayReport renders the offset report in a human-readable form
// for text mode.
func detectOffsetDisplayReport(rep *offset.OffsetReport, fileA, fileB string) {
	fmt.Println("Offset Detection")
	fmt.Println("================")
	fmt.Println()
	fmt.Printf("File A: %s\n", fileA)
	fmt.Printf("File B: %s\n", fileB)
	fmt.Println()

	fmt.Printf("Anchor matches:     %d\n", rep.AnchorMatches)
	fmt.Printf("Median offset (ms): %d\n", rep.MedianOffsetMs)
	fmt.Printf("Stddev (ms):        %d\n", rep.StddevMs)
	fmt.Println()

	if !rep.SameVideo {
		fmt.Println("Note: files share little dialogue; they may not describe the same video.")
	}

	if rep.DriftDetected {
		fmt.Println("Note: drift detected (offset varies across the timeline); consider 'sm rescale'.")
	}
}
