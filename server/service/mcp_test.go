package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/fleetmcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// The API-only check rejects uncatalogued routes before it looks at the
// embedded MCP's route set, so a tool calling one would fail for every
// restricted user.
func TestEmbeddedMCPRoutesAreInCatalog(t *testing.T) {
	for _, r := range fleetmcp.EmbeddedRoutes {
		assert.True(t, apiendpoints.IsInCatalog(fleet.NewAPIEndpointFromTpl(r.Method, r.Path).Fingerprint()), "%s %s", r.Method, r.Path)
	}
}
