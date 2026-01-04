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
