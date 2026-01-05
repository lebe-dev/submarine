use clap::Parser;
use cli::{Cli, Commands};
use logging::get_logging_config;

pub mod cli;
pub mod cmd;
pub mod logging;
pub mod output;

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
                csv_file,
                delimiter,
            } => {
                if let Err(e) = cmd::import::handle(&srt_file, &csv_file, &delimiter) {
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
            Commands::Verify { file1, file2 } => {
                if let Err(e) = cmd::verify::handle(&file1, &file2) {
                    eprintln!("error: {}", e);
                    std::process::exit(1);
                }
            }
        }
    } else {
        eprintln!("No command specified. Use --help for usage information.");
        std::process::exit(1);
    }
}
