package fleet

import (
	"encoding/json"
	"time"
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
)

type HostListOptions struct {
	ListOptions

	// AdditionalFilters selects which host additional fields should be
	// populated.
	AdditionalFilters []string
	// StatusFilter selects the online status of the hosts.
	StatusFilter HostStatus
	// TeamFilter selects the hosts for specified team
	TeamFilter *uint

	PolicyIDFilter       *uint
	PolicyResponseFilter *bool

	SoftwareIDFilter *uint

	DisableFailingPolicies bool
}

func (h HostListOptions) Empty() bool {
	return h.ListOptions.Empty() && len(h.AdditionalFilters) == 0 && h.StatusFilter == "" && h.TeamFilter == nil && h.PolicyIDFilter == nil && h.PolicyResponseFilter == nil
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
	OsqueryHostID    string    `json:"-" db:"osquery_host_id" csv:"-"`
	DetailUpdatedAt  time.Time `json:"detail_updated_at" db:"detail_updated_at" csv:"detail_updated_at"` // Time that the host details were last updated
	LabelUpdatedAt   time.Time `json:"label_updated_at" db:"label_updated_at" csv:"label_updated_at"`    // Time that the host labels were last updated
	PolicyUpdatedAt  time.Time `json:"policy_updated_at" db:"policy_updated_at" csv:"policy_updated_at"` // Time that the host policies were last updated
	LastEnrolledAt   time.Time `json:"last_enrolled_at" db:"last_enrolled_at" csv:"last_enrolled_at"`    // Time that the host last enrolled
	SeenTime         time.Time `json:"seen_time" db:"seen_time" csv:"seen_time"`                         // Time that the host was last "seen"
	RefetchRequested bool      `json:"refetch_requested" db:"refetch_requested" csv:"refetch_requested"`
	NodeKey          string    `json:"-" db:"node_key" csv:"-"`
	Hostname         string    `json:"hostname" db:"hostname" csv:"hostname"` // there is a fulltext index on this field
	UUID             string    `json:"uuid" db:"uuid" csv:"uuid"`             // there is a fulltext index on this field
	// Platform is the host's platform as defined by osquery's os_version.platform.
	Platform       string        `json:"platform" csv:"platform"`
	OsqueryVersion string        `json:"osquery_version" db:"osquery_version" csv:"osquery_version"`
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

	HostIssues `json:"issues,omitempty" csv:"-"`

	Modified bool `json:"-" csv:"-"`
}

type HostIssues struct {
	TotalIssuesCount     int `json:"total_issues_count" db:"total_issues_count" csv:"-"`
	FailingPoliciesCount int `json:"failing_policies_count" db:"failing_policies_count" csv:"-"`
}

func (h Host) AuthzType() string {
	return "host"
}

// HostDetail provides the full host metadata along with associated labels and
// packs.
type HostDetail struct {
	Host
	// Labels is the list of labels the host is a member of.
	Labels []*Label `json:"labels"`
	// Packs is the list of packs the host is a member of.
	Packs []*Pack `json:"packs"`
	// Policies is the list of policies and whether it passes for the host
	Policies []*HostPolicy `json:"policies"`
}

const (
	HostKind = "host"
)

// HostSummary is a structure which represents a data summary about the total
// set of hosts in the database. This structure is returned by the HostService
// method GetHostSummary
type HostSummary struct {
	TeamID           *uint                  `json:"team_id,omitempty"`
	TotalsHostsCount uint                   `json:"totals_hosts_count" db:"total"`
	Platforms        []*HostSummaryPlatform `json:"platforms"`
	OnlineCount      uint                   `json:"online_count" db:"online"`
	OfflineCount     uint                   `json:"offline_count" db:"offline"`
	MIACount         uint                   `json:"mia_count" db:"mia"`
	NewCount         uint                   `json:"new_count" db:"new"`
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

	onlineInterval := h.ConfigTLSRefresh
	if h.DistributedInterval < h.ConfigTLSRefresh {
		onlineInterval = h.DistributedInterval
	}

	// Add a small buffer to prevent flapping
	onlineInterval += OnlineIntervalBuffer

	switch {
	case h.SeenTime.Add(MIADuration).Before(now):
		return StatusMIA
	case h.SeenTime.Add(time.Duration(onlineInterval) * time.Second).Before(now):
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

// HostLinuxOSs are the possible linux values for Host.Platform.
var HostLinuxOSs = []string{
	"linux", "ubuntu", "debian", "rhel", "centos", "sles", "kali", "gentoo", "amzn",
}

func isLinux(hostPlatform string) bool {
	for _, linuxPlatform := range HostLinuxOSs {
		if linuxPlatform == hostPlatform {
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
	case isLinux(hostPlatform):
		return "linux"
	case hostPlatform == "darwin", hostPlatform == "windows":
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

type HostMDM struct {
	EnrollmentStatus string `json:"enrollment_status"`
	ServerURL        string `json:"server_url"`
}

type MacadminsData struct {
	Munki *HostMunkiInfo `json:"munki"`
	MDM   *HostMDM       `json:"mobile_device_management"`
}

type AggregatedMunkiVersion struct {
	HostMunkiInfo
	HostsCount int `json:"hosts_count" db:"hosts_count"`
}

type AggregatedMDMStatus struct {
	EnrolledManualHostsCount    int `json:"enrolled_manual_hosts_count" db:"enrolled_manual_hosts_count"`
	EnrolledAutomatedHostsCount int `json:"enrolled_automated_hosts_count" db:"enrolled_automated_hosts_count"`
	UnenrolledHostsCount        int `json:"unenrolled_hosts_count" db:"unenrolled_hosts_count"`
	HostsCount                  int `json:"hosts_count" db:"hosts_count"`
}

type AggregatedMacadminsData struct {
	CountsUpdatedAt time.Time                `json:"counts_updated_at"`
	MunkiVersions   []AggregatedMunkiVersion `json:"munki_versions"`
	MDMStatus       AggregatedMDMStatus      `json:"mobile_device_management_enrollment_status"`
}

// HostShort is a minimal host representation returned when querying hosts.
type HostShort struct {
	ID       uint   `json:"id" db:"id"`
	Hostname string `json:"hostname" db:"hostname"`
}

type OSVersions struct {
	CountsUpdatedAt time.Time   `json:"counts_updated_at"`
	OSVersions      []OSVersion `json:"os_versions"`
}

type OSVersion struct {
	HostsCount int    `json:"hosts_count"`
	Name       string `json:"name"`
	Platform   string `json:"platform"`
}
