use crate::cli::ExportFormat;
use lib::subtitle::model::{Subtitle, SubtitleError};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

/// Parse range string in format "START-END" into (start, end) tuple
fn parse_range(range: &str) -> anyhow::Result<(u32, u32)> {
    let parts: Vec<&str> = range.split('-').collect();

    if parts.len() != 2 {
        error!("invalid range format: {}", range);
        eprintln!(
            "error: Invalid range format '{}'. Expected format: START-END (e.g., 1-50)",
            range
        );
        std::process::exit(1);
    }

    let start = parts[0].trim().parse::<u32>().map_err(|_| {
        error!("invalid start index in range: {}", parts[0]);
        eprintln!(
            "error: Invalid start index '{}'. Must be a positive number",
            parts[0]
        );
        std::process::exit(1);
    })?;

    let end = parts[1].trim().parse::<u32>().map_err(|_| {
        error!("invalid end index in range: {}", parts[1]);
        eprintln!(
            "error: Invalid end index '{}'. Must be a positive number",
            parts[1]
        );
        std::process::exit(1);
    })?;

    if start < 1 {
        error!("invalid start index: must be >= 1");
        eprintln!("error: Start index must be >= 1, got {}", start);
        std::process::exit(1);
    }

    if end < 1 {
        error!("invalid end index: must be >= 1");
        eprintln!("error: End index must be >= 1, got {}", end);
        std::process::exit(1);
    }

    if start > end {
        error!("invalid range: start {} > end {}", start, end);
        eprintln!(
            "error: Start index must be <= end index (got {} > {})",
            start, end
        );
        std::process::exit(1);
    }

    Ok((start, end))
}

/// Validate that the requested range is within available subtitle indices
fn validate_range_boundaries(start: u32, end: u32, subtitles: &[Subtitle]) -> anyhow::Result<()> {
    if subtitles.is_empty() {
        error!("cannot export from empty file");
        eprintln!("error: File contains no subtitles");
        std::process::exit(1);
    }

    let min_index = subtitles.iter().map(|s| *s.index.as_ref()).min().unwrap();

    let max_index = subtitles.iter().map(|s| *s.index.as_ref()).max().unwrap();

    debug!(
        "file contains indices {}-{}, requested range {}-{}",
        min_index, max_index, start, end
    );

    if start > max_index {
        error!(
            "start index {} exceeds maximum available index {}",
            start, max_index
        );
        eprintln!(
            "error: Start index {} is beyond the last subtitle (index {})",
            start, max_index
        );
        std::process::exit(1);
    }

    if end < min_index {
        error!(
            "end index {} is before minimum available index {}",
            end, min_index
        );
        eprintln!(
            "error: End index {} is before the first subtitle (index {})",
            end, min_index
        );
        std::process::exit(1);
    }

    Ok(())
}

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

pub fn handle(file: &str, range: &str, format: ExportFormat) -> anyhow::Result<()> {
    info!(
        "exporting subtitles from range {} from file: {}",
        range, file
    );

    let (start, end) = parse_range(range)?;
    debug!("parsed range: start={}, end={}", start, end);

    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

    if file_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("Failed to get current directory: {}", e))?;

        let resolved = current_dir.join(file_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("Failed to resolve file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("Failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", file_path);
            return Err(anyhow::anyhow!(
                "Invalid file path: path traversal not allowed"
            ));
        }
    }

    let canonical_path = file_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("Failed to resolve file path: {}", e))?;
    debug!("canonical path: {:?}", canonical_path);

    let base_dir = canonical_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("Invalid file path"))?
        .to_path_buf();
    debug!("base directory: {:?}", base_dir);

    let filename = canonical_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("Invalid file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("Invalid UTF-8 in filename"))?
        .to_string();
    debug!("filename: {}", filename);

    let service = SubRipService::new(base_dir);

    debug!("retrieving all subtitles for filtering..");
    match service.get_all(&filename) {
        Ok(subtitles) => {
            debug!("found {} total subtitles", subtitles.len());

            validate_range_boundaries(start, end, &subtitles)?;

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
                eprintln!("error: No subtitles found in range {}-{}", start, end);
                std::process::exit(1);
            }

            let output = match format {
                ExportFormat::Anchored => format_anchored(&range_subtitles),
            };

            print!("{}", output);
            info!(
                "successfully exported {} subtitle(s)",
                range_subtitles.len()
            );
            Ok(())
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    info!("file not found: {}", path);
                    eprintln!("error: File not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: Invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: Failed to parse subtitle file: {}", err);
                }
                SubtitleError::IoError(err) => {
                    error!("i/o error: {}", err);
                    eprintln!("error: Failed to read file: {}", err);
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
    fn test_parse_range_valid() {
        let (start, end) = parse_range("1-50").unwrap();
        assert_eq!(start, 1);
        assert_eq!(end, 50);
    }

    #[test]
    fn test_parse_range_single() {
        let (start, end) = parse_range("42-42").unwrap();
        assert_eq!(start, 42);
        assert_eq!(end, 42);
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
