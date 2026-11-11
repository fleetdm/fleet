package service

import (
	"strings"
	"testing"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRouteTableMatchesGorilla checks the endpointer's route table against the production gorilla router, so the table can replace
// walking the router. It also checks that the path recorded for each route resolves to the same API-only permission grant as the
// gorilla template the API-only check reads today: a mismatch would lock API-only users out of that route once the table is the
// source of the template.
func TestRouteTableMatchesGorilla(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	_, router := newAPIHandler(t, config.TestConfig(), productionFeatureRoutes(t, svc))
	routes := endpointer.RoutesOf(router).Routes()

	type gorillaRoute struct {
		name, template string
		methods        []string
	}
	var walked []gorillaRoute
	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := route.GetPathTemplate()
		require.NoError(t, err)
		methods, err := route.GetMethods()
		require.NoError(t, err)
		walked = append(walked, gorillaRoute{name: route.GetName(), template: tpl, methods: methods})
		return nil
	}))

	t.Run("table lists every gorilla route in order", func(t *testing.T) {
		recorded := make([]gorillaRoute, 0, len(routes))
		for _, rt := range routes {
			recorded = append(recorded, gorillaRoute{name: rt.Name, template: rt.Template(), methods: rt.Methods})
		}
		// Report the first divergence only: a route registered on gorilla directly shifts every route after it.
		for i := range max(len(walked), len(recorded)) {
			var gorilla, table gorillaRoute
			if i < len(walked) {
				gorilla = walked[i]
			}
			if i < len(recorded) {
				table = recorded[i]
			}
			require.Equalf(t, gorilla, table, "route %d: register routes through the endpointer, not on the gorilla router directly", i)
		}
	})

	t.Run("recorded path grants the same API-only permission as the gorilla template", func(t *testing.T) {
		granted := make(map[string]struct{})
		for _, rt := range routes {
			for _, method := range rt.Methods {
				fromPath := fleet.NewAPIEndpointFromTpl(method, rt.Path).Fingerprint()
				fromTemplate := fleet.NewAPIEndpointFromTpl(method, rt.Template()).Fingerprint()
				assert.Equalf(t, fromTemplate, fromPath, "%s %s", method, rt.Path)
				granted[fromPath] = struct{}{}
			}
		}
		for _, e := range apiendpoints.GetAPIEndpoints() {
			if strings.HasPrefix(e.Path, "/api/v1/fleet/scim/") && e.Path != "/api/v1/fleet/scim/details" {
				continue // the SCIM library routes these inside the handler mounted on the SCIM prefix
			}
			assert.Containsf(t, granted, e.Fingerprint(), "catalog endpoint %s %s is not a recorded route", e.Method, e.Path)
		}
	})
}
