use crate::logging::get_logging_config;
use clap::Parser;

pub mod logging;

#[derive(Parser)]
#[command(name = "sm")]
#[command(version)]
#[command(about = "SUBMARINE - tiny toolkit for LLM translation assistance", long_about = None)]
struct Cli {
    /// Logging level (off, trace, debug, info, warn, error)
    #[arg(long, default_value = "off")]
    log_level: String,

    /// Logging target (console, file)
    #[arg(long, default_value = "console")]
    log_target: String,
}

fn main() {
    let cli = Cli::parse();

    let logging_config = get_logging_config(&cli.log_level, &cli.log_target);
    log4rs::init_config(logging_config).expect("unable to init logging configuration");
}
