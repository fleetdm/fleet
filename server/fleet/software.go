package fleet

import (
	"fmt"
	"time"
)

type CVE struct {
	CVE         string `json:"cve" db:"cve"`
	DetailsLink string `json:"details_link" db:"-"`
	// These are double pointers so that we can omit them AND return nulls when needed.
	// 1. omitted when using the free tier
	// 2. null when using the premium tier, but there is no value available. This may be due to an issue with syncing cve scores.
	// 3. non-null when using the premium tier, and value is available.
	CVSSScore        **float64 `json:"cvss_score,omitempty" db:"cvss_score"`
	EPSSProbability  **float64 `json:"epss_probability,omitempty" db:"epss_probability"`
	CISAKnownExploit **bool    `json:"cisa_known_exploit,omitempty" db:"cisa_known_exploit"`
}

type CVEMeta struct {
	CVE string `db:"cve"`
	// CVSSScore is the Common Vulnerability Scoring System (CVSS) base score v3. The base score ranges from 0 - 10 and
	// takes into account several different metrics.
	// See https://nvd.nist.gov/vuln-metrics/cvss.
	CVSSScore *float64 `db:"cvss_score"`
	// EPSSProbability is the Exploit Prediction Scoring System (EPSS) score. It is the probability
	// that a software vulnerability will be exploited in the next 30 days.
	// See https://www.first.org/epss/.
	EPSSProbability *float64 `db:"epss_probability"`
	// CISAKnownExploit is whether the the software vulnerability is a known exploit according to CISA.
	// See https://www.cisa.gov/known-exploited-vulnerabilities.
	CISAKnownExploit *bool `db:"cisa_known_exploit"`
	// Published is when the cve was published according to NIST.score
	Published *time.Time `db:"published"`
}

// Must be kept in sync with the vendor column definition.
const SoftwareVendorMaxLength = 114
const SoftwareVendorMaxLengthFmt = "%.111s..."

// Software is a named and versioned piece of software installed on a device.
type Software struct {
	ID uint `json:"id" db:"id"`
	// Name is the reported name.
	Name string `json:"name" db:"name"`
	// Version is reported version.
	Version string `json:"version" db:"version"`
	// BundleIdentifier is the CFBundleIdentifier label from the info properties
	BundleIdentifier string `json:"bundle_identifier,omitempty" db:"bundle_identifier"`
	// Source is the source of the data (osquery table name).
	Source string `json:"source" db:"source"`

	// Release is the version of the OS this software was released on
	// (e.g. "30.el7" for a CentOS package).
	Release string `json:"release,omitempty" db:"release"`
	// Vendor is the supplier of the software (e.g. "CentOS").
	Vendor string `json:"vendor,omitempty" db:"vendor"`

	// TODO: Remove this as part of the clean up of https://github.com/fleetdm/fleet/pull/7297
	// DO NOT USE THIS, use 'Vendor' instead. We had to 'recreate' the vendor column because we
	// needed to make it wider - the old column was left and renamed to 'vendor_old'
	VendorOld string `json:"-" db:"vendor_old"`

	// Arch is the architecture of the software (e.g. "x86_64").
	Arch string `json:"arch,omitempty" db:"arch"`

	// GenerateCPE is the CPE23 string that corresponds to the current software
	GenerateCPE string `json:"generated_cpe" db:"generated_cpe"`

	// Vulnerabilities lists all the found CVEs for the CPE
	Vulnerabilities Vulnerabilities `json:"vulnerabilities"`
	// HostsCount indicates the number of hosts with that software, filled only
	// if explicitly requested.
	HostsCount int `json:"hosts_count,omitempty" db:"hosts_count"`
	// CountsUpdatedAt is the timestamp when the hosts count was last updated
	// for that software, filled only if hosts count is requested.
	CountsUpdatedAt time.Time `json:"-" db:"counts_updated_at"`
	// LastOpenedAt is the timestamp when that software was last opened on the
	// corresponding host. Only filled when the software list is requested for
	// a specific host (host_id is provided).
	LastOpenedAt *time.Time `json:"last_opened_at,omitempty" db:"last_opened_at"`
}

func (Software) AuthzType() string {
	return "software"
}

// AuthzSoftwareInventory is used for access controls on software inventory.
type AuthzSoftwareInventory struct {
	// TeamID is the ID of the team. A value of nil means global scope.
	TeamID *uint `json:"team_id"`
}

// AuthzType implements authz.AuthzTyper.
func (s *AuthzSoftwareInventory) AuthzType() string {
	return "software_inventory"
}

type Vulnerabilities []CVE

// HostSoftware is the set of software installed on a specific host
type HostSoftware struct {
	// Software is the software information.
	Software []Software `json:"software,omitempty" csv:"-"`
}

type SoftwareIterator interface {
	Next() bool
	Value() (*Software, error)
	Err() error
	Close() error
}

type SoftwareListOptions struct {
	ListOptions

	// HostID filters software to the specified host if not nil.
	HostID           *uint
	TeamID           *uint `query:"team_id,optional"`
	VulnerableOnly   bool  `query:"vulnerable,optional"`
	IncludeCVEScores bool

	// WithHostCounts indicates that the list of software should include the
	// counts of hosts per software, and include only those software that have
	// a count of hosts > 0.
	WithHostCounts bool
}

// SoftwareCPE represents an entry in the `software_cpe` table
type SoftwareCPE struct {
	ID         uint   `db:"id"`
	SoftwareID uint   `db:"software_id"`
	CPE        string `db:"cpe"`
}

// SoftwareVulnerability identifies a vulnerability on a specific software.
type SoftwareVulnerability struct {
	SoftwareID uint   `db:"software_id"`
	CVE        string `db:"cve"`
}

// String implements fmt.Stringer.
func (sv SoftwareVulnerability) String() string {
	return fmt.Sprintf("{%d,%s}", sv.SoftwareID, sv.CVE)
}

// Key returns a string representation of the SoftwareVulnerability
func (sv *SoftwareVulnerability) Key() string {
	return fmt.Sprintf("%d:%s", sv.SoftwareID, sv.CVE)
}

type VulnerabilitySource int

const (
	NVDSource VulnerabilitySource = iota
	UbuntuOVALSource
	RHELOVALSource
)
