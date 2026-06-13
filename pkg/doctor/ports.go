package doctor

// Service is the interface for SRT file diagnosis and repair. It is the port of
// the Rust trait `DoctorService`.
type Service interface {
	// Diagnose diagnoses a subtitle file for issues.
	//
	// filename is the name of the file to diagnose (relative to the base
	// directory).
	//
	// Returns the report of all issues found, or a *DoctorError if the file
	// cannot be read or the path is invalid.
	Diagnose(filename string) (DiagnosticReport, error)

	// Fix fixes issues in a subtitle file.
	//
	// filename is the name of the file to fix (relative to the base directory).
	//
	// Returns the report of fixes applied, or a *DoctorError if the file cannot
	// be read/written or backup fails.
	//
	// Behavior:
	//  1. Creates backup with timestamp: filename.srt.bak-YYYY-MM-DD-HH-MM-SS
	//  2. Removes empty lines inside blocks
	//  3. Normalizes separators to single empty line between blocks
	//  4. Skips unfixable blocks (invalid timestamps, missing fields)
	//  5. Validates result by attempting to parse
	Fix(filename string) (FixReport, error)
}
