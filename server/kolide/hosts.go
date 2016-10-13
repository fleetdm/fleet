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
	Hosts(opt ListOptions) ([]*Host, error)
	EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*Host, error)
	AuthenticateHost(nodeKey string) (*Host, error)
	MarkHostSeen(host *Host, t time.Time) error
}

type HostService interface {
	ListHosts(ctx context.Context, opt ListOptions) ([]*Host, error)
	GetHost(ctx context.Context, id uint) (*Host, error)
	HostStatus(ctx context.Context, host Host) string
	DeleteHost(ctx context.Context, id uint) error
}

type Host struct {
	ID               uint          `json:"id" gorm:"primary_key"`
	CreatedAt        time.Time     `json:"-"`
	UpdatedAt        time.Time     `json:"updated_at"`
	DetailUpdateTime time.Time     `json:"detail_updated_at"` // Time that the host details were last updated
	NodeKey          string        `json:"-" gorm:"unique_index:idx_host_unique_nodekey"`
	HostName         string        `json:"hostname"`
	UUID             string        `json:"uuid" gorm:"unique_index:idx_host_unique_uuid"`
	Platform         string        `json:"platform"`
	OsqueryVersion   string        `json:"osquery_version"`
	OSVersion        string        `json:"os_version"`
	Uptime           time.Duration `json:"uptime"`
	PhysicalMemory   int           `json:"memory" sql:"type:bigint"`
	PrimaryMAC       string        `json:"mac"`
	PrimaryIP        string        `json:"ip"`
}
