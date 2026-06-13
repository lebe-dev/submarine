package verify

import (
	"math"
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// makeTestSubtitle mirrors the Rust test helper make_test_subtitle.
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
	sub, err := subtitle.NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("NewSubtitle: %v", err)
	}
	return sub
}

// --- model.rs tests ---

func TestVerificationReportIsPerfect(t *testing.T) {
	report := VerificationReport{
		RefFile:             "ref.srt",
		TargetFile:          "target.srt",
		TotalRefCount:       10,
		TotalTargetCount:    10,
		PerfectMatches:      10,
		TimestampMismatches: nil,
		MissingInTarget:     nil,
		MatchedWithOffset:   nil,
		DetectedOffset:      nil,
		ExtraInTarget:       nil,
	}

	if !report.IsPerfect() {
		t.Errorf("expected IsPerfect() == true")
	}
	if report.HasIssues() {
		t.Errorf("expected HasIssues() == false")
	}
	if report.MatchPercentage() != 100.0 {
		t.Errorf("expected MatchPercentage() == 100.0, got %v", report.MatchPercentage())
	}
}

func TestVerificationReportHasIssues(t *testing.T) {
	sub := makeTestSubtitle(t, 1, 1000, 2000, "Test")
	entry := ComparisonEntry{
		RefSubtitle: sub,
		Status:      ComparisonStatus{Kind: MissingInTarget},
	}

	report := VerificationReport{
		RefFile:             "ref.srt",
		TargetFile:          "target.srt",
		TotalRefCount:       10,
		TotalTargetCount:    9,
		PerfectMatches:      9,
		TimestampMismatches: nil,
		MissingInTarget:     []ComparisonEntry{entry},
		MatchedWithOffset:   nil,
		DetectedOffset:      nil,
		ExtraInTarget:       nil,
	}

	if report.IsPerfect() {
		t.Errorf("expected IsPerfect() == false")
	}
	if !report.HasIssues() {
		t.Errorf("expected HasIssues() == true")
	}
	if report.MatchPercentage() != 90.0 {
		t.Errorf("expected MatchPercentage() == 90.0, got %v", report.MatchPercentage())
	}
}

func TestVerificationReportWithOffset(t *testing.T) {
	sub := makeTestSubtitle(t, 1, 1000, 2000, "Test")
	entry := ComparisonEntry{
		RefSubtitle: sub,
		Status: ComparisonStatus{
			Kind:        MatchedWithOffset,
			Offset:      -2,
			TargetIndex: 3,
		},
	}

	offset := int32(-2)
	report := VerificationReport{
		RefFile:             "ref.srt",
		TargetFile:          "target.srt",
		TotalRefCount:       10,
		TotalTargetCount:    10,
		PerfectMatches:      9,
		TimestampMismatches: nil,
		MissingInTarget:     nil,
		MatchedWithOffset:   []ComparisonEntry{entry},
		DetectedOffset:      &offset,
		ExtraInTarget:       nil,
	}

	if report.IsPerfect() {
		t.Errorf("expected IsPerfect() == false")
	}
	if report.HasIssues() {
		t.Errorf("expected HasIssues() == false")
	}
	if report.TotalMatched() != 10 {
		t.Errorf("expected TotalMatched() == 10, got %d", report.TotalMatched())
	}
	if report.MatchPercentage() != 100.0 {
		t.Errorf("expected MatchPercentage() == 100.0, got %v", report.MatchPercentage())
	}
}

func TestMatchPercentageZeroSubtitles(t *testing.T) {
	report := VerificationReport{
		RefFile:             "ref.srt",
		TargetFile:          "target.srt",
		TotalRefCount:       0,
		TotalTargetCount:    0,
		PerfectMatches:      0,
		TimestampMismatches: nil,
		MissingInTarget:     nil,
		MatchedWithOffset:   nil,
		DetectedOffset:      nil,
		ExtraInTarget:       nil,
	}

	if report.MatchPercentage() != 0.0 {
		t.Errorf("expected MatchPercentage() == 0.0, got %v", report.MatchPercentage())
	}
}

// --- service.rs tests ---

func TestTimestampsMatch(t *testing.T) {
	s1 := makeTestSubtitle(t, 1, 1000, 2000, "Text 1")
	s2 := makeTestSubtitle(t, 2, 1000, 2000, "Text 2")
	s3 := makeTestSubtitle(t, 3, 1000, 2001, "Text 3")

	if !timestampsMatch(&s1, &s2) {
		t.Errorf("expected timestampsMatch(s1, s2) == true")
	}
	if timestampsMatch(&s1, &s3) {
		t.Errorf("expected timestampsMatch(s1, s3) == false")
	}
}

func TestComparePerfectMatch(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "English 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "English 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "English 3"),
	}

	targetSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Russian 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Russian 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Russian 3"),
	}

	report := CompareSubtitles(refSubs, "ref.srt", targetSubs, "target.srt")

	if report.PerfectMatches != 3 {
		t.Errorf("expected PerfectMatches == 3, got %d", report.PerfectMatches)
	}
	if len(report.TimestampMismatches) != 0 {
		t.Errorf("expected TimestampMismatches empty, got %d", len(report.TimestampMismatches))
	}
	if len(report.MissingInTarget) != 0 {
		t.Errorf("expected MissingInTarget empty, got %d", len(report.MissingInTarget))
	}
	if len(report.MatchedWithOffset) != 0 {
		t.Errorf("expected MatchedWithOffset empty, got %d", len(report.MatchedWithOffset))
	}
	if len(report.ExtraInTarget) != 0 {
		t.Errorf("expected ExtraInTarget empty, got %d", len(report.ExtraInTarget))
	}
	if report.DetectedOffset != nil {
		t.Errorf("expected DetectedOffset == nil, got %v", *report.DetectedOffset)
	}
	if !report.IsPerfect() {
		t.Errorf("expected IsPerfect() == true")
	}
	if report.MatchPercentage() != 100.0 {
		t.Errorf("expected MatchPercentage() == 100.0, got %v", report.MatchPercentage())
	}
}

func TestCompareMissingSubtitles(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Text 3"),
	}

	targetSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
	}

	report := CompareSubtitles(refSubs, "ref.srt", targetSubs, "target.srt")

	if report.PerfectMatches != 1 {
		t.Errorf("expected PerfectMatches == 1, got %d", report.PerfectMatches)
	}
	if len(report.MissingInTarget) != 2 {
		t.Errorf("expected MissingInTarget len == 2, got %d", len(report.MissingInTarget))
	}
	if math.Abs(report.MatchPercentage()-33.333333333333336) >= 0.0001 {
		t.Errorf("expected MatchPercentage() ~ 33.333..., got %v", report.MatchPercentage())
	}
}

func TestCompareTimestampMismatch(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
	}

	targetSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2100, 3100, "Text 2"), // Different timestamps
	}

	report := CompareSubtitles(refSubs, "ref.srt", targetSubs, "target.srt")

	if report.PerfectMatches != 1 {
		t.Errorf("expected PerfectMatches == 1, got %d", report.PerfectMatches)
	}
	if len(report.TimestampMismatches) != 1 {
		t.Errorf("expected TimestampMismatches len == 1, got %d", len(report.TimestampMismatches))
	}
	if report.MatchPercentage() != 50.0 {
		t.Errorf("expected MatchPercentage() == 50.0, got %v", report.MatchPercentage())
	}
}

func TestCompareWithOffset(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 5, 1000, 2000, "English 5"),
		makeTestSubtitle(t, 6, 2000, 3000, "English 6"),
		makeTestSubtitle(t, 7, 3000, 4000, "English 7"),
	}

	// Target has indices offset by -2
	targetSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 3, 1000, 2000, "Russian 3"),
		makeTestSubtitle(t, 4, 2000, 3000, "Russian 4"),
		makeTestSubtitle(t, 5, 3000, 4000, "Russian 5"),
	}

	report := CompareSubtitles(refSubs, "ref.srt", targetSubs, "target.srt")

	if report.PerfectMatches != 0 {
		t.Errorf("expected PerfectMatches == 0, got %d", report.PerfectMatches)
	}
	if len(report.TimestampMismatches) != 1 { // Index 5 has different timestamps
		t.Errorf("expected TimestampMismatches len == 1, got %d", len(report.TimestampMismatches))
	}
	if len(report.MatchedWithOffset) != 2 { // Indices 6 and 7 with offset -2
		t.Errorf("expected MatchedWithOffset len == 2, got %d", len(report.MatchedWithOffset))
	}
	if report.DetectedOffset == nil || *report.DetectedOffset != -2 {
		t.Errorf("expected DetectedOffset == -2, got %v", report.DetectedOffset)
	}
	// Only 2 out of 3 matched (the third has timestamp mismatch)
	if math.Abs(report.MatchPercentage()-66.66666666666667) >= 0.0001 {
		t.Errorf("expected MatchPercentage() ~ 66.666..., got %v", report.MatchPercentage())
	}
	if report.IsPerfect() {
		t.Errorf("expected IsPerfect() == false")
	}
	if !report.HasIssues() {
		t.Errorf("expected HasIssues() == true")
	}
}

func TestCompareExtraInTarget(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
	}

	targetSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "Text 1"),
		makeTestSubtitle(t, 2, 2000, 3000, "Text 2"),
		makeTestSubtitle(t, 3, 3000, 4000, "Text 3"),
	}

	report := CompareSubtitles(refSubs, "ref.srt", targetSubs, "target.srt")

	if report.PerfectMatches != 1 {
		t.Errorf("expected PerfectMatches == 1, got %d", report.PerfectMatches)
	}
	if len(report.ExtraInTarget) != 2 {
		t.Errorf("expected ExtraInTarget len == 2, got %d", len(report.ExtraInTarget))
	}
	if report.MatchPercentage() != 100.0 {
		t.Errorf("expected MatchPercentage() == 100.0, got %v", report.MatchPercentage())
	}
}

func TestDetectOffsetSimple(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 1000, 2000, "Text"),
		makeTestSubtitle(t, 20, 2000, 3000, "Text"),
	}

	targetMap := map[uint32]subtitle.Subtitle{}
	for _, s := range []subtitle.Subtitle{
		makeTestSubtitle(t, 8, 1000, 2000, "Text"),
		makeTestSubtitle(t, 18, 2000, 3000, "Text"),
	} {
		targetMap[s.Index.Value()] = s
	}

	var missing []ComparisonEntry
	for _, s := range refSubs {
		missing = append(missing, ComparisonEntry{
			RefSubtitle: s,
			Status:      ComparisonStatus{Kind: MissingInTarget},
		})
	}

	offset := detectIndexOffset(missing, targetMap)
	if offset == nil || *offset != -2 {
		t.Errorf("expected detectIndexOffset == -2, got %v", offset)
	}
}

func TestDetectOffsetNoMatch(t *testing.T) {
	refSubs := []subtitle.Subtitle{
		makeTestSubtitle(t, 10, 1000, 2000, "Text"),
	}

	targetMap := map[uint32]subtitle.Subtitle{}
	for _, s := range []subtitle.Subtitle{
		makeTestSubtitle(t, 8, 9000, 9999, "Text"),
	} {
		targetMap[s.Index.Value()] = s
	}

	var missing []ComparisonEntry
	for _, s := range refSubs {
		missing = append(missing, ComparisonEntry{
			RefSubtitle: s,
			Status:      ComparisonStatus{Kind: MissingInTarget},
		})
	}

	offset := detectIndexOffset(missing, targetMap)
	if offset != nil {
		t.Errorf("expected detectIndexOffset == nil, got %v", *offset)
	}
}

func TestCompareEmptyFiles(t *testing.T) {
	var refSubs []subtitle.Subtitle
	var targetSubs []subtitle.Subtitle

	report := CompareSubtitles(refSubs, "ref.srt", targetSubs, "target.srt")

	if report.TotalRefCount != 0 {
		t.Errorf("expected TotalRefCount == 0, got %d", report.TotalRefCount)
	}
	if report.TotalTargetCount != 0 {
		t.Errorf("expected TotalTargetCount == 0, got %d", report.TotalTargetCount)
	}
	if report.MatchPercentage() != 0.0 {
		t.Errorf("expected MatchPercentage() == 0.0, got %v", report.MatchPercentage())
	}
}
