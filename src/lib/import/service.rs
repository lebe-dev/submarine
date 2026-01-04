use crate::import::model::CsvSubtitleRow;
use crate::import::ports::ImportService;
use crate::subtitle::model::SubtitleError;
use csv::ReaderBuilder;
use log::debug;
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
}
