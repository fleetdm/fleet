package kolide

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"golang.org/x/net/context"
)

type HostStore interface {
	NewHost(host *Host) (*Host, error)
	SaveHost(host *Host) error
	DeleteHost(host *Host) error
	Host(id uint) (*Host, error)
	ListHosts(opt ListOptions) ([]*Host, error)
	EnrollHost(uuid, hostname, platform string, nodeKeySize int) (*Host, error)
	AuthenticateHost(nodeKey string) (*Host, error)
	MarkHostSeen(host *Host, t time.Time) error
	SearchHosts(query string, omit ...uint) ([]Host, error)
	// DistributedQueriesForHost retrieves the distributed queries that the
	// given host should run. The result map is a mapping from campaign ID
	// to query text.
	DistributedQueriesForHost(host *Host) (map[uint]string, error)
}

type HostService interface {
	ListHosts(ctx context.Context, opt ListOptions) ([]*Host, error)
	GetHost(ctx context.Context, id uint) (*Host, error)
	HostStatus(ctx context.Context, host Host) string
	DeleteHost(ctx context.Context, id uint) error
}

type Host struct {
	UpdateCreateTimestamps
	DeleteFields
	ID               uint          `json:"id"`
	DetailUpdateTime time.Time     `json:"detail_updated_at" db:"detail_update_time"` // Time that the host details were last updated
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
}

// ResetPrimaryNetwork will determine if the PrimaryNetworkInterfaceID
// needs to change.  If it has not been set, it will default to the interface
// with the most IO.  If it doesn't match an existing nic (as in the nic got changed)
// is will be reset.  If there are not any nics, it will be set to nil.  In any
// case if it changes, this function will return true, indicating that the
// change should be written back to the database
func (h *Host) ResetPrimaryNetwork() bool {
	if h.PrimaryNetworkInterfaceID != nil {
		// No nic (should never happen)
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
