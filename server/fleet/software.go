package fleet

type SoftwareStore interface {
	SaveHostSoftware(host *Host) error
	LoadHostSoftware(host *Host) error
	AllSoftwareWithoutCPEIterator() (SoftwareIterator, error)
	AddCPEForSoftware(software Software, cpe string) error
}

// Software is a named and versioned piece of software installed on a device.
type Software struct {
	ID uint `json:"id" db:"id"`
	// Name is the reported name.
	Name string `json:"name" db:"name"`
	// Version is reported version.
	Version string `json:"version" db:"version"`
	// Source is the source of the data (osquery table name).
	Source string `json:"source" db:"source"`
}

type CPE struct {
	Cpe23 string `json:"cpe_23" db:"cpe"`
}

// HostSoftware is the set of software installed on a specific host
type HostSoftware struct {
	// Software is the software information.
	Software []Software `json:"software,omitempty"`
	// Modified is a boolean indicating whether this has been modified since
	// loading. If Modified is true, datastore implementations should save the
	// data. We track this here because saving the software set is likely to be
	// an expensive operation.
	Modified bool `json:"-"`
}

type SoftwareIterator interface {
	Next() bool
	Value() (*Software, error)
	Err() error
	Close() error
}

type CPEIterator interface {
	Next() bool
	Value() (*CPE, error)
	Err() error
	Close() error
}
