use anyhow::{Context, Result, bail};
use chrono::Duration;
use nutype::nutype;
use std::fmt::{Display, Formatter, Result as FmtResult};
use thiserror::Error;

/// Custom error type for subtitle operations
#[derive(Error, Debug)]
pub enum SubtitleError {
    #[error("Subtitle file not found: {0}")]
    FileNotFound(String),

    #[error("I/O error reading subtitle file: {0}")]
    IoError(#[from] std::io::Error),

    #[error("Failed to parse subtitle file: {0}")]
    ParseError(#[source] anyhow::Error),

    #[error("Invalid file path: {0}")]
    InvalidPath(String),

    #[error("Subtitle with index {0} not found in file")]
    SubtitleNotFound(u32),

    #[error("No fields specified for update (at least one of --start, --end, or --text required)")]
    NoFieldsToUpdate,

    #[error("Failed to create backup: {0}")]
    BackupFailed(String),

    #[error("Failed to write updated file: {0}")]
    WriteFailed(String),

    #[error(
        "Timestamp conflict: new subtitle (starts at {new_start}) must start at or after the last subtitle ends (at {last_end})"
    )]
    TimestampConflict { last_end: String, new_start: String },

    #[error("CSV parsing error at line {line}: {message}")]
    CsvParseError { line: usize, message: String },

    #[error("Invalid CSV header: expected 'start_time{0}end_time{0}text', got '{1}'")]
    InvalidCsvHeader(String, String),

    #[error(
        "Timestamp overlap detected: subtitle at CSV line {line} (starts at {new_start}) overlaps with existing subtitle (ends at {existing_end})"
    )]
    TimestampOverlap {
        line: usize,
        existing_end: String,
        new_start: String,
    },
}

#[nutype(
    validate(greater_or_equal = 1),
    derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Display, AsRef, Deref)
)]
pub struct SubtitleIndex(u32);

#[nutype(
    validate(predicate = |d| *d >= Duration::zero()),
    derive(Debug, Clone, Copy, PartialEq, Eq, Hash, AsRef, Deref)
)]
pub struct SubtitleTimestamp(Duration);

#[nutype(
    sanitize(trim),
    validate(not_empty),
    derive(Debug, Clone, PartialEq, Eq, Hash, Display, AsRef, Deref)
)]
pub struct SubtitleText(String);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Subtitle {
    pub index: SubtitleIndex,
    pub start_time: SubtitleTimestamp,
    pub end_time: SubtitleTimestamp,
    pub text: SubtitleText,
}

impl Subtitle {
    /// Create a new Subtitle with cross-field validation
    pub fn new(
        index: SubtitleIndex,
        start_time: SubtitleTimestamp,
        end_time: SubtitleTimestamp,
        text: SubtitleText,
    ) -> Result<Self> {
        if end_time.as_ref() <= start_time.as_ref() {
            bail!(
                "end time must be after start time (start: {}ms, end: {}ms)",
                start_time.as_ref().num_milliseconds(),
                end_time.as_ref().num_milliseconds()
            );
        }

        Ok(Subtitle {
            index,
            start_time,
            end_time,
            text,
        })
    }

    /// Parse SRT timestamp format "HH:MM:SS,mmm" to Duration
    pub fn parse_timestamp(s: &str) -> Result<Duration> {
        // Split by comma to separate seconds from milliseconds
        let parts: Vec<&str> = s.split(',').collect();
        if parts.len() != 2 {
            bail!("Invalid timestamp format '{}': expected HH:MM:SS,mmm", s);
        }

        // Parse time part (HH:MM:SS)
        let time_parts: Vec<&str> = parts[0].split(':').collect();
        if time_parts.len() != 3 {
            bail!("Invalid time format '{}': expected HH:MM:SS", parts[0]);
        }

        let hours: i64 = time_parts[0]
            .parse()
            .context(format!("Failed to parse hours from '{}'", s))?;
        let minutes: i64 = time_parts[1]
            .parse()
            .context(format!("Failed to parse minutes from '{}'", s))?;
        let seconds: i64 = time_parts[2]
            .parse()
            .context(format!("Failed to parse seconds from '{}'", s))?;
        let milliseconds: i64 = parts[1]
            .parse()
            .context(format!("Failed to parse milliseconds from '{}'", s))?;

        // Validate ranges
        if minutes >= 60 {
            bail!("Minutes must be < 60, got {} in '{}'", minutes, s);
        }
        if seconds >= 60 {
            bail!("Seconds must be < 60, got {} in '{}'", seconds, s);
        }
        if milliseconds >= 1000 {
            bail!(
                "Milliseconds must be < 1000, got {} in '{}'",
                milliseconds,
                s
            );
        }

        // Calculate total duration
        let total_ms = hours * 3_600_000 + minutes * 60_000 + seconds * 1_000 + milliseconds;

        Ok(Duration::milliseconds(total_ms))
    }

    /// Format Duration to SRT timestamp format "HH:MM:SS,mmm"
    pub fn format_timestamp(duration: &Duration) -> String {
        let total_ms = duration.num_milliseconds();

        let hours = total_ms / 3_600_000;
        let minutes = (total_ms % 3_600_000) / 60_000;
        let seconds = (total_ms % 60_000) / 1_000;
        let milliseconds = total_ms % 1_000;

        format!(
            "{:02}:{:02}:{:02},{:03}",
            hours, minutes, seconds, milliseconds
        )
    }

    /// Get the duration of this subtitle (end - start)
    pub fn duration(&self) -> Duration {
        *self.end_time.as_ref() - *self.start_time.as_ref()
    }

    /// Check if this subtitle contains HTML tags
    pub fn has_html_tags(&self) -> bool {
        self.text.as_ref().contains('<') && self.text.as_ref().contains('>')
    }

    /// Get text without HTML tags (simple strip, not HTML parsing)
    pub fn text_without_html(&self) -> String {
        let mut result = String::new();
        let mut in_tag = false;

        for ch in self.text.as_ref().chars() {
            match ch {
                '<' => in_tag = true,
                '>' => in_tag = false,
                _ if !in_tag => result.push(ch),
                _ => {}
            }
        }

        result
    }

    /// Get number of lines in the subtitle text
    pub fn line_count(&self) -> usize {
        self.text.as_ref().lines().count()
    }
}

impl Display for Subtitle {
    fn fmt(&self, f: &mut Formatter<'_>) -> FmtResult {
        // Line 1: Index
        writeln!(f, "{}", self.index)?;

        // Line 2: Timestamp range
        writeln!(
            f,
            "{} --> {}",
            Self::format_timestamp(self.start_time.as_ref()),
            Self::format_timestamp(self.end_time.as_ref())
        )?;

        // Line 3+: Text content (may be multi-line)
        write!(f, "{}", self.text)?;

        Ok(())
    }
}

/// Represents partial update to a subtitle
#[derive(Debug, Clone)]
pub struct SubtitleUpdate {
    pub start_time: Option<SubtitleTimestamp>,
    pub end_time: Option<SubtitleTimestamp>,
    pub text: Option<SubtitleText>,
}

impl SubtitleUpdate {
    /// Check if at least one field is specified
    pub fn has_updates(&self) -> bool {
        self.start_time.is_some() || self.end_time.is_some() || self.text.is_some()
    }

    /// Apply this update to an existing subtitle, creating a new one
    pub fn apply_to(&self, subtitle: &Subtitle) -> Result<Subtitle> {
        let new_start = self.start_time.unwrap_or(subtitle.start_time);
        let new_end = self.end_time.unwrap_or(subtitle.end_time);
        let new_text = self.text.clone().unwrap_or_else(|| subtitle.text.clone());

        // Use Subtitle::new() to get cross-field validation (end > start)
        Subtitle::new(subtitle.index, new_start, new_end, new_text)
    }
}

/// Report from a successful update operation
#[derive(Debug, Clone)]
pub struct UpdateReport {
    pub file_path: String,
    pub subtitle_index: u32,
    pub fields_updated: Vec<String>,
}

/// Report from a successful add operation
#[derive(Debug, Clone)]
pub struct AddReport {
    pub file_path: String,
    pub new_index: u32,
    pub total_subtitles: usize,
}

/// Report from a successful import operation
#[derive(Debug, Clone)]
pub struct ImportReport {
    pub file_path: String,
    pub backup_path: String,
    pub imported_count: usize,
    pub total_subtitles: usize,
    pub start_index: u32,
    pub end_index: u32,
}

#[cfg(test)]
mod tests {
    use super::*;

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
    fn test_subtitle_index_validation() {
        assert!(SubtitleIndex::try_new(1).is_ok());
        assert!(SubtitleIndex::try_new(999).is_ok());
        assert!(SubtitleIndex::try_new(0).is_err());
    }

    #[test]
    fn test_subtitle_timestamp_validation() {
        assert!(SubtitleTimestamp::try_new(Duration::zero()).is_ok());
        assert!(SubtitleTimestamp::try_new(Duration::milliseconds(1000)).is_ok());
        assert!(SubtitleTimestamp::try_new(Duration::milliseconds(-1)).is_err());
    }

    #[test]
    fn test_subtitle_text_validation() {
        assert!(SubtitleText::try_new("Hello".to_string()).is_ok());
        assert!(SubtitleText::try_new("   ".to_string()).is_err());
        assert!(SubtitleText::try_new("".to_string()).is_err());
    }

    #[test]
    fn test_subtitle_text_sanitization() {
        let text = SubtitleText::try_new("  Hello  ".to_string()).unwrap();
        assert_eq!(text.as_ref(), "Hello");
    }

    // Test 1: Timestamp parsing
    #[test]
    fn test_parse_timestamp() {
        let result = Subtitle::parse_timestamp("00:00:01,436").unwrap();
        assert_eq!(result.num_milliseconds(), 1436);

        let result = Subtitle::parse_timestamp("01:23:45,678").unwrap();
        assert_eq!(result.num_milliseconds(), 5025678);
    }

    // Test 2: Timestamp formatting
    #[test]
    fn test_format_timestamp() {
        let duration = Duration::milliseconds(1436);
        assert_eq!(Subtitle::format_timestamp(&duration), "00:00:01,436");

        let duration = Duration::milliseconds(5025678);
        assert_eq!(Subtitle::format_timestamp(&duration), "01:23:45,678");
    }

    // Test 3: Round-trip timestamp conversion
    #[test]
    fn test_timestamp_roundtrip() {
        let original = "01:23:45,678";
        let duration = Subtitle::parse_timestamp(original).unwrap();
        let formatted = Subtitle::format_timestamp(&duration);
        assert_eq!(original, formatted);
    }

    // Test 4: Parse invalid timestamps
    #[test]
    fn test_parse_invalid_timestamp() {
        assert!(Subtitle::parse_timestamp("invalid").is_err());
        assert!(Subtitle::parse_timestamp("00:60:00,000").is_err()); // invalid minutes
        assert!(Subtitle::parse_timestamp("00:00:60,000").is_err()); // invalid seconds
        assert!(Subtitle::parse_timestamp("00:00:00,1000").is_err()); // invalid ms
        assert!(Subtitle::parse_timestamp("00:00:00.000").is_err()); // period instead of comma
    }

    // Test 5: Subtitle creation with validation
    #[test]
    fn test_new_subtitle_valid() {
        let subtitle = Subtitle::new(
            SubtitleIndex::try_new(1).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(1436)).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(3481)).unwrap(),
            SubtitleText::try_new("Hello, Harry.".to_string()).unwrap(),
        )
        .unwrap();

        assert_eq!(*subtitle.index.as_ref(), 1);
        assert_eq!(subtitle.start_time.as_ref().num_milliseconds(), 1436);
        assert_eq!(subtitle.end_time.as_ref().num_milliseconds(), 3481);
    }

    // Test 6: Validation - end time must be after start time
    #[test]
    fn test_new_subtitle_invalid_time_order() {
        let result = Subtitle::new(
            SubtitleIndex::try_new(1).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap(),
            SubtitleTimestamp::try_new(Duration::milliseconds(1000)).unwrap(),
            SubtitleText::try_new("Text".to_string()).unwrap(),
        );
        assert!(result.is_err());
    }

    // Test 11: Display formatting
    #[test]
    fn test_display_format() {
        let subtitle =
            make_test_subtitle(1, 1436, 3481, "<i>Previously on\n\"Resident Alien\"...</i>");

        let expected =
            "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>";
        assert_eq!(subtitle.to_string(), expected);
    }

    // Test 13: Duration calculation
    #[test]
    fn test_subtitle_duration() {
        let subtitle = make_test_subtitle(1, 1436, 3481, "Text");

        assert_eq!(subtitle.duration().num_milliseconds(), 2045);
    }

    // Test 14: HTML tag detection
    #[test]
    fn test_has_html_tags() {
        let with_tags = make_test_subtitle(1, 1000, 2000, "<i>Italic text</i>");
        assert!(with_tags.has_html_tags());

        let without_tags = make_test_subtitle(2, 1000, 2000, "Plain text");
        assert!(!without_tags.has_html_tags());
    }

    // Test 15: HTML stripping
    #[test]
    fn test_text_without_html() {
        let subtitle =
            make_test_subtitle(1, 1000, 2000, "<i>Previously on\n\"Resident Alien\"...</i>");

        assert_eq!(
            subtitle.text_without_html(),
            "Previously on\n\"Resident Alien\"..."
        );
    }

    // Test 16: Clone and PartialEq
    #[test]
    fn test_clone_and_eq() {
        let subtitle1 = make_test_subtitle(1, 1000, 2000, "Text");

        let subtitle2 = subtitle1.clone();
        assert_eq!(subtitle1, subtitle2);
    }

    // Test 18: Zero-padded formatting
    #[test]
    fn test_zero_padding() {
        let duration = Duration::milliseconds(5); // 00:00:00,005
        assert_eq!(Subtitle::format_timestamp(&duration), "00:00:00,005");

        let duration = Duration::milliseconds(60005); // 00:01:00,005
        assert_eq!(Subtitle::format_timestamp(&duration), "00:01:00,005");
    }

    // Test 19: Line count with single line
    #[test]
    fn test_line_count_single() {
        let subtitle = make_test_subtitle(1, 1000, 2000, "Single line");
        assert_eq!(subtitle.line_count(), 1);
    }
}
