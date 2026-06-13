package backup

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// SubRipBackupService is the SubRip backup service implementation.
// Was Rust struct SubRipBackupService.
type SubRipBackupService struct{}

// NewSubRipBackupService creates a SubRipBackupService.
// Was SubRipBackupService::new.
func NewSubRipBackupService() *SubRipBackupService {
	return &SubRipBackupService{}
}

// CreateBackup creates a backup of the specified file. Returns (nil, nil) when
// the file does not exist (no backup needed), was Rust Ok(None).
func (s *SubRipBackupService) CreateBackup(filePath string) (*string, error) {
	slog.Debug(fmt.Sprintf("creating backup for file: %q", filePath))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("file does not exist, skipping backup")
		return nil, nil
	}

	backupsDir := "backups"
	slog.Debug(fmt.Sprintf("ensuring backups directory exists: %q", backupsDir))
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		return nil, NewCreationFailed(fmt.Sprintf("failed to create backups directory: %s", err))
	}

	filename := filepath.Base(filePath)
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return nil, NewCreationFailed("invalid file path")
	}

	now := time.Now()
	timestamp := fmt.Sprintf(
		"%s.%06d",
		now.Format("2006-01-02_15-04-05"),
		now.Nanosecond()/1000,
	)
	backupFilename := fmt.Sprintf("%s.%s", filename, timestamp)
	slog.Debug(fmt.Sprintf("generated backup filename with timestamp: %s", backupFilename))

	backupPath := filepath.Join(backupsDir, backupFilename)
	slog.Debug(fmt.Sprintf("copying file to backup location: %q", backupPath))

	if err := copyFile(filePath, backupPath); err != nil {
		return nil, NewCreationFailed(err.Error())
	}

	slog.Info(fmt.Sprintf("backup created: %s", backupPath))

	result := backupPath
	return &result, nil
}

// copyFile copies the contents of src to dst. Mirrors Rust std::fs::copy.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}
