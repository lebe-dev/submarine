package offset

import (
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// makeTestSubtitle mirrors the verify-package test helper.
func makeTestSubtitle(t *testing.T, index uint32, startMs, endMs int64, text string) subtitle.Subtitle {
	t.Helper()

	idx, err := subtitle.NewSubtitleIndex(index)
	if err != nil {
		t.Fatalf("NewSubtitleIndex(%d): %v", index, err)
	}
	start, err := subtitle.NewSubtitleTimestamp(time.Duration(startMs) * time.Millisecond)
	if err != nil {
		t.Fatalf("NewSubtitleTimestamp(%d): %v", startMs, err)
	}
	end, err := subtitle.NewSubtitleTimestamp(time.Duration(endMs) * time.Millisecond)
	if err != nil {
		t.Fatalf("NewSubtitleTimestamp(%d): %v", endMs, err)
	}
	txt, err := subtitle.NewSubtitleText(text)
	if err != nil {
		t.Fatalf("NewSubtitleText(%q): %v", text, err)
	}
	sub, err := subtitle.NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("NewSubtitle: %v", err)
	}
	return sub
}

// shift returns a copy of subs with every start/end time shifted by deltaMs and
// the text preserved, so the lines remain anchors that match the original.
func shift(t *testing.T, subs []subtitle.Subtitle, deltaMs int64) []subtitle.Subtitle {
	t.Helper()
	out := make([]subtitle.Subtitle, 0, len(subs))
	for _, s := range subs {
		out = append(out, makeTestSubtitle(t,
			s.Index.Value(),
			s.StartTime.Millis()+deltaMs,
			s.EndTime.Millis()+deltaMs,
			s.Text.Value()))
	}
	return out
}

// sixDistinctLines returns six unique, long-enough anchor lines (>= 8 runes).
func sixDistinctLines(t *testing.T) []subtitle.Subtitle {
	t.Helper()
	return []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "The quick brown fox jumps"),
		makeTestSubtitle(t, 2, 3000, 4000, "Over the lazy sleeping dog"),
		makeTestSubtitle(t, 3, 5000, 6000, "Pack my box with five jugs"),
		makeTestSubtitle(t, 4, 7000, 8000, "How vexingly quick daft zebras"),
		makeTestSubtitle(t, 5, 9000, 10000, "Sphinx of black quartz judge"),
		makeTestSubtitle(t, 6, 11000, 12000, "The five boxing wizards jump"),
	}
}

// --- normalizeText / anchorMap unit tests ---

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "Hello World", "hello world"},
		{"html strip", "<i>Hello</i> <b>World</b>", "hello world"},
		{"collapse whitespace", "Hello   \t\n World", "hello world"},
		{"leading trailing", "   Hello World   ", "hello world"},
		{"font tag", "<font color=\"red\">Run!</font>", "run!"},
		{"unterminated tag drops rest", "Hello <not closed", "hello"},
		{"empty", "", ""},
		{"only tags", "<i></i>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeText(tt.in); got != tt.want {
				t.Errorf("normalizeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMedianInt64(t *testing.T) {
	tests := []struct {
		name string
		in   []int64
		want int64
	}{
		{"empty", nil, 0},
		{"single", []int64{42}, 42},
		{"odd", []int64{3, 1, 2}, 2},
		{"even average", []int64{10, 20}, 15},
		{"even round half away positive", []int64{10, 11}, 11},    // 10.5 -> 11
		{"even round half away negative", []int64{-11, -10}, -11}, // -10.5 -> -11
		{"even exact", []int64{-200, -224}, -212},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := medianInt64(tt.in); got != tt.want {
				t.Errorf("medianInt64(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestStddevInt64(t *testing.T) {
	tests := []struct {
		name string
		in   []int64
		want int64
	}{
		{"empty", nil, 0},
		{"single", []int64{5}, 0},
		{"identical", []int64{7, 7, 7}, 0},
		{"spread", []int64{0, 200}, 100}, // population stddev of {0,200} = 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stddevInt64(tt.in); got != tt.want {
				t.Errorf("stddevInt64(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestAnchorMapExcludesShortAndDuplicate(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Yes"),                           // too short
		makeTestSubtitle(t, 2, 3000, 4000, "A unique and long anchor line"), // anchor
		makeTestSubtitle(t, 3, 5000, 6000, "Repeated dialogue here"),        // duplicate (1/2)
		makeTestSubtitle(t, 4, 7000, 8000, "Repeated dialogue here"),        // duplicate (2/2)
		makeTestSubtitle(t, 5, 9000, 10000, "<i>Yes</i>"),                   // too short after strip
	}

	anchors := anchorMap(subs)

	if len(anchors) != 1 {
		t.Fatalf("expected exactly 1 anchor, got %d: %v", len(anchors), keys(anchors))
	}
	if _, ok := anchors["a unique and long anchor line"]; !ok {
		t.Errorf("expected the unique long line to be an anchor, got %v", keys(anchors))
	}
}

func keys(m map[string]subtitle.Subtitle) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// --- DetectOffset tests ---

func TestDetectOffsetIdenticalFiles(t *testing.T) {
	subs := sixDistinctLines(t)

	report := DetectOffset(subs, subs)

	if report.AnchorMatches != 6 {
		t.Errorf("expected AnchorMatches == 6, got %d", report.AnchorMatches)
	}
	if report.MedianOffsetMs != 0 {
		t.Errorf("expected MedianOffsetMs == 0, got %d", report.MedianOffsetMs)
	}
	if report.StddevMs != 0 {
		t.Errorf("expected StddevMs == 0, got %d", report.StddevMs)
	}
	if !report.SameVideo {
		t.Errorf("expected SameVideo == true")
	}
	if report.DriftDetected {
		t.Errorf("expected DriftDetected == false")
	}
}

func TestDetectOffsetConstantShift(t *testing.T) {
	a := sixDistinctLines(t)
	b := shift(t, a, -212)

	report := DetectOffset(a, b)

	if report.AnchorMatches != 6 {
		t.Errorf("expected AnchorMatches == 6, got %d", report.AnchorMatches)
	}
	if report.MedianOffsetMs != -212 {
		t.Errorf("expected MedianOffsetMs == -212, got %d", report.MedianOffsetMs)
	}
	if report.StddevMs != 0 {
		t.Errorf("expected StddevMs == 0 (constant shift), got %d", report.StddevMs)
	}
	if report.DriftDetected {
		t.Errorf("expected DriftDetected == false for a constant shift")
	}
	if !report.SameVideo {
		t.Errorf("expected SameVideo == true")
	}
}

func TestDetectOffsetShortAndDuplicateExcluded(t *testing.T) {
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Ok"),                     // too short
		makeTestSubtitle(t, 2, 3000, 4000, "Shared anchor sentence"), // anchor
		makeTestSubtitle(t, 3, 5000, 6000, "Duplicate line text"),    // duplicate (excluded)
		makeTestSubtitle(t, 4, 7000, 8000, "Duplicate line text"),    // duplicate (excluded)
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1100, 2100, "Ok"),                     // too short
		makeTestSubtitle(t, 2, 3100, 4100, "Shared anchor sentence"), // anchor
		makeTestSubtitle(t, 3, 5100, 6100, "Duplicate line text"),    // duplicate (excluded)
		makeTestSubtitle(t, 4, 7100, 8100, "Duplicate line text"),    // duplicate (excluded)
	}

	report := DetectOffset(a, b)

	if report.AnchorMatches != 1 {
		t.Errorf("expected AnchorMatches == 1 (only the unique long line), got %d", report.AnchorMatches)
	}
	if len(report.Diffs) != 1 || report.Diffs[0] != 100 {
		t.Errorf("expected Diffs == [100], got %v", report.Diffs)
	}
	if report.MedianOffsetMs != 100 {
		t.Errorf("expected MedianOffsetMs == 100, got %d", report.MedianOffsetMs)
	}
	if report.SameVideo {
		t.Errorf("expected SameVideo == false (only 1 anchor)")
	}
	if report.DriftDetected {
		t.Errorf("expected DriftDetected == false (only 1 anchor)")
	}
}

func TestDetectOffsetNoSharedText(t *testing.T) {
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Alpha bravo charlie delta"),
		makeTestSubtitle(t, 2, 3000, 4000, "Echo foxtrot golf hotel"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "India juliett kilo lima"),
		makeTestSubtitle(t, 2, 3000, 4000, "Mike november oscar papa"),
	}

	report := DetectOffset(a, b)

	if report.AnchorMatches != 0 {
		t.Errorf("expected AnchorMatches == 0, got %d", report.AnchorMatches)
	}
	if report.MedianOffsetMs != 0 {
		t.Errorf("expected MedianOffsetMs == 0, got %d", report.MedianOffsetMs)
	}
	if report.StddevMs != 0 {
		t.Errorf("expected StddevMs == 0, got %d", report.StddevMs)
	}
	if report.SameVideo {
		t.Errorf("expected SameVideo == false")
	}
	if report.DriftDetected {
		t.Errorf("expected DriftDetected == false")
	}
}

func TestDetectOffsetEmptyInput(t *testing.T) {
	report := DetectOffset(nil, nil)

	if report.AnchorMatches != 0 {
		t.Errorf("expected AnchorMatches == 0, got %d", report.AnchorMatches)
	}
	if report.MedianOffsetMs != 0 || report.StddevMs != 0 {
		t.Errorf("expected median/stddev 0, got %d/%d", report.MedianOffsetMs, report.StddevMs)
	}
	if report.SameVideo || report.DriftDetected {
		t.Errorf("expected SameVideo and DriftDetected == false")
	}
	if len(report.Diffs) != 0 {
		t.Errorf("expected no Diffs, got %v", report.Diffs)
	}
}

func TestDetectOffsetDriftDetected(t *testing.T) {
	// Five anchor lines whose offset grows steadily (fps drift): the diffs span
	// a wide range, so the population stddev exceeds the 150ms threshold.
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Anchor sentence number one"),
		makeTestSubtitle(t, 2, 3000, 4000, "Anchor sentence number two"),
		makeTestSubtitle(t, 3, 5000, 6000, "Anchor sentence number three"),
		makeTestSubtitle(t, 4, 7000, 8000, "Anchor sentence number four"),
		makeTestSubtitle(t, 5, 9000, 10000, "Anchor sentence number five"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Anchor sentence number one"),   // diff 0
		makeTestSubtitle(t, 2, 3200, 4200, "Anchor sentence number two"),   // diff 200
		makeTestSubtitle(t, 3, 5400, 6400, "Anchor sentence number three"), // diff 400
		makeTestSubtitle(t, 4, 7600, 8600, "Anchor sentence number four"),  // diff 600
		makeTestSubtitle(t, 5, 9800, 10800, "Anchor sentence number five"), // diff 800
	}

	report := DetectOffset(a, b)

	if report.AnchorMatches != 5 {
		t.Errorf("expected AnchorMatches == 5, got %d", report.AnchorMatches)
	}
	if report.MedianOffsetMs != 400 {
		t.Errorf("expected MedianOffsetMs == 400, got %d", report.MedianOffsetMs)
	}
	if report.StddevMs <= driftStddevMs {
		t.Errorf("expected StddevMs > %d, got %d", driftStddevMs, report.StddevMs)
	}
	if !report.DriftDetected {
		t.Errorf("expected DriftDetected == true")
	}
	if !report.SameVideo {
		t.Errorf("expected SameVideo == true")
	}
}
