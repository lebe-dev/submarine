use crate::subtitle::model::Subtitle;
use crate::translation_status::model::{ChunkSuggestion, TranslationStatusReport};
use log::debug;
use std::collections::HashSet;

/// Check translation status by comparing reference and translation subtitle files
///
/// # Arguments
///
/// * `ref_subs` - Subtitles from the reference file
/// * `ref_file` - Reference file name for display
/// * `translation_subs` - Subtitles from the translation file
/// * `translation_file` - Translation file name for display
/// * `chunk_size` - Size of the suggested chunk for next translation
///
/// # Returns
///
/// A `TranslationStatusReport` containing progress information and next chunk suggestion
pub fn check_translation_status(
    ref_subs: Vec<Subtitle>,
    ref_file: String,
    translation_subs: Vec<Subtitle>,
    translation_file: String,
    chunk_size: usize,
) -> TranslationStatusReport {
    let total_count = ref_subs.len();

    debug!(
        "comparing {} reference subtitles with {} translation subtitles",
        total_count,
        translation_subs.len()
    );

    debug!("extracting indices from both files");
    let ref_indices: HashSet<u32> = ref_subs.iter().map(|s| *s.index.as_ref()).collect();

    let translation_indices: HashSet<u32> =
        translation_subs.iter().map(|s| *s.index.as_ref()).collect();

    debug!("finding translated indices (intersection)");
    let translated: HashSet<u32> = ref_indices
        .intersection(&translation_indices)
        .copied()
        .collect();

    let translated_count = translated.len();

    debug!("finding missing indices (in reference but not in translation)");
    let mut missing: Vec<u32> = ref_indices
        .difference(&translation_indices)
        .copied()
        .collect();

    missing.sort();
    let missing_count = missing.len();

    debug!(
        "found {} translated, {} missing",
        translated_count, missing_count
    );

    debug!("calculating next chunk suggestion");
    let next_chunk = if missing.is_empty() {
        None
    } else {
        let start_index = missing[0];
        let last_missing = missing[missing.len() - 1];
        let end_index = std::cmp::min(start_index + chunk_size as u32 - 1, last_missing);

        debug!("next chunk suggestion: {}-{}", start_index, end_index);

        Some(ChunkSuggestion {
            start_index,
            end_index,
        })
    };

    TranslationStatusReport {
        ref_file,
        translation_file,
        total_count,
        translated_count,
        missing_count,
        next_chunk,
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
    fn test_empty_translation() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
            make_test_subtitle(3, 3000, 4000, "Text 3"),
        ];

        let translation_subs = vec![];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            50,
        );

        assert_eq!(report.total_count, 3);
        assert_eq!(report.translated_count, 0);
        assert_eq!(report.missing_count, 3);
        assert_eq!(report.progress_percentage(), 0.0);
        assert!(!report.is_complete());

        assert!(report.next_chunk.is_some());
        let chunk = report.next_chunk.unwrap();
        assert_eq!(chunk.start_index, 1);
        assert_eq!(chunk.end_index, 3);
    }

    #[test]
    fn test_partial_translation() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
            make_test_subtitle(3, 3000, 4000, "Text 3"),
            make_test_subtitle(4, 4000, 5000, "Text 4"),
            make_test_subtitle(5, 5000, 6000, "Text 5"),
        ];

        let translation_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Translated 1"),
            make_test_subtitle(2, 2000, 3000, "Translated 2"),
        ];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            2,
        );

        assert_eq!(report.total_count, 5);
        assert_eq!(report.translated_count, 2);
        assert_eq!(report.missing_count, 3);
        assert_eq!(report.progress_percentage(), 40.0);
        assert!(!report.is_complete());

        assert!(report.next_chunk.is_some());
        let chunk = report.next_chunk.unwrap();
        assert_eq!(chunk.start_index, 3);
        assert_eq!(chunk.end_index, 4);
    }

    #[test]
    fn test_complete_translation() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
        ];

        let translation_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Translated 1"),
            make_test_subtitle(2, 2000, 3000, "Translated 2"),
        ];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            50,
        );

        assert_eq!(report.total_count, 2);
        assert_eq!(report.translated_count, 2);
        assert_eq!(report.missing_count, 0);
        assert_eq!(report.progress_percentage(), 100.0);
        assert!(report.is_complete());
        assert!(report.next_chunk.is_none());
    }

    #[test]
    fn test_translation_with_gaps() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
            make_test_subtitle(5, 5000, 6000, "Text 5"),
            make_test_subtitle(10, 10000, 11000, "Text 10"),
        ];

        let translation_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Translated 1"),
            make_test_subtitle(10, 10000, 11000, "Translated 10"),
        ];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            50,
        );

        assert_eq!(report.total_count, 4);
        assert_eq!(report.translated_count, 2);
        assert_eq!(report.missing_count, 2);
        assert_eq!(report.progress_percentage(), 50.0);

        assert!(report.next_chunk.is_some());
        let chunk = report.next_chunk.unwrap();
        assert_eq!(chunk.start_index, 2);
        assert_eq!(chunk.end_index, 5);
    }

    #[test]
    fn test_chunk_size_larger_than_remaining() {
        let ref_subs = vec![
            make_test_subtitle(1, 1000, 2000, "Text 1"),
            make_test_subtitle(2, 2000, 3000, "Text 2"),
            make_test_subtitle(3, 3000, 4000, "Text 3"),
        ];

        let translation_subs = vec![make_test_subtitle(1, 1000, 2000, "Translated 1")];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            100,
        );

        assert_eq!(report.missing_count, 2);

        assert!(report.next_chunk.is_some());
        let chunk = report.next_chunk.unwrap();
        assert_eq!(chunk.start_index, 2);
        assert_eq!(chunk.end_index, 3); // Capped at last missing index
    }

    #[test]
    fn test_empty_files() {
        let ref_subs = vec![];
        let translation_subs = vec![];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            50,
        );

        assert_eq!(report.total_count, 0);
        assert_eq!(report.translated_count, 0);
        assert_eq!(report.missing_count, 0);
        assert_eq!(report.progress_percentage(), 0.0);
        assert!(!report.is_complete());
        assert!(report.next_chunk.is_none());
    }

    #[test]
    fn test_non_sequential_indices() {
        let ref_subs = vec![
            make_test_subtitle(10, 1000, 2000, "Text 10"),
            make_test_subtitle(20, 2000, 3000, "Text 20"),
            make_test_subtitle(30, 3000, 4000, "Text 30"),
        ];

        let translation_subs = vec![make_test_subtitle(10, 1000, 2000, "Translated 10")];

        let report = check_translation_status(
            ref_subs,
            "ref.srt".to_string(),
            translation_subs,
            "trans.srt".to_string(),
            5,
        );

        assert_eq!(report.total_count, 3);
        assert_eq!(report.translated_count, 1);
        assert_eq!(report.missing_count, 2);

        assert!(report.next_chunk.is_some());
        let chunk = report.next_chunk.unwrap();
        assert_eq!(chunk.start_index, 20);
        assert_eq!(chunk.end_index, 24);
    }
}
