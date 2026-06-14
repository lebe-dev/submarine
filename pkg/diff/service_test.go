package diff

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

// indices extracts the index values from a slice of subtitles, for assertions.
func indices(subs []subtitle.Subtitle) []uint32 {
	out := make([]uint32, 0, len(subs))
	for i := range subs {
		out = append(out, subs[i].Index.Value())
	}
	return out
}

// equalIndices compares an index slice against an expected list.
func equalIndices(got []subtitle.Subtitle, want []uint32) bool {
	g := indices(got)
	if len(g) != len(want) {
		return false
	}
	for i := range g {
		if g[i] != want[i] {
			return false
		}
	}
	return true
}

// --- ParseByMode ---

func TestParseByMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ByMode
		wantErr bool
	}{
		{name: "time", input: "time", want: ByTime},
		{name: "text", input: "text", want: ByText},
		{name: "unknown", input: "frame", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "case-sensitive", input: "Time", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseByMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseByMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestByModeConstants(t *testing.T) {
	if ByTime != 0 {
		t.Errorf("expected ByTime == 0, got %d", ByTime)
	}
	if ByText != 1 {
		t.Errorf("expected ByText == 1, got %d", ByText)
	}
}

// --- normalizeText ---

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "Hello World", want: "hello world"},
		{name: "strips html", input: "<i>Hello</i> <b>World</b>", want: "hello world"},
		{name: "collapses whitespace", input: "Hello   \t World\n", want: "hello world"},
		{name: "leading trailing space", input: "  spaced  ", want: "spaced"},
		{name: "empty", input: "", want: ""},
		{name: "only tags", input: "<i></i>", want: ""},
		{name: "unclosed tag drops rest", input: "keep<this is dropped", want: "keep"},
		{name: "multiline", input: "line one\nline two", want: "line one line two"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeText(tt.input)
			if got != tt.want {
				t.Errorf("normalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Diff: ByText ---

func TestDiffByTextFindsContentDifferences(t *testing.T) {
	// Same content but different indices and timecodes must still match.
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "<i>Hello</i> world"),
		makeTestSubtitle(t, 2, 2000, 3000, "Unique to A"),
		makeTestSubtitle(t, 3, 3000, 4000, "Shared line"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 50000, 51000, "hello   WORLD"), // matches a[0] after normalize
		makeTestSubtitle(t, 11, 51000, 52000, "shared LINE"),   // matches a[2]
		makeTestSubtitle(t, 12, 52000, 53000, "Unique to B"),
	}

	res := Diff(a, b, ByText, 0)

	if !equalIndices(res.Common, []uint32{1, 3}) {
		t.Errorf("Common indices = %v, want [1 3]", indices(res.Common))
	}
	if !equalIndices(res.OnlyInA, []uint32{2}) {
		t.Errorf("OnlyInA indices = %v, want [2]", indices(res.OnlyInA))
	}
	if !equalIndices(res.OnlyInB, []uint32{12}) {
		t.Errorf("OnlyInB indices = %v, want [12]", indices(res.OnlyInB))
	}
}

func TestDiffByTextDonorOnlyLineShowsInOnlyInB(t *testing.T) {
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Common line"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Common line"),
		makeTestSubtitle(t, 2, 2000, 3000, "Donor only"),
	}

	res := Diff(a, b, ByText, 0)

	if !equalIndices(res.Common, []uint32{1}) {
		t.Errorf("Common indices = %v, want [1]", indices(res.Common))
	}
	if len(res.OnlyInA) != 0 {
		t.Errorf("OnlyInA = %v, want empty", indices(res.OnlyInA))
	}
	if !equalIndices(res.OnlyInB, []uint32{2}) {
		t.Errorf("OnlyInB indices = %v, want [2]", indices(res.OnlyInB))
	}
}

func TestDiffByTextGreedyConsumesEachBOnce(t *testing.T) {
	// Two identical A lines, but only one matching B line: the second A is OnlyInA.
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "duplicate"),
		makeTestSubtitle(t, 2, 2000, 3000, "duplicate"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "duplicate"),
	}

	res := Diff(a, b, ByText, 0)

	if !equalIndices(res.Common, []uint32{1}) {
		t.Errorf("Common indices = %v, want [1]", indices(res.Common))
	}
	if !equalIndices(res.OnlyInA, []uint32{2}) {
		t.Errorf("OnlyInA indices = %v, want [2]", indices(res.OnlyInA))
	}
	if len(res.OnlyInB) != 0 {
		t.Errorf("OnlyInB = %v, want empty", indices(res.OnlyInB))
	}
}

// --- Diff: ByTime ---

func TestDiffByTimePairsOverlappingWithinTolerance(t *testing.T) {
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "A one"),
		makeTestSubtitle(t, 2, 5000, 6000, "A two"),
		makeTestSubtitle(t, 3, 9000, 10000, "A three"),
	}
	b := []subtitle.Subtitle{
		// overlaps a[0] directly
		makeTestSubtitle(t, 10, 1500, 2500, "B one"),
		// 200ms after a[1] ends; within 300ms tolerance
		makeTestSubtitle(t, 11, 6200, 7000, "B two"),
		// far away from any A line
		makeTestSubtitle(t, 12, 50000, 51000, "B donor"),
	}

	res := Diff(a, b, ByTime, 300)

	if !equalIndices(res.Common, []uint32{1, 2}) {
		t.Errorf("Common indices = %v, want [1 2]", indices(res.Common))
	}
	if !equalIndices(res.OnlyInA, []uint32{3}) {
		t.Errorf("OnlyInA indices = %v, want [3]", indices(res.OnlyInA))
	}
	if !equalIndices(res.OnlyInB, []uint32{12}) {
		t.Errorf("OnlyInB indices = %v, want [12]", indices(res.OnlyInB))
	}
}

func TestDiffByTimeToleranceBoundary(t *testing.T) {
	// a: [1000,2000]. b: [2300,3300]. Gap is 300ms.
	// overlap(x,y) = (x.start - tol) < y.end AND y.start < (x.end + tol)
	//   second condition: 2300 < 2000 + tol => tol > 300 to match.
	a := []subtitle.Subtitle{makeTestSubtitle(t, 1, 1000, 2000, "A")}

	tests := []struct {
		name      string
		tolerance int64
		wantMatch bool
	}{
		{name: "tol below gap", tolerance: 299, wantMatch: false},
		{name: "tol equals gap (exclusive)", tolerance: 300, wantMatch: false},
		{name: "tol above gap", tolerance: 301, wantMatch: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := []subtitle.Subtitle{makeTestSubtitle(t, 1, 2300, 3300, "B")}
			res := Diff(a, b, ByTime, tt.tolerance)
			gotMatch := len(res.Common) == 1
			if gotMatch != tt.wantMatch {
				t.Errorf("tolerance %d: match=%v, want %v (Common=%v)",
					tt.tolerance, gotMatch, tt.wantMatch, indices(res.Common))
			}
		})
	}
}

func TestDiffByTimeGreedyConsumesEachBOnce(t *testing.T) {
	// Both A lines overlap the single B line; only the first A consumes it.
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 3000, "A one"),
		makeTestSubtitle(t, 2, 2000, 4000, "A two"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 9, 2500, 2800, "B mid"),
	}

	res := Diff(a, b, ByTime, 0)

	if !equalIndices(res.Common, []uint32{1}) {
		t.Errorf("Common indices = %v, want [1]", indices(res.Common))
	}
	if !equalIndices(res.OnlyInA, []uint32{2}) {
		t.Errorf("OnlyInA indices = %v, want [2]", indices(res.OnlyInA))
	}
	if len(res.OnlyInB) != 0 {
		t.Errorf("OnlyInB = %v, want empty", indices(res.OnlyInB))
	}
}

// --- shared edge cases ---

func TestDiffIdenticalFiles(t *testing.T) {
	mk := func() []subtitle.Subtitle {
		return []subtitle.Subtitle{
			makeTestSubtitle(t, 1, 1000, 2000, "Line one"),
			makeTestSubtitle(t, 2, 2000, 3000, "Line two"),
			makeTestSubtitle(t, 3, 3000, 4000, "Line three"),
		}
	}

	for _, by := range []ByMode{ByText, ByTime} {
		res := Diff(mk(), mk(), by, 0)
		if len(res.OnlyInA) != 0 {
			t.Errorf("mode %v: OnlyInA = %v, want empty", by, indices(res.OnlyInA))
		}
		if len(res.OnlyInB) != 0 {
			t.Errorf("mode %v: OnlyInB = %v, want empty", by, indices(res.OnlyInB))
		}
		if !equalIndices(res.Common, []uint32{1, 2, 3}) {
			t.Errorf("mode %v: Common indices = %v, want [1 2 3]", by, indices(res.Common))
		}
	}
}

func TestDiffEmptyInputs(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Only A"),
	}

	t.Run("both empty", func(t *testing.T) {
		res := Diff(nil, nil, ByText, 0)
		if len(res.OnlyInA) != 0 || len(res.OnlyInB) != 0 || len(res.Common) != 0 {
			t.Errorf("expected all empty, got OnlyInA=%v OnlyInB=%v Common=%v",
				indices(res.OnlyInA), indices(res.OnlyInB), indices(res.Common))
		}
	})

	t.Run("empty b -> all in OnlyInA", func(t *testing.T) {
		res := Diff(subs, nil, ByTime, 100)
		if !equalIndices(res.OnlyInA, []uint32{1}) {
			t.Errorf("OnlyInA = %v, want [1]", indices(res.OnlyInA))
		}
		if len(res.OnlyInB) != 0 || len(res.Common) != 0 {
			t.Errorf("expected OnlyInB and Common empty, got OnlyInB=%v Common=%v",
				indices(res.OnlyInB), indices(res.Common))
		}
	})

	t.Run("empty a -> all in OnlyInB", func(t *testing.T) {
		res := Diff(nil, subs, ByTime, 100)
		if !equalIndices(res.OnlyInB, []uint32{1}) {
			t.Errorf("OnlyInB = %v, want [1]", indices(res.OnlyInB))
		}
		if len(res.OnlyInA) != 0 || len(res.Common) != 0 {
			t.Errorf("expected OnlyInA and Common empty, got OnlyInA=%v Common=%v",
				indices(res.OnlyInA), indices(res.Common))
		}
	})
}

func TestDiffCommonHoldsASideSubtitles(t *testing.T) {
	// Confirm Common carries the A-side subtitle (its index/timecodes), not B's.
	a := []subtitle.Subtitle{makeTestSubtitle(t, 7, 1000, 2000, "match")}
	b := []subtitle.Subtitle{makeTestSubtitle(t, 99, 80000, 81000, "MATCH")}

	res := Diff(a, b, ByText, 0)

	if len(res.Common) != 1 {
		t.Fatalf("Common len = %d, want 1", len(res.Common))
	}
	if res.Common[0].Index.Value() != 7 {
		t.Errorf("Common[0] index = %d, want 7 (A-side)", res.Common[0].Index.Value())
	}
	if res.Common[0].StartTime.Millis() != 1000 {
		t.Errorf("Common[0] start = %dms, want 1000 (A-side)", res.Common[0].StartTime.Millis())
	}
}

func TestDiffPreservesOrder(t *testing.T) {
	a := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "alpha"),
		makeTestSubtitle(t, 2, 2000, 3000, "beta"),
		makeTestSubtitle(t, 3, 3000, 4000, "gamma"),
		makeTestSubtitle(t, 4, 4000, 5000, "delta"),
	}
	b := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 9000, 9100, "gamma"),
		makeTestSubtitle(t, 2, 9100, 9200, "alpha"),
		makeTestSubtitle(t, 3, 9200, 9300, "epsilon"),
	}

	res := Diff(a, b, ByText, 0)

	// Common preserves A order: alpha (1), gamma (3).
	if !equalIndices(res.Common, []uint32{1, 3}) {
		t.Errorf("Common indices = %v, want [1 3]", indices(res.Common))
	}
	// OnlyInA preserves A order: beta (2), delta (4).
	if !equalIndices(res.OnlyInA, []uint32{2, 4}) {
		t.Errorf("OnlyInA indices = %v, want [2 4]", indices(res.OnlyInA))
	}
	// OnlyInB preserves B order: epsilon (3).
	if !equalIndices(res.OnlyInB, []uint32{3}) {
		t.Errorf("OnlyInB indices = %v, want [3]", indices(res.OnlyInB))
	}
}
