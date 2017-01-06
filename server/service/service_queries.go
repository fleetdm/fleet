package service

import (
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) ListQueries(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Query, error) {
	return svc.ds.ListQueries(opt)
}

func (svc service) GetQuery(ctx context.Context, id uint) (*kolide.Query, error) {
	return svc.ds.Query(id)
}

func (svc service) NewQuery(ctx context.Context, p kolide.QueryPayload) (*kolide.Query, error) {
	query := &kolide.Query{Saved: true}

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Description != nil {
		query.Description = *p.Description
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	vc, ok := viewer.FromContext(ctx)
	if ok {
		query.AuthorID = vc.UserID()
		query.AuthorName = vc.FullName()
	}

	query, err := svc.ds.NewQuery(query)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (svc service) ModifyQuery(ctx context.Context, id uint, p kolide.QueryPayload) (*kolide.Query, error) {
	query, err := svc.ds.Query(id)
	if err != nil {
		return nil, err
	}

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Description != nil {
		query.Description = *p.Description
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	err = svc.ds.SaveQuery(query)
	if err != nil {
		return nil, err
	}

	return query, nil
}

func (svc service) DeleteQuery(ctx context.Context, id uint) error {
	return svc.ds.DeleteQuery(id)
}

func (svc service) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	return svc.ds.DeleteQueries(ids)
}
