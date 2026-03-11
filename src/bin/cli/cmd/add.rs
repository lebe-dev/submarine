use crate::cli::OutputFormat;
use crate::dto::{AddResultDto, SubtitleDto};
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::subtitle::model::{Subtitle, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use serde::Serialize;

#[derive(Serialize)]
struct AddDryRunDto {
    subtitle: SubtitleDto,
    #[serde(flatten)]
    result: AddResultDto,
}

pub fn handle(
    file: &str,
    timestamps: &str,
    text: &str,
    dry_run: bool,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!("adding subtitle to file: {}", file);

    // Validate text input
    utils::reject_control_chars(text, "text")?;

    let resolved = utils::resolve_new_path(file)?;

    // Parse timestamps
    debug!("parsing timestamps: {}", timestamps);
    let parts: Vec<&str> = timestamps.splitn(2, '-').collect();

    if parts.len() != 2 {
        output_error(
            format,
            "invalid_timestamp_format",
            "Invalid timestamp format (expected 'HH:MM:SS,mmm-HH:MM:SS,mmm')",
            Some("example: \"00:00:10,000-00:00:12,500\""),
        );
        return Err(anyhow::anyhow!(""));
    }

    let start_str = parts[0];
    let end_str = parts[1];

    let start_duration = Subtitle::parse_timestamp(start_str)
        .map_err(|e| anyhow::anyhow!("invalid start timestamp: {}", e))?;
    let start_timestamp = SubtitleTimestamp::try_new(start_duration)
        .map_err(|e| anyhow::anyhow!("invalid start timestamp value: {}", e))?;

    let end_duration = Subtitle::parse_timestamp(end_str)
        .map_err(|e| anyhow::anyhow!("invalid end timestamp: {}", e))?;
    let end_timestamp = SubtitleTimestamp::try_new(end_duration)
        .map_err(|e| anyhow::anyhow!("invalid end timestamp value: {}", e))?;

    let subtitle_text = SubtitleText::try_new(text.to_string())
        .map_err(|e| anyhow::anyhow!("invalid text: {}", e))?;

    if dry_run {
        info!("dry-run mode, previewing add");
        let service = SubRipService::new(resolved.base_dir);
        let existing = service.get_all(&resolved.filename).unwrap_or_default();
        let next_index = existing
            .iter()
            .map(|s| *s.index.as_ref())
            .max()
            .unwrap_or(0)
            + 1;

        let preview_idx = lib::subtitle::model::SubtitleIndex::try_new(next_index)
            .unwrap_or_else(|_| lib::subtitle::model::SubtitleIndex::try_new(1).unwrap());

        let preview = Subtitle::new(preview_idx, start_timestamp, end_timestamp, subtitle_text)
            .map_err(|e| anyhow::anyhow!("invalid subtitle: {}", e))?;

        let dto = AddDryRunDto {
            subtitle: SubtitleDto::from_subtitle(&preview),
            result: AddResultDto {
                new_index: next_index,
                total_subtitles: existing.len() + 1,
                backup_path: "N/A (dry-run)".into(),
                dry_run: true,
            },
        };

        output_success(format, &dto, || {
            println!("Dry-run: subtitle would be added");
            println!();
            println!("New index: {}", next_index);
            println!("Total subtitles: {}", existing.len() + 1);
            println!();
            println!("{}", preview);
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
            debug!("file does not exist, skipping backup");
            "N/A (new file)".to_string()
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

    debug!("adding new subtitle...");
    match service.add(
        &resolved.filename,
        start_timestamp,
        end_timestamp,
        subtitle_text,
    ) {
        Ok(report) => {
            info!(
                "subtitle added successfully with index {}",
                report.new_index
            );

            let dto = AddResultDto {
                new_index: report.new_index,
                total_subtitles: report.total_subtitles,
                backup_path: backup_path.clone(),
                dry_run: false,
            };

            output_success(format, &dto, || {
                println!("✓ Subtitle added successfully");
                println!();
                println!("New index: {}", report.new_index);
                println!("Total subtitles: {}", report.total_subtitles);
                println!("Backup: {}", backup_path);
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
