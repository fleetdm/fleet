package service

import (
	"context"
	"crypto/x509"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// HTTP paths for the Apple Platform SSO endpoints. All but the AASA path live
// under /api/mdm/apple/psso and are registered on the unauthenticated
// endpointer (see handler.go); auth is protocol-level (signed JWTs verified
// against registered device keys). The AASA document must be served at the
// /.well-known path(Apple requirement), so it stays on the root *http.ServeMux
// (see registerPSSO).
const (
	pssoNoncePath        = "/api/mdm/apple/psso/nonce"
	pssoRegistrationPath = "/api/mdm/apple/psso/registration"
	pssoTokenPath        = "/api/mdm/apple/psso/token" //nolint:gosec // G101 false positive, this is a URL path
	pssoJWKSPath         = "/api/mdm/apple/psso/jwks"
	pssoAASAPath         = "/.well-known/apple-app-site-association"
)

// pssoContentTypeLoginResponse is the Content-Type Apple's PSSO framework
// expects on token endpoint responses.
const pssoContentTypeLoginResponse = "application/platformsso-login-response+jwt"

////////////////////////////////////////////////////////////////////////////////
// POST /api/mdm/apple/psso/nonce
////////////////////////////////////////////////////////////////////////////////

type pssoNonceRequest struct{}

// DecodeBody drains and discards the request body. Apple's AppSSOAgent POSTs a
// urlencoded grant_type=srv_challenge form to the nonce endpoint, but Fleet
// needs nothing from it — it just mints a nonce. Draining (rather than leaving
// it unread) keeps the connection reusable; the reader is already size-limited
// by the endpointer. The method must exist so the endpoint framework routes the
// form body here instead of falling through to JSON decoding, which rejects the
// form as malformed.
func (pssoNonceRequest) DecodeBody(_ context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	_, _ = io.Copy(io.Discard, r)
	return nil
}

type pssoNonceResponse struct {
	// Nonce is PascalCase on the wire: Apple's AppSSOAgent consumes this
	// response directly and expects the capitalized key.
	Nonce string `json:"Nonce"`
	Err   error  `json:"error,omitempty"`
}

func (r pssoNonceResponse) Error() error { return r.Err }

func pssoNonceEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	nonce, err := svc.PSSONonce(ctx)
	if err != nil {
		return pssoNonceResponse{Err: err}, nil
	}
	return pssoNonceResponse{Nonce: nonce}, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /api/mdm/apple/psso/registration
////////////////////////////////////////////////////////////////////////////////

type pssoRegistrationRequest struct {
	fleet.PSSODeviceRegistrationRequest
}

// DecodeBody parses the urlencoded form the extension POSTs. The reader is
// already capped by the endpointer's request body size limit.
func (req *pssoRegistrationRequest) DecodeBody(ctx context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	form, err := parseURLEncodedForm(ctx, r)
	if err != nil {
		return err
	}
	req.DeviceUUID = form.Get("device_uuid")
	req.DeviceSigningKey = form.Get("device_signing_key")
	req.DeviceEncryptionKey = form.Get("device_encryption_key")
	req.SigningKeyID = form.Get("signing_key_id")
	req.EncryptionKeyID = form.Get("encryption_key_id")
	return nil
}

type pssoRegistrationResponse struct {
	Err error `json:"error,omitempty"`
}

func (r pssoRegistrationResponse) Error() error { return r.Err }

func (r pssoRegistrationResponse) Status() int { return http.StatusNoContent }

func pssoRegistrationEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*pssoRegistrationRequest)
	if err := svc.PSSORegisterDevice(ctx, req.PSSODeviceRegistrationRequest); err != nil {
		return pssoRegistrationResponse{Err: err}, nil
	}
	return pssoRegistrationResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /api/mdm/apple/psso/token
////////////////////////////////////////////////////////////////////////////////

type pssoTokenRequest struct {
	Assertion string
}

// DecodeBody parses the OAuth jwt-bearer-style urlencoded form whose
// `assertion` field holds the compact JWS signed by the device. The JWT must
// be extracted from the form, not read from the raw body.
func (req *pssoTokenRequest) DecodeBody(ctx context.Context, r io.Reader, _ url.Values, _ []*x509.Certificate) error {
	form, err := parseURLEncodedForm(ctx, r)
	if err != nil {
		return err
	}
	req.Assertion = form.Get("assertion")
	if req.Assertion == "" {
		return &fleet.BadRequestError{Message: "psso token: missing assertion"}
	}
	return nil
}

type pssoTokenResponse struct {
	Err error `json:"error,omitempty"`
	jwe []byte
}

func (r pssoTokenResponse) Error() error { return r.Err }

func (r pssoTokenResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", pssoContentTypeLoginResponse)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if n, err := w.Write(r.jwe); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func pssoTokenEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*pssoTokenRequest)
	out, err := svc.PSSOToken(ctx, []byte(req.Assertion))
	if err != nil {
		return pssoTokenResponse{Err: err}, nil
	}
	return pssoTokenResponse{jwe: out}, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /api/mdm/apple/psso/jwks
////////////////////////////////////////////////////////////////////////////////

type pssoJWKSRequest struct{}

type pssoJWKSResponse struct {
	Err  error `json:"error,omitempty"`
	body []byte
}

func (r pssoJWKSResponse) Error() error { return r.Err }

func (r pssoJWKSResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/jwk-set+json")
	if n, err := w.Write(r.body); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func pssoJWKSEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	body, err := svc.PSSOJWKS(ctx)
	if err != nil {
		return pssoJWKSResponse{Err: err}, nil
	}
	return pssoJWKSResponse{body: body}, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /.well-known/apple-app-site-association
////////////////////////////////////////////////////////////////////////////////

// pssoAASAHandler serves the Apple App Site Association JSON Apple's CDN
// fetches to validate the extension's `authsrv:` entitlement against this
// hostname. It stays a raw root-mux handler because the path is fixed by
// Apple's spec and can't live under /api.
func pssoAASAHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := svc.PSSOAASA(ctx)
		if err != nil {
			encodeError(ctx, err, w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body)
	})
}

// parseURLEncodedForm reads an x-www-form-urlencoded body from an
// already-size-limited reader.
func parseURLEncodedForm(ctx context.Context, r io.Reader) (url.Values, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read psso form body")
	}
	form, err := url.ParseQuery(string(raw))
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "invalid urlencoded form body", InternalErr: err}
	}
	return form, nil
}

// ----- core-side service-method stubs --------------------------------------
//
// All PSSO business logic lives in ee/server/service. The core stubs below
// return fleet.ErrMissingLicense so unlicensed Fleet deployments respond with
// a well-formed error instead of 404. The ee implementation overrides these
// methods on the embedded core Service.

func (svc *Service) PSSONonce(ctx context.Context) (string, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return "", fleet.ErrMissingLicense
}

func (svc *Service) PSSORegisterDevice(ctx context.Context, _ fleet.PSSODeviceRegistrationRequest) error {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

func (svc *Service) PSSOToken(ctx context.Context, _ []byte) ([]byte, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) PSSOJWKS(ctx context.Context) ([]byte, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) PSSOAASA(ctx context.Context) ([]byte, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}
