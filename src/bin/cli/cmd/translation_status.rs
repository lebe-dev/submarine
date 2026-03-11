use crate::cli::OutputFormat;
use crate::dto::{ChunkDto, TranslationStatusDto};
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::subtitle::model::Subtitle;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use lib::translation_status::service;
use log::{debug, info};

pub fn handle(
    reference: &str,
    translation: &str,
    chunk_size: usize,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!(
        "checking translation status for {} against {}",
        translation, reference
    );

    let (ref_subs, ref_filename) = load_subtitle_file(reference, format)?;
    let (translation_subs, translation_filename) = load_subtitle_file(translation, format)?;

    debug!(
        "loaded {} reference and {} translation subtitles",
        ref_subs.len(),
        translation_subs.len()
    );

    if ref_subs.is_empty() {
        output_error(format, "empty_file", "reference file is empty", None);
        return Err(anyhow::anyhow!(""));
    }

    let report = service::check_translation_status(
        ref_subs,
        ref_filename,
        translation_subs,
        translation_filename,
        chunk_size,
    );

    let dto = TranslationStatusDto {
        ref_file: report.ref_file.clone(),
        translation_file: report.translation_file.clone(),
        total_count: report.total_count,
        translated_count: report.translated_count,
        missing_count: report.missing_count,
        progress_percentage: report.progress_percentage(),
        is_complete: report.is_complete(),
        next_chunk: report.next_chunk.as_ref().map(|c| ChunkDto {
            start_index: c.start_index,
            end_index: c.end_index,
        }),
    };

    output_success(format, &dto, || {
        display_report(&report);
    });

    info!(
        "translation progress: {}/{} ({:.1}%)",
        report.translated_count,
        report.total_count,
        report.progress_percentage()
    );

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

fn display_report(report: &lib::translation_status::model::TranslationStatusReport) {
    let percentage = report.progress_percentage();

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
