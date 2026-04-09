package service

import (
	"context"
	"fmt"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
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
	opts = append(opts, kithttp.ServerErrorEncoder(acmeErrorEncoder))
	ae := newEndpointerWithNoAuth(svc, authMiddleware, opts, r)
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

	// POST-as-GET for order endpoint, as per RFC.
	ae.POST("/api/mdm/acme/{identifier}/orders/{order_id}", getOrderEndpoint, api_http.GetOrderRequest{})
	// POST-as-GET for list orders endpoint, as per RFC.
	ae.POST("/api/mdm/acme/{identifier}/accounts/{account_id}/orders", listOrdersEndpoint, api_http.ListOrdersRequest{})
	// POST-as-GET for download certificate endpoint, as per RFC.
	ae.POST("/api/mdm/acme/{identifier}/orders/{order_id}/certificate", getCertificateEndpoint, api_http.GetCertificateRequest{})

	ae.POST("/api/mdm/acme/{identifier}/authorizations/{authorization_id}", getAuthorizationEndpoint, api_http.GetAuthorizationRequest{})
	ae.POST("/api/mdm/acme/{identifier}/challenges/{challenge_id}", getChallengeEndpoint, api_http.DoChallengeRequest{})
	ae.POST("/api/mdm/acme/{identifier}/orders/{order_id}/finalize", finalizeOrderEndpoint, api_http.FinalizeOrderRequestContainer{})
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

	partialOrder := &types.Order{
		Identifiers: newOrderRequest.Identifiers,
		NotBefore:   newOrderRequest.NotBefore,
		NotAfter:    newOrderRequest.NotAfter,
	}
	orderResp, err := svc.CreateOrder(ctx, newOrderRequest.Enrollment, newOrderRequest.Account, partialOrder)
	if err != nil {
		return &api_http.CreateNewOrderResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	return &api_http.CreateNewOrderResponse{
		Nonces:        svc.NoncesStore(),
		OrderResponse: orderResp,
	}
}

// getOrderEndpoint handles POST-as-GET /api/mdm/acme/{identifier}/orders/{id} requests.
func getOrderEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetOrderRequest)
	req.PostAsGet = true
	orderRequest := &api_http.GetOrderDecodedRequest{OrderID: req.OrderID}
	err := svc.AuthenticateMessageFromAccount(ctx, &req.JWSRequestContainer, orderRequest)
	if err != nil {
		return &api_http.GetOrderResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	orderResp, err := svc.GetOrder(ctx, orderRequest.Enrollment, orderRequest.Account, orderRequest.OrderID)
	if err != nil {
		return &api_http.GetOrderResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	return &api_http.GetOrderResponse{
		Nonces:        svc.NoncesStore(),
		OrderResponse: orderResp,
	}
}

// listOrdersEndpoint handles POST-as-GET /api/mdm/acme/{identifier}/accounts/{id}/orders requests.
func listOrdersEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.ListOrdersRequest)
	req.PostAsGet = true
	ordersRequest := &api_http.ListOrdersDecodedRequest{AccountID: req.AccountID}
	err := svc.AuthenticateMessageFromAccount(ctx, &req.JWSRequestContainer, ordersRequest)
	if err != nil {
		return &api_http.ListOrdersResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	urls, err := svc.ListAccountOrders(ctx, req.Identifier, ordersRequest.Account)
	if err != nil {
		return &api_http.ListOrdersResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	return &api_http.ListOrdersResponse{
		Nonces: svc.NoncesStore(),
		Orders: urls,
	}
}

// getCertificateEndpoint handles POST-as-GET /api/mdm/acme/{identifier}/orders/{id}/certificate requests.
func getCertificateEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetCertificateRequest)
	req.PostAsGet = true
	certReq := &api_http.GetCertificateDecodedRequest{OrderID: req.OrderID}
	err := svc.AuthenticateMessageFromAccount(ctx, &req.JWSRequestContainer, certReq)
	if err != nil {
		return &api_http.GetCertificateResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	cert, err := svc.GetCertificate(ctx, certReq.Account.ID, certReq.OrderID)
	if err != nil {
		return &api_http.GetCertificateResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	return &api_http.GetCertificateResponse{
		Certificate: cert,
		Nonces:      svc.NoncesStore(),
	}
}

// getAuthorizationEndpoint handles POST /api/mdm/acme/{identifier}/authz/{authorization} requests.
func getAuthorizationEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetAuthorizationRequest)
	req.PostAsGet = true
	authzReq := &api_http.GetAuthorizationDecodedRequest{AuthorizationID: req.AuthorizationID}
	err := svc.AuthenticateMessageFromAccount(ctx, &req.JWSRequestContainer, authzReq)
	if err != nil {
		return &api_http.GetAuthorizationResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	authzResp, err := svc.GetAuthorization(ctx, authzReq.Enrollment, authzReq.Account, authzReq.AuthorizationID)
	if err != nil {
		return &api_http.GetAuthorizationResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	return &api_http.GetAuthorizationResponse{
		AuthorizationResponse: authzResp,
		Nonces:                svc.NoncesStore(),
	}
}

func getChallengeEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.DoChallengeRequest)
	decodedReq := &api_http.DoChallengeDecodedRequest{ChallengeID: req.ChallengeID}
	err := svc.AuthenticateMessageFromAccount(ctx, &req.JWSRequestContainer, decodedReq)
	if err != nil {
		return &api_http.DoChallengeResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	if decodedReq.AttestError != "" {
		return &api_http.DoChallengeResponse{
			Err:    types.UnauthorizedError(fmt.Sprintf("Attestation failure: %s", decodedReq.AttestError)),
			Nonces: svc.NoncesStore(),
		}
	}

	challengeResp, err := svc.ValidateChallenge(ctx, decodedReq.Enrollment, decodedReq.Account, decodedReq.ChallengeID, decodedReq.AttestationObject)
	if err != nil {
		return &api_http.DoChallengeResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	return &api_http.DoChallengeResponse{
		ChallengeResponse: challengeResp,
		Nonces:            svc.NoncesStore(),
	}
}

// finalizeOrderEndpoint handles POST /api/mdm/acme/{identifier}/orders/{order_id}/finalize requests.
func finalizeOrderEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.FinalizeOrderRequestContainer)
	finalizeOrderRequest := &api_http.FinalizeOrderRequest{OrderID: req.OrderID}
	err := svc.AuthenticateMessageFromAccount(ctx, &req.JWSRequestContainer, finalizeOrderRequest)
	if err != nil {
		return &api_http.FinalizeOrderResponse{Err: err, Nonces: svc.NoncesStore()}
	}
	order, err := svc.FinalizeOrder(ctx, finalizeOrderRequest.Enrollment, finalizeOrderRequest.Account, finalizeOrderRequest.OrderID, finalizeOrderRequest.CertificateSigningRequest)
	if err != nil {
		return &api_http.FinalizeOrderResponse{Err: err, Nonces: svc.NoncesStore()}
	}

	return &api_http.FinalizeOrderResponse{OrderResponse: order, Err: err, Nonces: svc.NoncesStore()}
}
