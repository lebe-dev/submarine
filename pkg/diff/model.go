// Package diff contains the subtitle diff logic: comparing two sets of
// subtitles either by text content or by overlapping timecodes.
package diff

import (
	"fmt"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// ByMode discriminates how two subtitle sets are matched against each other.
type ByMode int

const (
	// ByTime matches subtitles whose timecodes overlap within a tolerance.
	ByTime ByMode = iota
	// ByText matches subtitles by their normalized text content.
	ByText
)

// ParseByMode parses a ByMode from its string form ("time" / "text").
func ParseByMode(s string) (ByMode, error) {
	switch s {
	case "time":
		return ByTime, nil
	case "text":
		return ByText, nil
	}
	return ByTime, fmt.Errorf("invalid diff mode '%s': expected 'time' or 'text'", s)
}

// DiffResult is the outcome of a diff between two subtitle sets A and B.
// Common holds the matched A-side subtitles. All lists preserve input order.
type DiffResult struct {
	OnlyInA []subtitle.Subtitle
	OnlyInB []subtitle.Subtitle
	Common  []subtitle.Subtitle
}
