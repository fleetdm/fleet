package service

import (
	"context"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
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
func GetRoutes(svc api.Service) eu.HandlerRoutesFunc {
	return func(r *mux.Router, opts []kithttp.ServerOption) {
		attachFleetAPIRoutes(r, svc, opts)
	}
}

func attachFleetAPIRoutes(r *mux.Router, svc api.Service, opts []kithttp.ServerOption) {
	ae := newEndpointerWithNoAuth(svc, opts, r)
	// ACME endpoints use path identifier and JWS authn/z, so we use a middleware to mark
	// the standard Fleet auth as skipped/done so the endpoints don't return a Forbidden
	// error due to no standard auth done.
	ae = ae.WithCustomMiddlewareAfterAuth(skipStandardFleetAuth())

	// must support HEAD, GET and POST-as-GET for new_nonce as per
	// https://datatracker.ietf.org/doc/html/rfc8555/#section-6.3 and
	// https://datatracker.ietf.org/doc/html/rfc8555/#section-7.2
	ae.GET("/api/mdm/acme/{identifier}/new_nonce", getNewNonceEndpoint, api_http.GetNewNonceRequest{})
	ae.HEAD("/api/mdm/acme/{identifier}/new_nonce", getNewNonceEndpoint, api_http.GetNewNonceRequest{})
	ae.POST("/api/mdm/acme/{identifier}/new_nonce", getNewNonceEndpoint, api_http.GetNewNonceRequest{})

	// must support GET and POST-as-GET for directory as per
	// https://datatracker.ietf.org/doc/html/rfc8555/#section-6.3 and
	// https://datatracker.ietf.org/doc/html/rfc8555/#section-7.1.1
	ae.GET("/api/mdm/acme/{identifier}/directory", getDirectoryEndpoint, api_http.GetDirectoryRequest{})
	ae.POST("/api/mdm/acme/{identifier}/directory", getDirectoryEndpoint, api_http.GetDirectoryRequest{})

	ae.POST("/api/mdm/acme/{identifier}/new_account", createAccountEndpoint, api_http.JWSRequestContainer{})
	ae.POST("/api/mdm/acme/{identifier}/new_order", createOrderEndpoint, api_http.JWSRequestContainer{})
}

func skipStandardFleetAuth() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			if az, ok := authz_ctx.FromContext(ctx); ok {
				az.SetChecked()
			}
			return next(ctx, req)
		}
	}
}

// getNewNonceEndpoint handles HEAD/GET/POST /api/mdm/acme/{identifier}/new_nonce requests.
func getNewNonceEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetNewNonceRequest)
	err := svc.NewNonce(ctx, req.Identifier)
	if err != nil {
		return &api_http.GetNewNonceResponse{Err: err}
	}
	return &api_http.GetNewNonceResponse{
		HTTPMethod: req.HTTPMethod,
		Nonces:     svc.NoncesStore(),
	}
}

// getDirectoryEndpoint handles GET/POST /api/mdm/acme/{identifier}/directory requests.
func getDirectoryEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetDirectoryRequest)
	dir, err := svc.GetDirectory(ctx, req.Identifier)
	if err != nil {
		return api_http.GetDirectoryResponse{Err: err}
	}
	return api_http.GetDirectoryResponse{Directory: dir}
}

// createAccountEndpoint handles POST /api/mdm/acme/{identifier}/new_account requests.
func createAccountEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.JWSRequestContainer)
	newAccountRequest := &api_http.CreateNewAccountRequest{}
	err := svc.AuthenticateNewAccountMessage(ctx, req, newAccountRequest)
	if err != nil {
		return &api_http.CreateNewAccountResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	accountResp, err := svc.CreateAccount(ctx, req.Identifier, newAccountRequest.Enrollment.ID, *newAccountRequest.JSONWebKey, newAccountRequest.OnlyReturnExisting)
	if err != nil {
		return &api_http.CreateNewAccountResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	return &api_http.CreateNewAccountResponse{
		Nonces:          svc.NoncesStore(),
		AccountResponse: accountResp,
	}
}

// createOrderEndpoint handles POST /api/mdm/acme/{identifier}/new_order requests.
func createOrderEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.JWSRequestContainer)
	newOrderRequest := &api_http.CreateNewOrderRequest{}
	err := svc.AuthenticateMessageFromAccount(ctx, req, newOrderRequest)
	if err != nil {
		return &api_http.CreateNewOrderResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	_ = newOrderRequest
	panic("unimplemented")
}
