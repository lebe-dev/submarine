package concat

import (
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// makeTestSubtitle mirrors the shared test helper used across packages.
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

// assertSubtitle checks index, start, end (in ms) and text of a subtitle.
func assertSubtitle(t *testing.T, got subtitle.Subtitle, wantIndex uint32, wantStartMs, wantEndMs int64, wantText string) {
	t.Helper()

	if got.Index.Value() != wantIndex {
		t.Errorf("index: got %d, want %d", got.Index.Value(), wantIndex)
	}
	if got.StartTime.Millis() != wantStartMs {
		t.Errorf("start: got %dms, want %dms", got.StartTime.Millis(), wantStartMs)
	}
	if got.EndTime.Millis() != wantEndMs {
		t.Errorf("end: got %dms, want %dms", got.EndTime.Millis(), wantEndMs)
	}
	if got.Text.Value() != wantText {
		t.Errorf("text: got %q, want %q", got.Text.Value(), wantText)
	}
}

func TestConcat(t *testing.T) {
	tests := []struct {
		name  string
		parts [][]subtitle.Subtitle
		gapMs int64
		want  []struct {
			index          uint32
			startMs, endMs int64
			text           string
		}
	}{
		{
			name:  "empty input",
			parts: nil,
			gapMs: 1000,
			want:  nil,
		},
		{
			name: "single part unchanged except renumbering",
			parts: [][]subtitle.Subtitle{
				{
					makeTestSubtitle(t, 5, 1000, 2000, "First"),
					makeTestSubtitle(t, 6, 2000, 3000, "Second"),
				},
			},
			gapMs: 500,
			want: []struct {
				index          uint32
				startMs, endMs int64
				text           string
			}{
				{1, 1000, 2000, "First"},
				{2, 2000, 3000, "Second"},
			},
		},
		{
			name: "two parts: part2 offset by part1 length plus gap",
			parts: [][]subtitle.Subtitle{
				{
					makeTestSubtitle(t, 1, 1000, 2000, "P1 #1"),
					makeTestSubtitle(t, 2, 2000, 4000, "P1 #2"),
				},
				{
					makeTestSubtitle(t, 1, 0, 1000, "P2 #1"),
					makeTestSubtitle(t, 2, 1000, 2500, "P2 #2"),
				},
			},
			// part1 maxEnd = 4000ms, gap = 1000ms -> cumulative = 5000ms.
			gapMs: 1000,
			want: []struct {
				index          uint32
				startMs, endMs int64
				text           string
			}{
				{1, 1000, 2000, "P1 #1"},
				{2, 2000, 4000, "P1 #2"},
				{3, 5000, 6000, "P2 #1"},
				{4, 6000, 7500, "P2 #2"},
			},
		},
		{
			name: "zero gap joins parts back to back",
			parts: [][]subtitle.Subtitle{
				{
					makeTestSubtitle(t, 1, 0, 3000, "A"),
				},
				{
					makeTestSubtitle(t, 1, 0, 1000, "B"),
				},
			},
			// part1 maxEnd = 3000ms, gap = 0 -> cumulative = 3000ms.
			gapMs: 0,
			want: []struct {
				index          uint32
				startMs, endMs int64
				text           string
			}{
				{1, 0, 3000, "A"},
				{2, 3000, 4000, "B"},
			},
		},
		{
			name: "empty part contributes nothing and does not advance cumulative",
			parts: [][]subtitle.Subtitle{
				{
					makeTestSubtitle(t, 1, 0, 2000, "A"),
				},
				{}, // empty: skipped, no advance
				{
					makeTestSubtitle(t, 1, 0, 1000, "B"),
				},
			},
			// part1 maxEnd = 2000ms, gap = 500ms -> cumulative = 2500ms.
			// empty part does NOT add another gap.
			gapMs: 500,
			want: []struct {
				index          uint32
				startMs, endMs int64
				text           string
			}{
				{1, 0, 2000, "A"},
				{2, 2500, 3500, "B"},
			},
		},
		{
			name: "leading empty parts do not advance cumulative",
			parts: [][]subtitle.Subtitle{
				{},
				{},
				{
					makeTestSubtitle(t, 9, 1000, 2000, "Only"),
				},
			},
			gapMs: 1000,
			want: []struct {
				index          uint32
				startMs, endMs int64
				text           string
			}{
				{1, 1000, 2000, "Only"},
			},
		},
		{
			name: "max end uses largest end time within a part",
			parts: [][]subtitle.Subtitle{
				{
					makeTestSubtitle(t, 1, 0, 5000, "Long"),
					makeTestSubtitle(t, 2, 1000, 2000, "Short"),
				},
				{
					makeTestSubtitle(t, 1, 0, 1000, "Next"),
				},
			},
			// part1 maxEnd = 5000ms (from sub 1, not the last sub), gap = 0.
			gapMs: 0,
			want: []struct {
				index          uint32
				startMs, endMs int64
				text           string
			}{
				{1, 0, 5000, "Long"},
				{2, 1000, 2000, "Short"},
				{3, 5000, 6000, "Next"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Concat(tt.parts, tt.gapMs)
			if err != nil {
				t.Fatalf("Concat: unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("len: got %d, want %d", len(got), len(tt.want))
			}

			for i, w := range tt.want {
				assertSubtitle(t, got[i], w.index, w.startMs, w.endMs, w.text)
			}
		})
	}
}

func TestConcatRenumbersSequentially(t *testing.T) {
	parts := [][]subtitle.Subtitle{
		{
			makeTestSubtitle(t, 100, 0, 1000, "A"),
			makeTestSubtitle(t, 200, 1000, 2000, "B"),
		},
		{
			makeTestSubtitle(t, 7, 0, 1000, "C"),
			makeTestSubtitle(t, 8, 1000, 2000, "D"),
		},
	}

	got, err := Concat(parts, 0)
	if err != nil {
		t.Fatalf("Concat: unexpected error: %v", err)
	}

	for i, sub := range got {
		want := uint32(i + 1)
		if sub.Index.Value() != want {
			t.Errorf("index at %d: got %d, want %d", i, sub.Index.Value(), want)
		}
	}
}

func TestConcatSinglePartGap(t *testing.T) {
	// A single part with a gap is unchanged except renumbering: the gap only
	// affects spacing between parts, so it has no effect with one part.
	part := []subtitle.Subtitle{
		makeTestSubtitle(t, 42, 3000, 5000, "Hello"),
		makeTestSubtitle(t, 43, 5000, 7000, "World"),
	}

	got, err := Concat([][]subtitle.Subtitle{part}, 9999)
	if err != nil {
		t.Fatalf("Concat: unexpected error: %v", err)
	}

	assertSubtitle(t, got[0], 1, 3000, 5000, "Hello")
	assertSubtitle(t, got[1], 2, 5000, 7000, "World")
}

func TestConcatAllEmptyParts(t *testing.T) {
	got, err := Concat([][]subtitle.Subtitle{{}, {}, {}}, 1000)
	if err != nil {
		t.Fatalf("Concat: unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("len: got %d, want 0", len(got))
	}
}
