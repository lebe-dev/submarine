use lib::subtitle::model::Subtitle;
use log::{debug, error};

/// Parse range string in format "START-END" into (start, end) tuple
pub fn parse_range(range: &str) -> anyhow::Result<(u32, u32)> {
    let parts: Vec<&str> = range.split('-').collect();

    if parts.len() != 2 {
        error!("invalid range format: {}", range);
        eprintln!(
            "error: Invalid range format '{}'. Expected format: START-END (e.g., 1-50)",
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

    Ok((start, end))
}

/// Validate that the requested range is within available subtitle indices
pub fn validate_range_boundaries(
    start: u32,
    end: u32,
    subtitles: &[Subtitle],
) -> anyhow::Result<()> {
    if subtitles.is_empty() {
        error!("cannot process empty file");
        eprintln!("error: File contains no subtitles");
        std::process::exit(1);
    }

    let min_index = subtitles.iter().map(|s| *s.index.as_ref()).min().unwrap();

    let max_index = subtitles.iter().map(|s| *s.index.as_ref()).max().unwrap();

    debug!(
        "file contains indices {}-{}, requested range {}-{}",
        min_index, max_index, start, end
    );

    if start > max_index {
        error!(
            "start index {} exceeds maximum available index {}",
            start, max_index
        );
        eprintln!(
            "error: Start index {} is beyond the last subtitle (index {})",
            start, max_index
        );
        std::process::exit(1);
    }

    if end < min_index {
        error!(
            "end index {} is before minimum available index {}",
            end, min_index
        );
        eprintln!(
            "error: End index {} is before the first subtitle (index {})",
            end, min_index
        );
        std::process::exit(1);
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_range_valid() {
        let (start, end) = parse_range("1-50").unwrap();
        assert_eq!(start, 1);
        assert_eq!(end, 50);
    }

    #[test]
    fn test_parse_range_single() {
        let (start, end) = parse_range("42-42").unwrap();
        assert_eq!(start, 42);
        assert_eq!(end, 42);
    }

    #[test]
    fn test_parse_range_with_spaces() {
        let (start, end) = parse_range("10 - 20").unwrap();
        assert_eq!(start, 10);
        assert_eq!(end, 20);
    }
}
