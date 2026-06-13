package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdirTemp changes the working directory to a fresh temp dir for the duration
// of the test and restores it afterward. The Rust tests rely on a cwd-relative
// "backups" directory; isolating cwd keeps those side effects out of the repo.
func chdirTemp(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// test_create_backup_for_existing_file
func TestCreateBackupForExistingFile(t *testing.T) {
	chdirTemp(t)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.srt")
	if err := os.WriteFile(filePath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	service := NewSubRipBackupService()
	backupPath, err := service.CreateBackup(filePath)

	if err != nil {
		t.Fatalf("expected ok, got error: %v", err)
	}
	if backupPath == nil {
		t.Fatalf("expected backup path to be set")
	}

	backupPathStr := *backupPath
	if !strings.HasPrefix(backupPathStr, "backups/test.srt.") {
		t.Fatalf("expected backup path to start with %q, got %q", "backups/test.srt.", backupPathStr)
	}
	if _, statErr := os.Stat(backupPathStr); statErr != nil {
		t.Fatalf("expected backup file to exist: %v", statErr)
	}

	// Verify backup content matches original
	backupContent, readErr := os.ReadFile(backupPathStr)
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(backupContent) != "test content" {
		t.Fatalf("expected backup content %q, got %q", "test content", string(backupContent))
	}
}

// test_create_backup_for_nonexistent_file
func TestCreateBackupForNonexistentFile(t *testing.T) {
	chdirTemp(t)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nonexistent.srt")

	service := NewSubRipBackupService()
	backupPath, err := service.CreateBackup(filePath)

	if err != nil {
		t.Fatalf("expected ok, got error: %v", err)
	}
	if backupPath != nil {
		t.Fatalf("expected nil backup path, got %q", *backupPath)
	}
}

// test_backup_creates_directory
func TestBackupCreatesDirectory(t *testing.T) {
	chdirTemp(t)

	// Clean up backups directory if it exists
	backupsDir := "backups"
	if _, err := os.Stat(backupsDir); err == nil {
		_ = os.RemoveAll(backupsDir)
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.srt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	service := NewSubRipBackupService()
	_, err := service.CreateBackup(filePath)

	if err != nil {
		t.Fatalf("expected ok, got error: %v", err)
	}
	info, statErr := os.Stat(backupsDir)
	if statErr != nil {
		t.Fatalf("expected backups dir to exist: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected backups to be a directory")
	}
}

// test_backup_filename_format
func TestBackupFilenameFormat(t *testing.T) {
	chdirTemp(t)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "movie.srt")
	if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	service := NewSubRipBackupService()
	backupPathPtr, err := service.CreateBackup(filePath)

	if err != nil {
		t.Fatalf("expected ok, got error: %v", err)
	}
	if backupPathPtr == nil {
		t.Fatalf("expected backup path to be set")
	}
	backupPath := *backupPathPtr

	// Format should be: backups/movie.srt.YYYY-MM-DD_HH-MM-SS.microseconds
	if !strings.HasPrefix(backupPath, "backups/movie.srt.") {
		t.Fatalf("expected backup path to start with %q, got %q", "backups/movie.srt.", backupPath)
	}
	if !strings.Contains(backupPath, "_") { // date-time separator
		t.Fatalf("expected backup path to contain %q", "_")
	}
	if !strings.Contains(backupPath, "-") { // date components separator
		t.Fatalf("expected backup path to contain %q", "-")
	}

	// Extract timestamp part (after "movie.srt.")
	parts := strings.Split(backupPath, "movie.srt.")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	timestamp := parts[1]

	// Should contain date and time parts with microseconds
	// YYYY-MM-DD_HH-MM-SS.mmmmmm = 26 chars minimum
	if len(timestamp) < 26 {
		t.Fatalf("expected timestamp length >= 26, got %d (%q)", len(timestamp), timestamp)
	}
}
