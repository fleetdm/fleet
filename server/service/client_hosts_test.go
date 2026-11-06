package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostByIdentifierRejectsResponseWithoutHost(t *testing.T) {
	t.Parallel()

	// GET /hosts/count serves the identifier "count", and its response has no host.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/latest/fleet/hosts/count", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 3})
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, true, "", "")
	require.NoError(t, err)
	client.SetToken("test-token")

	host, err := client.HostByIdentifier("count")
	require.Nil(t, host)
	require.True(t, IsNotFoundErr(err), "fleetctl reports a not found error as a missing host: %v", err)
}
