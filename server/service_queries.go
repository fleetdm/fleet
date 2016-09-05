package server

import (
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (svc service) GetAllQueries(ctx context.Context) ([]*kolide.Query, error) {
	return svc.ds.Queries()
}

func (svc service) GetQuery(ctx context.Context, id uint) (*kolide.Query, error) {
	return svc.ds.Query(id)
}

func (svc service) NewQuery(ctx context.Context, p kolide.QueryPayload) (*kolide.Query, error) {
	var query kolide.Query

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	if p.Interval != nil {
		query.Interval = *p.Interval
	}

	if p.Snapshot != nil {
		query.Snapshot = *p.Snapshot
	}

	if p.Differential != nil {
		query.Differential = *p.Differential
	}

	if p.Platform != nil {
		query.Platform = *p.Platform
	}

	if p.Version != nil {
		query.Version = *p.Version
	}

	err := svc.ds.NewQuery(&query)
	if err != nil {
		return nil, err
	}
	return &query, nil
}

func (svc service) ModifyQuery(ctx context.Context, id uint, p kolide.QueryPayload) (*kolide.Query, error) {
	query, err := svc.ds.Query(id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	if p.Interval != nil {
		query.Interval = *p.Interval
	}

	if p.Snapshot != nil {
		query.Snapshot = *p.Snapshot
	}

	if p.Differential != nil {
		query.Differential = *p.Differential
	}

	if p.Platform != nil {
		query.Platform = *p.Platform
	}

	if p.Version != nil {
		query.Version = *p.Version
	}

	err = svc.ds.SaveQuery(query)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (svc service) DeleteQuery(ctx context.Context, id uint) error {
	query, err := svc.ds.Query(id)
	if err != nil {
		return err
	}

	err = svc.ds.DeleteQuery(query)
	if err != nil {
		return err
	}

	return nil
}
