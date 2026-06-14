// Package dedupe contains the subtitle deduplication logic: it collapses runs
// of adjacent subtitles that share the same text and overlap (within a time
// tolerance) into a single subtitle spanning the combined interval.
package dedupe

import (
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// DedupeResult is the outcome of a deduplication run.
type DedupeResult struct {
	Subtitles     []subtitle.Subtitle
	OriginalCount int
	Removed       int
	Merged        int
	FinalCount    int
}

// Dedupe collapses adjacent duplicate subtitles into single entries.
//
// A copy of the input is sorted by StartTime. Two subtitles are duplicates iff
// their normalized text is equal AND their intervals are within tolerance, i.e.
// for tol = timeToleranceMs: (X.start - tol) < Y.end AND Y.start < (X.end + tol).
//
// When ignoreHTML is true, HTML tags are stripped before normalizing; the
// shared normalize already strips HTML, so the normalized text is HTML-free
// regardless. When ignoreHTML is false, comparison is done on the raw text
// (trimmed, lowercased and whitespace-collapsed) WITHOUT stripping HTML.
//
// Each run of mutually-duplicate neighbours is greedily merged into ONE subtitle
// spanning [min start, max end], keeping the FIRST subtitle's text. Merged is
// the number of groups that absorbed at least one extra subtitle; Removed is
// OriginalCount - FinalCount. The result is renumbered 1..N.
func Dedupe(subs []subtitle.Subtitle, timeToleranceMs int64, ignoreHTML bool) (DedupeResult, error) {
	originalCount := len(subs)

	slog.Debug("deduplicating subtitles",
		"original_count", originalCount,
		"time_tolerance_ms", timeToleranceMs,
		"ignore_html", ignoreHTML)

	if originalCount == 0 {
		return DedupeResult{
			Subtitles:     nil,
			OriginalCount: 0,
			Removed:       0,
			Merged:        0,
			FinalCount:    0,
		}, nil
	}

	// Sort a copy by StartTime so we never mutate the caller's slice.
	sorted := make([]subtitle.Subtitle, len(subs))
	copy(sorted, subs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Millis() < sorted[j].StartTime.Millis()
	})

	merged := 0
	var result []subtitle.Subtitle

	i := 0
	for i < len(sorted) {
		// Start a new group with the current subtitle as its head.
		head := sorted[i]
		headKey := dedupeKey(head, ignoreHTML)
		minStart := head.StartTime.Millis()
		maxEnd := head.EndTime.Millis()
		absorbed := 0

		j := i + 1
		for j < len(sorted) {
			next := sorted[j]
			if dedupeKey(next, ignoreHTML) != headKey {
				break
			}
			if !overlapsWithin(minStart, maxEnd, next.StartTime.Millis(), next.EndTime.Millis(), timeToleranceMs) {
				break
			}

			// Absorb the neighbour into the running span.
			if next.StartTime.Millis() < minStart {
				minStart = next.StartTime.Millis()
			}
			if next.EndTime.Millis() > maxEnd {
				maxEnd = next.EndTime.Millis()
			}
			absorbed++
			j++
		}

		if absorbed > 0 {
			merged++
			slog.Debug("merged duplicate run",
				"head_index", head.Index.Value(),
				"absorbed", absorbed,
				"min_start_ms", minStart,
				"max_end_ms", maxEnd)
		}

		mergedSub, err := buildMerged(head, minStart, maxEnd)
		if err != nil {
			return DedupeResult{}, err
		}
		result = append(result, mergedSub)

		i = j
	}

	// Renumber 1..N.
	renumbered := make([]subtitle.Subtitle, 0, len(result))
	for idx, s := range result {
		newIndex, err := subtitle.NewSubtitleIndex(uint32(idx + 1))
		if err != nil {
			return DedupeResult{}, err
		}
		ns, err := subtitle.NewSubtitle(newIndex, s.StartTime, s.EndTime, s.Text)
		if err != nil {
			return DedupeResult{}, err
		}
		renumbered = append(renumbered, ns)
	}

	finalCount := len(renumbered)

	slog.Info("deduplication complete",
		"original_count", originalCount,
		"final_count", finalCount,
		"removed", originalCount-finalCount,
		"merged", merged)

	return DedupeResult{
		Subtitles:     renumbered,
		OriginalCount: originalCount,
		Removed:       originalCount - finalCount,
		Merged:        merged,
		FinalCount:    finalCount,
	}, nil
}

// buildMerged constructs the merged subtitle for a group, keeping the head's
// text and spanning [minStartMs, maxEndMs]. The head's index is reused here and
// later overwritten by the renumbering pass.
func buildMerged(head subtitle.Subtitle, minStartMs, maxEndMs int64) (subtitle.Subtitle, error) {
	start, err := subtitle.NewSubtitleTimestamp(millis(minStartMs))
	if err != nil {
		return subtitle.Subtitle{}, err
	}
	end, err := subtitle.NewSubtitleTimestamp(millis(maxEndMs))
	if err != nil {
		return subtitle.Subtitle{}, err
	}
	return subtitle.NewSubtitle(head.Index, start, end, head.Text)
}

// overlapsWithin reports whether intervals [xStart,xEnd] and [yStart,yEnd]
// overlap within tolerance tolMs: (xStart - tol) < yEnd AND yStart < (xEnd + tol).
func overlapsWithin(xStartMs, xEndMs, yStartMs, yEndMs, tolMs int64) bool {
	return (xStartMs-tolMs) < yEndMs && yStartMs < (xEndMs+tolMs)
}

// dedupeKey returns the comparison key for a subtitle. When ignoreHTML is true
// the text is normalized (which strips HTML); otherwise the raw text is
// lowercased and whitespace-collapsed without stripping HTML.
func dedupeKey(s subtitle.Subtitle, ignoreHTML bool) string {
	if ignoreHTML {
		return normalizeText(s.Text.Value())
	}
	return normalizeTextKeepHTML(s.Text.Value())
}

// normalizeText produces the shared normalized form of a piece of text:
//  1. strip HTML: remove every run from "<" to the next ">" inclusive;
//  2. lowercase via strings.ToLower;
//  3. collapse whitespace: strings.Fields then strings.Join(..., " ").
func normalizeText(text string) string {
	return collapseWhitespace(strings.ToLower(stripHTML(text)))
}

// normalizeTextKeepHTML is normalizeText without the HTML-stripping step: it
// lowercases and whitespace-collapses the raw text.
func normalizeTextKeepHTML(text string) string {
	return collapseWhitespace(strings.ToLower(text))
}

// stripHTML removes every run from "<" to the next ">" inclusive.
func stripHTML(text string) string {
	var b strings.Builder
	inTag := false
	for _, ch := range text {
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
		case !inTag:
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// collapseWhitespace collapses all runs of whitespace into single spaces and
// trims leading/trailing whitespace (strings.Fields + strings.Join).
func collapseWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// millis converts a millisecond count into a time.Duration at ms resolution.
func millis(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
