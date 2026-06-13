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

// addDryRunDto is a port of the Rust `struct AddDryRunDto`. The Rust struct uses
// `#[serde(flatten)]` on the AddResultDto, so its fields are merged at the top
// level alongside the nested `subtitle` object.
type addDryRunDto struct {
	Subtitle cli.SubtitleDto `json:"subtitle"`
	// Flattened AddResultDto fields:
	NewIndex       uint32 `json:"new_index"`
	TotalSubtitles int    `json:"total_subtitles"`
	BackupPath     string `json:"backup_path"`
	DryRun         bool   `json:"dry_run,omitempty"`
}

// HandleAdd is a 1-to-1 port of the Rust `cmd::add::handle`.
func HandleAdd(
	file string,
	timestamps string,
	text string,
	dryRun bool,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("adding subtitle to file: %s", file))

	// Validate text input
	if err := cli.RejectControlChars(text, "text"); err != nil {
		return err
	}

	resolved, err := cli.ResolveNewPath(file)
	if err != nil {
		return err
	}

	// Parse timestamps
	slog.Debug(fmt.Sprintf("parsing timestamps: %s", timestamps))
	parts := strings.SplitN(timestamps, "-", 2)

	if len(parts) != 2 {
		hint := "example: \"00:00:10,000-00:00:12,500\""
		cli.OutputError(
			format,
			"invalid_timestamp_format",
			"Invalid timestamp format (expected 'HH:MM:SS,mmm-HH:MM:SS,mmm')",
			&hint,
		)
		return errors.New("")
	}

	startStr := parts[0]
	endStr := parts[1]

	startDuration, err := subtitle.ParseTimestamp(startStr)
	if err != nil {
		return fmt.Errorf("invalid start timestamp: %s", err)
	}
	startTimestamp, err := subtitle.NewSubtitleTimestamp(startDuration)
	if err != nil {
		return fmt.Errorf("invalid start timestamp value: %s", err)
	}

	endDuration, err := subtitle.ParseTimestamp(endStr)
	if err != nil {
		return fmt.Errorf("invalid end timestamp: %s", err)
	}
	endTimestamp, err := subtitle.NewSubtitleTimestamp(endDuration)
	if err != nil {
		return fmt.Errorf("invalid end timestamp value: %s", err)
	}

	subtitleText, err := subtitle.NewSubtitleText(text)
	if err != nil {
		return fmt.Errorf("invalid text: %s", err)
	}

	if dryRun {
		slog.Info("dry-run mode, previewing add")
		service := subtitle.NewSubRipService(resolved.BaseDir)
		existing, getErr := service.GetAll(resolved.Filename)
		if getErr != nil {
			existing = nil
		}
		var maxIndex uint32
		for _, s := range existing {
			if s.Index.Value() > maxIndex {
				maxIndex = s.Index.Value()
			}
		}
		nextIndex := maxIndex + 1

		previewIdx, err := subtitle.NewSubtitleIndex(nextIndex)
		if err != nil {
			previewIdx, _ = subtitle.NewSubtitleIndex(1)
		}

		preview, err := subtitle.NewSubtitle(previewIdx, startTimestamp, endTimestamp, subtitleText)
		if err != nil {
			return fmt.Errorf("invalid subtitle: %s", err)
		}

		dto := addDryRunDto{
			Subtitle:       cli.SubtitleDtoFromSubtitle(&preview),
			NewIndex:       nextIndex,
			TotalSubtitles: len(existing) + 1,
			BackupPath:     "N/A (dry-run)",
			DryRun:         true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: subtitle would be added")
			fmt.Println()
			fmt.Printf("New index: %d\n", nextIndex)
			fmt.Printf("Total subtitles: %d\n", len(existing)+1)
			fmt.Println()
			fmt.Printf("%s\n", preview.String())
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
		slog.Debug("file does not exist, skipping backup")
		backupPath = "N/A (new file)"
	default:
		slog.Debug(fmt.Sprintf("backup created: %s", *backupResult))
		backupPath = *backupResult
	}

	service := subtitle.NewSubRipService(resolved.BaseDir)

	slog.Debug("adding new subtitle...")
	report, err := service.Add(
		resolved.Filename,
		startTimestamp,
		endTimestamp,
		subtitleText,
	)
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

	slog.Info(fmt.Sprintf("subtitle added successfully with index %d", report.NewIndex))

	dto := cli.AddResultDto{
		NewIndex:       report.NewIndex,
		TotalSubtitles: report.TotalSubtitles,
		BackupPath:     backupPath,
		DryRun:         false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Subtitle added successfully")
		fmt.Println()
		fmt.Printf("New index: %d\n", report.NewIndex)
		fmt.Printf("Total subtitles: %d\n", report.TotalSubtitles)
		fmt.Printf("Backup: %s\n", backupPath)
	})

	return nil
}
