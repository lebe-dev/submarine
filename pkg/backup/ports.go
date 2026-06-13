package backup

// Service is the interface for backup operations. It is the port of the Rust
// trait `BackupService`.
type Service interface {
	// CreateBackup creates a backup of the specified file.
	//
	// Arguments:
	//   - filePath: path to the file to backup
	//
	// Returns:
	//   - (nil, nil): file doesn't exist (new file, no backup needed) (was Ok(None))
	//   - (path, nil): backup created successfully, returns path to backup
	//   - (nil, err): failed to create backup
	CreateBackup(filePath string) (*string, error)
}
