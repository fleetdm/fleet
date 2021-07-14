package fleet

import (
	"context"
	"gopkg.in/guregu/null.v3"
)

type GlobalScheduleService interface {
	GetGlobalScheduledQueries(ctx context.Context, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyGlobalScheduledQueries(ctx context.Context, queries []GlobalScheduleQueryPayload) ([]*ScheduledQuery, error)
	DeleteGlobalScheduledQueries(ctx context.Context) error
}

type GlobalSchedulePayload struct {
	GlobalSchedule []GlobalScheduleQueryPayload `json:"global_schedule"`
}

type GlobalScheduleQueryPayload struct {
	QueryID  *uint     `json:"query_id"`
	Interval *uint     `json:"interval"`
	Snapshot *bool     `json:"snapshot"`
	Removed  *bool     `json:"removed"`
	Platform *string   `json:"platform"`
	Version  *string   `json:"version"`
	Shard    *null.Int `json:"shard"`
	Denylist *bool     `json:"denylist"`
}
