package subtitle

// Service is the interface for subtitle operations. It is the port of the Rust
// trait `SubtitleService`.
type Service interface {
	// GetByID retrieves a subtitle by its index from a SubRip (.srt) file.
	//
	// Returns (subtitle, nil) when found, (nil, nil) when the file exists and
	// is valid but no subtitle with the given index exists (was Ok(None)), and
	// (nil, *SubtitleError) on error.
	GetByID(filename string, id uint32) (*Subtitle, error)

	// GetAll retrieves all subtitles from a SubRip (.srt) file.
	GetAll(filename string) ([]Subtitle, error)

	// Set updates a subtitle by its index with new values.
	Set(filename string, id uint32, update SubtitleUpdate) (UpdateReport, error)

	// Add adds a new subtitle to the end of an SRT file.
	Add(filename string, start, end SubtitleTimestamp, text SubtitleText) (AddReport, error)

	// WriteAll writes all subtitles to a file, replacing existing content.
	WriteAll(filename string, subtitles []Subtitle) error
}
