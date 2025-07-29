package httpsig

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type key int

const hostIdentityKey key = 0

// NewContext creates a new context.Context with host identity cert.
func NewContext(ctx context.Context, hostIdentity types.HostIdentityCertificate) context.Context {
	return context.WithValue(ctx, hostIdentityKey, hostIdentity)
}

// FromContext returns a pointer to the host identity cert.
func FromContext(ctx context.Context) (types.HostIdentityCertificate, bool) {
	v, ok := ctx.Value(hostIdentityKey).(types.HostIdentityCertificate)
	return v, ok
}

// MiddlewareFunc is a function which receives an http.Handler and returns another http.Handler.
// Typically, the returned handler is a closure which does something with the http.ResponseWriter and http.Request passed
// to it, and then calls the handler passed as parameter to the MiddlewareFunc.
type MiddlewareFunc func(http.Handler) http.Handler

func Middleware(ds fleet.Datastore, requireSignature bool, logger kitlog.Logger) (MiddlewareFunc, error) {
	// Initialize HTTP signature verifier
	httpSig := NewHTTPSig(ds, logger)
	verifier, err := httpSig.Verifier()
	if err != nil {
		return nil, fmt.Errorf("setup httpsig verifier: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !strings.Contains(req.URL.Path, "/api/fleet/orbit/") && !strings.Contains(req.URL.Path, "/osquery/") {
				next.ServeHTTP(w, req)
				return
			}

			// We do not verify the "ping" endpoint since it is used to get server capabilities and does not carry any data.
			// This endpoint is unauthenticated.
			if strings.HasSuffix(req.URL.Path, "/api/fleet/orbit/ping") {
				next.ServeHTTP(w, req)
				return
			}

			// If the request does not have an HTTP message signature, we do not verify it AND
			// we do not set the host identity cert in the context
			if req.Header.Get("signature") == "" || req.Header.Get("signature-input") == "" {
				if requireSignature {
					handleError(req.Context(), w,
						ctxerr.Errorf(req.Context(), "missing required HTTP message signature: path=%s", req.URL.Path),
						http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, req)
				return
			}

			// Verify signature using certificate associated with the provided serial number.
			result, err := verifier.Verify(req)
			if err != nil {
				handleError(req.Context(), w,
					ctxerr.Wrap(req.Context(), err, "failed to verify request signature", fmt.Sprintf("path=%s", req.URL.Path)),
					http.StatusUnauthorized)
				return
			}
			keySpecer, ok := result.KeySpecer.(*KeySpecer)
			if !ok {
				handleError(req.Context(), w,
					ctxerr.New(req.Context(), fmt.Sprintf("could not extract host identity certificate key: path=%s", req.URL.Path)),
					http.StatusInternalServerError)
				return
			}
			if !result.Verified {
				handleError(req.Context(), w,
					ctxerr.New(req.Context(), fmt.Sprintf("request not verified: path=%s host_uuid=%s", req.URL.Path,
						keySpecer.hostIdentityCert.CommonName)),
					http.StatusUnauthorized)
				return
			}

			level.Debug(logger).Log("msg", "httpsig verified", "host_id", keySpecer.hostIdentityCert.HostID)

			// Signature is valid, we set the identity data in the context and proceed with processing the request.
			req = req.WithContext(NewContext(req.Context(), keySpecer.hostIdentityCert))
			next.ServeHTTP(w, req)
		})
	}, nil
}

func handleError(ctx context.Context, w http.ResponseWriter, err error, code int) {
	ctxerr.Handle(ctx, err)
	http.Error(w, err.Error(), code)
}
