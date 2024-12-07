package fleet

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"gopkg.in/guregu/null.v3"
)

// ScheduledQuery is a query that runs on a schedule.
//
// Source of documentation for the fields:
// https://osquery.readthedocs.io/en/stable/deployment/configuration/
type ScheduledQuery struct {
	UpdateCreateTimestamps
	ID          uint   `json:"id"`
	PackID      uint   `json:"pack_id" db:"pack_id"`
	Name        string `json:"name"`
	QueryID     uint   `json:"query_id" db:"query_id"`
	QueryName   string `json:"query_name" db:"query_name"`
	Query       string `json:"query"` // populated via a join on queries
	Description string `json:"description,omitempty"`
	// Interval specifies query frequency, in seconds.
	Interval uint  `json:"interval"`
	Snapshot *bool `json:"snapshot"`
	// Removed is a boolean to determine if "removed" actions
	// should be logged default is true.
	//
	// When the results from a table differ from the results when the
	// query was last executed, logs are emitted with {"action": "removed"}
	// or {"action": "added"} for the appropriate action.
	// References:
	//	https://osquery.readthedocs.io/en/stable/deployment/logging/#differential-logs
	Removed *bool `json:"removed"`
	// Platform is a comma-separated string that indicates the target platforms
	// for this scheduled query.
	//
	// Possible values are: "darwin", "linux" and "windows".
	// An empty string or nil means the scheduled query will run on all platforms.
	Platform *string `json:"platform,omitempty"`
	// Version can be set to only run on osquery versions greater
	// than or equal-to this version string.
	Version *string `json:"version,omitempty"`
	// Shard restricts this query to a percentage (1-100) of target hosts.
	Shard *uint `json:"shard"`
	// Denylist is a boolean to determine if this query may be denylisted
	// (when stopped by the Watchdog for excessive resource consumption),
	// default is true.
	Denylist *bool `json:"denylist"`

	AggregatedStats `json:"stats,omitempty"`

	/////////////////////////////////////////////////////////////////
	// WARNING: If you add to this struct make sure it's taken into
	// account in the ScheduledQuery Clone implementation!
	/////////////////////////////////////////////////////////////////
}

// Clone implements cloner for ScheduledQuery.
func (q *ScheduledQuery) Clone() (Cloner, error) {
	return q.Copy(), nil
}

// Copy returns a deep copy of the ScheduledQuery.
func (q *ScheduledQuery) Copy() *ScheduledQuery {
	if q == nil {
		return nil
	}

	clone := *q

	if q.Snapshot != nil {
		clone.Snapshot = ptr.Bool(*q.Snapshot)
	}
	if q.Removed != nil {
		clone.Removed = ptr.Bool(*q.Removed)
	}
	if q.Platform != nil {
		clone.Platform = ptr.String(*q.Platform)
	}
	if q.Version != nil {
		clone.Version = ptr.String(*q.Version)
	}
	if q.Shard != nil {
		clone.Shard = ptr.Uint(*q.Shard)
	}
	if q.Denylist != nil {
		clone.Denylist = ptr.Bool(*q.Denylist)
	}

	if q.AggregatedStats.SystemTimeP50 != nil {
		clone.AggregatedStats.SystemTimeP50 = ptr.Float64(*q.AggregatedStats.SystemTimeP50)
	}
	if q.AggregatedStats.SystemTimeP95 != nil {
		clone.AggregatedStats.SystemTimeP95 = ptr.Float64(*q.AggregatedStats.SystemTimeP95)
	}
	if q.AggregatedStats.UserTimeP50 != nil {
		clone.AggregatedStats.UserTimeP50 = ptr.Float64(*q.AggregatedStats.UserTimeP50)
	}
	if q.AggregatedStats.UserTimeP95 != nil {
		clone.AggregatedStats.UserTimeP95 = ptr.Float64(*q.AggregatedStats.UserTimeP95)
	}
	if q.AggregatedStats.TotalExecutions != nil {
		clone.AggregatedStats.TotalExecutions = ptr.Float64(*q.AggregatedStats.TotalExecutions)
	}
	return &clone
}

type ScheduledQueryList []*ScheduledQuery

func (sql ScheduledQueryList) Clone() (Cloner, error) {
	var cloned ScheduledQueryList
	for _, sq := range sql {
		cloned = append(cloned, sq.Copy())
	}
	return cloned, nil
}

type AggregatedStats struct {
	SystemTimeP50   *float64 `json:"system_time_p50" db:"system_time_p50"`
	SystemTimeP95   *float64 `json:"system_time_p95" db:"system_time_p95"`
	UserTimeP50     *float64 `json:"user_time_p50" db:"user_time_p50"`
	UserTimeP95     *float64 `json:"user_time_p95" db:"user_time_p95"`
	TotalExecutions *float64 `json:"total_executions" db:"total_executions"`
}

type ScheduledQueryPayload struct {
	PackID   *uint     `json:"pack_id"`
	QueryID  *uint     `json:"query_id"`
	Interval *uint     `json:"interval"`
	Snapshot *bool     `json:"snapshot"`
	Removed  *bool     `json:"removed"`
	Platform *string   `json:"platform"`
	Version  *string   `json:"version"`
	Shard    *null.Int `json:"shard"`
	Denylist *bool     `json:"denylist"`
}

type ScheduledQueryStats struct {
	ScheduledQueryName string `json:"scheduled_query_name,omitempty" db:"scheduled_query_name"`
	ScheduledQueryID   uint   `json:"scheduled_query_id,omitempty" db:"scheduled_query_id"`

	QueryName   string `json:"query_name,omitempty" db:"query_name"`
	Description string `json:"description,omitempty" db:"description"`

	PackName string `json:"pack_name,omitempty" db:"pack_name"`
	PackID   uint   `json:"pack_id,omitempty" db:"pack_id"`

	// From osquery directly
	AverageMemory uint64 `json:"average_memory" db:"average_memory"`
	Denylisted    bool   `json:"denylisted" db:"denylisted"`
	Executions    uint64 `json:"executions" db:"executions"`
	// Note schedule_interval is used for DB since "interval" is a reserved word in MySQL
	Interval           int        `json:"interval" db:"schedule_interval"`
	DiscardData        bool       `json:"discard_data" db:"discard_data"`
	LastFetched        *time.Time `json:"last_fetched" db:"last_fetched"`
	AutomationsEnabled bool       `json:"automations_enabled" db:"automations_enabled"`
	LastExecuted       time.Time  `json:"last_executed" db:"last_executed"`
	OutputSize         uint64     `json:"output_size" db:"output_size"`
	SystemTime         uint64     `json:"system_time" db:"system_time"`
	UserTime           uint64     `json:"user_time" db:"user_time"`
	// WallTimeMs from osquery maps to WallTime in the DB.
	WallTime uint64 `json:"wall_time" db:"wall_time"`
	// WallTimeMs should always be hidden (0-value) in the Fleet API response since it does not map to a field in the DB.
	WallTimeMs uint64 `json:"wall_time_ms,omitempty"`
}

// TeamID returns the team id if the stat is for a team query stat result
func (sqs *ScheduledQueryStats) TeamID() (*int, error) {
	if strings.HasPrefix(sqs.PackName, "team-") {
		teamIDParts := strings.Split(sqs.PackName, "-")
		if len(teamIDParts) != 2 {
			return nil, fmt.Errorf("invalid pack name: %s", sqs.PackName)
		}

		teamID, err := strconv.Atoi(teamIDParts[1])
		if err != nil {
			return nil, err
		}
		return &teamID, nil
	}

	return nil, nil
}

func ScheduledQueryFromQuery(query *Query) *ScheduledQuery {
	var (
		snapshot *bool
		removed  *bool
	)
	if query.Logging == "" || query.Logging == "snapshot" { //nolint:gocritic // ignore ifElseChain
		snapshot = ptr.Bool(true)
		removed = ptr.Bool(false)
	} else if query.Logging == "differential" {
		snapshot = ptr.Bool(false)
		removed = ptr.Bool(true)
	} else { // query.Logging == "differential_ignore_removals"
		snapshot = ptr.Bool(false)
		removed = ptr.Bool(false)
	}
	return &ScheduledQuery{
		ID:              query.ID,
		Name:            query.Name,
		QueryID:         query.ID,
		QueryName:       query.Name,
		Query:           query.Query,
		Description:     query.Description,
		Interval:        query.Interval,
		Snapshot:        snapshot,
		Removed:         removed,
		Platform:        &query.Platform,
		Version:         &query.MinOsqueryVersion,
		AggregatedStats: query.AggregatedStats,
	}
}

func ScheduledQueryToQueryPayloadForNewQuery(originalQuery *Query, scheduledQuery *ScheduledQuery) QueryPayload {
	logging := ptr.String(LoggingSnapshot) // default is snapshot.
	if scheduledQuery.Snapshot != nil && scheduledQuery.Removed != nil {
		if *scheduledQuery.Snapshot { //nolint:gocritic // ignore ifElseChain
			logging = ptr.String(LoggingSnapshot)
		} else if *scheduledQuery.Removed {
			logging = ptr.String(LoggingDifferential)
		} else {
			logging = ptr.String(LoggingDifferentialIgnoreRemovals)
		}
	}
	return QueryPayload{
		Name:               &originalQuery.Name,
		Description:        &originalQuery.Description,
		Query:              &originalQuery.Query,
		ObserverCanRun:     &originalQuery.ObserverCanRun,
		TeamID:             originalQuery.TeamID,
		Interval:           &scheduledQuery.Interval,
		Platform:           scheduledQuery.Platform,
		MinOsqueryVersion:  scheduledQuery.Version,
		AutomationsEnabled: ptr.Bool(true),
		Logging:            logging,
	}
}

// NOTE(lucas): payload.Snapshot and payload.Removed must both be set in order to
// change the logging behavior of a scheduled query.
// Document this API change.
func ScheduledQueryPayloadToQueryPayloadForModifyQuery(payload ScheduledQueryPayload) QueryPayload {
	var logging *string
	if payload.Snapshot != nil && payload.Removed != nil {
		if *payload.Snapshot { //nolint:gocritic // ignore ifElseChain
			logging = ptr.String(LoggingSnapshot)
		} else if *payload.Removed {
			logging = ptr.String(LoggingDifferential)
		} else {
			logging = ptr.String(LoggingDifferentialIgnoreRemovals)
		}
	}
	return QueryPayload{
		Interval:          payload.Interval,
		Platform:          payload.Platform,
		MinOsqueryVersion: payload.Version,
		Logging:           logging,
	}
}
