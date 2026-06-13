package cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// Percentage is an f64 whose JSON encoding matches Rust's serde_json (via ryu):
// floating-point numbers always carry a decimal point, so whole values render as
// "100.0" rather than Go's default "100". NaN/Inf encode as null, like serde_json.
type Percentage float64

// MarshalJSON renders the value the way serde_json does for an f64.
func (p Percentage) MarshalJSON() ([]byte, error) {
	f := float64(p)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return []byte(s), nil
}

// -- Subtitle DTO --

// SubtitleDto is a port of the Rust `struct SubtitleDto`.
type SubtitleDto struct {
	Index       uint32 `json:"index"`
	StartTime   string `json:"start_time"`
	StartTimeMs int64  `json:"start_time_ms"`
	EndTime     string `json:"end_time"`
	EndTimeMs   int64  `json:"end_time_ms"`
	DurationMs  int64  `json:"duration_ms"`
	Text        string `json:"text"`
	HasHTML     bool   `json:"has_html"`
}

// SubtitleDtoFromSubtitle is a port of `SubtitleDto::from_subtitle`.
func SubtitleDtoFromSubtitle(s *subtitle.Subtitle) SubtitleDto {
	return SubtitleDto{
		Index:       s.Index.Value(),
		StartTime:   subtitle.FormatTimestamp(s.StartTime.Value()),
		StartTimeMs: s.StartTime.Millis(),
		EndTime:     subtitle.FormatTimestamp(s.EndTime.Value()),
		EndTimeMs:   s.EndTime.Millis(),
		DurationMs:  int64(s.Duration() / time.Millisecond),
		Text:        s.Text.Value(),
		HasHTML:     s.HasHTMLTags(),
	}
}

// -- Info DTO --

// InfoDto is a port of the Rust `struct InfoDto`.
type InfoDto struct {
	File                      string `json:"file"`
	TotalCount                int    `json:"total_count"`
	TotalDurationMs           int64  `json:"total_duration_ms"`
	TotalDuration             string `json:"total_duration"`
	AverageSubtitleDurationMs int64  `json:"average_subtitle_duration_ms"`
	AverageGapMs              *int64 `json:"average_gap_ms"`
	TotalCharacters           int    `json:"total_characters"`
	TotalCharactersNoHTML     int    `json:"total_characters_no_html"`
	TotalWords                int    `json:"total_words"`
	TotalLines                int    `json:"total_lines"`
	SubtitlesWithHTML         int    `json:"subtitles_with_html"`
}

// -- Verify DTO --

// VerifyDto is a port of the Rust `struct VerifyDto`.
type VerifyDto struct {
	RefFile             string     `json:"ref_file"`
	TargetFile          string     `json:"target_file"`
	TotalRefCount       int        `json:"total_ref_count"`
	TotalTargetCount    int        `json:"total_target_count"`
	PerfectMatches      int        `json:"perfect_matches"`
	TotalMatched        int        `json:"total_matched"`
	MatchPercentage     Percentage `json:"match_percentage"`
	TimestampMismatches int        `json:"timestamp_mismatches"`
	MissingInTarget     int        `json:"missing_in_target"`
	ExtraInTarget       int        `json:"extra_in_target"`
	DetectedOffset      *int32     `json:"detected_offset"`
	Status              string     `json:"status"`
}

// -- Translation Status DTO --

// TranslationStatusDto is a port of the Rust `struct TranslationStatusDto`.
type TranslationStatusDto struct {
	RefFile            string     `json:"ref_file"`
	TranslationFile    string     `json:"translation_file"`
	TotalCount         int        `json:"total_count"`
	TranslatedCount    int        `json:"translated_count"`
	MissingCount       int        `json:"missing_count"`
	ProgressPercentage Percentage `json:"progress_percentage"`
	IsComplete         bool       `json:"is_complete"`
	NextChunk          *ChunkDto  `json:"next_chunk"`
}

// ChunkDto is a port of the Rust `struct ChunkDto`.
type ChunkDto struct {
	StartIndex uint32 `json:"start_index"`
	EndIndex   uint32 `json:"end_index"`
}

// -- Doctor DTOs --

// DiagnosticDto is a port of the Rust `struct DiagnosticDto`.
type DiagnosticDto struct {
	FilePath     string     `json:"file_path"`
	TotalLines   int        `json:"total_lines"`
	TotalBlocks  int        `json:"total_blocks"`
	IsParsable   bool       `json:"is_parsable"`
	ErrorCount   int        `json:"error_count"`
	WarningCount int        `json:"warning_count"`
	HasIssues    bool       `json:"has_issues"`
	Issues       []IssueDto `json:"issues"`
}

// IssueDto is a port of the Rust `struct IssueDto`.
type IssueDto struct {
	LineNumber  int    `json:"line_number"`
	BlockNumber *int   `json:"block_number"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// FixDto is a port of the Rust `struct FixDto`.
type FixDto struct {
	OriginalPath      string     `json:"original_path"`
	BackupPath        string     `json:"backup_path"`
	IssuesFixed       int        `json:"issues_fixed"`
	IssuesUnfixable   int        `json:"issues_unfixable"`
	ValidationSuccess bool       `json:"validation_success"`
	UnfixableIssues   []IssueDto `json:"unfixable_issues"`
}

// -- Mutation result DTOs --

// SetResultDto is a port of the Rust `struct SetResultDto`.
type SetResultDto struct {
	Index         uint32   `json:"index"`
	FieldsUpdated []string `json:"fields_updated"`
	BackupPath    string   `json:"backup_path"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// AddResultDto is a port of the Rust `struct AddResultDto`.
type AddResultDto struct {
	NewIndex       uint32 `json:"new_index"`
	TotalSubtitles int    `json:"total_subtitles"`
	BackupPath     string `json:"backup_path"`
	DryRun         bool   `json:"dry_run,omitempty"`
}

// ImportResultDto is a port of the Rust `struct ImportResultDto`.
type ImportResultDto struct {
	ImportedCount  int    `json:"imported_count"`
	StartIndex     uint32 `json:"start_index"`
	EndIndex       uint32 `json:"end_index"`
	TotalSubtitles int    `json:"total_subtitles"`
	BackupPath     string `json:"backup_path"`
	DryRun         bool   `json:"dry_run,omitempty"`
}

// DelayResultDto is a port of the Rust `struct DelayResultDto`.
type DelayResultDto struct {
	OffsetMs          int64         `json:"offset_ms"`
	SubtitlesAdjusted int           `json:"subtitles_adjusted"`
	BackupPath        string        `json:"backup_path"`
	DryRun            bool          `json:"dry_run,omitempty"`
	SampleBefore      []SubtitleDto `json:"sample_before,omitempty"`
	SampleAfter       []SubtitleDto `json:"sample_after,omitempty"`
}

// MassRenameResultDto is a port of the Rust `struct MassRenameResultDto`.
type MassRenameResultDto struct {
	RenamedCount int                  `json:"renamed_count"`
	SkippedCount int                  `json:"skipped_count"`
	Operations   []RenameOperationDto `json:"operations"`
	DryRun       bool                 `json:"dry_run,omitempty"`
}

// RenameOperationDto is a port of the Rust `struct RenameOperationDto`.
type RenameOperationDto struct {
	Original  string `json:"original"`
	NewName   string `json:"new_name"`
	Collision bool   `json:"collision"`
}

// -- Export DTO --

// ExportDto is a port of the Rust `struct ExportDto`.
type ExportDto struct {
	Subtitles  []SubtitleDto `json:"subtitles"`
	Format     string        `json:"format"`
	RangeStart uint32        `json:"range_start"`
	RangeEnd   uint32        `json:"range_end"`
	Count      int           `json:"count"`
}

// -- Describe DTOs --

// DescribeDto is a port of the Rust `struct DescribeDto`.
type DescribeDto struct {
	Tool     string           `json:"tool"`
	Version  string           `json:"version"`
	Commands []CommandDescDto `json:"commands"`
}

// CommandDescDto is a port of the Rust `struct CommandDescDto`.
type CommandDescDto struct {
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Mutates        bool         `json:"mutates"`
	SupportsDryRun bool         `json:"supports_dry_run"`
	SupportsJSON   bool         `json:"supports_json"`
	Args           []ArgDescDto `json:"args"`
}

// ArgDescDto is a port of the Rust `struct ArgDescDto`.
type ArgDescDto struct {
	Name        string `json:"name"`
	ArgType     string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// -- Helper to format duration to human-readable string --

// FormatDurationReadable is a port of the Rust `format_duration_readable`.
func FormatDurationReadable(duration time.Duration) string {
	totalSeconds := int64(duration / time.Second)

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	milliseconds := int64(duration/time.Millisecond) % 1000

	switch {
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	case seconds > 0:
		return fmt.Sprintf("%d.%03ds", seconds, milliseconds)
	default:
		return fmt.Sprintf("%dms", milliseconds)
	}
}
