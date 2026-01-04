use lib::subtitle::model::{Subtitle, SubtitleError, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

pub fn handle(file: &str, timestamps: &str, text: &str) -> anyhow::Result<()> {
    info!("adding subtitle to file: {}", file);

    // 1. Parse and validate file path with path traversal protection
    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

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

    // 2. Parse timestamps: split by "-" (hyphen)
    debug!("parsing timestamps: {}", timestamps);
    let parts: Vec<&str> = timestamps.splitn(2, '-').collect();

    if parts.len() != 2 {
        error!("invalid timestamp format: {}", timestamps);
        eprintln!("error: Invalid timestamp format (expected 'HH:MM:SS,mmm-HH:MM:SS,mmm')");
        eprintln!("example: \"00:00:10,000-00:00:12,500\"");
        std::process::exit(1);
    }

    let start_str = parts[0];
    let end_str = parts[1];
    debug!("start: {}, end: {}", start_str, end_str);

    // 3. Parse start timestamp
    debug!("parsing start timestamp: {}", start_str);
    let start_duration = Subtitle::parse_timestamp(start_str)
        .map_err(|e| anyhow::anyhow!("invalid start timestamp: {}", e))?;
    let start_timestamp = SubtitleTimestamp::try_new(start_duration)
        .map_err(|e| anyhow::anyhow!("invalid start timestamp value: {}", e))?;

    // 4. Parse end timestamp
    debug!("parsing end timestamp: {}", end_str);
    let end_duration = Subtitle::parse_timestamp(end_str)
        .map_err(|e| anyhow::anyhow!("invalid end timestamp: {}", e))?;
    let end_timestamp = SubtitleTimestamp::try_new(end_duration)
        .map_err(|e| anyhow::anyhow!("invalid end timestamp value: {}", e))?;

    // 5. Validate and sanitize text
    debug!("validating text (length: {})", text.len());
    let subtitle_text = SubtitleText::try_new(text.to_string())
        .map_err(|e| anyhow::anyhow!("invalid text: {}", e))?;

    // 6. Create service and execute add
    let service = SubRipService::new(base_dir);

    debug!("adding new subtitle...");
    match service.add(&filename, start_timestamp, end_timestamp, subtitle_text) {
        Ok(report) => {
            info!(
                "subtitle added successfully with index {}",
                report.new_index
            );
            debug!("backup created: {}", report.backup_path);

            println!("✓ Subtitle added successfully");
            println!();
            println!("New index: {}", report.new_index);
            println!("Total subtitles: {}", report.total_subtitles);
            println!("Backup created: {}", report.backup_path);

            Ok(())
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    error!("file not found: {}", path);
                    eprintln!("error: File not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: Invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: Failed to parse subtitle file: {}", err);
                    eprintln!("hint: Try running 'sm doctor --fix {}' first", file);
                }
                SubtitleError::BackupFailed(msg) => {
                    error!("backup failed: {}", msg);
                    eprintln!("error: Failed to create backup: {}", msg);
                }
                SubtitleError::WriteFailed(msg) => {
                    error!("write failed: {}", msg);
                    eprintln!("error: Failed to write updated file: {}", msg);
                }
                SubtitleError::IoError(err) => {
                    error!("i/o error: {}", err);
                    eprintln!("error: I/O error: {}", err);
                }
                SubtitleError::TimestampConflict {
                    last_end,
                    new_start,
                } => {
                    error!(
                        "timestamp conflict: last ends at {}, new starts at {}",
                        last_end, new_start
                    );
                    eprintln!("error: Timestamp conflict");
                    eprintln!("  Last subtitle ends at: {}", last_end);
                    eprintln!("  New subtitle starts at: {}", new_start);
                    eprintln!("  New subtitle must start at or after the last subtitle ends");
                }
                _ => {
                    error!("unexpected error: {}", e);
                    eprintln!("error: {}", e);
                }
            }
            std::process::exit(1);
        }
    }
}
