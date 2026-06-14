// Package merge implements merging a donor subtitle timeline into an
// authoritative base timeline. The base track is the source of truth; donor
// subtitles are optionally time-shifted and then either used to fill gaps,
// ignored where they overlap, or allowed to replace overlapping base lines.
package merge

import (
	"fmt"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// Strategy selects how donor subtitles are reconciled against the base track.
type Strategy int

const (
	// FillGaps keeps every base subtitle and adds donor subtitles only where
	// they overlap no base subtitle (filling holes in the base timeline).
	FillGaps Strategy = iota
	// KeepBase behaves identically to FillGaps when assembling the result; the
	// distinction is purely how the CLI handler reports the outcome.
	KeepBase
	// PreferDonor lets each donor subtitle replace the base subtitles it
	// overlaps; non-overlapping donor subtitles are added.
	PreferDonor
)

// String returns the CLI value-name for the strategy.
func (s Strategy) String() string {
	switch s {
	case FillGaps:
		return "fill-gaps"
	case KeepBase:
		return "keep-base"
	case PreferDonor:
		return "prefer-donor"
	}
	return ""
}

// ParseStrategy parses a Strategy from its CLI value-name.
func ParseStrategy(s string) (Strategy, error) {
	switch s {
	case "fill-gaps":
		return FillGaps, nil
	case "keep-base":
		return KeepBase, nil
	case "prefer-donor":
		return PreferDonor, nil
	}
	return 0, fmt.Errorf("invalid value '%s' for '--strategy <STRATEGY>'", s)
}

// MergeResult is the outcome of a Merge operation. Merged holds the final,
// sorted and renumbered subtitle track; AddedDonor and SkippedDonor carry the
// donor subtitles that were respectively added or skipped (in their shifted,
// pre-renumber form).
type MergeResult struct {
	BaseCount          int
	DonorCount         int
	Added              int
	SkippedOverlapping int
	Replaced           int
	TotalCount         int
	Merged             []subtitle.Subtitle
	AddedDonor         []subtitle.Subtitle
	SkippedDonor       []subtitle.Subtitle
}
