use crate::subtitle::model::{
    AddReport, Subtitle, SubtitleError, SubtitleText, SubtitleTimestamp, SubtitleUpdate,
    UpdateReport,
};

/// Service interface for subtitle operations
pub trait SubtitleService {
    /// Retrieves a subtitle by its index from a SubRip (.srt) file
    ///
    /// # Arguments
    ///
    /// * `filename` - The name of the subtitle file (relative to service's base directory)
    /// * `id` - The subtitle index to search for (must be >= 1)
    ///
    /// # Returns
    ///
    /// * `Ok(Some(Subtitle))` - If a subtitle with the given index is found
    /// * `Ok(None)` - If the file exists and is valid, but no subtitle with the given index exists
    /// * `Err(SubtitleError)` - If the file cannot be read, is malformed, or the filename is invalid
    ///
    /// # SubRip Format Example
    ///
    /// ```text
    /// 1
    /// 00:00:01,436 --> 00:00:03,481
    /// <i>Previously on
    /// "Resident Alien"...</i>
    ///
    /// 2
    /// 00:00:03,481 --> 00:00:05,135
    /// Hello, Harry.
    ///
    /// 5
    /// 00:00:07,790 --> 00:00:09,357
    /// <i>You should be
    /// helping each other,</i>
    /// ```
    ///
    /// Each subtitle entry consists of:
    /// - Line 1: Index number (1-based, gaps allowed)
    /// - Line 2: Timestamps in format `HH:MM:SS,mmm --> HH:MM:SS,mmm`
    /// - Line 3+: Subtitle text (can be multi-line, may contain HTML tags)
    /// - Blank line separates entries
    ///
    /// # Errors
    ///
    /// * `SubtitleError::FileNotFound` - The subtitle file does not exist
    /// * `SubtitleError::IoError` - Failed to read the file
    /// * `SubtitleError::ParseError` - The file format is invalid or corrupted
    /// * `SubtitleError::InvalidPath` - The filename contains path traversal attempts or invalid characters
    ///
    /// # Examples
    ///
    /// ```
    /// use submarine::subtitle::service::SubRipService;
    /// use submarine::subtitle::ports::SubtitleService;
    ///
    /// let service = SubRipService::new("/path/to/subtitles");
    ///
    /// // Get subtitle with index 2
    /// match service.get_by_id("movie.srt", 2) {
    ///     Ok(Some(subtitle)) => {
    ///         println!("Found: {}", subtitle.text.as_ref());
    ///     }
    ///     Ok(None) => {
    ///         println!("Subtitle with index 2 not found");
    ///     }
    ///     Err(e) => {
    ///         eprintln!("Error: {}", e);
    ///     }
    /// }
    /// ```
    fn get_by_id(&self, filename: &str, id: u32) -> Result<Option<Subtitle>, SubtitleError>;

    /// Retrieves all subtitles from a SubRip (.srt) file
    ///
    /// # Arguments
    ///
    /// * `filename` - The name of the subtitle file (relative to service's base directory)
    ///
    /// # Returns
    ///
    /// * `Ok(Vec<Subtitle>)` - Vector of all subtitles in the file
    /// * `Err(SubtitleError)` - If the file cannot be read, is malformed, or the filename is invalid
    ///
    /// # Errors
    ///
    /// * `SubtitleError::FileNotFound` - The subtitle file does not exist
    /// * `SubtitleError::IoError` - Failed to read the file
    /// * `SubtitleError::ParseError` - The file format is invalid or corrupted
    /// * `SubtitleError::InvalidPath` - The filename contains path traversal attempts or invalid characters
    fn get_all(&self, filename: &str) -> Result<Vec<Subtitle>, SubtitleError>;

    /// Updates a subtitle by its index with new values
    ///
    /// # Arguments
    ///
    /// * `filename` - The name of the subtitle file (relative to service's base directory)
    /// * `id` - The subtitle index to update (must be >= 1)
    /// * `update` - Struct containing optional field updates
    ///
    /// # Returns
    ///
    /// * `Ok(UpdateReport)` - Details about the update including backup path
    /// * `Err(SubtitleError)` - If file not found, index doesn't exist, or validation fails
    ///
    /// # Behavior
    ///
    /// - Creates automatic backup before modification with format: filename.YYYY-MM-DD_HH-MM-SS
    /// - Validates all new field values (timestamps must be valid, text non-empty, end > start)
    /// - Writes entire file back in SRT format with proper spacing
    /// - At least one field in `update` must be Some, otherwise returns NoFieldsToUpdate error
    ///
    /// # Errors
    ///
    /// * `SubtitleError::FileNotFound` - The subtitle file does not exist
    /// * `SubtitleError::SubtitleNotFound` - No subtitle with the given index exists
    /// * `SubtitleError::NoFieldsToUpdate` - No fields were specified for update
    /// * `SubtitleError::ParseError` - The file format is invalid or corrupted
    /// * `SubtitleError::InvalidPath` - The filename contains path traversal attempts
    /// * `SubtitleError::BackupFailed` - Failed to create backup file
    /// * `SubtitleError::WriteFailed` - Failed to write updated file
    /// * `SubtitleError::IoError` - Failed to read/write the file
    fn set(
        &self,
        filename: &str,
        id: u32,
        update: SubtitleUpdate,
    ) -> Result<UpdateReport, SubtitleError>;

    /// Adds a new subtitle to the end of an SRT file
    ///
    /// # Arguments
    ///
    /// * `filename` - The name of the subtitle file (relative to service's base directory)
    /// * `start_time` - Start timestamp for the new subtitle
    /// * `end_time` - End timestamp for the new subtitle (must be > start_time)
    /// * `text` - The subtitle text content
    ///
    /// # Returns
    ///
    /// * `Ok(AddReport)` - Details about the addition including backup path and new index
    /// * `Err(SubtitleError)` - If file not found, validation fails, or write fails
    ///
    /// # Behavior
    ///
    /// - Reads all existing subtitles using `get_all()` to validate file format
    /// - Automatically calculates new index as max(existing_indices) + 1
    /// - Creates automatic backup before modification with format: filename.YYYY-MM-DD_HH-MM-SS
    /// - Validates timestamps (must be valid format, end > start)
    /// - Appends new subtitle to the array
    /// - Writes entire file back in SRT format with proper spacing
    ///
    /// # Errors
    ///
    /// * `SubtitleError::FileNotFound` - The subtitle file does not exist
    /// * `SubtitleError::ParseError` - The file format is invalid (run doctor --fix first)
    /// * `SubtitleError::InvalidPath` - The filename contains path traversal attempts
    /// * `SubtitleError::BackupFailed` - Failed to create backup file
    /// * `SubtitleError::WriteFailed` - Failed to write updated file
    /// * `SubtitleError::IoError` - Failed to read/write the file
    fn add(
        &self,
        filename: &str,
        start_time: SubtitleTimestamp,
        end_time: SubtitleTimestamp,
        text: SubtitleText,
    ) -> Result<AddReport, SubtitleError>;
}
