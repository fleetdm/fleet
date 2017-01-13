package kolide

// YARAFilePaths represents the files_path section of an osquery config. The
// key maps to file_paths section_name and maps to one or more YARA signature
// group names
type YARAFilePaths map[string][]string

type YARAStore interface {
	// NewYARASignatureGroup creates a new mapping of a name to
	// a group of YARA signatures
	NewYARASignatureGroup(*YARASignatureGroup) (*YARASignatureGroup, error)
	// NewYARAFilePath maps a named set of files to one or more
	// groups of YARA signatures
	NewYARAFilePath(fileSectionName, sigGroupName string) error
	// YARASection creates the osquery configuration YARA section
	YARASection() (*YARASection, error)
}

// YARASignatureGroup maps a name to a group of YARA Signatures
// See https://osquery.readthedocs.io/en/stable/deployment/yara/
type YARASignatureGroup struct {
	ID            uint
	SignatureName string   `db:"signature_name"`
	Paths         []string `db:"-"`
}

// YARASection represents the osquery config for YARA
type YARASection struct {
	Signatures map[string][]string `json:"signatures"`
	FilePaths  map[string][]string `json:"file_paths"`
}
