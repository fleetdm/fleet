package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Get Query
////////////////////////////////////////////////////////////////////////////////

type getQueryRequest struct {
	ID uint
}

type getQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r getQueryResponse) error() error { return r.Err }

func makeGetQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getQueryRequest)
		query, err := svc.GetQuery(ctx, req.ID)
		if err != nil {
			return getQueryResponse{Err: err}, nil
		}
		return getQueryResponse{query, nil}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Queries
////////////////////////////////////////////////////////////////////////////////
type listQueriesRequest struct {
	ListOptions fleet.ListOptions
}

type listQueriesResponse struct {
	Queries []fleet.Query `json:"queries"`
	Err     error         `json:"error,omitempty"`
}

func (r listQueriesResponse) error() error { return r.Err }

func makeListQueriesEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listQueriesRequest)
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
}

////////////////////////////////////////////////////////////////////////////////
// Create Query
////////////////////////////////////////////////////////////////////////////////

type createQueryRequest struct {
	payload fleet.QueryPayload
}

type createQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r createQueryResponse) error() error { return r.Err }

func makeCreateQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createQueryRequest)
		query, err := svc.NewQuery(ctx, req.payload)
		if err != nil {
			return createQueryResponse{Err: err}, nil
		}
		return createQueryResponse{query, nil}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Query
////////////////////////////////////////////////////////////////////////////////

type modifyQueryRequest struct {
	ID      uint
	payload fleet.QueryPayload
}

type modifyQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r modifyQueryResponse) error() error { return r.Err }

func makeModifyQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyQueryRequest)
		query, err := svc.ModifyQuery(ctx, req.ID, req.payload)
		if err != nil {
			return modifyQueryResponse{Err: err}, nil
		}
		return modifyQueryResponse{query, nil}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Query
////////////////////////////////////////////////////////////////////////////////

type deleteQueryRequest struct {
	Name string
}

type deleteQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryResponse) error() error { return r.Err }

func makeDeleteQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteQueryRequest)
		err := svc.DeleteQuery(ctx, req.Name)
		if err != nil {
			return deleteQueryResponse{Err: err}, nil
		}
		return deleteQueryResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Query By ID
////////////////////////////////////////////////////////////////////////////////

type deleteQueryByIDRequest struct {
	ID uint
}

type deleteQueryByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryByIDResponse) error() error { return r.Err }

func makeDeleteQueryByIDEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteQueryByIDRequest)
		err := svc.DeleteQueryByID(ctx, req.ID)
		if err != nil {
			return deleteQueryByIDResponse{Err: err}, nil
		}
		return deleteQueryByIDResponse{}, nil
	}
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

func makeDeleteQueriesEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteQueriesRequest)
		deleted, err := svc.DeleteQueries(ctx, req.IDs)
		if err != nil {
			return deleteQueriesResponse{Err: err}, nil
		}
		return deleteQueriesResponse{Deleted: deleted}, nil
	}
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

func makeApplyQuerySpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyQuerySpecsRequest)
		err := svc.ApplyQuerySpecs(ctx, req.Specs)
		if err != nil {
			return applyQuerySpecsResponse{Err: err}, nil
		}
		return applyQuerySpecsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Spec
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecsResponse struct {
	Specs []*fleet.QuerySpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getQuerySpecsResponse) error() error { return r.Err }

func makeGetQuerySpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		specs, err := svc.GetQuerySpecs(ctx)
		if err != nil {
			return getQuerySpecsResponse{Err: err}, nil
		}
		return getQuerySpecsResponse{Specs: specs}, nil
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

func makeGetQuerySpecEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGenericSpecRequest)
		spec, err := svc.GetQuerySpec(ctx, req.Name)
		if err != nil {
			return getQuerySpecResponse{Err: err}, nil
		}
		return getQuerySpecResponse{Spec: spec}, nil
	}
}
