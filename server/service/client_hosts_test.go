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

func TestHostByIdentifierRejectsHostIDOnlyMatch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"host": map[string]any{"id": 42, "hardware_serial": "1234567"}})
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, true, "", "")
	require.NoError(t, err)
	client.SetToken("test-token")

	for identifier, wantFound := range map[string]bool{"42": false, "1234567": true} {
		_, err := client.HostByIdentifier(identifier)
		require.Equal(t, wantFound, err == nil, "identifier %q: %v", identifier, err)
	}
}
