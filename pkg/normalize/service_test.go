package normalize

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

func TestNormalizeEmptyInput(t *testing.T) {
	result, err := Normalize(nil, true, true, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(result.Subtitles) != 0 {
		t.Errorf("expected 0 subtitles, got %d", len(result.Subtitles))
	}
	if !result.Sorted {
		t.Errorf("expected Sorted == true")
	}
	if !result.Renumbered {
		t.Errorf("expected Renumbered == true")
	}
	if result.OverlapsFixed != 0 {
		t.Errorf("expected OverlapsFixed == 0, got %d", result.OverlapsFixed)
	}
}

func TestNormalizeDoesNotMutateInput(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 5, 3000, 4000, "Third"),
		makeTestSubtitle(t, 6, 1000, 2000, "First"),
	}

	_, err := Normalize(subs, true, true, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	// Original slice order and indices must be untouched.
	if subs[0].Index.Value() != 5 || subs[0].StartTime.Millis() != 3000 {
		t.Errorf("input subs[0] was mutated: %+v", subs[0])
	}
	if subs[1].Index.Value() != 6 || subs[1].StartTime.Millis() != 1000 {
		t.Errorf("input subs[1] was mutated: %+v", subs[1])
	}
}

func TestNormalizeSort(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 3000, 4000, "Third"),
		makeTestSubtitle(t, 2, 1000, 2000, "First"),
		makeTestSubtitle(t, 3, 2000, 3000, "Second"),
	}

	result, err := Normalize(subs, true, false, false)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if !result.Sorted {
		t.Errorf("expected Sorted == true")
	}
	if result.Renumbered {
		t.Errorf("expected Renumbered == false")
	}

	wantStarts := []int64{1000, 2000, 3000}
	for i, want := range wantStarts {
		if got := result.Subtitles[i].StartTime.Millis(); got != want {
			t.Errorf("subs[%d].StartTime = %d, want %d", i, got, want)
		}
	}
	// Indices are untouched when not renumbering.
	wantIdx := []uint32{2, 3, 1}
	for i, want := range wantIdx {
		if got := result.Subtitles[i].Index.Value(); got != want {
			t.Errorf("subs[%d].Index = %d, want %d", i, got, want)
		}
	}
}

func TestNormalizeSortIsStable(t *testing.T) {
	// Two subtitles share the same start time; their relative order must be
	// preserved after a stable sort.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 1000, 1500, "A"),
		makeTestSubtitle(t, 11, 1000, 1600, "B"),
		makeTestSubtitle(t, 12, 500, 900, "C"),
	}

	result, err := Normalize(subs, true, false, false)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	wantIdx := []uint32{12, 10, 11}
	for i, want := range wantIdx {
		if got := result.Subtitles[i].Index.Value(); got != want {
			t.Errorf("subs[%d].Index = %d, want %d", i, got, want)
		}
	}
}

func TestNormalizeRenumber(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 7, 1000, 2000, "First"),
		makeTestSubtitle(t, 9, 2000, 3000, "Second"),
		makeTestSubtitle(t, 42, 3000, 4000, "Third"),
	}

	result, err := Normalize(subs, false, true, false)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if !result.Renumbered {
		t.Errorf("expected Renumbered == true")
	}

	for i := range result.Subtitles {
		want := uint32(i + 1)
		if got := result.Subtitles[i].Index.Value(); got != want {
			t.Errorf("subs[%d].Index = %d, want %d", i, got, want)
		}
	}
	// Timestamps and text preserved.
	if result.Subtitles[0].StartTime.Millis() != 1000 || result.Subtitles[0].Text.Value() != "First" {
		t.Errorf("renumber altered timestamps/text: %+v", result.Subtitles[0])
	}
}

func TestNormalizeFixOverlapTrimmedAndCounted(t *testing.T) {
	// First subtitle ends after the second starts; second start > first start,
	// so the first end is trimmed down to the second start.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2500, "First"),
		makeTestSubtitle(t, 2, 2000, 3000, "Second"),
	}

	result, err := Normalize(subs, false, false, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if result.OverlapsFixed != 1 {
		t.Errorf("expected OverlapsFixed == 1, got %d", result.OverlapsFixed)
	}
	if got := result.Subtitles[0].EndTime.Millis(); got != 2000 {
		t.Errorf("subs[0].EndTime = %d, want 2000 (trimmed to next start)", got)
	}
	// Second subtitle untouched.
	if result.Subtitles[1].StartTime.Millis() != 2000 || result.Subtitles[1].EndTime.Millis() != 3000 {
		t.Errorf("subs[1] was altered: %+v", result.Subtitles[1])
	}
}

func TestNormalizeFixOverlapNoOverlapNotCounted(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "First"),
		makeTestSubtitle(t, 2, 2000, 3000, "Second"),
	}

	result, err := Normalize(subs, false, false, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if result.OverlapsFixed != 0 {
		t.Errorf("expected OverlapsFixed == 0, got %d", result.OverlapsFixed)
	}
	if result.Subtitles[0].EndTime.Millis() != 2000 {
		t.Errorf("subs[0].EndTime altered: %d", result.Subtitles[0].EndTime.Millis())
	}
}

func TestNormalizeFixOverlapCannotTrimWithoutInverting(t *testing.T) {
	// Next start (1000) is at or before current start (1000), so trimming the
	// current end down to it would make end <= start. Leave unchanged, no count.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 5000, "First"),
		makeTestSubtitle(t, 2, 1000, 6000, "Second"),
	}

	result, err := Normalize(subs, false, false, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if result.OverlapsFixed != 0 {
		t.Errorf("expected OverlapsFixed == 0, got %d", result.OverlapsFixed)
	}
	if result.Subtitles[0].EndTime.Millis() != 5000 {
		t.Errorf("subs[0].EndTime should be unchanged (5000), got %d", result.Subtitles[0].EndTime.Millis())
	}
}

func TestNormalizeFixOverlapNextStartBeforeCurStartNotCounted(t *testing.T) {
	// Next start (800) is strictly before current start (1000). Trimming would
	// invert; leave unchanged, no count.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 5000, "First"),
		makeTestSubtitle(t, 2, 800, 6000, "Second"),
	}

	result, err := Normalize(subs, false, false, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if result.OverlapsFixed != 0 {
		t.Errorf("expected OverlapsFixed == 0, got %d", result.OverlapsFixed)
	}
	if result.Subtitles[0].EndTime.Millis() != 5000 {
		t.Errorf("subs[0].EndTime should be unchanged (5000), got %d", result.Subtitles[0].EndTime.Millis())
	}
}

func TestNormalizeAllOperations(t *testing.T) {
	// Out of order, non-contiguous indices, overlapping after sort.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 30, 3000, 4000, "Third"),
		makeTestSubtitle(t, 10, 1000, 2500, "First"), // overlaps Second after sort
		makeTestSubtitle(t, 20, 2000, 3000, "Second"),
	}

	result, err := Normalize(subs, true, true, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if !result.Sorted || !result.Renumbered {
		t.Errorf("expected Sorted and Renumbered == true")
	}
	if result.OverlapsFixed != 1 {
		t.Errorf("expected OverlapsFixed == 1, got %d", result.OverlapsFixed)
	}

	// Sorted by start time, renumbered 1..N contiguous.
	wantStarts := []int64{1000, 2000, 3000}
	wantEnds := []int64{2000, 3000, 4000} // first trimmed from 2500 -> 2000
	for i := range result.Subtitles {
		if got := result.Subtitles[i].Index.Value(); got != uint32(i+1) {
			t.Errorf("subs[%d].Index = %d, want %d", i, got, i+1)
		}
		if got := result.Subtitles[i].StartTime.Millis(); got != wantStarts[i] {
			t.Errorf("subs[%d].StartTime = %d, want %d", i, got, wantStarts[i])
		}
		if got := result.Subtitles[i].EndTime.Millis(); got != wantEnds[i] {
			t.Errorf("subs[%d].EndTime = %d, want %d", i, got, wantEnds[i])
		}
	}
}

func TestNormalizeChainedOverlaps(t *testing.T) {
	// Each subtitle overlaps the next; every adjacent pair should be trimmed.
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 3000, "A"),
		makeTestSubtitle(t, 2, 2000, 4000, "B"),
		makeTestSubtitle(t, 3, 3000, 5000, "C"),
	}

	result, err := Normalize(subs, true, false, true)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	if result.OverlapsFixed != 2 {
		t.Errorf("expected OverlapsFixed == 2, got %d", result.OverlapsFixed)
	}
	wantEnds := []int64{2000, 3000, 5000}
	for i, want := range wantEnds {
		if got := result.Subtitles[i].EndTime.Millis(); got != want {
			t.Errorf("subs[%d].EndTime = %d, want %d", i, got, want)
		}
	}
}
