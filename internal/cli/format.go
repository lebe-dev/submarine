// Package cli holds the CLI infrastructure shared by command handlers: the
// output-format value-enums, all DTOs, the JSON success/error envelopes, the
// logging setup, and shared helpers. It is a 1-to-1 port of the Rust
// `src/bin/cli` infrastructure (dto.rs, json_output.rs, logging.rs, utils.rs,
// and the value-enums from cli.rs). It imports domain packages but never the
// command-handler package (to avoid an import cycle).
package cli

import "fmt"

// ExportFormat is a port of the Rust `enum ExportFormat` (clap ValueEnum).
type ExportFormat int

const (
	// ExportFormatAnchored -> ExportFormat::Anchored ("[INDEX] TEXT").
	ExportFormatAnchored ExportFormat = iota
)

// String returns the clap value-name for the export format.
func (f ExportFormat) String() string {
	switch f {
	case ExportFormatAnchored:
		return "anchored"
	}
	return ""
}

// ParseExportFormat parses an ExportFormat from its clap value-name.
func ParseExportFormat(s string) (ExportFormat, error) {
	switch s {
	case "anchored":
		return ExportFormatAnchored, nil
	}
	return 0, fmt.Errorf("invalid value '%s' for '--format <FORMAT>'", s)
}

// ImportFormat is a port of the Rust `enum ImportFormat` (clap ValueEnum).
type ImportFormat int

const (
	// ImportFormatCsv -> ImportFormat::Csv ("start_time|end_time|text").
	ImportFormatCsv ImportFormat = iota
	// ImportFormatAnchored -> ImportFormat::Anchored ("[INDEX] TEXT").
	ImportFormatAnchored
)

// String returns the clap value-name for the import format.
func (f ImportFormat) String() string {
	switch f {
	case ImportFormatCsv:
		return "csv"
	case ImportFormatAnchored:
		return "anchored"
	}
	return ""
}

// ParseImportFormat parses an ImportFormat from its clap value-name.
func ParseImportFormat(s string) (ImportFormat, error) {
	switch s {
	case "csv":
		return ImportFormatCsv, nil
	case "anchored":
		return ImportFormatAnchored, nil
	}
	return 0, fmt.Errorf("invalid value '%s' for '--format <FORMAT>'", s)
}

// OutputFormat is a port of the Rust `enum OutputFormat` (clap ValueEnum). The
// default variant is Text (Rust #[default]).
type OutputFormat int

const (
	// OutputFormatText -> OutputFormat::Text (default, human-readable).
	OutputFormatText OutputFormat = iota
	// OutputFormatJson -> OutputFormat::Json (machine-readable).
	OutputFormatJson
)

// String returns the clap value-name for the output format.
func (f OutputFormat) String() string {
	switch f {
	case OutputFormatText:
		return "text"
	case OutputFormatJson:
		return "json"
	}
	return ""
}

// ParseOutputFormat parses an OutputFormat from its clap value-name.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch s {
	case "text":
		return OutputFormatText, nil
	case "json":
		return OutputFormatJson, nil
	}
	return 0, fmt.Errorf("invalid value '%s' for '--output <OUTPUT>'", s)
}
