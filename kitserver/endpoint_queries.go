package kitserver

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Get Query
////////////////////////////////////////////////////////////////////////////////

type getQueryRequest struct {
	ID uint
}

type getQueryResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Interval     uint   `json:"interval"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	Err          error  `json:"error, omitempty"`
}

func (r getQueryResponse) error() error { return r.Err }

func makeGetQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getQueryRequest)
		query, err := svc.GetQuery(ctx, req.ID)
		if err != nil {
			return getQueryResponse{Err: err}, nil
		}
		return getQueryResponse{
			ID:           query.ID,
			Name:         query.Name,
			Query:        query.Query,
			Interval:     query.Interval,
			Snapshot:     query.Snapshot,
			Differential: query.Differential,
			Platform:     query.Platform,
			Version:      query.Version,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get All Queries
////////////////////////////////////////////////////////////////////////////////

type getAllQueriesResponse struct {
	Queries []getQueryResponse `json:"queries"`
	Err     error              `json:"error, omitempty"`
}

func (r getAllQueriesResponse) error() error { return r.Err }

func makeGetAllQueriesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		queries, err := svc.GetAllQueries(ctx)
		if err != nil {
			return nil, err
		}
		var resp getAllQueriesResponse
		for _, query := range queries {
			resp.Queries = append(resp.Queries, getQueryResponse{
				ID:           query.ID,
				Name:         query.Name,
				Query:        query.Query,
				Interval:     query.Interval,
				Snapshot:     query.Snapshot,
				Differential: query.Differential,
				Platform:     query.Platform,
				Version:      query.Version,
			})
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
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Interval     uint   `json:"interval"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	Err          error  `json:"error, omitempty"`
}

func (r createQueryResponse) error() error { return r.Err }

func makeCreateQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createQueryRequest)
		query, err := svc.NewQuery(ctx, req.payload)
		if err != nil {
			return createQueryResponse{Err: err}, nil
		}
		return createQueryResponse{
			ID:           query.ID,
			Name:         query.Name,
			Query:        query.Query,
			Interval:     query.Interval,
			Snapshot:     query.Snapshot,
			Differential: query.Differential,
			Platform:     query.Platform,
			Version:      query.Version,
		}, nil
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
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Interval     uint   `json:"interval"`
	Snapshot     bool   `json:"snapshot"`
	Differential bool   `json:"differential"`
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	Err          error  `json:"error, omitempty"`
}

func (r modifyQueryResponse) error() error { return r.Err }

func makeModifyQueryEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyQueryRequest)
		query, err := svc.ModifyQuery(ctx, req.ID, req.payload)
		if err != nil {
			return modifyQueryResponse{Err: err}, nil
		}
		return modifyQueryResponse{
			ID:           query.ID,
			Name:         query.Name,
			Query:        query.Query,
			Interval:     query.Interval,
			Snapshot:     query.Snapshot,
			Differential: query.Differential,
			Platform:     query.Platform,
			Version:      query.Version,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Query
////////////////////////////////////////////////////////////////////////////////

type deleteQueryRequest struct {
	ID uint
}

type deleteQueryResponse struct {
	Err error `json:"error, omitempty"`
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
