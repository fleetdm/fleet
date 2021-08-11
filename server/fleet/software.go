package fleet

import "encoding/json"

type SoftwareStore interface {
	SaveHostSoftware(host *Host) error
	LoadHostSoftware(host *Host) error
	AllSoftwareWithoutCPEIterator() (SoftwareIterator, error)
	AddCPEForSoftware(software Software, cpe string) error
	AllCPEs() ([]string, error)
	InsertCVEForCPE(cve string, cpes []string) error
}

type SoftwareCVE struct {
	CVE         string `json:"cve" db:"cve"`
	DetailsLink string `json:"details_link" db:"details_link"`
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

	// GenerateCPE is the CPE23 string that corresponds to the current software
	GenerateCPE string `json:"generated_cpe" db:"generated_cpe"`
	// Vulnerabilities lists all the found CVEs for the CPE
	Vulnerabilities VulnerabilitiesSlice `json:"vulnerabilities"`
}

type VulnerabilitiesSlice []SoftwareCVE

func (v *VulnerabilitiesSlice) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch typed := src.(type) {
	case []byte:
		err := json.Unmarshal(typed, v)
		if err != nil {
			return err
		}
	}
	return nil
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
