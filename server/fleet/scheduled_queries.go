package fleet

import (
	"context"
	"time"

	"gopkg.in/guregu/null.v3"
)

type ScheduledQueryStore interface {
	ListScheduledQueriesInPack(id uint, opts ListOptions) ([]*ScheduledQuery, error)
	NewScheduledQuery(sq *ScheduledQuery, opts ...OptionalArg) (*ScheduledQuery, error)
	SaveScheduledQuery(sq *ScheduledQuery) (*ScheduledQuery, error)
	DeleteScheduledQuery(id uint) error
	ScheduledQuery(id uint) (*ScheduledQuery, error)
	CleanupOrphanScheduledQueryStats() error
}

type ScheduledQueryService interface {
	GetScheduledQueriesInPack(ctx context.Context, id uint, opts ListOptions) (queries []*ScheduledQuery, err error)
	GetScheduledQuery(ctx context.Context, id uint) (query *ScheduledQuery, err error)
	ScheduleQuery(ctx context.Context, sq *ScheduledQuery) (query *ScheduledQuery, err error)
	DeleteScheduledQuery(ctx context.Context, id uint) (err error)
	ModifyScheduledQuery(ctx context.Context, id uint, p ScheduledQueryPayload) (query *ScheduledQuery, err error)
}

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
