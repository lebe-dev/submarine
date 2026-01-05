use lib::subtitle::model::{Subtitle, SubtitleError};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

/// Entry point for the compare command
///
/// Loads two SRT files and displays them side-by-side in a TUI.
/// Users can navigate with arrow keys (or j/k) and exit with Esc or 'q'.
pub fn handle(file1: &str, file2: &str) -> anyhow::Result<()> {
    info!("comparing files: {} and {}", file1, file2);

    let (subtitles1, filename1) = load_subtitle_file(file1)?;
    let (subtitles2, filename2) = load_subtitle_file(file2)?;

    debug!(
        "loaded {} and {} subtitles",
        subtitles1.len(),
        subtitles2.len()
    );

    crate::output::compare::run_tui(subtitles1, filename1, subtitles2, filename2)?;

    info!("comparison completed");
    Ok(())
}

/// Load and validate a subtitle file
///
/// Performs path canonicalization with traversal protection,
/// loads subtitles using SubRipService, and returns the subtitles
/// along with the filename for display.
fn load_subtitle_file(file: &str) -> anyhow::Result<(Vec<Subtitle>, String)> {
    debug!("loading subtitle file: {}", file);

    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

    if file_path.is_relative() {
        let current_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("failed to get current directory: {}", e))?;

        let resolved = current_dir.join(file_path);
        let normalized = resolved
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve file path: {}", e))?;

        let canonical_current_dir = current_dir
            .canonicalize()
            .map_err(|e| anyhow::anyhow!("failed to resolve current directory: {}", e))?;

        if !normalized.starts_with(&canonical_current_dir) {
            error!("path traversal attempt detected: {:?}", file_path);
            return Err(anyhow::anyhow!(
                "invalid file path: path traversal not allowed"
            ));
        }
    }

    let canonical_path = file_path
        .canonicalize()
        .map_err(|e| anyhow::anyhow!("failed to resolve file path: {}", e))?;
    debug!("canonical path: {:?}", canonical_path);

    let base_dir = canonical_path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid file path"))?
        .to_path_buf();
    debug!("base directory: {:?}", base_dir);

    let filename = canonical_path
        .file_name()
        .ok_or_else(|| anyhow::anyhow!("invalid file name"))?
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("invalid UTF-8 in filename"))?
        .to_string();
    debug!("filename: {}", filename);

    let service = SubRipService::new(base_dir);
    debug!("loading subtitles from file: {}", filename);

    match service.get_all(&filename) {
        Ok(subtitles) => {
            info!(
                "successfully loaded {} subtitle(s) from {}",
                subtitles.len(),
                filename
            );
            Ok((subtitles, filename))
        }
        Err(e) => {
            debug!("error occurred: {:?}", e);
            match e {
                SubtitleError::FileNotFound(path) => {
                    error!("file not found: {}", path);
                    eprintln!("error: file not found: {}", path);
                }
                SubtitleError::InvalidPath(msg) => {
                    error!("invalid file path: {}", msg);
                    eprintln!("error: invalid file path: {}", msg);
                }
                SubtitleError::ParseError(err) => {
                    error!("parse error: {}", err);
                    eprintln!("error: failed to parse subtitle file: {}", err);
                    eprintln!("hint: try running 'sm doctor --fix {}' first", filename);
                }
                SubtitleError::IoError(err) => {
                    error!("i/o error: {}", err);
                    eprintln!("error: failed to read file: {}", err);
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
