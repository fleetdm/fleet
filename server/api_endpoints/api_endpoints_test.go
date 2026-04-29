package apiendpoints

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestValidateAPIEndpoints(t *testing.T) {
	originalYAML := apiEndpointsYAML
	t.Cleanup(func() {
		apiEndpointsYAML = originalYAML
	})

	endpoints := []fleet.APIEndpoint{
		fleet.NewAPIEndpointFromTpl("GET", "/api/v1/fleet/hosts"),
		fleet.NewAPIEndpointFromTpl("POST", "/api/v1/fleet/hosts/:id/refetch"),
	}
	apiEndpointsYAML = []byte(`
- method: "GET"
  path: "/api/v1/fleet/hosts"
  display_name: "Route 1"
- method: "POST"
  path: "/api/v1/fleet/hosts/:id/refetch"
  display_name: "Route 2"`)

	routerWithEndpoints := func(endpoints []fleet.APIEndpoint) *mux.Router {
		r := mux.NewRouter()
		for _, e := range endpoints {
			path := strings.Replace(e.Path, "/_version_/", "/{fleetversion:(?:v1|latest)}/", 1)
			r.Handle(path, http.NotFoundHandler()).Methods(e.Method)
		}
		return r
	}

	t.Run("all routes present", func(t *testing.T) {
		err := Init(routerWithEndpoints(endpoints))
		require.NoError(t, err)
	})

	t.Run("missing route returns error", func(t *testing.T) {
		err := Init(routerWithEndpoints(endpoints[:1]))
		require.ErrorContains(t, err, endpoints[1].Method+" "+endpoints[1].Path)
	})

	t.Run("no routes registered returns error listing all missing", func(t *testing.T) {
		err := Init(mux.NewRouter())
		require.ErrorContains(t, err, "the following API endpoints are unknown")
	})

	t.Run("non-mux handler returns error", func(t *testing.T) {
		err := Init(http.NewServeMux())
		require.ErrorContains(t, err, "expected *mux.Router")
	})

	t.Run("empty endpoint list always passes", func(t *testing.T) {
		apiEndpointsYAML = []byte(``)
		err := Init(mux.NewRouter())
		require.NoError(t, err)
	})
}
