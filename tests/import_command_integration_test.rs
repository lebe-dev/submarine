use std::fs;
use std::path::PathBuf;
use std::process::Command;
use tempfile::TempDir;

// ========== Helper Functions ==========

fn run_import_command(srt_file: &str, csv_file: &str, delimiter: &str) -> std::process::Output {
    Command::new("cargo")
        .args(&[
            "run",
            "--bin",
            "sm",
            "--",
            "import",
            srt_file,
            csv_file,
            "--delimiter",
            delimiter,
        ])
        .output()
        .expect("Failed to execute command")
}

fn get_project_root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

fn test_data_path(filename: &str) -> String {
    get_project_root()
        .join("test-data")
        .join(filename)
        .to_str()
        .unwrap()
        .to_string()
}

fn create_temp_csv(dir: &TempDir, filename: &str, content: &str) -> PathBuf {
    let csv_path = dir.path().join(filename);
    fs::write(&csv_path, content).unwrap();
    csv_path
}

fn create_empty_srt(dir: &TempDir, filename: &str) -> PathBuf {
    let srt_path = dir.path().join(filename);
    fs::write(&srt_path, "").unwrap();
    srt_path
}

fn create_srt_with_content(dir: &TempDir, filename: &str, content: &str) -> PathBuf {
    let srt_path = dir.path().join(filename);
    fs::write(&srt_path, content).unwrap();
    srt_path
}

// ========== Mandatory Tests (User Requirements) ==========

#[test]
fn test_import_creates_new_srt_file() {
    // Test that import works with an empty SRT file (effectively creating new content)
    let temp_dir = TempDir::new().unwrap();

    let csv_content = "start_time|end_time|text
00:00:01,000|00:00:03,000|First subtitle
00:00:04,000|00:00:06,000|Second subtitle";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);
    let srt_file = create_empty_srt(&temp_dir, "new_file.srt");

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success(), "Command should succeed");

    // Verify file contains imported subtitles
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("1\n"));
    assert!(content.contains("00:00:01,000 --> 00:00:03,000"));
    assert!(content.contains("First subtitle"));
    assert!(content.contains("2\n"));
    assert!(content.contains("00:00:04,000 --> 00:00:06,000"));
    assert!(content.contains("Second subtitle"));

    // Verify stdout shows success
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Imported: 2 subtitles"));
    assert!(stdout.contains("✓ Subtitles imported successfully"));
}

#[test]
fn test_multiple_imports_accumulate_data() {
    // Test that multiple import runs accumulate data in the result file
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "accumulated.srt");

    // First import: 2 subtitles
    let csv1_content = "start_time|end_time|text
00:00:01,000|00:00:02,000|First
00:00:03,000|00:00:04,000|Second";
    let csv1_file = create_temp_csv(&temp_dir, "import1.csv", csv1_content);

    let output1 = run_import_command(srt_file.to_str().unwrap(), csv1_file.to_str().unwrap(), "|");
    assert!(output1.status.success());

    let stdout1 = String::from_utf8_lossy(&output1.stdout);
    assert!(stdout1.contains("Imported: 2 subtitles"));
    assert!(stdout1.contains("Index range: 1-2"));
    assert!(stdout1.contains("Total subtitles: 2"));

    // Verify first import content
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("First"));
    assert!(content.contains("Second"));

    // Second import: 3 more subtitles
    let csv2_content = "start_time|end_time|text
00:00:05,000|00:00:06,000|Third
00:00:07,000|00:00:08,000|Fourth
00:00:09,000|00:00:10,000|Fifth";
    let csv2_file = create_temp_csv(&temp_dir, "import2.csv", csv2_content);

    let output2 = run_import_command(srt_file.to_str().unwrap(), csv2_file.to_str().unwrap(), "|");
    assert!(output2.status.success());

    let stdout2 = String::from_utf8_lossy(&output2.stdout);
    assert!(stdout2.contains("Imported: 3 subtitles"));
    assert!(stdout2.contains("Index range: 3-5"));
    assert!(stdout2.contains("Total subtitles: 5"));

    // Verify all data from both imports is present
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("First"));
    assert!(content.contains("Second"));
    assert!(content.contains("Third"));
    assert!(content.contains("Fourth"));
    assert!(content.contains("Fifth"));

    // Third import: 1 more subtitle
    let csv3_content = "start_time|end_time|text
00:00:11,000|00:00:13,000|Sixth";
    let csv3_file = create_temp_csv(&temp_dir, "import3.csv", csv3_content);

    let output3 = run_import_command(srt_file.to_str().unwrap(), csv3_file.to_str().unwrap(), "|");
    assert!(output3.status.success());

    let stdout3 = String::from_utf8_lossy(&output3.stdout);
    assert!(stdout3.contains("Imported: 1 subtitles"));
    assert!(stdout3.contains("Index range: 6-6"));
    assert!(stdout3.contains("Total subtitles: 6"));

    // Verify final content has all 6 subtitles
    let final_content = fs::read_to_string(&srt_file).unwrap();
    assert!(final_content.contains("First"));
    assert!(final_content.contains("Second"));
    assert!(final_content.contains("Third"));
    assert!(final_content.contains("Fourth"));
    assert!(final_content.contains("Fifth"));
    assert!(final_content.contains("Sixth"));
}

// ========== Basic Functionality Tests ==========

#[test]
fn test_import_with_pipe_delimiter() {
    // Test import using the sample CSV file with pipe delimiter
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_file = test_data_path("sample-import.csv");

    let output = run_import_command(srt_file.to_str().unwrap(), &csv_file, "|");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Imported: 4 subtitles"));

    // Verify multiline text and HTML tags are preserved
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("<i>Previously on"));
    assert!(content.contains("\"Resident Alien\"...</i>"));
    assert!(content.contains("Hello, Harry."));
    assert!(content.contains("My name is Joseph."));
}

#[test]
fn test_import_with_semicolon_delimiter() {
    let temp_dir = TempDir::new().unwrap();

    let csv_content = "start_time;end_time;text
00:00:01,000;00:00:03,000;First subtitle
00:00:04,000;00:00:06,000;Second subtitle";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), ";");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Imported: 2 subtitles"));
}

// Note: Comma delimiter test is not included because SubRip format uses commas
// in timestamps (HH:MM:SS,mmm), which conflicts with CSV comma delimiter.
// Use pipe (|) or semicolon (;) as delimiters instead.

#[test]
fn test_import_to_existing_file_with_content() {
    let temp_dir = TempDir::new().unwrap();

    let existing_content = "1\n00:00:01,000 --> 00:00:02,000\nExisting first\n\n2\n00:00:03,000 --> 00:00:04,000\nExisting second\n";
    let srt_file = create_srt_with_content(&temp_dir, "existing.srt", existing_content);

    // Create existing SRT file with 2 subtitles

    // Import 2 new subtitles
    let csv_content = "start_time|end_time|text
00:00:05,000|00:00:06,000|New third
00:00:07,000|00:00:08,000|New fourth";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Imported: 2 subtitles"));
    assert!(stdout.contains("Index range: 3-4"));
    assert!(stdout.contains("Total subtitles: 4"));

    // Verify old subtitles unchanged and new ones added
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("Existing first"));
    assert!(content.contains("Existing second"));
    assert!(content.contains("New third"));
    assert!(content.contains("New fourth"));
}

// ========== Delimiter Validation Tests ==========

#[test]
fn test_import_invalid_delimiter_empty() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");
    let csv_file = create_temp_csv(&temp_dir, "test.csv", "header\ndata");

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("must be a single character"));
}

#[test]
fn test_import_invalid_delimiter_multiple_chars() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");
    let csv_file = create_temp_csv(&temp_dir, "test.csv", "header\ndata");

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "||");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("must be a single character"));
}

// ========== CSV Format Validation Tests ==========

#[test]
fn test_import_csv_invalid_header() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "wrong|header|format
00:00:01,000|00:00:03,000|Text";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Invalid CSV header"));
}

#[test]
fn test_import_csv_wrong_column_count() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "start_time|end_time|text
00:00:01,000|00:00:03,000"; // Missing text column

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.to_lowercase().contains("error"));
}

#[test]
fn test_import_csv_empty_file() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "start_time|end_time|text"; // Only header, no data

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("contains no data rows"));
}

// ========== Timestamp Validation Tests ==========

#[test]
fn test_import_invalid_timestamp_format() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "start_time|end_time|text
99:99:99,999|00:00:03,000|Invalid timestamp"; // Invalid timestamp

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("error"));
    assert!(stderr.contains("timestamp") || stderr.contains("line"));
}

#[test]
fn test_import_end_before_start() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "start_time|end_time|text
00:00:05,000|00:00:03,000|End before start"; // End < Start

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.to_lowercase().contains("error"));
}

#[test]
fn test_import_timestamp_conflict_with_existing() {
    let temp_dir = TempDir::new().unwrap();

    let existing_content = "1\n00:00:05,000 --> 00:00:10,000\nExisting subtitle\n";
    let srt_file = create_srt_with_content(&temp_dir, "test.srt", existing_content);

    // Create existing SRT file

    // Try to import subtitle that starts before existing ends
    let csv_content = "start_time|end_time|text
00:00:08,000|00:00:12,000|Overlapping subtitle"; // Starts at 8s, existing ends at 10s

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Timestamp conflict"));
    assert!(stderr.contains("00:00:10,000")); // Last ends at
    assert!(stderr.contains("00:00:08,000")); // New starts at
}

#[test]
fn test_import_timestamp_exactly_after_last() {
    let temp_dir = TempDir::new().unwrap();

    let existing_content = "1\n00:00:05,000 --> 00:00:10,000\nExisting subtitle\n";
    let srt_file = create_srt_with_content(&temp_dir, "test.srt", existing_content);

    // Create existing SRT file

    // Import subtitle that starts exactly when last ends
    let csv_content = "start_time|end_time|text
00:00:10,000|00:00:12,000|Exactly after";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("✓ Subtitles imported successfully"));
}

// ========== Backup Tests ==========

#[test]
fn test_import_creates_backup() {
    let temp_dir = TempDir::new().unwrap();

    let original_content = "1\n00:00:01,000 --> 00:00:02,000\nOriginal content\n";
    let srt_file = create_srt_with_content(&temp_dir, "test.srt", original_content);

    // Create existing SRT file

    // Import new data
    let csv_content = "start_time|end_time|text
00:00:03,000|00:00:04,000|New content";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Backup:"));

    // Backups are created in backups/ relative to current directory (project root)
    // Extract backup path from stdout
    let backup_line = stdout
        .lines()
        .find(|line| line.contains("Backup:"))
        .expect("Backup line not found");

    let backup_path_str = backup_line.replace("Backup:", "").trim().to_string();
    let backup_path = PathBuf::from(&backup_path_str);

    assert!(
        backup_path.exists(),
        "Backup file should exist at: {:?}",
        backup_path
    );

    // Verify backup contains original content (not new content)
    let backup_content = fs::read_to_string(&backup_path).unwrap();
    assert!(backup_content.contains("Original content"));
    assert!(!backup_content.contains("New content"));
}

#[test]
fn test_import_to_empty_srt_no_backup_error() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "empty.srt");

    // Create empty SRT file
    fs::write(&srt_file, "").unwrap();

    let csv_content = "start_time|end_time|text
00:00:01,000|00:00:03,000|First subtitle";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(
        output.status.success(),
        "Import to empty file should succeed"
    );
}

// ========== Path Traversal Tests ==========

#[test]
fn test_import_path_traversal_srt_rejected() {
    let temp_dir = TempDir::new().unwrap();
    let csv_file = create_temp_csv(&temp_dir, "test.csv", "start_time|end_time|text");

    let output = run_import_command("../../../etc/passwd", csv_file.to_str().unwrap(), "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("path traversal") || stderr.contains("failed to resolve"),
        "Should reject path traversal attempt. stderr was: {}",
        stderr
    );
}

#[test]
fn test_import_path_traversal_csv_rejected() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let output = run_import_command(srt_file.to_str().unwrap(), "../../../etc/passwd", "|");

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(
        stderr.contains("path traversal") || stderr.contains("failed to resolve"),
        "Should reject path traversal attempt. stderr was: {}",
        stderr
    );
}

// ========== File Handling Tests ==========

#[test]
fn test_import_csv_file_not_found() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let output = run_import_command(
        srt_file.to_str().unwrap(),
        "/tmp/nonexistent_csv_12345.csv",
        "|",
    );

    assert!(!output.status.success());

    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("failed to resolve csv file path"));
}

#[test]
fn test_import_multiline_text_conversion() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    // CSV with \n literal that should be converted to actual newlines
    let csv_content = "start_time|end_time|text
00:00:01,000|00:00:03,000|Line one\\nLine two\\nLine three";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    // Verify that \n was converted to actual newlines in SRT file
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("Line one\nLine two\nLine three"));
}

#[test]
fn test_import_preserves_html_tags() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "start_time|end_time|text
00:00:01,000|00:00:03,000|<i>Italic text</i>
00:00:04,000|00:00:06,000|<b>Bold text</b>";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    // Verify HTML tags are preserved
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("<i>Italic text</i>"));
    assert!(content.contains("<b>Bold text</b>"));
}

// ========== Output and Summary Tests ==========

#[test]
fn test_import_output_shows_summary() {
    let temp_dir = TempDir::new().unwrap();
    let srt_file = create_empty_srt(&temp_dir, "test.srt");

    let csv_content = "start_time|end_time|text
00:00:01,000|00:00:02,000|First
00:00:03,000|00:00:04,000|Second
00:00:05,000|00:00:06,000|Third";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("✓ Subtitles imported successfully"));
    assert!(stdout.contains("Imported: 3 subtitles"));
    assert!(stdout.contains("Index range: 1-3"));
    assert!(stdout.contains("Total subtitles: 3"));
}

#[test]
fn test_import_index_range_with_existing_gaps() {
    let temp_dir = TempDir::new().unwrap();

    let existing_content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond\n\n5\n00:00:05,000 --> 00:00:06,000\nFifth\n";
    let srt_file = create_srt_with_content(&temp_dir, "test.srt", existing_content);

    // Create SRT with gaps in indices (1, 2, 5)

    // Import 2 new subtitles
    let csv_content = "start_time|end_time|text
00:00:07,000|00:00:08,000|Sixth
00:00:09,000|00:00:10,000|Seventh";

    let csv_file = create_temp_csv(&temp_dir, "test.csv", csv_content);

    let output = run_import_command(srt_file.to_str().unwrap(), csv_file.to_str().unwrap(), "|");

    assert!(output.status.success());

    let stdout = String::from_utf8_lossy(&output.stdout);
    // New indices should be max(1,2,5) + 1 = 6, and max + 2 = 7
    assert!(stdout.contains("Index range: 6-7"));
    assert!(stdout.contains("Imported: 2 subtitles"));

    // Verify new subtitles have correct indices
    let content = fs::read_to_string(&srt_file).unwrap();
    assert!(content.contains("6\n"));
    assert!(content.contains("Sixth"));
    assert!(content.contains("7\n"));
    assert!(content.contains("Seventh"));
}
