use crate::subtitle::model::{Subtitle, SubtitleError};

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
}
