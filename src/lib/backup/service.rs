use super::model::BackupError;
use super::ports::BackupService;
use chrono::Local;
use log::{debug, info};
use std::fs;
use std::path::{Path, PathBuf};

/// SubRip backup service implementation
pub struct SubRipBackupService;

impl SubRipBackupService {
    pub fn new() -> Self {
        SubRipBackupService
    }
}

impl Default for SubRipBackupService {
    fn default() -> Self {
        Self::new()
    }
}

impl BackupService for SubRipBackupService {
    fn create_backup(&self, file_path: &Path) -> Result<Option<String>, BackupError> {
        debug!("creating backup for file: {:?}", file_path);

        if !file_path.exists() {
            debug!("file does not exist, skipping backup");
            return Ok(None);
        }

        let backups_dir = PathBuf::from("backups");
        debug!("ensuring backups directory exists: {:?}", backups_dir);
        fs::create_dir_all(&backups_dir).map_err(|e| {
            BackupError::CreationFailed(format!("failed to create backups directory: {}", e))
        })?;

        let filename = file_path
            .file_name()
            .ok_or_else(|| BackupError::CreationFailed("invalid file path".to_string()))?;

        let now = Local::now();
        let timestamp = format!(
            "{}.{:06}",
            now.format("%Y-%m-%d_%H-%M-%S"),
            now.timestamp_subsec_micros()
        );
        let backup_filename = format!("{}.{}", filename.to_string_lossy(), timestamp);
        debug!(
            "generated backup filename with timestamp: {}",
            backup_filename
        );

        let backup_path = backups_dir.join(&backup_filename);
        debug!("copying file to backup location: {:?}", backup_path);

        fs::copy(file_path, &backup_path)
            .map_err(|e| BackupError::CreationFailed(e.to_string()))?;

        info!("backup created: {}", backup_path.display());

        Ok(Some(backup_path.display().to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::TempDir;

    #[test]
    fn test_create_backup_for_existing_file() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.srt");
        fs::write(&file_path, "test content").unwrap();

        let service = SubRipBackupService::new();
        let result = service.create_backup(&file_path);

        assert!(result.is_ok());
        let backup_path = result.unwrap();
        assert!(backup_path.is_some());

        let backup_path_str = backup_path.unwrap();
        assert!(backup_path_str.starts_with("backups/test.srt."));
        assert!(PathBuf::from(&backup_path_str).exists());

        // Verify backup content matches original
        let backup_content = fs::read_to_string(&backup_path_str).unwrap();
        assert_eq!(backup_content, "test content");
    }

    #[test]
    fn test_create_backup_for_nonexistent_file() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("nonexistent.srt");

        let service = SubRipBackupService::new();
        let result = service.create_backup(&file_path);

        assert!(result.is_ok());
        assert!(result.unwrap().is_none());
    }

    #[test]
    fn test_backup_creates_directory() {
        // Clean up backups directory if it exists
        let backups_dir = PathBuf::from("backups");
        if backups_dir.exists() {
            fs::remove_dir_all(&backups_dir).ok();
        }

        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.srt");
        fs::write(&file_path, "test").unwrap();

        let service = SubRipBackupService::new();
        let result = service.create_backup(&file_path);

        assert!(result.is_ok());
        assert!(backups_dir.exists());
        assert!(backups_dir.is_dir());
    }

    #[test]
    fn test_backup_filename_format() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("movie.srt");
        fs::write(&file_path, "test").unwrap();

        let service = SubRipBackupService::new();
        let result = service.create_backup(&file_path);

        assert!(result.is_ok());
        let backup_path = result.unwrap().unwrap();

        // Format should be: backups/movie.srt.YYYY-MM-DD_HH-MM-SS.microseconds
        assert!(backup_path.starts_with("backups/movie.srt."));
        assert!(backup_path.contains("_")); // date-time separator
        assert!(backup_path.contains("-")); // date components separator

        // Extract timestamp part (after "movie.srt.")
        let parts: Vec<&str> = backup_path.split("movie.srt.").collect();
        assert_eq!(parts.len(), 2);
        let timestamp = parts[1];

        // Should contain date and time parts with microseconds
        assert!(timestamp.len() >= 26); // YYYY-MM-DD_HH-MM-SS.mmmmmm = 26 chars minimum
    }
}
