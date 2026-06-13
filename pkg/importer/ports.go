package importer

// ImportService is the interface for importing subtitles from various sources.
// It is the port of the Rust trait `ImportService`.
type ImportService interface {
	// ParseCsvFile parses a CSV file and returns validated subtitle data.
	//
	// Arguments:
	//   - csvPath: path to the CSV file
	//   - delimiter: column delimiter character (e.g., '|', ';', ',')
	//
	// Returns a slice of parsed CSV rows with line numbers for error reporting.
	//
	// Errors (all *subtitle.SubtitleError):
	//   - ErrIO: failed to read file
	//   - ErrCSVParse: invalid CSV format or structure
	//   - ErrInvalidCSVHeader: header doesn't match expected format
	ParseCsvFile(csvPath string, delimiter rune) ([]CsvSubtitleRow, error)

	// ParseAnchoredFile parses an anchored format file and returns validated
	// data.
	//
	// Arguments:
	//   - anchoredPath: path to the anchored format file
	//
	// Returns a slice of parsed anchored rows with line numbers for error
	// reporting.
	//
	// Errors (all *subtitle.SubtitleError):
	//   - ErrIO: failed to read file
	//   - ErrAnchoredParse: invalid anchored format
	//   - ErrInvalidAnchoredFormat: malformed index or structure
	ParseAnchoredFile(anchoredPath string) ([]AnchoredSubtitleRow, error)
}
