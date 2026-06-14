package diff

import (
	"log/slog"
	"strings"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// Diff compares two subtitle sets A and B using the given mode. For ByText two
// subtitles match when their normalized text is equal; for ByTime they match
// when their timecodes overlap within toleranceMs milliseconds. Matching is
// greedy and each B-side subtitle is consumed at most once. Common holds the
// matched A-side subtitles; unmatched A subtitles go to OnlyInA and unconsumed
// B subtitles go to OnlyInB. Input order is preserved in every output list.
func Diff(a, b []subtitle.Subtitle, by ByMode, toleranceMs int64) DiffResult {
	slog.Debug("diffing subtitle sets",
		"a_count", len(a), "b_count", len(b), "by", by, "tolerance_ms", toleranceMs)

	consumed := make([]bool, len(b))

	var onlyInA []subtitle.Subtitle
	var common []subtitle.Subtitle

	for i := range a {
		matchPos := findMatch(a[i], b, consumed, by, toleranceMs)
		if matchPos < 0 {
			onlyInA = append(onlyInA, a[i])
			continue
		}
		consumed[matchPos] = true
		common = append(common, a[i])
	}

	var onlyInB []subtitle.Subtitle
	for j := range b {
		if !consumed[j] {
			onlyInB = append(onlyInB, b[j])
		}
	}

	slog.Debug("diff results",
		"only_in_a", len(onlyInA), "only_in_b", len(onlyInB), "common", len(common))

	return DiffResult{
		OnlyInA: onlyInA,
		OnlyInB: onlyInB,
		Common:  common,
	}
}

// findMatch returns the index of the first not-yet-consumed B subtitle that
// matches the given A subtitle under the mode, or -1 when none matches.
func findMatch(s subtitle.Subtitle, b []subtitle.Subtitle, consumed []bool, by ByMode, toleranceMs int64) int {
	for j := range b {
		if consumed[j] {
			continue
		}
		if matches(s, b[j], by, toleranceMs) {
			return j
		}
	}
	return -1
}

// matches reports whether two subtitles match under the given mode.
func matches(x, y subtitle.Subtitle, by ByMode, toleranceMs int64) bool {
	if by == ByText {
		return normalizeText(x.Text.Value()) == normalizeText(y.Text.Value())
	}
	return overlaps(x, y, toleranceMs)
}

// overlaps reports whether subtitle x overlaps subtitle y within toleranceMs:
// (x.start - tol) < y.end AND y.start < (x.end + tol).
func overlaps(x, y subtitle.Subtitle, toleranceMs int64) bool {
	tol := time.Duration(toleranceMs) * time.Millisecond
	return (x.StartTime.Value()-tol) < y.EndTime.Value() &&
		y.StartTime.Value() < (x.EndTime.Value()+tol)
}

// normalizeText normalizes subtitle text for content comparison: strip HTML
// tags (every run from '<' to the next '>' inclusive), lowercase, then collapse
// all whitespace runs to single spaces.
func normalizeText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(stripHTML(s))), " ")
}

// stripHTML removes every run from '<' to the next '>' inclusive. A '<' with no
// following '>' drops the remainder of the string.
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
