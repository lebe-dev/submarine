use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "sm")]
#[command(version)]
#[command(about = "SUBMARINE - tiny toolkit for LLM translation assistance", long_about = None)]
pub struct Cli {
    /// Logging level (off, trace, debug, info, warn, error)
    #[arg(long, default_value = "off")]
    pub log_level: String,

    /// Logging target (console, file)
    #[arg(long, default_value = "console")]
    pub log_target: String,

    #[command(subcommand)]
    pub command: Option<Commands>,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Get a subtitle by its index from an SRT file
    Get {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,

        /// Subtitle index to retrieve (>= 1)
        #[arg(value_name = "INDEX")]
        index: u32,
    },

    /// Set/update subtitle properties by index
    Set {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,

        /// Subtitle index to update (>= 1)
        #[arg(value_name = "INDEX")]
        index: u32,

        /// New start timestamp in format HH:MM:SS,mmm
        #[arg(long)]
        start: Option<String>,

        /// New end timestamp in format HH:MM:SS,mmm
        #[arg(long)]
        end: Option<String>,

        /// New subtitle text (can be multi-line with \n)
        #[arg(long)]
        text: Option<String>,
    },

    /// Diagnose and fix issues in SRT subtitle files
    Doctor {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,

        /// Fix issues automatically
        #[arg(long)]
        fix: bool,
    },
}
