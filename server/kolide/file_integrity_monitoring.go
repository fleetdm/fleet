package kolide

import "context"

type FIMSections map[string][]string

type FileIntegrityMonitoringStore interface {
	// NewFIMSection creates a named group of file paths
	NewFIMSection(path *FIMSection, opts ...OptionalArg) (*FIMSection, error)
	// FIMSections returns all named file sections
	FIMSections() (FIMSections, error)
	// ClearFIMSections removes all the FIM information
	ClearFIMSections() error
}

// FileIntegrityMonitoringService methods to update
type FileIntegrityMonitoringService interface {
	// GetFIM returns the FIM config
	GetFIM(ctx context.Context) (*FIMConfig, error)
	// ModifyFIM replaces existing FIM.  To disable FIM send FIMConfig with
	// empty FilePaths
	ModifyFIM(ctx context.Context, fim FIMConfig) error
}

// FIMSection maps a name to a group of files for the osquery file_paths
// section.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type FIMSection struct {
	ID          uint
	SectionName string `db:"section_name"`
	Description string
	Paths       []string `db:"-"`
}

// FIMConfig information to set up File Integrity Monitoring
type FIMConfig struct {
	// Interval defines the frequency when the file monitor will run.
	Interval uint `json:"interval"`
	// FilePaths contains named groups of files to monitor. The hash key is the
	// name, the array of strings contains paths to be monitored.
	// See https://osquery.readthedocs.io/en/stable/deployment/file-integrity-monitoring/
	FilePaths FIMSections `json:"file_paths,omitempty"`
	// FileAccesses defines those name groups of FIMSections which will be monitored for file accesses
	FileAccesses []string `json:"file_accesses,omitempty"`
}
