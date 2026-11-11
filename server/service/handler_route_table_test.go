package service

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestRouteTableMatchesGorilla checks that the endpointer's route table lists exactly the routes of the production gorilla router,
// in the same order, so code can read the table instead of walking the router.
func TestRouteTableMatchesGorilla(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	_, router := newAPIHandler(t, config.TestConfig(), productionFeatureRoutes(t, svc))

	type route struct {
		name, template string
		methods        []string
	}
	var walked, recorded []route
	require.NoError(t, router.Walk(func(r *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := r.GetPathTemplate()
		require.NoError(t, err)
		methods, err := r.GetMethods()
		require.NoError(t, err)
		walked = append(walked, route{name: r.GetName(), template: tpl, methods: methods})
		return nil
	}))
	for _, rt := range endpointer.RoutesOf(router).Routes() {
		recorded = append(recorded, route{name: rt.Name, template: rt.Template(), methods: rt.Methods})
	}

	// Report the first divergence only: a route registered on gorilla directly shifts every route after it.
	for i := range max(len(walked), len(recorded)) {
		var gorilla, table route
		if i < len(walked) {
			gorilla = walked[i]
		}
		if i < len(recorded) {
			table = recorded[i]
		}
		require.Equalf(t, gorilla, table, "route %d: register routes through the endpointer, not on the gorilla router directly", i)
	}
}

// TestRouteTableReleasesDiscardedRouter checks that a production router is collected once nothing uses it. The endpointer's route
// tables hold every recorded handler, so a handler that references its own router would keep that router, and everything its
// handlers reach, alive for the rest of the process. Tests build many routers.
func TestRouteTableReleasesDiscardedRouter(t *testing.T) {
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	routerRef := func() weak.Pointer[mux.Router] {
		_, router := newAPIHandler(t, config.TestConfig(), productionFeatureRoutes(t, svc))
		require.NotEmpty(t, endpointer.RoutesOf(router).Routes())
		return weak.Make(router)
	}()

	collected := func() bool {
		runtime.GC()
		return routerRef.Value() == nil
	}
	require.Eventually(t, collected, 5*time.Second, 10*time.Millisecond, "a discarded router must not be retained by its route table")
}
