use crate::cli::ImportFormat;
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

pub fn handle(
    srt_file: &str,
    input_file: &str,
    format: ImportFormat,
    reference: Option<&str>,
    delimiter: &str,
    dry_run: bool,
    force: bool,
) -> anyhow::Result<()> {
    info!("importing subtitles from {} to {}", input_file, srt_file);

    match format {
        ImportFormat::Anchored => {
            if reference.is_none() {
                error!("anchored format requires --reference parameter");
                eprintln!("error: Anchored format requires --reference FILE parameter");
                eprintln!("hint: Use --reference to specify the SRT file containing timestamps");
                std::process::exit(1);
            }
        }
        ImportFormat::Csv => {
            if delimiter.len() != 1 {
                error!("invalid delimiter: must be a single character");
                eprintln!(
                    "error: Delimiter must be a single character, got: '{}'",
                    delimiter
                );
                std::process::exit(1);
            }
        }
    }

    let delimiter_char = if matches!(format, ImportFormat::Csv) {
        delimiter.chars().next().unwrap()
    } else {
        '|' // Default, won't be used for anchored
    };
    if matches!(format, ImportFormat::Csv) {
        debug!("using delimiter: '{}'", delimiter_char);
    }

    let srt_path = Path::new(srt_file);
    debug!("parsing srt file path: {:?}", srt_path);

    if srt_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;

        let resolved = current_dir.join(srt_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve srt file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", srt_path);
            return Err(anyhow::anyhow!(
                "invalid srt file path: path traversal not allowed"
            ));
        }
    }

    let canonical_srt_path = srt_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("failed to resolve srt file path: {}", e))?;
    debug!("canonical srt path: {:?}", canonical_srt_path);

    let base_dir = canonical_srt_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid srt file path"))?
        .to_path_buf();
    debug!("base directory: {:?}", base_dir);

    let filename = canonical_srt_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("invalid srt file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("invalid UTF-8 in filename"))?
        .to_string();
    debug!("srt filename: {}", filename);

    let input_path = Path::new(input_file);
    debug!("parsing input file path: {:?}", input_path);

    if input_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;

        let resolved = current_dir.join(input_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve input file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", input_path);
            return Err(anyhow::anyhow!(
                "invalid input file path: path traversal not allowed"
            ));
        }
    }

    let canonical_input_path = input_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("failed to resolve input file path: {}", e))?;
    debug!("canonical input path: {:?}", canonical_input_path);

    let subtitles = match format {
        ImportFormat::Csv => handle_csv_import(&canonical_input_path, delimiter_char)?,
        ImportFormat::Anchored => {
            let reference_path = reference.unwrap(); // Safe: validated above
            handle_anchored_import(&canonical_input_path, reference_path, &base_dir)?
        }
    };

    show_preview(&subtitles);

    if dry_run {
        info!("dry-run mode, operations not executed");
        println!("Dry-run mode: no subtitles were imported");
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
    let backup_result = backup_service.create_backup(&canonical_srt_path);

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
            eprintln!("error: Failed to create backup: {}", e);
            std::process::exit(1);
        }
    };

    let service = SubRipService::new(base_dir);

    let (imported_count, start_index, end_index, total_subtitles) = match format {
        ImportFormat::Csv => {
            debug!("adding {} subtitles...", subtitles.len());
            let mut added_count = 0;
            let mut start_idx = None;
            let mut end_idx = None;

            for subtitle in subtitles {
                match service.add(
                    &filename,
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
                        debug!(
                            "added subtitle {} (index {})",
                            added_count, report.new_index
                        );
                    }
                    Err(e) => {
                        error!("failed to add subtitle: {:?}", e);
                        handle_subtitle_error(e);
                        std::process::exit(1);
                    }
                }
            }

            let total = match service.get_all(&filename) {
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
            debug!(
                "merging {} subtitles with existing file...",
                subtitles.len()
            );

            let existing_subs = match service.get_all(&filename) {
                Ok(subs) => {
                    debug!("loaded {} existing subtitles", subs.len());
                    subs
                }
                Err(SubtitleError::FileNotFound(_)) => {
                    debug!("file does not exist, creating new file");
                    Vec::new()
                }
                Err(e) => {
                    error!("failed to load existing file: {:?}", e);
                    handle_subtitle_error(e);
                    std::process::exit(1);
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

            debug!("writing {} total subtitles to file...", merged_subs.len());

            match service.write_all(&filename, &merged_subs) {
                Ok(_) => {
                    debug!("wrote {} subtitles to file", merged_subs.len());
                }
                Err(e) => {
                    error!("failed to write updated file: {:?}", e);
                    handle_subtitle_error(e);
                    std::process::exit(1);
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

    println!("✓ Subtitles imported successfully");
    println!();
    println!("Imported: {} subtitles", imported_count);
    println!("Index range: {}-{}", start_index, end_index);
    println!("Total subtitles: {}", total_subtitles);
    println!("Backup: {}", backup_path);

    Ok(())
}

fn handle_csv_import(csv_path: &Path, delimiter: char) -> anyhow::Result<Vec<Subtitle>> {
    debug!("parsing csv file");
    let import_service = CsvImportService::new();
    let csv_rows = match import_service.parse_csv_file(csv_path, delimiter) {
        Ok(rows) => {
            info!("parsed {} rows from csv file", rows.len());
            rows
        }
        Err(e) => {
            debug!("csv parsing error: {:?}", e);
            handle_subtitle_error(e);
            std::process::exit(1);
        }
    };

    if csv_rows.is_empty() {
        error!("csv file is empty (no data rows)");
        eprintln!("error: CSV file contains no data rows");
        std::process::exit(1);
    }

    debug!("validating and converting {} csv rows...", csv_rows.len());
    convert_csv_to_subtitles(csv_rows)
}

fn convert_csv_to_subtitles(csv_rows: Vec<CsvSubtitleRow>) -> anyhow::Result<Vec<Subtitle>> {
    let mut subtitles = Vec::new();

    for csv_row in csv_rows {
        let start_duration = match Subtitle::parse_timestamp(&csv_row.start_time) {
            Ok(d) => d,
            Err(e) => {
                error!(
                    "failed to parse start timestamp at line {}: {}",
                    csv_row.line_number, e
                );
                eprintln!(
                    "error: Invalid start timestamp at CSV line {}: {}",
                    csv_row.line_number, e
                );
                std::process::exit(1);
            }
        };

        let end_duration = match Subtitle::parse_timestamp(&csv_row.end_time) {
            Ok(d) => d,
            Err(e) => {
                error!(
                    "failed to parse end timestamp at line {}: {}",
                    csv_row.line_number, e
                );
                eprintln!(
                    "error: Invalid end timestamp at CSV line {}: {}",
                    csv_row.line_number, e
                );
                std::process::exit(1);
            }
        };

        let start_time = match SubtitleTimestamp::try_new(start_duration) {
            Ok(t) => t,
            Err(e) => {
                error!(
                    "invalid start timestamp at line {}: {}",
                    csv_row.line_number, e
                );
                eprintln!(
                    "error: Invalid start timestamp at CSV line {}: {}",
                    csv_row.line_number, e
                );
                std::process::exit(1);
            }
        };

        let end_time = match SubtitleTimestamp::try_new(end_duration) {
            Ok(t) => t,
            Err(e) => {
                error!(
                    "invalid end timestamp at line {}: {}",
                    csv_row.line_number, e
                );
                eprintln!(
                    "error: Invalid end timestamp at CSV line {}: {}",
                    csv_row.line_number, e
                );
                std::process::exit(1);
            }
        };

        let text = match SubtitleText::try_new(csv_row.text.clone()) {
            Ok(t) => t,
            Err(e) => {
                error!("invalid text at line {}: {}", csv_row.line_number, e);
                eprintln!(
                    "error: Invalid text (empty or whitespace) at CSV line {}: {}",
                    csv_row.line_number, e
                );
                std::process::exit(1);
            }
        };

        let temp_index = lib::subtitle::model::SubtitleIndex::try_new((subtitles.len() + 1) as u32)
            .expect("valid subtitle index");
        let subtitle = match Subtitle::new(temp_index, start_time, end_time, text) {
            Ok(s) => s,
            Err(e) => {
                error!("invalid subtitle at line {}: {}", csv_row.line_number, e);
                eprintln!(
                    "error: Invalid subtitle at CSV line {}: {}",
                    csv_row.line_number, e
                );
                std::process::exit(1);
            }
        };

        subtitles.push(subtitle);
        debug!(
            "validated subtitle {} from CSV line {}",
            subtitles.len(),
            csv_row.line_number
        );
    }

    debug!("validated {} subtitles successfully", subtitles.len());
    Ok(subtitles)
}

fn handle_anchored_import(
    anchored_path: &Path,
    reference_file: &str,
    _base_dir: &Path,
) -> anyhow::Result<Vec<Subtitle>> {
    debug!("parsing anchored format file");

    let import_service = AnchoredImportService::new();
    let anchored_rows = match import_service.parse_anchored_file(anchored_path) {
        Ok(rows) => {
            info!("parsed {} entries from anchored file", rows.len());
            rows
        }
        Err(e) => {
            debug!("anchored parsing error: {:?}", e);
            handle_subtitle_error(e);
            std::process::exit(1);
        }
    };

    debug!("loading reference file: {}", reference_file);
    let ref_path = Path::new(reference_file);

    let canonical_ref_path = if ref_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;
        let resolved = current_dir.join(ref_path);
        resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve reference file path: {}", e))?
    } else {
        ref_path
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve reference file path: {}", e))?
    };

    let ref_base_dir = canonical_ref_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid reference file path"))?
        .to_path_buf();

    let ref_filename = canonical_ref_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("invalid reference file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("invalid UTF-8 in filename"))?;

    let service = SubRipService::new(ref_base_dir);
    let reference_subtitles = match service.get_all(ref_filename) {
        Ok(subs) => {
            info!("loaded {} subtitles from reference file", subs.len());
            subs
        }
        Err(e) => {
            error!("failed to load reference file: {:?}", e);
            eprintln!("error: Failed to load reference file: {}", e);
            std::process::exit(1);
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
                let new_text = match SubtitleText::try_new(anchored_row.text.clone()) {
                    Ok(t) => t,
                    Err(e) => {
                        error!("invalid text at line {}: {}", anchored_row.line_number, e);
                        eprintln!(
                            "error: Invalid text (empty or whitespace) at line {}: {}",
                            anchored_row.line_number, e
                        );
                        std::process::exit(1);
                    }
                };

                match Subtitle::new(
                    ref_subtitle.index,
                    ref_subtitle.start_time,
                    ref_subtitle.end_time,
                    new_text,
                ) {
                    Ok(subtitle) => {
                        updated_subtitles.push(subtitle);
                        debug!(
                            "matched index {} from line {}",
                            index, anchored_row.line_number
                        );
                    }
                    Err(e) => {
                        error!("failed to create subtitle for index {}: {}", index, e);
                        eprintln!(
                            "error: Failed to create subtitle for index {}: {}",
                            index, e
                        );
                        std::process::exit(1);
                    }
                }
            }
            None => {
                error!(
                    "reference file does not contain subtitle with index {}",
                    index
                );
                eprintln!(
                    "error: Reference file does not contain subtitle with index {}",
                    index
                );
                eprintln!("hint: Check that the reference file contains all required indices");
                std::process::exit(1);
            }
        }
    }

    let mut final_subtitles = updated_subtitles;

    final_subtitles.sort_by_key(|s| *s.index.as_ref());

    debug!(
        "built final subtitle list with {} entries from anchored file",
        final_subtitles.len()
    );

    Ok(final_subtitles)
}

fn handle_subtitle_error(e: SubtitleError) {
    match e {
        SubtitleError::FileNotFound(path) => {
            error!("file not found: {}", path);
            eprintln!("error: File not found: {}", path);
        }
        SubtitleError::InvalidPath(msg) => {
            error!("invalid file path: {}", msg);
            eprintln!("error: Invalid file path: {}", msg);
        }
        SubtitleError::CsvParseError { line, message } => {
            error!("csv parse error at line {}: {}", line, message);
            eprintln!("error: CSV parsing failed at line {}", line);
            eprintln!("  {}", message);
            eprintln!("hint: Check the CSV file format at line {}", line);
        }
        SubtitleError::InvalidCsvHeader(delimiter, actual) => {
            error!(
                "invalid csv header: expected start_time{}end_time{}text, got {}",
                delimiter, delimiter, actual
            );
            eprintln!("error: Invalid CSV header");
            eprintln!(
                "  Expected: start_time{}end_time{}text",
                delimiter, delimiter
            );
            eprintln!("  Got: {}", actual);
        }
        SubtitleError::AnchoredParseError { line, message } => {
            error!("anchored parse error at line {}: {}", line, message);
            eprintln!("error: Anchored format parsing failed at line {}", line);
            eprintln!("  {}", message);
            eprintln!("hint: Ensure format is [INDEX] TEXT with proper structure");
        }
        SubtitleError::ReferenceIndexNotFound { index } => {
            error!("reference file missing index: {}", index);
            eprintln!("error: Reference file does not contain index {}", index);
        }
        SubtitleError::TimestampConflict {
            last_end,
            new_start,
        } => {
            error!(
                "timestamp conflict: new subtitle starts at {}, but last subtitle ends at {}",
                new_start, last_end
            );
            eprintln!("error: Timestamp conflict detected");
            eprintln!("  Previous subtitle ends at: {}", last_end);
            eprintln!("  New subtitle starts at: {}", new_start);
            eprintln!("  Each subtitle must start at or after the previous one ends");
        }
        SubtitleError::ParseError(err) => {
            error!("parse error: {}", err);
            eprintln!("error: Failed to parse file: {}", err);
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
        _ => {
            error!("unexpected error: {}", e);
            eprintln!("error: {}", e);
        }
    }
}
