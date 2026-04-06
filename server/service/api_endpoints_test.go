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

	require.NoError(t, base.validate())

	t.Run("missing name", func(t *testing.T) {
		e := base
		e.Name = ""
		require.ErrorContains(t, e.validate(), "name is required")
	})

	t.Run("invalid method", func(t *testing.T) {
		e := base
		e.Method = "GTE"
		require.ErrorContains(t, e.validate(), "invalid HTTP method")
	})

	t.Run("path without leading slash", func(t *testing.T) {
		e := base
		e.Path = " "
		require.ErrorContains(t, e.validate(), "path is required")
	})
}

func TestValidateAPIEndpoints_ReturnsFalseWhenMissing(t *testing.T) {
	r := mux.NewRouter()
	// No routes registered — every YAML route is missing.
	ok, missing := ValidateAPIEndpoints(r)
	require.False(t, ok)
	require.NotEmpty(t, missing)
}

func TestValidateAPIEndpoints_ReturnsTrueWhenAllPresent(t *testing.T) {
	r := mux.NewRouter()
	// Register every route defined in the YAML so that validation succeeds.
	for _, route := range GetAPIEndpoints() {
		// Convert /_version_/ to a concrete versioned segment so gorilla/mux accepts it.
		path := strings.Replace(route.Path, "/_version_/", "/{fleetversion:(?:v1|latest)}/", 1)
		r.Handle(path, http.NotFoundHandler()).Methods(route.Method)
	}
	ok, missing := ValidateAPIEndpoints(r)
	require.True(t, ok)
	require.Empty(t, missing)
}

func TestValidateAPIEndpoints_ReturnsFalseForPartiallyMissingRoutes(t *testing.T) {
	routes := GetAPIEndpoints()
	if len(routes) < 2 {
		t.Skip("need at least 2 routes for this test")
	}

	r := mux.NewRouter()
	// Register all but the last route — validation must still return false.
	for _, route := range routes[:len(routes)-1] {
		path := strings.Replace(route.Path, "/_version_/", "/{fleetversion:(?:v1|latest)}/", 1)
		r.Handle(path, http.NotFoundHandler()).Methods(route.Method)
	}
	ok, missing := ValidateAPIEndpoints(r)
	require.False(t, ok)
	last := routes[len(routes)-1]
	require.Equal(t, []string{last.Method + " " + last.Path}, missing)
}

func TestValidateAPIEndpoints_PanicsForNonMuxRouter(t *testing.T) {
	require.PanicsWithValue(t,
		"ValidateAPIEndpoints: expected *mux.Router, got *http.ServeMux",
		func() { ValidateAPIEndpoints(http.NewServeMux()) },
	)
}
