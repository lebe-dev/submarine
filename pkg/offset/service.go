// Package offset detects a time offset (and possible drift) between two sets of
// subtitles by matching them on their normalized dialogue text. It is part of
// the submarine subtitle translation toolkit.
package offset

import (
	"log/slog"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// Heuristic thresholds for interpreting the matched anchors.
const (
	// minAnchorRunes is the minimum normalized-text length (in runes) for a line
	// to qualify as an anchor. Short lines ("yes", "no", "ok") are too generic
	// to reliably align two files, so they are excluded.
	minAnchorRunes = 8

	// driftStddevMs: a population standard deviation above this many milliseconds
	// across the per-anchor diffs is treated as fps drift (the offset grows over
	// the timeline) rather than a single constant shift. Heuristic.
	driftStddevMs = 150

	// minDriftAnchors is the minimum number of matched anchors required before
	// drift can be asserted; a spread computed from a single pair is meaningless.
	// Heuristic.
	minDriftAnchors = 2

	// minSameVideoAnchors is the minimum number of matched anchors required to
	// conclude that both files describe the same video; enough shared dialogue
	// makes a coincidental match very unlikely. Heuristic.
	minSameVideoAnchors = 5
)

// OffsetReport summarizes the detected timing relationship between two subtitle
// files, computed from their shared anchor lines.
type OffsetReport struct {
	// AnchorMatches is the number of anchor lines shared by both files.
	AnchorMatches int
	// MedianOffsetMs is the median of the per-anchor start-time diffs, in
	// milliseconds. Zero when there are no matches.
	MedianOffsetMs int64
	// StddevMs is the population standard deviation of the diffs, in
	// milliseconds (rounded). Zero when there are no matches.
	StddevMs int64
	// DriftDetected reports whether the spread suggests fps drift rather than a
	// constant offset.
	DriftDetected bool
	// SameVideo reports whether the files share enough dialogue to conclude they
	// describe the same video.
	SameVideo bool
	// Diffs holds every per-anchor start-time diff (b - a), in milliseconds, in
	// anchor-text iteration order.
	Diffs []int64
}

// DetectOffset matches subtitles in a and b by their normalized text and
// reports the time offset (and possible drift) between the two files.
//
// An anchor candidate is a normalized text that is unique within its own file
// and at least minAnchorRunes runes long. For every text that is an anchor in
// both files, the start-time diff (b - a) is recorded. The median of those
// diffs is the detected offset; their spread (population standard deviation)
// distinguishes a constant shift from fps drift.
func DetectOffset(a, b []subtitle.Subtitle) OffsetReport {
	slog.Debug("detecting offset between subtitle sets", "a_count", len(a), "b_count", len(b))

	anchorsA := anchorMap(a)
	anchorsB := anchorMap(b)

	slog.Debug("anchor candidates", "anchors_a", len(anchorsA), "anchors_b", len(anchorsB))

	var diffs []int64
	for text, subA := range anchorsA {
		subB, ok := anchorsB[text]
		if !ok {
			continue
		}
		diffs = append(diffs, subB.StartTime.Millis()-subA.StartTime.Millis())
	}

	anchorMatches := len(diffs)
	slog.Debug("anchor matches", "count", anchorMatches)

	median := medianInt64(diffs)
	stddev := stddevInt64(diffs)

	driftDetected := anchorMatches >= minDriftAnchors && stddev > driftStddevMs
	sameVideo := anchorMatches >= minSameVideoAnchors

	slog.Debug("offset report",
		"anchor_matches", anchorMatches,
		"median_ms", median,
		"stddev_ms", stddev,
		"drift_detected", driftDetected,
		"same_video", sameVideo)

	return OffsetReport{
		AnchorMatches:  anchorMatches,
		MedianOffsetMs: median,
		StddevMs:       stddev,
		DriftDetected:  driftDetected,
		SameVideo:      sameVideo,
		Diffs:          diffs,
	}
}

// anchorMap builds a map from anchor normalized text to its subtitle. A text is
// an anchor when it is unique within subs and at least minAnchorRunes runes
// long. Non-unique normalized texts are dropped entirely.
func anchorMap(subs []subtitle.Subtitle) map[string]subtitle.Subtitle {
	counts := make(map[string]int, len(subs))
	first := make(map[string]subtitle.Subtitle, len(subs))

	for i := range subs {
		text := normalizeText(subs[i].Text.Value())
		if utf8.RuneCountInString(text) < minAnchorRunes {
			continue
		}
		counts[text]++
		if counts[text] == 1 {
			first[text] = subs[i]
		}
	}

	anchors := make(map[string]subtitle.Subtitle, len(first))
	for text, sub := range first {
		if counts[text] == 1 {
			anchors[text] = sub
		}
	}
	return anchors
}

// normalizeText canonicalizes subtitle text for matching: it strips HTML tags,
// lowercases the result and collapses all whitespace runs to single spaces.
func normalizeText(s string) string {
	stripped := stripHTML(s)
	lowered := strings.ToLower(stripped)
	return strings.Join(strings.Fields(lowered), " ")
}

// stripHTML removes every run from '<' to the next '>' (inclusive). A '<'
// without a closing '>' drops the remainder of the string.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, ch := range s {
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

// medianInt64 returns the median of values. For an odd count it is the middle
// element; for an even count it is the average of the two middle elements,
// rounded half away from zero. Returns 0 for an empty slice.
func medianInt64(values []int64) int64 {
	n := len(values)
	if n == 0 {
		return 0
	}

	sorted := make([]int64, n)
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	if n%2 == 1 {
		return sorted[n/2]
	}

	lo := sorted[n/2-1]
	hi := sorted[n/2]
	return roundHalfAwayFromZero(float64(lo+hi) / 2.0)
}

// stddevInt64 returns the population standard deviation of values, rounded half
// away from zero. Returns 0 for an empty slice.
func stddevInt64(values []int64) int64 {
	n := len(values)
	if n == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += float64(v)
	}
	mean := sum / float64(n)

	var variance float64
	for _, v := range values {
		d := float64(v) - mean
		variance += d * d
	}
	variance /= float64(n)

	return roundHalfAwayFromZero(math.Sqrt(variance))
}

// roundHalfAwayFromZero rounds f to the nearest integer, breaking ties away
// from zero (matching math.Round semantics).
func roundHalfAwayFromZero(f float64) int64 {
	return int64(math.Round(f))
}
