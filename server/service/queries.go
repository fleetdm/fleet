package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Get Query
////////////////////////////////////////////////////////////////////////////////

type getQueryRequest struct {
	ID uint `url:"id"`
}

type getQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r getQueryResponse) error() error { return r.Err }

func getQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getQueryRequest)
	query, err := svc.GetQuery(ctx, req.ID)
	if err != nil {
		return getQueryResponse{Err: err}, nil
	}
	return getQueryResponse{query, nil}, nil
}

func (svc *Service) GetQuery(ctx context.Context, id uint) (*fleet.Query, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.Query(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// List Queries
////////////////////////////////////////////////////////////////////////////////

type listQueriesRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listQueriesResponse struct {
	Queries []fleet.Query `json:"queries"`
	Err     error         `json:"error,omitempty"`
}

func (r listQueriesResponse) error() error { return r.Err }

func listQueriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listQueriesRequest)
	queries, err := svc.ListQueries(ctx, req.ListOptions)
	if err != nil {
		return listQueriesResponse{Err: err}, nil
	}

	resp := listQueriesResponse{Queries: []fleet.Query{}}
	for _, query := range queries {
		resp.Queries = append(resp.Queries, *query)
	}
	return resp, nil
}

func (svc *Service) ListQueries(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Query, error) {
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

////////////////////////////////////////////////////////////////////////////////
// Create Query
////////////////////////////////////////////////////////////////////////////////

type createQueryRequest struct {
	fleet.QueryPayload
}

type createQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r createQueryResponse) error() error { return r.Err }

func createQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createQueryRequest)
	query, err := svc.NewQuery(ctx, req.QueryPayload)
	if err != nil {
		return createQueryResponse{Err: err}, nil
	}
	return createQueryResponse{query, nil}, nil
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

	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &badRequestError{
			message: fmt.Sprintf("query payload verification: %s", err),
		})
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

////////////////////////////////////////////////////////////////////////////////
// Modify Query
////////////////////////////////////////////////////////////////////////////////

type modifyQueryRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.QueryPayload
}

type modifyQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r modifyQueryResponse) error() error { return r.Err }

func modifyQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyQueryRequest)
	query, err := svc.ModifyQuery(ctx, req.ID, req.QueryPayload)
	if err != nil {
		return modifyQueryResponse{Err: err}, nil
	}
	return modifyQueryResponse{query, nil}, nil
}

func (svc *Service) ModifyQuery(ctx context.Context, id uint, p fleet.QueryPayload) (*fleet.Query, error) {
	// First make sure the user can read queries
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &badRequestError{
			message: fmt.Sprintf("query payload verification: %s", err),
		})
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

////////////////////////////////////////////////////////////////////////////////
// Delete Query
////////////////////////////////////////////////////////////////////////////////

type deleteQueryRequest struct {
	Name string `url:"name"`
}

type deleteQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryResponse) error() error { return r.Err }

func deleteQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteQueryRequest)
	err := svc.DeleteQuery(ctx, req.Name)
	if err != nil {
		return deleteQueryResponse{Err: err}, nil
	}
	return deleteQueryResponse{}, nil
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

////////////////////////////////////////////////////////////////////////////////
// Delete Query By ID
////////////////////////////////////////////////////////////////////////////////

type deleteQueryByIDRequest struct {
	ID uint `url:"id"`
}

type deleteQueryByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryByIDResponse) error() error { return r.Err }

func deleteQueryByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteQueryByIDRequest)
	err := svc.DeleteQueryByID(ctx, req.ID)
	if err != nil {
		return deleteQueryByIDResponse{Err: err}, nil
	}
	return deleteQueryByIDResponse{}, nil
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

////////////////////////////////////////////////////////////////////////////////
// Delete Queries
////////////////////////////////////////////////////////////////////////////////

type deleteQueriesRequest struct {
	IDs []uint `json:"ids"`
}

type deleteQueriesResponse struct {
	Deleted uint  `json:"deleted"`
	Err     error `json:"error,omitempty"`
}

func (r deleteQueriesResponse) error() error { return r.Err }

func deleteQueriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteQueriesRequest)
	deleted, err := svc.DeleteQueries(ctx, req.IDs)
	if err != nil {
		return deleteQueriesResponse{Err: err}, nil
	}
	return deleteQueriesResponse{Deleted: deleted}, nil
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

////////////////////////////////////////////////////////////////////////////////
// Apply Query Spec
////////////////////////////////////////////////////////////////////////////////

type applyQuerySpecsRequest struct {
	Specs []*fleet.QuerySpec `json:"specs"`
}

type applyQuerySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyQuerySpecsResponse) error() error { return r.Err }

func applyQuerySpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*applyQuerySpecsRequest)
	err := svc.ApplyQuerySpecs(ctx, req.Specs)
	if err != nil {
		return applyQuerySpecsResponse{Err: err}, nil
	}
	return applyQuerySpecsResponse{}, nil
}

func (svc *Service) ApplyQuerySpecs(ctx context.Context, specs []*fleet.QuerySpec) error {
	// check that the user can create queries
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionWrite); err != nil {
		return err
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

		// check that the user can update the query if it already exists
		query, err := svc.ds.QueryByName(ctx, query.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		} else if err == nil {
			if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
				return err
			}
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return ctxerr.New(ctx, "user must be authenticated to apply queries")
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

func queryFromSpec(spec *fleet.QuerySpec) *fleet.Query {
	return &fleet.Query{
		Name:        spec.Name,
		Description: spec.Description,
		Query:       spec.Query,
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Specs
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecsResponse struct {
	Specs []*fleet.QuerySpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getQuerySpecsResponse) error() error { return r.Err }

func getQuerySpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	specs, err := svc.GetQuerySpecs(ctx)
	if err != nil {
		return getQuerySpecsResponse{Err: err}, nil
	}
	return getQuerySpecsResponse{Specs: specs}, nil
}

func (svc *Service) GetQuerySpecs(ctx context.Context) ([]*fleet.QuerySpec, error) {
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

func specFromQuery(query *fleet.Query) *fleet.QuerySpec {
	return &fleet.QuerySpec{
		Name:        query.Name,
		Description: query.Description,
		Query:       query.Query,
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Spec
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecResponse struct {
	Spec *fleet.QuerySpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

func (r getQuerySpecResponse) error() error { return r.Err }

func getQuerySpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getGenericSpecRequest)
	spec, err := svc.GetQuerySpec(ctx, req.Name)
	if err != nil {
		return getQuerySpecResponse{Err: err}, nil
	}
	return getQuerySpecResponse{Spec: spec}, nil
}

func (svc *Service) GetQuerySpec(ctx context.Context, name string) (*fleet.QuerySpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Query{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	query, err := svc.ds.QueryByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return specFromQuery(query), nil
}
