use clap::{Parser, Subcommand, ValueEnum};

#[derive(Clone, Debug, ValueEnum)]
pub enum ExportFormat {
    /// Anchored format: [INDEX] TEXT
    Anchored,
}

#[derive(Clone, Debug, ValueEnum)]
pub enum ImportFormat {
    /// CSV format: start_time|end_time|text
    Csv,
    /// Anchored format: [INDEX] TEXT
    Anchored,
}

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

        /// Subtitle index to retrieve (>= 1) or range (e.g., 120-123)
        #[arg(value_name = "INDEX")]
        index: String,
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

    /// Add a new subtitle to the end of an SRT file
    Add {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,

        /// Timestamp range in format HH:MM:SS,mmm-HH:MM:SS,mmm
        #[arg(value_name = "TIMESTAMPS")]
        timestamps: String,

        /// Subtitle text (can be multi-line with \n)
        #[arg(value_name = "TEXT")]
        text: String,
    },

    /// Display statistics and information about an SRT file
    Info {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,
    },

    /// Import subtitles from a file into an SRT file
    Import {
        /// Path to the SRT file
        #[arg(value_name = "SRT_FILE")]
        srt_file: String,

        /// Path to the input file
        #[arg(value_name = "INPUT_FILE")]
        input_file: String,

        /// Import format
        #[arg(long, value_name = "FORMAT")]
        format: ImportFormat,

        /// Path to reference SRT file (required for anchored format)
        #[arg(long, value_name = "FILE")]
        reference: Option<String>,

        /// Delimiter character for CSV format (default: pipe '|')
        #[arg(long, default_value = "|")]
        delimiter: String,

        /// Preview changes without modifying the file
        #[arg(long)]
        dry_run: bool,

        /// Skip confirmation prompt
        #[arg(long)]
        force: bool,
    },

    /// Mass rename subtitle files using templates
    MassRename {
        /// File mask for case-insensitive matching
        #[arg(value_name = "FILE-MASK")]
        file_mask: String,

        /// Show preview without renaming
        #[arg(long)]
        dry_run: bool,

        /// Skip confirmation prompt
        #[arg(long)]
        force: bool,

        /// Enable series mode with auto-incrementing episodes
        #[arg(long)]
        series_mode: bool,

        /// Series/show name
        #[arg(long)]
        name: Option<String>,

        /// Season number
        #[arg(long)]
        season: Option<u32>,

        /// Language code
        #[arg(long)]
        language: Option<String>,

        /// Separator character
        #[arg(long, default_value = ".")]
        separator: String,

        /// File name template (Tera syntax)
        #[arg(
            long,
            default_value = "{{ name }}{{ separator }}S{{ season }}{{ separator }}E{{ episode }}.srt"
        )]
        file_template: String,
    },

    /// Compare two SRT files side-by-side in TUI mode
    Compare {
        /// Path to the first SRT file
        #[arg(value_name = "FILE1")]
        file1: String,

        /// Path to the second SRT file
        #[arg(value_name = "FILE2")]
        file2: String,
    },

    /// Verify two subtitle files for index and timestamp discrepancies
    Verify {
        /// Path to the reference SRT file (FILE1)
        #[arg(value_name = "FILE1")]
        file1: String,

        /// Path to the target SRT file to verify (FILE2)
        #[arg(value_name = "FILE2")]
        file2: String,

        /// Optional subtitle range in format START-END (e.g., "1-50")
        #[arg(long, value_name = "RANGE")]
        range: Option<String>,
    },

    /// Check translation progress against reference file
    #[command(visible_alias = "ts")]
    TranslationStatus {
        /// Path to the reference SRT file
        #[arg(long, short = 'r', value_name = "FILE")]
        reference: String,

        /// Path to the translation SRT file
        #[arg(value_name = "TRANSLATION_FILE")]
        translation: String,

        /// Chunk size for next translation suggestion
        #[arg(long, default_value = "50")]
        chunk_size: usize,
    },

    /// Export subtitles in specified format
    Export {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,

        /// Subtitle range in format START-END (e.g., "1-50")
        #[arg(value_name = "RANGE")]
        range: String,

        /// Export format
        #[arg(long, value_name = "FORMAT")]
        format: ExportFormat,
    },

    /// Adjust subtitle timestamps by specified milliseconds offset
    Delay {
        /// Path to the SRT file
        #[arg(value_name = "FILE")]
        file: String,

        /// Time offset in milliseconds (e.g., "+100", "-500")
        #[arg(value_name = "OFFSET", allow_hyphen_values = true)]
        offset: String,
    },
}
