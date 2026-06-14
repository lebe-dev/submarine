package merge

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

// --- model.go tests ---

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Strategy
		wantErr bool
	}{
		{name: "fill-gaps", input: "fill-gaps", want: FillGaps},
		{name: "keep-base", input: "keep-base", want: KeepBase},
		{name: "prefer-donor", input: "prefer-donor", want: PreferDonor},
		{name: "unknown", input: "merge", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "wrong case", input: "Fill-Gaps", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseStrategy(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseStrategy(%q): expected error, got %v", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStrategy(%q): unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseStrategy(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestStrategyString(t *testing.T) {
	tests := []struct {
		strategy Strategy
		want     string
	}{
		{FillGaps, "fill-gaps"},
		{KeepBase, "keep-base"},
		{PreferDonor, "prefer-donor"},
	}
	for _, tc := range tests {
		if got := tc.strategy.String(); got != tc.want {
			t.Errorf("Strategy(%d).String() = %q, want %q", tc.strategy, got, tc.want)
		}
	}
}

func TestFillGapsConstIsZero(t *testing.T) {
	if FillGaps != 0 {
		t.Errorf("expected FillGaps == 0, got %d", FillGaps)
	}
}

// --- helpers for assertions ---

// assertSortedAndRenumbered verifies the merged track is start-ascending and
// renumbered 1..N with no gaps.
func assertSortedAndRenumbered(t *testing.T, merged []subtitle.Subtitle) {
	t.Helper()
	for i := range merged {
		wantIdx := uint32(i + 1)
		if got := merged[i].Index.Value(); got != wantIdx {
			t.Errorf("merged[%d].Index = %d, want %d", i, got, wantIdx)
		}
		if i > 0 && merged[i].StartTime.Value() < merged[i-1].StartTime.Value() {
			t.Errorf("merged not sorted: merged[%d].start (%v) < merged[%d].start (%v)",
				i, merged[i].StartTime.Value(), i-1, merged[i-1].StartTime.Value())
		}
	}
}

// --- service.go tests ---

func TestMergeFillGapsAddsOnlyNonOverlapping(t *testing.T) {
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
		makeTestSubtitle(t, 2, 5000, 6000, "Base 2"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1500, 1800, "Donor overlaps base 1"), // overlaps -> skipped
		makeTestSubtitle(t, 2, 3000, 4000, "Donor fills gap"),       // no overlap -> added
	}

	res, err := Merge(base, donor, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if res.BaseCount != 2 || res.DonorCount != 2 {
		t.Errorf("counts: BaseCount=%d DonorCount=%d, want 2/2", res.BaseCount, res.DonorCount)
	}
	if res.Added != 1 {
		t.Errorf("Added = %d, want 1", res.Added)
	}
	if res.SkippedOverlapping != 1 {
		t.Errorf("SkippedOverlapping = %d, want 1", res.SkippedOverlapping)
	}
	if res.Replaced != 0 {
		t.Errorf("Replaced = %d, want 0", res.Replaced)
	}
	if res.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", res.TotalCount)
	}
	if len(res.AddedDonor) != 1 || res.AddedDonor[0].Text.Value() != "Donor fills gap" {
		t.Errorf("AddedDonor = %+v, want one 'Donor fills gap'", res.AddedDonor)
	}
	if len(res.SkippedDonor) != 1 || res.SkippedDonor[0].Text.Value() != "Donor overlaps base 1" {
		t.Errorf("SkippedDonor = %+v, want one 'Donor overlaps base 1'", res.SkippedDonor)
	}
	assertSortedAndRenumbered(t, res.Merged)
}

func TestMergeKeepBaseSameAsFillGaps(t *testing.T) {
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
		makeTestSubtitle(t, 2, 5000, 6000, "Base 2"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1500, 1800, "overlap"),
		makeTestSubtitle(t, 2, 3000, 4000, "gap"),
	}

	fill, err := Merge(base, donor, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge FillGaps: %v", err)
	}
	keep, err := Merge(base, donor, KeepBase, 0, 0)
	if err != nil {
		t.Fatalf("Merge KeepBase: %v", err)
	}

	if fill.Added != keep.Added || fill.SkippedOverlapping != keep.SkippedOverlapping ||
		fill.Replaced != keep.Replaced || fill.TotalCount != keep.TotalCount {
		t.Errorf("KeepBase result differs from FillGaps: fill=%+v keep=%+v", fill, keep)
	}
	if len(fill.Merged) != len(keep.Merged) {
		t.Fatalf("merged length differs: fill=%d keep=%d", len(fill.Merged), len(keep.Merged))
	}
	for i := range fill.Merged {
		if fill.Merged[i].Text.Value() != keep.Merged[i].Text.Value() ||
			fill.Merged[i].StartTime != keep.Merged[i].StartTime {
			t.Errorf("merged[%d] differs between strategies", i)
		}
	}
}

func TestMergeToleranceAbsorbsSmallOffset(t *testing.T) {
	// Base lines back-to-back; donor identical but shifted +50ms with no offset
	// correction. Without tolerance the donor would land in the seams and be
	// added; a tolerance of 100ms makes each donor line overlap its base line.
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Base 2"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1050, 2050, "Donor 1"),
		makeTestSubtitle(t, 2, 2050, 3050, "Donor 2"),
	}

	res, err := Merge(base, donor, FillGaps, 100, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if res.Added != 0 {
		t.Errorf("Added = %d, want 0 (tolerance should absorb the shift)", res.Added)
	}
	if res.SkippedOverlapping != 2 {
		t.Errorf("SkippedOverlapping = %d, want 2", res.SkippedOverlapping)
	}
	if res.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", res.TotalCount)
	}
}

func TestMergeOffsetShiftsDonorIntoOverlap(t *testing.T) {
	// Donor is 500ms early; an offset of +500ms aligns it onto the base line,
	// so with zero tolerance it now overlaps and is skipped under FillGaps.
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 500, 1500, "Donor 1"),
	}

	// Without offset: donor [500,1500] overlaps base [1000,2000] anyway, so use
	// a donor that only overlaps AFTER the offset.
	donor = []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 200, 400, "Donor early"), // [200,400] no overlap with [1000,2000]
	}

	noOffset, err := Merge(base, donor, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge no offset: %v", err)
	}
	if noOffset.Added != 1 {
		t.Errorf("no offset: Added = %d, want 1", noOffset.Added)
	}

	// Offset +900 -> donor becomes [1100,1300], overlapping base [1000,2000].
	withOffset, err := Merge(base, donor, FillGaps, 0, 900)
	if err != nil {
		t.Fatalf("Merge offset: %v", err)
	}
	if withOffset.Added != 0 || withOffset.SkippedOverlapping != 1 {
		t.Errorf("offset: Added=%d Skipped=%d, want 0/1", withOffset.Added, withOffset.SkippedOverlapping)
	}
}

func TestMergeNegativeOffsetClampAndDrop(t *testing.T) {
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 10000, 11000, "Base far"),
	}
	donor := []subtitle.Subtitle{
		// After offset -1000: [0, 200] kept (start clamped to 0, end > start).
		makeTestSubtitle(t, 1, 500, 1200, "Donor clamped"),
		// After offset -1000: start 200->... actually start 200-1000=-800 ->0,
		// end 900-1000=-100 -> end(-100) <= start(0) -> dropped.
		makeTestSubtitle(t, 2, 200, 900, "Donor dropped"),
	}

	res, err := Merge(base, donor, FillGaps, 0, -1000)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// One donor dropped during shift; the surviving one is far from base -> added.
	if res.Added != 1 {
		t.Errorf("Added = %d, want 1", res.Added)
	}
	if res.SkippedOverlapping != 0 {
		t.Errorf("SkippedOverlapping = %d, want 0", res.SkippedOverlapping)
	}
	if res.TotalCount != 2 { // base + 1 added donor
		t.Errorf("TotalCount = %d, want 2", res.TotalCount)
	}
	// The clamped donor must start at 0.
	if res.AddedDonor[0].StartTime.Millis() != 0 {
		t.Errorf("clamped donor start = %dms, want 0", res.AddedDonor[0].StartTime.Millis())
	}
	assertSortedAndRenumbered(t, res.Merged)
}

func TestMergePreferDonorReplacesOverlapping(t *testing.T) {
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Base 2"),
		makeTestSubtitle(t, 3, 5000, 6000, "Base 3"),
	}
	donor := []subtitle.Subtitle{
		// Spans base 1 and base 2 -> replaces both.
		makeTestSubtitle(t, 1, 1500, 2500, "Donor spans 1+2"),
		// No overlap -> added.
		makeTestSubtitle(t, 2, 7000, 8000, "Donor new"),
	}

	res, err := Merge(base, donor, PreferDonor, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if res.Replaced != 2 {
		t.Errorf("Replaced = %d, want 2", res.Replaced)
	}
	if res.Added != 1 {
		t.Errorf("Added = %d, want 1", res.Added)
	}
	// Survivors: base 3. Plus both donor lines added. Total = 1 + 2 = 3.
	if res.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", res.TotalCount)
	}
	// Replaced base lines must be gone from merged output.
	for _, s := range res.Merged {
		if s.Text.Value() == "Base 1" || s.Text.Value() == "Base 2" {
			t.Errorf("replaced base line %q still present", s.Text.Value())
		}
	}
	// Both donor lines are added under PreferDonor.
	if len(res.AddedDonor) != 2 {
		t.Errorf("AddedDonor len = %d, want 2", len(res.AddedDonor))
	}
	assertSortedAndRenumbered(t, res.Merged)
}

func TestMergePreferDonorSharedBaseCountedOnce(t *testing.T) {
	// Two donor lines both overlap the SAME base line; it must be removed once
	// (Replaced == 1), not twice.
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 5000, "Base wide"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1500, 2000, "Donor a"),
		makeTestSubtitle(t, 2, 3000, 3500, "Donor b"),
	}

	res, err := Merge(base, donor, PreferDonor, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if res.Replaced != 1 {
		t.Errorf("Replaced = %d, want 1", res.Replaced)
	}
	// Neither donor overlapped "nothing", so Added stays 0.
	if res.Added != 0 {
		t.Errorf("Added = %d, want 0", res.Added)
	}
	if res.TotalCount != 2 { // both donor lines, base removed
		t.Errorf("TotalCount = %d, want 2", res.TotalCount)
	}
}

func TestMergeDonorBeforeAndAfterBaseRange(t *testing.T) {
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 5000, 6000, "Base mid"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Donor before"), // new opening scene
		makeTestSubtitle(t, 2, 9000, 10000, "Donor after"), // new closing scene
	}

	for _, strat := range []Strategy{FillGaps, KeepBase, PreferDonor} {
		res, err := Merge(base, donor, strat, 0, 0)
		if err != nil {
			t.Fatalf("Merge(%s): %v", strat, err)
		}
		if res.Added != 2 {
			t.Errorf("%s: Added = %d, want 2", strat, res.Added)
		}
		if res.SkippedOverlapping != 0 {
			t.Errorf("%s: SkippedOverlapping = %d, want 0", strat, res.SkippedOverlapping)
		}
		if res.Replaced != 0 {
			t.Errorf("%s: Replaced = %d, want 0", strat, res.Replaced)
		}
		if res.TotalCount != 3 {
			t.Errorf("%s: TotalCount = %d, want 3", strat, res.TotalCount)
		}
		assertSortedAndRenumbered(t, res.Merged)
		// Verify the opening scene is first and the closing scene last.
		if res.Merged[0].Text.Value() != "Donor before" {
			t.Errorf("%s: first merged = %q, want 'Donor before'", strat, res.Merged[0].Text.Value())
		}
		if res.Merged[len(res.Merged)-1].Text.Value() != "Donor after" {
			t.Errorf("%s: last merged = %q, want 'Donor after'", strat, res.Merged[len(res.Merged)-1].Text.Value())
		}
	}
}

func TestMergeOutputSortedAndRenumbered(t *testing.T) {
	// Provide subtitles out of order across base/donor; result must be sorted
	// and renumbered 1..N with no gaps.
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 8000, 9000, "Base late"),
		makeTestSubtitle(t, 11, 1000, 2000, "Base early"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 4000, 5000, "Donor mid"),
	}

	res, err := Merge(base, donor, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if res.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", res.TotalCount)
	}
	assertSortedAndRenumbered(t, res.Merged)

	wantOrder := []string{"Base early", "Donor mid", "Base late"}
	for i, want := range wantOrder {
		if got := res.Merged[i].Text.Value(); got != want {
			t.Errorf("merged[%d] = %q, want %q", i, got, want)
		}
	}
}

func TestMergeEmptyDonor(t *testing.T) {
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Base 2"),
	}

	res, err := Merge(base, nil, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if res.Added != 0 || res.SkippedOverlapping != 0 || res.Replaced != 0 {
		t.Errorf("empty donor: Added=%d Skipped=%d Replaced=%d, want 0/0/0",
			res.Added, res.SkippedOverlapping, res.Replaced)
	}
	if res.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", res.TotalCount)
	}
	if res.DonorCount != 0 {
		t.Errorf("DonorCount = %d, want 0", res.DonorCount)
	}
	assertSortedAndRenumbered(t, res.Merged)
}

func TestMergeEmptyBase(t *testing.T) {
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Donor 1"),
		makeTestSubtitle(t, 2, 3000, 4000, "Donor 2"),
	}

	for _, strat := range []Strategy{FillGaps, KeepBase, PreferDonor} {
		res, err := Merge(nil, donor, strat, 0, 0)
		if err != nil {
			t.Fatalf("Merge(%s): %v", strat, err)
		}
		// No base -> every donor overlaps nothing -> all added.
		if res.Added != 2 {
			t.Errorf("%s: Added = %d, want 2", strat, res.Added)
		}
		if res.Replaced != 0 || res.SkippedOverlapping != 0 {
			t.Errorf("%s: Replaced=%d Skipped=%d, want 0/0", strat, res.Replaced, res.SkippedOverlapping)
		}
		if res.TotalCount != 2 {
			t.Errorf("%s: TotalCount = %d, want 2", strat, res.TotalCount)
		}
		if res.BaseCount != 0 {
			t.Errorf("%s: BaseCount = %d, want 0", strat, res.BaseCount)
		}
		assertSortedAndRenumbered(t, res.Merged)
	}
}

func TestMergeEmptyBoth(t *testing.T) {
	res, err := Merge(nil, nil, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if res.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", res.TotalCount)
	}
	if len(res.Merged) != 0 {
		t.Errorf("Merged len = %d, want 0", len(res.Merged))
	}
}

func TestMergeAdjacentNoToleranceNoOverlap(t *testing.T) {
	// Touching boundaries with zero tolerance must NOT count as overlap:
	// donor [2000,3000] vs base [1000,2000]: (2000-0) < 2000 is false.
	base := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Base 1"),
	}
	donor := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 2000, 3000, "Donor adjacent"),
	}

	res, err := Merge(base, donor, FillGaps, 0, 0)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if res.Added != 1 {
		t.Errorf("Added = %d, want 1 (adjacent is not overlap at zero tolerance)", res.Added)
	}
	if res.SkippedOverlapping != 0 {
		t.Errorf("SkippedOverlapping = %d, want 0", res.SkippedOverlapping)
	}
}
