package fleet

import (
	"time"

	"gopkg.in/guregu/null.v3"
)

type ScheduledQuery struct {
	UpdateCreateTimestamps
	ID          uint    `json:"id"`
	PackID      uint    `json:"pack_id" db:"pack_id"`
	Name        string  `json:"name"`
	QueryID     uint    `json:"query_id" db:"query_id"`
	QueryName   string  `json:"query_name" db:"query_name"`
	Query       string  `json:"query"` // populated via a join on queries
	Description string  `json:"description,omitempty"`
	Interval    uint    `json:"interval"`
	Snapshot    *bool   `json:"snapshot"`
	Removed     *bool   `json:"removed"`
	Platform    *string `json:"platform,omitempty"`
	Version     *string `json:"version,omitempty"`
	Shard       *uint   `json:"shard"`
	Denylist    *bool   `json:"denylist"`

	AggregatedStats `json:"stats,omitempty"`
}

type AggregatedStats struct {
	SystemTimeP50   *float64 `json:"system_time_p50" db:"system_time_p50"`
	SystemTimeP95   *float64 `json:"system_time_p95" db:"system_time_p95"`
	UserTimeP50     *float64 `json:"user_time_p50" db:"user_time_p50"`
	UserTimeP95     *float64 `json:"user_time_p_95" db:"user_time_p95"`
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
	AverageMemory int  `json:"average_memory" db:"average_memory"`
	Denylisted    bool `json:"denylisted" db:"denylisted"`
	Executions    int  `json:"executions" db:"executions"`
	// Note schedule_interval is used for DB since "interval" is a reserved word in MySQL
	Interval     int       `json:"interval" db:"schedule_interval"`
	LastExecuted time.Time `json:"last_executed" db:"last_executed"`
	OutputSize   int       `json:"output_size" db:"output_size"`
	SystemTime   int       `json:"system_time" db:"system_time"`
	UserTime     int       `json:"user_time" db:"user_time"`
	WallTime     int       `json:"wall_time" db:"wall_time"`
}
