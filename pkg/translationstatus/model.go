// Package translationstatus contains the translation status checking logic for
// the submarine subtitle translation toolkit. It is a 1-to-1 port of the Rust
// crate `lib::translation_status`.
package translationstatus

// TranslationStatusReport is the report structure for a translation status
// check. Was Rust struct TranslationStatusReport.
type TranslationStatusReport struct {
	// RefFile is the reference file name.
	RefFile string
	// TranslationFile is the translation file name.
	TranslationFile string
	// TotalCount is the total number of subtitles in the reference file.
	TotalCount int
	// TranslatedCount is the number of subtitles translated.
	TranslatedCount int
	// MissingCount is the number of subtitles remaining.
	MissingCount int
	// NextChunk is the suggestion for the next chunk to translate.
	NextChunk *ChunkSuggestion
}

// ChunkSuggestion is the suggestion for the next chunk of subtitles to
// translate. Was Rust struct ChunkSuggestion.
type ChunkSuggestion struct {
	// StartIndex is the starting index of the chunk.
	StartIndex uint32
	// EndIndex is the ending index of the chunk.
	EndIndex uint32
}

// ProgressPercentage calculates the translation progress as a percentage.
// Was TranslationStatusReport::progress_percentage.
func (r *TranslationStatusReport) ProgressPercentage() float64 {
	if r.TotalCount == 0 {
		return 0.0
	}
	return (float64(r.TranslatedCount) / float64(r.TotalCount)) * 100.0
}

// IsComplete reports whether the translation is complete.
// Was TranslationStatusReport::is_complete.
func (r *TranslationStatusReport) IsComplete() bool {
	return r.MissingCount == 0 && r.TotalCount > 0
}
