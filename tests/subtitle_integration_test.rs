use lib::subtitle::parser::parse_srt_entry;
use std::fs;

#[test]
fn test_parse_real_srt_file() {
    // Read the actual SRT test file
    let content =
        fs::read_to_string("test-data/Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt")
            .expect("Failed to read test SRT file");

    // Split into subtitle blocks (separated by blank lines)
    let blocks: Vec<&str> = content
        .split("\n\n")
        .filter(|s| !s.trim().is_empty())
        .collect();

    // Parse first few subtitles to verify format
    assert!(
        blocks.len() > 10,
        "SRT file should have multiple subtitle blocks"
    );

    // Test parsing subtitle 1
    let subtitle1 = parse_srt_entry(blocks[0]).expect("Failed to parse subtitle 1");
    assert_eq!(*subtitle1.index.as_ref(), 1);
    assert_eq!(subtitle1.start_time.as_ref().num_milliseconds(), 1436);
    assert_eq!(subtitle1.end_time.as_ref().num_milliseconds(), 3481);
    assert!(subtitle1.text.as_ref().contains("Previously on"));

    // Test parsing subtitle 2
    let subtitle2 = parse_srt_entry(blocks[1]).expect("Failed to parse subtitle 2");
    assert_eq!(*subtitle2.index.as_ref(), 2);
    assert!(subtitle2.text.as_ref().contains("Hello, Harry"));

    // Test parsing subtitle 3
    let subtitle3 = parse_srt_entry(blocks[2]).expect("Failed to parse subtitle 3");
    assert_eq!(*subtitle3.index.as_ref(), 3);
    assert_eq!(subtitle3.text.as_ref(), "You are a Grey.");

    // Verify we can format them back
    let formatted1 = subtitle1.to_string();
    assert!(formatted1.contains("00:00:01,436 --> 00:00:03,481"));

    println!(
        "Successfully parsed {} subtitle blocks from real SRT file",
        blocks.len()
    );
}

#[test]
fn test_parse_all_subtitles_from_file() {
    // Read the actual SRT test file
    let content =
        fs::read_to_string("test-data/Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt")
            .expect("Failed to read test SRT file");

    // Split into subtitle blocks
    let blocks: Vec<&str> = content
        .split("\n\n")
        .filter(|s| !s.trim().is_empty())
        .collect();

    let mut parsed_count = 0;
    let mut failed_count = 0;

    for (i, block) in blocks.iter().enumerate() {
        match parse_srt_entry(block) {
            Ok(subtitle) => {
                // Verify index matches expected
                assert_eq!(
                    *subtitle.index.as_ref() as usize,
                    i + 1,
                    "Index mismatch at block {}",
                    i + 1
                );

                // Verify duration is positive
                assert!(
                    subtitle.duration().num_milliseconds() > 0,
                    "Non-positive duration at subtitle {}",
                    i + 1
                );

                // Verify text is not empty
                assert!(
                    !subtitle.text.as_ref().trim().is_empty(),
                    "Empty text at subtitle {}",
                    i + 1
                );

                parsed_count += 1;
            }
            Err(e) => {
                eprintln!("Failed to parse block {}: {}", i + 1, e);
                eprintln!("Block content:\n{}", block);
                failed_count += 1;
            }
        }
    }

    println!(
        "Parsed: {} subtitles, Failed: {} subtitles",
        parsed_count, failed_count
    );
    assert_eq!(failed_count, 0, "Some subtitles failed to parse");
    assert!(parsed_count > 0, "No subtitles were parsed");
}
