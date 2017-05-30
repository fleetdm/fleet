package kolide

type FIMSections map[string][]string

type FileIntegrityMonitoringStore interface {
	// NewFIMSection creates a named group of file paths
	NewFIMSection(path *FIMSection, opts ...OptionalArg) (*FIMSection, error)
	// FIMSections returns all named file sections
	FIMSections() (FIMSections, error)
}

// FilePath maps a name to a group of files for the osquery file_paths
// section.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type FIMSection struct {
	ID          uint
	SectionName string `db:"section_name"`
	Description string
	Paths       []string `db:"-"`
}
