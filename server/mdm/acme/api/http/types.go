// Package http provides HTTP request and response types for the ACME bounded context.
// These types are used exclusively by the ACME endpoint handler.
package http

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
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

type CreateNewAccountRequest struct {
	Enrollment *types.ACMEEnrollment `json:"-"`
	JSONWebKey *jose.JSONWebKey      `json:"-"`

	// OnlyReturnExisting indicates that no new account should be created but the
	// existing account for this key should be returned if it exists. This is the
	// only actual parameter read from the payload of the JWS request
	OnlyReturnExisting bool `json:"onlyReturnExisting"`
}

type CreateNewAccountResponse struct {
	Status string `json:"status"`
	Orders string `json:"orders"`
	// TODO(mna): maybe no Nonce field, always create it in the response? (unless 500, maybe)
	Nonce    string `json:"-"`
	Location string `json:"-"`
	Err      error  `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r CreateNewAccountResponse) Error() error { return r.Err }

type CreateNewOrderRequest struct {
	types.AccountAuthenticatedRequestBase

	Identifiers []types.Identifier `json:"identifiers"`
}

type CreateNewOrderResponse struct {
	// TODO(mna): must return the JSON order
	Nonce string
	Err   error `json:"error,omitempty"`
}

// Error implements the platform_http.Errorer interface.
func (r CreateNewOrderResponse) Error() error { return r.Err }

// JWS Request container is a container for doing basic decoding and validation operations common to all
// authenticated ACME requests, which come in the form of a JWS in flattened serialization syntax. This is
// parsed into a jose.JSONWebSignature with some basic validation done on it and then the downstream
// handler can use the included JWK or KeyID to do further authentication and authorization as needed.
type JWSRequestContainer struct {
	JWS jose.JSONWebSignature

	Key        *jose.JSONWebKey
	KeyID      *string
	Identifier string `url:"identifier"`
}

func (req *JWSRequestContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	jwsBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading soap mdm request")
	}
	req.Identifier = u.Get("identifier")
	jws, err := jose.ParseJWS(string(jwsBytes))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing jws")
	}
	// The JWS must have exactly one signature because ACME uses the "flattened" JWS JSON serialization
	if len(jws.Signatures) == 0 {
		return ctxerr.New(ctx, "jws must have a signature")
	}
	if len(jws.Signatures) > 1 {
		return ctxerr.New(ctx, "jws must have only one signature")
	}
	// All ACME requests should have either a JWK in the header or a KeyID that points to an account, but never both
	if jws.Signatures[0].Protected.JSONWebKey == nil && jws.Signatures[0].Protected.KeyID == "" {
		return ctxerr.New(ctx, "jws must have a key or key ID in the protected header")
	}

	if jws.Signatures[0].Protected.JSONWebKey != nil {
		req.Key = jws.Signatures[0].Protected.JSONWebKey
	}
	// KeyID should be the account URL
	if jws.Signatures[0].Protected.KeyID != "" {
		req.KeyID = &jws.Signatures[0].Protected.KeyID
	}
	return nil
}
