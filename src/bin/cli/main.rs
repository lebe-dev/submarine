use clap::Parser;
use cli::{Cli, Commands};
use logging::get_logging_config;

pub mod cli;
pub mod cmd;
pub mod logging;

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
            Commands::GetRange { file, range } => {
                if let Err(e) = cmd::get_range::handle(&file, &range) {
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
        }
    } else {
        eprintln!("No command specified. Use --help for usage information.");
        std::process::exit(1);
    }
}
