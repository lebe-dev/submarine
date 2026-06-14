package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/gaps"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleGaps detects timing gaps between subtitles in a file and reports any
// gap whose duration is at least minGapMs milliseconds. It is a read-only
// command: it loads the file and never writes.
func HandleGaps(file string, minGapMs int64, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("detecting gaps in file: %s", file))

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

	found := gaps.FindGaps(subtitles, minGapMs)
	slog.Info(fmt.Sprintf("found %d gap(s) >= %d ms", len(found), minGapMs))

	gapDtos := make([]cli.GapDto, 0, len(found))
	for i := range found {
		g := &found[i]
		gapDtos = append(gapDtos, cli.GapDto{
			AfterIndex: g.AfterIndex,
			Start:      subtitle.FormatTimestamp(g.Start),
			End:        subtitle.FormatTimestamp(g.End),
			DurationMs: g.Duration.Milliseconds(),
		})
	}

	dto := cli.GapsResultDto{
		File:     file,
		MinGapMs: minGapMs,
		Count:    len(found),
		Gaps:     gapDtos,
	}

	cli.OutputSuccess(format, dto, func() {
		if len(found) == 0 {
			fmt.Println("No gaps found")
			return
		}
		for i := range found {
			g := &found[i]
			fmt.Printf(
				"after index %d: %s --> %s, duration %s\n",
				g.AfterIndex,
				subtitle.FormatTimestamp(g.Start),
				subtitle.FormatTimestamp(g.End),
				cli.FormatDurationReadable(g.Duration),
			)
		}
	})

	return nil
}
