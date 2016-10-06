package kolide

import (
	"time"

	"golang.org/x/net/context"
)

type HostStore interface {
	NewHost(host *Host) (*Host, error)
	SaveHost(host *Host) error
	DeleteHost(host *Host) error
	Host(id uint) (*Host, error)
	Hosts() ([]*Host, error)
	EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*Host, error)
	AuthenticateHost(nodeKey string) (*Host, error)
	MarkHostSeen(host *Host, t time.Time) error
}

type HostService interface {
	GetAllHosts(ctx context.Context) ([]*Host, error)
	GetHost(ctx context.Context, id uint) (*Host, error)
	DeleteHost(ctx context.Context, id uint) error
}

type HostPayload struct {
	NodeKey   *string
	HostName  *string
	UUID      *string
	IPAddress *string
	Platform  *string
}

type Host struct {
	ID               uint          `gorm:"primary_key" json:"id"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	DetailUpdateTime time.Time     `json:"detail_updated_at"` // Time that the host details were last updated
	NodeKey          string        `gorm:"unique_index:idx_host_unique_nodekey" json:"-"`
	HostName         string        `json:"hostname"`
	UUID             string        `gorm:"unique_index:idx_host_unique_uuid" json:"uuid"`
	Platform         string        `json:"platform"`
	OsqueryVersion   string        `json:"osquery_version"`
	OSVersion        string        `json:"os_version"`
	Uptime           time.Duration `json:"uptime"`
	PhysicalMemory   int           `sql:"type:bigint" json:"memory"`
	PrimaryMAC       string        `json:"mac"`
	PrimaryIP        string        `json:"ip"`
}
