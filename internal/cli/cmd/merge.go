package cmd

// merge.go implements the `sm merge` command handler: it merges a donor
// subtitle timeline into an authoritative base timeline, optionally
// auto-detecting the time offset between them, and writes the result to a new
// output file.

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/merge"
	"github.com/lebe-dev/submarine/pkg/offset"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleMerge merges a donor subtitle track into an authoritative base track
// and writes the reconciled result to outFile.
func HandleMerge(
	baseFile, donorFile, outFile, strategy string,
	overlapToleranceMs, offsetMs int64,
	autoOffset, dryRun bool,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("merging donor %s into base %s", donorFile, baseFile))

	baseSubs, err := mergeLoadSubtitleFile(baseFile, format)
	if err != nil {
		return err
	}

	donorSubs, err := mergeLoadSubtitleFile(donorFile, format)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("loaded %d base and %d donor subtitles", len(baseSubs), len(donorSubs)))

	parsedStrategy, err := merge.ParseStrategy(strategy)
	if err != nil {
		cli.OutputError(format, "invalid_strategy", err.Error(), nil)
		return errors.New("")
	}

	var appliedOffset int64
	if autoOffset {
		report := offset.DetectOffset(baseSubs, donorSubs)
		appliedOffset = report.MedianOffsetMs
		slog.Info(fmt.Sprintf(
			"auto-detected offset: %d ms (anchor_matches: %d)",
			appliedOffset, report.AnchorMatches,
		))
	} else {
		appliedOffset = offsetMs
	}

	result, err := merge.Merge(baseSubs, donorSubs, parsedStrategy, overlapToleranceMs, appliedOffset)
	if err != nil {
		cli.OutputError(format, "merge_error", fmt.Sprintf("%s", err), nil)
		return errors.New("")
	}

	dto := cli.MergeResultDto{
		BaseCount:          result.BaseCount,
		DonorCount:         result.DonorCount,
		Added:              result.Added,
		SkippedOverlapping: result.SkippedOverlapping,
		Replaced:           result.Replaced,
		AppliedOffsetMs:    appliedOffset,
		TotalCount:         result.TotalCount,
		Output:             outFile,
		DryRun:             dryRun,
	}

	if dryRun {
		slog.Info("dry-run mode, no file will be written")

		if parsedStrategy == merge.KeepBase {
			for i := range result.SkippedDonor {
				s := &result.SkippedDonor[i]
				slog.Info(fmt.Sprintf(
					"keep-base: skipping donor [%d] %s --> %s: %s",
					s.Index.Value(),
					subtitle.FormatTimestamp(s.StartTime.Value()),
					subtitle.FormatTimestamp(s.EndTime.Value()),
					mergeOneLine(s.Text.Value()),
				))
			}
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: merge would be applied")
			fmt.Println()
			fmt.Printf("Strategy: %s\n", parsedStrategy.String())
			fmt.Printf("Applied offset: %d ms\n", appliedOffset)
			fmt.Println()
			fmt.Printf("Base subtitles:  %d\n", result.BaseCount)
			fmt.Printf("Donor subtitles: %d\n", result.DonorCount)
			fmt.Println()
			fmt.Printf("Would add:     %d\n", result.Added)
			fmt.Printf("Would skip (overlapping): %d\n", result.SkippedOverlapping)
			fmt.Printf("Would replace: %d\n", result.Replaced)
			fmt.Printf("Total after merge: %d\n", result.TotalCount)
			fmt.Println()
			fmt.Printf("Output (not written): %s\n", outFile)
		})

		return nil
	}

	outResolved, err := cli.ResolveNewPath(outFile)
	if err != nil {
		return err
	}

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
		slog.Debug("output file does not exist, skipping backup")
		backupPath = "N/A (new file)"
	}

	service := subtitle.NewSubRipService(outResolved.BaseDir)
	if err := service.WriteAll(outResolved.Filename, result.Merged); err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, outFile)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}

	slog.Info(fmt.Sprintf(
		"merge complete: added %d, skipped %d, replaced %d, total %d",
		result.Added, result.SkippedOverlapping, result.Replaced, result.TotalCount,
	))

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Merge completed successfully")
		fmt.Println()
		fmt.Printf("Backup: %s\n", backupPath)
		fmt.Printf("Applied offset: %d ms\n", appliedOffset)
		fmt.Printf("Added: %d\n", result.Added)
		fmt.Printf("Skipped (overlapping): %d\n", result.SkippedOverlapping)
		fmt.Printf("Replaced: %d\n", result.Replaced)
		fmt.Printf("Total subtitles: %d\n", result.TotalCount)
		fmt.Printf("Output: %s\n", outFile)
	})

	return nil
}

// mergeLoadSubtitleFile resolves and loads an existing subtitle file. On error
// it emits a structured CLI error and returns an empty (anyhow) error.
func mergeLoadSubtitleFile(file string, format cli.OutputFormat) ([]subtitle.Subtitle, error) {
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

	slog.Info(fmt.Sprintf("loaded %d subtitle(s) from %s", len(subtitles), resolved.Filename))
	return subtitles, nil
}

// mergeOneLine collapses newlines in subtitle text to single spaces for compact
// log output.
func mergeOneLine(text string) string {
	out := make([]rune, 0, len(text))
	for _, ch := range text {
		if ch == '\n' || ch == '\r' {
			ch = ' '
		}
		out = append(out, ch)
	}
	return string(out)
}
