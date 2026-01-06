use lib::subtitle::model::{Subtitle, SubtitleError};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use lib::verify::model::{ComparisonStatus, VerificationReport};
use lib::verify::service;
use log::{debug, error, info};
use std::path::Path;

/// Parse range string in format "START-END" into (start, end) tuple
fn parse_range(range: &str) -> anyhow::Result<(u32, u32)> {
    let parts: Vec<&str> = range.split('-').collect();

    if parts.len() != 2 {
        error!("invalid range format: {}", range);
        eprintln!(
            "error: invalid range format '{}'. Expected format: START-END (e.g., 1-50)",
            range
        );
        std::process::exit(1);
    }

    let start = parts[0].trim().parse::<u32>().map_err(|_| {
        error!("invalid start index in range: {}", parts[0]);
        eprintln!(
            "error: invalid start index '{}'. Must be a positive number",
            parts[0]
        );
        std::process::exit(1);
    })?;

    let end = parts[1].trim().parse::<u32>().map_err(|_| {
        error!("invalid end index in range: {}", parts[1]);
        eprintln!(
            "error: invalid end index '{}'. Must be a positive number",
            parts[1]
        );
        std::process::exit(1);
    })?;

    Ok((start, end))
}

/// Entry point for the verify command
///
/// Loads two SRT files and compares them for index and timestamp discrepancies.
/// The first file is treated as the reference (authoritative).
/// Optionally accepts a range parameter to verify only subtitles within that range.
pub fn handle(file1: &str, file2: &str, range: Option<&str>) -> anyhow::Result<()> {
    info!("verifying files: {} and {}", file1, file2);

    let (mut ref_subs, ref_filename) = load_subtitle_file(file1)?;
    let (mut target_subs, target_filename) = load_subtitle_file(file2)?;

    debug!(
        "loaded {} and {} subtitles",
        ref_subs.len(),
        target_subs.len()
    );

    let range_info = if let Some(range_str) = range {
        let (start, end) = parse_range(range_str)?;

        if start < 1 {
            error!("invalid start index: must be >= 1");
            eprintln!("error: start index must be >= 1, got {}", start);
            std::process::exit(1);
        }

        if end < 1 {
            error!("invalid end index: must be >= 1");
            eprintln!("error: end index must be >= 1, got {}", end);
            std::process::exit(1);
        }

        if start > end {
            error!("invalid range: start {} > end {}", start, end);
            eprintln!(
                "error: start index must be <= end index (got {} > {})",
                start, end
            );
            std::process::exit(1);
        }

        info!("filtering subtitles to range {}-{}", start, end);

        ref_subs = ref_subs
            .into_iter()
            .filter(|s| {
                let index = *s.index.as_ref();
                index >= start && index <= end
            })
            .collect();

        target_subs = target_subs
            .into_iter()
            .filter(|s| {
                let index = *s.index.as_ref();
                index >= start && index <= end
            })
            .collect();

        debug!(
            "after filtering: {} ref subtitles, {} target subtitles",
            ref_subs.len(),
            target_subs.len()
        );

        Some((start, end))
    } else {
        None
    };

    if ref_subs.is_empty() {
        eprintln!("error: reference file is empty");
        std::process::exit(1);
    }

    if target_subs.is_empty() {
        eprintln!("error: target file is empty");
        std::process::exit(1);
    }

    let report = service::compare_subtitles(ref_subs, ref_filename, target_subs, target_filename);

    display_verification_report(&report, range_info);

    info!(
        "verification completed: {:.1}% match",
        report.match_percentage()
    );

    if report.has_issues() || !report.is_perfect() {
        std::process::exit(1);
    }

    Ok(())
}

/// Load and validate a subtitle file
///
/// Performs path canonicalization with traversal protection,
/// loads subtitles using SubRipService, and returns the subtitles
/// along with the filename for display.
fn load_subtitle_file(file: &str) -> anyhow::Result<(Vec<Subtitle>, String)> {
    debug!("loading subtitle file: {}", file);

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

    let service = SubRipService::new(base_dir);
    debug!("loading subtitles from file: {}", filename);

    match service.get_all(&filename) {
        Ok(subtitles) => {
            info!(
                "successfully loaded {} subtitle(s) from {}",
                subtitles.len(),
                filename
            );
            Ok((subtitles, filename))
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    error!("file not found: {}", path);
                    eprintln!("error: file not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: failed to parse subtitle file: {}", err);
                    eprintln!("hint: try running 'sm doctor --fix {}' first", filename);
                }
                SubtitleError::IoError(err) => {
                    error!("i/o error: {}", err);
                    eprintln!("error: failed to read file: {}", err);
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

/// Display the verification report to the user
fn display_verification_report(report: &VerificationReport, range_info: Option<(u32, u32)>) {
    println!();
    if let Some((start, end)) = range_info {
        println!("Verifying subtitle files (range: {}-{})", start, end);
    } else {
        println!("Verifying subtitle files");
    }
    println!("========================");
    println!();
    println!(
        "Reference file: {} ({} subtitles)",
        report.ref_file, report.total_ref_count
    );
    println!(
        "Target file:    {} ({} subtitles)",
        report.target_file, report.total_target_count
    );
    println!();

    println!("Results");
    println!("=======");
    println!();

    let match_pct = report.match_percentage();
    println!(
        "Matched: {}/{} ({:.1}%)",
        report.total_matched(),
        report.total_ref_count,
        match_pct
    );

    // Display timestamp mismatches
    if !report.timestamp_mismatches.is_empty() {
        println!(
            "Timestamp mismatches: {}",
            report.timestamp_mismatches.len()
        );
    }

    // Display missing subtitles
    if !report.missing_in_target.is_empty() {
        println!(
            "Missing in {}: {}",
            report.target_file,
            report.missing_in_target.len()
        );
    }

    // Display detected offset
    if let Some(offset) = report.detected_offset {
        println!("Index offset detected: {}", offset);
    }

    // Display extra subtitles
    if !report.extra_in_target.is_empty() {
        println!(
            "Extra in {}: {}",
            report.target_file,
            report.extra_in_target.len()
        );
    }

    println!();

    // Display detailed timestamp mismatches (limited to first 10)
    if !report.timestamp_mismatches.is_empty() {
        println!("Timestamp mismatches:");
        let limit = 10.min(report.timestamp_mismatches.len());
        for entry in &report.timestamp_mismatches[..limit] {
            if let ComparisonStatus::TimestampMismatch {
                ref_start,
                ref_end,
                target_start,
                target_end,
            } = &entry.status
            {
                println!(
                    "  [{}] Reference: {} --> {}",
                    entry.ref_subtitle.index, ref_start, ref_end
                );
                println!("       Target:    {} --> {}", target_start, target_end);
            }
        }
        if report.timestamp_mismatches.len() > 10 {
            println!("  ... and {} more", report.timestamp_mismatches.len() - 10);
        }
        println!();
    }

    // Display detailed missing subtitles (limited to first 10)
    if !report.missing_in_target.is_empty() {
        println!("Missing subtitles:");
        let limit = 10.min(report.missing_in_target.len());
        for entry in &report.missing_in_target[..limit] {
            let sub = &entry.ref_subtitle;
            println!(
                "  [{}] {} --> {} (not found in {})",
                sub.index,
                Subtitle::format_timestamp(sub.start_time.as_ref()),
                Subtitle::format_timestamp(sub.end_time.as_ref()),
                report.target_file
            );
        }
        if report.missing_in_target.len() > 10 {
            println!("  ... and {} more", report.missing_in_target.len() - 10);
        }
        println!();
    }

    // Display detailed extra subtitles (limited to first 10)
    if !report.extra_in_target.is_empty() {
        println!("Extra subtitles in {}:", report.target_file);
        let limit = 10.min(report.extra_in_target.len());
        for sub in &report.extra_in_target[..limit] {
            println!(
                "  [{}] {} --> {}",
                sub.index,
                Subtitle::format_timestamp(sub.start_time.as_ref()),
                Subtitle::format_timestamp(sub.end_time.as_ref())
            );
        }
        if report.extra_in_target.len() > 10 {
            println!("  ... and {} more", report.extra_in_target.len() - 10);
        }
        println!();
    }

    // Display verification status
    if report.is_perfect() {
        println!("Verification: SUCCESS");
    } else if !report.extra_in_target.is_empty() && !report.has_issues() {
        println!("Verification: WARNING (extra subtitles found)");
    } else {
        println!("Verification: FAILED");
    }
    println!();
}
