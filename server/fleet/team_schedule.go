package fleet

import "context"

type TeamScheduleService interface {
	TeamScheduleQuery(ctx context.Context, teamID uint, sq *ScheduledQuery) (*ScheduledQuery, error)
	GetTeamScheduledQueries(ctx context.Context, teamID uint, opts ListOptions) ([]*ScheduledQuery, error)
	ModifyTeamScheduledQueries(ctx context.Context, teamID uint, scheduledQueryID uint, q ScheduledQueryPayload) (*ScheduledQuery, error)
	DeleteTeamScheduledQueries(ctx context.Context, teamID uint, id uint) error
}
