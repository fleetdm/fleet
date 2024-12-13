package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver"
)

type HostStatus string

const (
	// StatusOnline host is active.
	StatusOnline = HostStatus("online")
	// StatusOffline no communication with host for OfflineDuration.
	StatusOffline = HostStatus("offline")
	// StatusMIA no communication with host for MIADuration.
	StatusMIA = HostStatus("mia")
	// StatusNew means the host has enrolled in the interval defined by
	// NewDuration. It is independent of offline and online.
	StatusNew = HostStatus("new")
	// StatusMissing means the host is missing for 30 days. It is identical
	// with StatusMIA, but StatusMIA is deprecated.
	StatusMissing = HostStatus("missing")

	// NewDuration if a host has been created within this time period it's
	// considered new.
	NewDuration = 24 * time.Hour

	// MIADuration if a host hasn't been in communication for this period it
	// is considered MIA.
	MIADuration = 30 * 24 * time.Hour

	// OnlineIntervalBuffer is the additional time in seconds to add to the
	// online interval to avoid flapping of hosts that check in a bit later
	// than their expected checkin interval.
	OnlineIntervalBuffer = 60

	// HostIdentiferNotFound is the error message returned when a search for a host by its
	// identifier (hostname, UUID, or serial number) does not return any results.
	HostIdentiferNotFound = "Host doesn't exist. Make sure you provide a valid hostname, UUID, or serial number. Learn more about host identifiers: https://fleetdm.com/learn-more-about/host-identifiers"
)

func (s HostStatus) IsValid() bool {
	switch s {
	case StatusOnline, StatusOffline, StatusNew, StatusMissing, StatusMIA:
		return true
	default:
		return false
	}
}

// MDMEnrollStatus defines the possible MDM enrollment statuses.
type MDMEnrollStatus string

const (
	MDMEnrollStatusManual     = MDMEnrollStatus("manual")
	MDMEnrollStatusAutomatic  = MDMEnrollStatus("automatic")
	MDMEnrollStatusPending    = MDMEnrollStatus("pending")
	MDMEnrollStatusUnenrolled = MDMEnrollStatus("unenrolled")
	MDMEnrollStatusEnrolled   = MDMEnrollStatus("enrolled") // combination of "manual" and "automatic"
)

// OSSettingsStatus defines the possible statuses of the host's OS settings, which is derived from the
// status of MDM configuration profiles and non-profile settings applied the host.
type OSSettingsStatus string

const (
	OSSettingsVerified  OSSettingsStatus = "verified"
	OSSettingsVerifying OSSettingsStatus = "verifying"
	OSSettingsPending   OSSettingsStatus = "pending"
	OSSettingsFailed    OSSettingsStatus = "failed"
)

func (s OSSettingsStatus) IsValid() bool {
	switch s {
	case OSSettingsFailed, OSSettingsPending, OSSettingsVerifying, OSSettingsVerified:
		return true
	default:
		return false
	}
}

// MDMBootstrapPackageStatus defines the possible statuses of the host's MDM bootstrap package,
// which is derived from the status of the MDM command to install the bootstrap package.
//
// See https://developer.apple.com/documentation/devicemanagement/installenterpriseapplicationresponse
type MDMBootstrapPackageStatus string

const (
	// MDMBootstrapPackageInstalled means the bootstrap package has been installed on the host. It
	// corresponds to InstallEnterpriseApplicationResponse.Status "Acknowledged".
	MDMBootstrapPackageInstalled = MDMBootstrapPackageStatus("installed")
	// MDMBootstrapPackageFailed means the bootstrap package failed to install on the host. It
	// corresponds to InstallEnterpriseApplicationResponse.Status "Error".
	MDMBootstrapPackageFailed = MDMBootstrapPackageStatus("failed")
	// MDMBootstrapPackagePending means the bootstrap package is pending installation on the host.
	// It applies if no InstallEnterpriseApplicationResponse has been received or if the response is
	// anything other than InstallEnterpriseApplicationResponse.Status "Acknowledged" or "Error".
	MDMBootstrapPackagePending = MDMBootstrapPackageStatus("pending")
)

func (s MDMBootstrapPackageStatus) IsValid() bool {
	switch s {
	case MDMBootstrapPackageInstalled, MDMBootstrapPackagePending, MDMBootstrapPackageFailed:
		return true
	default:
		return false
	}
}

// NOTE: any changes to the hosts filters is likely to impact at least the following
// endpoints, due to how they share the same implementation at the Datastore level:
//
// - GET /hosts (list hosts)
// - GET /hosts/count (count hosts, which calls svc.CountHosts or svc.CountHostsInLabel)
// - GET /labels/{id}/hosts (list hosts in label)
// - GET /hosts/report
// - POST /hosts/delete (calls svc.hostIDsFromFilters)
// - POST /hosts/transfer/filter (calls svc.hostIDsFromFilters)
//
// Make sure the docs are updated accordingly and all endpoints behave as expected.
type HostListOptions struct {
	ListOptions

	// DeviceMapping joins device user email mapping for each host if available
	DeviceMapping bool

	// AdditionalFilters selects which host additional fields should be
	// populated.
	AdditionalFilters []string
	// StatusFilter selects the online status of the hosts.
	StatusFilter HostStatus
	// TeamFilter selects the hosts for specified team
	TeamFilter *uint

	PolicyIDFilter       *uint
	PolicyResponseFilter *bool

	// Deprecated: SoftwareIDFilter is deprecated as of Fleet 4.42. It is
	// maintained for backwards compatibility. Use SoftwareVersionIDFilter
	// instead.
	SoftwareIDFilter *uint
	// SoftwareVersionIDFilter filters the hosts by the software version ID that
	// they use. This identifies a specific version of a "software title".
	SoftwareVersionIDFilter *uint
	// SoftwareTitleIDFilter filers the hosts by the software title ID that they
	// use. This identifies a "software title" independent of the specific
	// version.
	SoftwareTitleIDFilter *uint
	// SoftwareStatusFilter filters the hosts by the status of the software installer, if any,
	// managed by Fleet. If specified, the SoftwareTitleIDFilter must also be specified.
	SoftwareStatusFilter *SoftwareInstallerStatus

	OSIDFilter        *uint
	OSNameFilter      *string
	OSVersionFilter   *string
	OSVersionIDFilter *uint

	DisableIssues bool

	// MacOSSettingsFilter filters the hosts by the status of MDM configuration profiles
	// applied to the hosts.
	MacOSSettingsFilter OSSettingsStatus

	// MacOSSettingsDiskEncryptionFilter filters the hosts by the status of the disk encryption
	// MDM profile.
	MacOSSettingsDiskEncryptionFilter DiskEncryptionStatus

	// OSSettingsFilter filters the hosts by the status of MDM configuration profiles and
	// non-profile settings applied to the hosts.
	OSSettingsFilter OSSettingsStatus
	// OSSettingsDiskEncryptionFilter filters the hosts by the status of the disk encryption
	// OS setting.
	OSSettingsDiskEncryptionFilter DiskEncryptionStatus

	// MDMBootstrapPackageFilter filters the hosts by the status of the MDM bootstrap package.
	MDMBootstrapPackageFilter *MDMBootstrapPackageStatus

	// MDMIDFilter filters the hosts by MDM ID.
	MDMIDFilter *uint
	// MDMNameFilter filters the hosts by MDM solution name (e.g. one of the
	// fleet.WellKnownMDM... constants).
	MDMNameFilter *string
	// MDMEnrollmentStatusFilter filters the host by their MDM enrollment status.
	MDMEnrollmentStatusFilter MDMEnrollStatus
	// MunkiIssueIDFilter filters the hosts by munki issue ID.
	MunkiIssueIDFilter *uint

	// LowDiskSpaceFilter filters the hosts by low disk space (defined as a host
	// with less than N gigs of disk space available). Note that this is a Fleet
	// Premium feature, Fleet Free ignores the setting (it forces it to nil to
	// disable it).
	LowDiskSpaceFilter *int

	// PopulateSoftware adds the `Software` field to all Hosts returned.
	PopulateSoftware bool

	// PopulateSoftwareVulnerabilityDetails adds description, fix version, etc. fields to software vulnerabilities
	// (this is a Premium feature that gets forced to false on Fleet Free)
	PopulateSoftwareVulnerabilityDetails bool

	// PopulatePolicies adds the `Policies` array field to all Hosts returned.
	PopulatePolicies bool

	// VulnerabilityFilter filters the hosts by the presence of a vulnerability (CVE)
	VulnerabilityFilter *string

	// ConnectedToFleetFilter filters hosts that have an active MDM
	// connection with this Fleet instance.
	ConnectedToFleetFilter *bool
}

// TODO(Sarah): Are we missing any filters here? Should all MDM filters be included?
func (h HostListOptions) Empty() bool {
	return h.ListOptions.Empty() &&
		!h.DeviceMapping &&
		len(h.AdditionalFilters) == 0 &&
		h.StatusFilter == "" &&
		h.TeamFilter == nil &&
		h.PolicyIDFilter == nil &&
		h.PolicyResponseFilter == nil &&
		h.SoftwareIDFilter == nil &&
		h.SoftwareVersionIDFilter == nil &&
		h.SoftwareTitleIDFilter == nil &&
		h.SoftwareStatusFilter == nil &&
		h.OSIDFilter == nil &&
		h.OSNameFilter == nil &&
		h.OSVersionFilter == nil &&
		!h.DisableIssues &&
		h.MacOSSettingsFilter == "" &&
		h.MacOSSettingsDiskEncryptionFilter == "" &&
		h.MDMBootstrapPackageFilter == nil &&
		h.MDMIDFilter == nil &&
		h.MDMNameFilter == nil &&
		h.MDMEnrollmentStatusFilter == "" &&
		h.MunkiIssueIDFilter == nil &&
		h.LowDiskSpaceFilter == nil &&
		h.OSSettingsFilter == "" &&
		h.OSSettingsDiskEncryptionFilter == ""
}

type HostUser struct {
	Uid       uint   `json:"uid" db:"uid"`
	Username  string `json:"username" db:"username"`
	Type      string `json:"type" db:"user_type"`
	GroupName string `json:"groupname" db:"groupname"`
	Shell     string `json:"shell" db:"shell"`
}

type Host struct {
	UpdateCreateTimestamps
	HostSoftware
	ID uint `json:"id" csv:"id"`
	// OsqueryHostID is the key used in the request context that is
	// used to retrieve host information.  It is sent from osquery and may currently be
	// a GUID or a Host Name, but in either case, it MUST be unique
	OsqueryHostID    *string   `json:"-" db:"osquery_host_id" csv:"-"`
	DetailUpdatedAt  time.Time `json:"detail_updated_at" db:"detail_updated_at" csv:"detail_updated_at"` // Time that the host details were last updated
	LabelUpdatedAt   time.Time `json:"label_updated_at" db:"label_updated_at" csv:"label_updated_at"`    // Time that the host labels were last updated
	PolicyUpdatedAt  time.Time `json:"policy_updated_at" db:"policy_updated_at" csv:"policy_updated_at"` // Time that the host policies were last updated
	LastEnrolledAt   time.Time `json:"last_enrolled_at" db:"last_enrolled_at" csv:"last_enrolled_at"`    // Time that the host last enrolled
	SeenTime         time.Time `json:"seen_time" db:"seen_time" csv:"seen_time"`                         // Time that the host was last "seen"
	RefetchRequested bool      `json:"refetch_requested" db:"refetch_requested" csv:"refetch_requested"`
	NodeKey          *string   `json:"-" db:"node_key" csv:"-"`
	OrbitNodeKey     *string   `json:"-" db:"orbit_node_key" csv:"-"`
	Hostname         string    `json:"hostname" db:"hostname" csv:"hostname"` // there is a fulltext index on this field
	UUID             string    `json:"uuid" db:"uuid" csv:"uuid"`             // there is a fulltext index on this field
	// Platform is the host's platform as defined by osquery's os_version.platform.
	Platform       string        `json:"platform" csv:"platform"`
	OsqueryVersion string        `json:"osquery_version" db:"osquery_version" csv:"osquery_version"`
	OrbitVersion   *string       `json:"orbit_version" db:"orbit_version" csv:"orbit_version"`
	DesktopVersion *string       `json:"fleet_desktop_version" db:"fleet_desktop_version" csv:"fleet_desktop_version"`
	ScriptsEnabled *bool         `json:"scripts_enabled" db:"scripts_enabled" csv:"scripts_enabled"`
	OSVersion      string        `json:"os_version" db:"os_version" csv:"os_version"`
	Build          string        `json:"build" csv:"build"`
	PlatformLike   string        `json:"platform_like" db:"platform_like" csv:"platform_like"`
	CodeName       string        `json:"code_name" db:"code_name" csv:"code_name"`
	Uptime         time.Duration `json:"uptime" csv:"uptime"`
	Memory         int64         `json:"memory" sql:"type:bigint" db:"memory" csv:"memory"`
	// system_info fields
	CPUType          string `json:"cpu_type" db:"cpu_type" csv:"cpu_type"`
	CPUSubtype       string `json:"cpu_subtype" db:"cpu_subtype" csv:"cpu_subtype"`
	CPUBrand         string `json:"cpu_brand" db:"cpu_brand" csv:"cpu_brand"`
	CPUPhysicalCores int    `json:"cpu_physical_cores" db:"cpu_physical_cores" csv:"cpu_physical_cores"`
	CPULogicalCores  int    `json:"cpu_logical_cores" db:"cpu_logical_cores" csv:"cpu_logical_cores"`
	HardwareVendor   string `json:"hardware_vendor" db:"hardware_vendor" csv:"hardware_vendor"`
	HardwareModel    string `json:"hardware_model" db:"hardware_model" csv:"hardware_model"`
	HardwareVersion  string `json:"hardware_version" db:"hardware_version" csv:"hardware_version"`
	HardwareSerial   string `json:"hardware_serial" db:"hardware_serial" csv:"hardware_serial"`
	ComputerName     string `json:"computer_name" db:"computer_name" csv:"computer_name"`
	// PrimaryNetworkInterfaceID if present indicates to primary network for the host, the details of which
	// can be found in the NetworkInterfaces element with the same ip_address.
	PrimaryNetworkInterfaceID *uint               `json:"primary_ip_id,omitempty" db:"primary_ip_id" csv:"primary_ip_id"`
	NetworkInterfaces         []*NetworkInterface `json:"-" db:"-" csv:"-"`
	PublicIP                  string              `json:"public_ip" db:"public_ip" csv:"public_ip"`
	PrimaryIP                 string              `json:"primary_ip" db:"primary_ip" csv:"primary_ip"`
	PrimaryMac                string              `json:"primary_mac" db:"primary_mac" csv:"primary_mac"`
	DistributedInterval       uint                `json:"distributed_interval" db:"distributed_interval" csv:"distributed_interval"`
	ConfigTLSRefresh          uint                `json:"config_tls_refresh" db:"config_tls_refresh" csv:"config_tls_refresh"`
	LoggerTLSPeriod           uint                `json:"logger_tls_period" db:"logger_tls_period" csv:"logger_tls_period"`
	TeamID                    *uint               `json:"team_id" db:"team_id" csv:"team_id"`

	// Loaded via JOIN in DB
	PackStats []PackStats `json:"pack_stats" csv:"-"`
	// TeamName is the name of the team, loaded by JOIN to the teams table.
	TeamName *string `json:"team_name" db:"team_name" csv:"team_name"`
	// Additional is the additional information from the host
	// additional_queries. This should be stored in a separate DB table.
	Additional *json.RawMessage `json:"additional,omitempty" db:"additional" csv:"-"`

	// Users currently in the host
	Users []HostUser `json:"users,omitempty" csv:"-"`

	GigsDiskSpaceAvailable    float64 `json:"gigs_disk_space_available" db:"gigs_disk_space_available" csv:"gigs_disk_space_available"`
	PercentDiskSpaceAvailable float64 `json:"percent_disk_space_available" db:"percent_disk_space_available" csv:"percent_disk_space_available"`
	GigsTotalDiskSpace        float64 `json:"gigs_total_disk_space" db:"gigs_total_disk_space" csv:"gigs_total_disk_space"`

	// DiskEncryptionEnabled is only returned by GET /host/{id} and so is not
	// exportable as CSV (which is the result of List Hosts endpoint). It is
	// a *bool because for some Linux we set it to NULL and omit it from the JSON
	// response if the host does not have disk encryption enabled. It is also
	// omitted if we don't have encryption information yet.
	DiskEncryptionEnabled *bool `json:"disk_encryption_enabled,omitempty" db:"disk_encryption_enabled" csv:"-"`

	HostIssues `json:"issues,omitempty" csv:"-"`

	// DeviceMapping is in fact included in the CSV export, but it is not directly
	// encoded from this column, it is processed before marshaling, hence why the
	// struct tag here has csv:"-".
	DeviceMapping *json.RawMessage `json:"device_mapping,omitempty" db:"device_mapping" csv:"-"`

	MDM MDMHostData `json:"mdm" db:"mdm_host_data" csv:"-"`

	// RefetchCriticalQueriesUntil can be set to a timestamp up to which the
	// "critical" queries will be constantly reported to the host that checks in
	// to be re-executed until a condition is met (or the timestamp expires). The
	// notion of "critical query" is voluntarily loosely defined so that future
	// requirements may use this mechanism. The difference with RefetchRequested
	// is that the latter is a one-time request, while this one is a persistent
	// until the timestamp expires. The initial use-case is to check for a host
	// to be unenrolled from its old MDM solution, in the "migrate to Fleet MDM"
	// workflow (both Apple and Windows).
	//
	// In the future, if we want to use it for more than one use-case, we could
	// add a "reason" field with well-known labels so we know what condition(s)
	// are expected to clear the timestamp. For now there's a single use-case
	// so we don't need this.
	RefetchCriticalQueriesUntil *time.Time `json:"refetch_critical_queries_until" db:"refetch_critical_queries_until" csv:"-"`

	// DEPAssignedToFleet is set to true if the host is assigned to Fleet in Apple Business Manager.
	// It is a *bool becase we want it to be returned from only a subset of endpoints related to
	// Orbit and Fleet Desktop. Otherwise, it will be set to NULL so it is omitted from JSON
	// responses.
	//
	// The boolean is based on information ingested from the Apple DEP API that is stored in the
	// host_dep_assignments table.
	DEPAssignedToFleet *bool `json:"dep_assigned_to_fleet,omitempty" db:"dep_assigned_to_fleet" csv:"-"`

	// LastRestartedAt is a UNIX timestamp that indicates when the Host was last restarted.
	LastRestartedAt time.Time `json:"last_restarted_at" db:"last_restarted_at" csv:"last_restarted_at"`

	// Policies is the list of policies and whether it passes for the host
	Policies *[]*HostPolicy `json:"policies,omitempty" csv:"-"`
}

// HostOrbitInfo maps to the host_orbit_info table in the database, which maps to the orbit_info agent table.
type HostOrbitInfo struct {
	Version        string  `json:"version" db:"version"`
	DesktopVersion *string `json:"desktop_version" db:"desktop_version"`
	ScriptsEnabled *bool   `json:"scripts_enabled" db:"scripts_enabled"`
}

// HostHealth contains a subset of Host data that indicates how healthy a Host is. For fields with
// the same name, see the comments/docs for the Host field above.
type HostHealth struct {
	UpdatedAt                    time.Time                      `json:"updated_at,omitempty" db:"updated_at"`
	OsVersion                    string                         `json:"os_version,omitempty" db:"os_version"`
	DiskEncryptionEnabled        *bool                          `json:"disk_encryption_enabled,omitempty" db:"disk_encryption_enabled"`
	FailingPoliciesCount         int                            `json:"failing_policies_count"`
	FailingCriticalPoliciesCount *int                           `json:"failing_critical_policies_count,omitempty"` // Fleet Premium Only
	VulnerableSoftware           []HostHealthVulnerableSoftware `json:"vulnerable_software,omitempty"`
	FailingPolicies              []*HostHealthFailingPolicy     `json:"failing_policies,omitempty"`
	Platform                     string                         `json:"-" db:"platform"`                // Needed to fetch failing policies. Not returned in HTTP responses.
	TeamID                       *uint                          `json:"team_id,omitempty" db:"team_id"` // Needed to verify that user can access this host's health data. Not returned in HTTP responses.
}

type HostHealthVulnerableSoftware struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HostHealthFailingPolicy struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	Critical   *bool   `json:"critical,omitempty"` // Fleet Premium Only
	Resolution *string `json:"resolution"`
}

func (hh HostHealth) AuthzType() string {
	return "host_health"
}

type MDMHostData struct {
	// For CSV columns, since the CSV is flattened, we keep the "mdm." prefix
	// along with the column name.

	// EnrollmentStatus is a string representation of state derived from
	// booleans stored in the host_mdm table, loaded by JOIN in datastore
	EnrollmentStatus *string `json:"enrollment_status" db:"-" csv:"mdm.enrollment_status"`
	// DEPProfileError is a boolean representing whether Fleet received a "FAILED" response when
	// attempting to assign a DEP profile for the host.
	// See https://developer.apple.com/documentation/devicemanagement/assignprofileresponse
	//
	// It is not filled in by all host-returning datastore methods.
	DEPProfileError bool `json:"dep_profile_error" db:"dep_profile_error" csv:"mdm.dep_profile_error"`
	// ServerURL is the server_url stored in the host_mdm table, loaded by
	// JOIN in datastore
	ServerURL *string `json:"server_url" db:"-" csv:"mdm.server_url"`
	// Name is the name of the MDM solution for the host.
	Name string `json:"name" db:"name" csv:"-"`

	// EncryptionKeyAvailable indicates if Fleet was able to retrieve and
	// decode an encryption key for the host.
	EncryptionKeyAvailable bool `json:"encryption_key_available" db:"-" csv:"-"`

	// this is set to nil if the key exists but decryptable is NULL in the db, 1
	// if decryptable, 0 if non-decryptable and -1 if no disk encryption key row
	// exists for this host. Used internally to determine the disk_encryption
	// status and action_required fields. See MDMHostData.Scan as for where this
	// gets filled.
	rawDecryptable *int

	// OSSettings contains information related to operating systems settings that are managed for
	// MDM-enrolled hosts and/or Linux hosts with disk encryption enabled, which don't require MDM.
	//
	// Note: Additional information for macOS hosts is currently stored in MacOSSettings.
	OSSettings *HostMDMOSSettings `json:"os_settings,omitempty" db:"-" csv:"-"`

	// Profiles is a list of HostMDMProfiles for the host. Note that as for many
	// other host fields, it is not filled in by all host-returning datastore methods.
	//
	// It is a pointer to a slice so that when set, it gets marhsaled even
	// if the slice is empty, but when unset, it doesn't get marshaled
	// (e.g. we don't return that information for the List Hosts endpoint).
	Profiles *[]HostMDMProfile `json:"profiles,omitempty" db:"profiles" csv:"-"`

	// MacOSSettings indicates macOS-specific MDM settings for the host, such
	// as disk encryption status and whether any user action is required to
	// complete the disk encryption process.
	//
	// It is not filled in by all host-returning datastore methods.
	MacOSSettings *MDMHostMacOSSettings `json:"macos_settings,omitempty" db:"-" csv:"-"`

	// MacOSSetup indicates macOS-specific MDM setup for the host, such
	// as the status of the bootstrap package.
	//
	// It is not filled in by all host-returning datastore methods.
	MacOSSetup *HostMDMMacOSSetup `json:"macos_setup,omitempty" db:"-" csv:"-"`

	// The DeviceStatus and PendingAction fields are not stored in the database
	// directly, they are read from the GetHostLockWipeStatus datastore method
	// and determined from those results. They are not filled by all
	// host-returning methods.
	DeviceStatus  *string `json:"device_status,omitempty" db:"-" csv:"-"`
	PendingAction *string `json:"pending_action,omitempty" db:"-" csv:"-"`

	// ConnectedToFleet indicates if the host has an active MDM connection
	// with this Fleet instance. This boolean is not filled by all
	// host-returning methods.
	ConnectedToFleet *bool `json:"connected_to_fleet" csv:"-" db:"connected_to_fleet"`
}

type HostMDMOSSettings struct {
	DiskEncryption HostMDMDiskEncryption `json:"disk_encryption" db:"-" csv:"-"`
}

type HostMDMDiskEncryption struct {
	Status *DiskEncryptionStatus `json:"status" db:"-" csv:"-"`
	Detail string                `json:"detail" db:"-" csv:"-"`
}

type DiskEncryptionStatus string

const (
	DiskEncryptionVerified            DiskEncryptionStatus = "verified"
	DiskEncryptionVerifying           DiskEncryptionStatus = "verifying"
	DiskEncryptionActionRequired      DiskEncryptionStatus = "action_required"
	DiskEncryptionEnforcing           DiskEncryptionStatus = "enforcing"
	DiskEncryptionFailed              DiskEncryptionStatus = "failed"
	DiskEncryptionRemovingEnforcement DiskEncryptionStatus = "removing_enforcement"
)

func (s DiskEncryptionStatus) addrOf() *DiskEncryptionStatus {
	return &s
}

func (s DiskEncryptionStatus) IsValid() bool {
	switch s {
	case
		DiskEncryptionVerifying,
		DiskEncryptionVerified,
		DiskEncryptionActionRequired,
		DiskEncryptionEnforcing,
		DiskEncryptionFailed,
		DiskEncryptionRemovingEnforcement:
		return true
	default:
		return false
	}
}

type ActionRequiredState string

const (
	ActionRequiredLogOut    ActionRequiredState = "log_out"
	ActionRequiredRotateKey ActionRequiredState = "rotate_key"
)

func (s ActionRequiredState) addrOf() *ActionRequiredState {
	return &s
}

type MDMHostMacOSSettings struct {
	DiskEncryption *DiskEncryptionStatus `json:"disk_encryption" csv:"-"`
	ActionRequired *ActionRequiredState  `json:"action_required" csv:"-"`
}

type HostMDMMacOSSetup struct {
	BootstrapPackageStatus MDMBootstrapPackageStatus `db:"bootstrap_package_status" json:"bootstrap_package_status" csv:"-"`
	Result                 []byte                    `db:"result" json:"-" csv:"-"`
	Detail                 string                    `db:"-" json:"detail" csv:"-"`
	BootstrapPackageName   string                    `db:"bootstrap_package_name" json:"bootstrap_package_name" csv:"-"`
}

// PopulateOSSettingsAndMacOSSettings populates the OSSettings and MacOSSettings
// on the MDMHostData struct. It determines the disk encryption status for the
// host based on the file-vault profile in its list of profiles and whether its
// disk encryption key is available and decryptable. The file-vault profile
// identifier is received as an argument to avoid a circular dependency.
//
// NOTE: This overwrites both OSSettings and MacOSSettings on the MDMHostData struct. Any existing
// data in those fields will be lost.
func (d *MDMHostData) PopulateOSSettingsAndMacOSSettings(profiles []HostMDMAppleProfile, fileVaultIdentifier string) {
	var settings MDMHostMacOSSettings

	var fvprof *HostMDMAppleProfile
	for _, p := range profiles {
		p := p
		if p.Identifier == fileVaultIdentifier {
			fvprof = &p
			break
		}
	}
	if fvprof != nil {
		switch fvprof.OperationType {
		case MDMOperationTypeInstall:
			switch {
			case fvprof.Status != nil && (*fvprof.Status == MDMDeliveryVerifying || *fvprof.Status == MDMDeliveryVerified):
				if d.rawDecryptable != nil && *d.rawDecryptable == 1 { //nolint:gocritic // ignore ifElseChain
					//  if a FileVault profile has been successfully installed on the host
					//  AND we have fetched and are able to decrypt the key
					switch *fvprof.Status {
					case MDMDeliveryVerifying:
						settings.DiskEncryption = DiskEncryptionVerifying.addrOf()
					case MDMDeliveryVerified:
						settings.DiskEncryption = DiskEncryptionVerified.addrOf()
					}
				} else if d.rawDecryptable != nil {
					// if a FileVault profile has been successfully installed on the host
					// but either we didn't get an encryption key or we're not able to
					// decrypt the key we've got
					settings.DiskEncryption = DiskEncryptionActionRequired.addrOf()
					settings.ActionRequired = ActionRequiredRotateKey.addrOf()
				} else {
					// if [a FileVault profile is pending to be installed or] the
					// matching row in host_disk_encryption_keys has a field decryptable
					// = NULL
					switch *fvprof.Status {
					case MDMDeliveryVerifying, MDMDeliveryVerified:
						settings.DiskEncryption = DiskEncryptionVerifying.addrOf()
					case MDMDeliveryPending:
						settings.DiskEncryption = DiskEncryptionEnforcing.addrOf()
					}
				}

			case fvprof.Status != nil && *fvprof.Status == MDMDeliveryFailed:
				// if a FileVault profile failed to be installed [or removed]
				settings.DiskEncryption = DiskEncryptionFailed.addrOf()

			default:
				// if a FileVault profile is pending to be installed [or the matching
				// row in host_disk_encryption_keys has a field decryptable = NULL]
				settings.DiskEncryption = DiskEncryptionEnforcing.addrOf()
			}

		case MDMOperationTypeRemove:
			switch {
			case fvprof.Status != nil && *fvprof.Status == MDMDeliveryVerifying:
				// successfully removed, same as if	no filevault profile was found

			case fvprof.Status != nil && *fvprof.Status == MDMDeliveryFailed:
				// if a FileVault profile failed to be [installed or] removed
				settings.DiskEncryption = DiskEncryptionFailed.addrOf()

			default:
				// if a FileVault profile is pending to be removed
				settings.DiskEncryption = DiskEncryptionRemovingEnforcement.addrOf()
			}
		}
	}
	d.MacOSSettings = &settings

	var hde HostMDMDiskEncryption
	if settings.DiskEncryption != nil {
		hde.Status = settings.DiskEncryption
	}
	if fvprof != nil {
		hde.Detail = fvprof.Detail
	}
	d.OSSettings = &HostMDMOSSettings{DiskEncryption: hde}
}

func (d *MDMHostData) ProfileStatusFromDiskEncryptionState(currStatus *MDMDeliveryStatus) *MDMDeliveryStatus {
	if d.MacOSSettings == nil || d.MacOSSettings.DiskEncryption == nil {
		return currStatus
	}
	switch *d.MacOSSettings.DiskEncryption {
	case DiskEncryptionActionRequired, DiskEncryptionEnforcing, DiskEncryptionRemovingEnforcement:
		return &MDMDeliveryPending
	case DiskEncryptionFailed:
		return &MDMDeliveryFailed
	case DiskEncryptionVerifying:
		return &MDMDeliveryVerifying
	case DiskEncryptionVerified:
		return &MDMDeliveryVerified
	default:
		return currStatus
	}
}

// Only exposed for Datastore tests, to be able to assert the rawDecryptable
// unexported field.
func (d *MDMHostData) TestGetRawDecryptable() *int {
	return d.rawDecryptable
}

// Scan implements the Scanner interface for sqlx, to support unmarshaling a
// JSON object from the database into a MDMHostData struct.
func (d *MDMHostData) Scan(v interface{}) error {
	var dst struct {
		MDMHostData
		RawDecryptable *int `json:"raw_decryptable"`
	}
	switch v := v.(type) {
	case []byte:
		if err := json.Unmarshal(v, &dst); err != nil {
			return err
		}
		*d = dst.MDMHostData
		d.rawDecryptable = dst.RawDecryptable
		return nil

	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// IsOsqueryEnrolled returns true if the host is enrolled via osquery.
func (h *Host) IsOsqueryEnrolled() bool {
	return h.OsqueryHostID != nil && *h.OsqueryHostID != ""
}

// IsDEPAssignedToFleet returns true if the host was assigned to the Fleet
// server in ABM.
func (h *Host) IsDEPAssignedToFleet() bool {
	return h.DEPAssignedToFleet != nil && *h.DEPAssignedToFleet
}

// IsLUKSSupported returns true if the host's platform is Linux and running
// one of the supported OS versions.
func (h *Host) IsLUKSSupported() bool {
	return h.Platform == "ubuntu" || strings.Contains(h.OSVersion, "Fedora") // fedora h.Platform reports as "rhel"
}

// IsEligibleForWindowsMDMUnenrollment returns true if the host must be
// unenrolled from Fleet's Windows MDM (if it MDM was disabled).
func (h *Host) IsEligibleForWindowsMDMUnenrollment(isConnectedToFleetMDM bool) bool {
	return h.FleetPlatform() == "windows" &&
		h.IsOsqueryEnrolled() &&
		isConnectedToFleetMDM
}

// HostDisplayName returns ComputerName if it isn't empty. Otherwise, it returns Hostname if it isn't
// empty. If Hostname is empty and both HardwareSerial and HardwareModel are not empty, it returns a
// composite string with HardwareModel and HardwareSerial. If all else fails, it returns an empty
// string.
func HostDisplayName(computerName string, hostname string, hardwareModel string, hardwareSerial string) string {
	switch {
	case computerName != "":
		return computerName
	case hostname != "":
		return hostname
	case hardwareModel != "" && hardwareSerial != "":
		return fmt.Sprintf("%s (%s)", hardwareModel, hardwareSerial)
	default:
		return ""
	}
}

func (h *Host) DisplayName() string {
	return HostDisplayName(h.ComputerName, h.Hostname, h.HardwareModel, h.HardwareSerial)
}

type HostIssues struct {
	FailingPoliciesCount         uint64  `json:"failing_policies_count" db:"failing_policies_count" csv:"-"`
	CriticalVulnerabilitiesCount *uint64 `json:"critical_vulnerabilities_count,omitempty" db:"critical_vulnerabilities_count" csv:"-"` // We set it to nil if the license is not premium
	TotalIssuesCount             uint64  `json:"total_issues_count" db:"total_issues_count" csv:"issues"`                              // when exporting in CSV, we want that value as the "issues" column
}

func (h Host) AuthzType() string {
	return "host"
}

// HostDetail provides the full host metadata along with associated labels and
// packs. It also includes policies, batteries, maintenance window, and MDM profiles, as applicable.
type HostDetail struct {
	Host
	// Labels is the list of labels the host is a member of.
	Labels []*Label `json:"labels"`
	// Packs is the list of packs the host is a member of.
	Packs []*Pack `json:"packs"`
	// Batteries is the list of batteries for the host. It is a pointer to a
	// slice so that when set, it gets marhsaled even if the slice is empty,
	// but when unset, it doesn't get marshaled (e.g. we don't return that
	// information for the List Hosts endpoint).
	Batteries *[]*HostBattery `json:"batteries,omitempty"`

	// MaintenanceWindow contains the host user's calendar IANA timezone and the start time of the next scheduled maintenance window.
	MaintenanceWindow *HostMaintenanceWindow `json:"maintenance_window,omitempty"`
}

type HostMaintenanceWindow struct {
	//  StartsAt is the start time of the future maintenance window, retrieved from calendar_events,
	//  represented as a time.Time in the host's associated google calendar user's timezone, which is represented as a time.Location
	StartsAt time.Time `json:"starts_at" db:"start_time"`
	// TimeZone is the IANA timezone of the user's google calendar, retrieved from calendar_events
	TimeZone *string `json:"timezone" db:"timezone"`
}

const (
	HostKind = "host"
)

// HostSummary is a structure which represents a data summary about the total
// set of hosts in the database. This structure is returned by the HostService
// method GetHostSummary
type HostSummary struct {
	TeamID             *uint                  `json:"team_id,omitempty" db:"-"`
	TotalsHostsCount   uint                   `json:"totals_hosts_count" db:"total"`
	OnlineCount        uint                   `json:"online_count" db:"online"`
	OfflineCount       uint                   `json:"offline_count" db:"offline"`
	MIACount           uint                   `json:"mia_count" db:"mia"`
	Missing30DaysCount uint                   `json:"missing_30_days_count" db:"missing_30_days_count"`
	NewCount           uint                   `json:"new_count" db:"new"`
	AllLinuxCount      uint                   `json:"all_linux_count" db:"-"`
	LowDiskSpaceCount  *uint                  `json:"low_disk_space_count,omitempty" db:"low_disk_space"`
	BuiltinLabels      []*LabelSummary        `json:"builtin_labels" db:"-"`
	Platforms          []*HostSummaryPlatform `json:"platforms" db:"-"`
}

// HostSummaryPlatform represents the hosts statistics for a given platform,
// as returned inside the HostSummary struct by the GetHostSummary service.
type HostSummaryPlatform struct {
	Platform   string `json:"platform" db:"platform"`
	HostsCount uint   `json:"hosts_count" db:"total"`
}

// Status calculates the online status of the host
func (h *Host) Status(now time.Time) HostStatus {
	// The logic in this function should remain synchronized with
	// GenerateHostStatusStatistics and CountHostsInTargets
	// NOTE: As of Fleet 4.15 StatusMIA is deprecated and will be removed in Fleet 5.0

	onlineInterval := h.ConfigTLSRefresh
	if h.DistributedInterval < h.ConfigTLSRefresh {
		onlineInterval = h.DistributedInterval
	}

	// Add a small buffer to prevent flapping
	onlineInterval += OnlineIntervalBuffer

	switch {
	case h.SeenTime.Add(time.Duration(onlineInterval) * time.Second).Before(now): //nolint:gosec // dismiss G115
		return StatusOffline
	default:
		return StatusOnline
	}
}

func (h *Host) IsNew(now time.Time) bool {
	withDuration := h.CreatedAt.Add(NewDuration)
	if withDuration.After(now) ||
		withDuration.Equal(now) {
		return true
	}
	return false
}

// FleetPlatform returns the host's generic platform as supported by Fleet.
func (h *Host) FleetPlatform() string {
	return PlatformFromHost(h.Platform)
}

// SupportsOsquery returns whether the device runs osquery.
func (h *Host) SupportsOsquery() bool {
	return h.Platform != "ios" && h.Platform != "ipados"
}

// HostLinuxOSs are the possible linux values for Host.Platform.
var HostLinuxOSs = []string{
	"linux", "ubuntu", "debian", "rhel", "centos", "sles", "kali", "gentoo", "amzn", "pop", "arch", "linuxmint", "void", "nixos", "endeavouros", "manjaro", "opensuse-leap", "opensuse-tumbleweed", "tuxedo",
}

func IsLinux(hostPlatform string) bool {
	for _, linuxPlatform := range HostLinuxOSs {
		if linuxPlatform == hostPlatform {
			return true
		}
	}
	return false
}

func IsUnixLike(hostPlatform string) bool {
	unixLikeOSs := HostLinuxOSs
	unixLikeOSs = append(unixLikeOSs, "darwin")
	for _, p := range unixLikeOSs {
		if p == hostPlatform {
			return true
		}
	}
	return false
}

// PlatformFromHost converts the given host platform into
// the generic platforms known by osquery
// https://osquery.readthedocs.io/en/stable/deployment/configuration/
// and supported by Fleet.
//
// Returns empty string if hostPlatform is unknnown.
func PlatformFromHost(hostPlatform string) string {
	switch {
	case IsLinux(hostPlatform):
		return "linux"
	case hostPlatform == "darwin", hostPlatform == "windows",
		// Some customers have custom agents that support ChromeOS
		// TODO remove this once that customer migrates to Fleetd for Chrome
		hostPlatform == "CrOS",
		// Fleet now supports Chrome via fleetd
		hostPlatform == "chrome",
		hostPlatform == "ios",
		hostPlatform == "ipados":
		return hostPlatform
	default:
		return ""
	}
}

// ExpandPlatform returns the list of platforms corresponding to the (possibly
// generic) platform provided. For example, "linux" expands to all the platform
// identifiers considered to be linux, while "debian" returns only "debian",
// "windows" => "windows", etc.
func ExpandPlatform(platform string) []string {
	switch platform {
	case "linux":
		// return a copy to make sure the caller cannot modify the slice
		linuxOSs := make([]string, len(HostLinuxOSs))
		copy(linuxOSs, HostLinuxOSs)
		return linuxOSs
	default:
		return []string{platform}
	}
}

// List of valid sources for HostDeviceMapping (host_emails table in the
// database).
const (
	DeviceMappingGoogleChromeProfiles = "google_chrome_profiles"
	DeviceMappingMDMIdpAccounts       = "mdm_idp_accounts"
	DeviceMappingCustomInstaller      = "custom_installer" // set by fleetd via device-authenticated API
	DeviceMappingCustomOverride       = "custom_override"  // set by user via user-authenticated API

	DeviceMappingCustomPrefix      = "custom_" // if host_emails.source starts with this, replace with DeviceMappingCustomReplacement
	DeviceMappingCustomReplacement = "custom"  // replaces a source that starts with CustomPrefix - in the UI, we want to display those as only "custom"
)

// HostDeviceMapping represents a mapping of a user email address to a host,
// as reported by the specified source (e.g. Google Chrome Profiles).
type HostDeviceMapping struct {
	ID     uint   `json:"-" db:"id"`
	HostID uint   `json:"-" db:"host_id"`
	Email  string `json:"email" db:"email"`
	Source string `json:"source" db:"source"`
}

type HostMunkiInfo struct {
	Version string `json:"version"`
}

// HostMDM represents a host_mdm row, with information about the MDM solution
// used by a host. Note that it uses a different JSON representation than its
// struct - it implements a custom JSON marshaler.
type HostMDM struct {
	HostID                 uint    `db:"host_id" json:"-" csv:"-"`
	Enrolled               bool    `db:"enrolled" json:"-" csv:"-"`
	ServerURL              string  `db:"server_url" json:"-" csv:"-"`
	InstalledFromDep       bool    `db:"installed_from_dep" json:"-" csv:"-"`
	IsServer               bool    `db:"is_server" json:"-" csv:"-"`
	MDMID                  *uint   `db:"mdm_id" json:"-" csv:"-"`
	Name                   string  `db:"name" json:"-" csv:"-"`
	DEPProfileAssignStatus *string `db:"dep_profile_assign_status" json:"-" csv:"-"`
}

// HasJSONProfileAssigned returns true if Fleet has assigned an ADE/DEP JSON
// profile to the host, and it'll be enrolled into Fleet the next time the host
// performs automatic enrollment.
func (h *HostMDM) HasJSONProfileAssigned() bool {
	// TODO: get rid of h != nil with a better solution once we stablish
	// the pattern for dealing with a nil HostMDM
	return h != nil &&
		h.DEPProfileAssignStatus != nil &&
		*h.DEPProfileAssignStatus == string(DEPAssignProfileResponseSuccess)
}

// HostMunkiIssue represents a single munki issue for a host.
type HostMunkiIssue struct {
	MunkiIssueID       uint      `db:"munki_issue_id" json:"id"`
	Name               string    `db:"name" json:"name"`
	IssueType          string    `db:"issue_type" json:"type"`
	HostIssueCreatedAt time.Time `db:"created_at" json:"created_at"`
}

// List of well-known MDM solution names. Those correspond to names stored in
// the mobile_device_management_solutions table.
const (
	UnknownMDMName        = ""
	WellKnownMDMKandji    = "Kandji"
	WellKnownMDMJamf      = "Jamf"
	WellKnownMDMJumpCloud = "JumpCloud"
	WellKnownMDMVMWare    = "VMware Workspace ONE"
	WellKnownMDMIntune    = "Intune"
	WellKnownMDMSimpleMDM = "SimpleMDM"
	WellKnownMDMFleet     = "Fleet"
)

var mdmNameFromServerURLChecks = map[string]string{
	"kandji":    WellKnownMDMKandji,
	"jamf":      WellKnownMDMJamf,
	"jumpcloud": WellKnownMDMJumpCloud,
	"airwatch":  WellKnownMDMVMWare,
	"awmdm":     WellKnownMDMVMWare,
	"microsoft": WellKnownMDMIntune,
	"simplemdm": WellKnownMDMSimpleMDM,
	"fleetdm":   WellKnownMDMFleet,
}

// MDMNameFromServerURL returns the MDM solution name corresponding to the
// given server URL. If no match is found, it returns the unknown MDM name.
func MDMNameFromServerURL(serverURL string) string {
	serverURL = strings.ToLower(serverURL)

	for check, name := range mdmNameFromServerURLChecks {
		if strings.Contains(serverURL, check) {
			return name
		}
	}
	return UnknownMDMName
}

func (h *HostMDM) EnrollmentStatus() string {
	switch {
	case h.Enrolled && !h.InstalledFromDep:
		return "On (manual)"
	case h.Enrolled && h.InstalledFromDep:
		return "On (automatic)"
	case !h.Enrolled && h.InstalledFromDep:
		return "Pending"
	default:
		return "Off"
	}
}

func (h *HostMDM) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	if h.IsServer {
		return []byte("null"), nil
	}
	var jsonMDM struct {
		EnrollmentStatus string `json:"enrollment_status"`
		ServerURL        string `json:"server_url"`
		Name             string `json:"name,omitempty"`
		ID               *uint  `json:"id,omitempty"`
	}

	jsonMDM.ServerURL = h.ServerURL
	jsonMDM.EnrollmentStatus = h.EnrollmentStatus()
	jsonMDM.Name = h.Name
	jsonMDM.ID = h.MDMID
	return json.Marshal(jsonMDM)
}

func (h *HostMDM) UnmarshalJSON(b []byte) error {
	// fail attempts to unmarshal in this struct, to prevent using e.g.
	// getMacadminsDataResponse in tests, as it can't unmarshal in a meaningful
	// way.
	return errors.New("JSON unmarshaling is not supported for HostMDM")
}

// HostBattery represents a host's battery, as reported by the osquery battery
// table.
type HostBattery struct {
	ID           uint   `json:"-" db:"id"`
	HostID       uint   `json:"-" db:"host_id"`
	SerialNumber string `json:"-" db:"serial_number"`
	CycleCount   int    `json:"cycle_count" db:"cycle_count"`
	Health       string `json:"health" db:"health"`
}

type MacadminsData struct {
	Munki       *HostMunkiInfo    `json:"munki"`
	MDM         *HostMDM          `json:"mobile_device_management"`
	MunkiIssues []*HostMunkiIssue `json:"munki_issues"`
}

type AggregatedMunkiVersion struct {
	HostMunkiInfo
	HostsCount int `json:"hosts_count" db:"hosts_count"`
}

// MunkiIssue represents a single munki issue, as returned by the list hosts
// endpoint when a muniki issue ID is provided as filter.
type MunkiIssue struct {
	ID        uint   `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	IssueType string `json:"type" db:"issue_type"`
}

type AggregatedMunkiIssue struct {
	MunkiIssue
	HostsCount int `json:"hosts_count" db:"hosts_count"`
}

type AggregatedMDMStatus struct {
	EnrolledManualHostsCount    int `json:"enrolled_manual_hosts_count" db:"enrolled_manual_hosts_count"`
	EnrolledAutomatedHostsCount int `json:"enrolled_automated_hosts_count" db:"enrolled_automated_hosts_count"`
	PendingHostsCount           int `json:"pending_hosts_count" db:"pending_hosts_count"`
	UnenrolledHostsCount        int `json:"unenrolled_hosts_count" db:"unenrolled_hosts_count"`
	HostsCount                  int `json:"hosts_count" db:"hosts_count"`
}

// AggregatedMDMData contains aggregated data from mdm installations.
type AggregatedMDMData struct {
	CountsUpdatedAt time.Time                `json:"counts_updated_at"`
	MDMStatus       AggregatedMDMStatus      `json:"mobile_device_management_enrollment_status"`
	MDMSolutions    []AggregatedMDMSolutions `json:"mobile_device_management_solution"`
}

// MDMSolution represents a single MDM solution, as returned by the list hosts
// endpoint when an MDM Solution ID is provided as filter.
type MDMSolution struct {
	ID        uint   `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	ServerURL string `json:"server_url" db:"server_url"`
}

type AggregatedMDMSolutions struct {
	MDMSolution
	HostsCount int `json:"hosts_count" db:"hosts_count"`
}

type AggregatedMacadminsData struct {
	CountsUpdatedAt time.Time                `json:"counts_updated_at"`
	MunkiVersions   []AggregatedMunkiVersion `json:"munki_versions"`
	MunkiIssues     []AggregatedMunkiIssue   `json:"munki_issues"`
	MDMStatus       AggregatedMDMStatus      `json:"mobile_device_management_enrollment_status"`
	MDMSolutions    []AggregatedMDMSolutions `json:"mobile_device_management_solution"`
}

// HostVulnerabilitySummary type used with webhooks and third-party vulnerability automations.
// Contains all pertinent host info plus the installed paths of all affected software.
type HostVulnerabilitySummary struct {
	// ID Is the ID of the host
	ID uint `json:"id" db:"id"`
	// Hostname the host's hostname
	Hostname string `json:"hostname" db:"hostname"`
	// DisplayName either the 'computer_name' or the 'host_name' (whatever is not empty)
	DisplayName string `json:"display_name" db:"display_name"`
	// SoftwareInstalledPaths paths of vulnerable software installed on the host.
	SoftwareInstalledPaths []string `json:"software_installed_paths,omitempty" db:"software_installed_paths"`
}

func (hvs *HostVulnerabilitySummary) AddSoftwareInstalledPath(p string) {
	if p != "" {
		hvs.SoftwareInstalledPaths = append(hvs.SoftwareInstalledPaths, p)
	}
}

type OSVersions struct {
	CountsUpdatedAt time.Time   `json:"counts_updated_at"`
	OSVersions      []OSVersion `json:"os_versions"`
}

type VulnerableOS struct {
	OSVersion
	ResolvedInVersion *string `json:"resolved_in_version"`
}

type OSVersion struct {
	// ID is the unique id of the operating system.
	ID uint `json:"id,omitempty"`
	// OSVersionID is a uniqe NameOnly/Version combination for the operating system.
	OSVersionID uint `json:"os_version_id"`
	// HostsCount is the number of hosts that have reported the operating system.
	HostsCount int `json:"hosts_count"`
	// Name is the name and alphanumeric version of the operating system. e.g., "Microsoft Windows 11 Enterprise",
	// "Ubuntu", or "macOS". NOTE: In Fleet 5.0, this field will no longer include the alphanumeric version.
	Name string `json:"name"`
	// NameOnly is the name of the operating system, e.g., "Microsoft Windows 11 Enterprise",
	// "Ubuntu", or "macOS". NOTE: In Fleet 5.0, this field be removed.
	NameOnly string `json:"name_only"`
	// Version is the alphanumeric version of the operating system, e.g., "21H2", "20.4.0", or "12.5".
	Version string `json:"version"`
	// Platform is the platform of the operating system, e.g., "windows", "ubuntu", or "darwin".
	Platform string `json:"platform"`
	// GeneratedCPE is the Common Platform Enumeration (CPE) name for the operating system.
	// It is currently only generated for Operating Systems scanned for vulnerabilities
	// in NVD (macOS only)
	GeneratedCPEs []string `json:"generated_cpes,omitempty"`
	// Vulnerabilities are the vulnerabilities associated with the operating system.
	Vulnerabilities Vulnerabilities `json:"vulnerabilities"`
}

type HostDetailOptions struct {
	IncludeCVEScores                    bool
	IncludeCriticalVulnerabilitiesCount bool
	IncludePolicies                     bool
	ExcludeSoftware                     bool
}

// EnrollHostLimiter defines the methods to support enforcement of enrolled
// hosts limit, as defined by the user's license.
type EnrollHostLimiter interface {
	CanEnrollNewHost(ctx context.Context) (ok bool, err error)
	SyncEnrolledHostIDs(ctx context.Context) error
}

type HostMDMCheckinInfo struct {
	HardwareSerial     string `json:"hardware_serial" db:"hardware_serial"`
	InstalledFromDEP   bool   `json:"installed_from_dep" db:"installed_from_dep"`
	DisplayName        string `json:"display_name" db:"display_name"`
	TeamID             uint   `json:"team_id" db:"team_id"`
	DEPAssignedToFleet bool   `json:"dep_assigned_to_fleet" db:"dep_assigned_to_fleet"`
	OsqueryEnrolled    bool   `json:"osquery_enrolled" db:"osquery_enrolled"`

	SCEPRenewalInProgress bool   `json:"-" db:"scep_renewal_in_progress"`
	Platform              string `json:"-" db:"platform"`
}

type HostDiskEncryptionKey struct {
	HostID          uint      `json:"-" db:"host_id"`
	Base64Encrypted string    `json:"-" db:"base64_encrypted"`
	Decryptable     *bool     `json:"-" db:"decryptable"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	DecryptedValue  string    `json:"key" db:"-"`
	ClientError     string    `json:"-" db:"client_error"`
}

// HostSoftwareInstalledPath represents where in the file system a software on a host was installed
type HostSoftwareInstalledPath struct {
	// ID row id
	ID uint `db:"id"`
	// HostID is the id of the host where the software in question is installed
	HostID uint `db:"host_id"`
	// SoftwareID is the id of the software
	SoftwareID uint `db:"software_id"`
	// InstalledPath is the file system path where the software is installed
	InstalledPath string `db:"installed_path"`
	// TeamIdentifier (not to be confused with Fleet's team IDs) is the Apple's "Team ID" (aka "Developer ID"
	// or "Signing ID") of signed applications, see https://developer.apple.com/help/account/manage-your-team/locate-your-team-id.
	TeamIdentifier string `db:"team_identifier"`
}

// HostMacOSProfile represents a macOS profile installed on a host as reported by the macos_profiles
// table of the macadmins oquery extension
type HostMacOSProfile struct {
	// DisplayName is the display name of the profile.
	DisplayName string `json:"display_name" db:"display_name"`
	// Identifier is the identifier of the profile.
	Identifier string `json:"identifier" db:"identifier"`
	// InstallDate is the date the profile was installed on the host as reported by the host's clock.
	InstallDate time.Time `json:"install_date" db:"install_date"`
}

// HostLite contains a subset of Host fields.
type HostLite struct {
	ID                  uint      `db:"id"`
	TeamID              *uint     `db:"team_id"`
	Hostname            string    `db:"hostname"`
	OsqueryHostID       string    `db:"osquery_host_id"`
	NodeKey             string    `db:"node_key"`
	UUID                string    `db:"uuid"`
	HardwareSerial      string    `db:"hardware_serial"`
	SeenTime            time.Time `db:"seen_time"`
	DistributedInterval uint      `db:"distributed_interval"`
	ConfigTLSRefresh    uint      `db:"config_tls_refresh"`
}

// IsEligibleForDEPMigration returns true if the host fulfills all requirements
// for DEP migration from a third-party provider into Fleet.
func IsEligibleForDEPMigration(host *Host, mdmInfo *HostMDM, isConnectedToFleetMDM bool) bool {
	return mdmInfo != nil &&
		host.IsOsqueryEnrolled() &&
		host.IsDEPAssignedToFleet() &&
		mdmInfo.HasJSONProfileAssigned() &&
		mdmInfo.Enrolled &&
		// as a special case for migration with user interaction, we
		// also check the information stored in host_mdm, and assume
		// the host needs migration if it's not Fleet
		//
		// this is because we can't always rely on nano setting
		// `nano_enrollment.active = 1` since sometimes Fleet won't get
		// the checkout message from the host.
		(!isConnectedToFleetMDM || mdmInfo.Name != WellKnownMDMFleet)
}

var macOSADEMigrationOnlyLastVersion = semver.MustParse("14")

// IsEligibleForManualMigration returns true if the host is manually enrolled into a 3rd party MDM
// and is able to migrate to Fleet.
func IsEligibleForManualMigration(host *Host, mdmInfo *HostMDM, isConnectedToFleetMDM bool) (bool, error) {
	goodVersion, err := IsMacOSMajorVersionOK(host)
	if err != nil {
		return false, fmt.Errorf("checking macOS version for manual migration eligibility: %w", err)
	}

	return goodVersion &&
		host.IsOsqueryEnrolled() &&
		!host.IsDEPAssignedToFleet() &&
		mdmInfo != nil &&
		!mdmInfo.InstalledFromDep &&
		!mdmInfo.HasJSONProfileAssigned() &&
		mdmInfo.Enrolled &&
		(!isConnectedToFleetMDM || mdmInfo.Name != WellKnownMDMFleet), nil
}

func IsMacOSMajorVersionOK(host *Host) (bool, error) {
	if host == nil {
		return false, nil
	}

	parts := strings.Split(host.OSVersion, " ")

	if len(parts) < 2 || parts[0] != "macOS" {
		return false, nil
	}

	version, err := semver.NewVersion(parts[1])
	if err != nil {
		return false, fmt.Errorf("parsing macOS version \"%s\": %w", parts[1], err)
	}

	if version.GreaterThan(macOSADEMigrationOnlyLastVersion) {
		return true, nil
	}

	return false, nil
}
