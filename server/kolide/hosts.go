package kolide

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	OnlineIntervalBuffer = 30
)

type HostStore interface {
	// NewHost is deprecated and will be removed. Hosts should always be
	// enrolled via EnrollHost.
	NewHost(host *Host) (*Host, error)
	SaveHost(host *Host) error
	DeleteHost(hid uint) error
	Host(id uint) (*Host, error)
	// EnrollHost will enroll a new host with the given identifier, setting the
	// node key, and enroll secret used for the host. Implementations of this
	// method should respect the provided host enrollment cooldown, by returning
	// an error if the host has enrolled within the cooldown period.
	EnrollHost(osqueryHostId, nodeKey, secretName string, cooldown time.Duration) (*Host, error)
	ListHosts(opt HostListOptions) ([]*Host, error)
	// AuthenticateHost authenticates and returns host metadata by node key.
	// This method should not return the host "additional" information as this
	// is not typically necessary for the operations performed by the osquery
	// endpoints.
	AuthenticateHost(nodeKey string) (*Host, error)
	MarkHostSeen(host *Host, t time.Time) error
	MarkHostsSeen(hostIDs []uint, t time.Time) error
	SearchHosts(query string, omit ...uint) ([]*Host, error)
	// CleanupIncomingHosts deletes hosts that have enrolled but never
	// updated their status details. This clears dead "incoming hosts" that
	// never complete their registration.
	//
	// A host is considered incoming if both the hostname and
	// osquery_version fields are empty. This means that multiple different
	// osquery queries failed to populate details.
	CleanupIncomingHosts(now time.Time) error
	// GenerateHostStatusStatistics retrieves the count of online, offline,
	// MIA and new hosts.
	GenerateHostStatusStatistics(now time.Time) (online, offline, mia, new uint, err error)
	// HostIDsByName Retrieve the IDs associated with the given hostnames
	HostIDsByName(hostnames []string) ([]uint, error)
	// HostByIdentifier returns one host matching the provided identifier.
	// Possible matches can be on osquery_host_identifier, node_key, UUID, or
	// hostname.
	HostByIdentifier(identifier string) (*Host, error)
}

type HostService interface {
	ListHosts(ctx context.Context, opt HostListOptions) (hosts []*Host, err error)
	GetHost(ctx context.Context, id uint) (host *HostDetail, err error)
	GetHostSummary(ctx context.Context) (summary *HostSummary, err error)
	DeleteHost(ctx context.Context, id uint) (err error)
	// HostByIdentifier returns one host matching the provided identifier.
	// Possible matches can be on osquery_host_identifier, node_key, UUID, or
	// hostname.
	HostByIdentifier(ctx context.Context, identifier string) (*HostDetail, error)

	FlushSeenHosts(ctx context.Context) error
}

type HostListOptions struct {
	ListOptions

	// AdditionalFilters selects which host additional fields should be
	// populated.
	AdditionalFilters []string
	// StatusFilter selects the online status of the hosts.
	StatusFilter HostStatus
	// MatchQuery is the query string to match in various columns of the host.
	MatchQuery string
}

type Host struct {
	UpdateCreateTimestamps
	ID uint `json:"id"`
	// OsqueryHostID is the key used in the request context that is
	// used to retrieve host information.  It is sent from osquery and may currently be
	// a GUID or a Host Name, but in either case, it MUST be unique
	OsqueryHostID    string        `json:"-" db:"osquery_host_id"`
	DetailUpdateTime time.Time     `json:"detail_updated_at" db:"detail_update_time"` // Time that the host details were last updated
	LabelUpdateTime  time.Time     `json:"label_updated_at" db:"label_update_time"`   // Time that the host details were last updated
	LastEnrollTime   time.Time     `json:"last_enrolled_at" db:"last_enroll_time"`    // Time that the host last enrolled
	SeenTime         time.Time     `json:"seen_time" db:"seen_time"`                  // Time that the host was last "seen"
	NodeKey          string        `json:"-" db:"node_key"`
	HostName         string        `json:"hostname" db:"host_name"` // there is a fulltext index on this field
	UUID             string        `json:"uuid" db:"uuid"`          // there is a fulltext index on this field
	Platform         string        `json:"platform"`
	OsqueryVersion   string        `json:"osquery_version" db:"osquery_version"`
	OSVersion        string        `json:"os_version" db:"os_version"`
	Build            string        `json:"build"`
	PlatformLike     string        `json:"platform_like" db:"platform_like"`
	CodeName         string        `json:"code_name" db:"code_name"`
	Uptime           time.Duration `json:"uptime"`
	PhysicalMemory   int64         `json:"memory" sql:"type:bigint" db:"physical_memory"`
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
	Additional                *json.RawMessage    `json:"additional,omitempty" db:"additional"`
	EnrollSecretName          string              `json:"enroll_secret_name" db:"enroll_secret_name"`
}

// HostDetail provides the full host metadata along with associated labels and
// packs.
type HostDetail struct {
	Host
	// Labels is the list of labels the host is a member of.
	Labels []Label `json:"labels"`
	// Packs is the list of packs the host is a member of.
	Packs []Pack `json:"packs"`
}

const (
	HostKind = "host"
)

// HostSummary is a structure which represents a data summary about the total
// set of hosts in the database. This structure is returned by the HostService
// method GetHostSummary
type HostSummary struct {
	OnlineCount  uint `json:"online_count"`
	OfflineCount uint `json:"offline_count"`
	MIACount     uint `json:"mia_count"`
	NewCount     uint `json:"new_count"`
}

// RandomText returns a stdEncoded string of
// just what it says
func RandomText(keySize int) (string, error) {
	key := make([]byte, keySize)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
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
