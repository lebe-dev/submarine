// Package rename contains the domain models and services for the submarine
// mass-rename feature. It is a 1-to-1 port of the Rust crate `lib::rename`.
package rename

import (
	"fmt"
	"path/filepath"
)

// RenameOperation describes a single planned rename. Was Rust struct
// RenameOperation.
type RenameOperation struct {
	OriginalPath string
	NewName      string
	Collision    bool
}

// NewRenameOperation creates a RenameOperation. Was RenameOperation::new.
func NewRenameOperation(originalPath string, newName string, collision bool) RenameOperation {
	return RenameOperation{
		OriginalPath: originalPath,
		NewName:      newName,
		Collision:    collision,
	}
}

// OriginalFilename returns the file name component of the original path, or
// empty string with ok=false when it cannot be determined. Was
// RenameOperation::original_filename returning Option<String>.
func (o RenameOperation) OriginalFilename() (string, bool) {
	if o.OriginalPath == "" {
		return "", false
	}
	name := filepath.Base(o.OriginalPath)
	if name == "." || name == string(filepath.Separator) {
		return "", false
	}
	return name, true
}

// RenameReport accumulates counters for a mass-rename run. Was Rust struct
// RenameReport.
type RenameReport struct {
	TotalFiles int
	Renamed    int
	Skipped    int
	Failed     int
}

// NewRenameReport creates a zeroed RenameReport. Was RenameReport::new /
// RenameReport::default.
func NewRenameReport() RenameReport {
	return RenameReport{
		TotalFiles: 0,
		Renamed:    0,
		Skipped:    0,
		Failed:     0,
	}
}

// IncrementRenamed increments the renamed counter. Was
// RenameReport::increment_renamed.
func (r *RenameReport) IncrementRenamed() {
	r.Renamed++
}

// IncrementSkipped increments the skipped counter. Was
// RenameReport::increment_skipped.
func (r *RenameReport) IncrementSkipped() {
	r.Skipped++
}

// IncrementFailed increments the failed counter. Was
// RenameReport::increment_failed.
func (r *RenameReport) IncrementFailed() {
	r.Failed++
}

// RenameErrorKind discriminates the variants of RenameError. It mirrors the
// variants of the Rust `enum RenameError`.
type RenameErrorKind int

const (
	// ErrNoFilesFound -> RenameError::NoFilesFound
	ErrNoFilesFound RenameErrorKind = iota
	// ErrTemplateError -> RenameError::TemplateError
	ErrTemplateError
	// ErrIoError -> RenameError::IoError
	ErrIoError
	// ErrInvalidTemplate -> RenameError::InvalidTemplate
	ErrInvalidTemplate
	// ErrGlobError -> RenameError::GlobError
	ErrGlobError
)

// RenameError mirrors Rust enum RenameError. Msg holds the formatted message
// (identical text to the Rust #[error(...)] attribute). Wrapped carries the
// underlying I/O error for the IoError variant.
type RenameError struct {
	Kind    RenameErrorKind
	Msg     string
	Wrapped error // for IoError source
}

// Error returns the formatted error message (matching the Rust #[error(...)]).
func (e *RenameError) Error() string { return e.Msg }

// Unwrap returns the wrapped source error (IoError).
func (e *RenameError) Unwrap() error { return e.Wrapped }

// NewNoFilesFound -> RenameError::NoFilesFound(pattern)
func NewNoFilesFound(pattern string) *RenameError {
	return &RenameError{Kind: ErrNoFilesFound, Msg: fmt.Sprintf("no files found matching pattern: %s", pattern)}
}

// NewTemplateError -> RenameError::TemplateError(msg)
func NewTemplateError(msg string) *RenameError {
	return &RenameError{Kind: ErrTemplateError, Msg: fmt.Sprintf("template rendering failed: %s", msg)}
}

// NewIoError -> RenameError::IoError(err) (#[from] std::io::Error)
func NewIoError(err error) *RenameError {
	return &RenameError{Kind: ErrIoError, Msg: fmt.Sprintf("file operation failed: %s", err), Wrapped: err}
}

// NewInvalidTemplate -> RenameError::InvalidTemplate(msg)
func NewInvalidTemplate(msg string) *RenameError {
	return &RenameError{Kind: ErrInvalidTemplate, Msg: fmt.Sprintf("invalid template: %s", msg)}
}

// NewGlobError -> RenameError::GlobError(msg)
func NewGlobError(msg string) *RenameError {
	return &RenameError{Kind: ErrGlobError, Msg: fmt.Sprintf("glob pattern error: %s", msg)}
}
