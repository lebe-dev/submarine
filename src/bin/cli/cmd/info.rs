use crate::cli::OutputFormat;
use crate::dto::{self, InfoDto};
use crate::json_output::{output_error, output_success};
use crate::utils;
use chrono::Duration;
use lib::subtitle::model::Subtitle;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, info};

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

pub fn handle(file: &str, format: &OutputFormat) -> anyhow::Result<()> {
    info!("getting statistics for file: {}", file);

    let resolved = utils::resolve_existing_path(file)?;
    let service = SubRipService::new(resolved.base_dir);

    debug!("retrieving all subtitles for analysis");
    match service.get_all(&resolved.filename) {
        Ok(subtitles) => {
            debug!("found {} subtitles", subtitles.len());

            let stats = calculate_statistics(&subtitles);

            let info_dto = InfoDto {
                file: file.to_string(),
                total_count: stats.total_count,
                total_duration_ms: stats.total_duration.num_milliseconds(),
                total_duration: Subtitle::format_timestamp(&stats.total_duration),
                average_subtitle_duration_ms: stats.average_subtitle_duration.num_milliseconds(),
                average_gap_ms: stats.average_gap.map(|g| g.num_milliseconds()),
                total_characters: stats.total_characters,
                total_characters_no_html: stats.total_characters_no_html,
                total_words: stats.total_words,
                total_lines: stats.total_lines,
                subtitles_with_html: stats.subtitles_with_html,
            };

            output_success(format, &info_dto, || {
                display_statistics(&stats, file);
            });

            info!("statistics displayed successfully");
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

/// Calculate comprehensive statistics from subtitles
fn calculate_statistics(subtitles: &[Subtitle]) -> SubtitleStats {
    debug!("calculating statistics for {} subtitles", subtitles.len());

    if subtitles.is_empty() {
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

    let first_start = subtitles.first().unwrap().start_time.as_ref();
    let last_end = subtitles.last().unwrap().end_time.as_ref();
    let total_duration = *last_end - *first_start;

    let sum_durations: i64 = subtitles
        .iter()
        .map(|s| s.duration().num_milliseconds())
        .sum();
    let average_subtitle_duration = Duration::milliseconds(sum_durations / total_count as i64);

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
            Some(Duration::milliseconds(gap_sum / gap_count as i64))
        } else {
            None
        }
    } else {
        None
    };

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

/// Display statistics in a formatted, user-friendly way
fn display_statistics(stats: &SubtitleStats, filename: &str) {
    println!("Subtitle File Information");
    println!("========================");
    println!();
    println!("File: {}", filename);
    println!();

    if stats.total_count == 0 {
        println!("No subtitles found in file.");
        return;
    }

    println!("Basic Information:");
    println!("  Total subtitles: {}", stats.total_count);
    println!(
        "  Total duration:  {} ({})",
        Subtitle::format_timestamp(&stats.total_duration),
        dto::format_duration_readable(&stats.total_duration)
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
    fn test_calculate_statistics_overlapping_subtitles() {
        let subtitles = vec![
            make_test_subtitle(1, 1000, 3000, "First"),
            make_test_subtitle(2, 2500, 4000, "Second"),
        ];
        let stats = calculate_statistics(&subtitles);

        assert!(stats.average_gap.is_none());
    }
}
