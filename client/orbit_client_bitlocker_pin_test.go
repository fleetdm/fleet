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

func TestGetDiskEncryptionPINDetails(t *testing.T) {
	t.Parallel()

	const pin = `my "PIN" \ 123`

	for _, tc := range []struct {
		name         string
		status       int
		wantPIN      string
		wantUUID     string
		wantNotFound bool
	}{
		{name: "collected", status: http.StatusOK, wantPIN: pin, wantUUID: "request-uuid"},
		{name: "nothing to collect", status: http.StatusNotFound, wantNotFound: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/fleet/orbit/disk_encryption_pin/details", r.URL.Path)
				w.WriteHeader(tc.status)
				if tc.wantNotFound {
					_, _ = w.Write([]byte(`{"message":"not found"}`))
					return
				}
				assert.NoError(t, json.NewEncoder(w).Encode(fleet.OrbitGetDiskEncryptionPINDetailsResponse{
					PIN:         tc.wantPIN,
					RequestUUID: tc.wantUUID,
				}))
			}))
			defer srv.Close()

			_, nodeKeyPath := newNodeKeyFile(t, "node-key")
			oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

			gotPIN, gotUUID, err := oc.GetDiskEncryptionPINDetails()
			if tc.wantNotFound {
				require.True(t, IsNotFoundErr(err))
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantPIN, gotPIN)
			require.Equal(t, tc.wantUUID, gotUUID)
		})
	}
}

func TestSetDiskEncryptionPINResult(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		status       int
		wantNotFound bool
	}{
		{name: "recorded", status: http.StatusNoContent},
		{name: "no longer wanted", status: http.StatusNotFound, wantNotFound: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got fleet.OrbitPostDiskEncryptionPINResultRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/fleet/orbit/disk_encryption_pin/result", r.URL.Path)
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			_, nodeKeyPath := newNodeKeyFile(t, "node-key")
			oc := newReenrollTestClient(t, srv.URL, nodeKeyPath)

			err := oc.SetDiskEncryptionPINResult("request-uuid", fleet.BitLockerPINRequestFailed, "PIN already set")
			if tc.wantNotFound {
				require.True(t, IsNotFoundErr(err))
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, fleet.OrbitPostDiskEncryptionPINResultRequest{
				OrbitNodeKey: "node-key",
				RequestUUID:  "request-uuid",
				Outcome:      fleet.BitLockerPINRequestFailed,
				ClientError:  "PIN already set",
			}, got)
		})
	}
}
