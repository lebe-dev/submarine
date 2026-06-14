// Package gaps detects timing gaps between subtitles in a subtitle track.
package gaps

import (
	"log/slog"
	"sort"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// Gap describes a span of empty time between subtitles. AfterIndex is the index
// of the subtitle whose end time opens the gap; Start and End are the gap
// boundaries and Duration is End - Start.
type Gap struct {
	AfterIndex uint32
	Start      time.Duration
	End        time.Duration
	Duration   time.Duration
}

// FindGaps returns the gaps between consecutive subtitles whose duration is at
// least minGapMs milliseconds.
//
// A copy of subs is sorted by start time ascending and scanned once. A running
// end time and the index that produced it are tracked, so overlapping and
// nested subtitles never produce spurious gaps. Empty or single-element input
// yields no gaps.
func FindGaps(subs []subtitle.Subtitle, minGapMs int64) []Gap {
	gaps := []Gap{}

	if len(subs) < 2 {
		slog.Debug("not enough subtitles to detect gaps", "count", len(subs))
		return gaps
	}

	sorted := make([]subtitle.Subtitle, len(subs))
	copy(sorted, subs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Value() < sorted[j].StartTime.Value()
	})

	slog.Debug("scanning for gaps", "count", len(sorted), "min_gap_ms", minGapMs)

	runningEnd := sorted[0].EndTime.Value()
	runningIndex := sorted[0].Index.Value()

	for i := 1; i < len(sorted); i++ {
		start := sorted[i].StartTime.Value()
		gap := start - runningEnd

		if gap > 0 && gap.Milliseconds() >= minGapMs {
			gaps = append(gaps, Gap{
				AfterIndex: runningIndex,
				Start:      runningEnd,
				End:        start,
				Duration:   gap,
			})
		}

		if end := sorted[i].EndTime.Value(); end > runningEnd {
			runningEnd = end
			runningIndex = sorted[i].Index.Value()
		}
	}

	slog.Debug("gap detection complete", "gaps_found", len(gaps))

	return gaps
}
