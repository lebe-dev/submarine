use lib::subtitle::model::SubtitleError;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

/// Parse range string in format "START-END" into (start, end) tuple
fn parse_range(range: &str) -> anyhow::Result<(u32, u32)> {
    let parts: Vec<&str> = range.split('-').collect();

    if parts.len() != 2 {
        error!("invalid range format: {}", range);
        eprintln!(
            "error: Invalid range format '{}'. Expected format: START-END (e.g., 120-123)",
            range
        );
        std::process::exit(1);
    }

    let start = parts[0].trim().parse::<u32>().map_err(|_| {
        error!("invalid start index in range: {}", parts[0]);
        eprintln!(
            "error: Invalid start index '{}'. Must be a positive number",
            parts[0]
        );
        std::process::exit(1);
    })?;

    let end = parts[1].trim().parse::<u32>().map_err(|_| {
        error!("invalid end index in range: {}", parts[1]);
        eprintln!(
            "error: Invalid end index '{}'. Must be a positive number",
            parts[1]
        );
        std::process::exit(1);
    })?;

    Ok((start, end))
}

pub fn handle(file: &str, index_str: &str) -> anyhow::Result<()> {
    if index_str.contains('-') {
        debug!("detected range format, processing range");
        return handle_range(file, index_str);
    }

    let index = index_str.parse::<u32>().map_err(|_| {
        error!("invalid index: {}", index_str);
        eprintln!(
            "error: Invalid index '{}'. Must be a positive number or range (e.g., 120-123)",
            index_str
        );
        std::process::exit(1);
    })?;

    info!("getting subtitle {} from file: {}", index, file);

    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

    if file_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("Failed to get current directory: {}", e))?;

        let resolved = current_dir.join(file_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("Failed to resolve file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("Failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", file_path);
            return Err(anyhow::anyhow!(
                "Invalid file path: path traversal not allowed"
            ));
        }
    }

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

    let service = SubRipService::new(base_dir);

    debug!("retrieving subtitle by id: {}..", index);
    match service.get_by_id(&filename, index) {
        Ok(Some(subtitle)) => {
            info!("subtitle {} found successfully", index);
            debug!("subtitle content: {:?}", subtitle);
            println!("{}", subtitle);
            Ok(())
        }
        Ok(None) => {
            info!("subtitle {} not found in file", index);
            eprintln!("subtitle with index {} not found in {}", index, file);
            std::process::exit(1);
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    info!("file not found: {}", path);
                    eprintln!("error: File not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: Invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: Failed to parse subtitle file: {}", err);
                }
                SubtitleError::IoError(err) => {
                    error!("i/o error: {}", err);
                    eprintln!("error: Failed to read file: {}", err);
                }
                _ => {
                    error!("unexpected error: {}", e);
                    eprintln!("error: {}", e);
                }
            }
            std::process::exit(1);
        }
    }
}

fn handle_range(file: &str, range: &str) -> anyhow::Result<()> {
    info!("getting subtitles in range {} from file: {}", range, file);

    let (start, end) = parse_range(range)?;
    debug!("parsed range: start={}, end={}", start, end);

    if start < 1 {
        error!("invalid start index: must be >= 1");
        eprintln!("error: Start index must be >= 1, got {}", start);
        std::process::exit(1);
    }

    if end < 1 {
        error!("invalid end index: must be >= 1");
        eprintln!("error: End index must be >= 1, got {}", end);
        std::process::exit(1);
    }

    if start > end {
        error!("invalid range: start {} > end {}", start, end);
        eprintln!(
            "error: Start index must be <= end index (got {} > {})",
            start, end
        );
        std::process::exit(1);
    }

    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

    if file_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("Failed to get current directory: {}", e))?;

        let resolved = current_dir.join(file_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("Failed to resolve file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("Failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", file_path);
            return Err(anyhow::anyhow!(
                "Invalid file path: path traversal not allowed"
            ));
        }
    }

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

    let service = SubRipService::new(base_dir);

    debug!("retrieving all subtitles for filtering..");
    match service.get_all(&filename) {
        Ok(subtitles) => {
            debug!("found {} total subtitles", subtitles.len());

            let range_subtitles: Vec<_> = subtitles
                .into_iter()
                .filter(|s| {
                    let index = *s.index.as_ref();
                    index >= start && index <= end
                })
                .collect();

            debug!(
                "found {} subtitle(s) in range {}-{}",
                range_subtitles.len(),
                start,
                end
            );

            if range_subtitles.is_empty() {
                info!("no subtitles found in range {}-{}", start, end);
                println!("No subtitles found in range {}-{}", start, end);
            } else {
                info!(
                    "found {} subtitle(s) in range {}-{}",
                    range_subtitles.len(),
                    start,
                    end
                );

                for (i, subtitle) in range_subtitles.iter().enumerate() {
                    print!("{}", subtitle);
                    if i < range_subtitles.len() - 1 {
                        println!("\n");
                    } else {
                        println!();
                    }
                }
            }

            Ok(())
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    info!("file not found: {}", path);
                    eprintln!("error: File not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: Invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: Failed to parse subtitle file: {}", err);
                }
                SubtitleError::IoError(err) => {
                    error!("i/o error: {}", err);
                    eprintln!("error: Failed to read file: {}", err);
                }
                _ => {
                    error!("unexpected error: {}", e);
                    eprintln!("error: {}", e);
                }
            }
            std::process::exit(1);
        }
    }
}
