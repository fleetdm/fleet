package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// HTTP paths for the Apple Platform SSO IdP endpoints. These follow the SCEP
// path convention (no /api/ or /v1/ prefix) because they're raw protocol
// endpoints the Mac extension talks to directly, not user-facing API. The
// paths are registered on the root *http.ServeMux (see registerPSSO) rather
// than the versioned /api router, so they resolve exactly as written here.
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

// pssoHandlers returns the set of root-mux PSSO handlers keyed by path. The
// caller is responsible for registering them on a *http.ServeMux. Each handler
// dispatches on r.Method internally since *http.ServeMux does not support
// method-based routing.
func pssoHandlers(svc fleet.Service, logger *slog.Logger) map[string]http.Handler {
	return map[string]http.Handler{
		pssoNoncePath:    pssoNonceHandler(svc, logger),
		pssoRegisterPath: pssoRegisterHandler(svc, logger),
		pssoTokenPath:    pssoTokenHandler(svc, logger),
		pssoJWKSPath:     pssoJWKSHandler(svc, logger),
		pssoAASAPath:     pssoAASAHandler(svc, logger),
	}
}

// pssoNonceHandler serves POST /mdm/apple/psso/nonce — returns a short-lived
// JSON nonce the device includes in subsequent register/token requests.
func pssoNonceHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			encodeError(ctx, &fleet.BadRequestError{Message: "method not allowed"}, w)
			return
		}
		nonce, err := svc.PSSONonce(ctx)
		if err != nil {
			encodeError(ctx, err, w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"Nonce": nonce})
	})
}

// pssoRegisterHandler serves both halves of the registration handshake on
// /mdm/apple/psso/register. GET returns a 302 to the configured upstream OIDC
// authorize URL; POST receives the device's signing/encryption keys and the
// authorization code from the IdP callback.
func pssoRegisterHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		switch r.Method {
		case http.MethodGet:
			redirectURL, err := svc.PSSORegisterBegin(ctx)
			if err != nil {
				encodeError(ctx, err, w)
				return
			}
			http.Redirect(w, r, redirectURL, http.StatusFound)
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				encodeError(ctx, ctxerr.Wrap(ctx, err, "parse psso register form"), w)
				return
			}
			req := fleet.PSSORegisterRequest{
				DeviceUUID:          r.FormValue("deviceUUID"),
				DeviceSigningKey:    r.FormValue("deviceSigningKey"),
				DeviceEncryptionKey: r.FormValue("deviceEncryptionKey"),
				SignKeyID:           r.FormValue("signKeyID"),
				EncKeyID:            r.FormValue("encKeyID"),
				Code:                r.FormValue("code"),
				State:               r.FormValue("state"),
			}
			if err := svc.PSSORegisterComplete(ctx, req); err != nil {
				encodeError(ctx, err, w)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			encodeError(ctx, &fleet.BadRequestError{Message: "method not allowed"}, w)
		}
	})
}

// pssoTokenHandler serves POST /mdm/apple/psso/token — receives a compact JWS
// from the extension and returns a JWE wrapped with the PSSO-specific
// Content-Type Apple's framework expects.
func pssoTokenHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method != http.MethodPost {
			encodeError(ctx, &fleet.BadRequestError{Message: "method not allowed"}, w)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			encodeError(ctx, ctxerr.Wrap(ctx, err, "read psso token body"), w)
			return
		}
		out, err := svc.PSSOToken(ctx, body)
		if err != nil {
			encodeError(ctx, err, w)
			return
		}
		w.Header().Set("Content-Type", pssoContentTypeLoginResponse)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(out)
	})
}

// pssoJWKSHandler serves GET /.well-known/jwks.json — exposes Fleet's PSSO
// server signing key as a JWKS so the device extension can verify server JWTs.
func pssoJWKSHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method != http.MethodGet {
			encodeError(ctx, &fleet.BadRequestError{Message: "method not allowed"}, w)
			return
		}
		body, err := svc.PSSOJWKS(ctx)
		if err != nil {
			encodeError(ctx, err, w)
			return
		}
		w.Header().Set("Content-Type", "application/jwk-set+json")
		_, _ = w.Write(body)
	})
}

// pssoAASAHandler serves GET /.well-known/apple-app-site-association — the
// Apple App Site Association JSON the extension fetches at install time to
// validate the `authsrv:` entitlement against this hostname.
func pssoAASAHandler(svc fleet.Service, _ *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			encodeError(ctx, &fleet.BadRequestError{Message: "method not allowed"}, w)
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

func (svc *Service) PSSORegisterBegin(ctx context.Context) (string, error) {
	// skipauth: Implementation returns only the license error; nothing to authorize.
	svc.authz.SkipAuthorization(ctx)
	return "", fleet.ErrMissingLicense
}

func (svc *Service) PSSORegisterComplete(ctx context.Context, _ fleet.PSSORegisterRequest) error {
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
