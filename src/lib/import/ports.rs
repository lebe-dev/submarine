use super::model::{AnchoredSubtitleRow, CsvSubtitleRow};
use crate::subtitle::model::SubtitleError;
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

    /// Parse anchored format file and return validated data
    ///
    /// # Arguments
    /// * `anchored_path` - Path to the anchored format file
    ///
    /// # Returns
    /// Vector of parsed anchored rows with line numbers for error reporting
    ///
    /// # Errors
    /// * `SubtitleError::IoError` - Failed to read file
    /// * `SubtitleError::AnchoredParseError` - Invalid anchored format
    /// * `SubtitleError::InvalidAnchoredFormat` - Malformed index or structure
    fn parse_anchored_file(
        &self,
        anchored_path: &Path,
    ) -> Result<Vec<AnchoredSubtitleRow>, SubtitleError>;
}
