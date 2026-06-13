package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// parseOffset parses an offset string in format "+100" or "-500" to
// milliseconds. Port of the Rust `parse_offset`.
func parseOffset(offsetStr string) (int64, error) {
	slog.Debug(fmt.Sprintf("parsing offset string: %s", offsetStr))

	if len(offsetStr) == 0 || (offsetStr[0] != '+' && offsetStr[0] != '-') {
		return 0, fmt.Errorf("offset must start with '+' or '-' (e.g., '+100', '-500')")
	}

	millis, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		// anyhow's `.context("...")` Display ("{}") shows only the context
		// message, not the wrapped cause; match that here.
		return 0, errors.New("failed to parse offset as number")
	}

	slog.Debug(fmt.Sprintf("parsed offset: %d milliseconds", millis))
	return millis, nil
}

// applyOffset applies a time offset to a subtitle, returning a new subtitle with
// adjusted timestamps. Port of the Rust `apply_offset`.
func applyOffset(sub *subtitle.Subtitle, offset time.Duration) (subtitle.Subtitle, error) {
	newStart := sub.StartTime.Value() + offset
	newEnd := sub.EndTime.Value() + offset

	if newStart < 0 || newEnd < 0 {
		return subtitle.Subtitle{}, fmt.Errorf(
			"offset would result in negative timestamp for subtitle %d",
			sub.Index.Value(),
		)
	}

	// anyhow's `.context("...")` Display ("{}") shows only the context message,
	// not the wrapped cause; match that here.
	newStartTs, err := subtitle.NewSubtitleTimestamp(newStart)
	if err != nil {
		return subtitle.Subtitle{}, fmt.Errorf(
			"failed to create start timestamp for subtitle %d",
			sub.Index.Value(),
		)
	}

	newEndTs, err := subtitle.NewSubtitleTimestamp(newEnd)
	if err != nil {
		return subtitle.Subtitle{}, fmt.Errorf(
			"failed to create end timestamp for subtitle %d",
			sub.Index.Value(),
		)
	}

	result, err := subtitle.NewSubtitle(
		sub.Index,
		newStartTs,
		newEndTs,
		sub.Text,
	)
	if err != nil {
		return subtitle.Subtitle{}, fmt.Errorf(
			"failed to create subtitle %d with new timestamps",
			sub.Index.Value(),
		)
	}
	return result, nil
}

// HandleDelay is a 1-to-1 port of the Rust `cmd::delay::handle`.
func HandleDelay(file string, offsetStr string, dryRun bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("applying time offset to file: %s", file))

	offsetMillis, err := parseOffset(offsetStr)
	if err != nil {
		return err
	}
	offset := time.Duration(offsetMillis) * time.Millisecond
	slog.Info(fmt.Sprintf("parsed offset: %d milliseconds", offsetMillis))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}

	// Load subtitles first (for both dry-run and real mode)
	service := subtitle.NewSubRipService(resolved.BaseDir)

	subtitles, err := service.GetAll(resolved.Filename)
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
	slog.Info(fmt.Sprintf("loaded %d subtitles", len(subtitles)))

	if len(subtitles) == 0 {
		slog.Info("no subtitles found in file")
		dto := cli.DelayResultDto{
			OffsetMs:          offsetMillis,
			SubtitlesAdjusted: 0,
			BackupPath:        "N/A",
			DryRun:            dryRun,
			SampleBefore:      nil,
			SampleAfter:       nil,
		}
		cli.OutputSuccess(format, dto, func() {
			fmt.Println("No subtitles found in file")
		})
		return nil
	}

	// Apply offset to all subtitles
	adjustedSubtitles := make([]subtitle.Subtitle, 0, len(subtitles))
	for i := range subtitles {
		adjusted, err := applyOffset(&subtitles[i], offset)
		if err != nil {
			cli.OutputError(format, "offset_error", fmt.Sprintf("%s", err), nil)
			return errors.New("")
		}
		adjustedSubtitles = append(adjustedSubtitles, adjusted)
	}

	if dryRun {
		slog.Info("dry-run mode, previewing delay")

		// Show sample: first 3 subtitles before/after
		sampleCount := 3
		if len(subtitles) < sampleCount {
			sampleCount = len(subtitles)
		}

		sampleBefore := make([]cli.SubtitleDto, 0, sampleCount)
		for i := 0; i < sampleCount; i++ {
			sampleBefore = append(sampleBefore, cli.SubtitleDtoFromSubtitle(&subtitles[i]))
		}
		sampleAfter := make([]cli.SubtitleDto, 0, sampleCount)
		for i := 0; i < sampleCount; i++ {
			sampleAfter = append(sampleAfter, cli.SubtitleDtoFromSubtitle(&adjustedSubtitles[i]))
		}

		dto := cli.DelayResultDto{
			OffsetMs:          offsetMillis,
			SubtitlesAdjusted: len(subtitles),
			BackupPath:        "N/A (dry-run)",
			DryRun:            true,
			SampleBefore:      sampleBefore,
			SampleAfter:       sampleAfter,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: time offset would be applied")
			fmt.Println()
			fmt.Printf("Offset: %d ms\n", offsetMillis)
			fmt.Printf("Subtitles affected: %d\n", len(subtitles))
			fmt.Println()
			fmt.Printf("Sample (first %d subtitles):\n", sampleCount)
			for i := 0; i < sampleCount; i++ {
				fmt.Printf(
					"  [%d] %s --> %s  =>  %s --> %s\n",
					subtitles[i].Index.Value(),
					subtitle.FormatTimestamp(subtitles[i].StartTime.Value()),
					subtitle.FormatTimestamp(subtitles[i].EndTime.Value()),
					subtitle.FormatTimestamp(adjustedSubtitles[i].StartTime.Value()),
					subtitle.FormatTimestamp(adjustedSubtitles[i].EndTime.Value()),
				)
			}
		})

		return nil
	}

	// Create backup
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
		slog.Error(fmt.Sprintf("file does not exist: %s", file))
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

	// Write adjusted subtitles
	if err := service.WriteAll(resolved.Filename, adjustedSubtitles); err != nil {
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

	slog.Info(fmt.Sprintf("successfully applied offset to %d subtitles", len(adjustedSubtitles)))

	dto := cli.DelayResultDto{
		OffsetMs:          offsetMillis,
		SubtitlesAdjusted: len(adjustedSubtitles),
		BackupPath:        backupPath,
		DryRun:            false,
		SampleBefore:      nil,
		SampleAfter:       nil,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Time offset applied successfully")
		fmt.Println()
		fmt.Printf("Backup: %s\n", backupPath)
		fmt.Printf("Offset: %d ms\n", offsetMillis)
		fmt.Printf("Subtitles adjusted: %d\n", len(adjustedSubtitles))
	})

	return nil
}
