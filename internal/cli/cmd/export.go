package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// formatAnchored formats subtitles in anchored format: [INDEX] TEXT.
// Port of the Rust `format_anchored`.
func formatAnchored(subtitles []subtitle.Subtitle) string {
	var output strings.Builder

	for i := range subtitles {
		index := subtitles[i].Index.Value()
		text := subtitles[i].Text.Value()

		lines := exportLines(text)
		if len(lines) == 0 {
			continue
		}

		output.WriteString(fmt.Sprintf("[%d] %s\n", index, lines[0]))

		for _, line := range lines[1:] {
			output.WriteString(line)
			output.WriteString("\n")
		}
	}

	return output.String()
}

// HandleExport is the port of the Rust `cmd::export::handle`.
func HandleExport(file string, rng string, exportFormat cli.ExportFormat, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("exporting subtitles from range %s from file: %s", rng, file))

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
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}

	slog.Debug(fmt.Sprintf("found %d total subtitles", len(subtitles)))

	if err := cli.ValidateRangeBoundaries(start, end, subtitles); err != nil {
		return err
	}

	var rangeSubtitles []subtitle.Subtitle
	for i := range subtitles {
		index := subtitles[i].Index.Value()
		if index >= start && index <= end {
			rangeSubtitles = append(rangeSubtitles, subtitles[i])
		}
	}

	if len(rangeSubtitles) == 0 {
		cli.OutputError(
			format,
			"no_subtitles_in_range",
			fmt.Sprintf("No subtitles found in range %d-%d", start, end),
			nil,
		)
		return errors.New("")
	}

	var formatName string
	switch exportFormat {
	case cli.ExportFormatAnchored:
		formatName = "anchored"
	}

	dtos := make([]cli.SubtitleDto, 0, len(rangeSubtitles))
	for i := range rangeSubtitles {
		dtos = append(dtos, cli.SubtitleDtoFromSubtitle(&rangeSubtitles[i]))
	}

	dto := cli.ExportDto{
		Subtitles:  dtos,
		Format:     formatName,
		RangeStart: start,
		RangeEnd:   end,
		Count:      len(rangeSubtitles),
	}

	cli.OutputSuccess(format, dto, func() {
		var output string
		switch exportFormat {
		case cli.ExportFormatAnchored:
			output = formatAnchored(rangeSubtitles)
		}
		fmt.Print(output)
	})

	slog.Info(fmt.Sprintf("successfully exported %d subtitle(s)", len(rangeSubtitles)))
	return nil
}

// exportLines mirrors Rust's str::lines(): splits on '\n', strips an optional
// trailing '\r' from each line, and yields no trailing empty element for a
// terminating '\n'. An empty string yields no lines.
func exportLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}
