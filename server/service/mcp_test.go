package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/fleetmcp"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyMCPAvailable(t *testing.T) {
	ds := new(mock.Store)
	premium, premiumCtx := newTestServiceWithConfig(t, ds, config.TestConfig(), nil, nil, &TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium}, SkipCreateTestUsers: true,
	})
	free, freeCtx := newTestServiceWithConfig(t, ds, config.TestConfig(), nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
	testErr := errors.New("test err")

	for _, tc := range []struct {
		name    string
		svc     fleet.Service
		ctx     context.Context
		cfg     *fleet.AppConfig
		cfgErr  error
		wantErr func(*testing.T, error)
	}{
		{"available", premium, premiumCtx, &fleet.AppConfig{}, nil, func(t *testing.T, err error) { require.NoError(t, err) }},
		{"app config error", premium, premiumCtx, nil, testErr, func(t *testing.T, err error) { require.ErrorIs(t, err, testErr) }},
		{"turned off", premium, premiumCtx, &fleet.AppConfig{ServerSettings: fleet.ServerSettings{MCPDisabled: true}}, nil, func(t *testing.T, err error) {
			var badRequest *fleet.BadRequestError
			require.ErrorAs(t, err, &badRequest)
			assert.Contains(t, badRequest.Message, "server_settings.mcp_disabled")
		}},
		{"Fleet Free", free, freeCtx, &fleet.AppConfig{}, nil, func(t *testing.T, err error) { require.ErrorIs(t, err, fleet.ErrMissingLicense) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) { return tc.cfg, tc.cfgErr }
			authzCtx := &authz_ctx.AuthorizationContext{}
			err := tc.svc.VerifyMCPAvailable(authz_ctx.NewContext(tc.ctx, authzCtx))
			tc.wantErr(t, err)
			// On success the request still has to authorize as usual.
			assert.Equal(t, err != nil, authzCtx.Checked())
		})
	}
}

func TestSameOrigin(t *testing.T) {
	const serverURL = "https://fleet.example.com/"
	for origin, want := range map[string]bool{
		"https://fleet.example.com":      true,
		"https://FLEET.example.com:443":  true,
		"https://fleet.example.com:8443": false,
		"http://fleet.example.com":       false,
		"https://evil.example.com":       false,
		"null":                           false,
	} {
		assert.Equal(t, want, sameOrigin(origin, serverURL), origin)
	}
	assert.False(t, sameOrigin("https://fleet.example.com", ""), "no server URL means no Origin is trusted")
	assert.True(t, sameOrigin("http://localhost:8080", "http://localhost:8080/fleet"))
}

// The internal listener serves the root handler, which only answers under the
// URL prefix, and the waits must come from this server's live query period.
func TestNewEmbeddedFleetClient(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "7s")

	var gotPath string
	api := http.NewServeMux()
	api.HandleFunc("/api/v1/fleet/hosts/count", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]int{"count": 2})
	})
	root := http.NewServeMux()
	root.Handle("/fleet/", http.StripPrefix("/fleet", api))

	conn := NewMCPConn()
	conn.Start(t.Context(), root)
	t.Cleanup(func() { _ = conn.Shutdown(context.Background()) })

	fc, err := newEmbeddedFleetClient(conn, "/fleet", slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	campaignWait, singleHostTimeout := fc.LiveQueryWaits()
	assert.Equal(t, 7*time.Second, campaignWait)
	assert.Equal(t, 37*time.Second, singleHostTimeout)

	n, err := fc.GetHostCount(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, "/api/v1/fleet/hosts/count", gotPath)
}

func TestMCPUserLogHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(mcpUserLogHandler{slog.NewTextHandler(&buf, nil)})
	v := &viewer.Viewer{User: &fleet.User{Email: "mcp@example.com"}}

	logger.InfoContext(logging.WithUserEmailer(t.Context(), v), "tool invoked", "tool", "get_host")
	assert.Contains(t, buf.String(), "user=mcp@example.com")
}

// An admin can only grant catalogued endpoints, so a tool calling any other
// route would be hidden from every restricted user.
func TestMCPToolRoutesAreInCatalog(t *testing.T) {
	for tool, routes := range fleetmcp.ToolRoutes {
		for _, r := range routes {
			assert.True(t, apiendpoints.IsInCatalog(fleet.NewAPIEndpointFromTpl(r.Method, r.Path).Fingerprint()), "%s: %s", tool, r)
		}
	}
}

func TestGrantedMCPRoutes(t *testing.T) {
	assert.Nil(t, grantedMCPRoutes(&fleet.User{}))
	assert.Nil(t, grantedMCPRoutes(&fleet.User{APIOnly: true}))

	granted := grantedMCPRoutes(&fleet.User{APIOnly: true, APIEndpoints: []fleet.APIEndpointRef{
		{Method: "POST", Path: "/api/v1/fleet/mcp"},
		{Method: "GET", Path: "/api/v1/fleet/hosts/:id"},
	}})
	require.NotNil(t, granted)
	assert.True(t, granted(fleetmcp.Route{Method: "GET", Path: "/api/v1/fleet/hosts/:host_id"}), "placeholder names don't matter")
	assert.False(t, granted(fleetmcp.Route{Method: "POST", Path: "/api/v1/fleet/hosts/:id"}))
	assert.False(t, granted(fleetmcp.Route{Method: "GET", Path: "/api/v1/fleet/hosts"}))
}
