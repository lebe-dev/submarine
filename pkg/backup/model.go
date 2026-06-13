// Package backup contains the backup domain model and service for the
// submarine subtitle translation toolkit. It is a 1-to-1 port of the Rust
// crate `lib::backup`.
package backup

import "fmt"

// ErrorKind discriminates the variants of BackupError. It mirrors the variants
// of the Rust `enum BackupError`.
type ErrorKind int

const (
	// ErrCreationFailed -> BackupError::CreationFailed(String)
	ErrCreationFailed ErrorKind = iota
	// ErrIO -> BackupError::IoError(std::io::Error)
	ErrIO
)

// BackupError mirrors Rust enum BackupError. Msg holds the formatted message
// (identical text to the Rust #[error(...)] attribute). Carry only the fields a
// variant needs; zero-value the rest.
type BackupError struct {
	Kind    ErrorKind
	Msg     string
	Wrapped error // for IoError source
}

// Error returns the formatted error message (matching the Rust #[error(...)]).
func (e *BackupError) Error() string { return e.Msg }

// Unwrap returns the wrapped source error (IoError).
func (e *BackupError) Unwrap() error { return e.Wrapped }

// NewCreationFailed -> BackupError::CreationFailed(s)
func NewCreationFailed(s string) *BackupError {
	return &BackupError{Kind: ErrCreationFailed, Msg: fmt.Sprintf("failed to create backup: %s", s)}
}

// NewIoError -> BackupError::IoError(err)
func NewIoError(err error) *BackupError {
	return &BackupError{Kind: ErrIO, Msg: fmt.Sprintf("I/O error: %s", err), Wrapped: err}
}
