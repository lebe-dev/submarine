package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// subtitleStats holds statistics collected from a subtitle file. Port of the
// Rust `struct SubtitleStats`.
type subtitleStats struct {
	totalCount              int
	totalDuration           time.Duration
	averageSubtitleDuration time.Duration
	averageGap              *time.Duration
	totalCharacters         int
	totalCharactersNoHTML   int
	totalWords              int
	totalLines              int
	subtitlesWithHTML       int
}

// HandleInfo is the port of the Rust `cmd::info::handle`.
func HandleInfo(file string, format cli.OutputFormat) error {
	slog.Info(fmt.Sprintf("getting statistics for file: %s", file))

	resolved, err := cli.ResolveExistingPath(file)
	if err != nil {
		return err
	}
	service := subtitle.NewSubRipService(resolved.BaseDir)

	slog.Debug("retrieving all subtitles for analysis")
	subtitles, err := service.GetAll(resolved.Filename)
	if err != nil {
		cliErr := infoFormatSubtitleError(err, file)
		cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
		return errors.New("")
	}

	slog.Debug(fmt.Sprintf("found %d subtitles", len(subtitles)))

	stats := calculateStatistics(subtitles)

	var averageGapMs *int64
	if stats.averageGap != nil {
		ms := int64(*stats.averageGap / time.Millisecond)
		averageGapMs = &ms
	}

	infoDto := cli.InfoDto{
		File:                      file,
		TotalCount:                stats.totalCount,
		TotalDurationMs:           int64(stats.totalDuration / time.Millisecond),
		TotalDuration:             subtitle.FormatTimestamp(stats.totalDuration),
		AverageSubtitleDurationMs: int64(stats.averageSubtitleDuration / time.Millisecond),
		AverageGapMs:              averageGapMs,
		TotalCharacters:           stats.totalCharacters,
		TotalCharactersNoHTML:     stats.totalCharactersNoHTML,
		TotalWords:                stats.totalWords,
		TotalLines:                stats.totalLines,
		SubtitlesWithHTML:         stats.subtitlesWithHTML,
	}

	cli.OutputSuccess(format, infoDto, func() {
		displayStatistics(&stats, file)
	})

	slog.Info("statistics displayed successfully")
	return nil
}

// calculateStatistics calculates comprehensive statistics from subtitles. Port
// of the Rust `calculate_statistics`.
func calculateStatistics(subtitles []subtitle.Subtitle) subtitleStats {
	slog.Debug(fmt.Sprintf("calculating statistics for %d subtitles", len(subtitles)))

	if len(subtitles) == 0 {
		return subtitleStats{
			totalCount:              0,
			totalDuration:           0,
			averageSubtitleDuration: 0,
			averageGap:              nil,
			totalCharacters:         0,
			totalCharactersNoHTML:   0,
			totalWords:              0,
			totalLines:              0,
			subtitlesWithHTML:       0,
		}
	}

	totalCount := len(subtitles)

	firstStart := subtitles[0].StartTime.Value()
	lastEnd := subtitles[len(subtitles)-1].EndTime.Value()
	totalDuration := lastEnd - firstStart

	var sumDurations int64
	for i := range subtitles {
		sumDurations += int64(subtitles[i].Duration() / time.Millisecond)
	}
	averageSubtitleDuration := time.Duration(sumDurations/int64(totalCount)) * time.Millisecond

	var averageGap *time.Duration
	if totalCount > 1 {
		var gapSum int64
		gapCount := 0

		for i := 0; i < len(subtitles)-1; i++ {
			currentEnd := subtitles[i].EndTime.Value()
			nextStart := subtitles[i+1].StartTime.Value()
			gap := nextStart - currentEnd

			gapMs := int64(gap / time.Millisecond)
			if gapMs > 0 {
				gapSum += gapMs
				gapCount++
			}
		}

		if gapCount > 0 {
			g := time.Duration(gapSum/int64(gapCount)) * time.Millisecond
			averageGap = &g
		}
	}

	totalCharacters := 0
	totalCharactersNoHTML := 0
	totalWords := 0
	totalLines := 0
	subtitlesWithHTML := 0

	for i := range subtitles {
		text := subtitles[i].Text.Value()
		textNoHTML := subtitles[i].TextWithoutHTML()

		totalCharacters += utf8.RuneCountInString(text)
		totalCharactersNoHTML += utf8.RuneCountInString(textNoHTML)

		words := strings.Fields(textNoHTML)
		totalWords += len(words)

		totalLines += subtitles[i].LineCount()

		if subtitles[i].HasHTMLTags() {
			subtitlesWithHTML++
		}
	}

	return subtitleStats{
		totalCount:              totalCount,
		totalDuration:           totalDuration,
		averageSubtitleDuration: averageSubtitleDuration,
		averageGap:              averageGap,
		totalCharacters:         totalCharacters,
		totalCharactersNoHTML:   totalCharactersNoHTML,
		totalWords:              totalWords,
		totalLines:              totalLines,
		subtitlesWithHTML:       subtitlesWithHTML,
	}
}

// displayStatistics displays statistics in a formatted, user-friendly way. Port
// of the Rust `display_statistics`.
func displayStatistics(stats *subtitleStats, filename string) {
	fmt.Println("Subtitle File Information")
	fmt.Println("========================")
	fmt.Println()
	fmt.Printf("File: %s\n", filename)
	fmt.Println()

	if stats.totalCount == 0 {
		fmt.Println("No subtitles found in file.")
		return
	}

	fmt.Println("Basic Information:")
	fmt.Printf("  Total subtitles: %d\n", stats.totalCount)
	fmt.Printf(
		"  Total duration:  %s (%s)\n",
		subtitle.FormatTimestamp(stats.totalDuration),
		cli.FormatDurationReadable(stats.totalDuration),
	)
	fmt.Println()

	fmt.Println("Timing Statistics:")
	fmt.Printf(
		"  Average subtitle duration: %s (%.2fs)\n",
		subtitle.FormatTimestamp(stats.averageSubtitleDuration),
		float64(int64(stats.averageSubtitleDuration/time.Millisecond))/1000.0,
	)

	if stats.averageGap != nil {
		fmt.Printf(
			"  Average gap between subtitles: %s (%.2fs)\n",
			subtitle.FormatTimestamp(*stats.averageGap),
			float64(int64(*stats.averageGap/time.Millisecond))/1000.0,
		)
	} else {
		fmt.Println("  Average gap between subtitles: N/A (no gaps or single subtitle)")
	}
	fmt.Println()

	fmt.Println("Text Statistics:")
	fmt.Printf("  Total characters: %d\n", stats.totalCharacters)

	if stats.subtitlesWithHTML > 0 {
		fmt.Printf(
			"  Total characters (without HTML tags): %d\n",
			stats.totalCharactersNoHTML,
		)
	}

	fmt.Printf("  Total words: %d\n", stats.totalWords)
	fmt.Printf("  Total lines: %d\n", stats.totalLines)

	if stats.totalCount > 0 {
		fmt.Printf(
			"  Average words per subtitle: %.1f\n",
			float64(stats.totalWords)/float64(stats.totalCount),
		)
		fmt.Printf(
			"  Average characters per subtitle: %.1f\n",
			float64(stats.totalCharactersNoHTML)/float64(stats.totalCount),
		)
	}
	fmt.Println()

	if stats.subtitlesWithHTML > 0 {
		fmt.Println("Formatting:")
		fmt.Printf(
			"  Subtitles with HTML tags: %d (%.1f%%)\n",
			stats.subtitlesWithHTML,
			(float64(stats.subtitlesWithHTML)/float64(stats.totalCount))*100.0,
		)
		fmt.Println()
	}
}

// infoFormatSubtitleError unwraps a service error into a *subtitle.SubtitleError
// and runs it through cli.FormatSubtitleError, mirroring the Rust
// `utils::format_subtitle_error` call.
func infoFormatSubtitleError(err error, file string) cli.CliError {
	var subErr *subtitle.SubtitleError
	if errors.As(err, &subErr) {
		return cli.FormatSubtitleError(subErr, file)
	}
	return cli.CliError{Code: "io_error", Message: err.Error()}
}
