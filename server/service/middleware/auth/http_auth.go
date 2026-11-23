package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth/externalauth"
	kithttp "github.com/go-kit/kit/transport/http"
)

// ExternalAuthenticator verifies bearer JWTs from external issuers.
type ExternalAuthenticator interface {
	AuthViewer(ctx context.Context, token, clientIP string) (*viewer.Viewer, error)
}

type requestContextOpts struct {
	external ExternalAuthenticator
	clientIP func(*http.Request) string
}

type RequestContextOption func(*requestContextOpts)

// WithExternalAuth sends JWT-shaped bearer tokens to a. clientIP keys its rate limit.
func WithExternalAuth(a ExternalAuthenticator, clientIP func(*http.Request) string) RequestContextOption {
	return func(o *requestContextOpts) {
		o.external, o.clientIP = a, clientIP
	}
}

// ViewerForBearer resolves a bearer token the way every Fleet API request does: a
// JWT-shaped bearer goes to the external authenticator (an error if none is configured)
// and never to the session lookup; anything else is a session key. On JWT success the
// returned context carries the externalauth marker.
func ViewerForBearer(ctx context.Context, r *http.Request, bearer string, svc fleet.Service, opts ...RequestContextOption) (context.Context, *viewer.Viewer, error) {
	var o requestContextOpts
	for _, fn := range opts {
		fn(&o)
	}
	if !externalauth.IsJWTShaped(bearer) {
		v, err := AuthViewer(ctx, bearer, svc)
		return ctx, v, err
	}
	// A session key never contains a dot, so a JWT-shaped bearer skips the session lookup.
	if o.external == nil {
		return ctx, nil, fleet.NewAuthRequiredError(errNoExternalAuth.Error())
	}
	v, err := o.external.AuthViewer(ctx, bearer, o.clientIP(r))
	if err != nil {
		return ctx, nil, err
	}
	return externalauth.NewContext(ctx), v, nil
}

// SetRequestsContexts updates the request with necessary context values for a request
func SetRequestsContexts(svc fleet.Service, opts ...RequestContextOption) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		bearer := token.FromHTTPRequest(r)
		ctx = token.NewContext(ctx, bearer)
		if bearer != "" {
			if vctx, v, err := ViewerForBearer(ctx, r, string(bearer), svc, opts...); err == nil {
				ctx = viewer.NewContext(vctx, *v)
				// Register viewer as error context provider for ctxerr enrichment
				ctx = ctxerr.AddErrorContextProvider(ctx, v)
				// Register viewer as user emailer for logging
				ctx = logging.WithUserEmailer(ctx, v)
			}
		}

		ctx = logging.NewContext(ctx, &logging.LoggingContext{})
		ctx = logging.WithStartTime(ctx)
		return ctx
	}
}

var errNoExternalAuth = errors.New("external jwt authentication is not configured")

func SetRequestsContextMiddleware(svc fleet.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := kithttp.PopulateRequestContext(r.Context(), r)
		ctx = SetRequestsContexts(svc)(ctx, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
