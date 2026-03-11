use crate::cli::OutputFormat;
use crate::json_output::output_error;
use crate::utils;
use lib::subtitle::model::Subtitle;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, info};

pub fn handle(file1: &str, file2: &str, format: &OutputFormat) -> anyhow::Result<()> {
    // Compare is TUI-only, reject JSON output
    if matches!(format, OutputFormat::Json) {
        output_error(
            format,
            "unsupported_format",
            "The compare command is TUI-only and does not support --output json",
            Some("Use 'sm verify' for machine-readable comparison output"),
        );
        return Err(anyhow::anyhow!(""));
    }

    info!("comparing files: {} and {}", file1, file2);

    let (subtitles1, filename1) = load_subtitle_file(file1, format)?;
    let (subtitles2, filename2) = load_subtitle_file(file2, format)?;

    debug!(
        "loaded {} and {} subtitles",
        subtitles1.len(),
        subtitles2.len()
    );

    crate::output::compare::run_tui(subtitles1, filename1, subtitles2, filename2)?;

    info!("comparison completed");
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
