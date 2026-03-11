use crate::cli::OutputFormat;
use crate::dto::{DiagnosticDto, FixDto, IssueDto};
use crate::json_output::{output_error, output_success};
use crate::utils;
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::doctor::model::Severity;
use lib::doctor::ports::DoctorService;
use lib::doctor::service::SubRipDoctorService;
use log::{debug, error, info};

pub fn handle(file: &str, fix: bool, format: &OutputFormat) -> anyhow::Result<()> {
    info!("running doctor for file: {}", file);

    let resolved = utils::resolve_existing_path(file)?;
    let service = SubRipDoctorService::new(resolved.base_dir);

    if fix {
        let backup_service = SubRipBackupService::new();
        let backup_result = backup_service.create_backup(&resolved.full_path);

        let backup_path = match backup_result {
            Ok(Some(path)) => {
                debug!("backup created: {}", path);
                path
            }
            Ok(None) => {
                error!("file does not exist, cannot fix");
                output_error(
                    format,
                    "file_not_found",
                    &format!("File does not exist: {}", file),
                    None,
                );
                return Err(anyhow::anyhow!(""));
            }
            Err(e) => {
                error!("failed to create backup: {}", e);
                output_error(
                    format,
                    "backup_failed",
                    &format!("Failed to create backup: {}", e),
                    None,
                );
                return Err(anyhow::anyhow!(""));
            }
        };

        match service.fix(&resolved.filename) {
            Ok(report) => {
                info!("fix completed successfully");

                let dto = FixDto {
                    original_path: report.original_path.clone(),
                    backup_path: backup_path.clone(),
                    issues_fixed: report.issues_fixed,
                    issues_unfixable: report.issues_unfixable,
                    validation_success: report.validation_success,
                    unfixable_issues: report
                        .unfixable_issues
                        .iter()
                        .map(|i| IssueDto {
                            line_number: i.line_number,
                            block_number: i.block_number,
                            severity: match i.severity {
                                Severity::Error => "error".into(),
                                Severity::Warning => "warning".into(),
                            },
                            description: format!("{}", i.issue_type),
                        })
                        .collect(),
                };

                output_success(format, &dto, || {
                    print_fix_report(&report, &backup_path);
                });

                Ok(())
            }
            Err(e) => {
                error!("fix failed: {}", e);
                let cli_err = utils::format_doctor_error(&e);
                output_error(
                    format,
                    &cli_err.code,
                    &cli_err.message,
                    cli_err.hint.as_deref(),
                );
                Err(anyhow::anyhow!(""))
            }
        }
    } else {
        match service.diagnose(&resolved.filename) {
            Ok(report) => {
                info!("diagnosis completed");

                let dto = DiagnosticDto {
                    file_path: report.file_path.clone(),
                    total_lines: report.total_lines,
                    total_blocks: report.total_blocks,
                    is_parsable: report.is_parsable,
                    error_count: report.error_count(),
                    warning_count: report.warning_count(),
                    has_issues: report.has_issues(),
                    issues: report
                        .issues
                        .iter()
                        .map(|i| IssueDto {
                            line_number: i.line_number,
                            block_number: i.block_number,
                            severity: match i.severity {
                                Severity::Error => "error".into(),
                                Severity::Warning => "warning".into(),
                            },
                            description: format!("{}", i.issue_type),
                        })
                        .collect(),
                };

                output_success(format, &dto, || {
                    print_diagnostic_report(&report);
                });

                if report.has_errors() {
                    return Err(anyhow::anyhow!(""));
                }

                Ok(())
            }
            Err(e) => {
                error!("diagnosis failed: {}", e);
                let cli_err = utils::format_doctor_error(&e);
                output_error(
                    format,
                    &cli_err.code,
                    &cli_err.message,
                    cli_err.hint.as_deref(),
                );
                Err(anyhow::anyhow!(""))
            }
        }
    }
}

fn print_diagnostic_report(report: &lib::doctor::model::DiagnosticReport) {
    println!("Diagnosing: {}", report.file_path);
    println!();

    println!("File Statistics:");
    println!("  Total lines: {}", report.total_lines);
    println!("  Total blocks: {}", report.total_blocks);
    println!();

    if report.has_issues() {
        println!("Issues Found:");
        println!();

        for issue in &report.issues {
            println!("{}", issue);
        }

        println!();
        println!("Summary:");

        let error_count = report.error_count();
        let warning_count = report.warning_count();

        if error_count > 0 {
            println!(
                "  {} error{} found",
                error_count,
                if error_count == 1 { "" } else { "s" }
            );
        }
        if warning_count > 0 {
            println!(
                "  {} warning{} found",
                warning_count,
                if warning_count == 1 { "" } else { "s" }
            );
        }

        if report.is_parsable {
            println!("  File is parsable (but has issues)");
        } else {
            println!("  File is NOT parsable");
        }

        println!();
        println!("Run with --fix to automatically repair these issues.");
    } else {
        println!("No issues found");
        println!("  File is valid");
    }
}

fn print_fix_report(report: &lib::doctor::model::FixReport, backup_path: &str) {
    println!("Fixing: {}", report.original_path);
    println!();

    println!("Backup: {}", backup_path);

    if report.issues_fixed > 0 {
        println!(
            "Fixed {} issue{}",
            report.issues_fixed,
            if report.issues_fixed == 1 { "" } else { "s" }
        );
    }

    if report.issues_unfixable > 0 {
        println!(
            "{} issue{} could not be fixed:",
            report.issues_unfixable,
            if report.issues_unfixable == 1 {
                ""
            } else {
                "s"
            }
        );
        for issue in &report.unfixable_issues {
            println!("  {}", issue);
        }
    }

    if report.validation_success {
        println!("Validation successful: file can now be parsed");
    } else {
        println!("Validation failed: file still contains errors");
    }

    println!();
    println!("Summary:");
    println!(
        "  {} issue{} fixed",
        report.issues_fixed,
        if report.issues_fixed == 1 { "" } else { "s" }
    );
    println!(
        "  {} issue{} unfixable",
        report.issues_unfixable,
        if report.issues_unfixable == 1 {
            ""
        } else {
            "s"
        }
    );

    if report.validation_success {
        println!("  File is now valid");
    } else {
        println!("  File still has errors");
    }
}
