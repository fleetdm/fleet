// Package http provides HTTP request and response types for the ACME bounded context.
// These types are used exclusively by the ACME endpoint handler.
package http

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/redis_nonces_store"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"go.step.sm/crypto/jose"
)

func generateAndRenderNonce(ctx context.Context, nonces *redis_nonces_store.RedisNoncesStore, w http.ResponseWriter) error {
	nonce := types.CreateNonceEncodedForHeader()
	if err := nonces.Store(ctx, nonce, redis_nonces_store.DefaultNonceExpiration); err != nil {
		return err
	}
	w.Header().Set("Replay-Nonce", nonce)
	w.Header().Set("Cache-Control", "no-store")
	return nil
}

type GetNewNonceRequest struct {
	// HTTPMethod used to make this request, populated by the parse custom URL
	// tag function of the ACME bounded context, which is one of the only ways
	// with our framework to access the *http.Request value.
	HTTPMethod string `url:"http_method"`
	Identifier string `url:"identifier"`
}

type GetNewNonceResponse struct {
	HTTPMethod string                               `json:"-"`
	Err        error                                `json:"error,omitempty"`
	Nonces     *redis_nonces_store.RedisNoncesStore `json:"-"`
}

// Error implements the platform_http.Errorer interface.
func (r *GetNewNonceResponse) Error() error { return r.Err }

// BeforeRender implements the beforeRenderer interface.
func (r *GetNewNonceResponse) BeforeRender(ctx context.Context, w http.ResponseWriter) {
	// only generate a new nonce if there are no error for this endpoint.
	if r.Err == nil {
		if err := generateAndRenderNonce(ctx, r.Nonces, w); err != nil {
			r.Err = err
			return
		}
	}
}

func (r *GetNewNonceResponse) Status() int {
	if r.HTTPMethod == http.MethodHead {
		return http.StatusOK
	}
	// for GET/POST-as-GET, return 204
	return http.StatusNoContent
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
	Enrollment *types.Enrollment `json:"-"`
	JSONWebKey *jose.JSONWebKey  `json:"-"`

	// OnlyReturnExisting indicates that no new account should be created but the
	// existing account for this key should be returned if it exists. This is the
	// only actual parameter read from the payload of the JWS request
	OnlyReturnExisting bool `json:"onlyReturnExisting"`
}

type CreateNewAccountResponse struct {
	*types.AccountResponse
	Err    error                                `json:"error,omitempty"`
	Nonces *redis_nonces_store.RedisNoncesStore `json:"-"`
}

// BeforeRender implements the beforeRenderer interface.
func (r *CreateNewAccountResponse) BeforeRender(ctx context.Context, w http.ResponseWriter) {
	// only generate a new nonce if there is no error or the error is due to a client error
	// other than "enrollment not found" (in which case the client has no reason to retry).
	if r.Err != nil {
		var acmeErr *types.ACMEError
		if !errors.As(r.Err, &acmeErr) || !acmeErr.ShouldReturnNonce() {
			return
		}
	}
	if err := generateAndRenderNonce(ctx, r.Nonces, w); err != nil {
		r.Err = err
		return
	}
}

// Status implements the statuser interface.
func (r *CreateNewAccountResponse) Status() int {
	if r.DidCreate {
		return http.StatusCreated
	}
	return http.StatusOK
}

// Error implements the platform_http.Errorer interface.
func (r *CreateNewAccountResponse) Error() error { return r.Err }

type CreateNewOrderRequest struct {
	types.AccountAuthenticatedRequestBase
	Identifiers []types.Identifier `json:"identifiers"`
}

type CreateNewOrderResponse struct {
	// TODO(mna): must return the JSON order
	Err    error                                `json:"error,omitempty"`
	Nonces *redis_nonces_store.RedisNoncesStore `json:"-"`
}

func (r *CreateNewOrderResponse) BeforeRender(ctx context.Context, w http.ResponseWriter) {
	// only generate a new nonce if there is no error or the error is due to a client error
	// other than "enrollment not found" (in which case the client has no reason to retry).
	if r.Err != nil {
		var acmeErr *types.ACMEError
		if !errors.As(r.Err, &acmeErr) || !acmeErr.ShouldReturnNonce() {
			return
		}
	}
	if err := generateAndRenderNonce(ctx, r.Nonces, w); err != nil {
		r.Err = err
		return
	}
}

// Error implements the platform_http.Errorer interface.
func (r *CreateNewOrderResponse) Error() error { return r.Err }

// JWS Request container is a container for doing basic decoding and validation operations common to all
// authenticated ACME requests, which come in the form of a JWS in flattened serialization syntax. This is
// parsed into a jose.JSONWebSignature with some basic validation done on it and then the downstream
// handler can use the included JWK or KeyID to do further authentication and authorization as needed.
type JWSRequestContainer struct {
	JWS          jose.JSONWebSignature
	JWSHeaderURL string

	Key        *jose.JSONWebKey
	KeyID      *string
	Identifier string `url:"identifier"`
	HTTPPath   string `url:"http_path"`
}

func (req *JWSRequestContainer) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	jwsBytes, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading soap mdm request")
	}
	// Note: req.Identifier is set from the mux path variable by the framework
	// (via the `url:"identifier"` struct tag), so we don't set it here.
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

	req.JWS = *jws

	if jws.Signatures[0].Protected.JSONWebKey != nil {
		req.Key = jws.Signatures[0].Protected.JSONWebKey
	}
	// KeyID should be the account URL
	if jws.Signatures[0].Protected.KeyID != "" {
		req.KeyID = &jws.Signatures[0].Protected.KeyID
	}

	// JWS must have a url field in the protected header:
	// https://datatracker.ietf.org/doc/html/rfc8555/#section-6.2
	headerURL, ok := jws.Signatures[0].Protected.ExtraHeaders["url"].(string)
	if !ok || headerURL == "" {
		return ctxerr.New(ctx, "jws must have a url in the protected header")
	}
	req.JWSHeaderURL = headerURL

	return nil
}
