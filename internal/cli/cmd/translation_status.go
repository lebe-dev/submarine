package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
	"github.com/lebe-dev/submarine/pkg/translationstatus"
)

// HandleTranslationStatus is a 1-to-1 port of the Rust
// `cmd::translation_status::handle`.
//
// It loads the reference and translation subtitle files, computes the
// translation progress, and outputs the report. It returns a non-nil error
// (with an empty message, like the Rust `anyhow::anyhow!("")`) when a
// precondition fails or a file fails to load.
func HandleTranslationStatus(
	reference string,
	translation string,
	chunkSize int,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("checking translation status for %s against %s", translation, reference))

	refSubs, refFilename, err := translationStatusLoadSubtitleFile(reference, format)
	if err != nil {
		return err
	}

	translationSubs, translationFilename, err := translationStatusLoadSubtitleFile(translation, format)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("loaded %d reference and %d translation subtitles",
		len(refSubs), len(translationSubs)))

	if len(refSubs) == 0 {
		cli.OutputError(format, "empty_file", "reference file is empty", nil)
		return errors.New("")
	}

	report := translationstatus.CheckTranslationStatus(
		refSubs,
		refFilename,
		translationSubs,
		translationFilename,
		chunkSize,
	)

	var nextChunk *cli.ChunkDto
	if report.NextChunk != nil {
		nextChunk = &cli.ChunkDto{
			StartIndex: report.NextChunk.StartIndex,
			EndIndex:   report.NextChunk.EndIndex,
		}
	}

	dto := cli.TranslationStatusDto{
		RefFile:            report.RefFile,
		TranslationFile:    report.TranslationFile,
		TotalCount:         report.TotalCount,
		TranslatedCount:    report.TranslatedCount,
		MissingCount:       report.MissingCount,
		ProgressPercentage: cli.Percentage(report.ProgressPercentage()),
		IsComplete:         report.IsComplete(),
		NextChunk:          nextChunk,
	}

	cli.OutputSuccess(format, dto, func() {
		translationStatusDisplayReport(&report)
	})

	slog.Info(fmt.Sprintf("translation progress: %d/%d (%.1f%%)",
		report.TranslatedCount, report.TotalCount, report.ProgressPercentage()))

	return nil
}

// translationStatusLoadSubtitleFile is a 1-to-1 port of the private
// `load_subtitle_file` in translation_status.rs. On error it emits a structured
// CLI error and returns an empty (anyhow) error.
func translationStatusLoadSubtitleFile(
	file string,
	format cli.OutputFormat,
) ([]subtitle.Subtitle, string, error) {
	slog.Debug(fmt.Sprintf("loading subtitle file: %s", file))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return nil, "", err
	}

	service := subtitle.NewSubRipService(resolved.BaseDir)

	subtitles, err := service.GetAll(resolved.Filename)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
		}
		return nil, "", errors.New("")
	}

	slog.Info(fmt.Sprintf("successfully loaded %d subtitle(s) from %s", len(subtitles), resolved.Filename))
	return subtitles, resolved.Filename, nil
}

// translationStatusDisplayReport is a 1-to-1 port of the private
// `display_report` in translation_status.rs.
func translationStatusDisplayReport(report *translationstatus.TranslationStatusReport) {
	percentage := report.ProgressPercentage()

	if report.IsComplete() {
		fmt.Printf("Progress: %d/%d (100%%)\n", report.TranslatedCount, report.TotalCount)
	} else {
		fmt.Printf("Progress: %d/%d (%.1f%%)\n", report.TranslatedCount, report.TotalCount, percentage)
	}

	if report.NextChunk != nil {
		fmt.Printf("Next chunk: %d-%d\n", report.NextChunk.StartIndex, report.NextChunk.EndIndex)
	} else if report.IsComplete() {
		fmt.Println("Translation complete!")
	}
}
