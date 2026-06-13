package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// getRangeDto is a port of the Rust `struct GetRangeDto`.
type getRangeDto struct {
	Subtitles  []cli.SubtitleDto `json:"subtitles"`
	Count      int               `json:"count"`
	RangeStart uint32            `json:"range_start"`
	RangeEnd   uint32            `json:"range_end"`
}

// HandleGet is the port of the Rust `cmd::get::handle`.
func HandleGet(file, indexStr string, format cli.OutputFormat) error {
	if strings.Contains(indexStr, "-") {
		slog.Debug("detected range format, processing range")
		return handleGetRange(file, indexStr, format)
	}

	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		return fmt.Errorf(
			"Invalid index '%s'. Must be a positive number or range (e.g., 120-123)",
			indexStr,
		)
	}
	idx := uint32(index)

	slog.Info(fmt.Sprintf("getting subtitle %d from file: %s", idx, file))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}
	service := subtitle.NewSubRipService(resolved.BaseDir)

	slog.Debug(fmt.Sprintf("retrieving subtitle by id: %d..", idx))
	sub, err := service.GetByID(resolved.Filename, idx)
	if err != nil {
		cliErr := getFormatSubtitleError(err, file)
		cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
		return errors.New("")
	}

	if sub == nil {
		cli.OutputError(
			format,
			"subtitle_not_found",
			fmt.Sprintf("subtitle with index %d not found in %s", idx, file),
			nil,
		)
		return errors.New("")
	}

	slog.Info(fmt.Sprintf("subtitle %d found successfully", idx))
	dto := cli.SubtitleDtoFromSubtitle(sub)
	cli.OutputSuccess(format, dto, func() {
		fmt.Println(sub.String())
	})
	return nil
}

// handleGetRange is the port of the Rust `cmd::get::handle_range`.
func handleGetRange(file, rng string, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("getting subtitles in range %s from file: %s", rng, file))

	start, end, err := cli.ParseRange(rng)
	if err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("parsed range: start=%d, end=%d", start, end))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}
	service := subtitle.NewSubRipService(resolved.BaseDir)

	slog.Debug("retrieving all subtitles for filtering..")
	subtitles, err := service.GetAll(resolved.Filename)
	if err != nil {
		cliErr := getFormatSubtitleError(err, file)
		cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
		return errors.New("")
	}

	slog.Debug(fmt.Sprintf("found %d total subtitles", len(subtitles)))

	rangeSubtitles := make([]subtitle.Subtitle, 0)
	for i := range subtitles {
		idx := subtitles[i].Index.Value()
		if idx >= start && idx <= end {
			rangeSubtitles = append(rangeSubtitles, subtitles[i])
		}
	}

	slog.Debug(fmt.Sprintf(
		"found %d subtitle(s) in range %d-%d",
		len(rangeSubtitles), start, end,
	))

	if len(rangeSubtitles) == 0 {
		slog.Info(fmt.Sprintf("no subtitles found in range %d-%d", start, end))
		dto := getRangeDto{
			Subtitles:  []cli.SubtitleDto{},
			Count:      0,
			RangeStart: start,
			RangeEnd:   end,
		}
		cli.OutputSuccess(format, dto, func() {
			fmt.Printf("No subtitles found in range %d-%d\n", start, end)
		})
		return nil
	}

	slog.Info(fmt.Sprintf(
		"found %d subtitle(s) in range %d-%d",
		len(rangeSubtitles), start, end,
	))

	dtos := make([]cli.SubtitleDto, 0, len(rangeSubtitles))
	for i := range rangeSubtitles {
		dtos = append(dtos, cli.SubtitleDtoFromSubtitle(&rangeSubtitles[i]))
	}

	dto := getRangeDto{
		Subtitles:  dtos,
		Count:      len(rangeSubtitles),
		RangeStart: start,
		RangeEnd:   end,
	}

	cli.OutputSuccess(format, dto, func() {
		for i := range rangeSubtitles {
			fmt.Print(rangeSubtitles[i].String())
			if i < len(rangeSubtitles)-1 {
				fmt.Print("\n\n")
			} else {
				fmt.Println()
			}
		}
	})
	return nil
}

// getFormatSubtitleError unwraps a service error into a *subtitle.SubtitleError
// and runs it through cli.FormatSubtitleError, mirroring the Rust
// `utils::format_subtitle_error` call.
func getFormatSubtitleError(err error, file string) cli.CliError {
	var subErr *subtitle.SubtitleError
	if errors.As(err, &subErr) {
		return cli.FormatSubtitleError(subErr, file)
	}
	return cli.CliError{Code: "io_error", Message: err.Error()}
}
