package service

import (
	"fmt"
	"net/http"
	"strings"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/contexts/token"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"golang.org/x/net/context"
)

// authentication error
type authError struct {
	reason string
	// client reason is used to provide
	// a different error message to the client
	// when security is a concern
	clientReason string
}

func (e authError) Error() string {
	return e.reason
}

func (e authError) AuthError() string {
	if e.clientReason != "" {
		return e.clientReason
	}
	return "authentication error"
}

// permissionError, set when user is authenticated, but not allowed to perform action
type permissionError struct {
	message string
	badArgs []invalidArgument
}

func (e permissionError) Error() string {
	switch len(e.badArgs) {
	case 0:
	case 1:
		e.message = fmt.Sprintf("unauthorized: %s",
			e.badArgs[0].reason,
		)
	default:
		e.message = fmt.Sprintf("unauthorized: %s and %d other errors",
			e.badArgs[0].reason,
			len(e.badArgs),
		)
	}
	if e.message == "" {
		return "unauthorized"
	}
	return e.message
}

func (e permissionError) PermissionError() []map[string]string {
	var forbidden []map[string]string
	if len(e.badArgs) == 0 {
		forbidden = append(forbidden, map[string]string{"reason": e.Error()})
		return forbidden
	}
	for _, arg := range e.badArgs {
		forbidden = append(forbidden, map[string]string{
			"name":   arg.name,
			"reason": arg.reason,
		})
	}
	return forbidden

}

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
