package kolide

import (
	"context"
)

type ScheduledQueryStore interface {
	ListScheduledQueriesInPack(id uint, opts ListOptions) ([]*ScheduledQuery, error)
}

type ScheduledQueryService interface {
	GetScheduledQueriesInPack(ctx context.Context, id uint, opts ListOptions) (queries []*ScheduledQuery, err error)
}

type ScheduledQuery struct {
	UpdateCreateTimestamps
	DeleteFields
	ID          uint    `json:"id"`
	PackID      uint    `json:"pack_id" db:"pack_id"`
	Name        string  `json:"name"`
	QueryID     uint    `json:"query_id" db:"query_id"`
	QueryName   string  `json:"query_name" db:"query_name"`
	Query       string  `json:"query"` // populated via a join on queries
	Description string  `json:"description"`
	Interval    uint    `json:"interval"`
	Snapshot    *bool   `json:"snapshot"`
	Removed     *bool   `json:"removed"`
	Platform    *string `json:"platform"`
	Version     *string `json:"version"`
	Shard       *uint   `json:"shard"`
}

type ScheduledQueryPayload struct {
	PackID   *uint   `json:"pack_id"`
	QueryID  *uint   `json:"query_id"`
	Interval *uint   `json:"interval"`
	Snapshot *bool   `json:"snapshot"`
	Removed  *bool   `json:"removed"`
	Platform *string `json:"platform"`
	Version  *string `json:"version"`
	Shard    *uint   `json:"shard"`
}
