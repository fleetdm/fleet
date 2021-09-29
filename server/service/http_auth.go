package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"

	kithttp "github.com/go-kit/kit/transport/http"
)

// setRequestsContexts updates the request with necessary context values for a request
func setRequestsContexts(svc fleet.Service) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		bearer := token.FromHTTPRequest(r)
		ctx = token.NewContext(ctx, bearer)
		v, err := authViewer(ctx, string(bearer), svc)
		if err == nil {
			ctx = viewer.NewContext(ctx, *v)
		}

		// get the user-id for request
		if strings.Contains(r.URL.Path, "users/") {
			ctx = withUserIDFromRequest(r, ctx)
		}

		ctx = logging.NewContext(ctx, &logging.LoggingContext{})
		ctx = logging.WithStartTime(ctx)
		return ctx
	}
}

func withUserIDFromRequest(r *http.Request, ctx context.Context) context.Context {
	id, _ := idFromRequest(r, "id")
	return context.WithValue(ctx, "request-id", id)
}
