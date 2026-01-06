/// Report structure for translation status check
#[derive(Debug, Clone)]
pub struct TranslationStatusReport {
    /// Reference file name
    pub ref_file: String,
    /// Translation file name
    pub translation_file: String,
    /// Total number of subtitles in reference file
    pub total_count: usize,
    /// Number of subtitles translated
    pub translated_count: usize,
    /// Number of subtitles remaining
    pub missing_count: usize,
    /// Suggestion for next chunk to translate
    pub next_chunk: Option<ChunkSuggestion>,
}

/// Suggestion for the next chunk of subtitles to translate
#[derive(Debug, Clone)]
pub struct ChunkSuggestion {
    /// Starting index of the chunk
    pub start_index: u32,
    /// Ending index of the chunk
    pub end_index: u32,
}

impl TranslationStatusReport {
    /// Calculate the translation progress as a percentage
    pub fn progress_percentage(&self) -> f64 {
        if self.total_count == 0 {
            0.0
        } else {
            (self.translated_count as f64 / self.total_count as f64) * 100.0
        }
    }

    /// Check if translation is complete
    pub fn is_complete(&self) -> bool {
        self.missing_count == 0 && self.total_count > 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_progress_percentage() {
        let report = TranslationStatusReport {
            ref_file: "ref.srt".to_string(),
            translation_file: "trans.srt".to_string(),
            total_count: 100,
            translated_count: 50,
            missing_count: 50,
            next_chunk: None,
        };

        assert_eq!(report.progress_percentage(), 50.0);
    }

    #[test]
    fn test_progress_percentage_zero_total() {
        let report = TranslationStatusReport {
            ref_file: "ref.srt".to_string(),
            translation_file: "trans.srt".to_string(),
            total_count: 0,
            translated_count: 0,
            missing_count: 0,
            next_chunk: None,
        };

        assert_eq!(report.progress_percentage(), 0.0);
    }

    #[test]
    fn test_is_complete() {
        let complete = TranslationStatusReport {
            ref_file: "ref.srt".to_string(),
            translation_file: "trans.srt".to_string(),
            total_count: 100,
            translated_count: 100,
            missing_count: 0,
            next_chunk: None,
        };

        assert!(complete.is_complete());

        let incomplete = TranslationStatusReport {
            ref_file: "ref.srt".to_string(),
            translation_file: "trans.srt".to_string(),
            total_count: 100,
            translated_count: 50,
            missing_count: 50,
            next_chunk: Some(ChunkSuggestion {
                start_index: 51,
                end_index: 100,
            }),
        };

        assert!(!incomplete.is_complete());
    }

    #[test]
    fn test_is_complete_empty_file() {
        let empty = TranslationStatusReport {
            ref_file: "ref.srt".to_string(),
            translation_file: "trans.srt".to_string(),
            total_count: 0,
            translated_count: 0,
            missing_count: 0,
            next_chunk: None,
        };

        assert!(!empty.is_complete());
    }
}
