use crate::import::model::{AnchoredSubtitleRow, CsvSubtitleRow};
use crate::import::ports::ImportService;
use crate::subtitle::model::SubtitleError;
use csv::ReaderBuilder;
use log::debug;
use std::fs;
use std::path::Path;

pub struct CsvImportService;

impl CsvImportService {
    pub fn new() -> Self {
        Self
    }

    fn validate_csv_header(
        headers: &csv::StringRecord,
        delimiter: char,
    ) -> Result<(), SubtitleError> {
        let actual = headers
            .iter()
            .collect::<Vec<_>>()
            .join(&delimiter.to_string());

        if headers.len() != 3
            || headers.get(0) != Some("start_time")
            || headers.get(1) != Some("end_time")
            || headers.get(2) != Some("text")
        {
            return Err(SubtitleError::InvalidCsvHeader(
                delimiter.to_string(),
                actual,
            ));
        }

        debug!("csv header validated successfully");
        Ok(())
    }
}

impl ImportService for CsvImportService {
    fn parse_csv_file(
        &self,
        csv_path: &Path,
        delimiter: char,
    ) -> Result<Vec<CsvSubtitleRow>, SubtitleError> {
        debug!(
            "parsing csv file: {:?} with delimiter '{}'",
            csv_path, delimiter
        );

        let mut reader = ReaderBuilder::new()
            .delimiter(delimiter as u8)
            .has_headers(true)
            .from_path(csv_path)
            .map_err(|e| SubtitleError::IoError(e.into()))?;

        let headers = reader.headers().map_err(|e| SubtitleError::CsvParseError {
            line: 1,
            message: format!("failed to read CSV header: {}", e),
        })?;

        debug!("validating csv header");
        Self::validate_csv_header(headers, delimiter)?;

        let mut rows = Vec::new();

        for (idx, result) in reader.records().enumerate() {
            let line_number = idx + 2; // +1 for 0-index, +1 for header

            let record = result.map_err(|e| SubtitleError::CsvParseError {
                line: line_number,
                message: format!("failed to parse CSV record: {}", e),
            })?;

            if record.len() != 3 {
                return Err(SubtitleError::CsvParseError {
                    line: line_number,
                    message: format!("expected 3 columns, got {}", record.len()),
                });
            }

            debug!(
                "parsed csv row {}: start={}, end={}",
                line_number, &record[0], &record[1]
            );

            rows.push(CsvSubtitleRow {
                line_number,
                start_time: record[0].trim().to_string(),
                end_time: record[1].trim().to_string(),
                text: record[2].replace("\\n", "\n"), // Convert \n to actual newlines
            });
        }

        debug!("parsed {} rows from csv file", rows.len());
        Ok(rows)
    }

    fn parse_anchored_file(
        &self,
        _anchored_path: &Path,
    ) -> Result<Vec<AnchoredSubtitleRow>, SubtitleError> {
        Err(SubtitleError::InvalidPath(
            "CsvImportService does not support anchored format parsing".to_string(),
        ))
    }
}

pub struct AnchoredImportService;

impl AnchoredImportService {
    pub fn new() -> Self {
        Self
    }

    /// Parse anchored format line to extract index and first line text
    fn parse_anchored_line(line: &str) -> Option<(u32, String)> {
        let trimmed = line.trim();
        if !trimmed.starts_with('[') {
            return None;
        }

        let close_bracket = trimmed.find(']')?;
        let index_str = &trimmed[1..close_bracket];
        let index = index_str.parse::<u32>().ok()?;

        let text = if close_bracket + 1 < trimmed.len() {
            trimmed[close_bracket + 1..].trim_start().to_string()
        } else {
            String::new()
        };

        Some((index, text))
    }
}

impl ImportService for AnchoredImportService {
    fn parse_csv_file(
        &self,
        _csv_path: &Path,
        _delimiter: char,
    ) -> Result<Vec<CsvSubtitleRow>, SubtitleError> {
        Err(SubtitleError::InvalidPath(
            "AnchoredImportService does not support CSV parsing".to_string(),
        ))
    }

    fn parse_anchored_file(
        &self,
        anchored_path: &Path,
    ) -> Result<Vec<AnchoredSubtitleRow>, SubtitleError> {
        debug!("parsing anchored format file: {:?}", anchored_path);

        let content = fs::read_to_string(anchored_path)?;

        let mut rows = Vec::new();
        let mut current_index: Option<u32> = None;
        let mut current_text = String::new();
        let mut current_line_number = 0;

        for (line_num, line) in content.lines().enumerate() {
            let line_number = line_num + 1;

            if let Some((index, first_line_text)) = Self::parse_anchored_line(line) {
                if let Some(idx) = current_index {
                    if current_text.trim().is_empty() {
                        return Err(SubtitleError::AnchoredParseError {
                            line: current_line_number,
                            message: format!("Subtitle [{}] has empty text", idx),
                        });
                    }
                    rows.push(AnchoredSubtitleRow {
                        line_number: current_line_number,
                        index: idx,
                        text: current_text.trim().to_string(),
                    });
                }

                current_index = Some(index);
                current_text = first_line_text;
                current_line_number = line_number;
            } else {
                if current_index.is_some() {
                    if !current_text.is_empty() {
                        current_text.push('\n');
                    }
                    current_text.push_str(line);
                } else if !line.trim().is_empty() {
                    return Err(SubtitleError::AnchoredParseError {
                        line: line_number,
                        message: "Text line found before any [INDEX] marker".to_string(),
                    });
                }
            }
        }

        if let Some(idx) = current_index {
            if current_text.trim().is_empty() {
                return Err(SubtitleError::AnchoredParseError {
                    line: current_line_number,
                    message: format!("Subtitle [{}] has empty text", idx),
                });
            }
            rows.push(AnchoredSubtitleRow {
                line_number: current_line_number,
                index: idx,
                text: current_text.trim().to_string(),
            });
        }

        if rows.is_empty() {
            return Err(SubtitleError::AnchoredParseError {
                line: 0,
                message: "No valid subtitle entries found in anchored file".to_string(),
            });
        }

        debug!("parsed {} entries from anchored file", rows.len());
        Ok(rows)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_parse_csv_valid() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "start_time|end_time|text").unwrap();
        writeln!(temp_file, "00:00:01,000|00:00:02,000|First subtitle").unwrap();
        writeln!(temp_file, "00:00:03,000|00:00:04,000|Second subtitle").unwrap();
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let rows = service.parse_csv_file(temp_file.path(), '|').unwrap();
        assert_eq!(rows.len(), 2);
        assert_eq!(rows[0].line_number, 2);
        assert_eq!(rows[0].start_time, "00:00:01,000");
        assert_eq!(rows[0].end_time, "00:00:02,000");
        assert_eq!(rows[0].text, "First subtitle");
        assert_eq!(rows[1].line_number, 3);
        assert_eq!(rows[1].text, "Second subtitle");
    }

    #[test]
    fn test_parse_csv_multiline_text() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "start_time|end_time|text").unwrap();
        writeln!(temp_file, "00:00:01,000|00:00:02,000|Line 1\\nLine 2").unwrap();
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let rows = service.parse_csv_file(temp_file.path(), '|').unwrap();
        assert_eq!(rows[0].text, "Line 1\nLine 2");
    }

    #[test]
    fn test_parse_csv_invalid_header() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "wrong|header|format").unwrap();
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let result = service.parse_csv_file(temp_file.path(), '|');
        assert!(matches!(result, Err(SubtitleError::InvalidCsvHeader(_, _))));
    }

    #[test]
    fn test_parse_csv_wrong_column_count() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "start_time|end_time|text").unwrap();
        writeln!(temp_file, "00:00:01,000|00:00:02,000").unwrap(); // Missing text column
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let result = service.parse_csv_file(temp_file.path(), '|');
        assert!(matches!(
            result,
            Err(SubtitleError::CsvParseError { line: 2, .. })
        ));
    }

    #[test]
    fn test_parse_csv_custom_delimiter() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "start_time;end_time;text").unwrap();
        writeln!(temp_file, "00:00:01,000;00:00:02,000;First").unwrap();
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let rows = service.parse_csv_file(temp_file.path(), ';').unwrap();
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].text, "First");
    }

    #[test]
    fn test_parse_csv_trim_whitespace() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "start_time|end_time|text").unwrap();
        writeln!(temp_file, "  00:00:01,000  |  00:00:02,000  |Text").unwrap();
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let rows = service.parse_csv_file(temp_file.path(), '|').unwrap();
        assert_eq!(rows[0].start_time, "00:00:01,000");
        assert_eq!(rows[0].end_time, "00:00:02,000");
    }

    #[test]
    fn test_parse_csv_html_tags() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "start_time|end_time|text").unwrap();
        writeln!(temp_file, "00:00:01,000|00:00:02,000|<i>Italic</i>").unwrap();
        temp_file.flush().unwrap();

        let service = CsvImportService::new();
        let rows = service.parse_csv_file(temp_file.path(), '|').unwrap();
        assert_eq!(rows[0].text, "<i>Italic</i>");
    }

    // ========== Anchored Format Tests ==========

    #[test]
    fn test_parse_anchored_single_line() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "[1] First subtitle").unwrap();
        writeln!(temp_file, "[2] Second subtitle").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let rows = service.parse_anchored_file(temp_file.path()).unwrap();

        assert_eq!(rows.len(), 2);
        assert_eq!(rows[0].index, 1);
        assert_eq!(rows[0].text, "First subtitle");
        assert_eq!(rows[0].line_number, 1);
        assert_eq!(rows[1].index, 2);
        assert_eq!(rows[1].text, "Second subtitle");
        assert_eq!(rows[1].line_number, 2);
    }

    #[test]
    fn test_parse_anchored_multiline() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "[1] Line 1 of sub 1").unwrap();
        writeln!(temp_file, "Line 2 of sub 1").unwrap();
        writeln!(temp_file, "Line 3 of sub 1").unwrap();
        writeln!(temp_file, "[2] Single line").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let rows = service.parse_anchored_file(temp_file.path()).unwrap();

        assert_eq!(rows.len(), 2);
        assert_eq!(
            rows[0].text,
            "Line 1 of sub 1\nLine 2 of sub 1\nLine 3 of sub 1"
        );
        assert_eq!(rows[1].text, "Single line");
    }

    #[test]
    fn test_parse_anchored_with_html() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "[1] <i>Italic text</i>").unwrap();
        writeln!(temp_file, "[2] <b>Bold text</b>").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let rows = service.parse_anchored_file(temp_file.path()).unwrap();

        assert_eq!(rows[0].text, "<i>Italic text</i>");
        assert_eq!(rows[1].text, "<b>Bold text</b>");
    }

    #[test]
    fn test_parse_anchored_invalid_no_bracket() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "Text without bracket").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let result = service.parse_anchored_file(temp_file.path());

        assert!(matches!(
            result,
            Err(SubtitleError::AnchoredParseError { line: 1, .. })
        ));
    }

    #[test]
    fn test_parse_anchored_empty_text() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "[1]").unwrap();
        writeln!(temp_file, "[2] Next").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let result = service.parse_anchored_file(temp_file.path());

        assert!(matches!(
            result,
            Err(SubtitleError::AnchoredParseError { line: 1, .. })
        ));
    }

    #[test]
    fn test_parse_anchored_empty_file() {
        let temp_file = NamedTempFile::new().unwrap();

        let service = AnchoredImportService::new();
        let result = service.parse_anchored_file(temp_file.path());

        assert!(matches!(
            result,
            Err(SubtitleError::AnchoredParseError { line: 0, .. })
        ));
    }

    #[test]
    fn test_parse_anchored_mixed_indices() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "[5] Fifth subtitle").unwrap();
        writeln!(temp_file, "[10] Tenth subtitle").unwrap();
        writeln!(temp_file, "[1] First subtitle").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let rows = service.parse_anchored_file(temp_file.path()).unwrap();

        assert_eq!(rows.len(), 3);
        assert_eq!(rows[0].index, 5);
        assert_eq!(rows[1].index, 10);
        assert_eq!(rows[2].index, 1);
    }

    #[test]
    fn test_parse_anchored_with_blank_lines() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "[1] First").unwrap();
        writeln!(temp_file).unwrap(); // blank line
        writeln!(temp_file, "[2] Second").unwrap();
        temp_file.flush().unwrap();

        let service = AnchoredImportService::new();
        let rows = service.parse_anchored_file(temp_file.path()).unwrap();

        assert_eq!(rows.len(), 2);
        // Blank line becomes part of the first subtitle's text
        assert_eq!(rows[0].text, "First");
        assert_eq!(rows[1].text, "Second");
    }
}
