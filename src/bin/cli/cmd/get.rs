use lib::subtitle::model::SubtitleError;
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use log::{debug, error, info};
use std::path::Path;

pub fn handle(file: &str, index: u32) -> anyhow::Result<()> {
    info!("getting subtitle {} from file: {}", index, file);

    let file_path = Path::new(file);
    debug!("parsing file path: {:?}", file_path);

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
            }
            std::process::exit(1);
        }
    }
}
