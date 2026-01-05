use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::doctor::model::DoctorError;
use lib::doctor::ports::DoctorService;
use lib::doctor::service::SubRipDoctorService;
use log::{debug, error, info};
use std::path::Path;

pub fn handle(file: &str, fix: bool) -> anyhow::Result<()> {
    info!("запуск doctor для файла: {}", file);

    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

    let canonical_path = file_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("Failed to resolve file path: {}", e))?;
    debug!("canonical path: {:?}", canonical_path);

    let base_dir = canonical_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("Invalid file path"))?
        .to_path_buf();
    debug!("base directory: {:?}", base_dir);

    let filename = canonical_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("Invalid file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("Invalid UTF-8 in filename"))?
        .to_string();
    debug!("filename: {}", filename);

    let service = SubRipDoctorService::new(base_dir);

    if fix {
        // Fix mode - create backup before fixing
        let backup_service = SubRipBackupService::new();
        let backup_result = backup_service.create_backup(&canonical_path);

        let backup_path = match backup_result {
            Ok(Some(path)) => {
                debug!("backup created: {}", path);
                path
            }
            Ok(None) => {
                error!("file does not exist, cannot fix");
                eprintln!("error: File does not exist: {}", file);
                std::process::exit(1);
            }
            Err(e) => {
                error!("failed to create backup: {}", e);
                eprintln!("error: Failed to create backup: {}", e);
                std::process::exit(1);
            }
        };

        match service.fix(&filename) {
            Ok(report) => {
                info!("исправление завершено успешно");
                print_fix_report(&report, &backup_path);
                Ok(())
            }
            Err(e) => {
                error!("исправление не удалось: {}", e);
                handle_error(e);
                std::process::exit(1);
            }
        }
    } else {
        // Diagnostic mode
        match service.diagnose(&filename) {
            Ok(report) => {
                info!("диагностика завершена");
                print_diagnostic_report(&report);

                if report.has_errors() {
                    std::process::exit(1);
                } else {
                    Ok(())
                }
            }
            Err(e) => {
                error!("диагностика не удалась: {}", e);
                handle_error(e);
                std::process::exit(1);
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
                "  ❌ {} error{} found",
                error_count,
                if error_count == 1 { "" } else { "s" }
            );
        }
        if warning_count > 0 {
            println!(
                "  ⚠️  {} warning{} found",
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
        println!("✓ No issues found");
        println!("  File is valid");
    }
}

fn print_fix_report(report: &lib::doctor::model::FixReport, backup_path: &str) {
    println!("Fixing: {}", report.original_path);
    println!();

    println!("✓ Backup: {}", backup_path);

    if report.issues_fixed > 0 {
        println!(
            "✓ Fixed {} issue{}",
            report.issues_fixed,
            if report.issues_fixed == 1 { "" } else { "s" }
        );
    }

    if report.issues_unfixable > 0 {
        println!(
            "✗ {} issue{} could not be fixed:",
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
        println!("✓ Validation successful: file can now be parsed");
    } else {
        println!("⚠️  Validation failed: file still contains errors");
    }

    println!();
    println!("Summary:");
    println!(
        "  ✓ {} issue{} fixed",
        report.issues_fixed,
        if report.issues_fixed == 1 { "" } else { "s" }
    );
    println!(
        "  ✗ {} issue{} unfixable",
        report.issues_unfixable,
        if report.issues_unfixable == 1 {
            ""
        } else {
            "s"
        }
    );

    if report.validation_success {
        println!("  ✓ File is now valid");
    } else {
        println!("  ✗ File still has errors");
    }
}

fn handle_error(e: DoctorError) {
    match e {
        DoctorError::FileNotFound(path) => {
            eprintln!("error: File not found: {}", path);
        }
        DoctorError::InvalidPath(msg) => {
            eprintln!("error: Invalid file path: {}", msg);
        }
        DoctorError::IoError(err) => {
            eprintln!("error: I/O error: {}", err);
        }
        DoctorError::BackupFailed(msg) => {
            eprintln!("error: Backup creation failed: {}", msg);
        }
        DoctorError::ValidationFailed(msg) => {
            eprintln!("error: Validation failed: {}", msg);
        }
    }
}
