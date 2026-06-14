package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/diff"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleDiff compares two subtitle files and reports the subtitles that appear
// only in A, only in B, and the count of common (matched) subtitles. It is a
// read-only command: it never mutates either file.
func HandleDiff(fileA, fileB, by string, toleranceMs int64, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("diffing files: %s and %s", fileA, fileB))

	mode, err := diff.ParseByMode(by)
	if err != nil {
		cli.OutputError(format, "invalid_by_mode", fmt.Sprintf("%s", err), nil)
		return errors.New("")
	}

	a, err := diffLoadFile(fileA, format)
	if err != nil {
		return err
	}

	b, err := diffLoadFile(fileB, format)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("loaded %d and %d subtitles", len(a), len(b)))

	res := diff.Diff(a, b, mode, toleranceMs)

	onlyInA := make([]cli.SubtitleDto, 0, len(res.OnlyInA))
	for i := range res.OnlyInA {
		onlyInA = append(onlyInA, cli.SubtitleDtoFromSubtitle(&res.OnlyInA[i]))
	}

	onlyInB := make([]cli.SubtitleDto, 0, len(res.OnlyInB))
	for i := range res.OnlyInB {
		onlyInB = append(onlyInB, cli.SubtitleDtoFromSubtitle(&res.OnlyInB[i]))
	}

	dto := cli.DiffResultDto{
		AFile:       fileA,
		BFile:       fileB,
		By:          by,
		ToleranceMs: toleranceMs,
		OnlyInA:     onlyInA,
		OnlyInB:     onlyInB,
		CommonCount: len(res.Common),
	}

	cli.OutputSuccess(format, dto, func() {
		diffDisplayReport(&res, fileA, fileB)
	})

	slog.Info(fmt.Sprintf(
		"diff completed: %d only in A, %d only in B, %d common",
		len(res.OnlyInA), len(res.OnlyInB), len(res.Common),
	))

	return nil
}

// diffLoadFile resolves and loads a subtitle file. On a SubtitleError it emits a
// structured CLI error and returns an empty (anyhow-style) error.
func diffLoadFile(file string, format cli.OutputFormat) ([]subtitle.Subtitle, error) {
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

// diffDisplayReport prints a unified-diff-style summary: counts followed by up
// to 10 "- only in A" lines and up to 10 "+ only in B" lines.
func diffDisplayReport(res *diff.DiffResult, fileA, fileB string) {
	fmt.Println()
	fmt.Println("Diff between subtitle files")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Printf("--- %s\n", fileA)
	fmt.Printf("+++ %s\n", fileB)
	fmt.Println()
	fmt.Printf("Common:    %d\n", len(res.Common))
	fmt.Printf("Only in A: %d\n", len(res.OnlyInA))
	fmt.Printf("Only in B: %d\n", len(res.OnlyInB))
	fmt.Println()

	if len(res.OnlyInA) != 0 {
		limit := min(10, len(res.OnlyInA))
		for i := 0; i < limit; i++ {
			s := &res.OnlyInA[i]
			fmt.Printf(
				"- [%d] %s --> %s  %s\n",
				s.Index.Value(),
				subtitle.FormatTimestamp(s.StartTime.Value()),
				subtitle.FormatTimestamp(s.EndTime.Value()),
				diffOneLine(s.Text.Value()),
			)
		}
		if len(res.OnlyInA) > 10 {
			fmt.Printf("  ... and %d more\n", len(res.OnlyInA)-10)
		}
	}

	if len(res.OnlyInB) != 0 {
		limit := min(10, len(res.OnlyInB))
		for i := 0; i < limit; i++ {
			s := &res.OnlyInB[i]
			fmt.Printf(
				"+ [%d] %s --> %s  %s\n",
				s.Index.Value(),
				subtitle.FormatTimestamp(s.StartTime.Value()),
				subtitle.FormatTimestamp(s.EndTime.Value()),
				diffOneLine(s.Text.Value()),
			)
		}
		if len(res.OnlyInB) > 10 {
			fmt.Printf("  ... and %d more\n", len(res.OnlyInB)-10)
		}
	}

	fmt.Println()
}

// diffOneLine truncates the text to 50 chars and collapses newlines to spaces
// for a single-line diff entry. Reuses the shared truncateAt50 helper.
func diffOneLine(text string) string {
	out := make([]rune, 0, len(text))
	for _, ch := range truncateAt50(text) {
		if ch == '\n' || ch == '\r' {
			out = append(out, ' ')
			continue
		}
		out = append(out, ch)
	}
	return string(out)
}
