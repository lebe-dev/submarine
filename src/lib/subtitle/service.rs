use crate::subtitle::model::{
    Subtitle, SubtitleError, SubtitleIndex, SubtitleText, SubtitleTimestamp,
};
use crate::subtitle::ports::SubtitleService;
use anyhow::{Context, Result, bail};
use std::fs;
use std::path::{Path, PathBuf};

/// File-based implementation of SubtitleService for SubRip (.srt) format
pub struct SubRipService {
    base_dir: PathBuf,
}

impl SubRipService {
    /// Create a new SubRipService with the specified base directory
    pub fn new<P: AsRef<Path>>(base_dir: P) -> Self {
        SubRipService {
            base_dir: base_dir.as_ref().to_path_buf(),
        }
    }

    /// Validate filename to prevent path traversal attacks
    fn validate_filename(filename: &str) -> Result<(), SubtitleError> {
        if filename.is_empty() {
            return Err(SubtitleError::InvalidPath(
                "filename cannot be empty".to_string(),
            ));
        }

        if filename.contains("..") || filename.contains('/') || filename.contains('\\') {
            return Err(SubtitleError::InvalidPath(format!(
                "invalid filename (path traversal detected): {}",
                filename
            )));
        }

        Ok(())
    }

    /// Build the full path to the subtitle file
    fn build_file_path(&self, filename: &str) -> PathBuf {
        self.base_dir.join(filename)
    }

    /// Parse SRT file content into a vector of subtitles
    fn parse_srt_file(content: &str) -> Result<Vec<Subtitle>> {
        let mut subtitles = Vec::new();

        let blocks: Vec<&str> = content.split("\n\n").collect();

        for (block_num, block) in blocks.iter().enumerate() {
            let trimmed_block = block.trim();

            if trimmed_block.is_empty() {
                continue;
            }

            let lines: Vec<&str> = trimmed_block.lines().collect();

            if lines.len() < 3 {
                bail!(
                    "block {}: insufficient lines (expected at least 3, got {})",
                    block_num + 1,
                    lines.len()
                );
            }

            let index: u32 = lines[0].parse().context(format!(
                "block {}: failed to parse index from '{}'",
                block_num + 1,
                lines[0]
            ))?;

            let timestamp_parts: Vec<&str> = lines[1].split(" --> ").collect();
            if timestamp_parts.len() != 2 {
                bail!(
                    "block {}: invalid timestamp format '{}' (expected 'HH:MM:SS,mmm --> HH:MM:SS,mmm')",
                    block_num + 1,
                    lines[1]
                );
            }

            let start_time =
                Subtitle::parse_timestamp(timestamp_parts[0].trim()).context(format!(
                    "block {}: failed to parse start timestamp '{}'",
                    block_num + 1,
                    timestamp_parts[0]
                ))?;

            let end_time =
                Subtitle::parse_timestamp(timestamp_parts[1].trim()).context(format!(
                    "block {}: failed to parse end timestamp '{}'",
                    block_num + 1,
                    timestamp_parts[1]
                ))?;

            let text = lines[2..].join("\n");

            let subtitle_index = SubtitleIndex::try_new(index).context(format!(
                "block {}: invalid index value {}",
                block_num + 1,
                index
            ))?;

            let subtitle_start = SubtitleTimestamp::try_new(start_time)
                .context(format!("block {}: invalid start timestamp", block_num + 1))?;

            let subtitle_end = SubtitleTimestamp::try_new(end_time)
                .context(format!("block {}: invalid end timestamp", block_num + 1))?;

            let subtitle_text = SubtitleText::try_new(text).context(format!(
                "block {}: invalid subtitle text (empty or whitespace-only)",
                block_num + 1
            ))?;

            let subtitle =
                Subtitle::new(subtitle_index, subtitle_start, subtitle_end, subtitle_text)
                    .context(format!(
                        "block {}: failed to create subtitle",
                        block_num + 1
                    ))?;

            subtitles.push(subtitle);
        }

        Ok(subtitles)
    }
}

impl SubtitleService for SubRipService {
    fn get_by_id(&self, filename: &str, id: u32) -> Result<Option<Subtitle>, SubtitleError> {
        Self::validate_filename(filename)?;

        let file_path = self.build_file_path(filename);

        if !file_path.exists() {
            return Err(SubtitleError::FileNotFound(file_path.display().to_string()));
        }

        let content = fs::read_to_string(&file_path)?;

        let subtitles = Self::parse_srt_file(&content).map_err(SubtitleError::ParseError)?;

        let result = subtitles
            .into_iter()
            .find(|subtitle| *subtitle.index.as_ref() == id);

        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::TempDir;

    // Helper to create test service with temp directory
    fn create_test_service() -> (SubRipService, TempDir) {
        let temp_dir = TempDir::new().unwrap();
        let service = SubRipService::new(temp_dir.path());
        (service, temp_dir)
    }

    // Helper to write test file
    fn write_test_file(dir: &TempDir, filename: &str, content: &str) {
        let path = dir.path().join(filename);
        let mut file = std::fs::File::create(path).unwrap();
        file.write_all(content.as_bytes()).unwrap();
    }

    #[test]
    fn test_get_subtitle_by_id_found() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nFirst subtitle\n\n\
                       2\n00:00:03,500 --> 00:00:05,500\nSecond subtitle";
        write_test_file(&temp_dir, "test.srt", content);

        let result = service.get_by_id("test.srt", 2).unwrap();
        assert!(result.is_some());

        let subtitle = result.unwrap();
        assert_eq!(*subtitle.index.as_ref(), 2);
        assert_eq!(subtitle.text.as_ref(), "Second subtitle");
    }

    #[test]
    fn test_get_subtitle_by_id_not_found() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nFirst subtitle";
        write_test_file(&temp_dir, "test.srt", content);

        let result = service.get_by_id("test.srt", 99).unwrap();
        assert!(result.is_none());
    }

    #[test]
    fn test_file_not_found() {
        let (service, _temp_dir) = create_test_service();

        let result = service.get_by_id("nonexistent.srt", 1);
        assert!(matches!(result, Err(SubtitleError::FileNotFound(_))));
    }

    #[test]
    fn test_parse_error() {
        let (service, temp_dir) = create_test_service();
        let content = "1\nINVALID FORMAT\nBroken";
        write_test_file(&temp_dir, "broken.srt", content);

        let result = service.get_by_id("broken.srt", 1);
        assert!(matches!(result, Err(SubtitleError::ParseError(_))));
    }

    #[test]
    fn test_invalid_path_traversal() {
        let (service, _temp_dir) = create_test_service();

        let result = service.get_by_id("../etc/passwd", 1);
        assert!(matches!(result, Err(SubtitleError::InvalidPath(_))));
    }

    #[test]
    fn test_invalid_path_absolute() {
        let (service, _temp_dir) = create_test_service();

        let result = service.get_by_id("/etc/passwd", 1);
        assert!(matches!(result, Err(SubtitleError::InvalidPath(_))));
    }

    #[test]
    fn test_invalid_path_empty() {
        let (service, _temp_dir) = create_test_service();

        let result = service.get_by_id("", 1);
        assert!(matches!(result, Err(SubtitleError::InvalidPath(_))));
    }

    #[test]
    fn test_empty_file() {
        let (service, temp_dir) = create_test_service();
        write_test_file(&temp_dir, "empty.srt", "");

        let result = service.get_by_id("empty.srt", 1).unwrap();
        assert!(result.is_none());
    }

    #[test]
    fn test_complex_subtitle_with_html() {
        let (service, temp_dir) = create_test_service();
        let content =
            "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>";
        write_test_file(&temp_dir, "complex.srt", content);

        let result = service.get_by_id("complex.srt", 1).unwrap();
        assert!(result.is_some());

        let subtitle = result.unwrap();
        assert!(subtitle.has_html_tags());
        assert_eq!(subtitle.line_count(), 2);
    }

    #[test]
    fn test_subtitle_with_gap_in_indices() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n\
                       5\n00:00:03,000 --> 00:00:04,000\nFifth";
        write_test_file(&temp_dir, "gaps.srt", content);

        let result = service.get_by_id("gaps.srt", 5).unwrap();
        assert!(result.is_some());

        let result_missing = service.get_by_id("gaps.srt", 3).unwrap();
        assert!(result_missing.is_none());
    }

    // Tests for parse_srt_file method

    #[test]
    fn test_parse_srt_simple() {
        let content = "1\n00:00:01,000 --> 00:00:03,000\nFirst subtitle\n\n\
                       2\n00:00:03,500 --> 00:00:05,500\nSecond subtitle\n\n\
                       3\n00:00:06,000 --> 00:00:08,000\nThird subtitle";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 3);
        assert_eq!(*result[0].index.as_ref(), 1);
        assert_eq!(result[0].text.as_ref(), "First subtitle");
        assert_eq!(*result[1].index.as_ref(), 2);
        assert_eq!(result[1].text.as_ref(), "Second subtitle");
        assert_eq!(*result[2].index.as_ref(), 3);
        assert_eq!(result[2].text.as_ref(), "Third subtitle");
    }

    #[test]
    fn test_parse_srt_multiline_text() {
        let content =
            "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 1);
        assert_eq!(*result[0].index.as_ref(), 1);
        assert_eq!(
            result[0].text.as_ref(),
            "<i>Previously on\n\"Resident Alien\"...</i>"
        );
        assert_eq!(result[0].line_count(), 2);
    }

    #[test]
    fn test_parse_srt_with_html() {
        let content = "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>\n\n\
                       2\n00:00:03,481 --> 00:00:05,135\nHello, Harry.";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 2);
        assert!(result[0].has_html_tags());
        assert!(!result[1].has_html_tags());
    }

    #[test]
    fn test_parse_srt_gaps_in_indices() {
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n\
                       5\n00:00:03,000 --> 00:00:04,000\nFifth";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 2);
        assert_eq!(*result[0].index.as_ref(), 1);
        assert_eq!(*result[1].index.as_ref(), 5);
    }

    #[test]
    fn test_parse_srt_empty_file() {
        let content = "";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 0);
    }

    #[test]
    fn test_parse_srt_single_subtitle() {
        let content = "1\n00:00:01,000 --> 00:00:03,000\nSingle subtitle";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 1);
        assert_eq!(*result[0].index.as_ref(), 1);
        assert_eq!(result[0].text.as_ref(), "Single subtitle");
    }

    #[test]
    fn test_parse_srt_invalid_index() {
        let content = "NOT_A_NUMBER\n00:00:01,000 --> 00:00:03,000\nText";

        let result = SubRipService::parse_srt_file(content);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(err_msg.contains("failed to parse index"));
    }

    #[test]
    fn test_parse_srt_invalid_timestamp_format() {
        let content = "1\nINVALID_TIMESTAMP\nText";

        let result = SubRipService::parse_srt_file(content);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(err_msg.contains("invalid timestamp format"));
    }

    #[test]
    fn test_parse_srt_missing_arrow() {
        let content = "1\n00:00:01,000 00:00:03,000\nText";

        let result = SubRipService::parse_srt_file(content);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(err_msg.contains("invalid timestamp format"));
    }

    #[test]
    fn test_parse_srt_empty_text() {
        // Test case where text line exists but contains only an empty string
        // Note: we need something after the empty text line to prevent trim from removing it
        let content =
            "1\n00:00:01,000 --> 00:00:03,000\n\n\n2\n00:00:04,000 --> 00:00:05,000\nValid text";

        let result = SubRipService::parse_srt_file(content);

        // The first subtitle should fail due to empty text
        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(
            err_msg.contains("invalid subtitle text")
                || err_msg.contains("empty")
                || err_msg.contains("insufficient")
        );
    }

    #[test]
    fn test_parse_srt_index_zero() {
        let content = "0\n00:00:01,000 --> 00:00:03,000\nText";

        let result = SubRipService::parse_srt_file(content);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(err_msg.contains("invalid index value"));
    }

    #[test]
    fn test_parse_srt_end_before_start() {
        let content = "1\n00:00:05,000 --> 00:00:03,000\nText";

        let result = SubRipService::parse_srt_file(content);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(
            err_msg.contains("End time must be after start time")
                || err_msg.contains("failed to create subtitle")
        );
    }

    #[test]
    fn test_parse_srt_whitespace_handling() {
        let content = "1\n00:00:01,000 --> 00:00:03,000\n  Trimmed text  \n\n\n2\n00:00:04,000 --> 00:00:06,000\nSecond";

        let result = SubRipService::parse_srt_file(content).unwrap();

        assert_eq!(result.len(), 2);
        assert_eq!(result[0].text.as_ref(), "Trimmed text");
    }

    #[test]
    fn test_parse_srt_insufficient_lines() {
        let content = "1\n00:00:01,000 --> 00:00:03,000";

        let result = SubRipService::parse_srt_file(content);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(err_msg.contains("insufficient lines"));
    }
}
