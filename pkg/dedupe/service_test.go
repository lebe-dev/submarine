package dedupe

import (
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// makeTestSubtitle mirrors the verify package test helper.
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

func TestDedupeEmptyInput(t *testing.T) {
	res, err := Dedupe(nil, 0, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}
	if res.OriginalCount != 0 || res.FinalCount != 0 || res.Removed != 0 || res.Merged != 0 {
		t.Errorf("expected all-zero result, got %+v", res)
	}
	if res.Subtitles != nil {
		t.Errorf("expected nil Subtitles, got %v", res.Subtitles)
	}
}

func TestDedupeExactDuplicateNearTimecodes(t *testing.T) {
	// Two identical-text subtitles at near-identical timecodes collapse into one.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Hello world"),
		makeTestSubtitle(t, 2, 1010, 2010, "Hello world"),
	}

	res, err := Dedupe(subs, 50, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}

	if res.OriginalCount != 2 {
		t.Errorf("expected OriginalCount == 2, got %d", res.OriginalCount)
	}
	if res.FinalCount != 1 {
		t.Errorf("expected FinalCount == 1, got %d", res.FinalCount)
	}
	if res.Removed != 1 {
		t.Errorf("expected Removed == 1, got %d", res.Removed)
	}
	if res.Merged != 1 {
		t.Errorf("expected Merged == 1, got %d", res.Merged)
	}

	if len(res.Subtitles) != 1 {
		t.Fatalf("expected 1 subtitle, got %d", len(res.Subtitles))
	}
	got := res.Subtitles[0]
	if got.Index.Value() != 1 {
		t.Errorf("expected renumbered index 1, got %d", got.Index.Value())
	}
	// Span is [min start, max end] = [1000, 2010].
	if got.StartTime.Millis() != 1000 {
		t.Errorf("expected start 1000, got %d", got.StartTime.Millis())
	}
	if got.EndTime.Millis() != 2010 {
		t.Errorf("expected end 2010, got %d", got.EndTime.Millis())
	}
	// First subtitle's text is kept.
	if got.Text.Value() != "Hello world" {
		t.Errorf("expected text %q, got %q", "Hello world", got.Text.Value())
	}
}

func TestDedupeDistinctTextPreserved(t *testing.T) {
	// Same timecodes but different text => nothing merged.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Hello"),
		makeTestSubtitle(t, 2, 1000, 2000, "World"),
	}

	res, err := Dedupe(subs, 100, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}

	if res.FinalCount != 2 {
		t.Errorf("expected FinalCount == 2, got %d", res.FinalCount)
	}
	if res.Removed != 0 {
		t.Errorf("expected Removed == 0, got %d", res.Removed)
	}
	if res.Merged != 0 {
		t.Errorf("expected Merged == 0, got %d", res.Merged)
	}
	if len(res.Subtitles) != 2 {
		t.Fatalf("expected 2 subtitles, got %d", len(res.Subtitles))
	}
	if res.Subtitles[0].Index.Value() != 1 || res.Subtitles[1].Index.Value() != 2 {
		t.Errorf("expected renumbered indices 1,2, got %d,%d",
			res.Subtitles[0].Index.Value(), res.Subtitles[1].Index.Value())
	}
}

func TestDedupeToleranceControlsMerge(t *testing.T) {
	// Same text, gap between subtitle 1's end (2000) and subtitle 2's start
	// (2100) is 100ms. With tol=0 the intervals do not overlap (2000 < 2100
	// only when tol pushes the boundary), so they stay separate; with a tol
	// large enough to bridge the gap they merge.
	mk := func() []subtitle.Subtitle {
		return []subtitle.Subtitle{
			makeTestSubtitle(t, 1, 1000, 2000, "Repeated line"),
			makeTestSubtitle(t, 2, 2100, 3000, "Repeated line"),
		}
	}

	tests := []struct {
		name        string
		tolMs       int64
		wantFinal   int
		wantRemoved int
		wantMerged  int
	}{
		{name: "no tolerance keeps separate", tolMs: 0, wantFinal: 2, wantRemoved: 0, wantMerged: 0},
		{name: "small tolerance still separate", tolMs: 50, wantFinal: 2, wantRemoved: 0, wantMerged: 0},
		{name: "tolerance bridges gap", tolMs: 200, wantFinal: 1, wantRemoved: 1, wantMerged: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Dedupe(mk(), tc.tolMs, false)
			if err != nil {
				t.Fatalf("Dedupe: %v", err)
			}
			if res.FinalCount != tc.wantFinal {
				t.Errorf("FinalCount: want %d, got %d", tc.wantFinal, res.FinalCount)
			}
			if res.Removed != tc.wantRemoved {
				t.Errorf("Removed: want %d, got %d", tc.wantRemoved, res.Removed)
			}
			if res.Merged != tc.wantMerged {
				t.Errorf("Merged: want %d, got %d", tc.wantMerged, res.Merged)
			}
		})
	}
}

func TestDedupeIgnoreHTML(t *testing.T) {
	// "<i>hi</i>" vs "hi": identical once HTML is stripped, distinct otherwise.
	mk := func() []subtitle.Subtitle {
		return []subtitle.Subtitle{
			makeTestSubtitle(t, 1, 1000, 2000, "<i>hi</i>"),
			makeTestSubtitle(t, 2, 1010, 2010, "hi"),
		}
	}

	t.Run("ignoreHTML merges", func(t *testing.T) {
		res, err := Dedupe(mk(), 50, true)
		if err != nil {
			t.Fatalf("Dedupe: %v", err)
		}
		if res.FinalCount != 1 {
			t.Errorf("expected FinalCount == 1, got %d", res.FinalCount)
		}
		if res.Removed != 1 {
			t.Errorf("expected Removed == 1, got %d", res.Removed)
		}
		if res.Merged != 1 {
			t.Errorf("expected Merged == 1, got %d", res.Merged)
		}
		// First subtitle's (raw) text is kept.
		if res.Subtitles[0].Text.Value() != "<i>hi</i>" {
			t.Errorf("expected kept text %q, got %q", "<i>hi</i>", res.Subtitles[0].Text.Value())
		}
	})

	t.Run("default keeps distinct", func(t *testing.T) {
		res, err := Dedupe(mk(), 50, false)
		if err != nil {
			t.Fatalf("Dedupe: %v", err)
		}
		if res.FinalCount != 2 {
			t.Errorf("expected FinalCount == 2, got %d", res.FinalCount)
		}
		if res.Removed != 0 {
			t.Errorf("expected Removed == 0, got %d", res.Removed)
		}
		if res.Merged != 0 {
			t.Errorf("expected Merged == 0, got %d", res.Merged)
		}
	})
}

func TestDedupeNormalizationWhitespaceAndCase(t *testing.T) {
	// Differing case and whitespace collapse to the same normalized key.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Hello   World"),
		makeTestSubtitle(t, 2, 1005, 2005, "hello world"),
	}

	res, err := Dedupe(subs, 50, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}
	if res.FinalCount != 1 {
		t.Errorf("expected FinalCount == 1, got %d", res.FinalCount)
	}
	if res.Merged != 1 {
		t.Errorf("expected Merged == 1, got %d", res.Merged)
	}
}

func TestDedupeRunOfThreeMergesOnce(t *testing.T) {
	// Three mutually-duplicate neighbours collapse into one group (Merged == 1).
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Same"),
		makeTestSubtitle(t, 2, 1900, 3000, "Same"),
		makeTestSubtitle(t, 3, 2900, 4000, "Same"),
	}

	res, err := Dedupe(subs, 0, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}
	if res.FinalCount != 1 {
		t.Errorf("expected FinalCount == 1, got %d", res.FinalCount)
	}
	if res.Removed != 2 {
		t.Errorf("expected Removed == 2, got %d", res.Removed)
	}
	if res.Merged != 1 {
		t.Errorf("expected Merged == 1, got %d", res.Merged)
	}
	got := res.Subtitles[0]
	if got.StartTime.Millis() != 1000 || got.EndTime.Millis() != 4000 {
		t.Errorf("expected span [1000,4000], got [%d,%d]", got.StartTime.Millis(), got.EndTime.Millis())
	}
}

func TestDedupeUnsortedInputIsSorted(t *testing.T) {
	// Input out of StartTime order: duplicates only adjacent after sorting.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 3000, 4000, "Beta"),
		makeTestSubtitle(t, 2, 1000, 2000, "Alpha"),
		makeTestSubtitle(t, 3, 1010, 2010, "Alpha"),
	}

	res, err := Dedupe(subs, 50, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}
	if res.FinalCount != 2 {
		t.Errorf("expected FinalCount == 2, got %d", res.FinalCount)
	}
	if res.Merged != 1 {
		t.Errorf("expected Merged == 1, got %d", res.Merged)
	}
	// After sorting, the Alpha pair (starting at 1000) comes first.
	if res.Subtitles[0].Text.Value() != "Alpha" {
		t.Errorf("expected first kept text %q, got %q", "Alpha", res.Subtitles[0].Text.Value())
	}
	if res.Subtitles[1].Text.Value() != "Beta" {
		t.Errorf("expected second kept text %q, got %q", "Beta", res.Subtitles[1].Text.Value())
	}
	// Input slice must not be mutated (still in original order).
	if subs[0].Text.Value() != "Beta" {
		t.Errorf("input slice was mutated: %q", subs[0].Text.Value())
	}
}

func TestDedupeSameTextNoOverlapKept(t *testing.T) {
	// Same text but far apart in time => not duplicates, both kept.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Echo"),
		makeTestSubtitle(t, 2, 60000, 61000, "Echo"),
	}

	res, err := Dedupe(subs, 100, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}
	if res.FinalCount != 2 {
		t.Errorf("expected FinalCount == 2, got %d", res.FinalCount)
	}
	if res.Merged != 0 {
		t.Errorf("expected Merged == 0, got %d", res.Merged)
	}
}

func TestDedupeSingleSubtitle(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 7, 1000, 2000, "Only one"),
	}

	res, err := Dedupe(subs, 100, false)
	if err != nil {
		t.Fatalf("Dedupe: %v", err)
	}
	if res.FinalCount != 1 || res.Removed != 0 || res.Merged != 0 {
		t.Errorf("unexpected result %+v", res)
	}
	// Renumbered to 1.
	if res.Subtitles[0].Index.Value() != 1 {
		t.Errorf("expected renumbered index 1, got %d", res.Subtitles[0].Index.Value())
	}
}
