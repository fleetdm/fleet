package auth

import (
	"context"
	"log/slog"
	"net/http"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
)

// RouteTemplateRequestFunc captures the gorilla/mux route template for the
// matched request and stores it in the context. Alias of the platform
// implementation, re-exported so callers that already import this package can
// continue to reference it here.
var RouteTemplateRequestFunc = eu.RouteTemplateRequestFunc

// APIOnlyEndpointCheck returns an endpoint.Endpoint middleware that enforces
// access control for API-only users (api_only=true) that have configured
// endpoint restrictions. It must be wired inside AuthenticatedUser (so a Viewer
// is already in context when it runs) and the enclosing transport must register
// RouteTemplateRequestFunc as a ServerBefore option so the mux route template
// is available in context.
//
// The check is skipped entirely for: non-API-only users, requests with no
// Viewer in context (AuthenticatedUser already rejects those), and API-only
// users with no endpoint restrictions configured — the latter are granted
// access to every registered route, gated only by role-based authz further
// down the chain.
//
// For API-only users with a non-empty restriction list (rows in
// user_api_endpoints), two checks are applied in order:
//  1. The requested route must appear in the API endpoint catalog. If not, a
//     403 with EndpointRestrictionDeniedMessage is returned.
//  2. The route must match one of the user's allowed endpoints. If not, a
//     403 with EndpointRestrictionDeniedMessage is returned.
//
// Both denials are logged at info level with the route template and denial
// reason so they can be distinguished from role-based permission denials.
func APIOnlyEndpointCheck(next endpoint.Endpoint) endpoint.Endpoint {
	return apiOnlyEndpointCheck(apiendpoints.IsInCatalog, next)
}

func apiOnlyEndpointCheck(isInCatalog func(string) bool, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		v, ok := viewer.FromContext(ctx)
		if !ok || v.User == nil || !v.User.APIOnly || len(v.User.APIEndpoints) == 0 {
			return next(ctx, request)
		}

		requestMethod, _ := ctx.Value(kithttp.ContextKeyRequestMethod).(string)
		routeTemplate, _ := eu.RouteTemplateFromContext(ctx)

		// Missing method or route template means the transport wasn't wired
		// with RouteTemplateRequestFunc; fail closed naming the misconfiguration.
		if requestMethod == "" {
			return nil, endpointRestrictionDenied(ctx, routeTemplate, "request method missing from request context")
		}
		if routeTemplate == "" {
			return nil, endpointRestrictionDenied(ctx, routeTemplate, "route template missing from request context")
		}

		fp := fleet.NewAPIEndpointFromTpl(requestMethod, routeTemplate).Fingerprint()

		if !isInCatalog(fp) {
			return nil, endpointRestrictionDenied(ctx, routeTemplate, "endpoint not in API endpoint catalog")
		}

		// Check whether the requested endpoint matches any of the user's allowed endpoints.
		for _, ep := range v.User.APIEndpoints {
			if fleet.NewAPIEndpointFromTpl(ep.Method, ep.Path).Fingerprint() == fp {
				return next(ctx, request)
			}
		}

		return nil, endpointRestrictionDenied(ctx, routeTemplate, "endpoint not in user's allowed API endpoints")
	}
}

// EndpointRestrictionDeniedMessage is the 403 response body for
// endpoint-restriction denials, distinct from role-based permission denials.
const EndpointRestrictionDeniedMessage = "endpoint not permitted for this API-only user"

// endpointRestrictionDenied rejects the request with a 403 that identifies the
// endpoint restriction (rather than the user's role) as the gate, and forces
// the request log line to info level (role-based 403s log at debug, which is
// how these denials went unnoticed during debugging).
func endpointRestrictionDenied(ctx context.Context, routeTemplate, reason string) error {
	if ac, ok := authz.FromContext(ctx); ok {
		ac.SetChecked()
	}
	logging.WithLevel(ctx, slog.LevelInfo)
	logging.WithExtras(ctx,
		"denied_by", "api_only_endpoint_restriction",
		"denial_reason", reason,
		"route", routeTemplate,
	)
	// PermissionError alone won't do: the error encoder discards its message
	// and renders a generic "Permission Denied" body. Wrapping it in a
	// UserMessageError routes it through the encoder branch that includes the
	// message in the response.
	return fleet.NewUserMessageError(
		fleet.NewPermissionError(EndpointRestrictionDeniedMessage),
		http.StatusForbidden,
	)
}
