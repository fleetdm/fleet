package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestEnrollSendsEUAToken(t *testing.T) {
	const (
		testToken   = "eyJhbGciOiJSUzI1NiJ9.test-eua-token"
		testNodeKey = "test-node-key-abc"
	)

	t.Run("eua_token included in enroll request when set", func(t *testing.T) {
		var receivedBody fleet.EnrollOrbitRequest

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &receivedBody))

			resp := fleet.EnrollOrbitResponse{OrbitNodeKey: testNodeKey}
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer srv.Close()

		oc := &OrbitClient{
			enrollSecret: "secret",
			hostInfo:     fleet.OrbitHostInfo{HardwareUUID: "uuid-1", Platform: "windows"},
			euaToken:     testToken,
		}
		bc, err := NewBaseClient(srv.URL, true, "", "", nil, fleet.CapabilityMap{}, nil)
		require.NoError(t, err)
		oc.BaseClient = bc

		nodeKey, err := oc.enroll()
		require.NoError(t, err)
		require.Equal(t, testNodeKey, nodeKey)
		require.Equal(t, testToken, receivedBody.EUAToken)
		require.Equal(t, "secret", receivedBody.EnrollSecret)
		require.Equal(t, "uuid-1", receivedBody.HardwareUUID)
	})

	t.Run("eua_token omitted from enroll request when empty", func(t *testing.T) {
		var rawBody []byte

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			rawBody, err = io.ReadAll(r.Body)
			require.NoError(t, err)

			resp := fleet.EnrollOrbitResponse{OrbitNodeKey: testNodeKey}
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}))
		defer srv.Close()

		oc := &OrbitClient{
			enrollSecret: "secret",
			hostInfo:     fleet.OrbitHostInfo{HardwareUUID: "uuid-1", Platform: "windows"},
			// euaToken not set — should be omitted from JSON (omitempty)
		}
		bc, err := NewBaseClient(srv.URL, true, "", "", nil, fleet.CapabilityMap{}, nil)
		require.NoError(t, err)
		oc.BaseClient = bc

		_, err = oc.enroll()
		require.NoError(t, err)

		// Verify the eua_token key is not present in the JSON body.
		require.False(t, bytes.Contains(rawBody, []byte(`"eua_token"`)),
			"eua_token should not appear in JSON when empty, got: %s", string(rawBody))
	})
}

func TestSetEUAToken(t *testing.T) {
	oc := &OrbitClient{}
	require.Empty(t, oc.euaToken)

	oc.SetEUAToken("some-token")
	require.Equal(t, "some-token", oc.euaToken)

	oc.SetEUAToken("")
	require.Empty(t, oc.euaToken)
}
