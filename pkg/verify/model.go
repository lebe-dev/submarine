// Package verify contains the subtitle verification (comparison) logic. It is a
// 1-to-1 port of the Rust crate `lib::verify`.
package verify

import "github.com/lebe-dev/submarine/pkg/subtitle"

// ComparisonStatusKind discriminates the variants of ComparisonStatus. It
// mirrors the variants of the Rust `enum ComparisonStatus`.
type ComparisonStatusKind int

const (
	// PerfectMatch: same index, same timestamps.
	PerfectMatch ComparisonStatusKind = iota
	// TimestampMismatch: same index but timestamps differ.
	TimestampMismatch
	// MatchedWithOffset: found in target file with offset applied (same timestamps).
	MatchedWithOffset
	// MissingInTarget: not found in target file at all.
	MissingInTarget
)

// ComparisonStatus is the result category for each subtitle comparison. It is a
// tagged union mirroring the Rust `enum ComparisonStatus`. The data-carrying
// fields are populated only for the relevant Kind; the rest are zero-valued.
type ComparisonStatus struct {
	Kind ComparisonStatusKind

	// TimestampMismatch fields
	RefStart    string
	RefEnd      string
	TargetStart string
	TargetEnd   string

	// MatchedWithOffset fields
	Offset      int32
	TargetIndex uint32
}

// ComparisonEntry holds details about a single subtitle comparison. Was Rust
// struct ComparisonEntry.
type ComparisonEntry struct {
	RefSubtitle subtitle.Subtitle
	Status      ComparisonStatus
}

// VerificationReport is the overall verification report. Was Rust struct
// VerificationReport.
type VerificationReport struct {
	RefFile             string
	TargetFile          string
	TotalRefCount       int
	TotalTargetCount    int
	PerfectMatches      int
	TimestampMismatches []ComparisonEntry
	MissingInTarget     []ComparisonEntry
	MatchedWithOffset   []ComparisonEntry
	DetectedOffset      *int32
	ExtraInTarget       []subtitle.Subtitle
}

// HasIssues reports whether the verification has any issues. Was
// VerificationReport::has_issues.
func (r *VerificationReport) HasIssues() bool {
	return len(r.TimestampMismatches) != 0 ||
		len(r.MissingInTarget) != 0 ||
		len(r.ExtraInTarget) != 0
}

// IsPerfect reports whether the verification is perfect (100% match). Was
// VerificationReport::is_perfect.
func (r *VerificationReport) IsPerfect() bool {
	return r.PerfectMatches == r.TotalRefCount &&
		len(r.TimestampMismatches) == 0 &&
		len(r.MissingInTarget) == 0 &&
		len(r.MatchedWithOffset) == 0 &&
		len(r.ExtraInTarget) == 0
}

// TotalMatched returns the total number of matched subtitles (perfect + with
// offset). Was VerificationReport::total_matched.
func (r *VerificationReport) TotalMatched() int {
	return r.PerfectMatches + len(r.MatchedWithOffset)
}

// MatchPercentage returns the match percentage. Was
// VerificationReport::match_percentage.
func (r *VerificationReport) MatchPercentage() float64 {
	if r.TotalRefCount == 0 {
		return 0.0
	}
	return (float64(r.TotalMatched()) / float64(r.TotalRefCount)) * 100.0
}
