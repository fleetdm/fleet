package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
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
	opts = append(opts, kithttp.ServerErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) {
		fmt.Println(">>>>> ERROR ENCODING RESPONSE:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	ae := newEndpointerWithNoAuth(svc, opts, r)

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
}

// getNewNonceEndpoint handles HEAD/GET /api/mdm/acme/{identifier}/new_nonce requests.
func getNewNonceEndpoint(ctx context.Context, request any, svc api.Service) platform_http.Errorer {
	req := request.(*api_http.GetNewNonceRequest)
	nonce, err := svc.NewNonce(ctx, req.Identifier)
	if err != nil {
		fmt.Println(">>>>>> ERROR GETTING NONCE:", err)
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
