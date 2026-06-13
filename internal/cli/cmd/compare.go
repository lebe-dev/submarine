// Package cmd holds the command handlers for the submarine CLI. Each handler is
// a 1-to-1 port of the corresponding Rust `src/bin/cli/cmd/*.rs` module.
package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/internal/cli/output"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// HandleCompare runs the TUI comparison view for two subtitle files. Compare is
// TUI-only and rejects JSON output. Port of the Rust `cmd::compare::handle`.
func HandleCompare(file1, file2 string, format cli.OutputFormat) error {
	// Compare is TUI-only, reject JSON output
	if format == cli.OutputFormatJson {
		cli.OutputError(
			format,
			"unsupported_format",
			"The compare command is TUI-only and does not support --output json",
			strPtr("Use 'sm verify' for machine-readable comparison output"),
		)
		return errors.New("")
	}

	slog.Info(fmt.Sprintf("comparing files: %s and %s", file1, file2))

	subtitles1, filename1, err := loadSubtitleFile(file1, format)
	if err != nil {
		return err
	}
	subtitles2, filename2, err := loadSubtitleFile(file2, format)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("loaded %d and %d subtitles", len(subtitles1), len(subtitles2)))

	if err := output.RunTUI(subtitles1, filename1, subtitles2, filename2); err != nil {
		return err
	}

	slog.Info("comparison completed")
	return nil
}

// loadSubtitleFile loads all subtitles from a file via the SubRip service. On
// error it prints a formatted error and returns an empty-message error to
// signal an already-reported failure. Port of the Rust `load_subtitle_file`.
func loadSubtitleFile(file string, format cli.OutputFormat) ([]subtitle.Subtitle, string, error) {
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

// strPtr returns a pointer to s (helper for optional hints).
func strPtr(s string) *string { return &s }
