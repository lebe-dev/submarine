use lib::subtitle::model::{Subtitle, SubtitleError};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use lib::translation_status::model::TranslationStatusReport;
use lib::translation_status::service;
use log::{debug, error, info};
use std::path::Path;

/// Entry point for the translation-status command
///
/// Loads reference and translation SRT files and checks translation progress.
/// The reference file is the authoritative source of what needs translation.
pub fn handle(reference: &str, translation: &str, chunk_size: usize) -> anyhow::Result<()> {
    info!(
        "checking translation status for {} against {}",
        translation, reference
    );

    let (ref_subs, ref_filename) = load_subtitle_file(reference)?;
    let (translation_subs, translation_filename) = load_subtitle_file(translation)?;

    debug!(
        "loaded {} reference and {} translation subtitles",
        ref_subs.len(),
        translation_subs.len()
    );

    // Check for empty reference file
    if ref_subs.is_empty() {
        error!("reference file is empty");
        eprintln!("error: reference file is empty");
        std::process::exit(1);
    }

    let report = service::check_translation_status(
        ref_subs,
        ref_filename,
        translation_subs,
        translation_filename,
        chunk_size,
    );

    display_report(&report);

    info!(
        "translation progress: {}/{} ({:.1}%)",
        report.translated_count,
        report.total_count,
        report.progress_percentage()
    );

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

/// Display the translation status report to the user
fn display_report(report: &TranslationStatusReport) {
    let percentage = report.progress_percentage();

    // Show exactly 100% only when translation is complete
    // Otherwise show one decimal place to avoid misleading rounding
    if report.is_complete() {
        println!(
            "Progress: {}/{} (100%)",
            report.translated_count, report.total_count
        );
    } else {
        println!(
            "Progress: {}/{} ({:.1}%)",
            report.translated_count, report.total_count, percentage
        );
    }

    if let Some(chunk) = &report.next_chunk {
        println!("Next chunk: {}-{}", chunk.start_index, chunk.end_index);
    } else if report.is_complete() {
        println!("Translation complete!");
    }
}
