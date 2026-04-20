package auth

import (
	"context"
	"net/http"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// contextKeyRouteTemplate is the context key type for the mux route template.
type contextKeyRouteTemplate struct{}

var routeTemplateKey = contextKeyRouteTemplate{}

// RouteTemplateRequestFunc captures the gorilla/mux route template for the
// matched request and stores it in the context.
func RouteTemplateRequestFunc(ctx context.Context, r *http.Request) context.Context {
	route := mux.CurrentRoute(r)
	if route == nil {
		return ctx
	}
	tpl, err := route.GetPathTemplate()
	if err != nil {
		return ctx
	}
	return context.WithValue(ctx, routeTemplateKey, tpl)
}

// APIOnlyEndpointCheck returns an endpoint.Endpoint middleware that enforces
// access control for API-only users (api_only=true). It must be wired inside
// AuthenticatedUser (so a Viewer is already in context when it runs) and the
// enclosing transport must register RouteTemplateRequestFunc as a ServerBefore
// option so the mux route template is available in context.
//
// For non-API-only users the check is skipped entirely. When there is no Viewer
// in context, the call passes through — AuthenticatedUser guarantees that any
// request that needs a Viewer has already been rejected before reaching here.
//
// For API-only users two checks are applied in order:
//  1. The requested route must appear in the API endpoint catalog. If not, a
//     permission error (403) is returned.
//  2. If the user has configured endpoint restrictions (rows in
//     user_api_endpoints), the route must match one of them. If not, a
//     permission error (403) is returned. An empty restriction list grants
//     full access to all catalog endpoints.
func APIOnlyEndpointCheck(next endpoint.Endpoint) endpoint.Endpoint {
	return apiOnlyEndpointCheck(apiendpoints.IsInCatalog, next)
}

func apiOnlyEndpointCheck(isInCatalog func(string) bool, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		v, ok := viewer.FromContext(ctx)
		if !ok || v.User == nil || !v.User.APIOnly {
			return next(ctx, request)
		}

		requestMethod, _ := ctx.Value(kithttp.ContextKeyRequestMethod).(string)
		routeTemplate, _ := ctx.Value(routeTemplateKey).(string)

		fp := fleet.NewAPIEndpointFromTpl(requestMethod, routeTemplate).Fingerprint()

		if !isInCatalog(fp) {
			return nil, permissionDenied(ctx)
		}

		// No endpoint restrictions: full access to all catalog endpoints.
		if len(v.User.APIEndpoints) == 0 {
			return next(ctx, request)
		}

		// Check whether the requested endpoint matches any of the user's allowed endpoints.
		for _, ep := range v.User.APIEndpoints {
			if fleet.NewAPIEndpointFromTpl(ep.Method, ep.Path).Fingerprint() == fp {
				return next(ctx, request)
			}
		}

		return nil, permissionDenied(ctx)
	}
}

func permissionDenied(ctx context.Context) error {
	if ac, ok := authz.FromContext(ctx); ok {
		ac.SetChecked()
	}
	return fleet.NewPermissionError("forbidden")
}
