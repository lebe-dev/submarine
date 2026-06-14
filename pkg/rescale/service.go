// Package rescale contains the linear time-correction logic for subtitle
// timecodes. Every timestamp t is mapped to t' = a*t + b, where the scale
// factor a and offset b are derived either from a frame-rate conversion or
// from two anchor points.
package rescale

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// FactorFromFps returns the scale factor for converting timecodes authored at
// fromFps to playback at toFps. The factor is fromFps/toFps: a file authored
// at a higher source rate plays faster (timecodes shrink) when the target rate
// is lower, and vice versa.
//
// Returns an error if either frame rate is non-positive (a frame rate must be
// greater than zero) or if toFps is zero.
func FactorFromFps(fromFps, toFps float64) (float64, error) {
	slog.Debug("computing scale factor from frame rates", "from_fps", fromFps, "to_fps", toFps)

	if fromFps <= 0 {
		return 0, fmt.Errorf("source frame rate must be positive, got %v", fromFps)
	}
	if toFps <= 0 {
		return 0, fmt.Errorf("target frame rate must be positive, got %v", toFps)
	}

	return fromFps / toFps, nil
}

// ComputeAnchorTransform solves the linear transform t' = a*t + b from two
// anchor points: the old/new timestamps of a first reference point
// (t1Old -> t1New) and a second reference point (t2Old -> t2New).
//
// Returns an error if t1Old == t2Old (the two anchor points share the same
// source timestamp), which makes the transform degenerate.
func ComputeAnchorTransform(t1Old, t1New, t2Old, t2New time.Duration) (a, b float64, err error) {
	slog.Debug("computing anchor transform",
		"t1_old", subtitle.FormatTimestamp(t1Old),
		"t1_new", subtitle.FormatTimestamp(t1New),
		"t2_old", subtitle.FormatTimestamp(t2Old),
		"t2_new", subtitle.FormatTimestamp(t2New))

	x1 := float64(t1Old / time.Millisecond)
	y1 := float64(t1New / time.Millisecond)
	x2 := float64(t2Old / time.Millisecond)
	y2 := float64(t2New / time.Millisecond)

	if x1 == x2 {
		return 0, 0, fmt.Errorf("degenerate anchors: both anchor points share the same source timestamp (%s)", subtitle.FormatTimestamp(t1Old))
	}

	a = (y2 - y1) / (x2 - x1)
	b = y1 - a*x1

	return a, b, nil
}

// Rescale applies the linear transform t' = a*t + b to every subtitle's start
// and end timestamp. Timestamps are computed in milliseconds from .Millis(),
// transformed, then rounded to the nearest whole millisecond. The start is
// clamped to be non-negative; the end must remain strictly greater than the
// start, otherwise an error identifying the offending subtitle index is
// returned. Subtitle indices and text are preserved unchanged.
func Rescale(subs []subtitle.Subtitle, a, b float64) ([]subtitle.Subtitle, error) {
	slog.Debug("rescaling subtitles", "count", len(subs), "a", a, "b", b)

	result := make([]subtitle.Subtitle, 0, len(subs))

	for i := range subs {
		sub := subs[i]

		newStartMs := int64(math.Round(a*float64(sub.StartTime.Millis()) + b))
		newEndMs := int64(math.Round(a*float64(sub.EndTime.Millis()) + b))

		if newStartMs < 0 {
			newStartMs = 0
		}

		if newEndMs <= newStartMs {
			return nil, fmt.Errorf(
				"rescaled subtitle with index %d has invalid duration (start: %dms, end: %dms)",
				sub.Index.Value(), newStartMs, newEndMs,
			)
		}

		newStart, err := subtitle.NewSubtitleTimestamp(time.Duration(newStartMs) * time.Millisecond)
		if err != nil {
			return nil, err
		}
		newEnd, err := subtitle.NewSubtitleTimestamp(time.Duration(newEndMs) * time.Millisecond)
		if err != nil {
			return nil, err
		}

		rescaled, err := subtitle.NewSubtitle(sub.Index, newStart, newEnd, sub.Text)
		if err != nil {
			return nil, err
		}

		result = append(result, rescaled)
	}

	return result, nil
}
