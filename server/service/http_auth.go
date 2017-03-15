package service

import (
	"context"
	"net/http"
	"strings"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/kolide/kolide/server/contexts/token"
	"github.com/kolide/kolide/server/contexts/viewer"
	"github.com/kolide/kolide/server/kolide"
)

// setRequestsContexts updates the request with necessary context values for a request
func setRequestsContexts(svc kolide.Service, jwtKey string) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		bearer := token.FromHTTPRequest(r)
		ctx = token.NewContext(ctx, bearer)
		v, err := authViewer(ctx, jwtKey, bearer, svc)
		if err == nil {
			ctx = viewer.NewContext(ctx, *v)
		}

		// get the user-id for request
		if strings.Contains(r.URL.Path, "users/") {
			ctx = withUserIDFromRequest(r, ctx)
		}
		return ctx
	}
}

func withUserIDFromRequest(r *http.Request, ctx context.Context) context.Context {
	id, _ := idFromRequest(r, "id")
	return context.WithValue(ctx, "request-id", id)
}
