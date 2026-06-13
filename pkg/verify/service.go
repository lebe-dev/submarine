package verify

import (
	"log/slog"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// CompareSubtitles compares two sets of subtitles and generates a verification
// report. Was the free function `compare_subtitles`.
//
// The reference subtitles are considered authoritative (correct). The target
// subtitles are verified against the reference.
func CompareSubtitles(
	refSubs []subtitle.Subtitle,
	refFile string,
	targetSubs []subtitle.Subtitle,
	targetFile string,
) VerificationReport {
	totalRefCount := len(refSubs)
	totalTargetCount := len(targetSubs)

	slog.Debug("comparing reference subtitles with target subtitles",
		"ref_count", totalRefCount, "target_count", totalTargetCount)

	// _ref_map in Rust is unused (prefixed with underscore); omitted here.

	targetMap := make(map[uint32]subtitle.Subtitle, len(targetSubs))
	for _, s := range targetSubs {
		targetMap[s.Index.Value()] = s
	}

	perfectMatches := 0
	var timestampMismatches []ComparisonEntry
	var missingInTarget []ComparisonEntry

	slog.Debug("phase 1: exact index matching")

	for i := range refSubs {
		refSub := refSubs[i]
		refIdx := refSub.Index.Value()

		if targetSub, ok := targetMap[refIdx]; ok {
			if timestampsMatch(&refSub, &targetSub) {
				perfectMatches++
			} else {
				timestampMismatches = append(timestampMismatches, ComparisonEntry{
					RefSubtitle: refSub,
					Status: ComparisonStatus{
						Kind:        TimestampMismatch,
						RefStart:    subtitle.FormatTimestamp(refSub.StartTime.Value()),
						RefEnd:      subtitle.FormatTimestamp(refSub.EndTime.Value()),
						TargetStart: subtitle.FormatTimestamp(targetSub.StartTime.Value()),
						TargetEnd:   subtitle.FormatTimestamp(targetSub.EndTime.Value()),
					},
				})
			}
		} else {
			missingInTarget = append(missingInTarget, ComparisonEntry{
				RefSubtitle: refSub,
				Status:      ComparisonStatus{Kind: MissingInTarget},
			})
		}
	}

	slog.Debug("phase 1 results",
		"perfect_matches", perfectMatches,
		"timestamp_mismatches", len(timestampMismatches),
		"missing", len(missingInTarget))

	var detectedOffset *int32
	if len(missingInTarget) != 0 {
		slog.Debug("phase 2: offset detection (range -10 to +10)")
		detectedOffset = detectIndexOffset(missingInTarget, targetMap)
	} else {
		slog.Debug("phase 2: skipping offset detection (no missing subtitles)")
		detectedOffset = nil
	}

	if detectedOffset != nil {
		slog.Debug("detected offset", "offset", *detectedOffset)
	} else {
		slog.Debug("no offset detected")
	}

	var matchedWithOffset []ComparisonEntry
	var remainingMissing []ComparisonEntry

	if detectedOffset != nil {
		offset := *detectedOffset
		slog.Debug("phase 3: reclassifying with offset", "offset", offset)

		for _, entry := range missingInTarget {
			refIdx := entry.RefSubtitle.Index.Value()
			targetIdx := uint32(int32(refIdx) + offset)

			if targetSub, ok := targetMap[targetIdx]; ok {
				if timestampsMatch(&entry.RefSubtitle, &targetSub) {
					matchedWithOffset = append(matchedWithOffset, ComparisonEntry{
						RefSubtitle: entry.RefSubtitle,
						Status: ComparisonStatus{
							Kind:        MatchedWithOffset,
							Offset:      offset,
							TargetIndex: targetIdx,
						},
					})
				} else {
					remainingMissing = append(remainingMissing, entry)
				}
			} else {
				remainingMissing = append(remainingMissing, entry)
			}
		}

		slog.Debug("phase 3 results",
			"matched_with_offset", len(matchedWithOffset),
			"still_missing", len(remainingMissing))
	} else {
		remainingMissing = missingInTarget
	}

	slog.Debug("phase 4: finding extra subtitles in target")

	matchedTargetIndices := make(map[uint32]bool)

	for i := range refSubs {
		refIdx := refSubs[i].Index.Value()
		if _, ok := targetMap[refIdx]; ok {
			matchedTargetIndices[refIdx] = true
		}
	}

	for _, entry := range matchedWithOffset {
		if entry.Status.Kind == MatchedWithOffset {
			matchedTargetIndices[entry.Status.TargetIndex] = true
		}
	}

	var extraInTarget []subtitle.Subtitle
	for _, s := range targetSubs {
		idx := s.Index.Value()
		if !matchedTargetIndices[idx] {
			extraInTarget = append(extraInTarget, s)
		}
	}

	slog.Debug("phase 4 results", "extra_in_target", len(extraInTarget))

	return VerificationReport{
		RefFile:             refFile,
		TargetFile:          targetFile,
		TotalRefCount:       totalRefCount,
		TotalTargetCount:    totalTargetCount,
		PerfectMatches:      perfectMatches,
		TimestampMismatches: timestampMismatches,
		MissingInTarget:     remainingMissing,
		MatchedWithOffset:   matchedWithOffset,
		DetectedOffset:      detectedOffset,
		ExtraInTarget:       extraInTarget,
	}
}

// timestampsMatch reports whether two subtitles have matching timestamps. Was
// the private fn `timestamps_match`.
func timestampsMatch(s1, s2 *subtitle.Subtitle) bool {
	return s1.StartTime == s2.StartTime && s1.EndTime == s2.EndTime
}

// detectIndexOffset detects the most common index offset between reference and
// target. Was the private fn `detect_index_offset`.
//
// Tests offsets from -10 to +10 and returns the most frequent offset that
// results in timestamp matches.
func detectIndexOffset(
	missing []ComparisonEntry,
	targetMap map[uint32]subtitle.Subtitle,
) *int32 {
	offsetCounts := make(map[int32]int)

	for _, entry := range missing {
		refIdx := entry.RefSubtitle.Index.Value()

		for offset := int32(-10); offset <= 10; offset++ {
			if offset == 0 {
				continue
			}

			targetIdx := int32(refIdx) + offset
			if targetIdx < 1 {
				continue
			}

			if targetSub, ok := targetMap[uint32(targetIdx)]; ok && timestampsMatch(&entry.RefSubtitle, &targetSub) {
				offsetCounts[offset]++
			}
		}
	}

	// Equivalent of Rust's
	//   offset_counts.into_iter().max_by_key(|(offset, count)| (*count, -offset.abs()))
	// Key ordering: primarily by count (higher wins), then by -offset.abs()
	// (smaller absolute offset wins). HashMap iteration order is unspecified in
	// both Rust and Go, so true ties resolve non-deterministically in both.
	found := false
	var bestOffset int32
	var bestCount int
	var bestNegAbs int32
	for offset, count := range offsetCounts {
		negAbs := -absI32(offset)
		if !found || count > bestCount || (count == bestCount && negAbs > bestNegAbs) {
			found = true
			bestOffset = offset
			bestCount = count
			bestNegAbs = negAbs
		}
	}

	if !found {
		return nil
	}
	return &bestOffset
}

// absI32 returns the absolute value of an int32 (Rust i32::abs).
func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
