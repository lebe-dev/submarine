use crate::cli::OutputFormat;
use crate::dto::SubtitleDto;
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, info};
use serde::Serialize;

#[derive(Serialize)]
struct GetRangeDto {
    subtitles: Vec<SubtitleDto>,
    count: usize,
    range_start: u32,
    range_end: u32,
}

pub fn handle(file: &str, index_str: &str, format: &OutputFormat) -> anyhow::Result<()> {
    if index_str.contains('-') {
        debug!("detected range format, processing range");
        return handle_range(file, index_str, format);
    }

    let index = index_str.parse::<u32>().map_err(|_| {
        anyhow::anyhow!(
            "Invalid index '{}'. Must be a positive number or range (e.g., 120-123)",
            index_str
        )
    })?;

    info!("getting subtitle {} from file: {}", index, file);

    let resolved = utils::resolve_existing_path(file)?;
    let service = SubRipService::new(resolved.base_dir);

    debug!("retrieving subtitle by id: {}..", index);
    match service.get_by_id(&resolved.filename, index) {
        Ok(Some(subtitle)) => {
            info!("subtitle {} found successfully", index);
            let dto = SubtitleDto::from_subtitle(&subtitle);
            output_success(format, &dto, || {
                println!("{}", subtitle);
            });
            Ok(())
        }
        Ok(None) => {
            output_error(
                format,
                "subtitle_not_found",
                &format!("subtitle with index {} not found in {}", index, file),
                None,
            );
            Err(anyhow::anyhow!(""))
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

fn handle_range(file: &str, range: &str, format: &OutputFormat) -> anyhow::Result<()> {
    info!("getting subtitles in range {} from file: {}", range, file);

    let (start, end) = utils::parse_range(range)?;
    debug!("parsed range: start={}, end={}", start, end);

    let resolved = utils::resolve_existing_path(file)?;
    let service = SubRipService::new(resolved.base_dir);

    debug!("retrieving all subtitles for filtering..");
    match service.get_all(&resolved.filename) {
        Ok(subtitles) => {
            debug!("found {} total subtitles", subtitles.len());

            let range_subtitles: Vec<_> = subtitles
                .into_iter()
                .filter(|s| {
                    let index = *s.index.as_ref();
                    index >= start && index <= end
                })
                .collect();

            debug!(
                "found {} subtitle(s) in range {}-{}",
                range_subtitles.len(),
                start,
                end
            );

            if range_subtitles.is_empty() {
                info!("no subtitles found in range {}-{}", start, end);
                let dto = GetRangeDto {
                    subtitles: vec![],
                    count: 0,
                    range_start: start,
                    range_end: end,
                };
                output_success(format, &dto, || {
                    println!("No subtitles found in range {}-{}", start, end);
                });
            } else {
                info!(
                    "found {} subtitle(s) in range {}-{}",
                    range_subtitles.len(),
                    start,
                    end
                );

                let dto = GetRangeDto {
                    subtitles: range_subtitles
                        .iter()
                        .map(SubtitleDto::from_subtitle)
                        .collect(),
                    count: range_subtitles.len(),
                    range_start: start,
                    range_end: end,
                };

                output_success(format, &dto, || {
                    for (i, subtitle) in range_subtitles.iter().enumerate() {
                        print!("{}", subtitle);
                        if i < range_subtitles.len() - 1 {
                            println!("\n");
                        } else {
                            println!();
                        }
                    }
                });
            }

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
