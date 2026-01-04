use chrono::Duration;
use lib::subtitle::model::{Subtitle, SubtitleError};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

/// Statistics collected from subtitle file
struct SubtitleStats {
    total_count: usize,
    total_duration: Duration,
    average_subtitle_duration: Duration,
    average_gap: Option<Duration>,
    total_characters: usize,
    total_characters_no_html: usize,
    total_words: usize,
    total_lines: usize,
    subtitles_with_html: usize,
}

pub fn handle(file: &str) -> anyhow::Result<()> {
    info!("getting statistics for file: {}", file);

    debug!("validating and canonicalizing path");
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

    debug!("creating service and retrieving subtitles");
    let service = SubRipService::new(base_dir);

    debug!("retrieving all subtitles for analysis");
    match service.get_all(&filename) {
        Ok(subtitles) => {
            debug!("found {} subtitles", subtitles.len());

            debug!("calculating statistics");
            let stats = calculate_statistics(&subtitles);

            debug!("displaying results");
            display_statistics(&stats, file);

            info!("statistics displayed successfully");
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
                    eprintln!("hint: Try running 'sm doctor --fix {}' first", file);
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

/// Calculate comprehensive statistics from subtitles
fn calculate_statistics(subtitles: &[Subtitle]) -> SubtitleStats {
    debug!("calculating statistics for {} subtitles", subtitles.len());

    if subtitles.is_empty() {
        debug!("no subtitles found, returning empty stats");
        return SubtitleStats {
            total_count: 0,
            total_duration: Duration::zero(),
            average_subtitle_duration: Duration::zero(),
            average_gap: None,
            total_characters: 0,
            total_characters_no_html: 0,
            total_words: 0,
            total_lines: 0,
            subtitles_with_html: 0,
        };
    }

    let total_count = subtitles.len();
    debug!("total count: {}", total_count);

    debug!("calculating total duration from first start to last end");
    let first_start = subtitles.first().unwrap().start_time.as_ref();
    let last_end = subtitles.last().unwrap().end_time.as_ref();
    let total_duration = *last_end - *first_start;
    debug!(
        "total duration: {} ms (from {} to {})",
        total_duration.num_milliseconds(),
        Subtitle::format_timestamp(first_start),
        Subtitle::format_timestamp(last_end)
    );

    debug!("calculating average subtitle duration");
    let sum_durations: i64 = subtitles
        .iter()
        .map(|s| s.duration().num_milliseconds())
        .sum();
    let average_subtitle_duration = Duration::milliseconds(sum_durations / total_count as i64);
    debug!(
        "average subtitle duration: {} ms",
        average_subtitle_duration.num_milliseconds()
    );

    debug!("calculating gaps between subtitles");
    let average_gap = if total_count > 1 {
        let mut gap_sum: i64 = 0;
        let mut gap_count = 0;

        for i in 0..subtitles.len() - 1 {
            let current_end = subtitles[i].end_time.as_ref();
            let next_start = subtitles[i + 1].start_time.as_ref();
            let gap = *next_start - *current_end;

            if gap.num_milliseconds() > 0 {
                gap_sum += gap.num_milliseconds();
                gap_count += 1;
            }
        }

        if gap_count > 0 {
            let avg = Duration::milliseconds(gap_sum / gap_count as i64);
            debug!(
                "average gap: {} ms ({} gaps)",
                avg.num_milliseconds(),
                gap_count
            );
            Some(avg)
        } else {
            debug!("no positive gaps found");
            None
        }
    } else {
        debug!("only one subtitle, no gaps");
        None
    };

    debug!("calculating character and word counts");
    let mut total_characters = 0;
    let mut total_characters_no_html = 0;
    let mut total_words = 0;
    let mut total_lines = 0;
    let mut subtitles_with_html = 0;

    for subtitle in subtitles {
        let text = subtitle.text.as_ref();
        let text_no_html = subtitle.text_without_html();

        total_characters += text.chars().count();
        total_characters_no_html += text_no_html.chars().count();

        let words: Vec<&str> = text_no_html.split_whitespace().collect();
        total_words += words.len();

        total_lines += subtitle.line_count();

        if subtitle.has_html_tags() {
            subtitles_with_html += 1;
        }
    }

    debug!(
        "text stats - chars: {}, chars (no HTML): {}, words: {}, lines: {}, with HTML: {}",
        total_characters, total_characters_no_html, total_words, total_lines, subtitles_with_html
    );

    SubtitleStats {
        total_count,
        total_duration,
        average_subtitle_duration,
        average_gap,
        total_characters,
        total_characters_no_html,
        total_words,
        total_lines,
        subtitles_with_html,
    }
}

/// Format duration to human-readable string (e.g., "1h 23m 45s")
fn format_duration_readable(duration: &Duration) -> String {
    let total_seconds = duration.num_seconds();

    let hours = total_seconds / 3600;
    let minutes = (total_seconds % 3600) / 60;
    let seconds = total_seconds % 60;
    let milliseconds = duration.num_milliseconds() % 1000;

    if hours > 0 {
        format!("{}h {}m {}s", hours, minutes, seconds)
    } else if minutes > 0 {
        format!("{}m {}s", minutes, seconds)
    } else if seconds > 0 {
        format!("{}.{:03}s", seconds, milliseconds)
    } else {
        format!("{}ms", milliseconds)
    }
}

/// Display statistics in a formatted, user-friendly way
fn display_statistics(stats: &SubtitleStats, filename: &str) {
    debug!("displaying statistics");

    println!("Subtitle File Information");
    println!("========================");
    println!();
    println!("File: {}", filename);
    println!();

    if stats.total_count == 0 {
        debug!("handling empty file case");
        println!("No subtitles found in file.");
        return;
    }

    println!("Basic Information:");
    println!("  Total subtitles: {}", stats.total_count);
    println!(
        "  Total duration:  {} ({})",
        Subtitle::format_timestamp(&stats.total_duration),
        format_duration_readable(&stats.total_duration)
    );
    println!();

    println!("Timing Statistics:");
    println!(
        "  Average subtitle duration: {} ({:.2}s)",
        Subtitle::format_timestamp(&stats.average_subtitle_duration),
        stats.average_subtitle_duration.num_milliseconds() as f64 / 1000.0
    );

    if let Some(avg_gap) = stats.average_gap {
        println!(
            "  Average gap between subtitles: {} ({:.2}s)",
            Subtitle::format_timestamp(&avg_gap),
            avg_gap.num_milliseconds() as f64 / 1000.0
        );
    } else {
        println!("  Average gap between subtitles: N/A (no gaps or single subtitle)");
    }
    println!();

    println!("Text Statistics:");
    println!("  Total characters: {}", stats.total_characters);

    if stats.subtitles_with_html > 0 {
        println!(
            "  Total characters (without HTML tags): {}",
            stats.total_characters_no_html
        );
    }

    println!("  Total words: {}", stats.total_words);
    println!("  Total lines: {}", stats.total_lines);

    if stats.total_count > 0 {
        println!(
            "  Average words per subtitle: {:.1}",
            stats.total_words as f64 / stats.total_count as f64
        );
        println!(
            "  Average characters per subtitle: {:.1}",
            stats.total_characters_no_html as f64 / stats.total_count as f64
        );
    }
    println!();

    if stats.subtitles_with_html > 0 {
        println!("Formatting:");
        println!(
            "  Subtitles with HTML tags: {} ({:.1}%)",
            stats.subtitles_with_html,
            (stats.subtitles_with_html as f64 / stats.total_count as f64) * 100.0
        );
        println!();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
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
    fn test_calculate_statistics_empty() {
        let subtitles: Vec<Subtitle> = vec![];
        let stats = calculate_statistics(&subtitles);

        assert_eq!(stats.total_count, 0);
        assert_eq!(stats.total_duration, Duration::zero());
        assert_eq!(stats.total_words, 0);
        assert!(stats.average_gap.is_none());
    }

    #[test]
    fn test_calculate_statistics_single() {
        let subtitles = vec![make_test_subtitle(1, 1000, 3000, "Hello world")];
        let stats = calculate_statistics(&subtitles);

        assert_eq!(stats.total_count, 1);
        assert_eq!(stats.total_duration.num_milliseconds(), 2000);
        assert_eq!(stats.average_subtitle_duration.num_milliseconds(), 2000);
        assert!(stats.average_gap.is_none());
        assert_eq!(stats.total_words, 2);
    }

    #[test]
    fn test_calculate_statistics_multiple() {
        let subtitles = vec![
            make_test_subtitle(1, 1000, 2000, "First"),
            make_test_subtitle(2, 3000, 4000, "Second subtitle"),
        ];
        let stats = calculate_statistics(&subtitles);

        assert_eq!(stats.total_count, 2);
        assert_eq!(stats.total_duration.num_milliseconds(), 3000);
        assert_eq!(stats.average_subtitle_duration.num_milliseconds(), 1000);
        assert!(stats.average_gap.is_some());
        assert_eq!(stats.average_gap.unwrap().num_milliseconds(), 1000);
        assert_eq!(stats.total_words, 3);
    }

    #[test]
    fn test_calculate_statistics_with_html() {
        let subtitles = vec![
            make_test_subtitle(1, 1000, 2000, "<i>Italic text</i>"),
            make_test_subtitle(2, 3000, 4000, "Plain text"),
        ];
        let stats = calculate_statistics(&subtitles);

        assert_eq!(stats.subtitles_with_html, 1);
        assert_eq!(stats.total_characters, 28);
        assert_eq!(stats.total_characters_no_html, 21);
    }

    #[test]
    fn test_calculate_statistics_multiline() {
        let subtitles = vec![make_test_subtitle(1, 1000, 2000, "Line 1\nLine 2\nLine 3")];
        let stats = calculate_statistics(&subtitles);

        assert_eq!(stats.total_lines, 3);
        assert_eq!(stats.total_words, 6);
    }

    #[test]
    fn test_format_duration_readable_hours() {
        let duration = Duration::seconds(3661);
        assert_eq!(format_duration_readable(&duration), "1h 1m 1s");
    }

    #[test]
    fn test_format_duration_readable_minutes() {
        let duration = Duration::seconds(125);
        assert_eq!(format_duration_readable(&duration), "2m 5s");
    }

    #[test]
    fn test_format_duration_readable_seconds() {
        let duration = Duration::milliseconds(2450);
        assert_eq!(format_duration_readable(&duration), "2.450s");
    }

    #[test]
    fn test_format_duration_readable_milliseconds() {
        let duration = Duration::milliseconds(450);
        assert_eq!(format_duration_readable(&duration), "450ms");
    }

    #[test]
    fn test_calculate_statistics_overlapping_subtitles() {
        let subtitles = vec![
            make_test_subtitle(1, 1000, 3000, "First"),
            make_test_subtitle(2, 2500, 4000, "Second"),
        ];
        let stats = calculate_statistics(&subtitles);

        assert!(stats.average_gap.is_none());
    }
}
