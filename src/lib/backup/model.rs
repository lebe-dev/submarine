use thiserror::Error;

#[derive(Error, Debug)]
pub enum BackupError {
    #[error("failed to create backup: {0}")]
    CreationFailed(String),

    #[error("I/O error: {0}")]
    IoError(#[from] std::io::Error),
}
