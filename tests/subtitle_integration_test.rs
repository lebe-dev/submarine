use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use std::path::PathBuf;

#[test]
fn test_parse_real_srt_file() {
    // Create service pointing to test-data directory
    let service = SubRipService::new(PathBuf::from("test-data"));
    let filename = "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt";

    // Test subtitle 1
    let subtitle1 = service
        .get_by_id(filename, 1)
        .expect("Failed to get subtitle 1")
        .expect("Subtitle 1 not found");

    assert_eq!(*subtitle1.index.as_ref(), 1);
    assert_eq!(subtitle1.start_time.as_ref().num_milliseconds(), 1436);
    assert_eq!(subtitle1.end_time.as_ref().num_milliseconds(), 3481);
    assert!(subtitle1.text.as_ref().contains("Previously on"));

    // Test subtitle 2
    let subtitle2 = service
        .get_by_id(filename, 2)
        .expect("Failed to get subtitle 2")
        .expect("Subtitle 2 not found");

    assert_eq!(*subtitle2.index.as_ref(), 2);
    assert!(subtitle2.text.as_ref().contains("Hello, Harry"));

    // Test subtitle 3
    let subtitle3 = service
        .get_by_id(filename, 3)
        .expect("Failed to get subtitle 3")
        .expect("Subtitle 3 not found");

    assert_eq!(*subtitle3.index.as_ref(), 3);
    assert_eq!(subtitle3.text.as_ref(), "You are a Grey.");

    // Verify we can format them back
    let formatted1 = subtitle1.to_string();
    assert!(formatted1.contains("00:00:01,436 --> 00:00:03,481"));

    println!("Successfully tested subtitle retrieval from real SRT file");
}

#[test]
fn test_parse_all_subtitles_from_file() {
    // Create service pointing to test-data directory
    let service = SubRipService::new(PathBuf::from("test-data"));
    let filename = "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt";

    // Test retrieving subtitles with various indices
    // We'll test the first 20 subtitles (assuming sequential indices)
    let mut found_count = 0;
    let mut last_found_index = 0;

    for i in 1..=100 {
        match service.get_by_id(filename, i) {
            Ok(Some(subtitle)) => {
                // Verify index matches
                assert_eq!(
                    *subtitle.index.as_ref(),
                    i,
                    "Index mismatch at subtitle {}",
                    i
                );

                // Verify duration is positive
                assert!(
                    subtitle.duration().num_milliseconds() > 0,
                    "Non-positive duration at subtitle {}",
                    i
                );

                // Verify text is not empty
                assert!(
                    !subtitle.text.as_ref().trim().is_empty(),
                    "Empty text at subtitle {}",
                    i
                );

                found_count += 1;
                last_found_index = i;
            }
            Ok(None) => {
                // Subtitle with this index doesn't exist, continue
                continue;
            }
            Err(e) => {
                panic!("Failed to retrieve subtitle {}: {}", i, e);
            }
        }
    }

    println!(
        "Successfully retrieved and validated {} subtitles (last index: {})",
        found_count, last_found_index
    );
    assert!(found_count > 10, "Should have found more than 10 subtitles");
}

#[test]
fn test_nonexistent_subtitle() {
    let service = SubRipService::new(PathBuf::from("test-data"));
    let filename = "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt";

    // Try to get a subtitle with a very high index that likely doesn't exist
    let result = service.get_by_id(filename, 99999);

    assert!(result.is_ok());
    assert!(
        result.unwrap().is_none(),
        "Should return None for non-existent subtitle"
    );
}

#[test]
fn test_subtitle_file_not_found() {
    let service = SubRipService::new(PathBuf::from("test-data"));

    let result = service.get_by_id("nonexistent.srt", 1);

    assert!(result.is_err(), "Should return error for non-existent file");
}

#[test]
fn test_parse_well_formed_srt_file() {
    // This test verifies that the parser correctly handles well-formed SRT files
    let service = SubRipService::new(PathBuf::from("test-data"));
    let filename = "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt";

    // Test a few subtitles to ensure parsing works correctly
    let subtitle1 = service
        .get_by_id(filename, 1)
        .expect("Failed to get subtitle 1")
        .expect("Subtitle 1 not found");

    assert_eq!(*subtitle1.index.as_ref(), 1);
    assert!(subtitle1.text.as_ref().contains("Previously on"));

    let subtitle100 = service
        .get_by_id(filename, 100)
        .expect("Failed to get subtitle 100")
        .expect("Subtitle 100 not found");

    assert_eq!(*subtitle100.index.as_ref(), 100);

    println!("Successfully verified parser works with well-formed SRT files");
}

#[test]
fn test_parse_russian_srt_with_blank_lines() {
    // Test parsing of Russian subtitle file with blank lines within text
    let service = SubRipService::new(PathBuf::from("test-data"));
    let filename = "russian-with-blank-lines-in-text.rus.srt";

    let result = service.get_all(filename);
    assert!(
        result.is_ok(),
        "Failed to parse {}: {:?}",
        filename,
        result.err()
    );

    let subtitles = result.unwrap();
    assert_eq!(subtitles.len(), 4, "Expected 4 subtitles in {}", filename);

    // Verify block 3 contains both lines with blank line preserved
    let sub3 = &subtitles[2]; // 0-indexed
    assert_eq!(*sub3.index.as_ref(), 3);
    assert_eq!(
        sub3.text.as_ref(),
        "Может, он не умер.\n\nЧто?",
        "Block 3 should contain both lines separated by blank line"
    );

    // Verify other subtitles parsed correctly
    let sub1 = &subtitles[0];
    assert_eq!(*sub1.index.as_ref(), 1);
    assert_eq!(sub1.text.as_ref(), "Ранее в сериале...");

    let sub2 = &subtitles[1];
    assert_eq!(*sub2.index.as_ref(), 2);
    assert_eq!(sub2.text.as_ref(), "Залезай!");

    let sub4 = &subtitles[3];
    assert_eq!(*sub4.index.as_ref(), 4);
    assert_eq!(sub4.text.as_ref(), "Мы должны вернуться и проверить.");

    println!("Successfully parsed Russian SRT file with blank lines in subtitle text");
}
