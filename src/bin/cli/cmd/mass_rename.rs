use crate::cli::OutputFormat;
use crate::dto::{MassRenameResultDto, RenameOperationDto};
use crate::json_output::{output_error, output_success};
use lib::backup::ports::BackupService;
use lib::backup::service::SubRipBackupService;
use lib::rename::model::RenameError;
use lib::rename::ports::{RenameService, TemplateContext};
use lib::rename::service::FileRenameService;
use log::{debug, error, info};
use std::io::{self, Write};

#[allow(clippy::too_many_arguments)]
pub fn handle(
    file_mask: &str,
    dry_run: bool,
    force: bool,
    series_mode: bool,
    name: Option<String>,
    season: Option<u32>,
    language: Option<String>,
    separator: &str,
    file_template: &str,
    format: &OutputFormat,
) -> anyhow::Result<()> {
    info!("starting mass-rename with mask: {}", file_mask);
    debug!(
        "dry_run: {}, force: {}, series_mode: {}",
        dry_run, force, series_mode
    );

    let current_dir = std::env::current_dir()?;
    let service = FileRenameService::new(current_dir.clone());

    let files = match service.find_files(file_mask) {
        Ok(files) => files,
        Err(e) => {
            let (code, msg) = format_rename_error(&e);
            output_error(format, code, msg, None);
            return Err(anyhow::anyhow!(""));
        }
    };

    info!("found {} files", files.len());

    let context = TemplateContext::new()
        .with_name(name)
        .with_season(season)
        .with_language(language)
        .with_separator(Some(separator.to_string()));

    let operations =
        match service.prepare_rename_operations(files, file_template, &context, series_mode) {
            Ok(ops) => ops,
            Err(e) => {
                let (code, msg) = format_rename_error(&e);
                output_error(format, code, msg, None);
                return Err(anyhow::anyhow!(""));
            }
        };

    let op_dtos: Vec<RenameOperationDto> = operations
        .iter()
        .map(|op| RenameOperationDto {
            original: op
                .original_filename()
                .unwrap_or_else(|| "unknown".to_string()),
            new_name: op.new_name.clone(),
            collision: op.collision,
        })
        .collect();

    if dry_run {
        info!("dry-run mode, operations not executed");

        let dto = MassRenameResultDto {
            renamed_count: 0,
            skipped_count: 0,
            operations: op_dtos,
            dry_run: true,
        };

        output_success(format, &dto, || {
            println!("Preview:");
            println!();
            for op in &operations {
                let original_name = op
                    .original_filename()
                    .unwrap_or_else(|| "unknown".to_string());
                println!("  {} -> {}", original_name, op.new_name);
                if op.collision {
                    println!("    Warning: file already exists");
                }
            }
            println!();
            println!("Dry-run mode: no files were renamed");
        });

        return Ok(());
    }

    // Show preview in text mode
    if matches!(format, OutputFormat::Text) {
        println!("Preview:");
        println!();
        for op in &operations {
            let original_name = op
                .original_filename()
                .unwrap_or_else(|| "unknown".to_string());
            println!("  {} -> {}", original_name, op.new_name);
            if op.collision {
                println!("    Warning: file already exists");
            }
        }
        println!();
    }

    if !force && matches!(format, OutputFormat::Text) {
        print!("Proceed with renaming {} files? (y/N): ", operations.len());
        io::stdout().flush()?;

        let mut input = String::new();
        io::stdin().read_line(&mut input)?;

        if !input.trim().eq_ignore_ascii_case("y") {
            info!("operation cancelled by user");
            println!("Cancelled");
            return Ok(());
        }
    }

    // Execute rename operations
    info!("starting file renaming");
    let backup_service = SubRipBackupService::new();
    let mut renamed_count = 0;
    let mut skipped_count = 0;

    for op in operations {
        let original_name = op
            .original_filename()
            .unwrap_or_else(|| "unknown".to_string());

        if op.collision && matches!(format, OutputFormat::Text) {
            print!("File '{}' already exists. Overwrite? (y/N): ", op.new_name);
            io::stdout().flush()?;

            let mut input = String::new();
            io::stdin().read_line(&mut input)?;

            if !input.trim().eq_ignore_ascii_case("y") {
                info!("skipping file due to collision: {:?}", op.original_path);
                println!("  Skipped: {}", original_name);
                skipped_count += 1;
                continue;
            }
        } else if op.collision {
            // In JSON mode, skip collisions without prompting
            skipped_count += 1;
            continue;
        }

        match backup_service.create_backup(&op.original_path) {
            Ok(Some(_)) => {}
            Ok(None) => {
                error!("file does not exist, cannot be renamed");
                if matches!(format, OutputFormat::Text) {
                    eprintln!("  Error: File not found: {}", original_name);
                }
                skipped_count += 1;
                continue;
            }
            Err(e) => {
                error!("failed to create backup: {}", e);
                if matches!(format, OutputFormat::Text) {
                    eprintln!(
                        "  Error: Failed to create backup for {}: {}",
                        original_name, e
                    );
                }
                skipped_count += 1;
                continue;
            }
        }

        let new_path = current_dir.join(&op.new_name);

        match std::fs::rename(&op.original_path, &new_path) {
            Ok(_) => {
                info!("file renamed: {} -> {}", original_name, op.new_name);
                if matches!(format, OutputFormat::Text) {
                    println!("  {} -> {}", original_name, op.new_name);
                }
                renamed_count += 1;
            }
            Err(e) => {
                error!("failed to rename file: {}", e);
                if matches!(format, OutputFormat::Text) {
                    eprintln!("  Failed to rename {}: {}", original_name, e);
                }
                skipped_count += 1;
            }
        }
    }

    let dto = MassRenameResultDto {
        renamed_count,
        skipped_count,
        operations: op_dtos,
        dry_run: false,
    };

    output_success(format, &dto, || {
        println!();
        println!("Summary:");
        println!(
            "  {} file{} renamed",
            renamed_count,
            if renamed_count == 1 { "" } else { "s" }
        );
        if skipped_count > 0 {
            println!(
                "  {} file{} skipped",
                skipped_count,
                if skipped_count == 1 { "" } else { "s" }
            );
        }
    });

    Ok(())
}

fn format_rename_error(e: &RenameError) -> (&str, &str) {
    match e {
        RenameError::NoFilesFound(_) => ("no_files_found", "No files found matching pattern"),
        RenameError::TemplateError(_) => ("template_error", "Template rendering failed"),
        RenameError::IoError(_) => ("io_error", "File operation failed"),
        RenameError::InvalidTemplate(_) => ("invalid_template", "Invalid template"),
        RenameError::GlobError(_) => ("glob_error", "Glob pattern error"),
    }
}
