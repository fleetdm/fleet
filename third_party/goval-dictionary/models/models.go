package models

import (
	"time"

	"gorm.io/gorm"
)

// LatestSchemaVersion manages the Schema version used in the latest goval-dictionary.
const LatestSchemaVersion = 2

// FetchMeta has DB information
type FetchMeta struct {
	gorm.Model        `json:"-"`
	GovalDictRevision string
	SchemaVersion     uint
	LastFetchedAt     time.Time
}

// OutDated checks whether last fetched feed is out dated
func (f FetchMeta) OutDated() bool {
	return f.SchemaVersion != LatestSchemaVersion
}

// Root is root struct
type Root struct {
	ID          uint   `gorm:"primary_key"`
	Family      string `gorm:"type:varchar(255)"`
	OSVersion   string `gorm:"type:varchar(255)"`
	Definitions []Definition
	Timestamp   time.Time
}

// Definition : >definitions>definition
type Definition struct {
	ID     uint `gorm:"primary_key" json:"-"`
	RootID uint `gorm:"index:idx_definition_root_id" json:"-" xml:"-"`

	DefinitionID  string `gorm:"type:varchar(255)"`
	Title         string `gorm:"type:text"`
	Description   string // If the type:text, varchar(255) is specified, MySQL overflows and gives an error. No problem in GORMv2. (https://github.com/go-gorm/mysql/tree/15e2cbc6fd072be99215a82292e025dab25e2e16#configuration)
	Advisory      Advisory
	Debian        *Debian
	AffectedPacks []Package
	References    []Reference
}

// Package affected
type Package struct {
	ID           uint `gorm:"primary_key" json:"-"`
	DefinitionID uint `gorm:"index:idx_packages_definition_id" json:"-" xml:"-"`

	Name            string `gorm:"index:idx_packages_name"` // If the type:text, varchar(255) is specified, MySQL overflows and gives an error. No problem in GORMv2. (https://github.com/go-gorm/mysql/tree/15e2cbc6fd072be99215a82292e025dab25e2e16#configuration)
	Version         string `gorm:"type:varchar(255)"`       // affected earlier than this version
	Arch            string `gorm:"type:varchar(255)"`       // Used for Amazon Linux, Oracle Linux and Fedora
	NotFixedYet     bool   // Used for RedHat, Ubuntu
	ModularityLabel string `gorm:"type:varchar(255)"` // RHEL 8 or later only
}

// Reference : >definitions>definition>metadata>reference
type Reference struct {
	ID           uint `gorm:"primary_key" json:"-"`
	DefinitionID uint `gorm:"index:idx_reference_definition_id" json:"-" xml:"-"`

	Source string `gorm:"type:varchar(255)"`
	RefID  string `gorm:"type:varchar(255)"`
	RefURL string `gorm:"type:text"`
}

// Advisory : >definitions>definition>metadata>advisory
type Advisory struct {
	ID           uint `gorm:"primary_key" json:"-"`
	DefinitionID uint `gorm:"index:idx_advisories_definition_id" json:"-" xml:"-"`

	Severity           string `gorm:"type:varchar(255)"`
	Cves               []Cve
	Bugzillas          []Bugzilla
	AffectedResolution []Resolution
	AffectedCPEList    []Cpe
	AffectedRepository string `gorm:"type:varchar(255)"` // Amazon Linux 2 Only
	Issued             time.Time
	Updated            time.Time
}

// Cve : >definitions>definition>metadata>advisory>cve
type Cve struct {
	ID         uint `gorm:"primary_key" json:"-"`
	AdvisoryID uint `gorm:"index:idx_cves_advisory_id" json:"-" xml:"-"`

	CveID  string `gorm:"type:varchar(255)"`
	Cvss2  string `gorm:"type:varchar(255)"`
	Cvss3  string `gorm:"type:varchar(255)"`
	Cwe    string `gorm:"type:varchar(255)"`
	Impact string `gorm:"type:varchar(255)"`
	Href   string `gorm:"type:varchar(255)"`
	Public string `gorm:"type:varchar(255)"`
}

// Bugzilla : >definitions>definition>metadata>advisory>bugzilla
type Bugzilla struct {
	ID         uint `gorm:"primary_key" json:"-"`
	AdvisoryID uint `gorm:"index:idx_bugzillas_advisory_id" json:"-" xml:"-"`

	BugzillaID string `gorm:"type:varchar(255)"`
	URL        string `gorm:"type:varchar(255)"`
	Title      string `gorm:"type:varchar(255)"`
}

// Resolution : >definitions>definition>metadata>advisory>affected>resolution
type Resolution struct {
	ID         uint `gorm:"primary_key" json:"-"`
	AdvisoryID uint `gorm:"index:idx_resolution_advisory_id" json:"-" xml:"-"`

	State      string `gorm:"type:varchar(255)"`
	Components []Component
}

// Component : >definitions>definition>metadata>advisory>affected>resolution>component
type Component struct {
	ID           uint `gorm:"primary_key" json:"-"`
	ResolutionID uint `gorm:"index:idx_component_resolution_id" json:"-" xml:"-"`

	Component string `gorm:"type:varchar(255)"`
}

// Cpe : >definitions>definition>metadata>advisory>affected_cpe_list
type Cpe struct {
	ID         uint `gorm:"primary_key" json:"-"`
	AdvisoryID uint `gorm:"index:idx_cpes_advisory_id" json:"-" xml:"-"`

	Cpe string `gorm:"type:varchar(255)"`
}

// Debian : >definitions>definition>metadata>debian
type Debian struct {
	ID           uint `gorm:"primary_key" json:"-"`
	DefinitionID uint `gorm:"index:idx_debian_definition_id" json:"-" xml:"-"`

	DSA      string `gorm:"type:varchar(255)"`
	MoreInfo string `gorm:"type:text"`

	Date time.Time
}
