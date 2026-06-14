package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/dedupe"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleDedupe collapses adjacent duplicate subtitles (matching text within a
// time tolerance) into single entries, writing the result back to the file in
// place. It honors --dry-run (computes counts and previews without writing) and
// backs the file up before any mutating write.
func HandleDedupe(file string, timeToleranceMs int64, ignoreHTML, dryRun bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("deduplicating subtitles in file: %s", file))

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

	res, err := dedupe.Dedupe(subtitles, timeToleranceMs, ignoreHTML)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		cli.OutputError(format, "dedupe_error", fmt.Sprintf("%s", err), nil)
		return errors.New("")
	}

	if dryRun {
		slog.Info("dry-run mode, previewing deduplication")

		dto := cli.DedupeResultDto{
			File:            file,
			OriginalCount:   res.OriginalCount,
			Removed:         res.Removed,
			Merged:          res.Merged,
			FinalCount:      res.FinalCount,
			TimeToleranceMs: timeToleranceMs,
			BackupPath:      "N/A (dry-run)",
			DryRun:          true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: duplicates would be removed")
			fmt.Println()
			fmt.Printf("Time tolerance: %d ms\n", timeToleranceMs)
			fmt.Printf("Original subtitles: %d\n", res.OriginalCount)
			fmt.Printf("Removed: %d\n", res.Removed)
			fmt.Printf("Merged groups: %d\n", res.Merged)
			fmt.Printf("Final subtitles: %d\n", res.FinalCount)
		})

		return nil
	}

	backupService := backup.NewSubRipBackupService()
	backupResult, backupErr := backupService.CreateBackup(resolved.FullPath)

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

	if err := service.WriteAll(resolved.Filename, res.Subtitles); err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}

	slog.Info(fmt.Sprintf(
		"deduplication complete: removed %d, merged %d, final %d",
		res.Removed, res.Merged, res.FinalCount,
	))

	dto := cli.DedupeResultDto{
		File:            file,
		OriginalCount:   res.OriginalCount,
		Removed:         res.Removed,
		Merged:          res.Merged,
		FinalCount:      res.FinalCount,
		TimeToleranceMs: timeToleranceMs,
		BackupPath:      backupPath,
		DryRun:          false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Duplicates removed successfully")
		fmt.Println()
		fmt.Printf("Backup: %s\n", backupPath)
		fmt.Printf("Original subtitles: %d\n", res.OriginalCount)
		fmt.Printf("Removed: %d\n", res.Removed)
		fmt.Printf("Merged groups: %d\n", res.Merged)
		fmt.Printf("Final subtitles: %d\n", res.FinalCount)
	})

	return nil
}
