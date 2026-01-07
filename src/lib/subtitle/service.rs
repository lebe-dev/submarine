use crate::subtitle::model::{
    AddReport, Subtitle, SubtitleError, SubtitleIndex, SubtitleText, SubtitleTimestamp,
    SubtitleUpdate, UpdateReport,
};
use crate::subtitle::ports::SubtitleService;
use anyhow::{Context, Result, bail};
use log::{debug, info};
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

    /// Check if a blank line at the given position is a block boundary
    ///
    /// A blank line is a block boundary if the next non-blank line is a valid subtitle index (u32 >= 1)
    fn is_block_boundary(lines: &[&str], current_pos: usize) -> bool {
        let mut next_pos = current_pos + 1;

        while next_pos < lines.len() && lines[next_pos].trim().is_empty() {
            next_pos += 1;
        }

        if next_pos >= lines.len() {
            return true;
        }

        let next_line = lines[next_pos].trim();
        if let Ok(index) = next_line.parse::<u32>() {
            index >= 1
        } else {
            false
        }
    }

    /// Helper to create subtitle with validation
    fn create_subtitle(
        index: u32,
        start_time: u64,
        end_time: u64,
        text_lines: &[&str],
        line_num: usize,
    ) -> Result<Subtitle> {
        let text = text_lines.join("\n");

        let subtitle_index = SubtitleIndex::try_new(index)
            .context(format!("line {}: invalid index value {}", line_num, index))?;

        let subtitle_start =
            SubtitleTimestamp::try_new(chrono::Duration::milliseconds(start_time as i64))
                .context(format!("line {}: invalid start timestamp", line_num))?;

        let subtitle_end =
            SubtitleTimestamp::try_new(chrono::Duration::milliseconds(end_time as i64))
                .context(format!("line {}: invalid end timestamp", line_num))?;

        let subtitle_text = SubtitleText::try_new(text).context(format!(
            "line {}: invalid subtitle text (empty or whitespace-only)",
            line_num
        ))?;

        Subtitle::new(subtitle_index, subtitle_start, subtitle_end, subtitle_text)
            .context(format!("line {}: failed to create subtitle", line_num))
    }

    /// Parse SRT file content into a vector of subtitles
    ///
    /// This parser handles multi-line subtitles with blank lines within the text content.
    /// Blank lines are treated as block separators only when followed by a valid subtitle index.
    fn parse_srt_file(content: &str) -> Result<Vec<Subtitle>> {
        #[derive(Debug, PartialEq)]
        enum ParserState {
            ExpectingIndex,
            ExpectingTimestamp,
            ReadingText,
        }

        let mut subtitles = Vec::new();
        let lines: Vec<&str> = content.lines().collect();
        let mut state = ParserState::ExpectingIndex;

        let mut current_index: u32 = 0;
        let mut current_start: u64 = 0;
        let mut current_end: u64 = 0;
        let mut text_lines: Vec<&str> = Vec::new();
        let mut line_num;

        for (i, line) in lines.iter().enumerate() {
            line_num = i + 1;

            match state {
                ParserState::ExpectingIndex => {
                    if line.trim().is_empty() {
                        continue;
                    }

                    current_index = line.trim().parse().context(format!(
                        "line {}: expected subtitle index, found '{}'",
                        line_num,
                        line.trim()
                    ))?;

                    state = ParserState::ExpectingTimestamp;
                }

                ParserState::ExpectingTimestamp => {
                    let parts: Vec<&str> = line.split(" --> ").collect();
                    if parts.len() != 2 {
                        bail!(
                            "line {}: invalid timestamp format '{}' (expected 'HH:MM:SS,mmm --> HH:MM:SS,mmm')",
                            line_num,
                            line
                        );
                    }

                    current_start = Subtitle::parse_timestamp(parts[0].trim())
                        .context(format!(
                            "line {}: failed to parse start timestamp '{}'",
                            line_num, parts[0]
                        ))?
                        .num_milliseconds() as u64;

                    current_end = Subtitle::parse_timestamp(parts[1].trim())
                        .context(format!(
                            "line {}: failed to parse end timestamp '{}'",
                            line_num, parts[1]
                        ))?
                        .num_milliseconds() as u64;

                    text_lines.clear();
                    state = ParserState::ReadingText;
                }

                ParserState::ReadingText => {
                    // check if this blank line is a block boundary
                    if line.trim().is_empty() && Self::is_block_boundary(&lines, i) {
                        // finalize current subtitle
                        let subtitle = Self::create_subtitle(
                            current_index,
                            current_start,
                            current_end,
                            &text_lines,
                            line_num,
                        )?;
                        subtitles.push(subtitle);
                        state = ParserState::ExpectingIndex;
                    } else {
                        // add line to text (including blank lines within text)
                        text_lines.push(line);
                    }
                }
            }
        }

        // handle last subtitle if file doesn't end with blank line
        if state == ParserState::ReadingText {
            if text_lines.is_empty() {
                bail!("line {}: incomplete subtitle (missing text)", lines.len());
            }
            let subtitle = Self::create_subtitle(
                current_index,
                current_start,
                current_end,
                &text_lines,
                lines.len(),
            )?;
            subtitles.push(subtitle);
        }

        Ok(subtitles)
    }

    /// Serialize subtitles to SRT format
    fn serialize_to_srt(subtitles: &[Subtitle]) -> String {
        subtitles
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<String>>()
            .join("\n\n")
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

    fn get_all(&self, filename: &str) -> Result<Vec<Subtitle>, SubtitleError> {
        Self::validate_filename(filename)?;

        let file_path = self.build_file_path(filename);

        if !file_path.exists() {
            return Err(SubtitleError::FileNotFound(file_path.display().to_string()));
        }

        let content = fs::read_to_string(&file_path)?;

        Self::parse_srt_file(&content).map_err(SubtitleError::ParseError)
    }

    fn set(
        &self,
        filename: &str,
        id: u32,
        update: SubtitleUpdate,
    ) -> Result<UpdateReport, SubtitleError> {
        if !update.has_updates() {
            return Err(SubtitleError::NoFieldsToUpdate);
        }

        Self::validate_filename(filename)?;

        let file_path = self.build_file_path(filename);
        if !file_path.exists() {
            return Err(SubtitleError::FileNotFound(file_path.display().to_string()));
        }

        let mut subtitles = self.get_all(filename)?;

        let subtitle_pos = subtitles
            .iter()
            .position(|s| *s.index.as_ref() == id)
            .ok_or(SubtitleError::SubtitleNotFound(id))?;

        let updated_subtitle = update
            .apply_to(&subtitles[subtitle_pos])
            .map_err(SubtitleError::ParseError)?;

        subtitles[subtitle_pos] = updated_subtitle;

        let content = Self::serialize_to_srt(&subtitles);
        fs::write(&file_path, content).map_err(|e| SubtitleError::WriteFailed(e.to_string()))?;

        let mut fields_updated = Vec::new();
        if update.start_time.is_some() {
            fields_updated.push("start_time".to_string());
        }
        if update.end_time.is_some() {
            fields_updated.push("end_time".to_string());
        }
        if update.text.is_some() {
            fields_updated.push("text".to_string());
        }

        Ok(UpdateReport {
            file_path: file_path.display().to_string(),
            subtitle_index: id,
            fields_updated,
        })
    }

    fn add(
        &self,
        filename: &str,
        start_time: SubtitleTimestamp,
        end_time: SubtitleTimestamp,
        text: SubtitleText,
    ) -> Result<AddReport, SubtitleError> {
        info!("adding subtitle '{}' to file: {}", text.as_ref(), filename);
        debug!(
            "subtitle timestamps: {} --> {}",
            Subtitle::format_timestamp(start_time.as_ref()),
            Subtitle::format_timestamp(end_time.as_ref())
        );
        debug!("subtitle text length: {}", text.as_ref().len());

        debug!("validating filename");
        Self::validate_filename(filename)?;

        let file_path = self.build_file_path(filename);
        debug!("checking file existence: {:?}", file_path);

        let mut subtitles = if file_path.exists() {
            debug!("reading existing subtitles");
            let subs = self.get_all(filename)?;
            debug!("found {} existing subtitles", subs.len());
            subs
        } else {
            info!("file does not exist, will create new file: {:?}", file_path);
            Vec::new()
        };

        let new_index = subtitles
            .iter()
            .map(|s| *s.index.as_ref())
            .max()
            .unwrap_or(0)
            + 1;
        debug!("calculated new index: {}", new_index);

        if !subtitles.is_empty() {
            debug!("validating timestamp against existing subtitles");
            let max_end_time = subtitles.iter().map(|s| s.end_time.as_ref()).max().unwrap();
            debug!(
                "last subtitle ends at: {}",
                Subtitle::format_timestamp(max_end_time)
            );

            if start_time.as_ref() < max_end_time {
                debug!("timestamp conflict detected");
                return Err(SubtitleError::TimestampConflict {
                    last_end: Subtitle::format_timestamp(max_end_time),
                    new_start: Subtitle::format_timestamp(start_time.as_ref()),
                });
            }
            debug!("timestamp validation passed");
        }

        debug!("creating new subtitle with index {}", new_index);
        let subtitle_index = SubtitleIndex::try_new(new_index)
            .map_err(|e| SubtitleError::ParseError(anyhow::anyhow!("invalid index: {}", e)))?;

        let new_subtitle = Subtitle::new(subtitle_index, start_time, end_time, text)
            .map_err(SubtitleError::ParseError)?;

        subtitles.push(new_subtitle);
        debug!(
            "added subtitle to collection, total count: {}",
            subtitles.len()
        );

        debug!("serializing subtitles to srt format");
        let content = Self::serialize_to_srt(&subtitles);

        debug!("writing file: {:?}", file_path);
        fs::write(&file_path, content).map_err(|e| SubtitleError::WriteFailed(e.to_string()))?;

        info!("subtitle added successfully with index {}", new_index);
        Ok(AddReport {
            file_path: file_path.display().to_string(),
            new_index,
            total_subtitles: subtitles.len(),
        })
    }

    fn write_all(&self, filename: &str, subtitles: &[Subtitle]) -> Result<(), SubtitleError> {
        debug!("writing {} subtitles to {}", subtitles.len(), filename);

        Self::validate_filename(filename)?;
        let file_path = self.build_file_path(filename);

        let mut content = String::new();
        for subtitle in subtitles {
            content.push_str(&subtitle.to_string());
            content.push_str("\n\n");
        }

        debug!("writing file: {:?}", file_path);
        fs::write(&file_path, content.trim_end())
            .map_err(|e| SubtitleError::WriteFailed(format!("failed to write file: {}", e)))?;

        info!("wrote {} subtitles to {}", subtitles.len(), filename);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Duration;
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

    // Helper function for creating test subtitles
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
        assert!(err_msg.contains("expected subtitle index") || err_msg.contains("invalid digit"));
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
        assert!(err_msg.contains("incomplete subtitle") || err_msg.contains("missing text"));
    }

    // Tests for blank lines within subtitle text

    #[test]
    fn test_parse_srt_with_blank_line_in_text() {
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst line\n\nSecond line";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subtitles = result.unwrap();
        assert_eq!(subtitles.len(), 1);
        assert_eq!(subtitles[0].text.as_ref(), "First line\n\nSecond line");
    }

    #[test]
    fn test_parse_srt_multiple_blank_lines_in_text() {
        let content = "1\n00:00:01,000 --> 00:00:02,000\nLine 1\n\n\nLine 2";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subtitles = result.unwrap();
        assert_eq!(subtitles.len(), 1);
        assert_eq!(subtitles[0].text.as_ref(), "Line 1\n\n\nLine 2");
    }

    #[test]
    fn test_parse_srt_file_ends_without_blank_line() {
        let content = "1\n00:00:01,000 --> 00:00:02,000\nText without trailing blank";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subtitles = result.unwrap();
        assert_eq!(subtitles.len(), 1);
        assert_eq!(subtitles[0].text.as_ref(), "Text without trailing blank");
    }

    #[test]
    fn test_parse_srt_with_leading_blank_lines() {
        let content = "\n\n1\n00:00:01,000 --> 00:00:02,000\nText";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subtitles = result.unwrap();
        assert_eq!(subtitles.len(), 1);
        assert_eq!(subtitles[0].text.as_ref(), "Text");
    }

    #[test]
    fn test_parse_srt_multiple_subtitles_with_blank_text() {
        let content = "1\n00:00:01,000 --> 00:00:02,000\nText A\n\nMore A\n\n2\n00:00:03,000 --> 00:00:04,000\nText B";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subs = result.unwrap();
        assert_eq!(subs.len(), 2);
        assert_eq!(subs[0].text.as_ref(), "Text A\n\nMore A");
        assert_eq!(subs[1].text.as_ref(), "Text B");
    }

    #[test]
    fn test_parse_srt_multiple_blank_lines_between_blocks() {
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subs = result.unwrap();
        assert_eq!(subs.len(), 2);
        assert_eq!(subs[0].text.as_ref(), "First");
        assert_eq!(subs[1].text.as_ref(), "Second");
    }

    #[test]
    fn test_parse_srt_complex_multiline_with_blanks() {
        let content = "1\n00:00:01,000 --> 00:00:03,000\nLine 1\n\nLine 2\nLine 3\n\nLine 4\n\n2\n00:00:04,000 --> 00:00:06,000\nSimple";

        let result = SubRipService::parse_srt_file(content);
        assert!(result.is_ok());

        let subs = result.unwrap();
        assert_eq!(subs.len(), 2);
        assert_eq!(subs[0].text.as_ref(), "Line 1\n\nLine 2\nLine 3\n\nLine 4");
        assert_eq!(subs[1].text.as_ref(), "Simple");
    }

    // Tests for set functionality

    #[test]
    fn test_serialize_to_srt() {
        let subtitle1 = make_test_subtitle(1, 1000, 2000, "First");
        let subtitle2 = make_test_subtitle(2, 3000, 4000, "Second");

        let subtitles = vec![subtitle1, subtitle2];
        let content = SubRipService::serialize_to_srt(&subtitles);

        let expected =
            "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond";
        assert_eq!(content, expected);
    }

    #[test]
    fn test_set_subtitle_text() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nOriginal text";
        write_test_file(&temp_dir, "test.srt", content);

        let new_text = SubtitleText::try_new("Updated text".to_string()).unwrap();

        let update = SubtitleUpdate {
            start_time: None,
            end_time: None,
            text: Some(new_text),
        };

        let report = service.set("test.srt", 1, update).unwrap();
        assert_eq!(report.subtitle_index, 1);
        assert_eq!(report.fields_updated, vec!["text"]);

        let subtitle = service.get_by_id("test.srt", 1).unwrap().unwrap();
        assert_eq!(subtitle.text.as_ref(), "Updated text");
        assert_eq!(subtitle.start_time.as_ref().num_milliseconds(), 1000);
        assert_eq!(subtitle.end_time.as_ref().num_milliseconds(), 3000);
    }

    #[test]
    fn test_set_subtitle_start_time() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nOriginal text";
        write_test_file(&temp_dir, "test.srt", content);

        let new_start = SubtitleTimestamp::try_new(chrono::Duration::milliseconds(2000)).unwrap();

        let update = SubtitleUpdate {
            start_time: Some(new_start),
            end_time: None,
            text: None,
        };

        let report = service.set("test.srt", 1, update).unwrap();
        assert_eq!(report.subtitle_index, 1);
        assert_eq!(report.fields_updated, vec!["start_time"]);

        let subtitle = service.get_by_id("test.srt", 1).unwrap().unwrap();
        assert_eq!(subtitle.start_time.as_ref().num_milliseconds(), 2000);
        assert_eq!(subtitle.text.as_ref(), "Original text");
    }

    #[test]
    fn test_set_subtitle_all_fields() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nOriginal text";
        write_test_file(&temp_dir, "test.srt", content);

        let update = SubtitleUpdate {
            start_time: Some(
                SubtitleTimestamp::try_new(chrono::Duration::milliseconds(2000)).unwrap(),
            ),
            end_time: Some(
                SubtitleTimestamp::try_new(chrono::Duration::milliseconds(4000)).unwrap(),
            ),
            text: Some(SubtitleText::try_new("New text".to_string()).unwrap()),
        };

        let report = service.set("test.srt", 1, update).unwrap();
        assert_eq!(report.fields_updated.len(), 3);

        let subtitle = service.get_by_id("test.srt", 1).unwrap().unwrap();
        assert_eq!(subtitle.start_time.as_ref().num_milliseconds(), 2000);
        assert_eq!(subtitle.end_time.as_ref().num_milliseconds(), 4000);
        assert_eq!(subtitle.text.as_ref(), "New text");
    }

    #[test]
    fn test_set_subtitle_not_found() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nText";
        write_test_file(&temp_dir, "test.srt", content);

        let update = SubtitleUpdate {
            start_time: None,
            end_time: None,
            text: Some(SubtitleText::try_new("New".to_string()).unwrap()),
        };

        let result = service.set("test.srt", 99, update);
        assert!(matches!(result, Err(SubtitleError::SubtitleNotFound(99))));
    }

    #[test]
    fn test_set_subtitle_no_fields() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nText";
        write_test_file(&temp_dir, "test.srt", content);

        let update = SubtitleUpdate {
            start_time: None,
            end_time: None,
            text: None,
        };

        let result = service.set("test.srt", 1, update);
        assert!(matches!(result, Err(SubtitleError::NoFieldsToUpdate)));
    }

    #[test]
    fn test_set_subtitle_invalid_time_order() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nText";
        write_test_file(&temp_dir, "test.srt", content);

        // Set end time to be less than current start time
        let update = SubtitleUpdate {
            start_time: None,
            end_time: Some(
                SubtitleTimestamp::try_new(chrono::Duration::milliseconds(500)).unwrap(),
            ),
            text: None,
        };

        let result = service.set("test.srt", 1, update);
        assert!(result.is_err());
    }

    #[test]
    fn test_set_subtitle_backup_created() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nText";
        write_test_file(&temp_dir, "test.srt", content);

        let update = SubtitleUpdate {
            start_time: None,
            end_time: None,
            text: Some(SubtitleText::try_new("New".to_string()).unwrap()),
        };

        let report = service.set("test.srt", 1, update).unwrap();

        // Verify fields were updated
        assert_eq!(report.subtitle_index, 1);
        assert_eq!(report.fields_updated, vec!["text"]);
    }

    #[test]
    fn test_set_subtitle_preserves_other_subtitles() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond\n\n3\n00:00:05,000 --> 00:00:06,000\nThird";
        write_test_file(&temp_dir, "test.srt", content);

        let update = SubtitleUpdate {
            start_time: None,
            end_time: None,
            text: Some(SubtitleText::try_new("MODIFIED".to_string()).unwrap()),
        };

        service.set("test.srt", 2, update).unwrap();

        // Verify all three subtitles still exist
        let all = service.get_all("test.srt").unwrap();
        assert_eq!(all.len(), 3);

        // Verify only subtitle 2 was modified
        assert_eq!(all[0].text.as_ref(), "First");
        assert_eq!(all[1].text.as_ref(), "MODIFIED");
        assert_eq!(all[2].text.as_ref(), "Third");
    }

    #[test]
    fn test_set_subtitle_multiline_text() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:03,000\nOriginal";
        write_test_file(&temp_dir, "test.srt", content);

        let update = SubtitleUpdate {
            start_time: None,
            end_time: None,
            text: Some(SubtitleText::try_new("Line 1\nLine 2\nLine 3".to_string()).unwrap()),
        };

        service.set("test.srt", 1, update).unwrap();

        let subtitle = service.get_by_id("test.srt", 1).unwrap().unwrap();
        assert_eq!(subtitle.line_count(), 3);
        assert_eq!(subtitle.text.as_ref(), "Line 1\nLine 2\nLine 3");
    }

    // Tests for add functionality

    #[test]
    fn test_add_to_empty_file() {
        let (service, temp_dir) = create_test_service();
        write_test_file(&temp_dir, "empty.srt", "");

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(1000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(2000)).unwrap();
        let text = SubtitleText::try_new("First subtitle".to_string()).unwrap();

        let report = service.add("empty.srt", start, end, text).unwrap();
        assert_eq!(report.new_index, 1);
        assert_eq!(report.total_subtitles, 1);

        let subtitles = service.get_all("empty.srt").unwrap();
        assert_eq!(subtitles.len(), 1);
        assert_eq!(*subtitles[0].index.as_ref(), 1);
        assert_eq!(subtitles[0].text.as_ref(), "First subtitle");
    }

    #[test]
    fn test_add_to_existing_file() {
        let (service, temp_dir) = create_test_service();
        let content =
            "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond";
        write_test_file(&temp_dir, "test.srt", content);

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(6000)).unwrap();
        let text = SubtitleText::try_new("Third".to_string()).unwrap();

        let report = service.add("test.srt", start, end, text).unwrap();
        assert_eq!(report.new_index, 3); // max(1,2) + 1 = 3
        assert_eq!(report.total_subtitles, 3);

        let subtitles = service.get_all("test.srt").unwrap();
        assert_eq!(subtitles.len(), 3);
        assert_eq!(*subtitles[2].index.as_ref(), 3);
        assert_eq!(subtitles[2].text.as_ref(), "Third");
    }

    #[test]
    fn test_add_with_gap_in_indices() {
        let (service, temp_dir) = create_test_service();
        let content =
            "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n5\n00:00:03,000 --> 00:00:04,000\nFifth";
        write_test_file(&temp_dir, "gaps.srt", content);

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(6000)).unwrap();
        let text = SubtitleText::try_new("Sixth".to_string()).unwrap();

        let report = service.add("gaps.srt", start, end, text).unwrap();
        assert_eq!(report.new_index, 6); // max(1,5) + 1 = 6

        let subtitles = service.get_all("gaps.srt").unwrap();
        assert_eq!(subtitles.len(), 3);
        assert_eq!(*subtitles[2].index.as_ref(), 6);
    }

    #[test]
    fn test_add_invalid_timestamps() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst";
        write_test_file(&temp_dir, "test.srt", content);

        // End before start - should fail
        let start = SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(3000)).unwrap();
        let text = SubtitleText::try_new("Bad".to_string()).unwrap();

        let result = service.add("test.srt", start, end, text);
        assert!(result.is_err());
    }

    #[test]
    fn test_add_multiline_text() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst";
        write_test_file(&temp_dir, "test.srt", content);

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(3000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap();
        let text = SubtitleText::try_new("Line 1\nLine 2\nLine 3".to_string()).unwrap();

        let report = service.add("test.srt", start, end, text).unwrap();
        assert_eq!(report.new_index, 2);

        let subtitle = service.get_by_id("test.srt", 2).unwrap().unwrap();
        assert_eq!(subtitle.line_count(), 3);
        assert_eq!(subtitle.text.as_ref(), "Line 1\nLine 2\nLine 3");
    }

    #[test]
    fn test_add_backup_created() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:01,000 --> 00:00:02,000\nFirst";
        write_test_file(&temp_dir, "test.srt", content);

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(3000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(5000)).unwrap();
        let text = SubtitleText::try_new("Second".to_string()).unwrap();

        let report = service.add("test.srt", start, end, text).unwrap();

        // Verify the subtitle was added
        assert_eq!(report.new_index, 2);
        assert_eq!(report.total_subtitles, 2);
    }

    #[test]
    fn test_add_creates_new_file() {
        let (service, temp_dir) = create_test_service();

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(1000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(2000)).unwrap();
        let text = SubtitleText::try_new("First subtitle".to_string()).unwrap();

        let file_path = temp_dir.path().join("new_file.srt");
        assert!(!file_path.exists());

        let result = service.add("new_file.srt", start, end, text);
        assert!(result.is_ok());

        let report = result.unwrap();
        assert_eq!(report.new_index, 1);
        assert_eq!(report.total_subtitles, 1);

        // Verify file was created
        assert!(file_path.exists());

        // Verify content
        let content = fs::read_to_string(&file_path).unwrap();
        assert!(content.contains("1\n"));
        assert!(content.contains("00:00:01,000 --> 00:00:02,000"));
        assert!(content.contains("First subtitle"));
    }

    #[test]
    fn test_add_malformed_file() {
        let (service, temp_dir) = create_test_service();
        write_test_file(&temp_dir, "bad.srt", "INVALID CONTENT\nNOT SRT FORMAT");

        let start = SubtitleTimestamp::try_new(Duration::milliseconds(1000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(2000)).unwrap();
        let text = SubtitleText::try_new("Text".to_string()).unwrap();

        let result = service.add("bad.srt", start, end, text);
        assert!(matches!(result, Err(SubtitleError::ParseError(_))));
    }

    #[test]
    fn test_add_timestamp_before_last() {
        let (service, temp_dir) = create_test_service();
        let content =
            "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond";
        write_test_file(&temp_dir, "test.srt", content);

        // Try to add subtitle that starts before the last one ends (at 00:00:07,000)
        let start = SubtitleTimestamp::try_new(Duration::milliseconds(6000)).unwrap(); // 00:00:06,000
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(8000)).unwrap();
        let text = SubtitleText::try_new("Too early".to_string()).unwrap();

        let result = service.add("test.srt", start, end, text);
        assert!(matches!(
            result,
            Err(SubtitleError::TimestampConflict { .. })
        ));

        if let Err(SubtitleError::TimestampConflict {
            last_end,
            new_start,
        }) = result
        {
            assert_eq!(last_end, "00:00:07,000");
            assert_eq!(new_start, "00:00:06,000");
        }
    }

    #[test]
    fn test_add_timestamp_overlapping() {
        let (service, temp_dir) = create_test_service();
        let content = "1\n00:00:10,000 --> 00:00:15,000\nFirst";
        write_test_file(&temp_dir, "test.srt", content);

        // Try to add subtitle that overlaps (starts at 00:00:14,000, last ends at 00:00:15,000)
        let start = SubtitleTimestamp::try_new(Duration::milliseconds(14000)).unwrap();
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(16000)).unwrap();
        let text = SubtitleText::try_new("Overlapping".to_string()).unwrap();

        let result = service.add("test.srt", start, end, text);
        assert!(matches!(
            result,
            Err(SubtitleError::TimestampConflict { .. })
        ));
    }

    #[test]
    fn test_add_timestamp_exactly_after() {
        let (service, temp_dir) = create_test_service();
        let content =
            "1\n00:00:01,000 --> 00:00:02,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond";
        write_test_file(&temp_dir, "test.srt", content);

        // Add subtitle that starts exactly when the last one ends (at 00:00:07,000)
        let start = SubtitleTimestamp::try_new(Duration::milliseconds(7000)).unwrap(); // 00:00:07,000
        let end = SubtitleTimestamp::try_new(Duration::milliseconds(9000)).unwrap();
        let text = SubtitleText::try_new("Exactly after".to_string()).unwrap();

        let result = service.add("test.srt", start, end, text);
        assert!(result.is_ok());

        let report = result.unwrap();
        assert_eq!(report.new_index, 3);
        assert_eq!(report.total_subtitles, 3);
    }
}
