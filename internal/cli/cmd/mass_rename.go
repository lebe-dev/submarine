package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/rename"
)

// HandleMassRename is a 1-to-1 port of the Rust `cmd::mass_rename::handle`.
func HandleMassRename(
	fileMask string,
	dryRun bool,
	force bool,
	seriesMode bool,
	name *string,
	season *uint32,
	language *string,
	separator string,
	fileTemplate string,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("starting mass-rename with mask: %s", fileMask))
	slog.Debug(fmt.Sprintf(
		"dry_run: %t, force: %t, series_mode: %t",
		dryRun, force, seriesMode,
	))

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	service := rename.NewFileRenameService(currentDir)

	files, findErr := service.FindFiles(fileMask)
	if findErr != nil {
		var re *rename.RenameError
		errors.As(findErr, &re)
		code, msg := formatRenameError(re)
		cli.OutputError(format, code, msg, nil)
		return errors.New("")
	}

	slog.Info(fmt.Sprintf("found %d files", len(files)))

	sep := separator
	context := rename.NewTemplateContext().
		WithName(name).
		WithSeason(season).
		WithLanguage(language).
		WithSeparator(&sep)

	operations, prepErr := service.PrepareRenameOperations(files, fileTemplate, &context, seriesMode)
	if prepErr != nil {
		var re *rename.RenameError
		errors.As(prepErr, &re)
		code, msg := formatRenameError(re)
		cli.OutputError(format, code, msg, nil)
		return errors.New("")
	}

	opDtos := make([]cli.RenameOperationDto, 0, len(operations))
	for _, op := range operations {
		original := renameOriginalName(op)
		opDtos = append(opDtos, cli.RenameOperationDto{
			Original:  original,
			NewName:   op.NewName,
			Collision: op.Collision,
		})
	}

	if dryRun {
		slog.Info("dry-run mode, operations not executed")

		dto := cli.MassRenameResultDto{
			RenamedCount: 0,
			SkippedCount: 0,
			Operations:   opDtos,
			DryRun:       true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Preview:")
			fmt.Println()
			for _, op := range operations {
				originalName := renameOriginalName(op)
				fmt.Printf("  %s -> %s\n", originalName, op.NewName)
				if op.Collision {
					fmt.Println("    Warning: file already exists")
				}
			}
			fmt.Println()
			fmt.Println("Dry-run mode: no files were renamed")
		})

		return nil
	}

	// Show preview in text mode
	if format == cli.OutputFormatText {
		fmt.Println("Preview:")
		fmt.Println()
		for _, op := range operations {
			originalName := renameOriginalName(op)
			fmt.Printf("  %s -> %s\n", originalName, op.NewName)
			if op.Collision {
				fmt.Println("    Warning: file already exists")
			}
		}
		fmt.Println()
	}

	if !force && format == cli.OutputFormatText {
		fmt.Printf("Proceed with renaming %d files? (y/N): ", len(operations))
		if flushErr := massRenameStdoutFlush(); flushErr != nil {
			return flushErr
		}

		input, readErr := massRenameReadLine()
		if readErr != nil {
			return readErr
		}

		if !strings.EqualFold(strings.TrimSpace(input), "y") {
			slog.Info("operation cancelled by user")
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Execute rename operations
	slog.Info("starting file renaming")
	backupService := backup.NewSubRipBackupService()
	renamedCount := 0
	skippedCount := 0

	for _, op := range operations {
		originalName := renameOriginalName(op)

		if op.Collision && format == cli.OutputFormatText {
			fmt.Printf("File '%s' already exists. Overwrite? (y/N): ", op.NewName)
			if flushErr := massRenameStdoutFlush(); flushErr != nil {
				return flushErr
			}

			input, readErr := massRenameReadLine()
			if readErr != nil {
				return readErr
			}

			if !strings.EqualFold(strings.TrimSpace(input), "y") {
				slog.Info(fmt.Sprintf("skipping file due to collision: %q", op.OriginalPath))
				fmt.Printf("  Skipped: %s\n", originalName)
				skippedCount++
				continue
			}
		} else if op.Collision {
			// In JSON mode, skip collisions without prompting
			skippedCount++
			continue
		}

		backupResult, backupErr := backupService.CreateBackup(op.OriginalPath)
		switch {
		case backupErr != nil:
			slog.Error(fmt.Sprintf("failed to create backup: %s", backupErr))
			if format == cli.OutputFormatText {
				fmt.Fprintf(os.Stderr,
					"  Error: Failed to create backup for %s: %s\n",
					originalName, backupErr,
				)
			}
			skippedCount++
			continue
		case backupResult == nil:
			slog.Error("file does not exist, cannot be renamed")
			if format == cli.OutputFormatText {
				fmt.Fprintf(os.Stderr, "  Error: File not found: %s\n", originalName)
			}
			skippedCount++
			continue
		}

		newPath := filepath.Join(currentDir, op.NewName)

		if renameErr := os.Rename(op.OriginalPath, newPath); renameErr != nil {
			slog.Error(fmt.Sprintf("failed to rename file: %s", renameErr))
			if format == cli.OutputFormatText {
				fmt.Fprintf(os.Stderr, "  Failed to rename %s: %s\n", originalName, renameErr)
			}
			skippedCount++
			continue
		}

		slog.Info(fmt.Sprintf("file renamed: %s -> %s", originalName, op.NewName))
		if format == cli.OutputFormatText {
			fmt.Printf("  %s -> %s\n", originalName, op.NewName)
		}
		renamedCount++
	}

	dto := cli.MassRenameResultDto{
		RenamedCount: renamedCount,
		SkippedCount: skippedCount,
		Operations:   opDtos,
		DryRun:       false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Printf(
			"  %d file%s renamed\n",
			renamedCount,
			massRenamePlural(renamedCount),
		)
		if skippedCount > 0 {
			fmt.Printf(
				"  %d file%s skipped\n",
				skippedCount,
				massRenamePlural(skippedCount),
			)
		}
	})

	return nil
}

// renameOriginalName returns the original filename for an operation, defaulting
// to "unknown" (was `op.original_filename().unwrap_or_else(|| "unknown".to_string())`).
func renameOriginalName(op rename.RenameOperation) string {
	if name, ok := op.OriginalFilename(); ok {
		return name
	}
	return "unknown"
}

// formatRenameError is a 1-to-1 port of the Rust `format_rename_error`.
func formatRenameError(e *rename.RenameError) (string, string) {
	switch e.Kind {
	case rename.ErrNoFilesFound:
		return "no_files_found", "No files found matching pattern"
	case rename.ErrTemplateError:
		return "template_error", "Template rendering failed"
	case rename.ErrIoError:
		return "io_error", "File operation failed"
	case rename.ErrInvalidTemplate:
		return "invalid_template", "Invalid template"
	case rename.ErrGlobError:
		return "glob_error", "Glob pattern error"
	}
	return "io_error", "File operation failed"
}

// osStdoutFlush flushes stdout. Mirrors the Rust `io::stdout().flush()?`. The Go
// standard os.Stdout is unbuffered, so this is a no-op kept for parity.
func massRenameStdoutFlush() error {
	return nil
}

// massRenameStdin is a shared buffered reader over stdin, so that consecutive
// readLine calls do not lose buffered-but-unconsumed input (mirroring Rust's
// single locked io::stdin() handle).
var massRenameStdin = bufio.NewReader(os.Stdin)

// readLine reads a single line from stdin, mirroring the Rust
// `io::stdin().read_line(&mut input)?`.
func massRenameReadLine() (string, error) {
	line, err := massRenameStdin.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}

// massRenamePlural returns "" when n == 1 and "s" otherwise. Mirrors the Rust
// inline `if count == 1 { "" } else { "s" }`.
func massRenamePlural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
