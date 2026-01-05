use std::fs;
use std::path::PathBuf;
use std::process::Command;
use tempfile::TempDir;

fn run_add_command(file: &str, timestamps: &str, text: &str) -> std::process::Output {
    Command::new("cargo")
        .args(&["run", "--bin", "sm", "--", "add", file, timestamps, text])
        .output()
        .expect("Failed to execute command")
}

#[test]
fn test_add_to_existing_file() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:03,000-00:00:05,000",
        "Second subtitle",
    );

    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("✓ Subtitle added successfully"));
    assert!(stdout.contains("New index: 2"));
    assert!(stdout.contains("Total subtitles: 2"));

    // Verify file content
    let content = fs::read_to_string(&test_file).unwrap();
    assert!(content.contains("2\n"));
    assert!(content.contains("00:00:03,000 --> 00:00:05,000"));
    assert!(content.contains("Second subtitle"));
}

#[test]
fn test_add_to_empty_file() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("empty.srt");

    fs::write(&test_file, "").unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:01,000-00:00:03,000",
        "First subtitle",
    );

    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("New index: 1"));
    assert!(stdout.contains("Total subtitles: 1"));

    // Verify file content
    let content = fs::read_to_string(&test_file).unwrap();
    assert!(content.contains("1\n"));
    assert!(content.contains("00:00:01,000 --> 00:00:03,000"));
    assert!(content.contains("First subtitle"));
}

#[test]
fn test_add_with_multiline_text() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:03,000-00:00:05,000",
        "Line 1\nLine 2",
    );

    assert!(output.status.success());

    // Verify file content contains multiline text
    let content = fs::read_to_string(&test_file).unwrap();
    assert!(content.contains("Line 1\nLine 2"));
}

#[test]
fn test_add_with_html_tags() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:03,000-00:00:05,000",
        "<i>Italic text</i>",
    );

    assert!(output.status.success());

    // Verify file content contains HTML tags
    let content = fs::read_to_string(&test_file).unwrap();
    assert!(content.contains("<i>Italic text</i>"));
}

#[test]
fn test_add_creates_new_file() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("new_file.srt");

    // File doesn't exist yet - verify
    assert!(!test_file.exists());

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:01,000-00:00:03,000",
        "First subtitle in new file",
    );

    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("✓ Subtitle added successfully"));
    assert!(stdout.contains("New index: 1"));
    assert!(stdout.contains("Total subtitles: 1"));
    assert!(stdout.contains("N/A (new file)"));

    // Verify file was created
    assert!(test_file.exists());

    // Verify file content
    let content = fs::read_to_string(&test_file).unwrap();
    assert!(content.contains("1\n"));
    assert!(content.contains("00:00:01,000 --> 00:00:03,000"));
    assert!(content.contains("First subtitle in new file"));
}

#[test]
fn test_add_invalid_timestamp_format() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    // Missing hyphen separator
    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:01,000 00:00:03,000",
        "Text",
    );

    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Invalid timestamp format"));
}

#[test]
fn test_add_end_before_start() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    // End timestamp before start timestamp
    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:05,000-00:00:03,000",
        "Text",
    );

    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("error"));
}

#[test]
fn test_add_backup_created() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:03,000-00:00:05,000",
        "Second",
    );

    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);

    // Extract backup path from stdout
    let backup_line = stdout
        .lines()
        .find(|line| line.contains("Backup created:"))
        .expect("Backup line not found");

    // Backup file should exist
    let backup_path_str = backup_line.replace("Backup created:", "");
    let backup_path_trimmed = backup_path_str.trim();
    let backup_path = PathBuf::from(backup_path_trimmed);
    assert!(
        backup_path.exists(),
        "Backup file should exist at: {:?}",
        backup_path
    );

    // Backup should contain original content
    let backup_content = fs::read_to_string(&backup_path).unwrap();
    assert!(backup_content.contains("First"));
    assert!(!backup_content.contains("Second"));
}

#[test]
fn test_add_with_gap_in_indices() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content =
        "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n5\n00:00:03,000 --> 00:00:04,000\nFifth\n";
    fs::write(&test_file, content).unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:05,000-00:00:06,000",
        "Sixth",
    );

    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("New index: 6")); // max(1,5) + 1 = 6
    assert!(stdout.contains("Total subtitles: 3"));

    // Verify file content
    let content = fs::read_to_string(&test_file).unwrap();
    assert!(content.contains("6\n"));
    assert!(content.contains("Sixth"));
}

#[test]
fn test_add_malformed_file() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("bad.srt");

    fs::write(&test_file, "INVALID CONTENT\nNOT SRT FORMAT").unwrap();

    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:01,000-00:00:03,000",
        "Text",
    );

    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Failed to parse subtitle file"));
    assert!(stderr.contains("sm doctor --fix"));
}

#[test]
fn test_add_timestamp_before_last_subtitle() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content =
        "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond\n";
    fs::write(&test_file, content).unwrap();

    // Try to add subtitle that starts before the last one ends (at 00:00:07,000)
    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:06,000-00:00:08,000", // starts at 00:00:06,000, before last ends
        "Too early",
    );

    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Timestamp conflict"));
    assert!(stderr.contains("00:00:07,000")); // last ends at
    assert!(stderr.contains("00:00:06,000")); // new starts at
}

#[test]
fn test_add_timestamp_exactly_after_last() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content =
        "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond\n";
    fs::write(&test_file, content).unwrap();

    // Add subtitle that starts exactly when the last one ends (at 00:00:07,000)
    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:07,000-00:00:09,000",
        "Exactly after",
    );

    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("✓ Subtitle added successfully"));
    assert!(stdout.contains("New index: 3"));
}

#[test]
fn test_add_timestamp_overlapping() {
    let temp_dir = TempDir::new().unwrap();
    let test_file = temp_dir.path().join("test.srt");

    let content = "1\n00:00:10,000 --> 00:00:15,000\nFirst\n";
    fs::write(&test_file, content).unwrap();

    // Try to add subtitle that overlaps with the last one
    let output = run_add_command(
        test_file.to_str().unwrap(),
        "00:00:14,000-00:00:16,000", // starts at 14s, last ends at 15s
        "Overlapping",
    );

    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("Timestamp conflict"));
}
