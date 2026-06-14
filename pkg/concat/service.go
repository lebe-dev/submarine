// Package concat contains the logic for sequentially joining multiple parts of
// subtitles into a single timeline.
package concat

import (
	"log/slog"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// Concat sequentially joins parts of subtitles into a single timeline.
//
// A running cumulative offset starts at zero. For each part in order, every
// subtitle is shifted by the current cumulative offset (start += cumulative,
// end += cumulative) and appended to the result. After a part, the cumulative
// offset advances to that part's maximum shifted EndTime plus gapMs
// milliseconds, so the next part begins after the gap. An empty part
// contributes nothing and does not advance the cumulative offset. After all
// parts are joined, the result is renumbered sequentially from 1.
func Concat(parts [][]subtitle.Subtitle, gapMs int64) ([]subtitle.Subtitle, error) {
	slog.Debug("concatenating subtitle parts", "parts", len(parts), "gap_ms", gapMs)

	cumulative := time.Duration(0)
	gap := time.Duration(gapMs) * time.Millisecond

	var result []subtitle.Subtitle

	for partIdx, part := range parts {
		if len(part) == 0 {
			slog.Debug("skipping empty part", "part", partIdx)
			continue
		}

		var maxEnd time.Duration

		for _, sub := range part {
			newStartDur := sub.StartTime.Value() + cumulative
			newEndDur := sub.EndTime.Value() + cumulative

			newStart, err := subtitle.NewSubtitleTimestamp(newStartDur)
			if err != nil {
				return nil, err
			}
			newEnd, err := subtitle.NewSubtitleTimestamp(newEndDur)
			if err != nil {
				return nil, err
			}

			// Index is reassigned during renumbering below; reuse the original
			// here only to satisfy NewSubtitle's cross-field validation.
			shifted, err := subtitle.NewSubtitle(sub.Index, newStart, newEnd, sub.Text)
			if err != nil {
				return nil, err
			}

			result = append(result, shifted)

			if newEndDur > maxEnd {
				maxEnd = newEndDur
			}
		}

		cumulative = maxEnd + gap
	}

	return renumber(result)
}

// renumber reassigns sequential indices from 1 to the joined subtitles,
// rebuilding each one via NewSubtitle (which preserves the end > start
// invariant).
func renumber(subs []subtitle.Subtitle) ([]subtitle.Subtitle, error) {
	renumbered := make([]subtitle.Subtitle, 0, len(subs))

	for i, sub := range subs {
		idx, err := subtitle.NewSubtitleIndex(uint32(i + 1))
		if err != nil {
			return nil, err
		}

		s, err := subtitle.NewSubtitle(idx, sub.StartTime, sub.EndTime, sub.Text)
		if err != nil {
			return nil, err
		}

		renumbered = append(renumbered, s)
	}

	return renumbered, nil
}
