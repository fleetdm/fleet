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
	EnrollHost(uuid, hostname, ip, platform string, nodeKeySize int) (*Host, error)
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
	Uptime           time.Duration `json:"uptime"`
	PhysicalMemory   int           `json:"memory" sql:"type:bigint" db:"physical_memory"`
	PrimaryMAC       string        `json:"mac" db:"primary_mac"`
	PrimaryIP        string        `json:"ip" db:"primary_ip"` // there is a fulltext index on this field
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
