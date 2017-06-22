package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

////////////////////////////////////////////////////////////////////////////////
// Get Query
////////////////////////////////////////////////////////////////////////////////

type getQueryRequest struct {
	ID uint
}

type getQueryResponse struct {
	Query *kolide.Query `json:"query,omitempty"`
	Err   error         `json:"error,omitempty"`
}

func (r getQueryResponse) error() error { return r.Err }

func makeGetQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
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
	ListOptions kolide.ListOptions
}

type listQueriesResponse struct {
	Queries []kolide.Query `json:"queries"`
	Err     error          `json:"error,omitempty"`
}

func (r listQueriesResponse) error() error { return r.Err }

func makeListQueriesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listQueriesRequest)
		queries, err := svc.ListQueries(ctx, req.ListOptions)
		if err != nil {
			return listQueriesResponse{Err: err}, nil
		}

		resp := listQueriesResponse{Queries: []kolide.Query{}}
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
	payload kolide.QueryPayload
}

type createQueryResponse struct {
	Query *kolide.Query `json:"query,omitempty"`
	Err   error         `json:"error,omitempty"`
}

func (r createQueryResponse) error() error { return r.Err }

func makeCreateQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
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
	payload kolide.QueryPayload
}

type modifyQueryResponse struct {
	Query *kolide.Query `json:"query,omitempty"`
	Err   error         `json:"error,omitempty"`
}

func (r modifyQueryResponse) error() error { return r.Err }

func makeModifyQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
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
	ID uint
}

type deleteQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryResponse) error() error { return r.Err }

func makeDeleteQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteQueryRequest)
		err := svc.DeleteQuery(ctx, req.ID)
		if err != nil {
			return deleteQueryResponse{Err: err}, nil
		}
		return deleteQueryResponse{}, nil
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

func makeDeleteQueriesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteQueriesRequest)
		deleted, err := svc.DeleteQueries(ctx, req.IDs)
		if err != nil {
			return deleteQueriesResponse{Err: err}, nil
		}
		return deleteQueriesResponse{Deleted: deleted}, nil
	}
}
