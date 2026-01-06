use chrono::Duration;
use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use lib::translation_status::service;
use std::fs;
use tempfile::TempDir;

fn make_test_subtitle(index: u32, start_ms: i64, end_ms: i64, text: &str) -> Subtitle {
    Subtitle::new(
        SubtitleIndex::try_new(index).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(start_ms)).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(end_ms)).unwrap(),
        SubtitleText::try_new(text.to_string()).unwrap(),
    )
    .unwrap()
}

fn create_srt_file(dir: &TempDir, filename: &str, subtitles: Vec<Subtitle>) -> String {
    let file_path = dir.path().join(filename);
    let service = SubRipService::new(dir.path().to_path_buf());

    // Create empty file first
    fs::write(&file_path, "").unwrap();

    // Add each subtitle
    for sub in subtitles {
        service
            .add(filename, sub.start_time, sub.end_time, sub.text)
            .unwrap();
    }

    file_path.to_str().unwrap().to_string()
}

#[test]
fn test_empty_translation_file() {
    let temp_dir = TempDir::new().unwrap();

    let ref_subs = vec![
        make_test_subtitle(1, 1000, 2000, "English 1"),
        make_test_subtitle(2, 2000, 3000, "English 2"),
        make_test_subtitle(3, 3000, 4000, "English 3"),
    ];

    let translation_subs = vec![];

    create_srt_file(&temp_dir, "reference.srt", ref_subs.clone());
    create_srt_file(&temp_dir, "translation.srt", translation_subs.clone());

    let report = service::check_translation_status(
        ref_subs,
        "reference.srt".to_string(),
        translation_subs,
        "translation.srt".to_string(),
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
    let temp_dir = TempDir::new().unwrap();

    let ref_subs = vec![
        make_test_subtitle(1, 1000, 2000, "English 1"),
        make_test_subtitle(2, 2000, 3000, "English 2"),
        make_test_subtitle(3, 3000, 4000, "English 3"),
        make_test_subtitle(4, 4000, 5000, "English 4"),
        make_test_subtitle(5, 5000, 6000, "English 5"),
    ];

    let translation_subs = vec![
        make_test_subtitle(1, 1000, 2000, "Russian 1"),
        make_test_subtitle(2, 2000, 3000, "Russian 2"),
    ];

    create_srt_file(&temp_dir, "reference.srt", ref_subs.clone());
    create_srt_file(&temp_dir, "translation.srt", translation_subs.clone());

    let report = service::check_translation_status(
        ref_subs,
        "reference.srt".to_string(),
        translation_subs,
        "translation.srt".to_string(),
        50,
    );

    assert_eq!(report.total_count, 5);
    assert_eq!(report.translated_count, 2);
    assert_eq!(report.missing_count, 3);
    assert_eq!(report.progress_percentage(), 40.0);
    assert!(!report.is_complete());

    assert!(report.next_chunk.is_some());
    let chunk = report.next_chunk.unwrap();
    assert_eq!(chunk.start_index, 3);
    assert_eq!(chunk.end_index, 5);
}

#[test]
fn test_complete_translation() {
    let temp_dir = TempDir::new().unwrap();

    let ref_subs = vec![
        make_test_subtitle(1, 1000, 2000, "English 1"),
        make_test_subtitle(2, 2000, 3000, "English 2"),
        make_test_subtitle(3, 3000, 4000, "English 3"),
    ];

    let translation_subs = vec![
        make_test_subtitle(1, 1000, 2000, "Russian 1"),
        make_test_subtitle(2, 2000, 3000, "Russian 2"),
        make_test_subtitle(3, 3000, 4000, "Russian 3"),
    ];

    create_srt_file(&temp_dir, "reference.srt", ref_subs.clone());
    create_srt_file(&temp_dir, "translation.srt", translation_subs.clone());

    let report = service::check_translation_status(
        ref_subs,
        "reference.srt".to_string(),
        translation_subs,
        "translation.srt".to_string(),
        50,
    );

    assert_eq!(report.total_count, 3);
    assert_eq!(report.translated_count, 3);
    assert_eq!(report.missing_count, 0);
    assert_eq!(report.progress_percentage(), 100.0);
    assert!(report.is_complete());
    assert!(report.next_chunk.is_none());
}

#[test]
fn test_translation_with_gaps() {
    let temp_dir = TempDir::new().unwrap();

    let ref_subs = vec![
        make_test_subtitle(1, 1000, 2000, "English 1"),
        make_test_subtitle(2, 2000, 3000, "English 2"),
        make_test_subtitle(5, 5000, 6000, "English 5"),
        make_test_subtitle(10, 10000, 11000, "English 10"),
    ];

    let translation_subs = vec![
        make_test_subtitle(1, 1000, 2000, "Russian 1"),
        make_test_subtitle(10, 10000, 11000, "Russian 10"),
    ];

    create_srt_file(&temp_dir, "reference.srt", ref_subs.clone());
    create_srt_file(&temp_dir, "translation.srt", translation_subs.clone());

    let report = service::check_translation_status(
        ref_subs,
        "reference.srt".to_string(),
        translation_subs,
        "translation.srt".to_string(),
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
fn test_custom_chunk_size() {
    let temp_dir = TempDir::new().unwrap();

    let ref_subs: Vec<Subtitle> = (1..=100)
        .map(|i| {
            make_test_subtitle(
                i,
                i as i64 * 1000,
                (i as i64 + 1) * 1000,
                &format!("Text {}", i),
            )
        })
        .collect();

    let translation_subs = vec![];

    create_srt_file(&temp_dir, "reference.srt", ref_subs.clone());
    create_srt_file(&temp_dir, "translation.srt", translation_subs.clone());

    // Test with chunk size of 25
    let report = service::check_translation_status(
        ref_subs.clone(),
        "reference.srt".to_string(),
        translation_subs.clone(),
        "translation.srt".to_string(),
        25,
    );

    assert!(report.next_chunk.is_some());
    let chunk = report.next_chunk.unwrap();
    assert_eq!(chunk.start_index, 1);
    assert_eq!(chunk.end_index, 25);

    // Test with chunk size of 10
    let report = service::check_translation_status(
        ref_subs,
        "reference.srt".to_string(),
        translation_subs,
        "translation.srt".to_string(),
        10,
    );

    assert!(report.next_chunk.is_some());
    let chunk = report.next_chunk.unwrap();
    assert_eq!(chunk.start_index, 1);
    assert_eq!(chunk.end_index, 10);
}

#[test]
fn test_chunk_size_larger_than_remaining() {
    let temp_dir = TempDir::new().unwrap();

    let ref_subs = vec![
        make_test_subtitle(1, 1000, 2000, "English 1"),
        make_test_subtitle(2, 2000, 3000, "English 2"),
        make_test_subtitle(3, 3000, 4000, "English 3"),
    ];

    let translation_subs = vec![make_test_subtitle(1, 1000, 2000, "Russian 1")];

    create_srt_file(&temp_dir, "reference.srt", ref_subs.clone());
    create_srt_file(&temp_dir, "translation.srt", translation_subs.clone());

    let report = service::check_translation_status(
        ref_subs,
        "reference.srt".to_string(),
        translation_subs,
        "translation.srt".to_string(),
        100,
    );

    assert!(report.next_chunk.is_some());
    let chunk = report.next_chunk.unwrap();
    assert_eq!(chunk.start_index, 2);
    assert_eq!(chunk.end_index, 3); // Capped at last missing index
}
