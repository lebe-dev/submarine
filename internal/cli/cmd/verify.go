package cmd

// verify.go is a 1-to-1 port of the Rust `src/bin/cli/cmd/verify.rs` command
// handler. The shared package doc comment lives elsewhere in package cmd.

import (
	"errors"
	"fmt"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
	"github.com/lebe-dev/submarine/pkg/verify"

	"log/slog"
)

// HandleVerify is a 1-to-1 port of the Rust `cmd::verify::handle`.
//
// It loads two subtitle files, optionally filters them to a range, compares
// them, and outputs a verification report. It returns a non-nil error (with an
// empty message, like the Rust `anyhow::anyhow!("")`) when the verification has
// issues or is not perfect, or when a precondition fails.
func HandleVerify(
	file1 string,
	file2 string,
	rng *string,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("verifying files: %s and %s", file1, file2))

	refSubs, refFilename, err := verifyLoadSubtitleFile(file1, format)
	if err != nil {
		return err
	}

	targetSubs, targetFilename, err := verifyLoadSubtitleFile(file2, format)
	if err != nil {
		return err
	}

	slog.Debug(fmt.Sprintf("loaded %d and %d subtitles", len(refSubs), len(targetSubs)))

	var rangeInfo *[2]uint32
	if rng != nil {
		start, end, err := cli.ParseRange(*rng)
		if err != nil {
			return err
		}

		slog.Info(fmt.Sprintf("filtering subtitles to range %d-%d", start, end))

		refSubs = verifyRetainInRange(refSubs, start, end)
		targetSubs = verifyRetainInRange(targetSubs, start, end)

		rangeInfo = &[2]uint32{start, end}
	}

	if len(refSubs) == 0 {
		cli.OutputError(format, "empty_file", "reference file is empty", nil)
		return errors.New("")
	}

	if len(targetSubs) == 0 {
		cli.OutputError(format, "empty_file", "target file is empty", nil)
		return errors.New("")
	}

	report := verify.CompareSubtitles(refSubs, refFilename, targetSubs, targetFilename)

	var status string
	switch {
	case report.IsPerfect():
		status = "success"
	case len(report.ExtraInTarget) != 0 && !report.HasIssues():
		status = "warning"
	default:
		status = "failed"
	}

	dto := cli.VerifyDto{
		RefFile:             report.RefFile,
		TargetFile:          report.TargetFile,
		TotalRefCount:       report.TotalRefCount,
		TotalTargetCount:    report.TotalTargetCount,
		PerfectMatches:      report.PerfectMatches,
		TotalMatched:        report.TotalMatched(),
		MatchPercentage:     cli.Percentage(report.MatchPercentage()),
		TimestampMismatches: len(report.TimestampMismatches),
		MissingInTarget:     len(report.MissingInTarget),
		ExtraInTarget:       len(report.ExtraInTarget),
		DetectedOffset:      report.DetectedOffset,
		Status:              status,
	}

	cli.OutputSuccess(format, dto, func() {
		displayVerificationReport(&report, rangeInfo)
	})

	slog.Info(fmt.Sprintf("verification completed: %.1f%% match", report.MatchPercentage()))

	if report.HasIssues() || !report.IsPerfect() {
		return errors.New("")
	}

	return nil
}

// verifyRetainInRange mirrors the Rust `Vec::retain` filtering by index range
// (inclusive on both ends).
func verifyRetainInRange(subs []subtitle.Subtitle, start, end uint32) []subtitle.Subtitle {
	result := subs[:0]
	for _, s := range subs {
		index := s.Index.Value()
		if index >= start && index <= end {
			result = append(result, s)
		}
	}
	return result
}

// verifyLoadSubtitleFile is a 1-to-1 port of the private `load_subtitle_file`
// in verify.rs. On error it emits a structured CLI error and returns an empty
// (anyhow) error.
func verifyLoadSubtitleFile(
	file string,
	format cli.OutputFormat,
) ([]subtitle.Subtitle, string, error) {
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

// displayVerificationReport is a 1-to-1 port of the private
// `display_verification_report` in verify.rs.
func displayVerificationReport(report *verify.VerificationReport, rangeInfo *[2]uint32) {
	fmt.Println()
	if rangeInfo != nil {
		fmt.Printf("Verifying subtitle files (range: %d-%d)\n", rangeInfo[0], rangeInfo[1])
	} else {
		fmt.Println("Verifying subtitle files")
	}
	fmt.Println("========================")
	fmt.Println()
	fmt.Printf("Reference file: %s (%d subtitles)\n", report.RefFile, report.TotalRefCount)
	fmt.Printf("Target file:    %s (%d subtitles)\n", report.TargetFile, report.TotalTargetCount)
	fmt.Println()

	fmt.Println("Results")
	fmt.Println("=======")
	fmt.Println()

	matchPct := report.MatchPercentage()
	fmt.Printf("Matched: %d/%d (%.1f%%)\n", report.TotalMatched(), report.TotalRefCount, matchPct)

	if len(report.TimestampMismatches) != 0 {
		fmt.Printf("Timestamp mismatches: %d\n", len(report.TimestampMismatches))
	}

	if len(report.MissingInTarget) != 0 {
		fmt.Printf("Missing in %s: %d\n", report.TargetFile, len(report.MissingInTarget))
	}

	if report.DetectedOffset != nil {
		fmt.Printf("Index offset detected: %d\n", *report.DetectedOffset)
	}

	if len(report.ExtraInTarget) != 0 {
		fmt.Printf("Extra in %s: %d\n", report.TargetFile, len(report.ExtraInTarget))
	}

	fmt.Println()

	if len(report.TimestampMismatches) != 0 {
		fmt.Println("Timestamp mismatches:")
		limit := min(10, len(report.TimestampMismatches))
		for _, entry := range report.TimestampMismatches[:limit] {
			if entry.Status.Kind == verify.TimestampMismatch {
				fmt.Printf("  [%s] Reference: %s --> %s\n",
					entry.RefSubtitle.Index.String(), entry.Status.RefStart, entry.Status.RefEnd)
				fmt.Printf("       Target:    %s --> %s\n", entry.Status.TargetStart, entry.Status.TargetEnd)
			}
		}
		if len(report.TimestampMismatches) > 10 {
			fmt.Printf("  ... and %d more\n", len(report.TimestampMismatches)-10)
		}
		fmt.Println()
	}

	if len(report.MissingInTarget) != 0 {
		fmt.Println("Missing subtitles:")
		limit := min(10, len(report.MissingInTarget))
		for _, entry := range report.MissingInTarget[:limit] {
			sub := entry.RefSubtitle
			fmt.Printf("  [%s] %s --> %s (not found in %s)\n",
				sub.Index.String(),
				subtitle.FormatTimestamp(sub.StartTime.Value()),
				subtitle.FormatTimestamp(sub.EndTime.Value()),
				report.TargetFile,
			)
		}
		if len(report.MissingInTarget) > 10 {
			fmt.Printf("  ... and %d more\n", len(report.MissingInTarget)-10)
		}
		fmt.Println()
	}

	if len(report.ExtraInTarget) != 0 {
		fmt.Printf("Extra subtitles in %s:\n", report.TargetFile)
		limit := min(10, len(report.ExtraInTarget))
		for _, sub := range report.ExtraInTarget[:limit] {
			fmt.Printf("  [%s] %s --> %s\n",
				sub.Index.String(),
				subtitle.FormatTimestamp(sub.StartTime.Value()),
				subtitle.FormatTimestamp(sub.EndTime.Value()),
			)
		}
		if len(report.ExtraInTarget) > 10 {
			fmt.Printf("  ... and %d more\n", len(report.ExtraInTarget)-10)
		}
		fmt.Println()
	}

	switch {
	case report.IsPerfect():
		fmt.Println("Verification: SUCCESS")
	case len(report.ExtraInTarget) != 0 && !report.HasIssues():
		fmt.Println("Verification: WARNING (extra subtitles found)")
	default:
		fmt.Println("Verification: FAILED")
	}
	fmt.Println()
}
