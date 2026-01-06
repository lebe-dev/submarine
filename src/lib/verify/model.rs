use crate::subtitle::model::Subtitle;

/// Result category for each subtitle comparison
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ComparisonStatus {
    /// Perfect match: same index, same timestamps
    PerfectMatch,

    /// Same index but timestamps differ
    TimestampMismatch {
        ref_start: String,
        ref_end: String,
        target_start: String,
        target_end: String,
    },

    /// Found in target file with offset applied (same timestamps)
    MatchedWithOffset { offset: i32, target_index: u32 },

    /// Not found in target file at all
    MissingInTarget,
}

/// Details about a single subtitle comparison
#[derive(Debug, Clone)]
pub struct ComparisonEntry {
    pub ref_subtitle: Subtitle,
    pub status: ComparisonStatus,
}

/// Overall verification report
#[derive(Debug, Clone)]
pub struct VerificationReport {
    pub ref_file: String,
    pub target_file: String,
    pub total_ref_count: usize,
    pub total_target_count: usize,
    pub perfect_matches: usize,
    pub timestamp_mismatches: Vec<ComparisonEntry>,
    pub missing_in_target: Vec<ComparisonEntry>,
    pub matched_with_offset: Vec<ComparisonEntry>,
    pub detected_offset: Option<i32>,
    pub extra_in_target: Vec<Subtitle>,
}

impl VerificationReport {
    /// Check if verification has any issues
    pub fn has_issues(&self) -> bool {
        !self.timestamp_mismatches.is_empty()
            || !self.missing_in_target.is_empty()
            || !self.extra_in_target.is_empty()
    }

    /// Check if verification is perfect (100% match)
    pub fn is_perfect(&self) -> bool {
        self.perfect_matches == self.total_ref_count
            && self.timestamp_mismatches.is_empty()
            && self.missing_in_target.is_empty()
            && self.matched_with_offset.is_empty()
            && self.extra_in_target.is_empty()
    }

    /// Get total number of matched subtitles (perfect + with offset)
    pub fn total_matched(&self) -> usize {
        self.perfect_matches + self.matched_with_offset.len()
    }

    /// Get match percentage
    pub fn match_percentage(&self) -> f64 {
        if self.total_ref_count == 0 {
            return 0.0;
        }
        (self.total_matched() as f64 / self.total_ref_count as f64) * 100.0
    }
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
    fn test_verification_report_is_perfect() {
        let report = VerificationReport {
            ref_file: "ref.srt".to_string(),
            target_file: "target.srt".to_string(),
            total_ref_count: 10,
            total_target_count: 10,
            perfect_matches: 10,
            timestamp_mismatches: vec![],
            missing_in_target: vec![],
            matched_with_offset: vec![],
            detected_offset: None,
            extra_in_target: vec![],
        };

        assert!(report.is_perfect());
        assert!(!report.has_issues());
        assert_eq!(report.match_percentage(), 100.0);
    }

    #[test]
    fn test_verification_report_has_issues() {
        let sub = make_test_subtitle(1, 1000, 2000, "Test");
        let entry = ComparisonEntry {
            ref_subtitle: sub.clone(),
            status: ComparisonStatus::MissingInTarget,
        };

        let report = VerificationReport {
            ref_file: "ref.srt".to_string(),
            target_file: "target.srt".to_string(),
            total_ref_count: 10,
            total_target_count: 9,
            perfect_matches: 9,
            timestamp_mismatches: vec![],
            missing_in_target: vec![entry],
            matched_with_offset: vec![],
            detected_offset: None,
            extra_in_target: vec![],
        };

        assert!(!report.is_perfect());
        assert!(report.has_issues());
        assert_eq!(report.match_percentage(), 90.0);
    }

    #[test]
    fn test_verification_report_with_offset() {
        let sub = make_test_subtitle(1, 1000, 2000, "Test");
        let entry = ComparisonEntry {
            ref_subtitle: sub.clone(),
            status: ComparisonStatus::MatchedWithOffset {
                offset: -2,
                target_index: 3,
            },
        };

        let report = VerificationReport {
            ref_file: "ref.srt".to_string(),
            target_file: "target.srt".to_string(),
            total_ref_count: 10,
            total_target_count: 10,
            perfect_matches: 9,
            timestamp_mismatches: vec![],
            missing_in_target: vec![],
            matched_with_offset: vec![entry],
            detected_offset: Some(-2),
            extra_in_target: vec![],
        };

        assert!(!report.is_perfect());
        assert!(!report.has_issues());
        assert_eq!(report.total_matched(), 10);
        assert_eq!(report.match_percentage(), 100.0);
    }

    #[test]
    fn test_match_percentage_zero_subtitles() {
        let report = VerificationReport {
            ref_file: "ref.srt".to_string(),
            target_file: "target.srt".to_string(),
            total_ref_count: 0,
            total_target_count: 0,
            perfect_matches: 0,
            timestamp_mismatches: vec![],
            missing_in_target: vec![],
            matched_with_offset: vec![],
            detected_offset: None,
            extra_in_target: vec![],
        };

        assert_eq!(report.match_percentage(), 0.0);
    }
}
