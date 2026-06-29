package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollSendsEUAToken(t *testing.T) {
	// nolint:gosec // not a real credential, test-only JWT fragment
	euaTokenValue := "eyJhbGciOiJSUzI1NiJ9.test-eua-token"
	const testNodeKey = "test-node-key-abc"

	testCases := []struct {
		name   string
		token  string
		assert func(t *testing.T, receivedBody fleet.EnrollOrbitRequest, rawBody []byte)
	}{
		{
			name:  "eua_token included in enroll request when set",
			token: euaTokenValue,
			assert: func(t *testing.T, receivedBody fleet.EnrollOrbitRequest, rawBody []byte) {
				require.Equal(t, euaTokenValue, receivedBody.EUAToken)
			},
		},
		{
			name:  "eua_token omitted from enroll request when empty",
			token: "",
			assert: func(t *testing.T, receivedBody fleet.EnrollOrbitRequest, rawBody []byte) {
				// Verify the eua_token key is not present in the JSON body (omitempty).
				require.Falsef(t, bytes.Contains(rawBody, []byte(`"eua_token"`)),
					"eua_token should not appear in JSON when empty, got: %s", string(rawBody))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var receivedBody fleet.EnrollOrbitRequest
			var rawBody []byte

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var err error
				rawBody, err = io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(rawBody, &receivedBody))

				resp := fleet.EnrollOrbitResponse{OrbitNodeKey: testNodeKey}
				w.Header().Set("Content-Type", "application/json")
				err = json.NewEncoder(w).Encode(resp)
				assert.NoError(t, err)
			}))
			defer srv.Close()

			oc := &OrbitClient{
				enrollSecret: "secret",
				hostInfo:     fleet.OrbitHostInfo{HardwareUUID: "uuid-1", Platform: "windows"},
			}
			oc.SetEUAToken(tc.token)
			bc, err := NewBaseClient(srv.URL, true, "", "", nil, fleet.CapabilityMap{}, nil)
			require.NoError(t, err)
			oc.BaseClient = bc

			nodeKey, err := oc.enroll()
			require.NoError(t, err)
			require.Equal(t, testNodeKey, nodeKey)
			require.Equal(t, "secret", receivedBody.EnrollSecret)
			require.Equal(t, "uuid-1", receivedBody.HardwareUUID)

			tc.assert(t, receivedBody, rawBody)
		})
	}
}
