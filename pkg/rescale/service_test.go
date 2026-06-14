package rescale

import (
	"math"
	"testing"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// makeTestSubtitle mirrors the verify package test helper.
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

// --- FactorFromFps ---

func TestFactorFromFps(t *testing.T) {
	tests := []struct {
		name    string
		fromFps float64
		toFps   float64
		want    float64
		wantErr bool
	}{
		{name: "identity 25->25", fromFps: 25, toFps: 25, want: 1.0},
		{name: "half 12.5->25", fromFps: 12.5, toFps: 25, want: 0.5},
		{name: "ntsc film 23.976->25", fromFps: 23.976, toFps: 25, want: 0.95904},
		{name: "pal->ntsc 25->23.976", fromFps: 25, toFps: 23.976, want: 1.0427093760427095},
		{name: "to_fps zero", fromFps: 25, toFps: 0, wantErr: true},
		{name: "to_fps negative", fromFps: 25, toFps: -25, wantErr: true},
		{name: "from_fps zero", fromFps: 0, toFps: 25, wantErr: true},
		{name: "from_fps negative", fromFps: -25, toFps: 25, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FactorFromFps(tt.fromFps, tt.toFps)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got factor %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tt.want) >= 1e-9 {
				t.Errorf("FactorFromFps(%v, %v) = %v, want %v", tt.fromFps, tt.toFps, got, tt.want)
			}
		})
	}
}

func TestFactorFromFpsNtscApprox(t *testing.T) {
	got, err := FactorFromFps(23.976, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(got-0.959) >= 0.001 {
		t.Errorf("FactorFromFps(23.976, 25) = %v, want ~0.959", got)
	}
}

// --- ComputeAnchorTransform ---

func TestComputeAnchorTransform(t *testing.T) {
	tests := []struct {
		name                       string
		t1Old, t1New, t2Old, t2New time.Duration
		wantA, wantB               float64
		wantErr                    bool
	}{
		{
			name:  "identity",
			t1Old: 1000 * time.Millisecond, t1New: 1000 * time.Millisecond,
			t2Old: 2000 * time.Millisecond, t2New: 2000 * time.Millisecond,
			wantA: 1.0, wantB: 0.0,
		},
		{
			name:  "pure offset +500ms",
			t1Old: 1000 * time.Millisecond, t1New: 1500 * time.Millisecond,
			t2Old: 5000 * time.Millisecond, t2New: 5500 * time.Millisecond,
			wantA: 1.0, wantB: 500.0,
		},
		{
			name:  "half scale",
			t1Old: 2000 * time.Millisecond, t1New: 1000 * time.Millisecond,
			t2Old: 6000 * time.Millisecond, t2New: 3000 * time.Millisecond,
			wantA: 0.5, wantB: 0.0,
		},
		{
			name:  "scale and offset",
			t1Old: 1000 * time.Millisecond, t1New: 1100 * time.Millisecond,
			t2Old: 11000 * time.Millisecond, t2New: 12100 * time.Millisecond,
			wantA: 1.1, wantB: 0.0,
		},
		{
			name:  "degenerate same source",
			t1Old: 3000 * time.Millisecond, t1New: 1000 * time.Millisecond,
			t2Old: 3000 * time.Millisecond, t2New: 2000 * time.Millisecond,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b, err := ComputeAnchorTransform(tt.t1Old, tt.t1New, tt.t2Old, tt.t2New)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got a=%v b=%v", a, b)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(a-tt.wantA) >= 1e-9 {
				t.Errorf("a = %v, want %v", a, tt.wantA)
			}
			if math.Abs(b-tt.wantB) >= 1e-9 {
				t.Errorf("b = %v, want %v", b, tt.wantB)
			}
		})
	}
}

func TestComputeAnchorTransformMapsAnchorsExactly(t *testing.T) {
	t1Old := 1234 * time.Millisecond
	t1New := 5000 * time.Millisecond
	t2Old := 60000 * time.Millisecond
	t2New := 75000 * time.Millisecond

	a, b, err := ComputeAnchorTransform(t1Old, t1New, t2Old, t2New)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got1 := a*float64(t1Old/time.Millisecond) + b
	got2 := a*float64(t2Old/time.Millisecond) + b

	if math.Abs(got1-float64(t1New/time.Millisecond)) >= 1.0 {
		t.Errorf("anchor 1 maps to %vms, want %vms", got1, t1New/time.Millisecond)
	}
	if math.Abs(got2-float64(t2New/time.Millisecond)) >= 1.0 {
		t.Errorf("anchor 2 maps to %vms, want %vms", got2, t2New/time.Millisecond)
	}
}

// --- Rescale ---

func TestRescaleIdentity(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "One"),
		makeTestSubtitle(t, 2, 3000, 4000, "Two"),
	}

	got, err := Rescale(subs, 1.0, 0.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(subs) {
		t.Fatalf("expected %d subtitles, got %d", len(subs), len(got))
	}
	for i := range subs {
		if got[i] != subs[i] {
			t.Errorf("subtitle %d changed under identity transform: got %+v, want %+v", i, got[i], subs[i])
		}
	}
}

func TestRescaleHalvesTimecodes(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 2000, 4000, "One"),
		makeTestSubtitle(t, 2, 6000, 8000, "Two"),
	}

	got, err := Rescale(subs, 0.5, 0.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantStart := []int64{1000, 3000}
	wantEnd := []int64{2000, 4000}
	for i := range got {
		if got[i].StartTime.Millis() != wantStart[i] {
			t.Errorf("subtitle %d start = %dms, want %dms", i, got[i].StartTime.Millis(), wantStart[i])
		}
		if got[i].EndTime.Millis() != wantEnd[i] {
			t.Errorf("subtitle %d end = %dms, want %dms", i, got[i].EndTime.Millis(), wantEnd[i])
		}
		if got[i].Index.Value() != subs[i].Index.Value() {
			t.Errorf("subtitle %d index changed: got %d, want %d", i, got[i].Index.Value(), subs[i].Index.Value())
		}
		if got[i].Text.Value() != subs[i].Text.Value() {
			t.Errorf("subtitle %d text changed: got %q, want %q", i, got[i].Text.Value(), subs[i].Text.Value())
		}
	}
}

func TestRescaleRoundsToNearestMillisecond(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1001, 2002, "One"),
	}

	// factor fromFps/toFps for 23.976 -> 25
	a, err := FactorFromFps(23.976, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := Rescale(subs, a, 0.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantStart := int64(math.Round(a * 1001))
	wantEnd := int64(math.Round(a * 2002))
	if got[0].StartTime.Millis() != wantStart {
		t.Errorf("start = %dms, want %dms", got[0].StartTime.Millis(), wantStart)
	}
	if got[0].EndTime.Millis() != wantEnd {
		t.Errorf("end = %dms, want %dms", got[0].EndTime.Millis(), wantEnd)
	}
}

func TestRescaleClampsStartToZero(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 1, 1000, 2000, "One"),
	}

	// offset of -1500ms would push start to -500ms; it must clamp to 0.
	got, err := Rescale(subs, 1.0, -1500.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].StartTime.Millis() != 0 {
		t.Errorf("start = %dms, want clamped to 0", got[0].StartTime.Millis())
	}
	if got[0].EndTime.Millis() != 500 {
		t.Errorf("end = %dms, want 500", got[0].EndTime.Millis())
	}
}

func TestRescaleEmptyInput(t *testing.T) {
	got, err := Rescale(nil, 0.5, 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d subtitles", len(got))
	}
}

func TestRescaleInvalidDurationErrors(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 7, 1000, 1001, "One"),
	}

	// Collapsing factor: both start and end round to the same millisecond,
	// so end <= start and the subtitle (index 7) becomes invalid.
	_, err := Rescale(subs, 0.0001, 0.0)
	if err == nil {
		t.Fatalf("expected error for collapsed duration")
	}
	if !contains(err.Error(), "index 7") {
		t.Errorf("expected error to identify index 7, got %q", err.Error())
	}
}

func TestRescaleClampedStartCollapseErrors(t *testing.T) {
	subs := []subtitle.Subtitle{
		makeTestSubtitle(t, 3, 1000, 2000, "One"),
	}

	// Large negative offset clamps start to 0 and pushes end negative -> end <= start.
	_, err := Rescale(subs, 1.0, -5000.0)
	if err == nil {
		t.Fatalf("expected error when rescaled end <= clamped start")
	}
	if !contains(err.Error(), "index 3") {
		t.Errorf("expected error to identify index 3, got %q", err.Error())
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
