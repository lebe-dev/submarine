package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
	"github.com/lebe-dev/submarine/pkg/translationstatus"
)

// makeTestSubtitle mirrors the Rust `make_test_subtitle` helper.
func makeTestSubtitle(t *testing.T, index uint32, startMs, endMs int64, text string) subtitle.Subtitle {
	t.Helper()
	idx, err := subtitle.NewSubtitleIndex(index)
	if err != nil {
		t.Fatalf("invalid index: %v", err)
	}
	start, err := subtitle.NewSubtitleTimestamp(time.Duration(startMs) * time.Millisecond)
	if err != nil {
		t.Fatalf("invalid start: %v", err)
	}
	end, err := subtitle.NewSubtitleTimestamp(time.Duration(endMs) * time.Millisecond)
	if err != nil {
		t.Fatalf("invalid end: %v", err)
	}
	txt, err := subtitle.NewSubtitleText(text)
	if err != nil {
		t.Fatalf("invalid text: %v", err)
	}
	sub, err := subtitle.NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("invalid subtitle: %v", err)
	}
	return sub
}

// createSrtFile mirrors the Rust `create_srt_file` helper: writes an empty SRT
// file in dir, then adds each subtitle through the SubRipService. Returns the
// full path to the created file.
func createSrtFile(t *testing.T, dir, filename string, subtitles []subtitle.Subtitle) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	service := subtitle.NewSubRipService(dir)

	if err := os.WriteFile(filePath, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", filePath, err)
	}

	for _, sub := range subtitles {
		if _, err := service.Add(filename, sub.StartTime, sub.EndTime, sub.Text); err != nil {
			t.Fatalf("failed to add subtitle: %v", err)
		}
	}

	return filePath
}

// TestIntegrationEmptyTranslationFile ports `test_empty_translation_file`.
func TestIntegrationEmptyTranslationFile(t *testing.T) {
	dir := t.TempDir()

	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "English 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "English 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "English 3"),
	}
	var translationSubs []subtitle.Subtitle

	createSrtFile(t, dir, "reference.srt", refSubs)
	createSrtFile(t, dir, "translation.srt", translationSubs)

	report := translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 50,
	)

	if report.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", report.TotalCount)
	}
	if report.TranslatedCount != 0 {
		t.Errorf("TranslatedCount = %d, want 0", report.TranslatedCount)
	}
	if report.MissingCount != 3 {
		t.Errorf("MissingCount = %d, want 3", report.MissingCount)
	}
	if report.ProgressPercentage() != 0.0 {
		t.Errorf("ProgressPercentage = %v, want 0.0", report.ProgressPercentage())
	}
	if report.IsComplete() {
		t.Error("IsComplete = true, want false")
	}
	if report.NextChunk == nil {
		t.Fatal("NextChunk should be set")
	}
	if report.NextChunk.StartIndex != 1 {
		t.Errorf("chunk.StartIndex = %d, want 1", report.NextChunk.StartIndex)
	}
	if report.NextChunk.EndIndex != 3 {
		t.Errorf("chunk.EndIndex = %d, want 3", report.NextChunk.EndIndex)
	}
}

// TestIntegrationPartialTranslation ports `test_partial_translation`.
func TestIntegrationPartialTranslation(t *testing.T) {
	dir := t.TempDir()

	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "English 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "English 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "English 3"),
		makeTestSubtitle(t, 4, 4000, 5000, "English 4"),
		makeTestSubtitle(t, 5, 5000, 6000, "English 5"),
	}
	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Russian 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Russian 2"),
	}

	createSrtFile(t, dir, "reference.srt", refSubs)
	createSrtFile(t, dir, "translation.srt", translationSubs)

	report := translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 50,
	)

	if report.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", report.TotalCount)
	}
	if report.TranslatedCount != 2 {
		t.Errorf("TranslatedCount = %d, want 2", report.TranslatedCount)
	}
	if report.MissingCount != 3 {
		t.Errorf("MissingCount = %d, want 3", report.MissingCount)
	}
	if report.ProgressPercentage() != 40.0 {
		t.Errorf("ProgressPercentage = %v, want 40.0", report.ProgressPercentage())
	}
	if report.IsComplete() {
		t.Error("IsComplete = true, want false")
	}
	if report.NextChunk == nil {
		t.Fatal("NextChunk should be set")
	}
	if report.NextChunk.StartIndex != 3 {
		t.Errorf("chunk.StartIndex = %d, want 3", report.NextChunk.StartIndex)
	}
	if report.NextChunk.EndIndex != 5 {
		t.Errorf("chunk.EndIndex = %d, want 5", report.NextChunk.EndIndex)
	}
}

// TestIntegrationCompleteTranslation ports `test_complete_translation`.
func TestIntegrationCompleteTranslation(t *testing.T) {
	dir := t.TempDir()

	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "English 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "English 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "English 3"),
	}
	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Russian 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Russian 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Russian 3"),
	}

	createSrtFile(t, dir, "reference.srt", refSubs)
	createSrtFile(t, dir, "translation.srt", translationSubs)

	report := translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 50,
	)

	if report.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", report.TotalCount)
	}
	if report.TranslatedCount != 3 {
		t.Errorf("TranslatedCount = %d, want 3", report.TranslatedCount)
	}
	if report.MissingCount != 0 {
		t.Errorf("MissingCount = %d, want 0", report.MissingCount)
	}
	if report.ProgressPercentage() != 100.0 {
		t.Errorf("ProgressPercentage = %v, want 100.0", report.ProgressPercentage())
	}
	if !report.IsComplete() {
		t.Error("IsComplete = false, want true")
	}
	if report.NextChunk != nil {
		t.Errorf("NextChunk should be nil, got %+v", report.NextChunk)
	}
}

// TestIntegrationTranslationWithGaps ports `test_translation_with_gaps`.
func TestIntegrationTranslationWithGaps(t *testing.T) {
	dir := t.TempDir()

	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "English 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "English 2"),
		makeTestSubtitle(t, 5, 5000, 6000, "English 5"),
		makeTestSubtitle(t, 10, 10000, 11000, "English 10"),
	}
	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Russian 1"),
		makeTestSubtitle(t, 10, 10000, 11000, "Russian 10"),
	}

	createSrtFile(t, dir, "reference.srt", refSubs)
	createSrtFile(t, dir, "translation.srt", translationSubs)

	report := translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 50,
	)

	if report.TotalCount != 4 {
		t.Errorf("TotalCount = %d, want 4", report.TotalCount)
	}
	if report.TranslatedCount != 2 {
		t.Errorf("TranslatedCount = %d, want 2", report.TranslatedCount)
	}
	if report.MissingCount != 2 {
		t.Errorf("MissingCount = %d, want 2", report.MissingCount)
	}
	if report.ProgressPercentage() != 50.0 {
		t.Errorf("ProgressPercentage = %v, want 50.0", report.ProgressPercentage())
	}
	if report.NextChunk == nil {
		t.Fatal("NextChunk should be set")
	}
	if report.NextChunk.StartIndex != 2 {
		t.Errorf("chunk.StartIndex = %d, want 2", report.NextChunk.StartIndex)
	}
	if report.NextChunk.EndIndex != 2 {
		t.Errorf("chunk.EndIndex = %d, want 2", report.NextChunk.EndIndex)
	}
}

// TestIntegrationCustomChunkSize ports `test_custom_chunk_size`.
func TestIntegrationCustomChunkSize(t *testing.T) {
	dir := t.TempDir()

	refSubs := make([]subtitle.Subtitle, 0, 100)
	for i := int64(1); i <= 100; i++ {
		refSubs = append(refSubs, makeTestSubtitle(t, uint32(i), i*1000, (i+1)*1000, "Text"))
	}
	var translationSubs []subtitle.Subtitle

	createSrtFile(t, dir, "reference.srt", refSubs)
	createSrtFile(t, dir, "translation.srt", translationSubs)

	// chunk size 25
	report := translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 25,
	)
	if report.NextChunk == nil {
		t.Fatal("NextChunk should be set (chunk size 25)")
	}
	if report.NextChunk.StartIndex != 1 {
		t.Errorf("chunk.StartIndex = %d, want 1", report.NextChunk.StartIndex)
	}
	if report.NextChunk.EndIndex != 25 {
		t.Errorf("chunk.EndIndex = %d, want 25", report.NextChunk.EndIndex)
	}

	// chunk size 10
	report = translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 10,
	)
	if report.NextChunk == nil {
		t.Fatal("NextChunk should be set (chunk size 10)")
	}
	if report.NextChunk.StartIndex != 1 {
		t.Errorf("chunk.StartIndex = %d, want 1", report.NextChunk.StartIndex)
	}
	if report.NextChunk.EndIndex != 10 {
		t.Errorf("chunk.EndIndex = %d, want 10", report.NextChunk.EndIndex)
	}
}

// TestIntegrationChunkSizeLargerThanRemaining ports
// `test_chunk_size_larger_than_remaining`.
func TestIntegrationChunkSizeLargerThanRemaining(t *testing.T) {
	dir := t.TempDir()

	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "English 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "English 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "English 3"),
	}
	translationSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Russian 1"),
	}

	createSrtFile(t, dir, "reference.srt", refSubs)
	createSrtFile(t, dir, "translation.srt", translationSubs)

	report := translationstatus.CheckTranslationStatus(
		refSubs, "reference.srt", translationSubs, "translation.srt", 100,
	)

	if report.NextChunk == nil {
		t.Fatal("NextChunk should be set")
	}
	if report.NextChunk.StartIndex != 2 {
		t.Errorf("chunk.StartIndex = %d, want 2", report.NextChunk.StartIndex)
	}
	if report.NextChunk.EndIndex != 3 {
		t.Errorf("chunk.EndIndex = %d, want 3 (capped at last missing index)", report.NextChunk.EndIndex)
	}
}
