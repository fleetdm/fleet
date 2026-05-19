package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// HTTP paths for the Apple Platform SSO IdP endpoints. These follow the SCEP
// path convention (no /api/ or /v1/ prefix) because they're raw protocol
// endpoints the Mac extension talks to directly, not user-facing API.
const (
	pssoNoncePath    = "/mdm/apple/psso/nonce"
	pssoRegisterPath = "/mdm/apple/psso/register"
	pssoTokenPath    = "/mdm/apple/psso/token"
	pssoJWKSPath     = "/.well-known/jwks.json"
	pssoAASAPath     = "/.well-known/apple-app-site-association"
)

// pssoContentTypeLoginResponse is the Content-Type Apple's PSSO framework
// expects on token endpoint responses.
const pssoContentTypeLoginResponse = "application/platformsso-login-response+jwt"

// ----- /mdm/apple/psso/nonce ------------------------------------------------

type pssoNonceRequest struct{}

type pssoNonceResponse struct {
	Nonce string `json:"nonce"`
	Err   error  `json:"error,omitempty"`
}

func (r pssoNonceResponse) Error() error { return r.Err }

func pssoNonceEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	nonce, err := svc.PSSONonce(ctx)
	if err != nil {
		return pssoNonceResponse{Err: err}, err
	}
	return pssoNonceResponse{Nonce: nonce}, nil
}

func (svc *Service) PSSONonce(ctx context.Context) (string, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return "", fleet.ErrMissingLicense
}

// ----- /mdm/apple/psso/register (GET + POST) --------------------------------

type pssoRegisterBeginRequest struct{}

type pssoRegisterBeginResponse struct {
	redirectURL string
	Err         error `json:"error,omitempty"`
}

func (r pssoRegisterBeginResponse) Error() error { return r.Err }

// HijackRender emits a 302 redirect to the upstream IdP's OAuth flow.
func (r pssoRegisterBeginResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	w.Header().Set("Location", r.redirectURL)
	w.WriteHeader(http.StatusFound)
}

func pssoRegisterBeginEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	redirectURL, err := svc.PSSORegisterBegin(ctx)
	if err != nil {
		return pssoRegisterBeginResponse{Err: err}, err
	}
	return pssoRegisterBeginResponse{redirectURL: redirectURL}, nil
}

func (svc *Service) PSSORegisterBegin(ctx context.Context) (string, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return "", fleet.ErrMissingLicense
}

type pssoRegisterCompleteRequest struct {
	fleet.PSSORegisterRequest
}

// DecodeRequest reads the registration payload from either a form-encoded body
// or query string — the extension posts query-style key/value pairs.
func (pssoRegisterCompleteRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse psso register form")
	}
	return &pssoRegisterCompleteRequest{
		PSSORegisterRequest: fleet.PSSORegisterRequest{
			DeviceUUID:          r.FormValue("deviceUUID"),
			DeviceSigningKey:    r.FormValue("deviceSigningKey"),
			DeviceEncryptionKey: r.FormValue("deviceEncryptionKey"),
			SignKeyID:           r.FormValue("signKeyID"),
			EncKeyID:            r.FormValue("encKeyID"),
			Code:                r.FormValue("code"),
			State:               r.FormValue("state"),
		},
	}, nil
}

type pssoRegisterCompleteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r pssoRegisterCompleteResponse) Error() error { return r.Err }

func pssoRegisterCompleteEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*pssoRegisterCompleteRequest)
	if err := svc.PSSORegisterComplete(ctx, req.PSSORegisterRequest); err != nil {
		return pssoRegisterCompleteResponse{Err: err}, err
	}
	return pssoRegisterCompleteResponse{}, nil
}

func (svc *Service) PSSORegisterComplete(ctx context.Context, _ fleet.PSSORegisterRequest) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

// ----- /mdm/apple/psso/token ------------------------------------------------

type pssoTokenRequest struct {
	body []byte
}

// DecodeRequest reads the raw JWT body posted by the extension. The body is
// expected to be a compact JWS (`a.b.c`) and is passed through verbatim to the
// service layer for parsing.
func (pssoTokenRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read psso token body")
	}
	return &pssoTokenRequest{body: body}, nil
}

type pssoTokenResponse struct {
	body []byte
	Err  error `json:"error,omitempty"`
}

func (r pssoTokenResponse) Error() error { return r.Err }

// HijackRender writes the raw JWE response bytes with the PSSO-specific
// Content-Type Apple's framework expects.
func (r pssoTokenResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", pssoContentTypeLoginResponse)
	w.Header().Set("Content-Length", strconv.Itoa(len(r.body)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(r.body)
}

func pssoTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*pssoTokenRequest)
	body, err := svc.PSSOToken(ctx, req.body)
	if err != nil {
		return pssoTokenResponse{Err: err}, err
	}
	return pssoTokenResponse{body: body}, nil
}

func (svc *Service) PSSOToken(ctx context.Context, _ []byte) ([]byte, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

// ----- /.well-known/jwks.json -----------------------------------------------

type pssoJWKSRequest struct{}

type pssoJWKSResponse struct {
	body []byte
	Err  error `json:"error,omitempty"`
}

func (r pssoJWKSResponse) Error() error { return r.Err }

func (r pssoJWKSResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/jwk-set+json")
	w.Header().Set("Content-Length", strconv.Itoa(len(r.body)))
	_, _ = w.Write(r.body)
}

func pssoJWKSEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	body, err := svc.PSSOJWKS(ctx)
	if err != nil {
		return pssoJWKSResponse{Err: err}, err
	}
	return pssoJWKSResponse{body: body}, nil
}

func (svc *Service) PSSOJWKS(ctx context.Context) ([]byte, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

// ----- /.well-known/apple-app-site-association ------------------------------

type pssoAASARequest struct{}

type pssoAASAResponse struct {
	body []byte
	Err  error `json:"error,omitempty"`
}

func (r pssoAASAResponse) Error() error { return r.Err }

func (r pssoAASAResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(r.body)))
	_, _ = w.Write(r.body)
}

func pssoAASAEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	body, err := svc.PSSOAASA(ctx)
	if err != nil {
		return pssoAASAResponse{Err: err}, err
	}
	return pssoAASAResponse{body: body}, nil
}

func (svc *Service) PSSOAASA(ctx context.Context) ([]byte, error) {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

// Ensure response types serialize to clean JSON when used with the default
// (non-HijackRender) renderer for tests.
var (
	_ = json.Marshal
	_ = pssoNonceResponse{}
)
