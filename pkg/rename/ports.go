package rename

// Service is the port for rename operations. It is the port of the Rust trait
// `RenameService`.
type Service interface {
	// FindFiles locates .srt files whose names contain the given mask
	// (case-insensitive), sorted alphabetically.
	FindFiles(mask string) ([]string, error)

	// PrepareRenameOperations renders the new file names for each file using
	// the template and context, detecting collisions. When seriesMode is set,
	// an auto-incrementing `episode` variable (1-based) is injected.
	PrepareRenameOperations(files []string, template string, context *TemplateContext, seriesMode bool) ([]RenameOperation, error)
}

// TemplateContext carries the optional variables used to render new file
// names. Was Rust struct TemplateContext. Nil pointers represent absent
// (Option::None) values.
type TemplateContext struct {
	Name      *string
	Season    *uint32
	Language  *string
	Separator *string
}

// NewTemplateContext creates an empty TemplateContext. Was
// TemplateContext::new / TemplateContext::default.
func NewTemplateContext() TemplateContext {
	return TemplateContext{}
}

// WithName sets the name variable. Was TemplateContext::with_name.
func (c TemplateContext) WithName(name *string) TemplateContext {
	c.Name = name
	return c
}

// WithSeason sets the season variable. Was TemplateContext::with_season.
func (c TemplateContext) WithSeason(season *uint32) TemplateContext {
	c.Season = season
	return c
}

// WithLanguage sets the language variable. Was TemplateContext::with_language.
func (c TemplateContext) WithLanguage(language *string) TemplateContext {
	c.Language = language
	return c
}

// WithSeparator sets the separator variable. Was
// TemplateContext::with_separator.
func (c TemplateContext) WithSeparator(separator *string) TemplateContext {
	c.Separator = separator
	return c
}
