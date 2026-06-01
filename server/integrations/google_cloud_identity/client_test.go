package google_cloud_identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudidentity "google.golang.org/api/cloudidentity/v1beta1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// newTestClient wires a Client to point at a local httptest.Server. The
// official SDK obeys option.WithEndpoint, so this is sufficient — auth is
// skipped via option.WithoutAuthentication.
func newTestClient(t *testing.T, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	c, err := NewClient(
		context.Background(),
		option.WithEndpoint(srv.URL),
		option.WithoutAuthentication(),
	)
	require.NoError(t, err)
	return c, srv
}

func TestFindDeviceBySerial_Match(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1beta1/devices", r.URL.Path)
		assert.Equal(t, "customers/my_customer", r.URL.Query().Get("customer"))
		assert.Contains(t, r.URL.Query().Get("filter"), "H176YH")

		_ = json.NewEncoder(w).Encode(cloudidentity.ListDevicesResponse{
			Devices: []*cloudidentity.Device{
				{
					Name:         "devices/abc-encoded%3D",
					SerialNumber: "H176YH",
					LastSyncTime: "2026-05-29T13:23:01.550Z",
				},
			},
		})
	}))

	d, err := c.FindDeviceBySerial(context.Background(), "H176YH")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "devices/abc-encoded%3D", d.Name)
	assert.Equal(t, "H176YH", d.SerialNumber)
}

func TestFindDeviceBySerial_NoMatch(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(cloudidentity.ListDevicesResponse{Devices: nil})
	}))

	d, err := c.FindDeviceBySerial(context.Background(), "NOTREAL")
	require.NoError(t, err, "no match is not an error")
	assert.Nil(t, d)
}

func TestFindDeviceBySerial_MultipleMatchesPrefersMostRecent(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(cloudidentity.ListDevicesResponse{
			Devices: []*cloudidentity.Device{
				{Name: "devices/older", SerialNumber: "DUP", LastSyncTime: "2026-01-01T00:00:00Z"},
				{Name: "devices/newer", SerialNumber: "DUP", LastSyncTime: "2026-05-29T00:00:00Z"},
				{Name: "devices/middle", SerialNumber: "DUP", LastSyncTime: "2026-03-01T00:00:00Z"},
			},
		})
	}))

	d, err := c.FindDeviceBySerial(context.Background(), "DUP")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "devices/newer", d.Name)
}

func TestListDeviceUsers(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/deviceUsers")
		_ = json.NewEncoder(w).Encode(cloudidentity.ListDeviceUsersResponse{
			DeviceUsers: []*cloudidentity.DeviceUser{
				{Name: "devices/d/deviceUsers/u1", UserEmail: "alice@example.com"},
				{Name: "devices/d/deviceUsers/u2", UserEmail: "bob@example.com"},
			},
		})
	}))

	users, err := c.ListDeviceUsers(context.Background(), "devices/d")
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, "alice@example.com", users[0].UserEmail)
	assert.Equal(t, "bob@example.com", users[1].UserEmail)
}

func TestPatchClientState(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t,
			"/v1beta1/devices/d/deviceUsers/u/clientStates/fleet-0xxxxxxx",
			r.URL.Path,
		)
		assert.Equal(t, "customers/my_customer", r.URL.Query().Get("customer"))
		assert.Equal(t, "complianceState,managed", r.URL.Query().Get("updateMask"))

		var got cloudidentity.ClientState
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got)) {
			return
		}
		assert.Equal(t, "host-uuid-1", got.CustomId)
		assert.Equal(t, "NON_COMPLIANT", got.ComplianceState)
		assert.Equal(t, "MANAGED", got.Managed)

		// PATCH on Cloud Identity returns an Operation. Inline the new
		// ClientState (with etag) in Response.
		newState := cloudidentity.ClientState{
			Name: r.URL.Path[len("/v1beta1/"):],
			Etag: "new-etag",
		}
		respBytes, _ := json.Marshal(newState)
		_ = json.NewEncoder(w).Encode(cloudidentity.Operation{
			Done:     true,
			Response: respBytes,
		})
	}))

	state := &cloudidentity.ClientState{
		CustomId:        "host-uuid-1",
		ComplianceState: "NON_COMPLIANT",
		Managed:         "MANAGED",
	}
	op, err := c.PatchClientState(
		context.Background(),
		"devices/d/deviceUsers/u",
		"fleet-0xxxxxxx",
		state,
		"complianceState,managed",
	)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.True(t, op.Done)
	assert.Equal(t, "new-etag", etagFromOperation(op))
}

func TestPatchClientStateValidation(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when validation fails")
	}))

	state := &cloudidentity.ClientState{ComplianceState: "COMPLIANT"}

	t.Run("missing deviceUserName", func(t *testing.T) {
		_, err := c.PatchClientState(context.Background(), "", "fleet-x", state, "complianceState")
		require.Error(t, err)
	})
	t.Run("missing partner", func(t *testing.T) {
		_, err := c.PatchClientState(context.Background(), "devices/d/deviceUsers/u", "", state, "complianceState")
		require.Error(t, err)
	})
	t.Run("missing state", func(t *testing.T) {
		_, err := c.PatchClientState(context.Background(), "devices/d/deviceUsers/u", "fleet-x", nil, "complianceState")
		require.Error(t, err)
	})
}

func TestPermissionDeniedSurfacing(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"PERMISSION_DENIED","status":"PERMISSION_DENIED"}}`))
	}))

	_, err := c.PatchClientState(context.Background(),
		"devices/d/deviceUsers/u",
		"fleet-x",
		&cloudidentity.ClientState{ComplianceState: "COMPLIANT"},
		"complianceState",
	)
	require.Error(t, err)
	assert.True(t, IsPermissionDenied(err), "IsPermissionDenied should match 403 response")

	var apiErr *googleapi.Error
	require.ErrorAs(t, err, &apiErr)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.Code)
	assert.Contains(t, apiErr.Body, "PERMISSION_DENIED")
}

func TestNon200Surfacing(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	_, err := c.FindDeviceBySerial(context.Background(), "X")
	require.Error(t, err)
	assert.False(t, IsPermissionDenied(err), "500 must not be flagged as permission-denied")

	var apiErr *googleapi.Error
	require.ErrorAs(t, err, &apiErr)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Contains(t, strings.ToLower(apiErr.Body), "upstream error")
}
