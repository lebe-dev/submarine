package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/pkg/backup"
	"github.com/lebe-dev/submarine/pkg/importer"
	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// showPreview shows a preview of subtitles to be imported. Port of the Rust
// `show_preview`.
func showPreview(subtitles []subtitle.Subtitle) {
	fmt.Printf("Preview of %d subtitle(s) to be imported:\n", len(subtitles))
	fmt.Println()

	for i := range subtitles {
		if i >= 5 {
			fmt.Printf("... and %d more subtitle(s)\n", len(subtitles)-5)
			break
		}
		s := &subtitles[i]
		fmt.Printf(
			"  [%4d] %s --> %s\n",
			s.Index.Value(),
			subtitle.FormatTimestamp(s.StartTime.Value()),
			subtitle.FormatTimestamp(s.EndTime.Value()),
		)
		text := s.Text.Value()
		previewText := truncateAt50(text)
		fmt.Printf("       %s\n", strings.ReplaceAll(previewText, "\n", " "))
		fmt.Println()
	}
}

// truncateAt50 mirrors the Rust `text.char_indices().nth(50)` truncation: if the
// text has more than 50 characters, truncate to the first 50 characters and
// append "...", otherwise return the text unchanged.
func truncateAt50(text string) string {
	count := 0
	for byteIdx := range text {
		if count == 50 {
			return text[:byteIdx] + "..."
		}
		count++
	}
	return text
}

// confirmImport asks the user for confirmation to proceed with import. Port of
// the Rust `confirm_import`.
func confirmImport(count int) (bool, error) {
	fmt.Printf("Proceed with importing %d subtitle(s)? (y/N): ", count)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	return strings.EqualFold(strings.TrimSpace(input), "y"), nil
}

// HandleImport is the port of the Rust `cmd::import::handle`.
func HandleImport(
	srtFile string,
	inputFile string,
	importFormat cli.ImportFormat,
	reference *string,
	delimiter string,
	dryRun bool,
	force bool,
	format cli.OutputFormat,
) error {
	slog.Info(fmt.Sprintf("importing subtitles from %s to %s", inputFile, srtFile))

	switch importFormat {
	case cli.ImportFormatAnchored:
		if reference == nil {
			cli.OutputError(
				format,
				"missing_reference",
				"Anchored format requires --reference FILE parameter",
				importStrPtr("Use --reference to specify the SRT file containing timestamps"),
			)
			return errors.New("")
		}
	case cli.ImportFormatCsv:
		// Rust uses `delimiter.len() != 1`, which is the BYTE length of the str.
		if len(delimiter) != 1 {
			cli.OutputError(
				format,
				"invalid_delimiter",
				fmt.Sprintf("Delimiter must be a single character, got: '%s'", delimiter),
				nil,
			)
			return errors.New("")
		}
	}

	var delimiterChar rune
	if importFormat == cli.ImportFormatCsv {
		// Rust `delimiter.chars().next().unwrap()` takes the first char. Since
		// the byte-length check above guaranteed a single byte, this is the
		// first (and only) rune.
		delimiterChar = []rune(delimiter)[0]
	} else {
		delimiterChar = '|'
	}

	srtResolved, err := cli.ResolveExistingPath(srtFile)
	if err != nil {
		return err
	}
	inputResolved, err := cli.ResolveExistingPath(inputFile)
	if err != nil {
		return err
	}

	var subtitles []subtitle.Subtitle
	switch importFormat {
	case cli.ImportFormatCsv:
		subtitles, err = handleCsvImport(inputResolved.FullPath, delimiterChar, format)
		if err != nil {
			return err
		}
	case cli.ImportFormatAnchored:
		referencePath := *reference
		subtitles, err = handleAnchoredImport(inputResolved.FullPath, referencePath, srtResolved.BaseDir, format)
		if err != nil {
			return err
		}
	}

	showPreview(subtitles)

	if dryRun {
		slog.Info("dry-run mode, operations not executed")

		minIdx := minIndex(subtitles)
		maxIdx := maxIndex(subtitles)

		dto := cli.ImportResultDto{
			ImportedCount:  len(subtitles),
			StartIndex:     minIdx,
			EndIndex:       maxIdx,
			TotalSubtitles: 0, // Unknown in dry-run
			BackupPath:     "N/A (dry-run)",
			DryRun:         true,
		}

		cli.OutputSuccess(format, dto, func() {
			fmt.Println("Dry-run mode: no subtitles were imported")
		})
		return nil
	}

	if !force {
		confirmed, cErr := confirmImport(len(subtitles))
		if cErr != nil {
			return cErr
		}
		if !confirmed {
			slog.Info("operation cancelled by user")
			fmt.Println("Cancelled")
			return nil
		}
		slog.Info("user confirmed import operation")
	}

	backupService := backup.NewSubRipBackupService()
	backupResult, backupErr := backupService.CreateBackup(srtResolved.FullPath)

	var backupPath string
	switch {
	case backupErr != nil:
		slog.Error(fmt.Sprintf("failed to create backup: %s", backupErr))
		cli.OutputError(
			format,
			"backup_failed",
			fmt.Sprintf("Failed to create backup: %s", backupErr),
			nil,
		)
		return errors.New("")
	case backupResult != nil:
		slog.Debug(fmt.Sprintf("backup created: %s", *backupResult))
		backupPath = *backupResult
	default:
		slog.Debug("file does not exist, skipping backup")
		backupPath = "N/A (new file)"
	}

	service := subtitle.NewSubRipService(srtResolved.BaseDir)

	var importedCount int
	var startIndex uint32
	var endIndex uint32
	var totalSubtitles int

	switch importFormat {
	case cli.ImportFormatCsv:
		slog.Debug(fmt.Sprintf("adding %d subtitles...", len(subtitles)))
		addedCount := 0
		var startIdx *uint32
		var endIdx *uint32

		for i := range subtitles {
			report, addErr := service.Add(
				srtResolved.Filename,
				subtitles[i].StartTime,
				subtitles[i].EndTime,
				subtitles[i].Text,
			)
			if addErr != nil {
				var subErr *subtitle.SubtitleError
				if errors.As(addErr, &subErr) {
					cliErr := cli.FormatSubtitleError(subErr, srtFile)
					cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
					return errors.New("")
				}
				return addErr
			}
			if startIdx == nil {
				v := report.NewIndex
				startIdx = &v
			}
			ev := report.NewIndex
			endIdx = &ev
			addedCount++
		}

		total := addedCount
		if subs, getErr := service.GetAll(srtResolved.Filename); getErr == nil {
			total = len(subs)
		}

		importedCount = addedCount
		startIndex = derefOrZero(startIdx)
		endIndex = derefOrZero(endIdx)
		totalSubtitles = total

	case cli.ImportFormatAnchored:
		var existingSubs []subtitle.Subtitle
		subs, getErr := service.GetAll(srtResolved.Filename)
		if getErr != nil {
			var subErr *subtitle.SubtitleError
			if errors.As(getErr, &subErr) && subErr.Kind == subtitle.ErrFileNotFound {
				existingSubs = nil
			} else if errors.As(getErr, &subErr) {
				cliErr := cli.FormatSubtitleError(subErr, srtFile)
				cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
				return errors.New("")
			} else {
				return getErr
			}
		} else {
			existingSubs = subs
		}

		mergedMap := make(map[uint32]subtitle.Subtitle)

		for i := range existingSubs {
			mergedMap[existingSubs[i].Index.Value()] = existingSubs[i]
		}

		for i := range subtitles {
			mergedMap[subtitles[i].Index.Value()] = subtitles[i]
		}

		mergedSubs := make([]subtitle.Subtitle, 0, len(mergedMap))
		for _, sub := range mergedMap {
			mergedSubs = append(mergedSubs, sub)
		}
		sort.SliceStable(mergedSubs, func(a, b int) bool {
			return mergedSubs[a].Index.Value() < mergedSubs[b].Index.Value()
		})

		if writeErr := service.WriteAll(srtResolved.Filename, mergedSubs); writeErr != nil {
			var subErr *subtitle.SubtitleError
			if errors.As(writeErr, &subErr) {
				cliErr := cli.FormatSubtitleError(subErr, srtFile)
				cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
				return errors.New("")
			}
			return writeErr
		}

		minIdx := minIndex(subtitles)
		maxIdx := maxIndex(subtitles)

		importedCount = len(subtitles)
		startIndex = minIdx
		endIndex = maxIdx
		totalSubtitles = len(mergedSubs)
	}

	slog.Info(fmt.Sprintf(
		"imported %d subtitles successfully (indices %d-%d)",
		importedCount, startIndex, endIndex,
	))

	dto := cli.ImportResultDto{
		ImportedCount:  importedCount,
		StartIndex:     startIndex,
		EndIndex:       endIndex,
		TotalSubtitles: totalSubtitles,
		BackupPath:     backupPath,
		DryRun:         false,
	}

	cli.OutputSuccess(format, dto, func() {
		fmt.Println("✓ Subtitles imported successfully")
		fmt.Println()
		fmt.Printf("Imported: %d subtitles\n", importedCount)
		fmt.Printf("Index range: %d-%d\n", startIndex, endIndex)
		fmt.Printf("Total subtitles: %d\n", totalSubtitles)
		fmt.Printf("Backup: %s\n", backupPath)
	})

	return nil
}

// handleCsvImport parses a CSV file and converts it to subtitles. Port of the
// Rust `handle_csv_import`.
func handleCsvImport(csvPath string, delimiter rune, format cli.OutputFormat) ([]subtitle.Subtitle, error) {
	slog.Debug("parsing csv file")
	importService := importer.NewCsvImportService()
	csvRows, err := importService.ParseCsvFile(csvPath, delimiter)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, csvPath)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return nil, errors.New("")
		}
		return nil, err
	}
	slog.Info(fmt.Sprintf("parsed %d rows from csv file", len(csvRows)))

	if len(csvRows) == 0 {
		cli.OutputError(format, "empty_csv", "CSV file contains no data rows", nil)
		return nil, errors.New("")
	}

	return convertCsvToSubtitles(csvRows)
}

// convertCsvToSubtitles converts parsed CSV rows into validated subtitles. Port
// of the Rust `convert_csv_to_subtitles`.
func convertCsvToSubtitles(csvRows []importer.CsvSubtitleRow) ([]subtitle.Subtitle, error) {
	var subtitles []subtitle.Subtitle

	for _, csvRow := range csvRows {
		startDuration, err := subtitle.ParseTimestamp(csvRow.StartTime)
		if err != nil {
			return nil, fmt.Errorf("Invalid start timestamp at CSV line %d: %s", csvRow.LineNumber, err)
		}

		endDuration, err := subtitle.ParseTimestamp(csvRow.EndTime)
		if err != nil {
			return nil, fmt.Errorf("Invalid end timestamp at CSV line %d: %s", csvRow.LineNumber, err)
		}

		startTime, err := subtitle.NewSubtitleTimestamp(startDuration)
		if err != nil {
			return nil, fmt.Errorf("Invalid start timestamp at CSV line %d: %s", csvRow.LineNumber, err)
		}

		endTime, err := subtitle.NewSubtitleTimestamp(endDuration)
		if err != nil {
			return nil, fmt.Errorf("Invalid end timestamp at CSV line %d: %s", csvRow.LineNumber, err)
		}

		text, err := subtitle.NewSubtitleText(csvRow.Text)
		if err != nil {
			return nil, fmt.Errorf("Invalid text (empty or whitespace) at CSV line %d: %s", csvRow.LineNumber, err)
		}

		tempIndex, idxErr := subtitle.NewSubtitleIndex(uint32(len(subtitles) + 1))
		if idxErr != nil {
			panic("valid subtitle index")
		}
		sub, err := subtitle.NewSubtitle(tempIndex, startTime, endTime, text)
		if err != nil {
			return nil, fmt.Errorf("Invalid subtitle at CSV line %d: %s", csvRow.LineNumber, err)
		}

		subtitles = append(subtitles, sub)
	}

	return subtitles, nil
}

// handleAnchoredImport parses an anchored format file against a reference SRT
// file. Port of the Rust `handle_anchored_import`.
func handleAnchoredImport(anchoredPath string, referenceFile string, baseDir string, format cli.OutputFormat) ([]subtitle.Subtitle, error) {
	_ = baseDir
	slog.Debug("parsing anchored format file")

	importService := importer.NewAnchoredImportService()
	anchoredRows, err := importService.ParseAnchoredFile(anchoredPath)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, anchoredPath)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return nil, errors.New("")
		}
		return nil, err
	}
	slog.Info(fmt.Sprintf("parsed %d entries from anchored file", len(anchoredRows)))

	slog.Debug(fmt.Sprintf("loading reference file: %s", referenceFile))
	refResolved, err := cli.ResolveExistingPath(referenceFile)
	if err != nil {
		return nil, err
	}
	service := subtitle.NewSubRipService(refResolved.BaseDir)

	referenceSubtitles, err := service.GetAll(refResolved.Filename)
	if err != nil {
		var subErr *subtitle.SubtitleError
		if errors.As(err, &subErr) {
			cliErr := cli.FormatSubtitleError(subErr, referenceFile)
			cli.OutputError(format, cliErr.Code, cliErr.Message, cliErr.Hint)
			return nil, errors.New("")
		}
		return nil, err
	}
	slog.Info(fmt.Sprintf("loaded %d subtitles from reference file", len(referenceSubtitles)))

	refMap := make(map[uint32]*subtitle.Subtitle)
	for i := range referenceSubtitles {
		refMap[referenceSubtitles[i].Index.Value()] = &referenceSubtitles[i]
	}

	var updatedSubtitles []subtitle.Subtitle

	for i := range anchoredRows {
		anchoredRow := &anchoredRows[i]
		index := anchoredRow.Index

		refSubtitle, ok := refMap[index]
		if !ok {
			cli.OutputError(
				format,
				"reference_index_not_found",
				fmt.Sprintf("Reference file does not contain subtitle with index %d", index),
				importStrPtr("Check that the reference file contains all required indices"),
			)
			return nil, errors.New("")
		}

		newText, textErr := subtitle.NewSubtitleText(anchoredRow.Text)
		if textErr != nil {
			return nil, fmt.Errorf("Invalid text (empty or whitespace) at line %d: %s", anchoredRow.LineNumber, textErr)
		}

		sub, subErr := subtitle.NewSubtitle(
			refSubtitle.Index,
			refSubtitle.StartTime,
			refSubtitle.EndTime,
			newText,
		)
		if subErr != nil {
			return nil, fmt.Errorf("Failed to create subtitle for index %d: %s", index, subErr)
		}

		updatedSubtitles = append(updatedSubtitles, sub)
	}

	sort.SliceStable(updatedSubtitles, func(a, b int) bool {
		return updatedSubtitles[a].Index.Value() < updatedSubtitles[b].Index.Value()
	})

	return updatedSubtitles, nil
}

// minIndex returns the minimum subtitle index, or 0 when the slice is empty.
// Mirrors the Rust `.map(|s| *s.index.as_ref()).min().unwrap_or(0)`.
func minIndex(subtitles []subtitle.Subtitle) uint32 {
	if len(subtitles) == 0 {
		return 0
	}
	m := subtitles[0].Index.Value()
	for i := range subtitles {
		v := subtitles[i].Index.Value()
		if v < m {
			m = v
		}
	}
	return m
}

// maxIndex returns the maximum subtitle index, or 0 when the slice is empty.
// Mirrors the Rust `.map(|s| *s.index.as_ref()).max().unwrap_or(0)`.
func maxIndex(subtitles []subtitle.Subtitle) uint32 {
	if len(subtitles) == 0 {
		return 0
	}
	m := subtitles[0].Index.Value()
	for i := range subtitles {
		v := subtitles[i].Index.Value()
		if v > m {
			m = v
		}
	}
	return m
}

// derefOrZero returns *p or 0 when p is nil. Mirrors the Rust
// `Option::unwrap_or(0)`.
func derefOrZero(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}

// importStrPtr returns a pointer to s (helper for optional hints).
func importStrPtr(s string) *string { return &s }
