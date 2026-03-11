use chrono::Duration;
use lib::subtitle::model::Subtitle;
use serde::Serialize;

// -- Subtitle DTO --

#[derive(Serialize)]
pub struct SubtitleDto {
    pub index: u32,
    pub start_time: String,
    pub start_time_ms: i64,
    pub end_time: String,
    pub end_time_ms: i64,
    pub duration_ms: i64,
    pub text: String,
    pub has_html: bool,
}

impl SubtitleDto {
    pub fn from_subtitle(s: &Subtitle) -> Self {
        SubtitleDto {
            index: *s.index.as_ref(),
            start_time: Subtitle::format_timestamp(s.start_time.as_ref()),
            start_time_ms: s.start_time.as_ref().num_milliseconds(),
            end_time: Subtitle::format_timestamp(s.end_time.as_ref()),
            end_time_ms: s.end_time.as_ref().num_milliseconds(),
            duration_ms: s.duration().num_milliseconds(),
            text: s.text.as_ref().to_string(),
            has_html: s.has_html_tags(),
        }
    }
}

// -- Info DTO --

#[derive(Serialize)]
pub struct InfoDto {
    pub file: String,
    pub total_count: usize,
    pub total_duration_ms: i64,
    pub total_duration: String,
    pub average_subtitle_duration_ms: i64,
    pub average_gap_ms: Option<i64>,
    pub total_characters: usize,
    pub total_characters_no_html: usize,
    pub total_words: usize,
    pub total_lines: usize,
    pub subtitles_with_html: usize,
}

// -- Verify DTO --

#[derive(Serialize)]
pub struct VerifyDto {
    pub ref_file: String,
    pub target_file: String,
    pub total_ref_count: usize,
    pub total_target_count: usize,
    pub perfect_matches: usize,
    pub total_matched: usize,
    pub match_percentage: f64,
    pub timestamp_mismatches: usize,
    pub missing_in_target: usize,
    pub extra_in_target: usize,
    pub detected_offset: Option<i32>,
    pub status: String,
}

// -- Translation Status DTO --

#[derive(Serialize)]
pub struct TranslationStatusDto {
    pub ref_file: String,
    pub translation_file: String,
    pub total_count: usize,
    pub translated_count: usize,
    pub missing_count: usize,
    pub progress_percentage: f64,
    pub is_complete: bool,
    pub next_chunk: Option<ChunkDto>,
}

#[derive(Serialize)]
pub struct ChunkDto {
    pub start_index: u32,
    pub end_index: u32,
}

// -- Doctor DTOs --

#[derive(Serialize)]
pub struct DiagnosticDto {
    pub file_path: String,
    pub total_lines: usize,
    pub total_blocks: usize,
    pub is_parsable: bool,
    pub error_count: usize,
    pub warning_count: usize,
    pub has_issues: bool,
    pub issues: Vec<IssueDto>,
}

#[derive(Serialize)]
pub struct IssueDto {
    pub line_number: usize,
    pub block_number: Option<usize>,
    pub severity: String,
    pub description: String,
}

#[derive(Serialize)]
pub struct FixDto {
    pub original_path: String,
    pub backup_path: String,
    pub issues_fixed: usize,
    pub issues_unfixable: usize,
    pub validation_success: bool,
    pub unfixable_issues: Vec<IssueDto>,
}

// -- Mutation result DTOs --

#[derive(Serialize)]
pub struct SetResultDto {
    pub index: u32,
    pub fields_updated: Vec<String>,
    pub backup_path: String,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub dry_run: bool,
}

#[derive(Serialize)]
pub struct AddResultDto {
    pub new_index: u32,
    pub total_subtitles: usize,
    pub backup_path: String,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub dry_run: bool,
}

#[derive(Serialize)]
pub struct ImportResultDto {
    pub imported_count: usize,
    pub start_index: u32,
    pub end_index: u32,
    pub total_subtitles: usize,
    pub backup_path: String,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub dry_run: bool,
}

#[derive(Serialize)]
pub struct DelayResultDto {
    pub offset_ms: i64,
    pub subtitles_adjusted: usize,
    pub backup_path: String,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub dry_run: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sample_before: Option<Vec<SubtitleDto>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sample_after: Option<Vec<SubtitleDto>>,
}

#[derive(Serialize)]
pub struct MassRenameResultDto {
    pub renamed_count: usize,
    pub skipped_count: usize,
    pub operations: Vec<RenameOperationDto>,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub dry_run: bool,
}

#[derive(Serialize)]
pub struct RenameOperationDto {
    pub original: String,
    pub new_name: String,
    pub collision: bool,
}

// -- Export DTO --

#[derive(Serialize)]
pub struct ExportDto {
    pub subtitles: Vec<SubtitleDto>,
    pub format: String,
    pub range_start: u32,
    pub range_end: u32,
    pub count: usize,
}

// -- Describe DTOs --

#[derive(Serialize)]
pub struct DescribeDto {
    pub tool: String,
    pub version: String,
    pub commands: Vec<CommandDescDto>,
}

#[derive(Serialize)]
pub struct CommandDescDto {
    pub name: String,
    pub description: String,
    pub mutates: bool,
    pub supports_dry_run: bool,
    pub supports_json: bool,
    pub args: Vec<ArgDescDto>,
}

#[derive(Serialize)]
pub struct ArgDescDto {
    pub name: String,
    #[serde(rename = "type")]
    pub arg_type: String,
    pub required: bool,
    pub description: String,
}

// -- Helper to format duration to human-readable string --

pub fn format_duration_readable(duration: &Duration) -> String {
    let total_seconds = duration.num_seconds();

    let hours = total_seconds / 3600;
    let minutes = (total_seconds % 3600) / 60;
    let seconds = total_seconds % 60;
    let milliseconds = duration.num_milliseconds() % 1000;

    if hours > 0 {
        format!("{}h {}m {}s", hours, minutes, seconds)
    } else if minutes > 0 {
        format!("{}m {}s", minutes, seconds)
    } else if seconds > 0 {
        format!("{}.{:03}s", seconds, milliseconds)
    } else {
        format!("{}ms", milliseconds)
    }
}
