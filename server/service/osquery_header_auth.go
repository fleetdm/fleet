package service

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/osqueryauth"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Rejection reasons for the osquery pre-auth counter.
const (
	preAuthRejectMissing      = "missing"       // absent or wrong-scheme Authorization header
	preAuthRejectInvalidToken = "invalid_token" // header scheme matched but token failed validation
)

// osqueryPreAuthRejections counts osquery pre-auth rejections by reason and
// route. Operators can alert on sustained growth of the "invalid_token" or
// "missing" buckets. Initialized at package load; package panics if the OTEL
// counter cannot be created, so the var is always non-nil at use.
var osqueryPreAuthRejections = mustNewPreAuthRejectionsCounter()

func mustNewPreAuthRejectionsCounter() metric.Int64Counter {
	c, err := otel.Meter("fleet").Int64Counter(
		"fleet.osquery.preauth_rejections",
		metric.WithDescription("Count of osquery requests rejected by the HTTP-level header pre-auth by route and reason"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err)
	}
	return c
}

// preAuthRejectionAttrs returns the metric attributes for the pre-auth
// rejection counter.
func preAuthRejectionAttrs(route, reason string) metric.AddOption {
	return metric.WithAttributes(
		attribute.String("http.route", route),
		attribute.String("reason", reason),
	)
}

// osqueryHeaderAuthScheme is the canonical Authorization-header scheme used
// by osquery requests for header-based node key authentication. The format is:
//
//	Authorization: NodeKey <node_key>
const osqueryHeaderAuthScheme = "NodeKey"

// osqueryHeaderPreAuth returns an HTTP middleware that authenticates osquery
// requests from the Authorization header before the request body is read.
// It is registered ONLY when osquery.allow_body_auth_fallback is false; with
// the flag at its default of true the middleware is not installed at all and
// this function never runs (the legacy body-based auth path is the sole
// authenticator). Callers must guard the .WithHTTPPreAuth(...) call with the
// flag check.
//
// When installed, the middleware enforces strict header auth:
//
//   - Header present and token valid: authenticate, populate ctx so the
//     endpoint-layer authenticatedHost middleware becomes a passthrough.
//   - Header present and token invalid: 401 with node_invalid:true, body
//     not read.
//   - Header absent, malformed, or wrong scheme: 401 with node_invalid:true,
//     body not read.
func osqueryHeaderPreAuth(svc fleet.Service, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			nodeKey := extractNodeKeyFromHeader(r)

			if nodeKey == "" {
				osqueryPreAuthRejections.Add(ctx, 1, preAuthRejectionAttrs(r.URL.Path, preAuthRejectMissing))
				logger.WarnContext(ctx, "osquery request rejected: missing or malformed Authorization header",
					"path", r.URL.Path, "remote_addr", r.RemoteAddr)
				encodeError(ctx, newOsqueryErrorWithInvalidNode("authentication error: invalid authorization header"), w)
				return
			}

			host, debug, err := svc.AuthenticateHost(ctx, nodeKey)
			if err != nil {
				osqueryPreAuthRejections.Add(ctx, 1, preAuthRejectionAttrs(r.URL.Path, preAuthRejectInvalidToken))
				logger.WarnContext(ctx, "osquery request rejected: invalid Authorization header token",
					"path", r.URL.Path, "remote_addr", r.RemoteAddr, "err", err)
				encodeError(ctx, newOsqueryErrorWithInvalidNode("authentication error: invalid authorization header"), w)
				return
			}

			// Populate the ctx fields that work at this stage (plain
			// context.WithValue). Side effects that need the per-request
			// logging and authz contexts (SetAuthnMethod, instrumentHostLogger,
			// debug-mode request logging) are applied in the endpoint-layer
			// authenticatedHost passthrough after kithttp.ServerBefore runs.
			ctx = hostctx.NewContext(ctx, host)
			ctx = ctxerr.AddErrorContextProvider(ctx, &hostctx.HostAttributeProvider{Host: host})
			ctx = osqueryauth.NewPreAuthedContext(ctx)
			if debug {
				ctx = osqueryauth.NewDebugContext(ctx)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractNodeKeyFromHeader returns the node key parsed from the
// "Authorization: NodeKey <token>" header, or "" if the header is absent.
func extractNodeKeyFromHeader(r *http.Request) string {
	authz := r.Header.Get("Authorization")
	if authz == "" {
		return ""
	}
	scheme, token, ok := strings.Cut(authz, " ")
	if !ok || !strings.EqualFold(scheme, osqueryHeaderAuthScheme) {
		return ""
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return ""
	}
	return token
}

// osqueryCarveBlockHeaderPreAuth returns an HTTP middleware for
// /api/osquery/carve/block. Like osqueryHeaderPreAuth, it is registered ONLY
// when osquery.allow_body_auth_fallback is false. With the flag at its
// default of true this middleware is not installed and /carve/block falls
// back entirely to its existing byte-by-byte streaming-parse auth (session_id
// + request_id verified against the carve store).
//
// When installed, the streaming parser still runs after pre-auth succeeds, and
// CarveBlock additionally verifies that the carve's HostId matches the
// authenticated host.
//
//   - Header valid NodeKey + valid token: stash the authenticated host in
//     ctx; the streaming parser still runs, and CarveBlock verifies
//     ownership.
//   - Header valid NodeKey + invalid token: 401, body not read.
//   - Header absent or wrong scheme: 401, body not read.
func osqueryCarveBlockHeaderPreAuth(svc fleet.Service, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			nodeKey := extractNodeKeyFromHeader(r)
			if nodeKey == "" {
				osqueryPreAuthRejections.Add(ctx, 1, preAuthRejectionAttrs(r.URL.Path, preAuthRejectMissing))
				logger.WarnContext(ctx, "osquery carve/block rejected: missing or malformed Authorization header",
					"path", r.URL.Path, "remote_addr", r.RemoteAddr)
				encodeError(ctx, newOsqueryErrorWithInvalidNode("authentication error: invalid authorization header"), w)
				return
			}
			host, _, err := svc.AuthenticateHost(ctx, nodeKey)
			if err != nil {
				osqueryPreAuthRejections.Add(ctx, 1, preAuthRejectionAttrs(r.URL.Path, preAuthRejectInvalidToken))
				logger.WarnContext(ctx, "osquery carve/block rejected: invalid Authorization header token",
					"path", r.URL.Path, "remote_addr", r.RemoteAddr, "err", err)
				encodeError(ctx, newOsqueryErrorWithInvalidNode("authentication error: invalid authorization header"), w)
				return
			}
			// Stash the host so carveBlockEndpoint can enforce the
			// carve-ownership check.
			ctx = hostctx.NewContext(ctx, host)
			ctx = ctxerr.AddErrorContextProvider(ctx, &hostctx.HostAttributeProvider{Host: host})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
