package subtitle

import (
	"testing"
	"time"
)

func ms(n int64) time.Duration { return time.Duration(n) * time.Millisecond }

func makeTestSubtitle(t *testing.T, index uint32, startMs, endMs int64, text string) Subtitle {
	t.Helper()
	idx, err := NewSubtitleIndex(index)
	if err != nil {
		t.Fatalf("NewSubtitleIndex: %v", err)
	}
	start, err := NewSubtitleTimestamp(ms(startMs))
	if err != nil {
		t.Fatalf("NewSubtitleTimestamp(start): %v", err)
	}
	end, err := NewSubtitleTimestamp(ms(endMs))
	if err != nil {
		t.Fatalf("NewSubtitleTimestamp(end): %v", err)
	}
	txt, err := NewSubtitleText(text)
	if err != nil {
		t.Fatalf("NewSubtitleText: %v", err)
	}
	sub, err := NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("NewSubtitle: %v", err)
	}
	return sub
}

func TestSubtitleIndexValidation(t *testing.T) {
	if _, err := NewSubtitleIndex(1); err != nil {
		t.Errorf("expected ok for 1, got %v", err)
	}
	if _, err := NewSubtitleIndex(999); err != nil {
		t.Errorf("expected ok for 999, got %v", err)
	}
	if _, err := NewSubtitleIndex(0); err == nil {
		t.Errorf("expected err for 0")
	}
}

func TestSubtitleTimestampValidation(t *testing.T) {
	if _, err := NewSubtitleTimestamp(0); err != nil {
		t.Errorf("expected ok for zero, got %v", err)
	}
	if _, err := NewSubtitleTimestamp(ms(1000)); err != nil {
		t.Errorf("expected ok for 1000ms, got %v", err)
	}
	if _, err := NewSubtitleTimestamp(ms(-1)); err == nil {
		t.Errorf("expected err for -1ms")
	}
}

func TestSubtitleTextValidation(t *testing.T) {
	if _, err := NewSubtitleText("Hello"); err != nil {
		t.Errorf("expected ok for Hello, got %v", err)
	}
	if _, err := NewSubtitleText("   "); err == nil {
		t.Errorf("expected err for whitespace")
	}
	if _, err := NewSubtitleText(""); err == nil {
		t.Errorf("expected err for empty")
	}
}

func TestSubtitleTextSanitization(t *testing.T) {
	text, err := NewSubtitleText("  Hello  ")
	if err != nil {
		t.Fatalf("NewSubtitleText: %v", err)
	}
	if text.Value() != "Hello" {
		t.Errorf("expected 'Hello', got %q", text.Value())
	}
}

// Test 1: Timestamp parsing
func TestParseTimestamp(t *testing.T) {
	result, err := ParseTimestamp("00:00:01,436")
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	if got := int64(result / time.Millisecond); got != 1436 {
		t.Errorf("expected 1436, got %d", got)
	}

	result, err = ParseTimestamp("01:23:45,678")
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	if got := int64(result / time.Millisecond); got != 5025678 {
		t.Errorf("expected 5025678, got %d", got)
	}
}

// Test 2: Timestamp formatting
func TestFormatTimestamp(t *testing.T) {
	if got := FormatTimestamp(ms(1436)); got != "00:00:01,436" {
		t.Errorf("expected 00:00:01,436, got %q", got)
	}
	if got := FormatTimestamp(ms(5025678)); got != "01:23:45,678" {
		t.Errorf("expected 01:23:45,678, got %q", got)
	}
}

// Test 3: Round-trip timestamp conversion
func TestTimestampRoundtrip(t *testing.T) {
	original := "01:23:45,678"
	d, err := ParseTimestamp(original)
	if err != nil {
		t.Fatalf("ParseTimestamp: %v", err)
	}
	formatted := FormatTimestamp(d)
	if original != formatted {
		t.Errorf("expected %q, got %q", original, formatted)
	}
}

// Test 4: Parse invalid timestamps
func TestParseInvalidTimestamp(t *testing.T) {
	if _, err := ParseTimestamp("invalid"); err == nil {
		t.Errorf("expected err for 'invalid'")
	}
	if _, err := ParseTimestamp("00:60:00,000"); err == nil {
		t.Errorf("expected err for invalid minutes")
	}
	if _, err := ParseTimestamp("00:00:60,000"); err == nil {
		t.Errorf("expected err for invalid seconds")
	}
	if _, err := ParseTimestamp("00:00:00,1000"); err == nil {
		t.Errorf("expected err for invalid ms")
	}
	if _, err := ParseTimestamp("00:00:00.000"); err == nil {
		t.Errorf("expected err for period instead of comma")
	}
}

// Test 5: Subtitle creation with validation
func TestNewSubtitleValid(t *testing.T) {
	idx, _ := NewSubtitleIndex(1)
	start, _ := NewSubtitleTimestamp(ms(1436))
	end, _ := NewSubtitleTimestamp(ms(3481))
	txt, _ := NewSubtitleText("Hello, Harry.")

	subtitle, err := NewSubtitle(idx, start, end, txt)
	if err != nil {
		t.Fatalf("NewSubtitle: %v", err)
	}

	if subtitle.Index.Value() != 1 {
		t.Errorf("expected index 1, got %d", subtitle.Index.Value())
	}
	if subtitle.StartTime.Millis() != 1436 {
		t.Errorf("expected start 1436, got %d", subtitle.StartTime.Millis())
	}
	if subtitle.EndTime.Millis() != 3481 {
		t.Errorf("expected end 3481, got %d", subtitle.EndTime.Millis())
	}
}

// Test 6: Validation - end time must be after start time
func TestNewSubtitleInvalidTimeOrder(t *testing.T) {
	idx, _ := NewSubtitleIndex(1)
	start, _ := NewSubtitleTimestamp(ms(5000))
	end, _ := NewSubtitleTimestamp(ms(1000))
	txt, _ := NewSubtitleText("Text")

	if _, err := NewSubtitle(idx, start, end, txt); err == nil {
		t.Errorf("expected err for end before start")
	}
}

// Test 11: Display formatting
func TestDisplayFormat(t *testing.T) {
	subtitle := makeTestSubtitle(t, 1, 1436, 3481, "<i>Previously on\n\"Resident Alien\"...</i>")

	expected := "1\n00:00:01,436 --> 00:00:03,481\n<i>Previously on\n\"Resident Alien\"...</i>"
	if subtitle.String() != expected {
		t.Errorf("expected %q, got %q", expected, subtitle.String())
	}
}

// Test 13: Duration calculation
func TestSubtitleDuration(t *testing.T) {
	subtitle := makeTestSubtitle(t, 1, 1436, 3481, "Text")
	if got := int64(subtitle.Duration() / time.Millisecond); got != 2045 {
		t.Errorf("expected 2045, got %d", got)
	}
}

// Test 14: HTML tag detection
func TestHasHTMLTags(t *testing.T) {
	withTags := makeTestSubtitle(t, 1, 1000, 2000, "<i>Italic text</i>")
	if !withTags.HasHTMLTags() {
		t.Errorf("expected HTML tags detected")
	}

	withoutTags := makeTestSubtitle(t, 2, 1000, 2000, "Plain text")
	if withoutTags.HasHTMLTags() {
		t.Errorf("expected no HTML tags detected")
	}
}

// Test 15: HTML stripping
func TestTextWithoutHTML(t *testing.T) {
	subtitle := makeTestSubtitle(t, 1, 1000, 2000, "<i>Previously on\n\"Resident Alien\"...</i>")
	if got := subtitle.TextWithoutHTML(); got != "Previously on\n\"Resident Alien\"..." {
		t.Errorf("expected stripped text, got %q", got)
	}
}

// Test 16: Clone and PartialEq
func TestCloneAndEq(t *testing.T) {
	subtitle1 := makeTestSubtitle(t, 1, 1000, 2000, "Text")
	subtitle2 := subtitle1
	if subtitle1 != subtitle2 {
		t.Errorf("expected equal subtitles")
	}
}

// Test 18: Zero-padded formatting
func TestZeroPadding(t *testing.T) {
	if got := FormatTimestamp(ms(5)); got != "00:00:00,005" {
		t.Errorf("expected 00:00:00,005, got %q", got)
	}
	if got := FormatTimestamp(ms(60005)); got != "00:01:00,005" {
		t.Errorf("expected 00:01:00,005, got %q", got)
	}
}

// Test 19: Line count with single line
func TestLineCountSingle(t *testing.T) {
	subtitle := makeTestSubtitle(t, 1, 1000, 2000, "Single line")
	if subtitle.LineCount() != 1 {
		t.Errorf("expected line count 1, got %d", subtitle.LineCount())
	}
}
