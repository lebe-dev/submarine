use crate::doctor::model::{
    DiagnosticReport, DoctorError, FixReport, IssueType, Severity, ValidationIssue,
};
use crate::doctor::ports::DoctorService;
use crate::subtitle::ports::SubtitleService;
use crate::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::fs;
use std::path::PathBuf;

/// SubRip (.srt) doctor service implementation
pub struct SubRipDoctorService {
    base_dir: PathBuf,
}

/// Parser state machine for line-by-line analysis
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum ParserState {
    ExpectingTimestamp,
    ExpectingText,
    BetweenBlocks,
}

impl SubRipDoctorService {
    /// Create a new SubRip doctor service
    ///
    /// # Arguments
    /// * `base_dir` - Base directory for resolving relative file paths
    pub fn new(base_dir: PathBuf) -> Self {
        Self { base_dir }
    }

    /// Validate filename for path traversal attacks
    fn validate_filename(&self, filename: &str) -> Result<(), DoctorError> {
        if filename.is_empty() {
            return Err(DoctorError::InvalidPath(
                "filename cannot be empty".to_string(),
            ));
        }

        if filename.contains("..") || filename.contains('/') || filename.contains('\\') {
            return Err(DoctorError::InvalidPath(format!(
                "invalid filename (path traversal detected): {}",
                filename
            )));
        }

        Ok(())
    }

    /// Get full path to file
    fn get_file_path(&self, filename: &str) -> Result<PathBuf, DoctorError> {
        self.validate_filename(filename)?;
        let path = self.base_dir.join(filename);

        if !path.exists() {
            return Err(DoctorError::FileNotFound(format!("{}", path.display())));
        }

        Ok(path)
    }

    /// Parse a timestamp in HH:MM:SS,mmm format
    fn parse_timestamp(&self, timestamp_str: &str) -> Result<u64, String> {
        let parts: Vec<&str> = timestamp_str.split(&[':', ',']).collect();
        if parts.len() != 4 {
            return Err(format!(
                "expected format HH:MM:SS,mmm, got {}",
                timestamp_str
            ));
        }

        let hours: u64 = parts[0]
            .parse()
            .map_err(|_| format!("invalid hours: {}", parts[0]))?;
        let minutes: u64 = parts[1]
            .parse()
            .map_err(|_| format!("invalid minutes: {}", parts[1]))?;
        let seconds: u64 = parts[2]
            .parse()
            .map_err(|_| format!("invalid seconds: {}", parts[2]))?;
        let millis: u64 = parts[3]
            .parse()
            .map_err(|_| format!("invalid milliseconds: {}", parts[3]))?;

        if minutes >= 60 {
            return Err(format!("minutes must be < 60, got {}", minutes));
        }
        if seconds >= 60 {
            return Err(format!("seconds must be < 60, got {}", seconds));
        }
        if millis >= 1000 {
            return Err(format!("milliseconds must be < 1000, got {}", millis));
        }

        let total_ms = hours * 3600000 + minutes * 60000 + seconds * 1000 + millis;
        Ok(total_ms)
    }
}

impl DoctorService for SubRipDoctorService {
    fn diagnose(&self, filename: &str) -> Result<DiagnosticReport, DoctorError> {
        info!("diagnostics for file '{}'", filename);

        let file_path = self.get_file_path(filename)?;
        let content = fs::read_to_string(&file_path)?;
        let lines: Vec<&str> = content.lines().collect();
        let total_lines = lines.len();

        debug!("file contain lines: {}", total_lines);

        let mut issues = Vec::new();
        let mut state = ParserState::BetweenBlocks;
        let mut current_block = 0;
        let mut empty_line_count = 0;
        let mut text_collected = false;
        let mut consecutive_empty_start = 0;

        for (line_num, line) in lines.iter().enumerate() {
            let line_number = line_num + 1; // 1-based
            let is_empty = line.trim().is_empty();

            match state {
                ParserState::BetweenBlocks => {
                    if is_empty {
                        if empty_line_count == 0 {
                            consecutive_empty_start = line_number;
                        }
                        empty_line_count += 1;
                    } else {
                        if empty_line_count > 1 {
                            issues.push(ValidationIssue {
                                line_number: consecutive_empty_start,
                                block_number: None,
                                issue_type: IssueType::MultipleEmptyLines {
                                    count: empty_line_count,
                                },
                                severity: Severity::Warning,
                                context: Some(format!(
                                    "lines {}-{}",
                                    consecutive_empty_start,
                                    consecutive_empty_start + empty_line_count - 1
                                )),
                            });
                        }
                        empty_line_count = 0;
                        current_block += 1;
                        text_collected = false;

                        if line.trim().parse::<u32>().is_err() {
                            issues.push(ValidationIssue {
                                line_number,
                                block_number: Some(current_block),
                                issue_type: IssueType::InvalidIndex {
                                    value: line.to_string(),
                                    reason: "not a valid number".to_string(),
                                },
                                severity: Severity::Error,
                                context: None,
                            });
                        } else if let Ok(idx) = line.trim().parse::<u32>()
                            && idx < 1
                        {
                            issues.push(ValidationIssue {
                                line_number,
                                block_number: Some(current_block),
                                issue_type: IssueType::InvalidIndex {
                                    value: line.to_string(),
                                    reason: "must be >= 1".to_string(),
                                },
                                severity: Severity::Error,
                                context: None,
                            });
                        }
                        state = ParserState::ExpectingTimestamp;
                    }
                }
                ParserState::ExpectingTimestamp => {
                    if is_empty {
                        issues.push(ValidationIssue {
                            line_number,
                            block_number: Some(current_block),
                            issue_type: IssueType::EmptyLineInBlock,
                            severity: Severity::Error,
                            context: Some("expected timestamp line".to_string()),
                        });
                        state = ParserState::BetweenBlocks;
                    } else {
                        if !line.contains(" --> ") {
                            issues.push(ValidationIssue {
                                line_number,
                                block_number: Some(current_block),
                                issue_type: IssueType::MalformedTimestamp {
                                    value: line.to_string(),
                                    reason: "missing ' --> ' separator".to_string(),
                                },
                                severity: Severity::Error,
                                context: None,
                            });
                        } else {
                            let parts: Vec<&str> = line.split(" --> ").collect();
                            if parts.len() == 2 {
                                match (
                                    self.parse_timestamp(parts[0]),
                                    self.parse_timestamp(parts[1]),
                                ) {
                                    (Ok(start_ms), Ok(end_ms)) => {
                                        if end_ms <= start_ms {
                                            issues.push(ValidationIssue {
                                                line_number,
                                                block_number: Some(current_block),
                                                issue_type: IssueType::InvalidTimestamp {
                                                    start: parts[0].to_string(),
                                                    end: parts[1].to_string(),
                                                },
                                                severity: Severity::Error,
                                                context: None,
                                            });
                                        }
                                    }
                                    (Err(e), _) => {
                                        issues.push(ValidationIssue {
                                            line_number,
                                            block_number: Some(current_block),
                                            issue_type: IssueType::MalformedTimestamp {
                                                value: parts[0].to_string(),
                                                reason: e,
                                            },
                                            severity: Severity::Error,
                                            context: None,
                                        });
                                    }
                                    (_, Err(e)) => {
                                        issues.push(ValidationIssue {
                                            line_number,
                                            block_number: Some(current_block),
                                            issue_type: IssueType::MalformedTimestamp {
                                                value: parts[1].to_string(),
                                                reason: e,
                                            },
                                            severity: Severity::Error,
                                            context: None,
                                        });
                                    }
                                }
                            }
                        }
                        state = ParserState::ExpectingText;
                    }
                }
                ParserState::ExpectingText => {
                    if is_empty {
                        if !text_collected {
                            issues.push(ValidationIssue {
                                line_number,
                                block_number: Some(current_block),
                                issue_type: IssueType::EmptyText,
                                severity: Severity::Error,
                                context: Some("no text before empty line".to_string()),
                            });
                        }
                        state = ParserState::BetweenBlocks;
                        empty_line_count = 0;
                    } else {
                        text_collected = true;
                    }
                }
            }
        }

        if state == ParserState::BetweenBlocks && empty_line_count > 1 {
            issues.push(ValidationIssue {
                line_number: consecutive_empty_start,
                block_number: None,
                issue_type: IssueType::MultipleEmptyLines {
                    count: empty_line_count,
                },
                severity: Severity::Warning,
                context: Some("trailing empty lines".to_string()),
            });
        }

        debug!("detected problematic lines: {}", issues.len());

        let subtitle_service = SubRipService::new(self.base_dir.clone());
        let is_parsable = subtitle_service.get_all(filename).is_ok();

        info!(
            "diagnostics completed: found {} errors, {} warnings",
            issues
                .iter()
                .filter(|i| i.severity == Severity::Error)
                .count(),
            issues
                .iter()
                .filter(|i| i.severity == Severity::Warning)
                .count()
        );

        Ok(DiagnosticReport {
            file_path: file_path.display().to_string(),
            total_lines,
            total_blocks: current_block,
            issues,
            is_parsable,
        })
    }

    fn fix(&self, filename: &str) -> Result<FixReport, DoctorError> {
        info!("starting file fix: {}", filename);

        let file_path = self.get_file_path(filename)?;

        let content = fs::read_to_string(&file_path)?;
        let lines: Vec<&str> = content.lines().collect();

        let mut output_lines = Vec::new();
        let mut state = ParserState::BetweenBlocks;
        let mut current_block_lines = Vec::new();
        let mut empty_line_count = 0;
        let mut issues_fixed = 0;
        let unfixable_issues = Vec::new();

        for line in lines.iter() {
            let is_empty = line.trim().is_empty();

            match state {
                ParserState::BetweenBlocks => {
                    if is_empty {
                        empty_line_count += 1;
                    } else {
                        if !output_lines.is_empty() {
                            if empty_line_count > 1 {
                                issues_fixed += 1;
                            }
                            output_lines.push(String::new());
                        }
                        empty_line_count = 0;
                        current_block_lines.clear();
                        current_block_lines.push(line.to_string());
                        state = ParserState::ExpectingTimestamp;
                    }
                }
                ParserState::ExpectingTimestamp => {
                    if is_empty {
                        issues_fixed += 1;
                    } else {
                        current_block_lines.push(line.to_string());
                        state = ParserState::ExpectingText;
                    }
                }
                ParserState::ExpectingText => {
                    if is_empty {
                        // End of block - write collected lines
                        for block_line in &current_block_lines {
                            output_lines.push(block_line.clone());
                        }
                        state = ParserState::BetweenBlocks;
                        empty_line_count = 0;
                    } else {
                        current_block_lines.push(line.to_string());
                    }
                }
            }
        }

        if !current_block_lines.is_empty() {
            for block_line in &current_block_lines {
                output_lines.push(block_line.clone());
            }
        }

        let fixed_content = output_lines.join("\n");
        fs::write(&file_path, fixed_content)?;

        debug!("fixed file written");

        let subtitle_service = SubRipService::new(self.base_dir.clone());
        let validation_success = subtitle_service.get_all(filename).is_ok();

        if validation_success {
            info!("validation successful: file can be parsed");
        } else {
            error!("validation failed: file still contains errors");
        }

        info!("fix completed: {} issues fixed", issues_fixed);

        Ok(FixReport {
            original_path: file_path.display().to_string(),
            fixed_path: file_path.display().to_string(),
            issues_fixed,
            issues_unfixable: unfixable_issues.len(),
            unfixable_issues,
            validation_success,
        })
    }
}
