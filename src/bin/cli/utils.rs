use lib::doctor::model::DoctorError;
use lib::subtitle::model::{Subtitle, SubtitleError};
use log::{debug, error};
use std::path::{Path, PathBuf};

// -- Path resolution --

/// Resolved file path with components needed by command handlers.
pub struct ResolvedPath {
    /// Full canonical (or resolved) path to the file.
    pub full_path: PathBuf,
    /// Parent directory of the file.
    pub base_dir: PathBuf,
    /// File name component.
    pub filename: String,
}

/// Resolve a file path that must already exist, with path traversal protection.
pub fn resolve_existing_path(file: &str) -> anyhow::Result<ResolvedPath> {
    let file_path = Path::new(file);
    debug!("resolving existing path: {:?}", file_path);

    if file_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;

        let resolved = current_dir.join(file_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", file_path);
            return Err(anyhow::anyhow!(
                "invalid file path: path traversal not allowed"
            ));
        }
    }

    let canonical_path = file_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("failed to resolve file path: {}", e))?;
    debug!("canonical path: {:?}", canonical_path);

    let base_dir = canonical_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid file path"))?
        .to_path_buf();
    debug!("base directory: {:?}", base_dir);

    let filename = canonical_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("invalid file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("invalid UTF-8 in filename"))?
        .to_string();
    debug!("filename: {}", filename);

    Ok(ResolvedPath {
        full_path: canonical_path,
        base_dir,
        filename,
    })
}

/// Resolve a file path that may not exist yet, with path traversal protection.
pub fn resolve_new_path(file: &str) -> anyhow::Result<ResolvedPath> {
    let file_path = Path::new(file);
    debug!("resolving new path: {:?}", file_path);

    let resolved_path = if file_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;
        current_dir.join(file_path)
    } else {
        file_path.to_path_buf()
    };

    if resolved_path
        .components()
        .any(|c| matches!(c, std::path::Component::ParentDir))
    {
        error!("path traversal attempt detected: {:?}", file_path);
        return Err(anyhow::anyhow!(
            "invalid file path: path traversal not allowed"
        ));
    }

    let base_dir = resolved_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid file path"))?
        .to_path_buf();
    debug!("base directory: {:?}", base_dir);

    let filename = resolved_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("invalid file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("invalid UTF-8 in filename"))?
        .to_string();
    debug!("filename: {}", filename);

    Ok(ResolvedPath {
        full_path: resolved_path,
        base_dir,
        filename,
    })
}

// -- Error formatting --

/// Structured error with code, message, and optional hint.
pub struct CliError {
    pub code: String,
    pub message: String,
    pub hint: Option<String>,
}

/// Convert a SubtitleError into a structured CliError.
pub fn format_subtitle_error(e: &SubtitleError, file: &str) -> CliError {
    match e {
        SubtitleError::FileNotFound(path) => CliError {
            code: "file_not_found".into(),
            message: format!("File not found: {}", path),
            hint: None,
        },
        SubtitleError::InvalidPath(msg) => CliError {
            code: "invalid_path".into(),
            message: format!("Invalid file path: {}", msg),
            hint: None,
        },
        SubtitleError::ParseError(err) => CliError {
            code: "parse_error".into(),
            message: format!("Failed to parse subtitle file: {}", err),
            hint: Some(format!("Try running 'sm doctor --fix {}' first", file)),
        },
        SubtitleError::IoError(err) => CliError {
            code: "io_error".into(),
            message: format!("I/O error: {}", err),
            hint: None,
        },
        SubtitleError::SubtitleNotFound(idx) => CliError {
            code: "subtitle_not_found".into(),
            message: format!("Subtitle with index {} not found in file", idx),
            hint: None,
        },
        SubtitleError::NoFieldsToUpdate => CliError {
            code: "no_fields_to_update".into(),
            message: "At least one of --start, --end, or --text must be specified".into(),
            hint: None,
        },
        SubtitleError::BackupFailed(msg) => CliError {
            code: "backup_failed".into(),
            message: format!("Failed to create backup: {}", msg),
            hint: None,
        },
        SubtitleError::WriteFailed(msg) => CliError {
            code: "write_failed".into(),
            message: format!("Failed to write updated file: {}", msg),
            hint: None,
        },
        SubtitleError::TimestampConflict {
            last_end,
            new_start,
        } => CliError {
            code: "timestamp_conflict".into(),
            message: format!(
                "Timestamp conflict: last subtitle ends at {}, new starts at {}",
                last_end, new_start
            ),
            hint: Some("New subtitle must start at or after the last subtitle ends".into()),
        },
        SubtitleError::CsvParseError { line, message } => CliError {
            code: "csv_parse_error".into(),
            message: format!("CSV parsing failed at line {}: {}", line, message),
            hint: Some(format!("Check the CSV file format at line {}", line)),
        },
        SubtitleError::InvalidCsvHeader(delimiter, actual) => CliError {
            code: "invalid_csv_header".into(),
            message: format!(
                "Invalid CSV header. Expected: start_time{}end_time{}text, got: {}",
                delimiter, delimiter, actual
            ),
            hint: None,
        },
        SubtitleError::TimestampOverlap {
            line,
            existing_end,
            new_start,
        } => CliError {
            code: "timestamp_overlap".into(),
            message: format!(
                "Timestamp overlap at line {}: existing ends at {}, new starts at {}",
                line, existing_end, new_start
            ),
            hint: None,
        },
        SubtitleError::AnchoredParseError { line, message } => CliError {
            code: "anchored_parse_error".into(),
            message: format!(
                "Anchored format parsing failed at line {}: {}",
                line, message
            ),
            hint: Some("Ensure format is [INDEX] TEXT with proper structure".into()),
        },
        SubtitleError::ReferenceIndexNotFound { index } => CliError {
            code: "reference_index_not_found".into(),
            message: format!(
                "Reference file does not contain subtitle with index {}",
                index
            ),
            hint: Some("Check that the reference file contains all required indices".into()),
        },
        SubtitleError::InvalidAnchoredFormat { line, message } => CliError {
            code: "invalid_anchored_format".into(),
            message: format!("Invalid anchored format at line {}: {}", line, message),
            hint: None,
        },
    }
}

/// Convert a DoctorError into a structured CliError.
pub fn format_doctor_error(e: &DoctorError) -> CliError {
    match e {
        DoctorError::FileNotFound(path) => CliError {
            code: "file_not_found".into(),
            message: format!("File not found: {}", path),
            hint: None,
        },
        DoctorError::InvalidPath(msg) => CliError {
            code: "invalid_path".into(),
            message: format!("Invalid file path: {}", msg),
            hint: None,
        },
        DoctorError::IoError(err) => CliError {
            code: "io_error".into(),
            message: format!("I/O error: {}", err),
            hint: None,
        },
        DoctorError::BackupFailed(msg) => CliError {
            code: "backup_failed".into(),
            message: format!("Backup creation failed: {}", msg),
            hint: None,
        },
        DoctorError::ValidationFailed(msg) => CliError {
            code: "validation_failed".into(),
            message: format!("Validation failed: {}", msg),
            hint: None,
        },
    }
}

// -- Range parsing --

/// Parse range string in format "START-END" into (start, end) tuple.
pub fn parse_range(range: &str) -> anyhow::Result<(u32, u32)> {
    let parts: Vec<&str> = range.split('-').collect();

    if parts.len() != 2 {
        return Err(anyhow::anyhow!(
            "Invalid range format '{}'. Expected format: START-END (e.g., 1-50)",
            range
        ));
    }

    let start = parts[0].trim().parse::<u32>().map_err(|_| {
        anyhow::anyhow!(
            "Invalid start index '{}'. Must be a positive number",
            parts[0]
        )
    })?;

    let end = parts[1].trim().parse::<u32>().map_err(|_| {
        anyhow::anyhow!(
            "Invalid end index '{}'. Must be a positive number",
            parts[1]
        )
    })?;

    if start < 1 {
        return Err(anyhow::anyhow!("Start index must be >= 1, got {}", start));
    }

    if end < 1 {
        return Err(anyhow::anyhow!("End index must be >= 1, got {}", end));
    }

    if start > end {
        return Err(anyhow::anyhow!(
            "Start index must be <= end index (got {} > {})",
            start,
            end
        ));
    }

    Ok((start, end))
}

/// Validate that the requested range is within available subtitle indices.
pub fn validate_range_boundaries(
    start: u32,
    end: u32,
    subtitles: &[Subtitle],
) -> anyhow::Result<()> {
    if subtitles.is_empty() {
        return Err(anyhow::anyhow!("File contains no subtitles"));
    }

    let min_index = subtitles.iter().map(|s| *s.index.as_ref()).min().unwrap();
    let max_index = subtitles.iter().map(|s| *s.index.as_ref()).max().unwrap();

    debug!(
        "file contains indices {}-{}, requested range {}-{}",
        min_index, max_index, start, end
    );

    if start > max_index {
        return Err(anyhow::anyhow!(
            "Start index {} is beyond the last subtitle (index {})",
            start,
            max_index
        ));
    }

    if end < min_index {
        return Err(anyhow::anyhow!(
            "End index {} is before the first subtitle (index {})",
            end,
            min_index
        ));
    }

    Ok(())
}

// -- Input validation --

/// Reject control characters in user-provided text input.
/// Allows \n, \r, \t but rejects all other ASCII control characters (< 0x20).
pub fn reject_control_chars(input: &str, field: &str) -> anyhow::Result<()> {
    for (i, ch) in input.chars().enumerate() {
        if ch.is_ascii_control() && ch != '\n' && ch != '\r' && ch != '\t' {
            return Err(anyhow::anyhow!(
                "Invalid character in {}: control character (0x{:02x}) at position {}",
                field,
                ch as u32,
                i
            ));
        }
    }
    Ok(())
}

/// Reject percent-encoded sequences in filenames to prevent double-encoding.
pub fn reject_percent_encoding(input: &str, field: &str) -> anyhow::Result<()> {
    let bytes = input.as_bytes();
    for i in 0..bytes.len() {
        if bytes[i] == b'%' && i + 2 < bytes.len() {
            let hex1 = bytes[i + 1];
            let hex2 = bytes[i + 2];
            if hex1.is_ascii_hexdigit() && hex2.is_ascii_hexdigit() {
                return Err(anyhow::anyhow!(
                    "Invalid character in {}: percent-encoded sequence '%{}{}' detected. Use plain filenames",
                    field,
                    hex1 as char,
                    hex2 as char
                ));
            }
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_range_valid() {
        let (start, end) = parse_range("1-50").unwrap();
        assert_eq!(start, 1);
        assert_eq!(end, 50);
    }

    #[test]
    fn test_parse_range_single() {
        let (start, end) = parse_range("42-42").unwrap();
        assert_eq!(start, 42);
        assert_eq!(end, 42);
    }

    #[test]
    fn test_parse_range_with_spaces() {
        let (start, end) = parse_range("10 - 20").unwrap();
        assert_eq!(start, 10);
        assert_eq!(end, 20);
    }

    #[test]
    fn test_parse_range_invalid_format() {
        assert!(parse_range("invalid").is_err());
        assert!(parse_range("1-2-3").is_err());
    }

    #[test]
    fn test_parse_range_invalid_numbers() {
        assert!(parse_range("abc-50").is_err());
        assert!(parse_range("1-xyz").is_err());
    }

    #[test]
    fn test_parse_range_reversed() {
        assert!(parse_range("50-1").is_err());
    }

    #[test]
    fn test_reject_control_chars_valid() {
        assert!(reject_control_chars("hello world", "text").is_ok());
        assert!(reject_control_chars("line1\nline2", "text").is_ok());
        assert!(reject_control_chars("tab\there", "text").is_ok());
    }

    #[test]
    fn test_reject_control_chars_invalid() {
        assert!(reject_control_chars("hello\x00world", "text").is_err());
        assert!(reject_control_chars("hello\x01world", "text").is_err());
        assert!(reject_control_chars("hello\x1Fworld", "text").is_err());
    }

    #[test]
    fn test_reject_percent_encoding_valid() {
        assert!(reject_percent_encoding("simple.srt", "file").is_ok());
        assert!(reject_percent_encoding("file%name.srt", "file").is_ok()); // % not followed by two hex digits
    }

    #[test]
    fn test_reject_percent_encoding_invalid() {
        assert!(reject_percent_encoding("file%20name.srt", "file").is_err());
        assert!(reject_percent_encoding("%2F..%2Fetc", "file").is_err());
    }
}
