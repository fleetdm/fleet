package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRemovedEndpointsReturnGone(t *testing.T) {
	// Test-only entries exercise the methods the real list doesn't use yet.
	testOnly := []removedEndpoint{
		{Method: http.MethodGet, Path: "/removed-endpoint-test"},
		{Method: http.MethodPut, Path: "/removed-endpoint-test"},
		{Method: http.MethodPatch, Path: "/removed-endpoint-test"},
		{Method: http.MethodDelete, Path: "/removed-endpoint-test"},
	}
	testOnlyMessage := func(method, path string) (string, bool) { return "test message for " + method + " " + path, true }

	type check struct {
		removedEndpoint
		want string
	}
	var checks []check
	for _, re := range removedEndpoints {
		msg, ok := removedEndpointMessage(re.Method, re.Path)
		require.True(t, ok)
		require.Equal(t, `The "`+re.Method+` `+re.Path+`" endpoint is not available anymore. What to use instead: `+removednames.GuideURL, msg)
		checks = append(checks, check{re, msg})
	}
	for _, re := range testOnly {
		msg, _ := testOnlyMessage(re.Method, re.Path)
		checks = append(checks, check{re, msg})
	}

	for _, tier := range []string{fleet.TierFree, fleet.TierPremium} {
		t.Run(tier, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{}, nil
			}
			routeSvc, _ := newTestService(t, ds, nil, nil)
			_, server := RunServerForTestsWithDS(t, ds, &TestServerOpts{
				License:             &fleet.LicenseInfo{Tier: tier},
				SkipCreateTestUsers: true,
				FeatureRoutes: []endpointer.HandlerRoutesFunc{func(r *mux.Router, opts []kithttp.ServerOption) {
					registerRemovedEndpoints(newNoAuthEndpointer(routeSvc, opts, r, "v1", "2022-04"), testOnly, testOnlyMessage)
				}},
			})
			defer server.Close()

			for _, c := range checks {
				for _, prefix := range []string{"/api/v1/fleet", "/api/latest/fleet"} {
					// No auth header, and a body the old endpoint would have accepted.
					req, err := http.NewRequestWithContext(t.Context(), c.Method, server.URL+prefix+c.Path,
						strings.NewReader(`{"external_host_identifier":"x"}`))
					require.NoError(t, err)
					resp, err := fleethttp.NewClient().Do(req)
					require.NoError(t, err)

					var body struct {
						Message string `json:"message"`
						Errors  []struct {
							Name   string `json:"name"`
							Reason string `json:"reason"`
						} `json:"errors"`
					}
					require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
					resp.Body.Close()

					require.Equal(t, http.StatusGone, resp.StatusCode, c.Method+" "+prefix+c.Path)
					require.Equal(t, c.want, body.Message, c.Method+" "+prefix+c.Path)
					require.Len(t, body.Errors, 1)
					require.Equal(t, "base", body.Errors[0].Name)
					require.Equal(t, c.want, body.Errors[0].Reason)
				}
			}
		})
	}

	t.Run("missing removednames entry panics at registration", func(t *testing.T) {
		ne := newNoAuthEndpointer(nil, nil, mux.NewRouter(), "v1")
		require.Panics(t, func() {
			registerRemovedEndpoints(ne, []removedEndpoint{{Method: http.MethodPost, Path: "/not-in-removednames"}}, removedEndpointMessage)
		})
	})
}
