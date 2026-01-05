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
) -> anyhow::Result<()> {
    info!("starting mass-rename with mask: {}", file_mask);
    debug!(
        "dry_run: {}, force: {}, series_mode: {}",
        dry_run, force, series_mode
    );

    // Get current directory
    let current_dir = std::env::current_dir()?;
    debug!("current directory: {:?}", current_dir);

    // Create service
    let service = FileRenameService::new(current_dir.clone());

    // Find files
    let files = match service.find_files(file_mask) {
        Ok(files) => files,
        Err(e) => {
            error!("failed to find files: {}", e);
            handle_error(e);
            std::process::exit(1);
        }
    };

    info!("found {} files", files.len());

    // Create template context
    let context = TemplateContext::new()
        .with_name(name)
        .with_season(season)
        .with_language(language)
        .with_separator(Some(separator.to_string()));

    // Prepare rename operations
    let operations =
        match service.prepare_rename_operations(files, file_template, &context, series_mode) {
            Ok(ops) => ops,
            Err(e) => {
                error!("failed to prepare operations: {}", e);
                handle_error(e);
                std::process::exit(1);
            }
        };

    // Show preview
    println!("Preview:");
    println!();
    for op in &operations {
        let original_name = op
            .original_filename()
            .unwrap_or_else(|| "unknown".to_string());
        println!("  {} -> {}", original_name, op.new_name);
        if op.collision {
            println!("    ⚠️  Warning: file already exists");
        }
    }
    println!();

    // Exit if dry-run mode
    if dry_run {
        info!("dry-run mode, operations not executed");
        println!("Dry-run mode: no files were renamed");
        return Ok(());
    }

    // Request confirmation if --force not specified
    if !force {
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

        // Check collision
        if op.collision {
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
        }

        // Create backup
        debug!("creating backup for {:?}", op.original_path);
        match backup_service.create_backup(&op.original_path) {
            Ok(Some(backup_path)) => {
                debug!("backup created: {}", backup_path);
            }
            Ok(None) => {
                error!("file does not exist, cannot be renamed");
                eprintln!("  Error: File not found: {}", original_name);
                skipped_count += 1;
                continue;
            }
            Err(e) => {
                error!("failed to create backup: {}", e);
                eprintln!(
                    "  Error: Failed to create backup for {}: {}",
                    original_name, e
                );
                skipped_count += 1;
                continue;
            }
        }

        // Rename file
        let new_path = current_dir.join(&op.new_name);
        debug!("renaming {:?} -> {:?}", op.original_path, new_path);

        match std::fs::rename(&op.original_path, &new_path) {
            Ok(_) => {
                info!("file renamed: {} -> {}", original_name, op.new_name);
                println!("  ✓ {} -> {}", original_name, op.new_name);
                renamed_count += 1;
            }
            Err(e) => {
                error!("failed to rename file: {}", e);
                eprintln!("  ✗ Failed to rename {}: {}", original_name, e);
                skipped_count += 1;
            }
        }
    }

    // Final report
    println!();
    println!("Summary:");
    println!(
        "  ✓ {} file{} renamed",
        renamed_count,
        if renamed_count == 1 { "" } else { "s" }
    );
    if skipped_count > 0 {
        println!(
            "  ⚠️  {} file{} skipped",
            skipped_count,
            if skipped_count == 1 { "" } else { "s" }
        );
    }

    Ok(())
}

fn handle_error(e: RenameError) {
    match e {
        RenameError::NoFilesFound(mask) => {
            eprintln!("error: No files found matching pattern: {}", mask);
        }
        RenameError::TemplateError(err) => {
            eprintln!("error: Template rendering failed: {}", err);
        }
        RenameError::IoError(err) => {
            eprintln!("error: File operation failed: {}", err);
        }
        RenameError::InvalidTemplate(err) => {
            eprintln!("error: Invalid template: {}", err);
        }
        RenameError::GlobError(err) => {
            eprintln!("error: Glob pattern error: {}", err);
        }
    }
}
