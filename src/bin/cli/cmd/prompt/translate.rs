use super::template;
use crate::utils;
use lib::subtitle::model::{Subtitle, SubtitleError};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

/// Format subtitles for prompt output (similar to format_anchored from export.rs)
fn format_subtitles_for_prompt(subtitles: &[Subtitle]) -> String {
    let mut output = String::new();

    for subtitle in subtitles.iter() {
        let index = *subtitle.index.as_ref();
        let text = subtitle.text.as_ref();

        // First line with [INDEX], subsequent lines as-is
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
    language: &str,
    template_path: Option<&str>,
) -> anyhow::Result<()> {
    info!(
        "generating translation prompt for range {} from file: {}",
        range, file
    );

    let (start, end) = utils::parse_range(range)?;
    debug!("parsed range: start={}, end={}", start, end);

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

    let prompt_header = template::render_template(template_path, language)?;
    debug!("template rendered with language: {}", language);

    let service = SubRipService::new(base_dir);

    debug!("retrieving all subtitles for filtering..");
    match service.get_all(&filename) {
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

            debug!(
                "found {} subtitle(s) in range {}-{}",
                range_subtitles.len(),
                start,
                end
            );

            if range_subtitles.is_empty() {
                info!("no subtitles found in range {}-{}", start, end);
                eprintln!("error: no subtitles found in range {}-{}", start, end);
                std::process::exit(1);
            }

            let formatted = format_subtitles_for_prompt(&range_subtitles);
            print!("{}\n{}", prompt_header, formatted);

            info!(
                "successfully generated prompt for {} subtitle(s)",
                range_subtitles.len()
            );
            Ok(())
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    info!("file not found: {}", path);
                    eprintln!("error: file not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: failed to parse subtitle file: {}", err);
                    eprintln!("hint: try running 'sm doctor --fix {}' first", file);
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
    fn test_format_subtitles_single_line() {
        let subs = vec![make_test_subtitle(1, 1000, 2000, "Hello world")];
        let output = format_subtitles_for_prompt(&subs);
        assert_eq!(output, "[1] Hello world\n");
    }

    #[test]
    fn test_format_subtitles_multiline() {
        let subs = vec![make_test_subtitle(1, 1000, 2000, "Line 1\nLine 2\nLine 3")];
        let output = format_subtitles_for_prompt(&subs);
        assert_eq!(output, "[1] Line 1\nLine 2\nLine 3\n");
    }

    #[test]
    fn test_format_subtitles_with_html() {
        let subs = vec![make_test_subtitle(1, 1000, 2000, "<i>Italic text</i>")];
        let output = format_subtitles_for_prompt(&subs);
        assert_eq!(output, "[1] <i>Italic text</i>\n");
    }

    #[test]
    fn test_format_subtitles_multiple() {
        let subs = vec![
            make_test_subtitle(1, 1000, 2000, "First"),
            make_test_subtitle(2, 3000, 4000, "Second"),
        ];
        let output = format_subtitles_for_prompt(&subs);
        assert_eq!(output, "[1] First\n[2] Second\n");
    }

    #[test]
    fn test_format_subtitles_preserves_html_multiline() {
        let subs = vec![
            make_test_subtitle(5, 1000, 2000, "<i>Multi\nLine</i>"),
            make_test_subtitle(10, 3000, 4000, "Single"),
        ];
        let output = format_subtitles_for_prompt(&subs);

        assert!(output.contains("[5] <i>Multi"));
        assert!(output.contains("Line</i>"));
        assert!(output.contains("[10] Single"));
    }
}
