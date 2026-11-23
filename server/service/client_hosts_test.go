package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostByIdentifierRejectsResponseWithoutHost(t *testing.T) {
	t.Parallel()

	// A host named "count" can't be looked up by hostname: HostByIdentifier("count") requests GET /hosts/count, which is the count
	// endpoint, so the response has a count and no host.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/latest/fleet/hosts/count", r.URL.Path)
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

func TestHostByIdentifierAcceptsHostID(t *testing.T) {
	t.Parallel()

	// GET /hosts/:id resolves a host ID first, like any other identifier, so the client takes the host it returns.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"host": map[string]any{"id": 42, "hardware_serial": "1234567"}})
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, true, "", "")
	require.NoError(t, err)
	client.SetToken("test-token")

	for _, identifier := range []string{"42", "1234567"} {
		host, err := client.HostByIdentifier(identifier)
		require.NoError(t, err, "identifier %q", identifier)
		require.Equal(t, uint(42), host.ID, "identifier %q", identifier)
	}
}
