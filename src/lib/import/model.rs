/// Represents a parsed CSV row before validation and conversion to Subtitle
#[derive(Debug)]
pub struct CsvSubtitleRow {
    pub line_number: usize,
    pub start_time: String,
    pub end_time: String,
    pub text: String,
}

/// Represents a parsed anchored format row before validation
#[derive(Debug)]
pub struct AnchoredSubtitleRow {
    pub line_number: usize,
    pub index: u32,
    pub text: String,
}
