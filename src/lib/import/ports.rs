use crate::subtitle::model::SubtitleError;
use super::model::CsvSubtitleRow;
use std::path::Path;

/// Service interface for importing subtitles from various sources
pub trait ImportService {
    /// Parse CSV file and return validated subtitle data
    ///
    /// # Arguments
    /// * `csv_path` - Path to the CSV file
    /// * `delimiter` - Column delimiter character (e.g., '|', ';', ',')
    ///
    /// # Returns
    /// Vector of parsed CSV rows with line numbers for error reporting
    ///
    /// # Errors
    /// * `SubtitleError::IoError` - Failed to read file
    /// * `SubtitleError::CsvParseError` - Invalid CSV format or structure
    /// * `SubtitleError::InvalidCsvHeader` - Header doesn't match expected format
    fn parse_csv_file(
        &self,
        csv_path: &Path,
        delimiter: char,
    ) -> Result<Vec<CsvSubtitleRow>, SubtitleError>;
}
