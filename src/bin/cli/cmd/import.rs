use lib::import::ports::ImportService;
use lib::import::service::CsvImportService;
use lib::subtitle::model::{Subtitle, SubtitleError, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

pub fn handle(srt_file: &str, csv_file: &str, delimiter: &str) -> anyhow::Result<()> {
    info!("importing subtitles from {} to {}", csv_file, srt_file);

    // 1. Validate delimiter
    if delimiter.len() != 1 {
        error!("invalid delimiter: must be a single character");
        eprintln!(
            "error: Delimiter must be a single character, got: '{}'",
            delimiter
        );
        std::process::exit(1);
    }
    let delimiter_char = delimiter.chars().next().unwrap();
    debug!("using delimiter: '{}'", delimiter_char);

    // 2. Parse and validate SRT file path with path traversal protection
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

    // 3. Parse and validate CSV file path
    let csv_path = Path::new(csv_file);
    debug!("parsing csv file path: {:?}", csv_path);

    if csv_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;

        let resolved = current_dir.join(csv_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve csv file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", csv_path);
            return Err(anyhow::anyhow!(
                "invalid csv file path: path traversal not allowed"
            ));
        }
    }

    let canonical_csv_path = csv_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("failed to resolve csv file path: {}", e))?;
    debug!("canonical csv path: {:?}", canonical_csv_path);

    // 4. Parse CSV file
    debug!("parsing csv file");
    let import_service = CsvImportService::new();
    let csv_rows = match import_service.parse_csv_file(&canonical_csv_path, delimiter_char) {
        Ok(rows) => {
            info!("parsed {} rows from csv file", rows.len());
            rows
        }
        Err(e) => {
            debug!("csv parsing error: {:?}", e);
            handle_subtitle_error(e, srt_file, csv_file);
            std::process::exit(1);
        }
    };

    if csv_rows.is_empty() {
        error!("csv file is empty (no data rows)");
        eprintln!("error: CSV file contains no data rows");
        std::process::exit(1);
    }

    // 5. Create service and execute import
    let service = SubRipService::new(base_dir);

    debug!("converting and adding {} subtitles...", csv_rows.len());
    let mut added_count = 0;
    let mut start_index = None;
    let mut end_index = None;
    let mut last_backup_path = None;

    for csv_row in csv_rows {
        // Parse timestamps
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

        // Create validated types
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

        // Add subtitle using SubtitleService::add()
        match service.add(&filename, start_time, end_time, text) {
            Ok(report) => {
                if start_index.is_none() {
                    start_index = Some(report.new_index);
                }
                end_index = Some(report.new_index);
                last_backup_path = Some(report.backup_path);
                added_count += 1;
                debug!("added subtitle {} (index {})", added_count, report.new_index);
            }
            Err(e) => {
                error!(
                    "failed to add subtitle at CSV line {}: {:?}",
                    csv_row.line_number, e
                );
                handle_subtitle_error(e, srt_file, csv_file);
                std::process::exit(1);
            }
        }
    }

    // Get final count
    let total_subtitles = match service.get_all(&filename) {
        Ok(subs) => subs.len(),
        Err(_) => added_count, // fallback
    };

    info!(
        "imported {} subtitles successfully (indices {}-{})",
        added_count,
        start_index.unwrap_or(0),
        end_index.unwrap_or(0)
    );

    println!("✓ Subtitles imported successfully");
    println!();
    println!("Imported: {} subtitles", added_count);
    println!(
        "Index range: {}-{}",
        start_index.unwrap_or(0),
        end_index.unwrap_or(0)
    );
    println!("Total subtitles: {}", total_subtitles);
    if let Some(backup) = last_backup_path {
        println!("Last backup: {}", backup);
    }

    Ok(())
}

fn handle_subtitle_error(e: SubtitleError, srt_file: &str, _csv_file: &str) {
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
            eprintln!("error: Failed to parse SRT file: {}", err);
            eprintln!("hint: Try running 'sm doctor --fix {}' first", srt_file);
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
