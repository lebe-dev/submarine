use anyhow::{Context, Result, bail};
use chrono::Duration;
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::subtitle::model::{Subtitle, SubtitleError, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

/// Parse offset string in format "+100" or "-500" to milliseconds
fn parse_offset(offset_str: &str) -> Result<i64> {
    debug!("parsing offset string: {}", offset_str);

    if !offset_str.starts_with('+') && !offset_str.starts_with('-') {
        bail!("offset must start with '+' or '-' (e.g., '+100', '-500')");
    }

    let millis: i64 = offset_str
        .parse()
        .context("failed to parse offset as number")?;

    debug!("parsed offset: {} milliseconds", millis);
    Ok(millis)
}

/// Apply time offset to a subtitle, returning a new subtitle with adjusted timestamps
fn apply_offset(subtitle: &Subtitle, offset: Duration) -> Result<Subtitle> {
    let new_start = *subtitle.start_time.as_ref() + offset;
    let new_end = *subtitle.end_time.as_ref() + offset;

    debug!(
        "checking for negative timestamps: start={}, end={}",
        new_start.num_milliseconds(),
        new_end.num_milliseconds()
    );
    if new_start < Duration::zero() || new_end < Duration::zero() {
        error!(
            "offset would result in negative timestamp for subtitle {}",
            subtitle.index.as_ref()
        );
        bail!(
            "offset would result in negative timestamp for subtitle {}",
            subtitle.index.as_ref()
        );
    }

    let new_start_ts = SubtitleTimestamp::try_new(new_start).context(format!(
        "failed to create start timestamp for subtitle {}",
        subtitle.index.as_ref()
    ))?;

    let new_end_ts = SubtitleTimestamp::try_new(new_end).context(format!(
        "failed to create end timestamp for subtitle {}",
        subtitle.index.as_ref()
    ))?;

    Subtitle::new(
        subtitle.index,
        new_start_ts,
        new_end_ts,
        subtitle.text.clone(),
    )
    .context(format!(
        "failed to create subtitle {} with new timestamps",
        subtitle.index.as_ref()
    ))
}

pub fn handle(file: &str, offset_str: &str) -> Result<()> {
    info!("applying time offset to file: {}", file);
    debug!("offset string: {}", offset_str);

    debug!("parsing offset");
    let offset_millis = parse_offset(offset_str)?;
    let offset = Duration::milliseconds(offset_millis);
    info!("parsed offset: {} milliseconds", offset_millis);

    debug!("validating and resolving file path");
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

    debug!("creating backup");
    let backup_service = SubRipBackupService::new();
    let backup_result = backup_service.create_backup(&canonical_path);

    let backup_path = match backup_result {
        Ok(Some(path)) => {
            debug!("backup created: {}", path);
            path
        }
        Ok(None) => {
            error!("file does not exist: {}", file);
            eprintln!("error: File does not exist: {}", file);
            std::process::exit(1);
        }
        Err(e) => {
            error!("failed to create backup: {}", e);
            eprintln!("error: Failed to create backup: {}", e);
            std::process::exit(1);
        }
    };

    debug!("loading all subtitles");
    let service = SubRipService::new(base_dir);
    debug!("loading subtitles from file...");

    let subtitles = match service.get_all(&filename) {
        Ok(subs) => {
            info!("loaded {} subtitles", subs.len());
            subs
        }
        Err(e) => {
            error!("failed to load subtitles: {}", e);
            handle_subtitle_error(e, file);
            std::process::exit(1);
        }
    };

    if subtitles.is_empty() {
        info!("no subtitles found in file");
        println!("No subtitles found in file");
        return Ok(());
    }

    info!("applying offset to {} subtitles", subtitles.len());
    debug!("applying offset to {} subtitles...", subtitles.len());
    let mut adjusted_subtitles = Vec::with_capacity(subtitles.len());

    for subtitle in &subtitles {
        debug!(
            "processing subtitle {}: {} --> {}",
            subtitle.index.as_ref(),
            Subtitle::format_timestamp(subtitle.start_time.as_ref()),
            Subtitle::format_timestamp(subtitle.end_time.as_ref())
        );

        match apply_offset(subtitle, offset) {
            Ok(adjusted) => {
                debug!(
                    "adjusted to: {} --> {}",
                    Subtitle::format_timestamp(adjusted.start_time.as_ref()),
                    Subtitle::format_timestamp(adjusted.end_time.as_ref())
                );
                adjusted_subtitles.push(adjusted);
            }
            Err(e) => {
                error!("failed to apply offset: {}", e);
                eprintln!("error: {}", e);
                std::process::exit(1);
            }
        }
    }

    info!("writing adjusted subtitles back to file");
    debug!(
        "writing {} adjusted subtitles to file...",
        adjusted_subtitles.len()
    );
    match service.write_all(&filename, &adjusted_subtitles) {
        Ok(_) => {
            info!(
                "successfully applied offset to {} subtitles",
                adjusted_subtitles.len()
            );

            println!("✓ Time offset applied successfully");
            println!();
            println!("Backup: {}", backup_path);
            println!("Offset: {} ms", offset_millis);
            println!("Subtitles adjusted: {}", adjusted_subtitles.len());

            Ok(())
        }
        Err(e) => {
            error!("failed to write adjusted subtitles: {}", e);
            handle_subtitle_error(e, file);
            std::process::exit(1);
        }
    }
}

/// Handle SubtitleError variants with appropriate error messages
fn handle_subtitle_error(error: SubtitleError, file: &str) {
    match error {
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
        SubtitleError::WriteFailed(msg) => {
            error!("write failed: {}", msg);
            eprintln!("error: Failed to write updated file: {}", msg);
        }
        SubtitleError::IoError(err) => {
            error!("i/o error: {}", err);
            eprintln!("error: I/O error: {}", err);
        }
        _ => {
            error!("unexpected error: {}", error);
            eprintln!("error: {}", error);
        }
    }
}
