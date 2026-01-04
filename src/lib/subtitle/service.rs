use crate::subtitle::model::{
    Subtitle, SubtitleError, SubtitleIndex, SubtitleText, SubtitleTimestamp, SubtitleUpdate,
    UpdateReport,
};
use crate::subtitle::ports::SubtitleService;
use anyhow::{Context, Result, bail};
use chrono::Local;
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
    ///
    /// This parser expects well-formed SRT files. Files with formatting issues
    /// (empty lines inside blocks, incorrect timestamps, etc.) should be fixed
    /// using `sm doctor` command or manually before parsing.
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

    /// Serialize subtitles to SRT format
    fn serialize_to_srt(subtitles: &[Subtitle]) -> String {
        subtitles
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<String>>()
            .join("\n\n")
    }

    /// Create backup file with timestamp
    fn create_backup(&self, file_path: &Path) -> Result<String, SubtitleError> {
        let timestamp = Local::now().format("%Y-%m-%d_%H-%M-%S");
        let backup_path = format!("{}.{}", file_path.display(), timestamp);

        fs::copy(file_path, &backup_path)
            .map_err(|e| SubtitleError::BackupFailed(e.to_string()))?;

        Ok(backup_path)
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
        // 1. Validate at least one field is specified
        if !update.has_updates() {
            return Err(SubtitleError::NoFieldsToUpdate);
        }

        // 2. Validate filename (path traversal protection)
        Self::validate_filename(filename)?;

        // 3. Build file path and check existence
        let file_path = self.build_file_path(filename);
        if !file_path.exists() {
            return Err(SubtitleError::FileNotFound(file_path.display().to_string()));
        }

        // 4. Parse entire file to validate and load all subtitles
        let mut subtitles = self.get_all(filename)?;

        // 5. Find subtitle with the given index
        let subtitle_pos = subtitles
            .iter()
            .position(|s| *s.index.as_ref() == id)
            .ok_or(SubtitleError::SubtitleNotFound(id))?;

        // 6. Apply update (this performs cross-field validation)
        let updated_subtitle = update
            .apply_to(&subtitles[subtitle_pos])
            .map_err(SubtitleError::ParseError)?;

        // 7. Replace the subtitle in the list
        subtitles[subtitle_pos] = updated_subtitle;

        // 8. Create backup BEFORE writing
        let backup_path = self.create_backup(&file_path)?;

        // 9. Serialize and write back to file
        let content = Self::serialize_to_srt(&subtitles);
        fs::write(&file_path, content).map_err(|e| SubtitleError::WriteFailed(e.to_string()))?;

        // 10. Build report
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
            backup_path,
            subtitle_index: id,
            fields_updated,
        })
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
        assert!(report.backup_path.contains(".srt."));

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

        // Verify backup exists
        let backup_path = PathBuf::from(&report.backup_path);
        assert!(backup_path.exists());

        // Verify backup contains original content
        let backup_content = fs::read_to_string(&backup_path).unwrap();
        assert!(backup_content.contains("Text"));
        assert!(!backup_content.contains("New"));
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
}
