use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct RenameOperation {
    pub original_path: PathBuf,
    pub new_name: String,
    pub collision: bool,
}

impl RenameOperation {
    pub fn new(original_path: PathBuf, new_name: String, collision: bool) -> Self {
        Self {
            original_path,
            new_name,
            collision,
        }
    }

    pub fn original_filename(&self) -> Option<String> {
        self.original_path
            .file_name()
            .and_then(|n| n.to_str())
            .map(|s| s.to_string())
    }
}

#[derive(Debug)]
pub struct RenameReport {
    pub total_files: usize,
    pub renamed: usize,
    pub skipped: usize,
    pub failed: usize,
}

impl RenameReport {
    pub fn new() -> Self {
        Self {
            total_files: 0,
            renamed: 0,
            skipped: 0,
            failed: 0,
        }
    }

    pub fn increment_renamed(&mut self) {
        self.renamed += 1;
    }

    pub fn increment_skipped(&mut self) {
        self.skipped += 1;
    }

    pub fn increment_failed(&mut self) {
        self.failed += 1;
    }
}

impl Default for RenameReport {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(Debug, thiserror::Error)]
pub enum RenameError {
    #[error("no files found matching pattern: {0}")]
    NoFilesFound(String),

    #[error("template rendering failed: {0}")]
    TemplateError(String),

    #[error("file operation failed: {0}")]
    IoError(#[from] std::io::Error),

    #[error("invalid template: {0}")]
    InvalidTemplate(String),

    #[error("glob pattern error: {0}")]
    GlobError(String),
}
