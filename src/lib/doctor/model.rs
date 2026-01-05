use std::fmt::{Display, Formatter, Result as FmtResult};
use thiserror::Error;

/// Types of validation issues that can be detected
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum IssueType {
    /// Empty line found inside a subtitle block
    EmptyLineInBlock,
    /// Multiple consecutive empty lines between blocks
    MultipleEmptyLines { count: usize },
    /// Invalid timestamp (end < start)
    InvalidTimestamp { start: String, end: String },
    /// Missing required field (index, timestamp, or text)
    MissingField { field: String },
    /// Insufficient lines in block (need at least 3)
    InsufficientLines { found: usize },
    /// Invalid index format or value
    InvalidIndex { value: String, reason: String },
    /// Malformed timestamp format
    MalformedTimestamp { value: String, reason: String },
    /// Empty or whitespace-only text
    EmptyText,
}

/// Severity level for issues
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum Severity {
    /// Warning: issue doesn't prevent parsing but should be fixed
    Warning,
    /// Error: issue prevents correct parsing
    Error,
}

/// A single validation issue found in the file
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ValidationIssue {
    /// Line number where the issue was found (1-based)
    pub line_number: usize,
    /// Subtitle block number (if applicable, 1-based)
    pub block_number: Option<usize>,
    /// Type of issue
    pub issue_type: IssueType,
    /// Severity level
    pub severity: Severity,
    /// Additional context or description
    pub context: Option<String>,
}

/// Result of diagnostic operation
#[derive(Debug, Clone)]
pub struct DiagnosticReport {
    /// Path to the file that was diagnosed
    pub file_path: String,
    /// Total number of lines in the file
    pub total_lines: usize,
    /// Total number of subtitle blocks detected
    pub total_blocks: usize,
    /// List of all issues found
    pub issues: Vec<ValidationIssue>,
    /// Whether the file can be parsed successfully
    pub is_parsable: bool,
}

/// Result of fix operation
#[derive(Debug, Clone)]
pub struct FixReport {
    /// Path to the original file
    pub original_path: String,
    /// Path to the fixed file (same as original)
    pub fixed_path: String,
    /// Number of issues fixed
    pub issues_fixed: usize,
    /// Number of issues that couldn't be fixed
    pub issues_unfixable: usize,
    /// List of unfixable issues
    pub unfixable_issues: Vec<ValidationIssue>,
    /// Whether the fixed file can be parsed successfully
    pub validation_success: bool,
}

/// Custom error type for doctor operations
#[derive(Error, Debug)]
pub enum DoctorError {
    #[error("file not found: {0}")]
    FileNotFound(String),

    #[error("i/o error: {0}")]
    IoError(#[from] std::io::Error),

    #[error("invalid file path: {0}")]
    InvalidPath(String),

    #[error("backup creation failed: {0}")]
    BackupFailed(String),

    #[error("failed to validate fixed file: {0}")]
    ValidationFailed(String),
}

impl DiagnosticReport {
    /// Get count of errors (vs warnings)
    pub fn error_count(&self) -> usize {
        self.issues
            .iter()
            .filter(|i| i.severity == Severity::Error)
            .count()
    }

    /// Get count of warnings
    pub fn warning_count(&self) -> usize {
        self.issues
            .iter()
            .filter(|i| i.severity == Severity::Warning)
            .count()
    }

    /// Check if file has any issues
    pub fn has_issues(&self) -> bool {
        !self.issues.is_empty()
    }

    /// Check if file has any errors
    pub fn has_errors(&self) -> bool {
        self.error_count() > 0
    }
}

impl Display for IssueType {
    fn fmt(&self, f: &mut Formatter<'_>) -> FmtResult {
        match self {
            IssueType::EmptyLineInBlock => write!(f, "Empty line inside block"),
            IssueType::MultipleEmptyLines { count } => {
                write!(f, "Multiple empty lines ({} consecutive)", count)
            }
            IssueType::InvalidTimestamp { start, end } => {
                write!(f, "Invalid timestamp: end ({}) <= start ({})", end, start)
            }
            IssueType::MissingField { field } => write!(f, "Missing field: {}", field),
            IssueType::InsufficientLines { found } => {
                write!(f, "Insufficient lines (found {}, need at least 3)", found)
            }
            IssueType::InvalidIndex { value, reason } => {
                write!(f, "Invalid index '{}': {}", value, reason)
            }
            IssueType::MalformedTimestamp { value, reason } => {
                write!(f, "Malformed timestamp '{}': {}", value, reason)
            }
            IssueType::EmptyText => write!(f, "Empty or whitespace-only text"),
        }
    }
}

impl Display for ValidationIssue {
    fn fmt(&self, f: &mut Formatter<'_>) -> FmtResult {
        let severity_symbol = match self.severity {
            Severity::Error => "❌",
            Severity::Warning => "⚠️",
        };

        if let Some(block_num) = self.block_number {
            write!(
                f,
                "{} Line {}, Block {}: {}",
                severity_symbol, self.line_number, block_num, self.issue_type
            )
        } else {
            write!(
                f,
                "{} Line {}: {}",
                severity_symbol, self.line_number, self.issue_type
            )
        }
    }
}
