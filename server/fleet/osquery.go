package fleet

// OsqueryDistributedQueryResults represents the format of the results of an
// osquery distributed query.
type OsqueryDistributedQueryResults map[string][]map[string]string

// OsqueryStatus represents osquery status codes (0 = success, nonzero =
// failure)
type OsqueryStatus int

// QueryExcessiveCPUUsageThreshMs is the currently agreed upon value to
// consider a query to be "excessive" in CPU usage.
//
// This time includes both system_time (kernel space code) +
// user_time (user space code).
//
// NOTE(lucas): The value must match what we use in
// frontend/utilities/helpers.ts:performanceIndicator.
const QueryExcessiveCPUUsageThreshMs = 4000

// OsqueryStats are stats of a executed distributed query reported by a host.
type OsqueryStats struct {
	// WallTimeMs is the time in milliseconds that a clock on the wall
	// would measure as having elapsed between the start of the
	// process and 'now'.
	WallTimeMs uint64 `json:"wall_time_ms"`
	// UserTimeMs is the time in milliseconds spent in user space code.
	UserTimeMs uint64 `json:"user_time"`
	// SystemTimeMs is the time in milliseconds spent in kernel space code.
	SystemTimeMs uint64 `json:"system_time"`
	// Memory is the memory footprint in bytes.
	Memory uint64 `json:"memory"`
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
