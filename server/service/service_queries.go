package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
		if err := query.Verify(); err != nil {
			return ctxerr.Wrap(ctx, &badRequestError{
				message: fmt.Sprintf("query payload verification: %s", err),
			})
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
