package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) GetScheduledQuery(ctx context.Context, id uint) (*kolide.ScheduledQuery, error) {
	return svc.ds.ScheduledQuery(id)
}

func (svc service) GetScheduledQueriesInPack(ctx context.Context, id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	return svc.ds.ListScheduledQueriesInPack(id, opts)
}

func (svc service) ScheduleQuery(ctx context.Context, sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	return svc.ds.NewScheduledQuery(sq)
}

func (svc service) ModifyScheduledQuery(ctx context.Context, sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	return svc.ds.SaveScheduledQuery(sq)
}

func (svc service) DeleteScheduledQuery(ctx context.Context, id uint) error {
	return svc.ds.DeleteScheduledQuery(id)
}
