package service

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestGetAPIEndpoints(t *testing.T) {
	routes := GetAPIEndpoints()
	require.NotEmpty(t, routes)
	for _, r := range routes {
		require.NotEmpty(t, r.Method, "route method should not be empty")
		require.NotEmpty(t, r.Path, "route path should not be empty")
		require.NotEmpty(t, r.Name, "route name should not be empty")
		require.True(t, strings.HasPrefix(r.Path, "/"), "route path should start with /")
		_, validMethod := validHTTPMethods[r.Method]
		require.True(t, validMethod, "route method %q should be a valid HTTP method", r.Method)
	}
}

func TestAPIEndpointValidate(t *testing.T) {
	base := APIEndpoint{Method: "GET", Path: "/api/_version_/fleet/foo", Name: "foo"}

	tests := []struct {
		name    string
		modify  func(APIEndpoint) APIEndpoint
		wantErr string
	}{
		{
			name:   "valid endpoint",
			modify: func(e APIEndpoint) APIEndpoint { return e },
		},
		{
			name:    "missing name",
			modify:  func(e APIEndpoint) APIEndpoint { e.Name = ""; return e },
			wantErr: "name is required",
		},
		{
			name:    "whitespace name",
			modify:  func(e APIEndpoint) APIEndpoint { e.Name = "   "; return e },
			wantErr: "name is required",
		},
		{
			name:    "invalid method",
			modify:  func(e APIEndpoint) APIEndpoint { e.Method = "GTE"; return e },
			wantErr: "invalid HTTP method",
		},
		{
			name:    "empty path",
			modify:  func(e APIEndpoint) APIEndpoint { e.Path = " "; return e },
			wantErr: "path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.modify(base).validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAPIEndpoints(t *testing.T) {
	allRoutes := GetAPIEndpoints()

	routerWithRoutes := func(routes []APIEndpoint) *mux.Router {
		r := mux.NewRouter()
		for _, route := range routes {
			path := strings.Replace(route.Path, "/_version_/", "/{fleetversion:(?:v1|latest)}/", 1)
			r.Handle(path, http.NotFoundHandler()).Methods(route.Method)
		}
		return r
	}

	tests := []struct {
		name    string
		handler http.Handler
		wantErr string
	}{
		{
			name:    "all routes present",
			handler: routerWithRoutes(allRoutes),
		},
		{
			name:    "no routes registered",
			handler: mux.NewRouter(),
			wantErr: "the following API endpoints are missing",
		},
		{
			name:    "non-mux handler returns error",
			handler: http.NewServeMux(),
			wantErr: "expected *mux.Router, got *http.ServeMux",
		},
	}

	if len(allRoutes) >= 2 {
		last := allRoutes[len(allRoutes)-1]
		tests = append(tests, struct {
			name    string
			handler http.Handler
			wantErr string
		}{
			name:    "last route missing",
			handler: routerWithRoutes(allRoutes[:len(allRoutes)-1]),
			wantErr: last.Method + " " + last.Path,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIEndpoints(tt.handler)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
