// Package doctor provides SRT file diagnosis and repair. It is a 1-to-1 port of
// the Rust crate `lib::doctor`.
package doctor

import "fmt"

// IssueTypeKind discriminates the variants of IssueType. It mirrors the variants
// of the Rust `enum IssueType`.
type IssueTypeKind int

const (
	// EmptyLineInBlock -> IssueType::EmptyLineInBlock
	EmptyLineInBlock IssueTypeKind = iota
	// MultipleEmptyLines -> IssueType::MultipleEmptyLines { count }
	MultipleEmptyLines
	// InvalidTimestamp -> IssueType::InvalidTimestamp { start, end }
	InvalidTimestamp
	// MissingField -> IssueType::MissingField { field }
	MissingField
	// InsufficientLines -> IssueType::InsufficientLines { found }
	InsufficientLines
	// InvalidIndex -> IssueType::InvalidIndex { value, reason }
	InvalidIndex
	// MalformedTimestamp -> IssueType::MalformedTimestamp { value, reason }
	MalformedTimestamp
	// EmptyText -> IssueType::EmptyText
	EmptyText
)

// IssueType describes a kind of validation issue that can be detected. It is a
// port of the Rust `enum IssueType`; the associated data of each variant is
// carried in the fields below (zero-value the unused ones).
type IssueType struct {
	Kind IssueTypeKind

	// Count -> MultipleEmptyLines { count }
	Count int
	// Start, End -> InvalidTimestamp { start, end }
	Start string
	End   string
	// Field -> MissingField { field }
	Field string
	// Found -> InsufficientLines { found }
	Found int
	// Value, Reason -> InvalidIndex / MalformedTimestamp { value, reason }
	Value  string
	Reason string
}

// String implements the Display impl for IssueType.
func (t IssueType) String() string {
	switch t.Kind {
	case EmptyLineInBlock:
		return "Empty line inside block"
	case MultipleEmptyLines:
		return fmt.Sprintf("Multiple empty lines (%d consecutive)", t.Count)
	case InvalidTimestamp:
		return fmt.Sprintf("Invalid timestamp: end (%s) <= start (%s)", t.End, t.Start)
	case MissingField:
		return fmt.Sprintf("Missing field: %s", t.Field)
	case InsufficientLines:
		return fmt.Sprintf("Insufficient lines (found %d, need at least 3)", t.Found)
	case InvalidIndex:
		return fmt.Sprintf("Invalid index '%s': %s", t.Value, t.Reason)
	case MalformedTimestamp:
		return fmt.Sprintf("Malformed timestamp '%s': %s", t.Value, t.Reason)
	case EmptyText:
		return "Empty or whitespace-only text"
	}
	return ""
}

// Severity is the severity level for issues. It is a port of the Rust
// `enum Severity`. The ordering Warning < Error matches the Rust derive(Ord).
type Severity int

const (
	// Warning: issue doesn't prevent parsing but should be fixed.
	Warning Severity = iota
	// Error: issue prevents correct parsing.
	Error
)

// ValidationIssue is a single validation issue found in the file. It is a port
// of the Rust `struct ValidationIssue`.
type ValidationIssue struct {
	// LineNumber where the issue was found (1-based).
	LineNumber int
	// BlockNumber is the subtitle block number (if applicable, 1-based). A nil
	// pointer mirrors Rust's `None`.
	BlockNumber *int
	// IssueType is the type of issue.
	IssueType IssueType
	// Severity level.
	Severity Severity
	// Context is additional context or description. A nil pointer mirrors
	// Rust's `None`.
	Context *string
}

// String implements the Display impl for ValidationIssue.
func (v ValidationIssue) String() string {
	var severitySymbol string
	switch v.Severity {
	case Error:
		severitySymbol = "❌"
	case Warning:
		severitySymbol = "⚠️"
	}

	if v.BlockNumber != nil {
		return fmt.Sprintf(
			"%s Line %d, Block %d: %s",
			severitySymbol, v.LineNumber, *v.BlockNumber, v.IssueType,
		)
	}
	return fmt.Sprintf(
		"%s Line %d: %s",
		severitySymbol, v.LineNumber, v.IssueType,
	)
}

// DiagnosticReport is the result of a diagnostic operation. It is a port of the
// Rust `struct DiagnosticReport`.
type DiagnosticReport struct {
	// FilePath to the file that was diagnosed.
	FilePath string
	// TotalLines is the total number of lines in the file.
	TotalLines int
	// TotalBlocks is the total number of subtitle blocks detected.
	TotalBlocks int
	// Issues is the list of all issues found.
	Issues []ValidationIssue
	// IsParsable reports whether the file can be parsed successfully.
	IsParsable bool
}

// ErrorCount returns the count of errors (vs warnings).
// Was DiagnosticReport::error_count.
func (r *DiagnosticReport) ErrorCount() int {
	count := 0
	for _, i := range r.Issues {
		if i.Severity == Error {
			count++
		}
	}
	return count
}

// WarningCount returns the count of warnings.
// Was DiagnosticReport::warning_count.
func (r *DiagnosticReport) WarningCount() int {
	count := 0
	for _, i := range r.Issues {
		if i.Severity == Warning {
			count++
		}
	}
	return count
}

// HasIssues reports whether the file has any issues.
// Was DiagnosticReport::has_issues.
func (r *DiagnosticReport) HasIssues() bool {
	return len(r.Issues) != 0
}

// HasErrors reports whether the file has any errors.
// Was DiagnosticReport::has_errors.
func (r *DiagnosticReport) HasErrors() bool {
	return r.ErrorCount() > 0
}

// FixReport is the result of a fix operation. It is a port of the Rust
// `struct FixReport`.
type FixReport struct {
	// OriginalPath to the original file.
	OriginalPath string
	// FixedPath to the fixed file (same as original).
	FixedPath string
	// IssuesFixed is the number of issues fixed.
	IssuesFixed int
	// IssuesUnfixable is the number of issues that couldn't be fixed.
	IssuesUnfixable int
	// UnfixableIssues is the list of unfixable issues.
	UnfixableIssues []ValidationIssue
	// ValidationSuccess reports whether the fixed file can be parsed
	// successfully.
	ValidationSuccess bool
}

// DoctorErrorKind discriminates the variants of DoctorError. It mirrors the
// variants of the Rust `enum DoctorError`.
type DoctorErrorKind int

const (
	// ErrFileNotFound -> DoctorError::FileNotFound
	ErrFileNotFound DoctorErrorKind = iota
	// ErrIO -> DoctorError::IoError
	ErrIO
	// ErrInvalidPath -> DoctorError::InvalidPath
	ErrInvalidPath
	// ErrBackupFailed -> DoctorError::BackupFailed
	ErrBackupFailed
	// ErrValidationFailed -> DoctorError::ValidationFailed
	ErrValidationFailed
)

// DoctorError mirrors Rust enum DoctorError. Msg holds the formatted message
// (identical text to the Rust #[error(...)] attribute).
type DoctorError struct {
	Kind    DoctorErrorKind
	Msg     string
	Wrapped error // for IoError source
}

// Error returns the formatted error message (matching the Rust #[error(...)]).
func (e *DoctorError) Error() string { return e.Msg }

// Unwrap returns the wrapped source error (IoError).
func (e *DoctorError) Unwrap() error { return e.Wrapped }

// NewFileNotFound -> DoctorError::FileNotFound(s)
func NewFileNotFound(s string) *DoctorError {
	return &DoctorError{Kind: ErrFileNotFound, Msg: fmt.Sprintf("file not found: %s", s)}
}

// NewIoError -> DoctorError::IoError(err)
func NewIoError(err error) *DoctorError {
	return &DoctorError{Kind: ErrIO, Msg: fmt.Sprintf("i/o error: %s", err), Wrapped: err}
}

// NewInvalidPath -> DoctorError::InvalidPath(s)
func NewInvalidPath(s string) *DoctorError {
	return &DoctorError{Kind: ErrInvalidPath, Msg: fmt.Sprintf("invalid file path: %s", s)}
}

// NewBackupFailed -> DoctorError::BackupFailed(s)
func NewBackupFailed(s string) *DoctorError {
	return &DoctorError{Kind: ErrBackupFailed, Msg: fmt.Sprintf("backup creation failed: %s", s)}
}

// NewValidationFailed -> DoctorError::ValidationFailed(s)
func NewValidationFailed(s string) *DoctorError {
	return &DoctorError{Kind: ErrValidationFailed, Msg: fmt.Sprintf("failed to validate fixed file: %s", s)}
}
