// Package importer is a 1-to-1 port of the Rust crate `lib::import`. The
// package is named "importer" because `import` is a reserved word in Go.
package importer

// CsvSubtitleRow represents a parsed CSV row before validation and conversion
// to Subtitle. Was Rust struct CsvSubtitleRow.
type CsvSubtitleRow struct {
	LineNumber int
	StartTime  string
	EndTime    string
	Text       string
}

// AnchoredSubtitleRow represents a parsed anchored format row before
// validation. Was Rust struct AnchoredSubtitleRow.
type AnchoredSubtitleRow struct {
	LineNumber int
	Index      uint32
	Text       string
}
