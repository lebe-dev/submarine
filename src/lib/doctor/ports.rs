use crate::doctor::model::{DiagnosticReport, DoctorError, FixReport};

/// Service interface for SRT file diagnosis and repair
pub trait DoctorService {
    /// Diagnose a subtitle file for issues
    ///
    /// # Arguments
    /// * `filename` - Name of the file to diagnose (relative to base directory)
    ///
    /// # Returns
    /// * `Ok(DiagnosticReport)` - Report of all issues found
    /// * `Err(DoctorError)` - If file cannot be read or path is invalid
    fn diagnose(&self, filename: &str) -> Result<DiagnosticReport, DoctorError>;

    /// Fix issues in a subtitle file
    ///
    /// # Arguments
    /// * `filename` - Name of the file to fix (relative to base directory)
    ///
    /// # Returns
    /// * `Ok(FixReport)` - Report of fixes applied
    /// * `Err(DoctorError)` - If file cannot be read/written or backup fails
    ///
    /// # Behavior
    /// 1. Creates backup with timestamp: `filename.srt.bak-YYYY-MM-DD-HH-MM-SS`
    /// 2. Removes empty lines inside blocks
    /// 3. Normalizes separators to single empty line between blocks
    /// 4. Skips unfixable blocks (invalid timestamps, missing fields)
    /// 5. Validates result by attempting to parse
    fn fix(&self, filename: &str) -> Result<FixReport, DoctorError>;
}
