package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// GetRoutes returns a function that registers ACME routes on the router using the provided
// authMiddleware.
func GetRoutes(svc api.Service, authMiddleware endpoint.Middleware) eu.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, authMiddleware, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc api.Service, authMiddleware endpoint.Middleware, opts []kithttp.ServerOption) {
	ae := newEndpointerWithAuth(svc, authMiddleware, opts, r)

	// TODO(mna): double-check that it works with HEAD (I think we handle it automatically for GET)
	ae.GET("/api/mdm/acme/{identifier}/new_nonce", getNewNonceEndpoint, api_http.GetNewNonceRequest{})
	ae.GET("/api/mdm/acme/{identifier}/directory", getDirectoryEndpoint, api_http.GetDirectoryRequest{})
}

// getNewNonceEndpoint handles HEAD/GET /api/mdm/acme/{identifier}/new_nonce requests.
func getNewNonceEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetNewNonceRequest)
	nonce, err := svc.NewNonce(ctx, req.Identifier)
	if err != nil {
		return api_http.GetNewNonceResponse{Err: err}
	}
	return api_http.GetNewNonceResponse{
		HTTPMethod: req.HTTPMethod,
		Nonce:      nonce,
	}
}

// getDirectoryEndpoint handles GET /api/mdm/acme/{identifier}/directory requests.
func getDirectoryEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetDirectoryRequest)
	dir, err := svc.GetDirectory(ctx, req.Identifier)
	if err != nil {
		return api_http.GetDirectoryResponse{Err: err}
	}
	return api_http.GetDirectoryResponse{Directory: dir}
}
