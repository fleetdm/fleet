package server

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Get Pack
////////////////////////////////////////////////////////////////////////////////

type getPackRequest struct {
	ID uint
}

type getPackResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Err      error  `json:"error,omitempty"`
}

func (r getPackResponse) error() error { return r.Err }

func makeGetPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getPackRequest)
		pack, err := svc.GetPack(ctx, req.ID)
		if err != nil {
			return getPackResponse{Err: err}, nil
		}
		return getPackResponse{
			ID:       pack.ID,
			Name:     pack.Name,
			Platform: pack.Platform,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get All Packs
////////////////////////////////////////////////////////////////////////////////

type getAllPacksResponse struct {
	Packs []getPackResponse `json:"packs"`
	Err   error             `json:"error,omitempty"`
}

func (r getAllPacksResponse) error() error { return r.Err }

func makeGetAllPacksEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		packs, err := svc.GetAllPacks(ctx)
		if err != nil {
			return getPackResponse{Err: err}, nil
		}
		var resp getAllPacksResponse
		for _, pack := range packs {
			resp.Packs = append(resp.Packs, getPackResponse{
				ID:       pack.ID,
				Name:     pack.Name,
				Platform: pack.Platform,
			})
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Create Pack
////////////////////////////////////////////////////////////////////////////////

type createPackRequest struct {
	payload kolide.PackPayload
}

type createPackResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Err      error  `json:"error,omitempty"`
}

func (r createPackResponse) error() error { return r.Err }

func makeCreatePackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPackRequest)
		pack, err := svc.NewPack(ctx, req.payload)
		if err != nil {
			return createPackResponse{Err: err}, nil
		}
		return createPackResponse{
			ID:       pack.ID,
			Name:     pack.Name,
			Platform: pack.Platform,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Pack
////////////////////////////////////////////////////////////////////////////////

type modifyPackRequest struct {
	ID      uint
	payload kolide.PackPayload
}

type modifyPackResponse struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Err      error  `json:"error,omitempty"`
}

func (r modifyPackResponse) error() error { return r.Err }

func makeModifyPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyPackRequest)
		pack, err := svc.ModifyPack(ctx, req.ID, req.payload)
		if err != nil {
			return modifyPackResponse{Err: err}, nil
		}
		return modifyPackResponse{
			ID:       pack.ID,
			Name:     pack.Name,
			Platform: pack.Platform,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Pack
////////////////////////////////////////////////////////////////////////////////

type deletePackRequest struct {
	ID uint
}

type deletePackResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deletePackResponse) error() error { return r.Err }

func makeDeletePackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deletePackRequest)
		err := svc.DeletePack(ctx, req.ID)
		if err != nil {
			return deletePackResponse{Err: err}, nil
		}
		return deletePackResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Add Query To Pack
////////////////////////////////////////////////////////////////////////////////

type addQueryToPackRequest struct {
	QueryID uint
	PackID  uint
}

type addQueryToPackResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addQueryToPackResponse) error() error { return r.Err }

func makeAddQueryToPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addQueryToPackRequest)
		err := svc.AddQueryToPack(ctx, req.QueryID, req.PackID)
		if err != nil {
			return addQueryToPackResponse{Err: err}, nil
		}
		return addQueryToPackResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Queries In Pack
////////////////////////////////////////////////////////////////////////////////

type getQueriesInPackRequest struct {
	ID uint
}

type getQueriesInPackResponse struct {
	Queries []getQueryResponse
	Err     error `json:"error,omitempty"`
}

func (r getQueriesInPackResponse) error() error { return r.Err }

func makeGetQueriesInPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getQueriesInPackRequest)
		queries, err := svc.GetQueriesInPack(ctx, req.ID)
		if err != nil {
			return getQueriesInPackResponse{Err: err}, nil
		}

		var resp getQueriesInPackResponse
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
// Delete Query From Pack
////////////////////////////////////////////////////////////////////////////////

type deleteQueryFromPackRequest struct {
	QueryID uint
	PackID  uint
}

type deleteQueryFromPackResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryFromPackResponse) error() error { return r.Err }

func makeDeleteQueryFromPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteQueryFromPackRequest)
		err := svc.RemoveQueryFromPack(ctx, req.QueryID, req.PackID)
		if err != nil {
			return deleteQueryFromPackResponse{Err: err}, nil
		}
		return deleteQueryFromPackResponse{}, nil
	}
}
