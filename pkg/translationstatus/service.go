package translationstatus

import (
	"log/slog"
	"sort"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// CheckTranslationStatus checks translation status by comparing reference and
// translation subtitle files.
//
// Arguments:
//   - refSubs: subtitles from the reference file
//   - refFile: reference file name for display
//   - translationSubs: subtitles from the translation file
//   - translationFile: translation file name for display
//   - chunkSize: size of the suggested chunk for next translation
//
// Returns a TranslationStatusReport containing progress information and the next
// chunk suggestion. Was check_translation_status.
func CheckTranslationStatus(
	refSubs []subtitle.Subtitle,
	refFile string,
	translationSubs []subtitle.Subtitle,
	translationFile string,
	chunkSize int,
) TranslationStatusReport {
	totalCount := len(refSubs)

	slog.Debug("comparing reference subtitles with translation subtitles",
		"reference", totalCount, "translation", len(translationSubs))

	slog.Debug("extracting indices from both files")
	refIndices := make(map[uint32]struct{}, len(refSubs))
	for _, s := range refSubs {
		refIndices[s.Index.Value()] = struct{}{}
	}

	translationIndices := make(map[uint32]struct{}, len(translationSubs))
	for _, s := range translationSubs {
		translationIndices[s.Index.Value()] = struct{}{}
	}

	slog.Debug("finding translated indices (intersection)")
	translated := make(map[uint32]struct{})
	for idx := range refIndices {
		if _, ok := translationIndices[idx]; ok {
			translated[idx] = struct{}{}
		}
	}

	translatedCount := len(translated)

	slog.Debug("finding missing indices (in reference but not in translation)")
	missing := make([]uint32, 0)
	for idx := range refIndices {
		if _, ok := translationIndices[idx]; !ok {
			missing = append(missing, idx)
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	missingCount := len(missing)

	slog.Debug("found translated and missing", "translated", translatedCount, "missing", missingCount)

	slog.Debug("calculating next chunk suggestion")
	var nextChunk *ChunkSuggestion
	if len(missing) != 0 {
		startIndex := missing[0]
		endIndex := startIndex
		count := 1

		for _, item := range missing[1:] {
			if count >= chunkSize {
				break
			}

			if item == endIndex+1 {
				endIndex = item
				count++
			} else {
				break
			}
		}

		slog.Debug("next chunk suggestion", "start", startIndex, "end", endIndex)

		nextChunk = &ChunkSuggestion{
			StartIndex: startIndex,
			EndIndex:   endIndex,
		}
	}

	return TranslationStatusReport{
		RefFile:         refFile,
		TranslationFile: translationFile,
		TotalCount:      totalCount,
		TranslatedCount: translatedCount,
		MissingCount:    missingCount,
		NextChunk:       nextChunk,
	}
}
