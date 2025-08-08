package fleet

import (
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
)

// OsqueryDistributedQueryResults represents the format of the results of an
// osquery distributed query.
type OsqueryDistributedQueryResults map[string][]map[string]string

// OsqueryStatus represents osquery status codes (0 = success, nonzero =
// failure)
type OsqueryStatus int

var ErrLegacyQueryPack = errors.New("legacy query pack, storage not supported")

// Stats contains the performance statistics about the execution of a specific osquery query.
type Stats struct {
	WallTimeMs uint64 `json:"wall_time_ms"`
	UserTime   uint64 `json:"user_time"`
	SystemTime uint64 `json:"system_time"`
	Memory     uint64 `json:"memory"`
}

const (
	// StatusOK is the success code returned by osquery
	StatusOK OsqueryStatus = 0
)

// QueryContent is the format of a query stanza in an osquery configuration.
type QueryContent struct {
	Query       string  `json:"query"`
	Description string  `json:"description,omitempty"`
	Interval    uint    `json:"interval"`
	Platform    *string `json:"platform,omitempty"`
	Version     *string `json:"version,omitempty"`
	Snapshot    *bool   `json:"snapshot,omitempty"`
	Removed     *bool   `json:"removed,omitempty"`
	Shard       *uint   `json:"shard,omitempty"`
	Denylist    *bool   `json:"denylist,omitempty"`
}

type PermissiveQueryContent struct {
	QueryContent
	Interval interface{} `json:"interval"`
}

// Queries is a helper which represents the format of a set of queries in a pack.
type Queries map[string]QueryContent

type PermissiveQueries map[string]PermissiveQueryContent

// PackContent is the format of an osquery query pack.
type PackContent struct {
	Platform  string   `json:"platform,omitempty"`
	Version   string   `json:"version,omitempty"`
	Shard     uint     `json:"shard,omitempty"`
	Discovery []string `json:"discovery,omitempty"`
	Queries   Queries  `json:"queries"`
}

type PermissivePackContent struct {
	Platform  string            `json:"platform,omitempty"`
	Version   string            `json:"version,omitempty"`
	Shard     uint              `json:"shard,omitempty"`
	Discovery []string          `json:"discovery,omitempty"`
	Queries   PermissiveQueries `json:"queries"`
}

// Packs is a helper which represents the format of a list of osquery query packs.
type Packs map[string]PackContent

type PermissivePacks map[string]PermissivePackContent

// DatastoreEnrollHostConfig holds the configuration for datastore Host enrollment
type DatastoreEnrollHostConfig struct {
	IsMDMEnabled   bool
	OsqueryHostID  string
	HardwareUUID   string
	HardwareSerial string
	NodeKey        string
	TeamID         *uint
	Cooldown       time.Duration
	IdentityCert   *types.HostIdentityCertificate
}

// DatastoreEnrollHostOption is a functional option for configuring datastore Host enrollment
type DatastoreEnrollHostOption func(*DatastoreEnrollHostConfig)

// WithEnrollHostMDMEnabled sets the MDM enabled flag for datastore Host enrollment
func WithEnrollHostMDMEnabled(enabled bool) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.IsMDMEnabled = enabled
	}
}

// WithEnrollHostOsqueryHostID sets the osquery host ID for datastore Host enrollment
func WithEnrollHostOsqueryHostID(osqueryHostID string) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.OsqueryHostID = osqueryHostID
	}
}

// WithEnrollHostHardwareUUID sets the hardware UUID for datastore Host enrollment
func WithEnrollHostHardwareUUID(hardwareUUID string) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.HardwareUUID = hardwareUUID
	}
}

// WithEnrollHostHardwareSerial sets the hardware serial for datastore Host enrollment
func WithEnrollHostHardwareSerial(hardwareSerial string) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.HardwareSerial = hardwareSerial
	}
}

// WithEnrollHostNodeKey sets the node key for datastore Host enrollment
func WithEnrollHostNodeKey(nodeKey string) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.NodeKey = nodeKey
	}
}

// WithEnrollHostTeamID sets the team ID for datastore Host enrollment
func WithEnrollHostTeamID(teamID *uint) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.TeamID = teamID
	}
}

// WithEnrollHostCooldown sets the cooldown duration for datastore Host enrollment
func WithEnrollHostCooldown(cooldown time.Duration) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.Cooldown = cooldown
	}
}

func WithEnrollHostIdentityCert(identityCert *types.HostIdentityCertificate) DatastoreEnrollHostOption {
	return func(c *DatastoreEnrollHostConfig) {
		c.IdentityCert = identityCert
	}
}
