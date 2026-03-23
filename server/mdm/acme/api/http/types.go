// Package http provides HTTP request and response types for the ACME bounded context.
// These types are used exclusively by the ACME endpoint handler.
package http

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
)

type GetNewNonceRequest struct {
	// HTTPMethod used to make this request, populated by the parse custom URL
	// tag function of the ACME bounded context, which is one of the only ways
	// with our framework to access the *http.Request value.
	HTTPMethod string `url:"http_method"`
	Identifier string `url:"identifier"`
}

type GetNewNonceResponse struct {
	HTTPMethod string
	Nonce      string
	Err        error `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r GetNewNonceResponse) Error() error { return r.Err }

func (r GetNewNonceResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Replay-Nonce", r.Nonce)
	w.Header().Set("Cache-Control", "no-store")

	if r.HTTPMethod == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	// for GET/POST-as-GET, return 204
	w.WriteHeader(http.StatusNoContent)
}

type GetDirectoryRequest struct {
	Identifier string `url:"identifier"`
}

type GetDirectoryResponse struct {
	*types.Directory
	Err error `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r GetDirectoryResponse) Error() error { return r.Err }
