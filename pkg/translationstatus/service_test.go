package translationstatus

import (
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// --- ported from model.rs #[cfg(test)] ---

func TestProgressPercentage(t *testing.T) {
	report := TranslationStatusReport{
		RefFile:         "ref.srt",
		TranslationFile: "trans.srt",
		TotalCount:      100,
		TranslatedCount: 50,
		MissingCount:    50,
		NextChunk:       nil,
	}

	if got := report.ProgressPercentage(); got != 50.0 {
		t.Fatalf("ProgressPercentage() = %v, want 50.0", got)
	}
}

func TestProgressPercentageZeroTotal(t *testing.T) {
	report := TranslationStatusReport{
		RefFile:         "ref.srt",
		TranslationFile: "trans.srt",
		TotalCount:      0,
		TranslatedCount: 0,
		MissingCount:    0,
		NextChunk:       nil,
	}

	if got := report.ProgressPercentage(); got != 0.0 {
		t.Fatalf("ProgressPercentage() = %v, want 0.0", got)
	}
}

func TestIsComplete(t *testing.T) {
	complete := TranslationStatusReport{
		RefFile:         "ref.srt",
		TranslationFile: "trans.srt",
		TotalCount:      100,
		TranslatedCount: 100,
		MissingCount:    0,
		NextChunk:       nil,
	}

	if !complete.IsComplete() {
		t.Fatalf("complete.IsComplete() = false, want true")
	}

	incomplete := TranslationStatusReport{
		RefFile:         "ref.srt",
		TranslationFile: "trans.srt",
		TotalCount:      100,
		TranslatedCount: 50,
		MissingCount:    50,
		NextChunk: &ChunkSuggestion{
			StartIndex: 51,
			EndIndex:   100,
		},
	}

	if incomplete.IsComplete() {
		t.Fatalf("incomplete.IsComplete() = true, want false")
	}
}

func TestIsCompleteEmptyFile(t *testing.T) {
	empty := TranslationStatusReport{
		RefFile:         "ref.srt",
		TranslationFile: "trans.srt",
		TotalCount:      0,
		TranslatedCount: 0,
		MissingCount:    0,
		NextChunk:       nil,
	}

	if empty.IsComplete() {
		t.Fatalf("empty.IsComplete() = true, want false")
	}
}

// --- ported from service.rs #[cfg(test)] ---

func makeTestSubtitle(t *testing.T, index uint32, startMs, endMs int64, text string) subtitle.Subtitle {
	t.Helper()
	idx, err := subtitle.NewSubtitleIndex(index)
	if err != nil {
		t.Fatalf("NewSubtitleIndex(%d): %v", index, err)
	}
	start, err := subtitle.NewSubtitleTimestamp(time.Duration(startMs) * time.Millisecond)
	if err != nil {
		t.Fatalf("NewSubtitleTimestamp(%d): %v", startMs, err)
	}
	end, err := subtitle.NewSubtitleTimestamp(time.Duration(endMs) * time.Millisecond)
	if err != nil {
		t.Fatalf("NewSubtitleTimestamp(%d): %v", endMs, err)
	}
	txt, err := subtitle.NewSubtitleText(text)
	if err != nil {
		t.Fatalf("NewSubtitleText(%q): %v", text, err)
	}
	s, err := subtitle.NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("NewSubtitle: %v", err)
	}
	return s
}

func TestEmptyTranslation(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Text 3"),
	}

	translationSubs := []subtitle.Subtitle{}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		50,
	)

	if report.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", report.TotalCount)
	}
	if report.TranslatedCount != 0 {
		t.Fatalf("TranslatedCount = %d, want 0", report.TranslatedCount)
	}
	if report.MissingCount != 3 {
		t.Fatalf("MissingCount = %d, want 3", report.MissingCount)
	}
	if report.ProgressPercentage() != 0.0 {
		t.Fatalf("ProgressPercentage() = %v, want 0.0", report.ProgressPercentage())
	}
	if report.IsComplete() {
		t.Fatalf("IsComplete() = true, want false")
	}

	if report.NextChunk == nil {
		t.Fatalf("NextChunk = nil, want some")
	}
	chunk := report.NextChunk
	if chunk.StartIndex != 1 {
		t.Fatalf("StartIndex = %d, want 1", chunk.StartIndex)
	}
	if chunk.EndIndex != 3 {
		t.Fatalf("EndIndex = %d, want 3", chunk.EndIndex)
	}
}

func TestPartialTranslation(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Text 3"),
		makeTestSubtitle(t, 4, 4000, 5000, "Text 4"),
		makeTestSubtitle(t, 5, 5000, 6000, "Text 5"),
	}

	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Translated 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Translated 2"),
	}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		2,
	)

	if report.TotalCount != 5 {
		t.Fatalf("TotalCount = %d, want 5", report.TotalCount)
	}
	if report.TranslatedCount != 2 {
		t.Fatalf("TranslatedCount = %d, want 2", report.TranslatedCount)
	}
	if report.MissingCount != 3 {
		t.Fatalf("MissingCount = %d, want 3", report.MissingCount)
	}
	if report.ProgressPercentage() != 40.0 {
		t.Fatalf("ProgressPercentage() = %v, want 40.0", report.ProgressPercentage())
	}
	if report.IsComplete() {
		t.Fatalf("IsComplete() = true, want false")
	}

	if report.NextChunk == nil {
		t.Fatalf("NextChunk = nil, want some")
	}
	chunk := report.NextChunk
	if chunk.StartIndex != 3 {
		t.Fatalf("StartIndex = %d, want 3", chunk.StartIndex)
	}
	if chunk.EndIndex != 4 {
		t.Fatalf("EndIndex = %d, want 4", chunk.EndIndex)
	}
}

func TestCompleteTranslation(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
	}

	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Translated 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Translated 2"),
	}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		50,
	)

	if report.TotalCount != 2 {
		t.Fatalf("TotalCount = %d, want 2", report.TotalCount)
	}
	if report.TranslatedCount != 2 {
		t.Fatalf("TranslatedCount = %d, want 2", report.TranslatedCount)
	}
	if report.MissingCount != 0 {
		t.Fatalf("MissingCount = %d, want 0", report.MissingCount)
	}
	if report.ProgressPercentage() != 100.0 {
		t.Fatalf("ProgressPercentage() = %v, want 100.0", report.ProgressPercentage())
	}
	if !report.IsComplete() {
		t.Fatalf("IsComplete() = false, want true")
	}
	if report.NextChunk != nil {
		t.Fatalf("NextChunk = %v, want nil", report.NextChunk)
	}
}

func TestTranslationWithGaps(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
		makeTestSubtitle(t, 5, 5000, 6000, "Text 5"),
		makeTestSubtitle(t, 10, 10000, 11000, "Text 10"),
	}

	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Translated 1"),
		makeTestSubtitle(t, 10, 10000, 11000, "Translated 10"),
	}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		50,
	)

	if report.TotalCount != 4 {
		t.Fatalf("TotalCount = %d, want 4", report.TotalCount)
	}
	if report.TranslatedCount != 2 {
		t.Fatalf("TranslatedCount = %d, want 2", report.TranslatedCount)
	}
	if report.MissingCount != 2 {
		t.Fatalf("MissingCount = %d, want 2", report.MissingCount)
	}
	if report.ProgressPercentage() != 50.0 {
		t.Fatalf("ProgressPercentage() = %v, want 50.0", report.ProgressPercentage())
	}

	if report.NextChunk == nil {
		t.Fatalf("NextChunk = nil, want some")
	}
	chunk := report.NextChunk
	if chunk.StartIndex != 2 {
		t.Fatalf("StartIndex = %d, want 2", chunk.StartIndex)
	}
	if chunk.EndIndex != 2 { // Stops at the gap before index 5
		t.Fatalf("EndIndex = %d, want 2", chunk.EndIndex)
	}
}

func TestChunkSizeLargerThanRemaining(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Text 3"),
	}

	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Translated 1"),
	}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		100,
	)

	if report.MissingCount != 2 {
		t.Fatalf("MissingCount = %d, want 2", report.MissingCount)
	}

	if report.NextChunk == nil {
		t.Fatalf("NextChunk = nil, want some")
	}
	chunk := report.NextChunk
	if chunk.StartIndex != 2 {
		t.Fatalf("StartIndex = %d, want 2", chunk.StartIndex)
	}
	if chunk.EndIndex != 3 { // Capped at last missing index
		t.Fatalf("EndIndex = %d, want 3", chunk.EndIndex)
	}
}

func TestEmptyFiles(t *testing.T) {
	refSubs := []subtitle.Subtitle{}
	translationSubs := []subtitle.Subtitle{}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		50,
	)

	if report.TotalCount != 0 {
		t.Fatalf("TotalCount = %d, want 0", report.TotalCount)
	}
	if report.TranslatedCount != 0 {
		t.Fatalf("TranslatedCount = %d, want 0", report.TranslatedCount)
	}
	if report.MissingCount != 0 {
		t.Fatalf("MissingCount = %d, want 0", report.MissingCount)
	}
	if report.ProgressPercentage() != 0.0 {
		t.Fatalf("ProgressPercentage() = %v, want 0.0", report.ProgressPercentage())
	}
	if report.IsComplete() {
		t.Fatalf("IsComplete() = true, want false")
	}
	if report.NextChunk != nil {
		t.Fatalf("NextChunk = %v, want nil", report.NextChunk)
	}
}

func TestNonSequentialIndices(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 1000, 2000, "Text 10"),
		makeTestSubtitle(t, 20, 2000, 3000, "Text 20"),
		makeTestSubtitle(t, 30, 3000, 4000, "Text 30"),
	}

	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 1000, 2000, "Translated 10"),
	}

	report := CheckTranslationStatus(
		refSubs,
		"ref.srt",
		translationSubs,
		"trans.srt",
		5,
	)

	if report.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", report.TotalCount)
	}
	if report.TranslatedCount != 1 {
		t.Fatalf("TranslatedCount = %d, want 1", report.TranslatedCount)
	}
	if report.MissingCount != 2 {
		t.Fatalf("MissingCount = %d, want 2", report.MissingCount)
	}

	if report.NextChunk == nil {
		t.Fatalf("NextChunk = nil, want some")
	}
	chunk := report.NextChunk
	if chunk.StartIndex != 20 {
		t.Fatalf("StartIndex = %d, want 20", chunk.StartIndex)
	}
	if chunk.EndIndex != 20 { // No next index 21, stops at 20
		t.Fatalf("EndIndex = %d, want 20", chunk.EndIndex)
	}
}
