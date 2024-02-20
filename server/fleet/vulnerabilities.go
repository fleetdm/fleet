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
	CVSSScore         **float64   `json:"cvss_score,omitempty" db:"cvss_score"`
	EPSSProbability   **float64   `json:"epss_probability,omitempty" db:"epss_probability"`
	CISAKnownExploit  **bool      `json:"cisa_known_exploit,omitempty" db:"cisa_known_exploit"`
	CVEPublished      **time.Time `json:"cve_published,omitempty" db:"cve_published"`
	Description       **string    `json:"cve_description,omitempty" db:"description"`
	ResolvedInVersion **string    `json:"resolved_in_version,omitempty" db:"resolved_in_version"`
}

type CVEMeta struct {
	CVE string `db:"cve" json:"cve"`
	// CVSSScore is the Common Vulnerability Scoring System (CVSS) base score v3. The base score ranges from 0 - 10 and
	// takes into account several different metrics.
	// See https://nvd.nist.gov/vuln-metrics/cvss.
	CVSSScore *float64 `db:"cvss_score" json:"cvss_score,omitempty"`
	// EPSSProbability is the Exploit Prediction Scoring System (EPSS) score. It is the probability
	// that a software vulnerability will be exploited in the next 30 days.
	// See https://www.first.org/epss/.
	EPSSProbability *float64 `db:"epss_probability" json:"epss_probability,omitempty"`
	// CISAKnownExploit is whether the the software vulnerability is a known exploit according to CISA.
	// See https://www.cisa.gov/known-exploited-vulnerabilities.
	CISAKnownExploit *bool `db:"cisa_known_exploit" json:"cisa_known_exploit,omitempty"`
	// Published is when the cve was published according to NIST.score
	Published *time.Time `db:"published" json:"published,omitempty"`
	// CVE text description
	Description string `db:"description" json:"description,omitempty"`
}

// SoftwareCPE represents an entry in the `software_cpe` table.
type SoftwareCPE struct {
	ID         uint   `db:"id"`
	SoftwareID uint   `db:"software_id"`
	CPE        string `db:"cpe"`
}

// SoftwareVulnerability is a vulnerability on a software.
// Represents an entry in the `software_cve` table.
type SoftwareVulnerability struct {
	SoftwareID        uint    `db:"software_id"`
	CVE               string  `db:"cve"`
	ResolvedInVersion *string `db:"resolved_in_version"`
}

// String implements fmt.Stringer.
func (sv SoftwareVulnerability) String() string {
	return fmt.Sprintf("{%d,%s}", sv.SoftwareID, sv.CVE)
}

// Key returns a string representation of the software vulnerability.
// If we have a list of software vulnerabilities, the Key can be used
// as a discrimator for unique entries.
func (sv SoftwareVulnerability) Key() string {
	return fmt.Sprintf("software:%d:%s", sv.SoftwareID, sv.CVE)
}

func (sv SoftwareVulnerability) GetCVE() string {
	return sv.CVE
}

func (sv SoftwareVulnerability) Affected() uint {
	return sv.SoftwareID
}

// OSVulnerability is a vulnerability on a OS.
// Represents an entry in the `os_vulnerabilities` table.
type OSVulnerability struct {
	OSID uint   `db:"operating_system_id"`
	CVE  string `db:"cve"`
	// Source is the source of the vulnerability.
	Source VulnerabilitySource `db:"source"`
	// ResolvedInVersion is the version of the OS that resolves the vulnerability.
	ResolvedInVersion *string `db:"resolved_in_version"`
}

// String implements fmt.Stringer.
func (ov OSVulnerability) String() string {
	return fmt.Sprintf("{%d,%s}", ov.OSID, ov.CVE)
}

// Key returns a string representation of the os vulnerability.
// If we have a list of os vulnerabilities, the Key can be used
// as a discrimator for unique entries.
func (ov OSVulnerability) Key() string {
	var rv string
	if ov.ResolvedInVersion != nil {
		rv = *ov.ResolvedInVersion
	}
	return fmt.Sprintf("os:%d:%s:%s", ov.OSID, ov.CVE, rv)
}

func (ov OSVulnerability) GetCVE() string {
	return ov.CVE
}

func (ov OSVulnerability) Affected() uint {
	return ov.OSID
}

// Represents a vulnerability, e.g. an OS or a Software vulnerability.
type Vulnerability interface {
	OSVulnerability | SoftwareVulnerability
	GetCVE() string
	Affected() uint
	Key() string
}

type VulnerabilitySource int

const (
	NVDSource VulnerabilitySource = iota
	UbuntuOVALSource
	RHELOVALSource
	MSRCSource
	MacOfficeReleaseNotesSource
)

type VulnerabilityWithMetadata struct {
	CVEMeta
	HostsCount          uint                `db:"hosts_count" json:"hosts_count"`
	HostsCountUpdatedAt time.Time           `db:"hosts_count_updated_at" json:"hosts_count_updated_at"`
	CreatedAt           time.Time           `db:"created_at" json:"created_at"`
	DetailsLink         string              `json:"details_link"`
	Source              VulnerabilitySource `db:"source" json:"-"`
}

type VulnListOptions struct {
	ListOptions
	IsEE             bool
	ValidSortColumns []string
	TeamID           uint `query:"team_id,optional"`
	KnownExploit     bool `query:"exploit,optional"`
}

func (opt VulnListOptions) HasValidSortColumn() bool {
	if opt.OrderKey == "" || len(opt.ValidSortColumns) == 0 {
		return true
	}
	for _, c := range opt.ValidSortColumns {
		if c == opt.OrderKey {
			return true
		}
	}
	return false
}
