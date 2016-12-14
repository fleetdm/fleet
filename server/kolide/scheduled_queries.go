package kolide

import (
	"golang.org/x/net/context"
)

type ScheduledQueryStore interface {
	NewScheduledQuery(sq *ScheduledQuery) (*ScheduledQuery, error)
	SaveScheduledQuery(sq *ScheduledQuery) (*ScheduledQuery, error)
	DeleteScheduledQuery(id uint) error
	ScheduledQuery(id uint) (*ScheduledQuery, error)
	ListScheduledQueriesInPack(id uint, opts ListOptions) ([]*ScheduledQuery, error)
}

type ScheduledQueryService interface {
	GetScheduledQuery(ctx context.Context, id uint) (*ScheduledQuery, error)
	GetScheduledQueriesInPack(ctx context.Context, id uint, opts ListOptions) ([]*ScheduledQuery, error)
	ScheduleQuery(ctx context.Context, sq *ScheduledQuery) (*ScheduledQuery, error)
	DeleteScheduledQuery(ctx context.Context, id uint) error
	ModifyScheduledQuery(ctx context.Context, sq *ScheduledQuery) (*ScheduledQuery, error)
}

type ScheduledQuery struct {
	UpdateCreateTimestamps
	DeleteFields
	ID       uint    `json:"id"`
	PackID   uint    `json:"pack_id" db:"pack_id"`
	QueryID  uint    `json:"query_id" db:"query_id"`
	Query    string  `json:"query"` // populated via a join on queries
	Name     string  `json:"name"`  // populated via a join on queries
	Interval uint    `json:"interval"`
	Snapshot *bool   `json:"snapshot"`
	Removed  *bool   `json:"removed"`
	Platform *string `json:"platform"`
	Version  *string `json:"version"`
	Shard    *uint   `json:"shard"`
}
