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

func (l ListOptions) Empty() bool {
	return l.Page == 0 && l.PerPage == 0 && l.OrderKey == "" && l.OrderDirection == 0 && l.MatchQuery == ""
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
	ID uint `json:"id"`
	// OsqueryHostID is the key used in the request context that is
	// used to retrieve host information.  It is sent from osquery and may currently be
	// a GUID or a Host Name, but in either case, it MUST be unique
	OsqueryHostID    string    `json:"-" db:"osquery_host_id"`
	DetailUpdatedAt  time.Time `json:"detail_updated_at" db:"detail_updated_at"` // Time that the host details were last updated
	LabelUpdatedAt   time.Time `json:"label_updated_at" db:"label_updated_at"`   // Time that the host labels were last updated
	PolicyUpdatedAt  time.Time `json:"policy_updated_at" db:"policy_updated_at"` // Time that the host policies were last updated
	LastEnrolledAt   time.Time `json:"last_enrolled_at" db:"last_enrolled_at"`   // Time that the host last enrolled
	SeenTime         time.Time `json:"seen_time" db:"seen_time"`                 // Time that the host was last "seen"
	RefetchRequested bool      `json:"refetch_requested" db:"refetch_requested"`
	NodeKey          string    `json:"-" db:"node_key"`
	Hostname         string    `json:"hostname" db:"hostname"` // there is a fulltext index on this field
	UUID             string    `json:"uuid" db:"uuid"`         // there is a fulltext index on this field
	// Platform is the host's platform as defined by osquery's os_version.platform.
	Platform       string        `json:"platform"`
	OsqueryVersion string        `json:"osquery_version" db:"osquery_version"`
	OSVersion      string        `json:"os_version" db:"os_version"`
	Build          string        `json:"build"`
	PlatformLike   string        `json:"platform_like" db:"platform_like"`
	CodeName       string        `json:"code_name" db:"code_name"`
	Uptime         time.Duration `json:"uptime"`
	Memory         int64         `json:"memory" sql:"type:bigint" db:"memory"`
	// system_info fields
	CPUType          string `json:"cpu_type" db:"cpu_type"`
	CPUSubtype       string `json:"cpu_subtype" db:"cpu_subtype"`
	CPUBrand         string `json:"cpu_brand" db:"cpu_brand"`
	CPUPhysicalCores int    `json:"cpu_physical_cores" db:"cpu_physical_cores"`
	CPULogicalCores  int    `json:"cpu_logical_cores" db:"cpu_logical_cores"`
	HardwareVendor   string `json:"hardware_vendor" db:"hardware_vendor"`
	HardwareModel    string `json:"hardware_model" db:"hardware_model"`
	HardwareVersion  string `json:"hardware_version" db:"hardware_version"`
	HardwareSerial   string `json:"hardware_serial" db:"hardware_serial"`
	ComputerName     string `json:"computer_name" db:"computer_name"`
	// PrimaryNetworkInterfaceID if present indicates to primary network for the host, the details of which
	// can be found in the NetworkInterfaces element with the same ip_address.
	PrimaryNetworkInterfaceID *uint               `json:"primary_ip_id,omitempty" db:"primary_ip_id"`
	NetworkInterfaces         []*NetworkInterface `json:"-" db:"-"`
	PrimaryIP                 string              `json:"primary_ip" db:"primary_ip"`
	PrimaryMac                string              `json:"primary_mac" db:"primary_mac"`
	DistributedInterval       uint                `json:"distributed_interval" db:"distributed_interval"`
	ConfigTLSRefresh          uint                `json:"config_tls_refresh" db:"config_tls_refresh"`
	LoggerTLSPeriod           uint                `json:"logger_tls_period" db:"logger_tls_period"`
	TeamID                    *uint               `json:"team_id" db:"team_id"`

	// Loaded via JOIN in DB
	PackStats []PackStats `json:"pack_stats"`
	// TeamName is the name of the team, loaded by JOIN to the teams table.
	TeamName *string `json:"team_name" db:"team_name"`
	// Additional is the additional information from the host
	// additional_queries. This should be stored in a separate DB table.
	Additional *json.RawMessage `json:"additional,omitempty" db:"additional"`

	// Users currently in the host
	Users []HostUser `json:"users,omitempty"`

	GigsDiskSpaceAvailable    float64 `json:"gigs_disk_space_available" db:"gigs_disk_space_available"`
	PercentDiskSpaceAvailable float64 `json:"percent_disk_space_available" db:"percent_disk_space_available"`

	HostIssues `json:"issues,omitempty"`

	Modified bool `json:"-"`
}

type HostIssues struct {
	TotalIssuesCount     int `json:"total_issues_count" db:"total_issues_count"`
	FailingPoliciesCount int `json:"failing_policies_count" db:"failing_policies_count"`
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
	"linux", "ubuntu", "debian", "rhel", "centos", "sles", "kali", "gentoo",
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
