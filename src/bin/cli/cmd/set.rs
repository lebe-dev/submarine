use lib::subtitle::model::{
    Subtitle, SubtitleError, SubtitleText, SubtitleTimestamp, SubtitleUpdate,
};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

pub fn handle(
    file: &str,
    index: u32,
    start: Option<String>,
    end: Option<String>,
    text: Option<String>,
) -> anyhow::Result<()> {
    info!("setting subtitle {} in file: {}", index, file);

    // 1. Validate at least one option is provided (early check for better UX)
    if start.is_none() && end.is_none() && text.is_none() {
        error!("no fields specified for update");
        eprintln!("error: At least one of --start, --end, or --text must be specified");
        std::process::exit(1);
    }

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

    let start_timestamp = if let Some(start_str) = start {
        debug!("parsing start timestamp: {}", start_str);
        let duration = Subtitle::parse_timestamp(&start_str)
            .map_err(|e| anyhow::anyhow!("invalid start timestamp: {}", e))?;
        Some(
            SubtitleTimestamp::try_new(duration)
                .map_err(|e| anyhow::anyhow!("invalid start timestamp value: {}", e))?,
        )
    } else {
        None
    };

    let end_timestamp = if let Some(end_str) = end {
        debug!("parsing end timestamp: {}", end_str);
        let duration = Subtitle::parse_timestamp(&end_str)
            .map_err(|e| anyhow::anyhow!("invalid end timestamp: {}", e))?;
        Some(
            SubtitleTimestamp::try_new(duration)
                .map_err(|e| anyhow::anyhow!("invalid end timestamp value: {}", e))?,
        )
    } else {
        None
    };

    // 4. Parse and validate text option if provided
    let subtitle_text = if let Some(text_str) = text {
        debug!("validating text (length: {})", text_str.len());
        Some(SubtitleText::try_new(text_str).map_err(|e| anyhow::anyhow!("invalid text: {}", e))?)
    } else {
        None
    };

    // 5. Build update struct
    let update = SubtitleUpdate {
        start_time: start_timestamp,
        end_time: end_timestamp,
        text: subtitle_text,
    };

    // 6. Create service and execute update
    let service = SubRipService::new(base_dir);

    debug!("updating subtitle {}...", index);
    match service.set(&filename, index, update) {
        Ok(report) => {
            info!("subtitle {} updated successfully", index);
            debug!("backup created: {}", report.backup_path);

            println!("✓ Subtitle {} updated successfully", index);
            println!();
            println!("Backup created: {}", report.backup_path);
            println!("Fields updated: {}", report.fields_updated.join(", "));

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
                SubtitleError::SubtitleNotFound(idx) => {
                    error!("subtitle {} not found", idx);
                    eprintln!("error: Subtitle with index {} not found in file", idx);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: Failed to parse subtitle file: {}", err);
                    eprintln!("hint: Try running 'sm doctor --fix {}' first", file);
                }
                SubtitleError::NoFieldsToUpdate => {
                    error!("no fields to update");
                    eprintln!("error: At least one of --start, --end, or --text must be specified");
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
            }
            std::process::exit(1);
        }
    }
}
