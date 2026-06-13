package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// CsvImportService imports subtitles from CSV files. Was Rust struct
// CsvImportService.
type CsvImportService struct{}

// NewCsvImportService creates a new CsvImportService. Was CsvImportService::new.
func NewCsvImportService() *CsvImportService {
	return &CsvImportService{}
}

// validateCsvHeader validates the CSV header record. Was
// CsvImportService::validate_csv_header.
func validateCsvHeader(headers []string, delimiter rune) error {
	actual := strings.Join(headers, string(delimiter))

	if len(headers) != 3 ||
		headers[0] != "start_time" ||
		headers[1] != "end_time" ||
		headers[2] != "text" {
		return subtitle.NewInvalidCsvHeader(string(delimiter), actual)
	}

	slog.Debug("csv header validated successfully")
	return nil
}

// ParseCsvFile parses a CSV file. Was CsvImportService::parse_csv_file.
func (s *CsvImportService) ParseCsvFile(csvPath string, delimiter rune) ([]CsvSubtitleRow, error) {
	slog.Debug(fmt.Sprintf("parsing csv file: %q with delimiter '%c'", csvPath, delimiter))

	file, err := os.Open(csvPath)
	if err != nil {
		return nil, subtitle.NewIoError(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = delimiter
	// The Rust `csv` crate treats a quote as significant only at the start of a
	// field; a bare `"` in the middle of an unquoted field is kept verbatim.
	// Go's encoding/csv rejects bare quotes by default, so enable LazyQuotes to
	// match the Rust parser's tolerant quoting behavior.
	reader.LazyQuotes = true

	headers, err := reader.Read()
	if err != nil {
		return nil, subtitle.NewCsvParseError(1, fmt.Sprintf("failed to read CSV header: %s", err))
	}

	slog.Debug("validating csv header")
	if err := validateCsvHeader(headers, delimiter); err != nil {
		return nil, err
	}

	var rows []CsvSubtitleRow

	idx := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		lineNumber := idx + 2 // +1 for 0-index, +1 for header

		if err != nil {
			return nil, subtitle.NewCsvParseError(lineNumber, fmt.Sprintf("failed to parse CSV record: %s", err))
		}

		if len(record) != 3 {
			return nil, subtitle.NewCsvParseError(lineNumber, fmt.Sprintf("expected 3 columns, got %d", len(record)))
		}

		slog.Debug(fmt.Sprintf("parsed csv row %d: start=%s, end=%s", lineNumber, record[0], record[1]))

		rows = append(rows, CsvSubtitleRow{
			LineNumber: lineNumber,
			StartTime:  strings.TrimSpace(record[0]),
			EndTime:    strings.TrimSpace(record[1]),
			Text:       strings.ReplaceAll(record[2], "\\n", "\n"), // Convert \n to actual newlines
		})

		idx++
	}

	slog.Debug(fmt.Sprintf("parsed %d rows from csv file", len(rows)))
	return rows, nil
}

// ParseAnchoredFile is unsupported on CsvImportService. Was
// CsvImportService::parse_anchored_file.
func (s *CsvImportService) ParseAnchoredFile(anchoredPath string) ([]AnchoredSubtitleRow, error) {
	return nil, subtitle.NewInvalidPath("CsvImportService does not support anchored format parsing")
}

// AnchoredImportService imports subtitles from anchored format files. Was Rust
// struct AnchoredImportService.
type AnchoredImportService struct{}

// NewAnchoredImportService creates a new AnchoredImportService. Was
// AnchoredImportService::new.
func NewAnchoredImportService() *AnchoredImportService {
	return &AnchoredImportService{}
}

// parseAnchoredLine parses an anchored format line to extract index and first
// line text. Returns ok=false when the line is not an [INDEX] marker. Was
// AnchoredImportService::parse_anchored_line.
func parseAnchoredLine(line string) (uint32, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") {
		return 0, "", false
	}

	closeBracket := strings.IndexByte(trimmed, ']')
	if closeBracket < 0 {
		return 0, "", false
	}

	indexStr := trimmed[1:closeBracket]
	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		return 0, "", false
	}

	var text string
	if closeBracket+1 < len(trimmed) {
		text = strings.TrimLeftFunc(trimmed[closeBracket+1:], unicode.IsSpace)
	} else {
		text = ""
	}

	return uint32(index), text, true
}

// ParseCsvFile is unsupported on AnchoredImportService. Was
// AnchoredImportService::parse_csv_file.
func (s *AnchoredImportService) ParseCsvFile(csvPath string, delimiter rune) ([]CsvSubtitleRow, error) {
	return nil, subtitle.NewInvalidPath("AnchoredImportService does not support CSV parsing")
}

// ParseAnchoredFile parses an anchored format file. Was
// AnchoredImportService::parse_anchored_file.
func (s *AnchoredImportService) ParseAnchoredFile(anchoredPath string) ([]AnchoredSubtitleRow, error) {
	slog.Debug(fmt.Sprintf("parsing anchored format file: %q", anchoredPath))

	content, err := os.ReadFile(anchoredPath)
	if err != nil {
		return nil, subtitle.NewIoError(err)
	}

	var rows []AnchoredSubtitleRow
	var currentIndex uint32
	hasCurrentIndex := false
	currentText := ""
	currentLineNumber := 0

	for lineNum, line := range strLines(string(content)) {
		lineNumber := lineNum + 1

		if index, firstLineText, ok := parseAnchoredLine(line); ok {
			if hasCurrentIndex {
				if strings.TrimSpace(currentText) == "" {
					return nil, subtitle.NewAnchoredParseError(currentLineNumber, fmt.Sprintf("Subtitle [%d] has empty text", currentIndex))
				}
				rows = append(rows, AnchoredSubtitleRow{
					LineNumber: currentLineNumber,
					Index:      currentIndex,
					Text:       strings.TrimSpace(currentText),
				})
			}

			currentIndex = index
			hasCurrentIndex = true
			currentText = firstLineText
			currentLineNumber = lineNumber
		} else if hasCurrentIndex {
			if currentText != "" {
				currentText += "\n"
			}
			currentText += line
		} else if strings.TrimSpace(line) != "" {
			return nil, subtitle.NewAnchoredParseError(lineNumber, "Text line found before any [INDEX] marker")
		}
	}

	if hasCurrentIndex {
		if strings.TrimSpace(currentText) == "" {
			return nil, subtitle.NewAnchoredParseError(currentLineNumber, fmt.Sprintf("Subtitle [%d] has empty text", currentIndex))
		}
		rows = append(rows, AnchoredSubtitleRow{
			LineNumber: currentLineNumber,
			Index:      currentIndex,
			Text:       strings.TrimSpace(currentText),
		})
	}

	if len(rows) == 0 {
		return nil, subtitle.NewAnchoredParseError(0, "No valid subtitle entries found in anchored file")
	}

	slog.Debug(fmt.Sprintf("parsed %d entries from anchored file", len(rows)))
	return rows, nil
}

// strLines mirrors Rust's str::lines(): splits on '\n', strips an optional
// trailing '\r' from each line, and yields no trailing empty element for a
// terminating '\n'. An empty string yields no lines.
func strLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}
