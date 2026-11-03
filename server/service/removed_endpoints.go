package service

import (
	"context"
	"fmt"
	"net/http"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// removedEndpoint is an API endpoint that Fleet no longer serves. Path omits the /api/_version_/fleet prefix because
// the Fleet 5 removal message names endpoints by their short path.
type removedEndpoint struct {
	Method string
	Path   string
}

// removedEndpoints answer 410 so that callers learn the endpoint is gone instead of getting a bare 404.
var removedEndpoints = []removedEndpoint{
	{Method: http.MethodPost, Path: "/mdm/apple/profiles/preassign"},
	{Method: http.MethodPost, Path: "/mdm/apple/profiles/match"},
}

type removedEndpointError struct {
	endpoint removedEndpoint
}

func (e *removedEndpointError) Error() string {
	return fmt.Sprintf(`The "%s %s" endpoint is not available anymore. What to use instead: %s`,
		e.endpoint.Method, e.endpoint.Path, eu.Fleet5MigrationURL)
}

func (e *removedEndpointError) StatusCode() int { return http.StatusGone }

func (e *removedEndpointError) IsClientError() bool { return true }

type removedEndpointResponse struct {
	Err error `json:"error,omitempty"`
}

func (r removedEndpointResponse) Error() error { return r.Err }

// registerRemovedEndpoints registers each removed endpoint without authentication: the 410 reveals nothing, and a 401
// would hide that the endpoint is gone.
func registerRemovedEndpoints(ne *eu.CommonEndpointer[handlerFunc], endpoints []removedEndpoint) {
	for _, re := range endpoints {
		handler := func(ctx context.Context, _ any, _ fleet.Service) (fleet.Errorer, error) {
			if az, ok := authz_ctx.FromContext(ctx); ok {
				az.SetChecked()
			}
			return removedEndpointResponse{Err: &removedEndpointError{endpoint: re}}, nil
		}
		path := "/api/_version_/fleet" + re.Path
		switch re.Method {
		case http.MethodGet:
			ne.GET(path, handler, nil)
		case http.MethodPost:
			ne.POST(path, handler, nil)
		case http.MethodPut:
			ne.PUT(path, handler, nil)
		case http.MethodPatch:
			ne.PATCH(path, handler, nil)
		case http.MethodDelete:
			ne.DELETE(path, handler, nil)
		default:
			panic("unsupported method for a removed endpoint: " + re.Method)
		}
	}
}
