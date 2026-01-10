use crate::subtitle::model::Subtitle;
use crate::verify::model::{ComparisonEntry, ComparisonStatus, VerificationReport};
use log::debug;
use std::collections::HashMap;

/// Compare two sets of subtitles and generate a verification report
///
/// The reference subtitles are considered authoritative (correct).
/// The target subtitles are verified against the reference.
pub fn compare_subtitles(
    ref_subs: Vec<Subtitle>,
    ref_file: String,
    target_subs: Vec<Subtitle>,
    target_file: String,
) -> VerificationReport {
    let total_ref_count = ref_subs.len();
    let total_target_count = target_subs.len();

    debug!(
        "comparing {} reference subtitles with {} target subtitles",
        total_ref_count, total_target_count
    );

    let _ref_map: HashMap<u32, Subtitle> = ref_subs
        .iter()
        .map(|s| (*s.index.as_ref(), s.clone()))
        .collect();

    let target_map: HashMap<u32, Subtitle> = target_subs
        .iter()
        .map(|s| (*s.index.as_ref(), s.clone()))
        .collect();

    let mut perfect_matches = 0;
    let mut timestamp_mismatches = Vec::new();
    let mut missing_in_target = Vec::new();

    debug!("phase 1: exact index matching");

    for ref_sub in &ref_subs {
        let ref_idx = *ref_sub.index.as_ref();

        if let Some(target_sub) = target_map.get(&ref_idx) {
            if timestamps_match(ref_sub, target_sub) {
                perfect_matches += 1;
            } else {
                timestamp_mismatches.push(ComparisonEntry {
                    ref_subtitle: ref_sub.clone(),
                    status: ComparisonStatus::TimestampMismatch {
                        ref_start: Subtitle::format_timestamp(ref_sub.start_time.as_ref()),
                        ref_end: Subtitle::format_timestamp(ref_sub.end_time.as_ref()),
                        target_start: Subtitle::format_timestamp(target_sub.start_time.as_ref()),
                        target_end: Subtitle::format_timestamp(target_sub.end_time.as_ref()),
                    },
                });
            }
        } else {
            missing_in_target.push(ComparisonEntry {
                ref_subtitle: ref_sub.clone(),
                status: ComparisonStatus::MissingInTarget,
            });
        }
    }

    debug!(
        "phase 1 results: {} perfect matches, {} timestamp mismatches, {} missing",
        perfect_matches,
        timestamp_mismatches.len(),
        missing_in_target.len()
    );

    let detected_offset = if !missing_in_target.is_empty() {
        debug!("phase 2: offset detection (range -10 to +10)");
        detect_index_offset(&missing_in_target, &target_map)
    } else {
        debug!("phase 2: skipping offset detection (no missing subtitles)");
        None
    };

    if let Some(offset) = detected_offset {
        debug!("detected offset: {}", offset);
    } else {
        debug!("no offset detected");
    }

    let mut matched_with_offset = Vec::new();
    let mut remaining_missing = Vec::new();

    if let Some(offset) = detected_offset {
        debug!("phase 3: reclassifying with offset {}", offset);

        for entry in missing_in_target {
            let ref_idx = *entry.ref_subtitle.index.as_ref();
            let target_idx = (ref_idx as i32 + offset) as u32;

            if let Some(target_sub) = target_map.get(&target_idx) {
                if timestamps_match(&entry.ref_subtitle, target_sub) {
                    matched_with_offset.push(ComparisonEntry {
                        ref_subtitle: entry.ref_subtitle.clone(),
                        status: ComparisonStatus::MatchedWithOffset {
                            offset,
                            target_index: target_idx,
                        },
                    });
                } else {
                    remaining_missing.push(entry);
                }
            } else {
                remaining_missing.push(entry);
            }
        }

        debug!(
            "phase 3 results: {} matched with offset, {} still missing",
            matched_with_offset.len(),
            remaining_missing.len()
        );
    } else {
        remaining_missing = missing_in_target;
    }

    debug!("phase 4: finding extra subtitles in target");

    let mut matched_target_indices: HashMap<u32, bool> = HashMap::new();

    for ref_sub in &ref_subs {
        let ref_idx = *ref_sub.index.as_ref();
        if target_map.contains_key(&ref_idx) {
            matched_target_indices.insert(ref_idx, true);
        }
    }

    for entry in &matched_with_offset {
        if let ComparisonStatus::MatchedWithOffset { target_index, .. } = entry.status {
            matched_target_indices.insert(target_index, true);
        }
    }

    let extra_in_target: Vec<Subtitle> = target_subs
        .iter()
        .filter(|s| {
            let idx = *s.index.as_ref();
            !matched_target_indices.contains_key(&idx)
        })
        .cloned()
        .collect();

    debug!("phase 4 results: {} extra in target", extra_in_target.len());

    VerificationReport {
        ref_file,
        target_file,
        total_ref_count,
        total_target_count,
        perfect_matches,
        timestamp_mismatches,
        missing_in_target: remaining_missing,
        matched_with_offset,
        detected_offset,
        extra_in_target,
    }
}

/// Check if two subtitles have matching timestamps
fn timestamps_match(s1: &Subtitle, s2: &Subtitle) -> bool {
    s1.start_time == s2.start_time && s1.end_time == s2.end_time
}

/// Detect the most common index offset between reference and target
///
/// Tests offsets from -10 to +10 and returns the most frequent offset
/// that results in timestamp matches.
fn detect_index_offset(
    missing: &[ComparisonEntry],
    target_map: &HashMap<u32, Subtitle>,
) -> Option<i32> {
    let mut offset_counts: HashMap<i32, usize> = HashMap::new();

    for entry in missing {
        let ref_idx = *entry.ref_subtitle.index.as_ref();

        for offset in -10..=10 {
            if offset == 0 {
                continue;
            }

            let target_idx = ref_idx as i32 + offset;
            if target_idx < 1 {
                continue;
            }

            if let Some(target_sub) = target_map.get(&(target_idx as u32))
                && timestamps_match(&entry.ref_subtitle, target_sub)
            {
                *offset_counts.entry(offset).or_insert(0) += 1;
            }
        }
    }

    offset_counts
        .into_iter()
        .max_by_key(|(offset, count)| (*count, -offset.abs()))
        .map(|(offset, _)| offset)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::subtitle::model::{SubtitleIndex, SubtitleText, SubtitleTimestamp};
    use chrono::Duration;

    fn make_test_subtitle(index: u32, start_ms: i64, end_ms: i64, text: &str) -> Subtitle {
        Subtitle::new(
            SubtitleIndex::try_new(index).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(start_ms)).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(end_ms)).unwrap(),
            SubtitleText::try_new(text.to_string()).unwrap(),
        )
        .unwrap()
    }

    #[test]
    fn test_timestamps_match() {
        let s1 = make_test_subtitle(1, 1000, 2000, "Text 1");
        let s2 = make_test_subtitle(2, 1000, 2000, "Text 2");
        let s3 = make_test_subtitle(3, 1000, 2001, "Text 3");

        assert!(timestamps_match(&s1, &s2)); // Same timestamps
        assert!(!timestamps_match(&s1, &s3)); // Different end time
    }

    #[test]
    fn test_compare_perfect_match() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "English 1"),
            make_test_subtitle(2, 2000, 3000, "English 2"),
            make_test_subtitle(3, 3000, 4000, "English 3"),
        ];

        let target_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Russian 1"),
            make_test_subtitle(2, 2000, 3000, "Russian 2"),
            make_test_subtitle(3, 3000, 4000, "Russian 3"),
        ];

        let report = compare_subtitles(
            ref_subs,
            "ref.srt".to_string(),
            target_subs,
            "target.srt".to_string(),
        );

        assert_eq!(report.perfect_matches, 3);
        assert!(report.timestamp_mismatches.is_empty());
        assert!(report.missing_in_target.is_empty());
        assert!(report.matched_with_offset.is_empty());
        assert!(report.extra_in_target.is_empty());
        assert_eq!(report.detected_offset, None);
        assert!(report.is_perfect());
        assert_eq!(report.match_percentage(), 100.0);
    }

    #[test]
    fn test_compare_missing_subtitles() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
            make_test_subtitle(3, 3000, 4000, "Text 3"),
        ];

        let target_subs = vec![make_test_subtitle(1, 1000, 2000, "Text 1")];

        let report = compare_subtitles(
            ref_subs,
            "ref.srt".to_string(),
            target_subs,
            "target.srt".to_string(),
        );

        assert_eq!(report.perfect_matches, 1);
        assert_eq!(report.missing_in_target.len(), 2);
        assert!((report.match_percentage() - 33.333333333333336).abs() < 0.0001);
    }

    #[test]
    fn test_compare_timestamp_mismatch() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
        ];

        let target_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2100, 3100, "Text 2"), // Different timestamps
        ];

        let report = compare_subtitles(
            ref_subs,
            "ref.srt".to_string(),
            target_subs,
            "target.srt".to_string(),
        );

        assert_eq!(report.perfect_matches, 1);
        assert_eq!(report.timestamp_mismatches.len(), 1);
        assert_eq!(report.match_percentage(), 50.0);
    }

    #[test]
    fn test_compare_with_offset() {
        let ref_subs = vec![
            make_test_subtitle(5, 1000, 2000, "English 5"),
            make_test_subtitle(6, 2000, 3000, "English 6"),
            make_test_subtitle(7, 3000, 4000, "English 7"),
        ];

        // Target has indices offset by -2
        let target_subs = vec![
            make_test_subtitle(3, 1000, 2000, "Russian 3"),
            make_test_subtitle(4, 2000, 3000, "Russian 4"),
            make_test_subtitle(5, 3000, 4000, "Russian 5"),
        ];

        let report = compare_subtitles(
            ref_subs,
            "ref.srt".to_string(),
            target_subs,
            "target.srt".to_string(),
        );

        assert_eq!(report.perfect_matches, 0);
        assert_eq!(report.timestamp_mismatches.len(), 1); // Index 5 has different timestamps
        assert_eq!(report.matched_with_offset.len(), 2); // Indices 6 and 7 with offset -2
        assert_eq!(report.detected_offset, Some(-2));
        // Only 2 out of 3 matched (the third has timestamp mismatch)
        assert!((report.match_percentage() - 66.66666666666667).abs() < 0.0001);
        assert!(!report.is_perfect()); // Not perfect
        assert!(report.has_issues()); // Has timestamp mismatch issue
    }

    #[test]
    fn test_compare_extra_in_target() {
        let ref_subs = vec![make_test_subtitle(1, 1000, 2000, "Text 1")];

        let target_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
            make_test_subtitle(3, 3000, 4000, "Text 3"),
        ];

        let report = compare_subtitles(
            ref_subs,
            "ref.srt".to_string(),
            target_subs,
            "target.srt".to_string(),
        );

        assert_eq!(report.perfect_matches, 1);
        assert_eq!(report.extra_in_target.len(), 2);
        assert_eq!(report.match_percentage(), 100.0);
    }

    #[test]
    fn test_detect_offset_simple() {
        let ref_subs = vec![
            make_test_subtitle(10, 1000, 2000, "Text"),
            make_test_subtitle(20, 2000, 3000, "Text"),
        ];

        let target_map: HashMap<u32, Subtitle> = vec![
            make_test_subtitle(8, 1000, 2000, "Text"),
            make_test_subtitle(18, 2000, 3000, "Text"),
        ]
        .into_iter()
        .map(|s| (*s.index.as_ref(), s))
        .collect();

        let missing: Vec<ComparisonEntry> = ref_subs
            .into_iter()
            .map(|s| ComparisonEntry {
                ref_subtitle: s,
                status: ComparisonStatus::MissingInTarget,
            })
            .collect();

        let offset = detect_index_offset(&missing, &target_map);
        assert_eq!(offset, Some(-2));
    }

    #[test]
    fn test_detect_offset_no_match() {
        let ref_subs = vec![make_test_subtitle(10, 1000, 2000, "Text")];

        let target_map: HashMap<u32, Subtitle> = vec![make_test_subtitle(8, 9000, 9999, "Text")]
            .into_iter()
            .map(|s| (*s.index.as_ref(), s))
            .collect();

        let missing: Vec<ComparisonEntry> = ref_subs
            .into_iter()
            .map(|s| ComparisonEntry {
                ref_subtitle: s,
                status: ComparisonStatus::MissingInTarget,
            })
            .collect();

        let offset = detect_index_offset(&missing, &target_map);
        assert_eq!(offset, None);
    }

    #[test]
    fn test_compare_empty_files() {
        let ref_subs: Vec<Subtitle> = vec![];
        let target_subs: Vec<Subtitle> = vec![];

        let report = compare_subtitles(
            ref_subs,
            "ref.srt".to_string(),
            target_subs,
            "target.srt".to_string(),
        );

        assert_eq!(report.total_ref_count, 0);
        assert_eq!(report.total_target_count, 0);
        assert_eq!(report.match_percentage(), 0.0);
    }
}
