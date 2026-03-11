use crate::cli::{ImportFormat, OutputFormat};
use crate::dto::ImportResultDto;
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::import::model::CsvSubtitleRow;
use lib::import::ports::ImportService;
use lib::import::service::{AnchoredImportService, CsvImportService};
use lib::subtitle::model::{Subtitle, SubtitleError, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::collections::HashMap;
use std::io::{self, Write};
use std::path::Path;

/// Show preview of subtitles to be imported
fn show_preview(subtitles: &[Subtitle]) {
    println!("Preview of {} subtitle(s) to be imported:", subtitles.len());
    println!();

    for (i, subtitle) in subtitles.iter().enumerate() {
        if i >= 5 {
            println!("... and {} more subtitle(s)", subtitles.len() - 5);
            break;
        }
        println!(
            "  [{:>4}] {} --> {}",
            subtitle.index.into_inner(),
            Subtitle::format_timestamp(subtitle.start_time.as_ref()),
            Subtitle::format_timestamp(subtitle.end_time.as_ref())
        );
        let text = subtitle.text.as_str();
        let preview_text = match text.char_indices().nth(50) {
            Some((byte_idx, _)) => format!("{}...", &text[..byte_idx]),
            None => text.to_string(),
        };
        println!("       {}", preview_text.replace('\n', " "));
        println!();
    }
}

/// Ask user for confirmation to proceed with import
fn confirm_import(count: usize) -> anyhow::Result<bool> {
    print!("Proceed with importing {} subtitle(s)? (y/N): ", count);
    io::stdout().flush()?;

    let mut input = String::new();
    io::stdin().read_line(&mut input)?;

    Ok(input.trim().eq_ignore_ascii_case("y"))
}

#[allow(clippy::too_many_arguments)]
pub fn handle(
    srt_file: &str,
    input_file: &str,
    import_format: ImportFormat,
    reference: Option<&str>,
    delimiter: &str,
    dry_run: bool,
    force: bool,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!("importing subtitles from {} to {}", input_file, srt_file);

    match import_format {
        ImportFormat::Anchored => {
            if reference.is_none() {
                output_error(
                    format,
                    "missing_reference",
                    "Anchored format requires --reference FILE parameter",
                    Some("Use --reference to specify the SRT file containing timestamps"),
                );
                return Err(anyhow::anyhow!(""));
            }
        }
        ImportFormat::Csv => {
            if delimiter.len() != 1 {
                output_error(
                    format,
                    "invalid_delimiter",
                    &format!("Delimiter must be a single character, got: '{}'", delimiter),
                    None,
                );
                return Err(anyhow::anyhow!(""));
            }
        }
    }

    let delimiter_char = if matches!(import_format, ImportFormat::Csv) {
        delimiter.chars().next().unwrap()
    } else {
        '|'
    };

    let srt_resolved = utils::resolve_existing_path(srt_file)?;
    let input_resolved = utils::resolve_existing_path(input_file)?;

    let subtitles = match import_format {
        ImportFormat::Csv => handle_csv_import(&input_resolved.full_path, delimiter_char, format)?,
        ImportFormat::Anchored => {
            let reference_path = reference.unwrap();
            handle_anchored_import(
                &input_resolved.full_path,
                reference_path,
                &srt_resolved.base_dir,
                format,
            )?
        }
    };

    show_preview(&subtitles);

    if dry_run {
        info!("dry-run mode, operations not executed");

        let min_idx = subtitles
            .iter()
            .map(|s| *s.index.as_ref())
            .min()
            .unwrap_or(0);
        let max_idx = subtitles
            .iter()
            .map(|s| *s.index.as_ref())
            .max()
            .unwrap_or(0);

        let dto = ImportResultDto {
            imported_count: subtitles.len(),
            start_index: min_idx,
            end_index: max_idx,
            total_subtitles: 0, // Unknown in dry-run
            backup_path: "N/A (dry-run)".into(),
            dry_run: true,
        };

        output_success(format, &dto, || {
            println!("Dry-run mode: no subtitles were imported");
        });
        return Ok(());
    }

    if !force {
        if !confirm_import(subtitles.len())? {
            info!("operation cancelled by user");
            println!("Cancelled");
            return Ok(());
        }
        info!("user confirmed import operation");
    }

    let backup_service = SubRipBackupService::new();
    let backup_result = backup_service.create_backup(&srt_resolved.full_path);

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

    let service = SubRipService::new(srt_resolved.base_dir);

    let (imported_count, start_index, end_index, total_subtitles) = match import_format {
        ImportFormat::Csv => {
            debug!("adding {} subtitles...", subtitles.len());
            let mut added_count = 0;
            let mut start_idx = None;
            let mut end_idx = None;

            for subtitle in subtitles {
                match service.add(
                    &srt_resolved.filename,
                    subtitle.start_time,
                    subtitle.end_time,
                    subtitle.text,
                ) {
                    Ok(report) => {
                        if start_idx.is_none() {
                            start_idx = Some(report.new_index);
                        }
                        end_idx = Some(report.new_index);
                        added_count += 1;
                    }
                    Err(e) => {
                        let cli_err = utils::format_subtitle_error(&e, srt_file);
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

            let total = match service.get_all(&srt_resolved.filename) {
                Ok(subs) => subs.len(),
                Err(_) => added_count,
            };

            (
                added_count,
                start_idx.unwrap_or(0),
                end_idx.unwrap_or(0),
                total,
            )
        }
        ImportFormat::Anchored => {
            let existing_subs = match service.get_all(&srt_resolved.filename) {
                Ok(subs) => subs,
                Err(SubtitleError::FileNotFound(_)) => Vec::new(),
                Err(e) => {
                    let cli_err = utils::format_subtitle_error(&e, srt_file);
                    output_error(
                        format,
                        &cli_err.code,
                        &cli_err.message,
                        cli_err.hint.as_deref(),
                    );
                    return Err(anyhow::anyhow!(""));
                }
            };

            let mut merged_map: HashMap<u32, Subtitle> = HashMap::new();

            for sub in existing_subs {
                merged_map.insert(*sub.index.as_ref(), sub);
            }

            for sub in &subtitles {
                merged_map.insert(*sub.index.as_ref(), sub.clone());
            }

            let mut merged_subs: Vec<Subtitle> = merged_map.into_values().collect();
            merged_subs.sort_by_key(|s| *s.index.as_ref());

            match service.write_all(&srt_resolved.filename, &merged_subs) {
                Ok(_) => {}
                Err(e) => {
                    let cli_err = utils::format_subtitle_error(&e, srt_file);
                    output_error(
                        format,
                        &cli_err.code,
                        &cli_err.message,
                        cli_err.hint.as_deref(),
                    );
                    return Err(anyhow::anyhow!(""));
                }
            }

            let min_idx = subtitles
                .iter()
                .map(|s| *s.index.as_ref())
                .min()
                .unwrap_or(0);
            let max_idx = subtitles
                .iter()
                .map(|s| *s.index.as_ref())
                .max()
                .unwrap_or(0);

            (subtitles.len(), min_idx, max_idx, merged_subs.len())
        }
    };

    info!(
        "imported {} subtitles successfully (indices {}-{})",
        imported_count, start_index, end_index
    );

    let dto = ImportResultDto {
        imported_count,
        start_index,
        end_index,
        total_subtitles,
        backup_path: backup_path.clone(),
        dry_run: false,
    };

    output_success(format, &dto, || {
        println!("✓ Subtitles imported successfully");
        println!();
        println!("Imported: {} subtitles", imported_count);
        println!("Index range: {}-{}", start_index, end_index);
        println!("Total subtitles: {}", total_subtitles);
        println!("Backup: {}", backup_path);
    });

    Ok(())
}

fn handle_csv_import(
    csv_path: &Path,
    delimiter: char,
    format: &OutputFormat,
) -> anyhow::Result<Vec<Subtitle>> {
    debug!("parsing csv file");
    let import_service = CsvImportService::new();
    let csv_rows = match import_service.parse_csv_file(csv_path, delimiter) {
        Ok(rows) => {
            info!("parsed {} rows from csv file", rows.len());
            rows
        }
        Err(e) => {
            let cli_err = utils::format_subtitle_error(&e, &csv_path.display().to_string());
            output_error(
                format,
                &cli_err.code,
                &cli_err.message,
                cli_err.hint.as_deref(),
            );
            return Err(anyhow::anyhow!(""));
        }
    };

    if csv_rows.is_empty() {
        output_error(format, "empty_csv", "CSV file contains no data rows", None);
        return Err(anyhow::anyhow!(""));
    }

    convert_csv_to_subtitles(csv_rows)
}

fn convert_csv_to_subtitles(csv_rows: Vec<CsvSubtitleRow>) -> anyhow::Result<Vec<Subtitle>> {
    let mut subtitles = Vec::new();

    for csv_row in csv_rows {
        let start_duration = Subtitle::parse_timestamp(&csv_row.start_time).map_err(|e| {
            anyhow::anyhow!(
                "Invalid start timestamp at CSV line {}: {}",
                csv_row.line_number,
                e
            )
        })?;

        let end_duration = Subtitle::parse_timestamp(&csv_row.end_time).map_err(|e| {
            anyhow::anyhow!(
                "Invalid end timestamp at CSV line {}: {}",
                csv_row.line_number,
                e
            )
        })?;

        let start_time = SubtitleTimestamp::try_new(start_duration).map_err(|e| {
            anyhow::anyhow!(
                "Invalid start timestamp at CSV line {}: {}",
                csv_row.line_number,
                e
            )
        })?;

        let end_time = SubtitleTimestamp::try_new(end_duration).map_err(|e| {
            anyhow::anyhow!(
                "Invalid end timestamp at CSV line {}: {}",
                csv_row.line_number,
                e
            )
        })?;

        let text = SubtitleText::try_new(csv_row.text.clone()).map_err(|e| {
            anyhow::anyhow!(
                "Invalid text (empty or whitespace) at CSV line {}: {}",
                csv_row.line_number,
                e
            )
        })?;

        let temp_index = lib::subtitle::model::SubtitleIndex::try_new((subtitles.len() + 1) as u32)
            .expect("valid subtitle index");
        let subtitle = Subtitle::new(temp_index, start_time, end_time, text).map_err(|e| {
            anyhow::anyhow!(
                "Invalid subtitle at CSV line {}: {}",
                csv_row.line_number,
                e
            )
        })?;

        subtitles.push(subtitle);
    }

    Ok(subtitles)
}

fn handle_anchored_import(
    anchored_path: &Path,
    reference_file: &str,
    _base_dir: &Path,
    format: &OutputFormat,
) -> anyhow::Result<Vec<Subtitle>> {
    debug!("parsing anchored format file");

    let import_service = AnchoredImportService::new();
    let anchored_rows = match import_service.parse_anchored_file(anchored_path) {
        Ok(rows) => {
            info!("parsed {} entries from anchored file", rows.len());
            rows
        }
        Err(e) => {
            let cli_err = utils::format_subtitle_error(&e, &anchored_path.display().to_string());
            output_error(
                format,
                &cli_err.code,
                &cli_err.message,
                cli_err.hint.as_deref(),
            );
            return Err(anyhow::anyhow!(""));
        }
    };

    debug!("loading reference file: {}", reference_file);
    let ref_resolved = utils::resolve_existing_path(reference_file)?;
    let service = SubRipService::new(ref_resolved.base_dir);

    let reference_subtitles = match service.get_all(&ref_resolved.filename) {
        Ok(subs) => {
            info!("loaded {} subtitles from reference file", subs.len());
            subs
        }
        Err(e) => {
            let cli_err = utils::format_subtitle_error(&e, reference_file);
            output_error(
                format,
                &cli_err.code,
                &cli_err.message,
                cli_err.hint.as_deref(),
            );
            return Err(anyhow::anyhow!(""));
        }
    };

    let mut ref_map: HashMap<u32, &Subtitle> = HashMap::new();
    for subtitle in &reference_subtitles {
        ref_map.insert(*subtitle.index.as_ref(), subtitle);
    }

    let mut updated_subtitles = Vec::new();

    for anchored_row in &anchored_rows {
        let index = anchored_row.index;

        match ref_map.get(&index) {
            Some(ref_subtitle) => {
                let new_text = SubtitleText::try_new(anchored_row.text.clone()).map_err(|e| {
                    anyhow::anyhow!(
                        "Invalid text (empty or whitespace) at line {}: {}",
                        anchored_row.line_number,
                        e
                    )
                })?;

                let subtitle = Subtitle::new(
                    ref_subtitle.index,
                    ref_subtitle.start_time,
                    ref_subtitle.end_time,
                    new_text,
                )
                .map_err(|e| {
                    anyhow::anyhow!("Failed to create subtitle for index {}: {}", index, e)
                })?;

                updated_subtitles.push(subtitle);
            }
            None => {
                output_error(
                    format,
                    "reference_index_not_found",
                    &format!(
                        "Reference file does not contain subtitle with index {}",
                        index
                    ),
                    Some("Check that the reference file contains all required indices"),
                );
                return Err(anyhow::anyhow!(""));
            }
        }
    }

    updated_subtitles.sort_by_key(|s| *s.index.as_ref());

    Ok(updated_subtitles)
}
