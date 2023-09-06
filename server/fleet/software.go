package fleet

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	SoftwareVendorMaxLengthFmt = "%.111s..."
	SoftwareFieldSeparator     = "\u0000"

	//
	// The following length values must be kept in sync with the DB column definitions.
	//

	SoftwareNameMaxLength             = 255
	SoftwareVersionMaxLength          = 255
	SoftwareSourceMaxLength           = 64
	SoftwareBundleIdentifierMaxLength = 255

	SoftwareReleaseMaxLength = 64
	SoftwareVendorMaxLength  = 114
	SoftwareArchMaxLength    = 16
)

type Vulnerabilities []CVE

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

	// Vulnerabilities lists all found vulnerablities
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

// ToUniqueStr creates a unique string representation of the software
func (s Software) ToUniqueStr() string {
	ss := []string{s.Name, s.Version, s.Source, s.BundleIdentifier}
	// Release, Vendor and Arch fields were added on a migration,
	// thus we only include them in the string if at least one of them is defined.
	if s.Release != "" || s.Vendor != "" || s.Arch != "" {
		ss = append(ss, s.Release, s.Vendor, s.Arch)
	}
	return strings.Join(ss, SoftwareFieldSeparator)
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

type HostSoftwareEntry struct {
	// Software details
	Software
	// Where this software was installed on the host, value is derived from the
	// host_software_installed_paths table.
	InstalledPaths []string `json:"installed_paths,omitempty"`
}

// HostSoftware is the set of software installed on a specific host
type HostSoftware struct {
	// Software is the software information.
	Software []HostSoftwareEntry `json:"software,omitempty" csv:"-"`

	// SoftwareUpdatedAt is the time that the host software was last updated
	SoftwareUpdatedAt time.Time `json:"software_updated_at" db:"software_updated_at" csv:"software_updated_at"`
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

type SoftwareIterQueryOptions struct {
	ExcludedSources []string // what sources to exclude
	IncludedSources []string // what sources to include
}

// IsValid checks that either ExcludedSources or IncludedSources is specified but not both
func (siqo SoftwareIterQueryOptions) IsValid() bool {
	return !(len(siqo.IncludedSources) != 0 && len(siqo.ExcludedSources) != 0)
}

// UpdateHostSoftwareDBResult stores the 'result' of calling 'ds.UpdateHostSoftware' for a host,
// contains the software installed on the host pre-mutations all the mutations performed: what was
// inserted and what was deleted.
type UpdateHostSoftwareDBResult struct {
	// What software was installed on the host before performing any mutations
	WasCurrInstalled []Software
	// What software was deleted
	Deleted []Software
	// What software was inserted
	Inserted []Software
}

// CurrInstalled returns all software that should be currently installed on the host by looking at
// was currently installed, removing anything that was deleted and adding anything that was inserted
func (uhsdbr *UpdateHostSoftwareDBResult) CurrInstalled() []Software {
	var r []Software

	if uhsdbr == nil {
		return r
	}

	deleteMap := map[uint]struct{}{}
	for _, d := range uhsdbr.Deleted {
		deleteMap[d.ID] = struct{}{}
	}

	for _, c := range uhsdbr.WasCurrInstalled {
		if _, ok := deleteMap[c.ID]; !ok {
			r = append(r, c)
		}
	}

	for _, i := range uhsdbr.Inserted {
		r = append(r, i)
	}

	return r
}

// ParseSoftwareLastOpenedAtRowValue attempts to parse the last_opened_at
// software column value. If the value is empty or if the parsed value is
// less or equal than 0 it returns (time.Time{}, nil). We do this because
// some macOS apps return "-1.0" when the app was never opened and we hardcode
// to 0 for some tables that don't have such info.
func ParseSoftwareLastOpenedAtRowValue(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	lastOpenedEpoch, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return time.Time{}, err
	}
	if lastOpenedEpoch <= 0 {
		return time.Time{}, nil
	}
	return time.Unix(int64(lastOpenedEpoch), 0).UTC(), nil
}

// SoftwareFromOsqueryRow creates a fleet.Software from the values reported by osquery.
// Arguments name and source must be defined, all other fields are optional.
// This method doesn't fail if lastOpenedAt is empty or cannot be parsed.
//
// All fields are trimmed to fit on Fleet's database.
// The vendor field is currently trimmed by removing the extra characters and adding `...` at the end.
func SoftwareFromOsqueryRow(name, version, source, vendor, installedPath, release, arch, bundleIdentifier, lastOpenedAt string) (*Software, error) {
	if name == "" {
		return nil, errors.New("host reported software with empty name")
	}
	if source == "" {
		return nil, errors.New("host reported software with empty source")
	}

	// We don't fail if only the last_opened_at cannot be parsed.
	lastOpenedAtTime, _ := ParseSoftwareLastOpenedAtRowValue(lastOpenedAt)

	// Check whether the vendor is longer than the max allowed width and if so, truncate it.
	if utf8.RuneCountInString(vendor) >= SoftwareVendorMaxLength {
		vendor = fmt.Sprintf(SoftwareVendorMaxLengthFmt, vendor)
	}

	truncateString := func(str string, length int) string {
		runes := []rune(str)
		if len(runes) > length {
			return string(runes[:length])
		}
		return str
	}

	software := Software{
		Name:             truncateString(name, SoftwareNameMaxLength),
		Version:          truncateString(version, SoftwareVersionMaxLength),
		Source:           truncateString(source, SoftwareSourceMaxLength),
		BundleIdentifier: truncateString(bundleIdentifier, SoftwareBundleIdentifierMaxLength),

		Release: truncateString(release, SoftwareReleaseMaxLength),
		Vendor:  vendor,
		Arch:    truncateString(arch, SoftwareArchMaxLength),
	}
	if !lastOpenedAtTime.IsZero() {
		software.LastOpenedAt = &lastOpenedAtTime
	}
	return &software, nil
}
