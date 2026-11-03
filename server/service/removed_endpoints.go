package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// removedEndpoint is an API endpoint that Fleet no longer serves. Path omits the /api/_version_/fleet prefix.
type removedEndpoint struct {
	Method string
	Path   string
}

// removedEndpoints answer 410 with their pkg/removednames message. Only these are registered, not every removednames path entry:
// most of those (the team renames) are still served as deprecated aliases, and a handler here would conflict with them.
var removedEndpoints = []removedEndpoint{
	{Method: http.MethodPost, Path: "/mdm/apple/profiles/preassign"},
	{Method: http.MethodPost, Path: "/mdm/apple/profiles/match"},
}

// removedEndpointMessage returns the pkg/removednames message for a removed endpoint.
func removedEndpointMessage(method, path string) (string, bool) {
	entry, ok := removednames.Find(removednames.KindPath, "/api/_version_/fleet"+path, method)
	if !ok {
		return "", false
	}
	return entry.Message(), true
}

type removedEndpointError struct {
	message string
}

func (e *removedEndpointError) Error() string { return e.message }

func (e *removedEndpointError) StatusCode() int { return http.StatusGone }

func (e *removedEndpointError) IsClientError() bool { return true }

type removedEndpointResponse struct {
	Err error `json:"error,omitempty"`
}

func (r removedEndpointResponse) Error() error { return r.Err }

// registerRemovedEndpoints registers each removed endpoint without authentication: the 410 reveals nothing, and a 401
// would hide that the endpoint is gone. It panics at startup if an endpoint has no message.
func registerRemovedEndpoints(ne *eu.CommonEndpointer[handlerFunc], endpoints []removedEndpoint,
	message func(method, path string) (string, bool),
) {
	for _, re := range endpoints {
		msg, ok := message(re.Method, re.Path)
		if !ok {
			panic("removed endpoint " + re.Method + " " + re.Path + " has no pkg/removednames entry")
		}
		handler := func(ctx context.Context, _ any, _ fleet.Service) (fleet.Errorer, error) {
			if az, ok := authz_ctx.FromContext(ctx); ok {
				az.SetChecked()
			}
			return removedEndpointResponse{Err: &removedEndpointError{message: msg}}, nil
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
