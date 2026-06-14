package gaps

import (
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// makeTestSubtitle mirrors the verify test helper make_test_subtitle.
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

func ms(v int64) time.Duration { return time.Duration(v) * time.Millisecond }

func TestFindGaps(t *testing.T) {
	tests := []struct {
		name     string
		subs     []subtitle.Subtitle
		minGapMs int64
		want     []Gap
	}{
		{
			name:     "empty input yields no gaps",
			subs:     nil,
			minGapMs: 0,
			want:     []Gap{},
		},
		{
			name: "single subtitle yields no gaps",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 2000, "Only"),
			},
			minGapMs: 0,
			want:     []Gap{},
		},
		{
			name: "clear gap between two subs is reported",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 2000, "First"),
				makeTestSubtitle(t, 2, 5000, 6000, "Second"),
			},
			minGapMs: 100,
			want: []Gap{
				{AfterIndex: 1, Start: ms(2000), End: ms(5000), Duration: ms(3000)},
			},
		},
		{
			name: "gaps below the minimum are filtered",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 2000, "First"),
				makeTestSubtitle(t, 2, 2050, 3000, "Second (50ms gap)"),
				makeTestSubtitle(t, 3, 6000, 7000, "Third (3000ms gap)"),
			},
			minGapMs: 100,
			want: []Gap{
				{AfterIndex: 2, Start: ms(3000), End: ms(6000), Duration: ms(3000)},
			},
		},
		{
			name: "gap exactly at the minimum is reported",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 2000, "First"),
				makeTestSubtitle(t, 2, 2500, 3000, "Second (500ms gap)"),
			},
			minGapMs: 500,
			want: []Gap{
				{AfterIndex: 1, Start: ms(2000), End: ms(2500), Duration: ms(500)},
			},
		},
		{
			name: "contiguous subs produce nothing",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 2000, "First"),
				makeTestSubtitle(t, 2, 2000, 3000, "Second"),
				makeTestSubtitle(t, 3, 3000, 4000, "Third"),
			},
			minGapMs: 0,
			want:     []Gap{},
		},
		{
			name: "overlapping subs produce no spurious gaps",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 3000, "First"),
				makeTestSubtitle(t, 2, 2000, 4000, "Second overlaps"),
				makeTestSubtitle(t, 3, 3500, 5000, "Third overlaps"),
			},
			minGapMs: 0,
			want:     []Gap{},
		},
		{
			name: "nested sub does not advance running end and gap uses outer index",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 8000, "Outer"),
				makeTestSubtitle(t, 2, 2000, 3000, "Nested inside outer"),
				makeTestSubtitle(t, 3, 10000, 11000, "After outer"),
			},
			minGapMs: 100,
			want: []Gap{
				// Gap is measured from the outer sub's end (8000), not the
				// nested sub's end (3000), and is attributed to index 1.
				{AfterIndex: 1, Start: ms(8000), End: ms(10000), Duration: ms(2000)},
			},
		},
		{
			name: "unsorted input is sorted by start time before scanning",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 3, 9000, 10000, "Third"),
				makeTestSubtitle(t, 1, 1000, 2000, "First"),
				makeTestSubtitle(t, 2, 5000, 6000, "Second"),
			},
			minGapMs: 100,
			want: []Gap{
				{AfterIndex: 1, Start: ms(2000), End: ms(5000), Duration: ms(3000)},
				{AfterIndex: 2, Start: ms(6000), End: ms(9000), Duration: ms(3000)},
			},
		},
		{
			name: "multiple qualifying gaps are all reported",
			subs: []subtitle.Subtitle{
				makeTestSubtitle(t, 1, 1000, 2000, "First"),
				makeTestSubtitle(t, 2, 4000, 5000, "Second"),
				makeTestSubtitle(t, 3, 8000, 9000, "Third"),
			},
			minGapMs: 1000,
			want: []Gap{
				{AfterIndex: 1, Start: ms(2000), End: ms(4000), Duration: ms(2000)},
				{AfterIndex: 2, Start: ms(5000), End: ms(8000), Duration: ms(3000)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindGaps(tt.subs, tt.minGapMs)

			if len(got) != len(tt.want) {
				t.Fatalf("FindGaps len = %d, want %d (got %+v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("gap[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
