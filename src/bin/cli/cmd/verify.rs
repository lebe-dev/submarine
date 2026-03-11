use crate::cli::OutputFormat;
use crate::dto::VerifyDto;
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::subtitle::model::Subtitle;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use lib::verify::model::{ComparisonStatus, VerificationReport};
use lib::verify::service;
use log::{debug, info};

pub fn handle(
    file1: &str,
    file2: &str,
    range: Option<&str>,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!("verifying files: {} and {}", file1, file2);

    let (mut ref_subs, ref_filename) = load_subtitle_file(file1, format)?;
    let (mut target_subs, target_filename) = load_subtitle_file(file2, format)?;

    debug!(
        "loaded {} and {} subtitles",
        ref_subs.len(),
        target_subs.len()
    );

    let range_info = if let Some(range_str) = range {
        let (start, end) = utils::parse_range(range_str)?;

        info!("filtering subtitles to range {}-{}", start, end);

        ref_subs.retain(|s| {
            let index = *s.index.as_ref();
            index >= start && index <= end
        });

        target_subs.retain(|s| {
            let index = *s.index.as_ref();
            index >= start && index <= end
        });

        Some((start, end))
    } else {
        None
    };

    if ref_subs.is_empty() {
        output_error(format, "empty_file", "reference file is empty", None);
        return Err(anyhow::anyhow!(""));
    }

    if target_subs.is_empty() {
        output_error(format, "empty_file", "target file is empty", None);
        return Err(anyhow::anyhow!(""));
    }

    let report = service::compare_subtitles(ref_subs, ref_filename, target_subs, target_filename);

    let status = if report.is_perfect() {
        "success"
    } else if !report.extra_in_target.is_empty() && !report.has_issues() {
        "warning"
    } else {
        "failed"
    };

    let dto = VerifyDto {
        ref_file: report.ref_file.clone(),
        target_file: report.target_file.clone(),
        total_ref_count: report.total_ref_count,
        total_target_count: report.total_target_count,
        perfect_matches: report.perfect_matches,
        total_matched: report.total_matched(),
        match_percentage: report.match_percentage(),
        timestamp_mismatches: report.timestamp_mismatches.len(),
        missing_in_target: report.missing_in_target.len(),
        extra_in_target: report.extra_in_target.len(),
        detected_offset: report.detected_offset,
        status: status.to_string(),
    };

    output_success(format, &dto, || {
        display_verification_report(&report, range_info);
    });

    info!(
        "verification completed: {:.1}% match",
        report.match_percentage()
    );

    if report.has_issues() || !report.is_perfect() {
        return Err(anyhow::anyhow!(""));
    }

    Ok(())
}

fn load_subtitle_file(
    file: &str,
    format: &OutputFormat,
) -> anyhow::Result<(Vec<Subtitle>, String)> {
    debug!("loading subtitle file: {}", file);

    let resolved = utils::resolve_existing_path(file)?;
    let service = SubRipService::new(resolved.base_dir);

    match service.get_all(&resolved.filename) {
        Ok(subtitles) => {
            info!(
                "successfully loaded {} subtitle(s) from {}",
                subtitles.len(),
                resolved.filename
            );
            Ok((subtitles, resolved.filename))
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

    if !report.timestamp_mismatches.is_empty() {
        println!(
            "Timestamp mismatches: {}",
            report.timestamp_mismatches.len()
        );
    }

    if !report.missing_in_target.is_empty() {
        println!(
            "Missing in {}: {}",
            report.target_file,
            report.missing_in_target.len()
        );
    }

    if let Some(offset) = report.detected_offset {
        println!("Index offset detected: {}", offset);
    }

    if !report.extra_in_target.is_empty() {
        println!(
            "Extra in {}: {}",
            report.target_file,
            report.extra_in_target.len()
        );
    }

    println!();

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

    if report.is_perfect() {
        println!("Verification: SUCCESS");
    } else if !report.extra_in_target.is_empty() && !report.has_issues() {
        println!("Verification: WARNING (extra subtitles found)");
    } else {
        println!("Verification: FAILED");
    }
    println!();
}
