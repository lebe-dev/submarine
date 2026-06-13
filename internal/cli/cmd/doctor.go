// Package cmd holds the command handlers for the submarine CLI. Each Rust
// `pub fn handle(...)` in `src/bin/cli/cmd/*.rs` becomes an exported
// `func HandleX(...)` here. It is a 1-to-1 port of the Rust command handlers.
package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/doctor"
)

// HandleDoctor is a 1-to-1 port of the Rust `cmd::doctor::handle`.
func HandleDoctor(file string, fix bool, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("running doctor for file: %s", file))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}
	service := doctor.NewSubRipDoctorService(resolved.BaseDir)

	if fix {
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
			slog.Error("file does not exist, cannot fix")
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

		report, fixErr := service.Fix(resolved.Filename)
		if fixErr != nil {
			slog.Error(fmt.Sprintf("fix failed: %s", fixErr))
			var de *doctor.DoctorError
			errors.As(fixErr, &de)
			cliErr := cli.FormatDoctorError(de)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return errors.New("")
		}

		slog.Info("fix completed successfully")

		unfixableIssues := make([]cli.IssueDto, 0, len(report.UnfixableIssues))
		for _, i := range report.UnfixableIssues {
			unfixableIssues = append(unfixableIssues, cli.IssueDto{
				LineNumber:  i.LineNumber,
				BlockNumber: i.BlockNumber,
				Severity:    doctorSeverityString(i.Severity),
				Description: i.IssueType.String(),
			})
		}

		dto := cli.FixDto{
			OriginalPath:      report.OriginalPath,
			BackupPath:        backupPath,
			IssuesFixed:       report.IssuesFixed,
			IssuesUnfixable:   report.IssuesUnfixable,
			ValidationSuccess: report.ValidationSuccess,
			UnfixableIssues:   unfixableIssues,
		}

		cli.OutputSuccess(format, dto, func() {
			printFixReport(&report, backupPath)
		})

		return nil
	}

	report, diagErr := service.Diagnose(resolved.Filename)
	if diagErr != nil {
		slog.Error(fmt.Sprintf("diagnosis failed: %s", diagErr))
		var de *doctor.DoctorError
		errors.As(diagErr, &de)
		cliErr := cli.FormatDoctorError(de)
		cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
		return errors.New("")
	}

	slog.Info("diagnosis completed")

	issues := make([]cli.IssueDto, 0, len(report.Issues))
	for _, i := range report.Issues {
		issues = append(issues, cli.IssueDto{
			LineNumber:  i.LineNumber,
			BlockNumber: i.BlockNumber,
			Severity:    doctorSeverityString(i.Severity),
			Description: i.IssueType.String(),
		})
	}

	dto := cli.DiagnosticDto{
		FilePath:     report.FilePath,
		TotalLines:   report.TotalLines,
		TotalBlocks:  report.TotalBlocks,
		IsParsable:   report.IsParsable,
		ErrorCount:   report.ErrorCount(),
		WarningCount: report.WarningCount(),
		HasIssues:    report.HasIssues(),
		Issues:       issues,
	}

	cli.OutputSuccess(format, dto, func() {
		printDiagnosticReport(&report)
	})

	if report.HasErrors() {
		return errors.New("")
	}

	return nil
}

// doctorSeverityString maps a doctor.Severity to its DTO string. Was the inline
// `match i.severity { Severity::Error => "error", Severity::Warning => "warning" }`.
func doctorSeverityString(s doctor.Severity) string {
	switch s {
	case doctor.Error:
		return "error"
	case doctor.Warning:
		return "warning"
	}
	return ""
}

// printDiagnosticReport is a 1-to-1 port of the Rust `print_diagnostic_report`.
func printDiagnosticReport(report *doctor.DiagnosticReport) {
	fmt.Printf("Diagnosing: %s\n", report.FilePath)
	fmt.Println()

	fmt.Println("File Statistics:")
	fmt.Printf("  Total lines: %d\n", report.TotalLines)
	fmt.Printf("  Total blocks: %d\n", report.TotalBlocks)
	fmt.Println()

	if report.HasIssues() {
		fmt.Println("Issues Found:")
		fmt.Println()

		for _, issue := range report.Issues {
			fmt.Println(issue.String())
		}

		fmt.Println()
		fmt.Println("Summary:")

		errorCount := report.ErrorCount()
		warningCount := report.WarningCount()

		if errorCount > 0 {
			fmt.Printf(
				"  %d error%s found\n",
				errorCount,
				doctorPlural(errorCount),
			)
		}
		if warningCount > 0 {
			fmt.Printf(
				"  %d warning%s found\n",
				warningCount,
				doctorPlural(warningCount),
			)
		}

		if report.IsParsable {
			fmt.Println("  File is parsable (but has issues)")
		} else {
			fmt.Println("  File is NOT parsable")
		}

		fmt.Println()
		fmt.Println("Run with --fix to automatically repair these issues.")
	} else {
		fmt.Println("No issues found")
		fmt.Println("  File is valid")
	}
}

// printFixReport is a 1-to-1 port of the Rust `print_fix_report`.
func printFixReport(report *doctor.FixReport, backupPath string) {
	fmt.Printf("Fixing: %s\n", report.OriginalPath)
	fmt.Println()

	fmt.Printf("Backup: %s\n", backupPath)

	if report.IssuesFixed > 0 {
		fmt.Printf(
			"Fixed %d issue%s\n",
			report.IssuesFixed,
			doctorPlural(report.IssuesFixed),
		)
	}

	if report.IssuesUnfixable > 0 {
		fmt.Printf(
			"%d issue%s could not be fixed:\n",
			report.IssuesUnfixable,
			doctorPlural(report.IssuesUnfixable),
		)
		for _, issue := range report.UnfixableIssues {
			fmt.Printf("  %s\n", issue.String())
		}
	}

	if report.ValidationSuccess {
		fmt.Println("Validation successful: file can now be parsed")
	} else {
		fmt.Println("Validation failed: file still contains errors")
	}

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf(
		"  %d issue%s fixed\n",
		report.IssuesFixed,
		doctorPlural(report.IssuesFixed),
	)
	fmt.Printf(
		"  %d issue%s unfixable\n",
		report.IssuesUnfixable,
		doctorPlural(report.IssuesUnfixable),
	)

	if report.ValidationSuccess {
		fmt.Println("  File is now valid")
	} else {
		fmt.Println("  File still has errors")
	}
}

// plural returns "" when n == 1 and "s" otherwise. Mirrors the Rust
// `if n == 1 { "" } else { "s" }`.
func doctorPlural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
