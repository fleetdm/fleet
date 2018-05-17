package kolide

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net"
	"strings"
	"time"
)

const (
	// StatusOnline host is active.
	StatusOnline = "online"

	// StatusOffline no communication with host for OfflineDuration.
	StatusOffline = "offline"

	// StatusMIA no communication with host for MIADuration.
	StatusMIA = "mia"

	// NewDuration if a host has been created within this time period it's
	// considered new.
	NewDuration = 24 * time.Hour

	// OfflineDuration if a host hasn't been in communition for this
	// period it is considered MIA.
	MIADuration = 30 * 24 * time.Hour

	// OnlineIntervalBuffer is the additional time in seconds to add to the
	// online interval to avoid flapping of hosts that check in a bit later
	// than their expected checkin interval.
	OnlineIntervalBuffer = 30
)

type HostStore interface {
	NewHost(host *Host) (*Host, error)
	SaveHost(host *Host) error
	DeleteHost(hid uint) error
	Host(id uint) (*Host, error)
	ListHosts(opt ListOptions) ([]*Host, error)
	EnrollHost(osqueryHostId string, nodeKeySize int) (*Host, error)
	AuthenticateHost(nodeKey string) (*Host, error)
	MarkHostSeen(host *Host, t time.Time) error
	SearchHosts(query string, omit ...uint) ([]*Host, error)
	// GenerateHostStatusStatistics retrieves the count of online, offline,
	// MIA and new hosts.
	GenerateHostStatusStatistics(now time.Time) (online, offline, mia, new uint, err error)
	// DistributedQueriesForHost retrieves the distributed queries that the
	// given host should run. The result map is a mapping from campaign ID
	// to query text.
	DistributedQueriesForHost(host *Host) (map[uint]string, error)
}

type HostService interface {
	ListHosts(ctx context.Context, opt ListOptions) (hosts []*Host, err error)
	GetHost(ctx context.Context, id uint) (host *Host, err error)
	GetHostSummary(ctx context.Context) (summary *HostSummary, err error)
	DeleteHost(ctx context.Context, id uint) (err error)
}

type Host struct {
	UpdateCreateTimestamps
	DeleteFields
	ID uint `json:"id"`
	// OsqueryHostID is the key used in the request context that is
	// used to retrieve host information.  It is sent from osquery and may currently be
	// a GUID or a Host Name, but in either case, it MUST be unique
	OsqueryHostID    string        `json:"-" db:"osquery_host_id"`
	DetailUpdateTime time.Time     `json:"detail_updated_at" db:"detail_update_time"` // Time that the host details were last updated
	SeenTime         time.Time     `json:"seen_time" db:"seen_time"`                  // Time that the host was last "seen"
	NodeKey          string        `json:"-" db:"node_key"`
	HostName         string        `json:"hostname" db:"host_name"` // there is a fulltext index on this field
	UUID             string        `json:"uuid"`
	Platform         string        `json:"platform"`
	OsqueryVersion   string        `json:"osquery_version" db:"osquery_version"`
	OSVersion        string        `json:"os_version" db:"os_version"`
	Build            string        `json:"build"`
	PlatformLike     string        `json:"platform_like" db:"platform_like"`
	CodeName         string        `json:"code_name" db:"code_name"`
	Uptime           time.Duration `json:"uptime"`
	PhysicalMemory   int           `json:"memory" sql:"type:bigint" db:"physical_memory"`
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
	NetworkInterfaces         []*NetworkInterface `json:"network_interfaces" db:"-"`
	DistributedInterval       uint                `json:"distributed_interval" db:"distributed_interval"`
	ConfigTLSRefresh          uint                `json:"config_tls_refresh" db:"config_tls_refresh"`
	LoggerTLSPeriod           uint                `json:"logger_tls_period" db:"logger_tls_period"`
}

// HostSummary is a structure which represents a data summary about the total
// set of hosts in the database. This structure is returned by the HostService
// method GetHostSummary
type HostSummary struct {
	OnlineCount  uint `json:"online_count"`
	OfflineCount uint `json:"offline_count"`
	MIACount     uint `json:"mia_count"`
	NewCount     uint `json:"new_count"`
}

// ResetPrimaryNetwork will determine if the PrimaryNetworkInterfaceID
// needs to change.  If it has not been set, it will default to the interface
// with the most IO.  If it doesn't match an existing nic (as in the nic got changed)
// is will be reset.  If there are not any nics, it will be set to nil.  In any
// case if it changes, this function will return true, indicating that the
// change should be written back to the database
func (h *Host) ResetPrimaryNetwork() bool {
	if h.PrimaryNetworkInterfaceID != nil {
		// This *may* happen if other details of the host are fetched before network_interfaces
		if len(h.NetworkInterfaces) == 0 {
			h.PrimaryNetworkInterfaceID = nil
			return true
		}
		for _, nic := range h.NetworkInterfaces {
			if *h.PrimaryNetworkInterfaceID == nic.ID {
				return false
			}
		}
		h.PrimaryNetworkInterfaceID = nil
	}

	// nics are in descending order of IO
	// so we default to the most active nic
	if len(h.NetworkInterfaces) > 0 {
		// Check IPv4 address
		for _, nic := range h.NetworkInterfaces {
			if strings.Index(nic.IPAddress, "127.") == 0 {
				continue
			}
			var isIpAddress = net.ParseIP(nic.IPAddress)
			if isIpAddress.To4() != nil {
				h.PrimaryNetworkInterfaceID = &nic.ID
				return true
			}
		}
		// return IPv6 or other nic in place of IPv4
		h.PrimaryNetworkInterfaceID = &h.NetworkInterfaces[0].ID
		return true
	}

	return false

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
func (h *Host) Status(now time.Time) string {
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
