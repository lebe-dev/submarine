use chrono::Duration;
use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};

fn main() {
    println!("=== Subtitle Struct Usage Examples (with Nutype) ===\n");

    // Example 1: Creating a subtitle manually with nutype wrappers
    println!("1. Creating a subtitle manually with validated types:");
    let subtitle1 = Subtitle::new(
        SubtitleIndex::try_new(1).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(1436)).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(3481)).unwrap(),
        SubtitleText::try_new("<i>Previously on\n\"Resident Alien\"...</i>".to_string()).unwrap(),
    )
    .expect("Failed to create subtitle");

    println!("{}", subtitle1);
    println!();

    // Example 2: Creating another subtitle with timestamp parsing
    println!("2. Creating subtitle with timestamp parsing:");
    let start_time = Subtitle::parse_timestamp("00:00:03,481").expect("Failed to parse start");
    let end_time = Subtitle::parse_timestamp("00:00:05,135").expect("Failed to parse end");

    let subtitle2 = Subtitle::new(
        SubtitleIndex::try_new(2).unwrap(),
        SubtitleTimestamp::try_new(start_time).unwrap(),
        SubtitleTimestamp::try_new(end_time).unwrap(),
        SubtitleText::try_new("Hello, Harry.\nMy name is Joseph.".to_string()).unwrap(),
    )
    .expect("Failed to create subtitle");

    println!("Index: {}", subtitle2.index);
    println!(
        "Start: {}ms",
        subtitle2.start_time.as_ref().num_milliseconds()
    );
    println!("End: {}ms", subtitle2.end_time.as_ref().num_milliseconds());
    println!("Duration: {}ms", subtitle2.duration().num_milliseconds());
    println!("Text: {}", subtitle2.text);
    println!("Line count: {}", subtitle2.line_count());
    println!();

    // Example 3: Working with HTML tags
    println!("3. Working with HTML tags:");
    let subtitle3 = Subtitle::new(
        SubtitleIndex::try_new(3).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(8000)).unwrap(),
        SubtitleText::try_new("<i>Italic text</i> and <b>bold text</b>".to_string()).unwrap(),
    )
    .expect("Failed to create subtitle");

    println!("Has HTML tags: {}", subtitle3.has_html_tags());
    println!("Original text: {}", subtitle3.text);
    println!("Without HTML: {}", subtitle3.text_without_html());
    println!();

    // Example 4: Display trait (formatting back to SRT)
    println!("4. Display trait (formatting back to SRT):");
    println!("{}", subtitle1);
    println!();

    // Example 5: Nutype validation examples
    println!("5. Nutype validation examples:");

    // Invalid: index = 0
    match SubtitleIndex::try_new(0) {
        Ok(_) => println!("  ✗ Should have failed (index = 0)"),
        Err(e) => println!("  ✓ SubtitleIndex rejected 0: {:?}", e),
    }

    // Invalid: negative timestamp
    match SubtitleTimestamp::try_new(Duration::milliseconds(-1)) {
        Ok(_) => println!("  ✗ Should have failed (negative timestamp)"),
        Err(e) => println!("  ✓ SubtitleTimestamp rejected negative: {:?}", e),
    }

    // Invalid: empty text
    match SubtitleText::try_new("   ".to_string()) {
        Ok(_) => println!("  ✗ Should have failed (empty text)"),
        Err(e) => println!("  ✓ SubtitleText rejected whitespace: {:?}", e),
    }

    // Cross-field validation: end_time <= start_time
    match Subtitle::new(
        SubtitleIndex::try_new(1).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap(),
        SubtitleTimestamp::try_new(Duration::milliseconds(1000)).unwrap(),
        SubtitleText::try_new("Text".to_string()).unwrap(),
    ) {
        Ok(_) => println!("  ✗ Should have failed (end <= start)"),
        Err(e) => println!("  ✓ Correctly rejected: {}", e),
    }
    println!();

    // Example 6: Text sanitization
    println!("6. Text sanitization (automatic trimming):");
    let text_with_spaces = SubtitleText::try_new("  Hello World  ".to_string()).unwrap();
    println!("Original input: '  Hello World  '");
    println!("Sanitized text: '{}'", text_with_spaces.as_ref());
    println!();

    // Example 7: Timestamp formatting
    println!("7. Timestamp parsing and formatting:");
    let timestamp_str = "00:01:23,456";
    let parsed = Subtitle::parse_timestamp(timestamp_str).expect("Failed to parse");
    let formatted = Subtitle::format_timestamp(&parsed);
    println!("Original: {}", timestamp_str);
    println!("Parsed: {}ms", parsed.num_milliseconds());
    println!("Formatted: {}", formatted);
    println!("Round-trip: {}", timestamp_str == formatted);
}
