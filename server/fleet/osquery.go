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

// DatastoreEnrollOsqueryConfig holds the configuration for datastore Host enrollment
type DatastoreEnrollOsqueryConfig struct {
	IsMDMEnabled     bool
	OsqueryHostID    string
	HardwareUUID     string
	HardwareSerial   string
	NodeKey          string
	TeamID           *uint
	Cooldown         time.Duration
	IdentityCert     *types.HostIdentityCertificate
	IgnoreTeamUpdate bool // when true the host's team won't be updated on enrollment where an entry already exists.
}

// DatastoreEnrollOsqueryOption is a functional option for configuring datastore Host enrollment
type DatastoreEnrollOsqueryOption func(*DatastoreEnrollOsqueryConfig)

// WithEnrollOsqueryMDMEnabled sets the MDM enabled flag for datastore Host enrollment
func WithEnrollOsqueryMDMEnabled(enabled bool) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.IsMDMEnabled = enabled
	}
}

// WithEnrollOsqueryHostID sets the osquery host ID for datastore Host enrollment
func WithEnrollOsqueryHostID(osqueryHostID string) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.OsqueryHostID = osqueryHostID
	}
}

// WithEnrollOsqueryHardwareUUID sets the hardware UUID for datastore Host enrollment
func WithEnrollOsqueryHardwareUUID(hardwareUUID string) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.HardwareUUID = hardwareUUID
	}
}

// WithEnrollOsqueryHardwareSerial sets the hardware serial for datastore Host enrollment
func WithEnrollOsqueryHardwareSerial(hardwareSerial string) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.HardwareSerial = hardwareSerial
	}
}

// WithEnrollOsqueryNodeKey sets the node key for datastore Host enrollment
func WithEnrollOsqueryNodeKey(nodeKey string) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.NodeKey = nodeKey
	}
}

// WithEnrollOsqueryTeamID sets the team ID for datastore Host enrollment
func WithEnrollOsqueryTeamID(teamID *uint) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.TeamID = teamID
	}
}

// WithEnrollOsqueryCooldown sets the cooldown duration for datastore Host enrollment
func WithEnrollOsqueryCooldown(cooldown time.Duration) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.Cooldown = cooldown
	}
}

func WithEnrollOsqueryIdentityCert(identityCert *types.HostIdentityCertificate) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.IdentityCert = identityCert
	}
}

// WithEnrollOsqueryIgnoreTeamUpdate sets whether to ignore team updates for datastore Host enrollment
// it only acts on existing hosts (i.e. it won't ignore the team id on new hosts)
func WithEnrollOsqueryIgnoreTeamUpdate(ignore bool) DatastoreEnrollOsqueryOption {
	return func(c *DatastoreEnrollOsqueryConfig) {
		c.IgnoreTeamUpdate = ignore
	}
}
