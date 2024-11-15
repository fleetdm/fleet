package fleet

import (
	"encoding/json"
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
	SoftwareExtensionIDMaxLength      = 255
	SoftwareBrowserMaxLength          = 255

	SoftwareReleaseMaxLength = 64
	SoftwareVendorMaxLength  = 114
	SoftwareArchMaxLength    = 16

	// SoftwareTeamIdentifierMaxLength is the max length for Apple's Team ID,
	// see https://developer.apple.com/help/account/manage-your-team/locate-your-team-id
	SoftwareTeamIdentifierMaxLength = 10
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
	// ExtensionID is the browser extension id (from osquery chrome_extensions and firefox_addons)
	ExtensionID string `json:"extension_id,omitempty" db:"extension_id"`
	// Browser is the browser type (from osquery chrome_extensions)
	Browser string `json:"browser" db:"browser"`

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

	// TitleID is the ID of the associated software title, representing a unique combination of name
	// and source.
	TitleID *uint `json:"-" db:"title_id"`
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
	// ExtensionID and Browser were added in a single migration, so they are only included if they exist.
	// This way a blank ExtensionID/Browser matches the pre-migration unique string.
	if s.ExtensionID != "" || s.Browser != "" {
		ss = append(ss, s.ExtensionID, s.Browser)
	}
	return strings.Join(ss, SoftwareFieldSeparator)
}

type VulnerableSoftware struct {
	ID                uint    `json:"id" db:"id"`
	Name              string  `json:"name" db:"name"`
	Version           string  `json:"version" db:"version"`
	Source            string  `json:"source" db:"source"`
	Browser           string  `json:"browser" db:"browser"`
	GenerateCPE       string  `json:"generated_cpe" db:"generated_cpe"`
	HostsCount        int     `json:"hosts_count,omitempty" db:"hosts_count"`
	ResolvedInVersion *string `json:"resolved_in_version" db:"resolved_in_version"`
}

type VulnSoftwareFilter struct {
	HostID *uint
	Name   string // LIKE filter
	Source string // exact match
}

type SliceString []string

func (c *SliceString) Scan(v interface{}) error {
	if tv, ok := v.([]byte); ok {
		return json.Unmarshal(tv, &c)
	}
	return errors.New("unsupported type")
}

// SoftwareVersion is an abstraction over the `software` table to support the
// software titles APIs
type SoftwareVersion struct {
	ID uint `db:"id" json:"id"`
	// Version is the version string we grab for this specific software.
	Version string `db:"version" json:"version"`
	// Vulnerabilities is the list of CVE names for vulnerabilities found for this version.
	Vulnerabilities *SliceString `db:"vulnerabilities" json:"vulnerabilities"`
	// HostsCount is the number of hosts that use this software version.
	HostsCount *uint `db:"hosts_count" json:"hosts_count,omitempty"`

	// TitleID is used only as an auxiliary field and it's not part of the
	// JSON response.
	TitleID uint `db:"title_id" json:"-"`
}

// SoftwareTitle represents a title backed by the `software_titles` table.
type SoftwareTitle struct {
	ID uint `json:"id" db:"id"`
	// Name is the name reported by osquery.
	Name string `json:"name" db:"name"`
	// Source is the source reported by osquery.
	Source string `json:"source" db:"source"`
	// Browser is the browser type (e.g., "chrome", "firefox", "safari")
	Browser string `json:"browser,omitempty" db:"browser"`
	// HostsCount is the number of hosts that use this software title.
	HostsCount uint `json:"hosts_count" db:"hosts_count"`
	// VesionsCount is the number of versions that have the same title.
	VersionsCount uint `json:"versions_count" db:"versions_count"`
	// Versions countains information about the versions that use this title.
	Versions []SoftwareVersion `json:"versions" db:"-"`
	// CountsUpdatedAt is the timestamp when the hosts count
	// was last updated for that software title
	CountsUpdatedAt *time.Time `json:"counts_updated_at" db:"counts_updated_at"`
	// SoftwareInstallersCount is 0 or 1, indicating if the software title has an
	// installer. This is an internal field for an optimization so that the extra
	// queries to fetch installer information is done only if necessary.
	SoftwareInstallersCount int `json:"-" db:"software_installers_count"`
	// VPPAppsCount is 0 or 1, indicating if the software title has a VPP app.
	// This is an internal field for an optimization so that the extra queries to
	// fetch app information is done only if necessary.
	VPPAppsCount int `json:"-" db:"vpp_apps_count"`
	// SoftwarePackage is the software installer information for this title.
	SoftwarePackage *SoftwareInstaller `json:"software_package" db:"-"`
	// AppStoreApp is the VPP app information for this title.
	AppStoreApp *VPPAppStoreApp `json:"app_store_app" db:"-"`
	// BundleIdentifier is used by Apple installers to uniquely identify
	// the software installed. It's surfaced in software_titles to match
	// with existing software entries.
	BundleIdentifier *string `json:"bundle_identifier,omitempty" db:"bundle_identifier"`
}

// This type is essentially the same as the above SoftwareTitle type. The only difference is that
// SoftwarePackage is a string pointer here. This type is for use when listing out SoftwareTitles;
// the above type is used when fetching them individually.
type SoftwareTitleListResult struct {
	ID uint `json:"id" db:"id"`
	// Name is the name reported by osquery.
	Name string `json:"name" db:"name"`
	// Source is the source reported by osquery.
	Source string `json:"source" db:"source"`
	// Browser is the browser type (e.g., "chrome", "firefox", "safari")
	Browser string `json:"browser,omitempty" db:"browser"`
	// HostsCount is the number of hosts that use this software title.
	HostsCount uint `json:"hosts_count" db:"hosts_count"`
	// VesionsCount is the number of versions that have the same title.
	VersionsCount uint `json:"versions_count" db:"versions_count"`
	// Versions countains information about the versions that use this title.
	Versions []SoftwareVersion `json:"versions" db:"-"`
	// CountsUpdatedAt is the timestamp when the hosts count
	// was last updated for that software title
	CountsUpdatedAt *time.Time `json:"-" db:"counts_updated_at"`

	// SoftwarePackage provides software installer package information, it is
	// only present if a software installer is available for the software title.
	SoftwarePackage *SoftwarePackageOrApp `json:"software_package"`

	// AppStoreApp provides VPP app information, it is only present if a VPP app
	// is available for the software title.
	AppStoreApp *SoftwarePackageOrApp `json:"app_store_app"`
	// BundleIdentifier is used by Apple installers to uniquely identify
	// the software installed. It's surfaced in software_titles to match
	// with existing software entries.
	BundleIdentifier *string `json:"bundle_identifier,omitempty" db:"bundle_identifier"`
}

type SoftwareTitleListOptions struct {
	// ListOptions cannot be embedded in order to unmarshall with validation.
	ListOptions ListOptions `url:"list_options"`

	TeamID              *uint   `query:"team_id,optional"`
	VulnerableOnly      bool    `query:"vulnerable,optional"`
	AvailableForInstall bool    `query:"available_for_install,optional"`
	SelfServiceOnly     bool    `query:"self_service,optional"`
	KnownExploit        bool    `query:"exploit,optional"`
	MinimumCVSS         float64 `query:"min_cvss_score,optional"`
	MaximumCVSS         float64 `query:"max_cvss_score,optional"`
	PackagesOnly        bool    `query:"packages_only,optional"`
	Platform            string  `query:"platform,optional"`
}

type HostSoftwareTitleListOptions struct {
	// ListOptions cannot be embedded in order to unmarshal with validation.
	ListOptions ListOptions `url:"list_options"`

	// SelfServiceOnly limits the returned software titles to those that are
	// available to install by the end user via the self-service. Implies
	// AvailableForInstall.
	SelfServiceOnly bool `query:"self_service,optional"`

	// IncludeAvailableForInstall is not a query argument, it is set in the
	// service layer to indicate to the datastore if software available for
	// install (but not currently installed on the host) should be returned.
	IncludeAvailableForInstall bool

	// OnlyAvailableForInstall is set via a query argument that limits the
	// returned software titles to only those that are available for install on
	// the host.
	OnlyAvailableForInstall bool `query:"available_for_install,optional"`

	VulnerableOnly bool `query:"vulnerable,optional"`

	// Non-MDM-enabled hosts cannot install VPP apps
	IsMDMEnrolled bool
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
	InstalledPaths           []string                   `json:"installed_paths"`
	PathSignatureInformation []PathSignatureInformation `json:"signature_information"`
}

type PathSignatureInformation struct {
	InstalledPath  string `json:"installed_path"`
	TeamIdentifier string `json:"team_identifier"`
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
	// ListOptions cannot be embedded in order to unmarshall with validation.
	ListOptions ListOptions `url:"list_options"`

	// HostID filters software to the specified host if not nil.
	HostID           *uint
	TeamID           *uint `query:"team_id,optional"`
	VulnerableOnly   bool  `query:"vulnerable,optional"`
	IncludeCVEScores bool
	KnownExploit     bool    `query:"exploit,optional"`
	MinimumCVSS      float64 `query:"min_cvss_score,optional"`
	MaximumCVSS      float64 `query:"max_cvss_score,optional"`

	// WithHostCounts indicates that the list of software should include the
	// counts of hosts per software, and include only those software that have
	// a count of hosts > 0.
	WithHostCounts bool
}

type SoftwareIterQueryOptions struct {
	ExcludedSources []string // what sources to exclude
	IncludedSources []string // what sources to include
	NameMatch       string   // mysql regex to filter software by name
	NameExclude     string   // mysql regex to filter software by name
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

	r = append(r, uhsdbr.Inserted...)

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
func SoftwareFromOsqueryRow(
	name, version, source, vendor, installedPath, release, arch,
	bundleIdentifier, extensionId, browser, lastOpenedAt string,
) (*Software, error) {
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
		ExtensionID:      truncateString(extensionId, SoftwareExtensionIDMaxLength),
		Browser:          truncateString(browser, SoftwareBrowserMaxLength),

		Release: truncateString(release, SoftwareReleaseMaxLength),
		Vendor:  vendor,
		Arch:    truncateString(arch, SoftwareArchMaxLength),
	}
	if !lastOpenedAtTime.IsZero() {
		software.LastOpenedAt = &lastOpenedAtTime
	}
	return &software, nil
}

type VPPBatchPayload struct {
	AppStoreID         string `json:"app_store_id"`
	SelfService        bool   `json:"self_service"`
	InstallDuringSetup *bool  `json:"install_during_setup"` // keep saved value if nil, otherwise set as indicated
}

type VPPBatchPayloadWithPlatform struct {
	AppStoreID         string              `json:"app_store_id"`
	SelfService        bool                `json:"self_service"`
	Platform           AppleDevicePlatform `json:"platform"`
	InstallDuringSetup *bool               `json:"install_during_setup"` // keep saved value if nil, otherwise set as indicated
}
