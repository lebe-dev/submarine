package cli

import "testing"

// Ported from the #[cfg(test)] mod tests in src/bin/cli/utils.rs.

func TestParseRangeValid(t *testing.T) {
	start, end, err := ParseRange("1-50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 1 {
		t.Errorf("start = %d, want 1", start)
	}
	if end != 50 {
		t.Errorf("end = %d, want 50", end)
	}
}

func TestParseRangeSingle(t *testing.T) {
	start, end, err := ParseRange("42-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 42 {
		t.Errorf("start = %d, want 42", start)
	}
	if end != 42 {
		t.Errorf("end = %d, want 42", end)
	}
}

func TestParseRangeWithSpaces(t *testing.T) {
	start, end, err := ParseRange("10 - 20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != 10 {
		t.Errorf("start = %d, want 10", start)
	}
	if end != 20 {
		t.Errorf("end = %d, want 20", end)
	}
}

func TestParseRangeInvalidFormat(t *testing.T) {
	if _, _, err := ParseRange("invalid"); err == nil {
		t.Errorf("expected error for 'invalid'")
	}
	if _, _, err := ParseRange("1-2-3"); err == nil {
		t.Errorf("expected error for '1-2-3'")
	}
}

func TestParseRangeInvalidNumbers(t *testing.T) {
	if _, _, err := ParseRange("abc-50"); err == nil {
		t.Errorf("expected error for 'abc-50'")
	}
	if _, _, err := ParseRange("1-xyz"); err == nil {
		t.Errorf("expected error for '1-xyz'")
	}
}

func TestParseRangeReversed(t *testing.T) {
	if _, _, err := ParseRange("50-1"); err == nil {
		t.Errorf("expected error for '50-1'")
	}
}

func TestRejectControlCharsValid(t *testing.T) {
	if err := RejectControlChars("hello world", "text"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := RejectControlChars("line1\nline2", "text"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := RejectControlChars("tab\there", "text"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectControlCharsInvalid(t *testing.T) {
	if err := RejectControlChars("hello\x00world", "text"); err == nil {
		t.Errorf("expected error for NUL")
	}
	if err := RejectControlChars("hello\x01world", "text"); err == nil {
		t.Errorf("expected error for 0x01")
	}
	if err := RejectControlChars("hello\x1Fworld", "text"); err == nil {
		t.Errorf("expected error for 0x1F")
	}
}

func TestRejectPercentEncodingValid(t *testing.T) {
	if err := RejectPercentEncoding("simple.srt", "file"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// % not followed by two hex digits
	if err := RejectPercentEncoding("file%name.srt", "file"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectPercentEncodingInvalid(t *testing.T) {
	if err := RejectPercentEncoding("file%20name.srt", "file"); err == nil {
		t.Errorf("expected error for '%%20'")
	}
	if err := RejectPercentEncoding("%2F..%2Fetc", "file"); err == nil {
		t.Errorf("expected error for '%%2F'")
	}
}
