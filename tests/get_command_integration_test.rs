use std::path::PathBuf;
use std::process::Command;

// Helper to get project root
fn get_project_root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

// Helper to run CLI command
fn run_get_command(file: &str, index: u32) -> std::process::Output {
    Command::new("cargo")
        .args(&["run", "--bin", "sm", "--", "get", file, &index.to_string()])
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
    let output = run_get_command(&file, 1);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("1\n"));
    assert!(stdout.contains("00:00:01,000 --> 00:00:03,000"));
    assert!(stdout.contains("First subtitle"));

    // Test getting subtitle index 2
    let output = run_get_command(&file, 2);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("00:00:03,500 --> 00:00:05,500"));
    assert!(stdout.contains("Second subtitle"));

    // Test getting subtitle index 3
    let output = run_get_command(&file, 3);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("3\n"));
    assert!(stdout.contains("00:00:06,000 --> 00:00:08,000"));
    assert!(stdout.contains("Third subtitle"));
}

#[test]
fn test_get_subtitle_with_html() {
    let file = test_data_path("valid/complex.srt");

    let output = run_get_command(&file, 1);
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
    let output = run_get_command(&file, 5);
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
    let output = run_get_command(&file, 1);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("1\n"));
    assert!(stdout.contains("00:00:01,436 --> 00:00:03,481"));

    // Test getting subtitle index 2
    let output = run_get_command(&file, 2);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("2\n"));
    assert!(stdout.contains("00:00:03,481 --> 00:00:05,135"));
}

#[test]
fn test_get_edge_case_indices() {
    let file = test_data_path("valid/simple.srt");

    // Test first subtitle (index 1)
    let output = run_get_command(&file, 1);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("First subtitle"));

    // Test last subtitle (index 3 in simple.srt)
    let output = run_get_command(&file, 3);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Third subtitle"));
}

// ========== Negative Test Cases ==========

#[test]
fn test_get_nonexistent_index() {
    let file = test_data_path("valid/simple.srt");

    let output = run_get_command(&file, 99999);
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("not found"));
    assert!(stderr.contains("99999"));
}

#[test]
fn test_get_file_not_found() {
    let output = run_get_command("nonexistent.srt", 1);
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

    let output = run_get_command(&file, 1);
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr).to_lowercase();
    assert!(stderr.contains("error"));
    assert!(stderr.contains("parse") || stderr.contains("invalid"));
}

#[test]
fn test_get_empty_file() {
    let file = test_data_path("invalid/empty.srt");

    let output = run_get_command(&file, 1);
    assert!(!output.status.success());
    let stderr = String::from_utf8_lossy(&output.stderr);
    assert!(stderr.contains("not found"));
}

#[test]
fn test_get_path_traversal() {
    let output = run_get_command("../../../etc/passwd", 1);
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
    let output = run_get_command(&file, 0);
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
    let output = run_get_command(temp_file.to_str().unwrap(), 1);
    assert!(output.status.success());
    let stdout = String::from_utf8_lossy(&output.stdout);
    assert!(stdout.contains("Test subtitle"));
}
