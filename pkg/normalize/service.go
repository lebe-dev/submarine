// Package normalize contains the subtitle normalization logic: stable sorting
// by start time, contiguous renumbering, and overlap trimming between adjacent
// subtitles.
package normalize

import (
	"log/slog"
	"sort"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// NormalizeResult is the outcome of a normalization pass. Sorted and Renumbered
// echo the requested operations; OverlapsFixed counts the adjacent pairs whose
// end time was trimmed to remove an overlap.
type NormalizeResult struct {
	Subtitles     []subtitle.Subtitle
	Sorted        bool
	Renumbered    bool
	OverlapsFixed int
}

// Normalize applies the requested normalization operations to a copy of subs
// and returns the result. The input slice is never mutated.
//
// When doSort is set, the subtitles are stable-sorted by start time ascending.
// When fixOverlaps is set, each adjacent pair (in current order) whose current
// end time exceeds the next start time has its end time trimmed down to the
// next start time, but only when that keeps end > start; otherwise the pair is
// left unchanged and not counted. When doRenumber is set, indices are reassigned
// sequentially starting from 1.
//
// An error is returned only on an unexpected constructor failure.
func Normalize(subs []subtitle.Subtitle, doSort, doRenumber, fixOverlaps bool) (NormalizeResult, error) {
	slog.Debug("normalizing subtitles",
		"count", len(subs), "sort", doSort, "renumber", doRenumber, "fix_overlaps", fixOverlaps)

	// Work on a copy so the caller's slice is never mutated.
	working := make([]subtitle.Subtitle, len(subs))
	copy(working, subs)

	if doSort {
		slog.Debug("stable-sorting subtitles by start time")
		sort.SliceStable(working, func(i, j int) bool {
			return working[i].StartTime.Value() < working[j].StartTime.Value()
		})
	}

	overlapsFixed := 0

	if fixOverlaps {
		slog.Debug("fixing overlaps between adjacent subtitles")
		for i := 0; i+1 < len(working); i++ {
			cur := working[i]
			next := working[i+1]

			// no overlap: current already ends at or before the next start
			if cur.EndTime.Value() <= next.StartTime.Value() {
				continue
			}

			// trimming would invert or zero the duration: leave unchanged
			if next.StartTime.Value() <= cur.StartTime.Value() {
				continue
			}

			trimmed, err := subtitle.NewSubtitle(cur.Index, cur.StartTime, next.StartTime, cur.Text)
			if err != nil {
				return NormalizeResult{}, err
			}
			working[i] = trimmed
			overlapsFixed++
		}
		slog.Debug("overlaps fixed", "count", overlapsFixed)
	}

	if doRenumber {
		slog.Debug("renumbering subtitles from 1")
		for i := range working {
			index, err := subtitle.NewSubtitleIndex(uint32(i + 1))
			if err != nil {
				return NormalizeResult{}, err
			}
			s := working[i]
			renumbered, err := subtitle.NewSubtitle(index, s.StartTime, s.EndTime, s.Text)
			if err != nil {
				return NormalizeResult{}, err
			}
			working[i] = renumbered
		}
	}

	return NormalizeResult{
		Subtitles:     working,
		Sorted:        doSort,
		Renumbered:    doRenumber,
		OverlapsFixed: overlapsFixed,
	}, nil
}
