// Package cmd holds the command handlers for the submarine CLI. Each handler is
// a 1-to-1 port of the corresponding Rust `src/bin/cli/cmd/*.rs` module.
package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleSet is a 1-to-1 port of the Rust `cmd::set::handle`.
func HandleSet(
	file string,
	index uint32,
	start *string,
	end *string,
	text *string,
	dryRun bool,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("setting subtitle %d in file: %s", index, file))

	if start == nil && end == nil && text == nil {
		cli.OutputError(
			format,
			"no_fields_to_update",
			"At least one of --start, --end, or --text must be specified",
			nil,
		)
		return errors.New("")
	}

	// Validate text input
	if text != nil {
		if err := cli.RejectControlChars(*text, "text"); err != nil {
			return err
		}
	}

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}

	var startTimestamp *subtitle.SubtitleTimestamp
	if start != nil {
		slog.Debug(fmt.Sprintf("parsing start timestamp: %s", *start))
		duration, err := subtitle.ParseTimestamp(*start)
		if err != nil {
			return fmt.Errorf("invalid start timestamp: %s", err)
		}
		ts, err := subtitle.NewSubtitleTimestamp(duration)
		if err != nil {
			return fmt.Errorf("invalid start timestamp value: %s", err)
		}
		startTimestamp = &ts
	}

	var endTimestamp *subtitle.SubtitleTimestamp
	if end != nil {
		slog.Debug(fmt.Sprintf("parsing end timestamp: %s", *end))
		duration, err := subtitle.ParseTimestamp(*end)
		if err != nil {
			return fmt.Errorf("invalid end timestamp: %s", err)
		}
		ts, err := subtitle.NewSubtitleTimestamp(duration)
		if err != nil {
			return fmt.Errorf("invalid end timestamp value: %s", err)
		}
		endTimestamp = &ts
	}

	var subtitleText *subtitle.SubtitleText
	if text != nil {
		slog.Debug(fmt.Sprintf("validating text (length: %d)", len(*text)))
		st, err := subtitle.NewSubtitleText(*text)
		if err != nil {
			return fmt.Errorf("invalid text: %s", err)
		}
		subtitleText = &st
	}

	update := subtitle.SubtitleUpdate{
		StartTime: startTimestamp,
		EndTime:   endTimestamp,
		Text:      subtitleText,
	}

	if dryRun {
		slog.Info("dry-run mode, previewing changes")
		service := subtitle.NewSubRipService(resolved.BaseDir)

		// Check subtitle exists and show before/after
		existing, err := service.GetByID(resolved.Filename, index)
		if err != nil {
			var subErr *subtitle.SubtitleError
			if errors.As(err, &subErr) {
				cliErr := cli.FormatSubtitleError(subErr, file)
				cli.OutputError(
					format,
					cliErr.Code,
					cliErr.Message,
					cliErr.Hint,
				)
				return errors.New("")
			}
			return err
		}

		if existing == nil {
			cli.OutputError(
				format,
				"subtitle_not_found",
				fmt.Sprintf("Subtitle with index %d not found in file", index),
				nil,
			)
			return errors.New("")
		}

		updated, err := update.ApplyTo(*existing)
		if err != nil {
			return fmt.Errorf("invalid update: %s", err)
		}

		var fields []string
		if update.StartTime != nil {
			fields = append(fields, "start_time")
		}
		if update.EndTime != nil {
			fields = append(fields, "end_time")
		}
		if update.Text != nil {
			fields = append(fields, "text")
		}

		dto := cli.SetResultDto{
			Index:         index,
			FieldsUpdated: fields,
			BackupPath:    "N/A (dry-run)",
			DryRun:        true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Printf("Dry-run: Subtitle %d would be updated\n", index)
			fmt.Println()
			fmt.Println("Before:")
			fmt.Printf("%s\n", existing.String())
			fmt.Println()
			fmt.Println("After:")
			fmt.Printf("%s\n", updated.String())
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
	case backupResult == nil:
		slog.Error("file does not exist, cannot update")
		cli.OutputError(
			format,
			"file_not_found",
			fmt.Sprintf("File does not exist: %s", file),
			nil,
		)
		return errors.New("")
	default:
		slog.Debug(fmt.Sprintf("backup created: %s", *backupResult))
		backupPath = *backupResult
	}

	service := subtitle.NewSubRipService(resolved.BaseDir)

	slog.Debug(fmt.Sprintf("updating subtitle %d...", index))
	report, err := service.Set(resolved.Filename, index, update)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, file)
			cli.OutputError(
				format,
				cliErr.Code,
				cliErr.Message,
				cliErr.Hint,
			)
			return errors.New("")
		}
		return err
	}

	slog.Info(fmt.Sprintf("subtitle %d updated successfully", index))

	dto := cli.SetResultDto{
		Index:         index,
		FieldsUpdated: report.FieldsUpdated,
		BackupPath:    backupPath,
		DryRun:        false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Printf("✓ Subtitle %d updated successfully\n", index)
		fmt.Println()
		fmt.Printf("Backup: %s\n", backupPath)
		fmt.Printf("Fields updated: %s\n", strings.Join(report.FieldsUpdated, ", "))
	})

	return nil
}
