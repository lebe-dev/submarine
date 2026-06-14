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

// HandleDelay is a 1-to-1 port of the Rust `cmd::delay::handle`, extended with
// optional range-limited delay. When both rng and fromTimestamp are nil the
// whole file is shifted (the original behavior). When rng is set, only
// subtitles whose index falls inside the inclusive [start, end] bounds are
// shifted. When fromTimestamp is set, only subtitles whose start time is at or
// after the given timestamp are shifted. rng and fromTimestamp are mutually
// exclusive.
func HandleDelay(file, offsetStr string, rng *string, fromTimestamp *string, dryRun bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("applying time offset to file: %s", file))

	if rng != nil && fromTimestamp != nil {
		cli.OutputError(format, "invalid_args", "use either --range or --from-timestamp, not both", nil)
		return errors.New("")
	}

	offsetMillis, err := parseOffset(offsetStr)
	if err != nil {
		return err
	}
	offset := time.Duration(offsetMillis) * time.Millisecond
	slog.Info(fmt.Sprintf("parsed offset: %d milliseconds", offsetMillis))

	// Parse the scope selectors up front so input errors surface before path
	// resolution / file loading.
	var delayRangeStart *uint32
	var delayRangeEnd *uint32
	if rng != nil {
		start, end, rangeErr := cli.ParseRange(*rng)
		if rangeErr != nil {
			return rangeErr
		}
		slog.Info(fmt.Sprintf("limiting delay to range %d-%d", start, end))
		delayRangeStart = &start
		delayRangeEnd = &end
	}

	var delayFromDuration *time.Duration
	if fromTimestamp != nil {
		from, tsErr := subtitle.ParseTimestamp(*fromTimestamp)
		if tsErr != nil {
			return tsErr
		}
		slog.Info(fmt.Sprintf("limiting delay to subtitles starting at or after %s", subtitle.FormatTimestamp(from)))
		delayFromDuration = &from
	}

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
			RangeStart:        delayRangeStart,
			RangeEnd:          delayRangeEnd,
			FromTimestamp:     fromTimestamp,
		}
		cli.OutputSuccess(format, dto, func() {
			fmt.Println("No subtitles found in file")
		})
		return nil
	}

	// Apply offset only to in-scope subtitles, leaving the rest unchanged.
	// subtitlesAdjusted counts the subtitles actually shifted.
	adjustedSubtitles := make([]subtitle.Subtitle, 0, len(subtitles))
	subtitlesAdjusted := 0
	for i := range subtitles {
		if !delayInScope(&subtitles[i], delayRangeStart, delayRangeEnd, delayFromDuration) {
			adjustedSubtitles = append(adjustedSubtitles, subtitles[i])
			continue
		}
		adjusted, err := applyOffset(&subtitles[i], offset)
		if err != nil {
			cli.OutputError(format, "offset_error", fmt.Sprintf("%s", err), nil)
			return errors.New("")
		}
		adjustedSubtitles = append(adjustedSubtitles, adjusted)
		subtitlesAdjusted++
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
			SubtitlesAdjusted: subtitlesAdjusted,
			BackupPath:        "N/A (dry-run)",
			DryRun:            true,
			SampleBefore:      sampleBefore,
			SampleAfter:       sampleAfter,
			RangeStart:        delayRangeStart,
			RangeEnd:          delayRangeEnd,
			FromTimestamp:     fromTimestamp,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: time offset would be applied")
			fmt.Println()
			fmt.Printf("Offset: %d ms\n", offsetMillis)
			delayPrintScope(delayRangeStart, delayRangeEnd, fromTimestamp)
			fmt.Printf("Subtitles affected: %d\n", subtitlesAdjusted)
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

	slog.Info(fmt.Sprintf("successfully applied offset to %d subtitles", subtitlesAdjusted))

	dto := cli.DelayResultDto{
		OffsetMs:          offsetMillis,
		SubtitlesAdjusted: subtitlesAdjusted,
		BackupPath:        backupPath,
		DryRun:            false,
		SampleBefore:      nil,
		SampleAfter:       nil,
		RangeStart:        delayRangeStart,
		RangeEnd:          delayRangeEnd,
		FromTimestamp:     fromTimestamp,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Time offset applied successfully")
		fmt.Println()
		fmt.Printf("Backup: %s\n", backupPath)
		fmt.Printf("Offset: %d ms\n", offsetMillis)
		delayPrintScope(delayRangeStart, delayRangeEnd, fromTimestamp)
		fmt.Printf("Subtitles adjusted: %d\n", subtitlesAdjusted)
	})

	return nil
}

// delayInScope reports whether sub should be shifted given the optional range
// bounds and from-timestamp selectors. When all selectors are nil every
// subtitle is in scope (the original whole-file behavior). rangeStart/rangeEnd
// and from are mutually exclusive (validated by the caller), so at most one
// selector is active.
func delayInScope(sub *subtitle.Subtitle, rangeStart, rangeEnd *uint32, from *time.Duration) bool {
	if rangeStart != nil && rangeEnd != nil {
		index := sub.Index.Value()
		return index >= *rangeStart && index <= *rangeEnd
	}
	if from != nil {
		return sub.StartTime.Value() >= *from
	}
	return true
}

// delayPrintScope prints the active scope selector (range or from-timestamp) in
// text mode, mirroring the human-readable output of the verify command. It
// prints nothing when the whole file is in scope.
func delayPrintScope(rangeStart, rangeEnd *uint32, fromTimestamp *string) {
	if rangeStart != nil && rangeEnd != nil {
		fmt.Printf("Range: %d-%d\n", *rangeStart, *rangeEnd)
		return
	}
	if fromTimestamp != nil {
		fmt.Printf("From timestamp: %s\n", *fromTimestamp)
	}
}
