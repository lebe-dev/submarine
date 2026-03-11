use clap::Parser;
use cli::{Cli, Commands, OutputFormat};
use json_output::output_error;
use logging::get_logging_config;

pub mod cli;
pub mod cmd;
pub mod dto;
pub mod json_output;
pub mod logging;
pub mod output;
pub mod utils;

fn main() {
    let cli = Cli::parse();

    let logging_config = get_logging_config(&cli.log_level, &cli.log_target);
    log4rs::init_config(logging_config).expect("unable to init logging configuration");

    let format = &cli.output;

    if let Some(command) = cli.command {
        let result = match command {
            Commands::Get { file, index } => cmd::get::handle(&file, &index, format),
            Commands::Set {
                file,
                index,
                start,
                end,
                text,
                dry_run,
            } => cmd::set::handle(&file, index, start, end, text, dry_run, format),
            Commands::Doctor { file, fix } => cmd::doctor::handle(&file, fix, format),
            Commands::Add {
                file,
                timestamps,
                text,
                dry_run,
            } => cmd::add::handle(&file, &timestamps, &text, dry_run, format),
            Commands::Info { file } => cmd::info::handle(&file, format),
            Commands::Import {
                srt_file,
                input_file,
                format: import_format,
                reference,
                delimiter,
                dry_run,
                force,
            } => cmd::import::handle(
                &srt_file,
                &input_file,
                import_format,
                reference.as_deref(),
                &delimiter,
                dry_run,
                force,
                format,
            ),
            Commands::MassRename {
                file_mask,
                dry_run,
                force,
                series_mode,
                name,
                season,
                language,
                separator,
                file_template,
            } => cmd::mass_rename::handle(
                &file_mask,
                dry_run,
                force,
                series_mode,
                name,
                season,
                language,
                &separator,
                &file_template,
                format,
            ),
            Commands::Compare { file1, file2 } => cmd::compare::handle(&file1, &file2, format),
            Commands::Verify {
                file1,
                file2,
                range,
            } => cmd::verify::handle(&file1, &file2, range.as_deref(), format),
            Commands::TranslationStatus {
                reference,
                translation,
                chunk_size,
            } => cmd::translation_status::handle(&reference, &translation, chunk_size, format),
            Commands::Export {
                file,
                range,
                format: export_format,
            } => cmd::export::handle(&file, &range, export_format, format),
            Commands::Delay {
                file,
                offset,
                dry_run,
            } => cmd::delay::handle(&file, &offset, dry_run, format),
            Commands::Describe { command } => cmd::describe::handle(command.as_deref()),
        };

        if let Err(e) = result {
            // If the handler already output the error (via output_error), we just exit.
            // If it returned an anyhow error without outputting, we format it here.
            let msg = format!("{}", e);
            if !msg.is_empty() {
                output_error(format, "error", &msg, None);
            }
            std::process::exit(1);
        }
    } else {
        output_error(
            &OutputFormat::Text,
            "no_command",
            "No command specified. Use --help for usage information.",
            None,
        );
        std::process::exit(1);
    }
}
