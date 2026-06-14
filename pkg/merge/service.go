package merge

import (
	"log/slog"
	"sort"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// shiftedDonor holds a donor subtitle after offset shifting and clamping. The
// start/end are kept as raw millisecond values so overlap math (which may use a
// negative-tolerance lower bound) stays in plain integers.
type shiftedDonor struct {
	sub     subtitle.Subtitle
	startMs int64
	endMs   int64
}

// Merge reconciles a donor subtitle track into an authoritative base track.
//
// Every donor subtitle is first shifted by offsetMs (which may be negative):
// newStart = start + offset, newEnd = end + offset. A negative newStart is
// clamped to 0; if after clamping newEnd <= newStart the donor subtitle is
// dropped. Overlap is then evaluated against the base track with a tolerance of
// overlapToleranceMs milliseconds: donor D overlaps base B iff
// (D.start - tol) < B.end AND B.start < (D.end + tol).
//
// The selected Strategy decides what happens on overlap (see the Strategy
// constants). The merged set is finally sorted by start time (stable) and
// renumbered sequentially from 1.
func Merge(
	base, donor []subtitle.Subtitle,
	strategy Strategy,
	overlapToleranceMs, offsetMs int64,
) (MergeResult, error) {
	slog.Debug("merging donor track into base track",
		"base_count", len(base),
		"donor_count", len(donor),
		"strategy", strategy.String(),
		"overlap_tolerance_ms", overlapToleranceMs,
		"offset_ms", offsetMs)

	result := MergeResult{
		BaseCount:  len(base),
		DonorCount: len(donor),
	}

	shifted := shiftDonor(donor, offsetMs)
	slog.Debug("donor track shifted", "kept", len(shifted), "dropped", len(donor)-len(shifted))

	var assembled []subtitle.Subtitle

	switch strategy {
	case PreferDonor:
		assembled = mergePreferDonor(base, shifted, overlapToleranceMs, &result)
	default: // FillGaps and KeepBase assemble identically.
		assembled = mergeFillGaps(base, shifted, overlapToleranceMs, &result)
	}

	merged, err := sortAndRenumber(assembled)
	if err != nil {
		return MergeResult{}, err
	}

	result.Merged = merged
	result.TotalCount = len(merged)

	slog.Info("merge complete",
		"added", result.Added,
		"skipped_overlapping", result.SkippedOverlapping,
		"replaced", result.Replaced,
		"total", result.TotalCount)

	return result, nil
}

// mergeFillGaps keeps every base subtitle and adds a donor subtitle only when
// it overlaps no base subtitle. Shared by FillGaps and KeepBase.
func mergeFillGaps(
	base []subtitle.Subtitle,
	donor []shiftedDonor,
	tolMs int64,
	result *MergeResult,
) []subtitle.Subtitle {
	assembled := make([]subtitle.Subtitle, 0, len(base)+len(donor))
	assembled = append(assembled, base...)

	for _, d := range donor {
		if overlapsAnyBase(d, base, tolMs) {
			result.SkippedOverlapping++
			result.SkippedDonor = append(result.SkippedDonor, d.sub)
			continue
		}
		result.Added++
		result.AddedDonor = append(result.AddedDonor, d.sub)
		assembled = append(assembled, d.sub)
	}

	return assembled
}

// mergePreferDonor lets donor subtitles replace the base subtitles they
// overlap. Removal decisions are taken against the ORIGINAL base set, then the
// surviving base subtitles are assembled together with every donor subtitle.
func mergePreferDonor(
	base []subtitle.Subtitle,
	donor []shiftedDonor,
	tolMs int64,
	result *MergeResult,
) []subtitle.Subtitle {
	removed := make([]bool, len(base))

	for _, d := range donor {
		overlapped := false
		for i := range base {
			if overlaps(d, base[i], tolMs) {
				overlapped = true
				if !removed[i] {
					removed[i] = true
					result.Replaced++
				}
			}
		}
		if !overlapped {
			result.Added++
		}
		// Every donor subtitle is added under PreferDonor.
		result.AddedDonor = append(result.AddedDonor, d.sub)
	}

	assembled := make([]subtitle.Subtitle, 0, len(base)+len(donor))
	for i := range base {
		if !removed[i] {
			assembled = append(assembled, base[i])
		}
	}
	for _, d := range donor {
		assembled = append(assembled, d.sub)
	}

	return assembled
}

// shiftDonor applies offsetMs to every donor subtitle, clamps a negative start
// to zero, and drops any subtitle whose clamped end is not after its start.
func shiftDonor(donor []subtitle.Subtitle, offsetMs int64) []shiftedDonor {
	out := make([]shiftedDonor, 0, len(donor))

	for _, d := range donor {
		startMs := d.StartTime.Millis() + offsetMs
		endMs := d.EndTime.Millis() + offsetMs

		if startMs < 0 {
			startMs = 0
		}
		if endMs <= startMs {
			slog.Debug("dropping donor subtitle after shift (empty/negative duration)",
				"index", d.Index.Value(), "start_ms", startMs, "end_ms", endMs)
			continue
		}

		out = append(out, shiftedDonor{sub: rebuildTimes(d, startMs, endMs), startMs: startMs, endMs: endMs})
	}

	return out
}

// rebuildTimes returns a copy of the subtitle with its start/end replaced by
// the given millisecond values. The newtype constructors cannot fail here: the
// caller guarantees startMs >= 0 and endMs > startMs.
func rebuildTimes(s subtitle.Subtitle, startMs, endMs int64) subtitle.Subtitle {
	start, _ := subtitle.NewSubtitleTimestamp(time.Duration(startMs) * time.Millisecond)
	end, _ := subtitle.NewSubtitleTimestamp(time.Duration(endMs) * time.Millisecond)
	rebuilt, _ := subtitle.NewSubtitle(s.Index, start, end, s.Text)
	return rebuilt
}

// overlaps reports whether a shifted donor subtitle overlaps a base subtitle
// within tolerance tolMs: (D.start - tol) < B.end AND B.start < (D.end + tol).
func overlaps(d shiftedDonor, b subtitle.Subtitle, tolMs int64) bool {
	baseStart := b.StartTime.Millis()
	baseEnd := b.EndTime.Millis()
	return (d.startMs-tolMs) < baseEnd && baseStart < (d.endMs+tolMs)
}

// overlapsAnyBase reports whether a shifted donor subtitle overlaps any base
// subtitle within tolerance.
func overlapsAnyBase(d shiftedDonor, base []subtitle.Subtitle, tolMs int64) bool {
	for i := range base {
		if overlaps(d, base[i], tolMs) {
			return true
		}
	}
	return false
}

// sortAndRenumber sorts subtitles by start time (stable) and renumbers them
// sequentially from 1, rebuilding each via NewSubtitle.
func sortAndRenumber(subs []subtitle.Subtitle) ([]subtitle.Subtitle, error) {
	sort.SliceStable(subs, func(i, j int) bool {
		return subs[i].StartTime.Value() < subs[j].StartTime.Value()
	})

	out := make([]subtitle.Subtitle, 0, len(subs))
	for i := range subs {
		idx, err := subtitle.NewSubtitleIndex(uint32(i + 1))
		if err != nil {
			return nil, err
		}
		rebuilt, err := subtitle.NewSubtitle(idx, subs[i].StartTime, subs[i].EndTime, subs[i].Text)
		if err != nil {
			return nil, err
		}
		out = append(out, rebuilt)
	}

	return out, nil
}
