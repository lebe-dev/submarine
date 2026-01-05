use super::model::BackupError;
use std::path::Path;

pub trait BackupService {
    /// Creates a backup of the specified file
    ///
    /// # Arguments
    /// * `file_path` - Path to the file to backup
    ///
    /// # Returns
    /// * `Ok(None)` - File doesn't exist (new file, no backup needed)
    /// * `Ok(Some(path))` - Backup created successfully, returns path to backup
    /// * `Err(e)` - Failed to create backup
    fn create_backup(&self, file_path: &Path) -> Result<Option<String>, BackupError>;
}
