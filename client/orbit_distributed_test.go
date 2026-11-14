package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedReadWrite(t *testing.T) {
	_, nodeKeyPath := newNodeKeyFile(t, "orbit-key")

	var gotWrite DistributedWriteRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The node key must be present both in the body and in the
		// Authorization header, so the request authenticates regardless of the
		// server's allow_body_auth_fallback setting.
		assert.Equal(t, "NodeKey orbit-key", r.Header.Get("Authorization"))

		switch r.URL.Path {
		case "/api/osquery/distributed/read":
			var req struct {
				NodeKey string `json:"node_key"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "orbit-key", req.NodeKey)
			assert.NoError(t, json.NewEncoder(w).Encode(DistributedReadResponse{
				Queries:    map[string]string{"fleet_distributed_query_1": "SELECT 1"},
				Discovery:  map[string]string{"fleet_distributed_query_1": "SELECT 1"},
				Accelerate: 10,
			}))
		case "/api/osquery/distributed/write":
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotWrite))
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

	read, err := oc.DistributedRead()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"fleet_distributed_query_1": "SELECT 1"}, read.Queries)
	assert.Equal(t, uint(10), read.Accelerate)

	err = oc.DistributedWrite(&DistributedWriteRequest{
		Results: fleet.OsqueryDistributedQueryResults{
			"fleet_distributed_query_1": {{"1": "1"}},
		},
		Statuses: map[string]fleet.OsqueryStatus{"fleet_distributed_query_1": fleet.StatusOK},
		Messages: map[string]string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "orbit-key", gotWrite.NodeKey)
	require.Contains(t, gotWrite.Results, "fleet_distributed_query_1")
	assert.Equal(t, []map[string]string{{"1": "1"}}, gotWrite.Results["fleet_distributed_query_1"])
	assert.Equal(t, fleet.StatusOK, gotWrite.Statuses["fleet_distributed_query_1"])
}

func TestOsqueryConfig(t *testing.T) {
	_, nodeKeyPath := newNodeKeyFile(t, "orbit-key")

	type gotRequest struct {
		body           map[string]any
		fallbackChange string
	}
	var got []gotRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/osquery/config", r.URL.Path)
		assert.Equal(t, "NodeKey orbit-key", r.Header.Get("Authorization"))
		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		got = append(got, gotRequest{body: body, fallbackChange: r.Header.Get(fleet.OsqueryConfigFallbackChangeHeader)})
		_, _ = w.Write([]byte(`{"etag":"v1","options":{}}`))
	}))
	defer srv.Close()

	oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

	raw, err := oc.OsqueryConfig("", false)
	require.NoError(t, err)
	assert.JSONEq(t, `{"etag":"v1","options":{}}`, string(raw))

	_, err = oc.OsqueryConfig("v1", true)
	require.NoError(t, err)

	// The etag field is always sent, opting in to conditional requests.
	assert.Equal(t, []gotRequest{
		{body: map[string]any{"node_key": "orbit-key", "etag": ""}},
		{body: map[string]any{"node_key": "orbit-key", "etag": "v1"}, fallbackChange: "1"},
	}, got)
}
