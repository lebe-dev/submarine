use crate::cli::{ExportFormat, OutputFormat};
use crate::dto::{ExportDto, SubtitleDto};
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::subtitle::model::Subtitle;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, info};

/// Format subtitles in anchored format: [INDEX] TEXT
fn format_anchored(subtitles: &[Subtitle]) -> String {
    let mut output = String::new();

    for subtitle in subtitles.iter() {
        let index = *subtitle.index.as_ref();
        let text = subtitle.text.as_ref();

        let mut lines = text.lines();
        if let Some(first_line) = lines.next() {
            output.push_str(&format!("[{}] {}\n", index, first_line));

            for line in lines {
                output.push_str(line);
                output.push('\n');
            }
        }
    }

    output
}

pub fn handle(
    file: &str,
    range: &str,
    export_format: ExportFormat,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!(
        "exporting subtitles from range {} from file: {}",
        range, file
    );

    let (start, end) = utils::parse_range(range)?;
    debug!("parsed range: start={}, end={}", start, end);

    let resolved = utils::resolve_existing_path(file)?;
    let service = SubRipService::new(resolved.base_dir);

    debug!("retrieving all subtitles for filtering..");
    match service.get_all(&resolved.filename) {
        Ok(subtitles) => {
            debug!("found {} total subtitles", subtitles.len());

            utils::validate_range_boundaries(start, end, &subtitles)?;

            let range_subtitles: Vec<_> = subtitles
                .into_iter()
                .filter(|s| {
                    let index = *s.index.as_ref();
                    index >= start && index <= end
                })
                .collect();

            if range_subtitles.is_empty() {
                output_error(
                    format,
                    "no_subtitles_in_range",
                    &format!("No subtitles found in range {}-{}", start, end),
                    None,
                );
                return Err(anyhow::anyhow!(""));
            }

            let format_name = match export_format {
                ExportFormat::Anchored => "anchored",
            };

            let dto = ExportDto {
                subtitles: range_subtitles
                    .iter()
                    .map(SubtitleDto::from_subtitle)
                    .collect(),
                format: format_name.to_string(),
                range_start: start,
                range_end: end,
                count: range_subtitles.len(),
            };

            output_success(format, &dto, || {
                let output = match export_format {
                    ExportFormat::Anchored => format_anchored(&range_subtitles),
                };
                print!("{}", output);
            });

            info!(
                "successfully exported {} subtitle(s)",
                range_subtitles.len()
            );
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

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Duration;
    use lib::subtitle::model::{SubtitleIndex, SubtitleText, SubtitleTimestamp};

    fn make_test_subtitle(index: u32, start_ms: i64, end_ms: i64, text: &str) -> Subtitle {
        Subtitle::new(
            SubtitleIndex::try_new(index).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(start_ms)).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(end_ms)).unwrap(),
            SubtitleText::try_new(text.to_string()).unwrap(),
        )
        .unwrap()
    }

    #[test]
    fn test_format_anchored_single_line() {
        let subs = vec![make_test_subtitle(1, 1000, 2000, "Hello world")];
        let output = format_anchored(&subs);
        assert_eq!(output, "[1] Hello world\n");
    }

    #[test]
    fn test_format_anchored_multiline() {
        let subs = vec![make_test_subtitle(1, 1000, 2000, "Line 1\nLine 2\nLine 3")];
        let output = format_anchored(&subs);
        assert_eq!(output, "[1] Line 1\nLine 2\nLine 3\n");
    }

    #[test]
    fn test_format_anchored_with_html() {
        let subs = vec![make_test_subtitle(1, 1000, 2000, "<i>Italic text</i>")];
        let output = format_anchored(&subs);
        assert_eq!(output, "[1] <i>Italic text</i>\n");
    }

    #[test]
    fn test_format_anchored_multiple() {
        let subs = vec![
            make_test_subtitle(1, 1000, 2000, "First"),
            make_test_subtitle(2, 3000, 4000, "Second"),
        ];
        let output = format_anchored(&subs);
        assert_eq!(output, "[1] First\n[2] Second\n");
    }

    #[test]
    fn test_format_anchored_preserves_structure() {
        let subs = vec![
            make_test_subtitle(5, 1000, 2000, "<i>Multi\nLine</i>"),
            make_test_subtitle(10, 3000, 4000, "Single"),
        ];
        let output = format_anchored(&subs);

        assert!(output.contains("[5] <i>Multi"));
        assert!(output.contains("Line</i>"));
        assert!(output.contains("[10] Single"));

        assert!(!output.contains("-->"));
        assert!(!output.contains("00:00:"));
    }
}
