use clap::Parser;
use cli::{Cli, Commands};
use logging::get_logging_config;

pub mod cli;
pub mod cmd;
pub mod logging;
pub mod output;
pub mod utils;

fn main() {
    let cli = Cli::parse();

    let logging_config = get_logging_config(&cli.log_level, &cli.log_target);
    log4rs::init_config(logging_config).expect("unable to init logging configuration");

    if let Some(command) = cli.command {
        match command {
            Commands::Get { file, index } => {
                if let Err(e) = cmd::get::handle(&file, &index) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Set {
                file,
                index,
                start,
                end,
                text,
            } => {
                if let Err(e) = cmd::set::handle(&file, index, start, end, text) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Doctor { file, fix } => {
                if let Err(e) = cmd::doctor::handle(&file, fix) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Add {
                file,
                timestamps,
                text,
            } => {
                if let Err(e) = cmd::add::handle(&file, &timestamps, &text) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Info { file } => {
                if let Err(e) = cmd::info::handle(&file) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Import {
                srt_file,
                input_file,
                format,
                reference,
                delimiter,
                dry_run,
                force,
            } => {
                if let Err(e) = cmd::import::handle(
                    &srt_file,
                    &input_file,
                    format,
                    reference.as_deref(),
                    &delimiter,
                    dry_run,
                    force,
                ) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
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
            } => {
                if let Err(e) = cmd::mass_rename::handle(
                    &file_mask,
                    dry_run,
                    force,
                    series_mode,
                    name,
                    season,
                    language,
                    &separator,
                    &file_template,
                ) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Compare { file1, file2 } => {
                if let Err(e) = cmd::compare::handle(&file1, &file2) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Verify {
                file1,
                file2,
                range,
            } => {
                if let Err(e) = cmd::verify::handle(&file1, &file2, range.as_deref()) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::TranslationStatus {
                reference,
                translation,
                chunk_size,
            } => {
                if let Err(e) =
                    cmd::translation_status::handle(&reference, &translation, chunk_size)
                {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Export {
                file,
                range,
                format,
            } => {
                if let Err(e) = cmd::export::handle(&file, &range, format) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
            Commands::Prompt { command } => match command {
                cli::PromptCommands::Translate {
                    file,
                    range,
                    language,
                    template_file,
                } => {
                    if let Err(e) = cmd::prompt::translate::handle(
                        &file,
                        &range,
                        &language,
                        template_file.as_deref(),
                    ) {
                        eprintln!("error: {}", e);
                        std::process::exit(1);
                    }
                }
            },
        }
    } else {
        eprintln!("No command specified. Use --help for usage information.");
        std::process::exit(1);
    }
}
