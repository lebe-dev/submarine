use std::path::PathBuf;
use std::process::Command;

// Helper to get project root
fn get_project_root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

// Helper to run CLI command
fn run_get_command(file: &str, index: &str) -> std::process::Output {
    Command::new("cargo")
        .args(&["run", "--bin", "sm", "--", "get", file, index])
        .output()
        .expect("Failed to execute command")
}

// Helper to build test data path
fn test_data_path(filename: &str) -> String {
    get_project_root()
        .join("test-data")
        .join(filename)
        .to_str()
        .unwrap()
        .to_string()
}

// ========== Positive Test Cases ==========

#[test]
fn test_get_existing_subtitle_simple() {
    let file = test_data_path("valid/simple.srt");

    // Test getting subtitle index 1
    let output = run_get_command(&file, "1");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("1\n"));
    assert!(stdout.contains("00:00:01,000 --> 00:00:03,000"));
    assert!(stdout.contains("First subtitle"));

    // Test getting subtitle index 2
    let output = run_get_command(&file, "2");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("00:00:03,500 --> 00:00:05,500"));
    assert!(stdout.contains("Second subtitle"));

    // Test getting subtitle index 3
    let output = run_get_command(&file, "3");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("3\n"));
    assert!(stdout.contains("00:00:06,000 --> 00:00:08,000"));
    assert!(stdout.contains("Third subtitle"));
}

#[test]
fn test_get_subtitle_with_html() {
    let file = test_data_path("valid/complex.srt");

    let output = run_get_command(&file, "1");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("1\n"));
    assert!(stdout.contains("00:00:01,436 --> 00:00:03,481"));
    assert!(stdout.contains("<i>Previously on"));
    assert!(stdout.contains("\"Resident Alien\"...</i>"));
}

#[test]
fn test_get_subtitle_with_index_gap() {
    let file = test_data_path("valid/complex.srt");

    // Index 5 exists (skips 3, 4)
    let output = run_get_command(&file, "5");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("5\n"));
    assert!(stdout.contains("00:00:07,790 --> 00:00:09,357"));
    assert!(stdout.contains("<i>You should be"));
}

#[test]
fn test_get_subtitle_from_real_file() {
    let file = test_data_path("Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt");

    // Test getting subtitle index 1
    let output = run_get_command(&file, "1");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("1\n"));
    assert!(stdout.contains("00:00:01,436 --> 00:00:03,481"));

    // Test getting subtitle index 2
    let output = run_get_command(&file, "2");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("00:00:03,481 --> 00:00:05,135"));
}

#[test]
fn test_get_edge_case_indices() {
    let file = test_data_path("valid/simple.srt");

    // Test first subtitle (index 1)
    let output = run_get_command(&file, "1");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("First subtitle"));

    // Test last subtitle (index 3 in simple.srt)
    let output = run_get_command(&file, "3");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Third subtitle"));
}

// ========== Negative Test Cases ==========

#[test]
fn test_get_nonexistent_index() {
    let file = test_data_path("valid/simple.srt");

    let output = run_get_command(&file, "99999");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("not found"));
    assert!(stderr.contains("99999"));
}

#[test]
fn test_get_file_not_found() {
    let output = run_get_command("nonexistent.srt", "1");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("error") || stderr.contains("Error"));
    assert!(
        stderr.contains("not found")
            || stderr.contains("No such file")
            || stderr.contains("resolve file path")
    );
}

#[test]
fn test_get_malformed_file() {
    let file = test_data_path("invalid/malformed.srt");

    let output = run_get_command(&file, "1");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr).to_lowercase();
    assert!(stderr.contains("error"));
    assert!(stderr.contains("parse") || stderr.contains("invalid"));
}

#[test]
fn test_get_empty_file() {
    let file = test_data_path("invalid/empty.srt");

    let output = run_get_command(&file, "1");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("not found"));
}

#[test]
fn test_get_path_traversal() {
    let output = run_get_command("../../../etc/passwd", "1");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("error") || stderr.contains("Error"));
    assert!(
        stderr.contains("Invalid")
            || stderr.contains("path")
            || stderr.contains("resolve file path")
    );
}

#[test]
fn test_get_index_zero() {
    let file = test_data_path("valid/simple.srt");

    // Index 0 is not valid in SRT (indices start at 1)
    let output = run_get_command(&file, "0");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("not found"));
}

#[test]
fn test_get_with_absolute_path() {
    use std::fs;
    use tempfile::TempDir;

    // Create temp file with known content
    let temp_dir = TempDir::new().unwrap();
    let temp_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:03,000\nTest subtitle\n";
    fs::write(&temp_file, content).unwrap();

    // Test using absolute path
    let output = run_get_command(temp_file.to_str().unwrap(), "1");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Test subtitle"));
}

// ========== Tests for get with range syntax ==========

#[test]
fn test_get_with_range_all_exist() {
    let file = test_data_path("valid/simple.srt");

    // Use get command with range syntax
    let output = run_get_command(&file, "1-3");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should return all 3 subtitles
    assert!(stdout.contains("1\n"));
    assert!(stdout.contains("First subtitle"));
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("Second subtitle"));
    assert!(stdout.contains("3\n"));
    assert!(stdout.contains("Third subtitle"));
}

#[test]
fn test_get_with_range_single() {
    let file = test_data_path("valid/simple.srt");

    // Use get command with range syntax for single subtitle
    let output = run_get_command(&file, "2-2");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should return only subtitle 2
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("Second subtitle"));
    assert!(!stdout.contains("First subtitle"));
    assert!(!stdout.contains("Third subtitle"));
}

#[test]
fn test_get_with_range_invalid_format() {
    let file = test_data_path("valid/simple.srt");

    // Invalid range format
    let output = run_get_command(&file, "5-2");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("error"));
}

#[test]
fn test_get_with_invalid_index() {
    let file = test_data_path("valid/simple.srt");

    // Invalid non-numeric index
    let output = run_get_command(&file, "abc");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Invalid index"));
}

// ========== Additional Range Test Cases ==========

#[test]
fn test_get_range_partial_overlap() {
    let file = test_data_path("valid/complex.srt");

    // File has indices 1, 2, 5, 6, ...
    // Request 2-4 should only return subtitle 2
    let output = run_get_command(&file, "2-4");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should contain subtitle 2
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("00:00:03,481 --> 00:00:05,135"));
    assert!(stdout.contains("Hello, Harry."));

    // Should NOT contain subtitle 1
    assert!(!stdout.contains("Previously on"));

    // Should NOT contain subtitle 5
    assert!(!stdout.contains("You should be"));
}

#[test]
fn test_get_range_no_subtitles() {
    let file = test_data_path("valid/complex.srt");

    // File has indices 1, 2, 5, 6, ... (no 3, 4)
    // Request 3-4 should return empty result
    let output = run_get_command(&file, "3-4");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    assert!(stdout.contains("No subtitles found in range 3-4"));
}

#[test]
fn test_get_range_subset() {
    let file = test_data_path("valid/simple.srt");

    // Request range 1-2 (first two of three)
    let output = run_get_command(&file, "1-2");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should contain subtitles 1 and 2
    assert!(stdout.contains("First subtitle"));
    assert!(stdout.contains("Second subtitle"));

    // Should NOT contain subtitle 3
    assert!(!stdout.contains("Third subtitle"));
}

#[test]
fn test_get_range_with_html_tags() {
    let file = test_data_path("valid/complex.srt");

    // Get range that includes subtitles with HTML
    let output = run_get_command(&file, "1-2");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should preserve HTML tags
    assert!(stdout.contains("<i>Previously on"));
    assert!(stdout.contains("\"Resident Alien\"...</i>"));
    assert!(stdout.contains("Hello, Harry."));
}

#[test]
fn test_get_range_multiline_text() {
    let file = test_data_path("valid/complex.srt");

    // Subtitle 1 has multiline text
    let output = run_get_command(&file, "1-1");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should contain both lines
    assert!(stdout.contains("Previously on"));
    assert!(stdout.contains("Resident Alien"));
}

#[test]
fn test_get_range_beyond_end() {
    let file = test_data_path("valid/simple.srt");

    // Request range 2-100 (only 2 and 3 exist)
    let output = run_get_command(&file, "2-100");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should contain subtitles 2 and 3
    assert!(stdout.contains("Second subtitle"));
    assert!(stdout.contains("Third subtitle"));

    // Should NOT contain subtitle 1
    assert!(!stdout.contains("First subtitle"));
}

#[test]
fn test_get_range_start_zero() {
    let file = test_data_path("valid/simple.srt");

    // Start index 0 is invalid (SRT indices start at 1)
    let output = run_get_command(&file, "0-5");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);

    assert!(stderr.contains("error"));
    assert!(stderr.contains("Start index must be >= 1"));
}

#[test]
fn test_get_range_end_zero() {
    let file = test_data_path("valid/simple.srt");

    // End index 0 is invalid
    let output = run_get_command(&file, "1-0");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);

    assert!(stderr.contains("error"));
    // Will fail on either end validation OR range order validation
    assert!(stderr.contains("End index must be >= 1") || stderr.contains("Start index must be <="));
}

#[test]
fn test_get_range_empty_file() {
    let file = test_data_path("invalid/empty.srt");

    let output = run_get_command(&file, "1-5");
    assert!(output.status.success()); // Empty file is valid, just returns "No subtitles found"
    let stdout = String::from_utf8_lossy(&output.stdout);

    assert!(stdout.contains("No subtitles found in range 1-5"));
}

#[test]
fn test_get_range_output_format() {
    let file = test_data_path("valid/simple.srt");

    // Get range 1-2 and verify SRT formatting
    let output = run_get_command(&file, "1-2");
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Should have proper SRT format with double newlines between subtitles
    // Verify structure: index, timestamp, text, blank line, next subtitle
    let lines: Vec<&str> = stdout.lines().collect();

    // Find subtitle 1
    let idx1_pos = lines.iter().position(|&l| l == "1").unwrap();
    assert!(lines[idx1_pos + 1].contains("00:00:01,000 --> 00:00:03,000"));
    assert!(lines[idx1_pos + 2].contains("First subtitle"));

    // There should be a blank line after subtitle 1
    assert_eq!(lines[idx1_pos + 3], "");

    // Find subtitle 2
    let idx2_pos = lines.iter().position(|&l| l == "2").unwrap();
    assert!(lines[idx2_pos + 1].contains("00:00:03,500 --> 00:00:05,500"));
    assert!(lines[idx2_pos + 2].contains("Second subtitle"));
}

#[test]
fn test_get_range_invalid_format_variations() {
    let file = test_data_path("valid/simple.srt");

    // Test various invalid formats
    let output = run_get_command(&file, "123-");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Invalid"));

    let output = run_get_command(&file, "abc-123");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Invalid"));

    let output = run_get_command(&file, "123-abc");
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Invalid"));
}
