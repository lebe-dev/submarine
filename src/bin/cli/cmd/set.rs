use crate::cli::OutputFormat;
use crate::dto::SetResultDto;
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::subtitle::model::{Subtitle, SubtitleText, SubtitleTimestamp, SubtitleUpdate};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};

pub fn handle(
    file: &str,
    index: u32,
    start: Option<String>,
    end: Option<String>,
    text: Option<String>,
    dry_run: bool,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!("setting subtitle {} in file: {}", index, file);

    if start.is_none() && end.is_none() && text.is_none() {
        output_error(
            format,
            "no_fields_to_update",
            "At least one of --start, --end, or --text must be specified",
            None,
        );
        return Err(anyhow::anyhow!(""));
    }

    // Validate text input
    if let Some(ref t) = text {
        utils::reject_control_chars(t, "text")?;
    }

    let resolved = utils::resolve_existing_path(file)?;

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

    let subtitle_text = if let Some(text_str) = text {
        debug!("validating text (length: {})", text_str.len());
        Some(SubtitleText::try_new(text_str).map_err(|e| anyhow::anyhow!("invalid text: {}", e))?)
    } else {
        None
    };

    let update = SubtitleUpdate {
        start_time: start_timestamp,
        end_time: end_timestamp,
        text: subtitle_text,
    };

    if dry_run {
        info!("dry-run mode, previewing changes");
        let service = SubRipService::new(resolved.base_dir);

        // Check subtitle exists and show before/after
        match service.get_by_id(&resolved.filename, index) {
            Ok(Some(existing)) => {
                let updated = update
                    .apply_to(&existing)
                    .map_err(|e| anyhow::anyhow!("invalid update: {}", e))?;

                let mut fields = Vec::new();
                if update.start_time.is_some() {
                    fields.push("start_time".to_string());
                }
                if update.end_time.is_some() {
                    fields.push("end_time".to_string());
                }
                if update.text.is_some() {
                    fields.push("text".to_string());
                }

                let dto = SetResultDto {
                    index,
                    fields_updated: fields,
                    backup_path: "N/A (dry-run)".into(),
                    dry_run: true,
                };

                output_success(format, &dto, || {
                    println!("Dry-run: Subtitle {} would be updated", index);
                    println!();
                    println!("Before:");
                    println!("{}", existing);
                    println!();
                    println!("After:");
                    println!("{}", updated);
                });
                return Ok(());
            }
            Ok(None) => {
                output_error(
                    format,
                    "subtitle_not_found",
                    &format!("Subtitle with index {} not found in file", index),
                    None,
                );
                return Err(anyhow::anyhow!(""));
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
        }
    }

    let backup_service = SubRipBackupService::new();
    let backup_result = backup_service.create_backup(&resolved.full_path);

    let backup_path = match backup_result {
        Ok(Some(path)) => {
            debug!("backup created: {}", path);
            path
        }
        Ok(None) => {
            error!("file does not exist, cannot update");
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

    let service = SubRipService::new(resolved.base_dir);

    debug!("updating subtitle {}...", index);
    match service.set(&resolved.filename, index, update) {
        Ok(report) => {
            info!("subtitle {} updated successfully", index);

            let dto = SetResultDto {
                index,
                fields_updated: report.fields_updated.clone(),
                backup_path: backup_path.clone(),
                dry_run: false,
            };

            output_success(format, &dto, || {
                println!("✓ Subtitle {} updated successfully", index);
                println!();
                println!("Backup: {}", backup_path);
                println!("Fields updated: {}", report.fields_updated.join(", "));
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
