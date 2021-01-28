package kolide

import (
	"context"

	"gopkg.in/guregu/null.v3"
)

type ScheduledQueryStore interface {
	ListScheduledQueriesInPack(id uint, opts ListOptions) ([]*ScheduledQuery, error)
	NewScheduledQuery(sq *ScheduledQuery, opts ...OptionalArg) (*ScheduledQuery, error)
	SaveScheduledQuery(sq *ScheduledQuery) (*ScheduledQuery, error)
	DeleteScheduledQuery(id uint) error
	ScheduledQuery(id uint) (*ScheduledQuery, error)
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
