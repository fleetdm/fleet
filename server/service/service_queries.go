package service

import (
	"context"

	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func queryFromSpec(spec *kolide.QuerySpec) *kolide.Query {
	return &kolide.Query{
		Name:        spec.Name,
		Description: spec.Description,
		Query:       spec.Query,
	}
}

func specFromQuery(query *kolide.Query) *kolide.QuerySpec {
	return &kolide.QuerySpec{
		Name:        query.Name,
		Description: query.Description,
		Query:       query.Query,
	}
}

func (svc service) ApplyQuerySpecs(ctx context.Context, specs []*kolide.QuerySpec) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errors.New("user must be authenticated to apply queries")
	}

	queries := []*kolide.Query{}
	for _, spec := range specs {
		queries = append(queries, queryFromSpec(spec))
	}

	err := svc.ds.ApplyQueries(vc.UserID(), queries)
	return errors.Wrap(err, "applying queries")
}

func (svc service) GetQuerySpecs(ctx context.Context) ([]*kolide.QuerySpec, error) {
	queries, err := svc.ds.ListQueries(kolide.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "getting queries")
	}

	specs := []*kolide.QuerySpec{}
	for _, query := range queries {
		specs = append(specs, specFromQuery(query))
	}
	return specs, nil
}

func (svc service) GetQuerySpec(ctx context.Context, name string) (*kolide.QuerySpec, error) {
	query, err := svc.ds.QueryByName(name)
	if err != nil {
		return nil, err
	}
	return specFromQuery(query), nil
}

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
		query.AuthorID = uintPtr(vc.UserID())
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

func (svc service) DeleteQuery(ctx context.Context, name string) error {
	return svc.ds.DeleteQuery(name)
}

func (svc service) DeleteQueryByID(ctx context.Context, id uint) error {
	query, err := svc.ds.Query(id)
	if err != nil {
		return errors.Wrap(err, "lookup query by ID")
	}

	return errors.Wrap(svc.ds.DeleteQuery(query.Name), "delete query")
}

func (svc service) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	return svc.ds.DeleteQueries(ids)
}
