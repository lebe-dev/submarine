package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/normalize"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleNormalize normalizes a subtitle file: stable-sorting by start time,
// contiguous renumbering, and trimming overlaps between adjacent subtitles.
// It is a mutating command that writes the result back to the same file
// in-place, backing up first; --dry-run previews the changes without writing.
func HandleNormalize(file string, sortFlag, renumber, fixOverlaps, dryRun bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("normalizing file: %s", file))

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

	result, err := normalize.Normalize(subtitles, sortFlag, renumber, fixOverlaps)
	if err != nil {
		cli.OutputError(format, "normalize_error", fmt.Sprintf("%s", err), nil)
		return errors.New("")
	}

	if dryRun {
		slog.Info("dry-run mode, previewing normalize")

		dto := cli.NormalizeResultDto{
			File:          file,
			TotalCount:    len(result.Subtitles),
			Sorted:        sortFlag,
			Renumbered:    renumber,
			OverlapsFixed: result.OverlapsFixed,
			BackupPath:    "N/A (dry-run)",
			DryRun:        true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: normalization would be applied")
			fmt.Println()
			normalizeWriteActions(sortFlag, renumber, fixOverlaps, result.OverlapsFixed)
			fmt.Printf("Subtitles: %d\n", len(result.Subtitles))
		})

		return nil
	}

	// Create backup before writing.
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

	if err := service.WriteAll(resolved.Filename, result.Subtitles); err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}

	slog.Info(fmt.Sprintf("successfully normalized %d subtitles", len(result.Subtitles)))

	dto := cli.NormalizeResultDto{
		File:          file,
		TotalCount:    len(result.Subtitles),
		Sorted:        sortFlag,
		Renumbered:    renumber,
		OverlapsFixed: result.OverlapsFixed,
		BackupPath:    backupPath,
		DryRun:        false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Subtitles normalized successfully")
		fmt.Println()
		fmt.Printf("Backup: %s\n", backupPath)
		normalizeWriteActions(sortFlag, renumber, fixOverlaps, result.OverlapsFixed)
		fmt.Printf("Subtitles: %d\n", len(result.Subtitles))
	})

	return nil
}

// normalizeWriteActions prints the human-readable list of normalization actions
// that were (or would be) applied, mirroring the requested flags and the count
// of overlaps that were trimmed.
func normalizeWriteActions(sortFlag, renumber, fixOverlaps bool, overlapsFixed int) {
	if sortFlag {
		fmt.Println("Sorted: yes (by start time)")
	} else {
		fmt.Println("Sorted: no")
	}

	if renumber {
		fmt.Println("Renumbered: yes (from 1)")
	} else {
		fmt.Println("Renumbered: no")
	}

	if fixOverlaps {
		fmt.Printf("Overlaps fixed: %d\n", overlapsFixed)
	} else {
		fmt.Println("Overlaps fixed: no")
	}
}
