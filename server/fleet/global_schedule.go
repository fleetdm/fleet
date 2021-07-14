package fleet

import (
	"context"
)

type GlobalScheduleService interface {
	GlobalScheduleQuery(ctx context.Context, sq *ScheduledQuery) (*ScheduledQuery, error)
	GetGlobalScheduledQueries(ctx context.Context, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyGlobalScheduledQueries(ctx context.Context, id uint, q ScheduledQueryPayload) (*ScheduledQuery, error)
	DeleteGlobalScheduledQueries(ctx context.Context, id uint) error
}

type GlobalSchedulePayload struct {
	GlobalSchedule []*ScheduledQuery `json:"global_schedule"`
}
