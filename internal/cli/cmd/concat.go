package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/concat"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleConcat sequentially joins multiple subtitle parts into a single
// timeline and writes the result to outFile. Each part's timestamps are shifted
// so that it begins after the previous part (plus gapMs milliseconds), and the
// merged result is renumbered from 1.
func HandleConcat(parts []string, outFile string, gapMs int64, dryRun bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("concatenating %d subtitle parts into %s", len(parts), outFile))

	if len(parts) < 2 {
		cli.OutputError(format, "invalid_args", "concat needs at least two input files", nil)
		return errors.New("")
	}

	// Load each part in order.
	partsSubs := make([][]subtitle.Subtitle, 0, len(parts))
	for _, part := range parts {
		subs, err := concatLoadFile(part, format)
		if err != nil {
			return err
		}
		partsSubs = append(partsSubs, subs)
	}

	merged, err := concat.Concat(partsSubs, gapMs)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, outFile)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		cli.OutputError(format, "concat_error", fmt.Sprintf("%s", err), nil)
		return errors.New("")
	}

	slog.Info(fmt.Sprintf("merged %d subtitles from %d parts", len(merged), len(parts)))

	if dryRun {
		slog.Info("dry-run mode, previewing concat")

		dto := cli.ConcatResultDto{
			Parts:      len(parts),
			GapMs:      gapMs,
			TotalCount: len(merged),
			Output:     outFile,
			DryRun:     true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run: subtitles would be concatenated")
			fmt.Println()
			fmt.Printf("Parts: %s\n", strings.Join(parts, ", "))
			fmt.Printf("Gap: %d ms\n", gapMs)
			fmt.Printf("Total subtitles: %d\n", len(merged))
			fmt.Printf("Output: %s\n", outFile)
		})
		return nil
	}

	outResolved, err := cli.ResolveNewPath(outFile)
	if err != nil {
		return err
	}

	// Create backup of the output file if it already exists.
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
	if err := service.WriteAll(outResolved.Filename, merged); err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, outFile)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}
		return err
	}

	slog.Info(fmt.Sprintf("wrote %d subtitles to %s", len(merged), outFile))

	dto := cli.ConcatResultDto{
		Parts:      len(parts),
		GapMs:      gapMs,
		TotalCount: len(merged),
		Output:     outFile,
		DryRun:     false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Subtitles concatenated successfully")
		fmt.Println()
		fmt.Printf("Parts: %s\n", strings.Join(parts, ", "))
		fmt.Printf("Gap: %d ms\n", gapMs)
		fmt.Printf("Total subtitles: %d\n", len(merged))
		fmt.Printf("Output: %s\n", outFile)
		fmt.Printf("Backup: %s\n", backupPath)
	})

	return nil
}

// concatLoadFile resolves and loads all subtitles from an input part file. On
// error it emits a structured CLI error and returns an empty (anyhow) error.
func concatLoadFile(file string, format cli.OutputFormat) ([]subtitle.Subtitle, error) {
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
		}
		return nil, errors.New("")
	}

	slog.Info(fmt.Sprintf("loaded %d subtitle(s) from %s", len(subtitles), resolved.Filename))
	return subtitles, nil
}
