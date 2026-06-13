package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// subtitleTestDataService returns a SubRipService pointing at the repo's
// test-data directory, mirroring `SubRipService::new(PathBuf::from("test-data"))`
// in the Rust integration test.
func subtitleTestDataService(t *testing.T) *subtitle.SubRipService {
	t.Helper()
	return subtitle.NewSubRipService(filepath.Join(repoRoot(t), "test-data"))
}

// TestIntegrationParseRealSrtFile ports `test_parse_real_srt_file`.
func TestIntegrationParseRealSrtFile(t *testing.T) {
	service := subtitleTestDataService(t)
	filename := "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt"

	subtitle1, err := service.GetByID(filename, 1)
	if err != nil {
		t.Fatalf("Failed to get subtitle 1: %v", err)
	}
	if subtitle1 == nil {
		t.Fatal("Subtitle 1 not found")
	}
	if subtitle1.Index.Value() != 1 {
		t.Errorf("expected index 1, got %d", subtitle1.Index.Value())
	}
	if subtitle1.StartTime.Millis() != 1436 {
		t.Errorf("expected start 1436ms, got %d", subtitle1.StartTime.Millis())
	}
	if subtitle1.EndTime.Millis() != 3481 {
		t.Errorf("expected end 3481ms, got %d", subtitle1.EndTime.Millis())
	}
	if !strings.Contains(subtitle1.Text.Value(), "Previously on") {
		t.Errorf("expected text to contain 'Previously on', got %q", subtitle1.Text.Value())
	}

	subtitle2, err := service.GetByID(filename, 2)
	if err != nil {
		t.Fatalf("Failed to get subtitle 2: %v", err)
	}
	if subtitle2 == nil {
		t.Fatal("Subtitle 2 not found")
	}
	if subtitle2.Index.Value() != 2 {
		t.Errorf("expected index 2, got %d", subtitle2.Index.Value())
	}
	if !strings.Contains(subtitle2.Text.Value(), "Hello, Harry") {
		t.Errorf("expected text to contain 'Hello, Harry', got %q", subtitle2.Text.Value())
	}

	subtitle3, err := service.GetByID(filename, 3)
	if err != nil {
		t.Fatalf("Failed to get subtitle 3: %v", err)
	}
	if subtitle3 == nil {
		t.Fatal("Subtitle 3 not found")
	}
	if subtitle3.Index.Value() != 3 {
		t.Errorf("expected index 3, got %d", subtitle3.Index.Value())
	}
	if subtitle3.Text.Value() != "You are a Grey." {
		t.Errorf("expected 'You are a Grey.', got %q", subtitle3.Text.Value())
	}

	formatted1 := subtitle1.String()
	if !strings.Contains(formatted1, "00:00:01,436 --> 00:00:03,481") {
		t.Errorf("expected formatted subtitle to contain timestamps, got %q", formatted1)
	}
}

// TestIntegrationParseAllSubtitlesFromFile ports `test_parse_all_subtitles_from_file`.
func TestIntegrationParseAllSubtitlesFromFile(t *testing.T) {
	service := subtitleTestDataService(t)
	filename := "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt"

	foundCount := 0
	lastFoundIndex := uint32(0)

	for i := uint32(1); i <= 100; i++ {
		sub, err := service.GetByID(filename, i)
		if err != nil {
			t.Fatalf("Failed to retrieve subtitle %d: %v", i, err)
		}
		if sub == nil {
			continue
		}

		if sub.Index.Value() != i {
			t.Errorf("Index mismatch at subtitle %d", i)
		}
		if sub.Duration().Milliseconds() <= 0 {
			t.Errorf("Non-positive duration at subtitle %d", i)
		}
		if strings.TrimSpace(sub.Text.Value()) == "" {
			t.Errorf("Empty text at subtitle %d", i)
		}

		foundCount++
		lastFoundIndex = i
	}

	_ = lastFoundIndex
	if foundCount <= 10 {
		t.Errorf("Should have found more than 10 subtitles, found %d", foundCount)
	}
}

// TestIntegrationNonexistentSubtitle ports `test_nonexistent_subtitle`.
func TestIntegrationNonexistentSubtitle(t *testing.T) {
	service := subtitleTestDataService(t)
	filename := "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt"

	sub, err := service.GetByID(filename, 99999)
	if err != nil {
		t.Fatalf("expected ok, got error: %v", err)
	}
	if sub != nil {
		t.Error("Should return nil for non-existent subtitle")
	}
}

// TestIntegrationSubtitleFileNotFound ports `test_subtitle_file_not_found`.
func TestIntegrationSubtitleFileNotFound(t *testing.T) {
	service := subtitleTestDataService(t)

	_, err := service.GetByID("nonexistent.srt", 1)
	if err == nil {
		t.Error("Should return error for non-existent file")
	}
}

// TestIntegrationParseWellFormedSrtFile ports `test_parse_well_formed_srt_file`.
func TestIntegrationParseWellFormedSrtFile(t *testing.T) {
	service := subtitleTestDataService(t)
	filename := "Resident.Alien.S03E01.1080p.WEB-DL.RGzsRutracker.eng.srt"

	subtitle1, err := service.GetByID(filename, 1)
	if err != nil {
		t.Fatalf("Failed to get subtitle 1: %v", err)
	}
	if subtitle1 == nil {
		t.Fatal("Subtitle 1 not found")
	}
	if subtitle1.Index.Value() != 1 {
		t.Errorf("expected index 1, got %d", subtitle1.Index.Value())
	}
	if !strings.Contains(subtitle1.Text.Value(), "Previously on") {
		t.Errorf("expected text to contain 'Previously on', got %q", subtitle1.Text.Value())
	}

	subtitle100, err := service.GetByID(filename, 100)
	if err != nil {
		t.Fatalf("Failed to get subtitle 100: %v", err)
	}
	if subtitle100 == nil {
		t.Fatal("Subtitle 100 not found")
	}
	if subtitle100.Index.Value() != 100 {
		t.Errorf("expected index 100, got %d", subtitle100.Index.Value())
	}
}

// TestIntegrationParseRussianSrtWithBlankLines ports
// `test_parse_russian_srt_with_blank_lines`.
func TestIntegrationParseRussianSrtWithBlankLines(t *testing.T) {
	service := subtitleTestDataService(t)
	filename := "russian-with-blank-lines-in-text.rus.srt"

	subtitles, err := service.GetAll(filename)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filename, err)
	}

	if len(subtitles) != 4 {
		t.Fatalf("Expected 4 subtitles in %s, got %d", filename, len(subtitles))
	}

	sub3 := &subtitles[2] // 0-indexed
	if sub3.Index.Value() != 3 {
		t.Errorf("expected index 3, got %d", sub3.Index.Value())
	}
	if sub3.Text.Value() != "Может, он не умер.\n\nЧто?" {
		t.Errorf("Block 3 should contain both lines separated by blank line, got %q", sub3.Text.Value())
	}

	sub1 := &subtitles[0]
	if sub1.Index.Value() != 1 {
		t.Errorf("expected index 1, got %d", sub1.Index.Value())
	}
	if sub1.Text.Value() != "Ранее в сериале..." {
		t.Errorf("unexpected sub1 text: %q", sub1.Text.Value())
	}

	sub2 := &subtitles[1]
	if sub2.Index.Value() != 2 {
		t.Errorf("expected index 2, got %d", sub2.Index.Value())
	}
	if sub2.Text.Value() != "Залезай!" {
		t.Errorf("unexpected sub2 text: %q", sub2.Text.Value())
	}

	sub4 := &subtitles[3]
	if sub4.Index.Value() != 4 {
		t.Errorf("expected index 4, got %d", sub4.Index.Value())
	}
	if sub4.Text.Value() != "Мы должны вернуться и проверить." {
		t.Errorf("unexpected sub4 text: %q", sub4.Text.Value())
	}
}
