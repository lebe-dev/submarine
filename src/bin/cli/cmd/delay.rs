use crate::cli::OutputFormat;
use crate::dto::{DelayResultDto, SubtitleDto};
use crate::json_output::{output_error, output_success};
use crate::utils;
use anyhow::{Context, Result, bail};
use chrono::Duration;
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::subtitle::model::{Subtitle, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};

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

    if new_start < Duration::zero() || new_end < Duration::zero() {
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

pub fn handle(file: &str, offset_str: &str, dry_run: bool, format: &OutputFormat) -> Result<()> {
    info!("applying time offset to file: {}", file);

    let offset_millis = parse_offset(offset_str)?;
    let offset = Duration::milliseconds(offset_millis);
    info!("parsed offset: {} milliseconds", offset_millis);

    let resolved = utils::resolve_existing_path(file)?;

    // Load subtitles first (for both dry-run and real mode)
    let service = SubRipService::new(resolved.base_dir.clone());

    let subtitles = match service.get_all(&resolved.filename) {
        Ok(subs) => {
            info!("loaded {} subtitles", subs.len());
            subs
        }
        Err(e) => {
            let cli_err = utils::format_subtitle_error(&e, file);
            output_error(
                format,
                &cli_err.code,
                &cli_err.message,
                cli_err.hint.as_deref(),
            );
            return Err(anyhow::anyhow!(""));
        }
    };

    if subtitles.is_empty() {
        info!("no subtitles found in file");
        let dto = DelayResultDto {
            offset_ms: offset_millis,
            subtitles_adjusted: 0,
            backup_path: "N/A".into(),
            dry_run,
            sample_before: None,
            sample_after: None,
        };
        output_success(format, &dto, || {
            println!("No subtitles found in file");
        });
        return Ok(());
    }

    // Apply offset to all subtitles
    let mut adjusted_subtitles = Vec::with_capacity(subtitles.len());
    for subtitle in &subtitles {
        match apply_offset(subtitle, offset) {
            Ok(adjusted) => {
                adjusted_subtitles.push(adjusted);
            }
            Err(e) => {
                output_error(format, "offset_error", &format!("{}", e), None);
                return Err(anyhow::anyhow!(""));
            }
        }
    }

    if dry_run {
        info!("dry-run mode, previewing delay");

        // Show sample: first 3 subtitles before/after
        let sample_count = 3.min(subtitles.len());
        let sample_before: Vec<SubtitleDto> = subtitles[..sample_count]
            .iter()
            .map(SubtitleDto::from_subtitle)
            .collect();
        let sample_after: Vec<SubtitleDto> = adjusted_subtitles[..sample_count]
            .iter()
            .map(SubtitleDto::from_subtitle)
            .collect();

        let dto = DelayResultDto {
            offset_ms: offset_millis,
            subtitles_adjusted: subtitles.len(),
            backup_path: "N/A (dry-run)".into(),
            dry_run: true,
            sample_before: Some(sample_before),
            sample_after: Some(sample_after),
        };

        output_success(format, &dto, || {
            println!("Dry-run: time offset would be applied");
            println!();
            println!("Offset: {} ms", offset_millis);
            println!("Subtitles affected: {}", subtitles.len());
            println!();
            println!("Sample (first {} subtitles):", sample_count);
            for i in 0..sample_count {
                println!(
                    "  [{}] {} --> {}  =>  {} --> {}",
                    subtitles[i].index.as_ref(),
                    Subtitle::format_timestamp(subtitles[i].start_time.as_ref()),
                    Subtitle::format_timestamp(subtitles[i].end_time.as_ref()),
                    Subtitle::format_timestamp(adjusted_subtitles[i].start_time.as_ref()),
                    Subtitle::format_timestamp(adjusted_subtitles[i].end_time.as_ref()),
                );
            }
        });

        return Ok(());
    }

    // Create backup
    let backup_service = SubRipBackupService::new();
    let backup_result = backup_service.create_backup(&resolved.full_path);

    let backup_path = match backup_result {
        Ok(Some(path)) => {
            debug!("backup created: {}", path);
            path
        }
        Ok(None) => {
            error!("file does not exist: {}", file);
            output_error(
                format,
                "file_not_found",
                &format!("File does not exist: {}", file),
                None,
            );
            return Err(anyhow::anyhow!(""));
        }
        Err(e) => {
            error!("failed to create backup: {}", e);
            output_error(
                format,
                "backup_failed",
                &format!("Failed to create backup: {}", e),
                None,
            );
            return Err(anyhow::anyhow!(""));
        }
    };

    // Write adjusted subtitles
    match service.write_all(&resolved.filename, &adjusted_subtitles) {
        Ok(_) => {
            info!(
                "successfully applied offset to {} subtitles",
                adjusted_subtitles.len()
            );

            let dto = DelayResultDto {
                offset_ms: offset_millis,
                subtitles_adjusted: adjusted_subtitles.len(),
                backup_path: backup_path.clone(),
                dry_run: false,
                sample_before: None,
                sample_after: None,
            };

            output_success(format, &dto, || {
                println!("✓ Time offset applied successfully");
                println!();
                println!("Backup: {}", backup_path);
                println!("Offset: {} ms", offset_millis);
                println!("Subtitles adjusted: {}", adjusted_subtitles.len());
            });

            Ok(())
        }
        Err(e) => {
            let cli_err = utils::format_subtitle_error(&e, file);
            output_error(
                format,
                &cli_err.code,
                &cli_err.message,
                cli_err.hint.as_deref(),
            );
            Err(anyhow::anyhow!(""))
        }
    }
}
