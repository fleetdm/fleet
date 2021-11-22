package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func queryFromSpec(spec *fleet.QuerySpec) *fleet.Query {
	return &fleet.Query{
		Name:        spec.Name,
		Description: spec.Description,
		Query:       spec.Query,
	}
}

func specFromQuery(query *fleet.Query) *fleet.QuerySpec {
	return &fleet.QuerySpec{
		Name:        query.Name,
		Description: query.Description,
		Query:       query.Query,
	}
}

func (svc Service) ApplyQuerySpecs(ctx context.Context, specs []*fleet.QuerySpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionWrite); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return ctxerr.New(ctx, "user must be authenticated to apply queries")
	}

	queries := []*fleet.Query{}
	for _, spec := range specs {
		queries = append(queries, queryFromSpec(spec))
	}

	for _, query := range queries {
		if err := query.ValidateSQL(); err != nil {
			return err
		}
	}

	err := svc.ds.ApplyQueries(ctx, vc.UserID(), queries)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "applying queries")
	}

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeAppliedSpecSavedQuery,
		&map[string]interface{}{"specs": specs},
	)
}

func (svc Service) GetQuerySpecs(ctx context.Context) ([]*fleet.QuerySpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	queries, err := svc.ds.ListQueries(ctx, fleet.ListQueryOptions{})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting queries")
	}

	specs := []*fleet.QuerySpec{}
	for _, query := range queries {
		specs = append(specs, specFromQuery(query))
	}
	return specs, nil
}

func (svc Service) GetQuerySpec(ctx context.Context, name string) (*fleet.QuerySpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	query, err := svc.ds.QueryByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return specFromQuery(query), nil
}

func onlyShowObserverCanRunQueries(user *fleet.User) bool {
	if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleObserver {
		return true
	} else if len(user.Teams) > 0 {
		allObserver := true
		for _, team := range user.Teams {
			if team.Role != fleet.RoleObserver {
				allObserver = false
				break
			}
		}
		return allObserver
	}
	return false
}

func (svc Service) ListQueries(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Query, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	user := authz.UserFromContext(ctx)
	onlyShowObserverCanRun := onlyShowObserverCanRunQueries(user)

	queries, err := svc.ds.ListQueries(ctx, fleet.ListQueryOptions{
		ListOptions:        opt,
		OnlyObserverCanRun: onlyShowObserverCanRun,
	})
	if err != nil {
		return nil, err
	}

	return queries, nil
}

func (svc *Service) GetQuery(ctx context.Context, id uint) (*fleet.Query, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Query(ctx, id)
}

func (svc *Service) NewQuery(ctx context.Context, p fleet.QueryPayload) (*fleet.Query, error) {
	user := authz.UserFromContext(ctx)
	q := &fleet.Query{}
	if user != nil {
		q.AuthorID = ptr.Uint(user.ID)
	}
	if err := svc.authz.Authorize(ctx, q, fleet.ActionWrite); err != nil {
		return nil, err
	}

	query := &fleet.Query{Saved: true}

	if p.Name != nil {
		query.Name = *p.Name
	}

	if p.Description != nil {
		query.Description = *p.Description
	}

	if p.Query != nil {
		query.Query = *p.Query
	}

	logging.WithExtras(ctx, "name", query.Name, "sql", query.Query)

	if p.ObserverCanRun != nil {
		query.ObserverCanRun = *p.ObserverCanRun
	}

	vc, ok := viewer.FromContext(ctx)
	if ok {
		query.AuthorID = ptr.Uint(vc.UserID())
		query.AuthorName = vc.FullName()
		query.AuthorEmail = vc.Email()
	}

	if err := query.ValidateSQL(); err != nil {
		return nil, err
	}

	query, err := svc.ds.NewQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedSavedQuery,
		&map[string]interface{}{"query_id": query.ID, "query_name": query.Name},
	); err != nil {
		return nil, err
	}

	return query, nil
}

func (svc *Service) ModifyQuery(ctx context.Context, id uint, p fleet.QueryPayload) (*fleet.Query, error) {
	// First make sure the user can read queries
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	query, err := svc.ds.Query(ctx, id)
	if err != nil {
		return nil, err
	}

	// Then we make sure they can modify them
	if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
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

	logging.WithExtras(ctx, "name", query.Name, "sql", query.Query)

	if p.ObserverCanRun != nil {
		query.ObserverCanRun = *p.ObserverCanRun
	}

	if err := query.ValidateSQL(); err != nil {
		return nil, err
	}

	if err := svc.ds.SaveQuery(ctx, query); err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedSavedQuery,
		&map[string]interface{}{"query_id": query.ID, "query_name": query.Name},
	); err != nil {
		return nil, err
	}

	return query, nil
}

func (svc *Service) DeleteQuery(ctx context.Context, name string) error {
	// First make sure the user can read queries
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return err
	}

	query, err := svc.ds.QueryByName(ctx, name)
	if err != nil {
		return err
	}

	// Then we make sure they can modify them
	if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteQuery(ctx, name); err != nil {
		return err
	}

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedSavedQuery,
		&map[string]interface{}{"query_name": name},
	)
}

func (svc *Service) DeleteQueryByID(ctx context.Context, id uint) error {
	// First make sure the user can read queries
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return err
	}

	query, err := svc.ds.Query(ctx, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "lookup query by ID")
	}

	// Then we make sure they can modify them
	if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteQuery(ctx, query.Name); err != nil {
		return ctxerr.Wrap(ctx, err, "delete query")
	}

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedSavedQuery,
		&map[string]interface{}{"query_name": query.Name},
	)
}

func (svc *Service) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	// First make sure the user can read queries
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return 0, err
	}

	for _, id := range ids {
		query, err := svc.ds.Query(ctx, id)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "lookup query by ID")
		}

		// Then we make sure they can modify them
		if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
			return 0, err
		}
	}

	n, err := svc.ds.DeleteQueries(ctx, ids)
	if err != nil {
		return n, err
	}

	err = svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedMultipleSavedQuery,
		&map[string]interface{}{"query_ids": ids},
	)
	if err != nil {
		return n, err
	}

	return n, nil
}
