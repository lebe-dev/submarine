package rename

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// FileRenameService is the filesystem-backed implementation of Service. Was
// Rust struct FileRenameService.
type FileRenameService struct {
	baseDir string
}

// NewFileRenameService creates a FileRenameService with the given base
// directory. Was FileRenameService::new.
func NewFileRenameService(baseDir string) *FileRenameService {
	return &FileRenameService{baseDir: baseDir}
}

// FindFiles locates .srt files in the base directory whose names contain the
// mask (case-insensitive), returning them sorted alphabetically. Was
// FileRenameService::find_files.
func (s *FileRenameService) FindFiles(mask string) ([]string, error) {
	slog.Info(fmt.Sprintf("searching for files with mask: %s", mask))

	maskLower := strings.ToLower(mask)
	foundFiles := make([]string, 0)

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, NewIoError(fmt.Errorf("failed to read directory: %w", err))
	}

	for _, entry := range entries {
		path := filepath.Join(s.baseDir, entry.Name())

		info, statErr := os.Stat(path)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}

		ext := filepath.Ext(path)
		if ext == "" {
			continue
		}
		// Rust compares the extension (without the leading dot) to exactly
		// "srt" (case-sensitive).
		if strings.TrimPrefix(ext, ".") != "srt" {
			continue
		}

		filename := filepath.Base(path)
		if strings.Contains(strings.ToLower(filename), maskLower) {
			slog.Debug(fmt.Sprintf("found file: %q", path))
			foundFiles = append(foundFiles, path)
		}
	}

	if len(foundFiles) == 0 {
		return nil, NewNoFilesFound(mask)
	}

	// Sort alphabetically (matches Rust PathBuf sort over full paths).
	sort.Strings(foundFiles)

	slog.Info(fmt.Sprintf("found %d files", len(foundFiles)))
	return foundFiles, nil
}

// PrepareRenameOperations renders the new file name for each input file using
// the template and context, detecting collisions. When seriesMode is set, an
// auto-incrementing 1-based `episode` variable is injected. Was
// FileRenameService::prepare_rename_operations.
func (s *FileRenameService) PrepareRenameOperations(
	files []string,
	tmplStr string,
	context *TemplateContext,
	seriesMode bool,
) ([]RenameOperation, error) {
	slog.Info("preparing rename operations")
	slog.Debug(fmt.Sprintf("series_mode: %t", seriesMode))
	slog.Debug(fmt.Sprintf("template: %s", tmplStr))

	// Replace tera with text/template. missingkey=error makes referencing an
	// absent variable a render error, mirroring tera's behavior.
	tmpl, err := template.New("filename").Option("missingkey=error").Parse(tmplStr)
	if err != nil {
		return nil, NewInvalidTemplate(err.Error())
	}

	operations := make([]RenameOperation, 0, len(files))

	for index, filePath := range files {
		ctx := make(map[string]any)

		if context.Name != nil {
			ctx["name"] = *context.Name
		}

		if context.Season != nil {
			ctx["season"] = fmt.Sprintf("%02d", *context.Season)
		}

		if context.Language != nil {
			ctx["language"] = *context.Language
		}

		if context.Separator != nil {
			ctx["separator"] = *context.Separator
		}

		if seriesMode {
			episodeNumber := index + 1
			ctx["episode"] = fmt.Sprintf("%02d", episodeNumber)
			slog.Debug(fmt.Sprintf("series mode: file %d -> episode %d", index, episodeNumber))
		}

		var sb strings.Builder
		if execErr := tmpl.Execute(&sb, ctx); execErr != nil {
			return nil, NewTemplateError(execErr.Error())
		}
		newName := sb.String()

		slog.Debug(fmt.Sprintf("new name for %q: %s", filePath, newName))

		newPath := filepath.Join(s.baseDir, newName)
		_, statErr := os.Stat(newPath)
		exists := statErr == nil
		collision := exists && newPath != filePath

		if collision {
			slog.Debug(fmt.Sprintf("collision detected for: %s", newName))
		}

		operations = append(operations, NewRenameOperation(filePath, newName, collision))
	}

	slog.Info(fmt.Sprintf("prepared %d rename operations", len(operations)))
	return operations, nil
}
