package rename

import (
	"os"
	"path/filepath"
	"testing"
)

// createFile creates an empty file at dir/name, failing the test on error.
func createFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file %s: %v", name, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file %s: %v", name, err)
	}
	return path
}

func strPtr(s string) *string { return &s }

func u32Ptr(v uint32) *uint32 { return &v }

func TestFindFilesCaseInsensitive(t *testing.T) {
	basePath := t.TempDir()

	createFile(t, basePath, "Resident.Alien.S01E01.srt")
	createFile(t, basePath, "resident.alien.s01e02.srt")
	createFile(t, basePath, "Other.Show.S01E01.srt")

	service := NewFileRenameService(basePath)
	files, err := service.FindFiles("resident")
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestFindFilesOnlySrt(t *testing.T) {
	basePath := t.TempDir()

	createFile(t, basePath, "test.srt")
	createFile(t, basePath, "test.txt")
	createFile(t, basePath, "test.mp4")

	service := NewFileRenameService(basePath)
	files, err := service.FindFiles("test")
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if !hasSuffix(files[0], "test.srt") {
		t.Fatalf("expected file to end with test.srt, got %q", files[0])
	}
}

func TestFindFilesSorted(t *testing.T) {
	basePath := t.TempDir()

	createFile(t, basePath, "c.srt")
	createFile(t, basePath, "a.srt")
	createFile(t, basePath, "b.srt")

	service := NewFileRenameService(basePath)
	files, err := service.FindFiles(".srt")
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if filepath.Base(files[0]) != "a.srt" {
		t.Fatalf("expected files[0] == a.srt, got %q", filepath.Base(files[0]))
	}
	if filepath.Base(files[1]) != "b.srt" {
		t.Fatalf("expected files[1] == b.srt, got %q", filepath.Base(files[1]))
	}
	if filepath.Base(files[2]) != "c.srt" {
		t.Fatalf("expected files[2] == c.srt, got %q", filepath.Base(files[2]))
	}
}

func TestPrepareRenameOperationsSeriesMode(t *testing.T) {
	basePath := t.TempDir()

	file1 := createFile(t, basePath, "old1.srt")
	file2 := createFile(t, basePath, "old2.srt")

	service := NewFileRenameService(basePath)
	files := []string{file1, file2}

	context := NewTemplateContext().
		WithName(strPtr("Test Show")).
		WithSeason(u32Ptr(1)).
		WithSeparator(strPtr("."))

	// tera "{{ name }}.S{{ season }}E{{ episode }}.srt" -> Go template syntax.
	tmpl := "{{.name}}.S{{.season}}E{{.episode}}.srt"
	operations, err := service.PrepareRenameOperations(files, tmpl, &context, true)
	if err != nil {
		t.Fatalf("PrepareRenameOperations: %v", err)
	}

	if len(operations) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(operations))
	}
	if operations[0].NewName != "Test Show.S01E01.srt" {
		t.Fatalf("operations[0].NewName = %q, want %q", operations[0].NewName, "Test Show.S01E01.srt")
	}
	if operations[1].NewName != "Test Show.S01E02.srt" {
		t.Fatalf("operations[1].NewName = %q, want %q", operations[1].NewName, "Test Show.S01E02.srt")
	}
}

func TestPrepareRenameOperationsCollisionDetection(t *testing.T) {
	basePath := t.TempDir()

	file1 := createFile(t, basePath, "old.srt")
	createFile(t, basePath, "new.srt")

	service := NewFileRenameService(basePath)
	files := []string{file1}

	context := NewTemplateContext()
	tmpl := "new.srt"

	operations, err := service.PrepareRenameOperations(files, tmpl, &context, false)
	if err != nil {
		t.Fatalf("PrepareRenameOperations: %v", err)
	}

	if len(operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(operations))
	}
	if !operations[0].Collision {
		t.Fatalf("expected collision to be true")
	}
}

func TestTemplateWithOptionalVariables(t *testing.T) {
	basePath := t.TempDir()

	file1 := createFile(t, basePath, "old.srt")

	service := NewFileRenameService(basePath)
	files := []string{file1}

	context := NewTemplateContext().
		WithName(strPtr("Show")).
		WithSeparator(strPtr("."))

	// tera "{{ name }}{{ separator }}subtitle.srt" -> Go template syntax.
	tmpl := "{{.name}}{{.separator}}subtitle.srt"
	operations, err := service.PrepareRenameOperations(files, tmpl, &context, false)
	if err != nil {
		t.Fatalf("PrepareRenameOperations: %v", err)
	}

	if len(operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(operations))
	}
	if operations[0].NewName != "Show.subtitle.srt" {
		t.Fatalf("operations[0].NewName = %q, want %q", operations[0].NewName, "Show.subtitle.srt")
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
