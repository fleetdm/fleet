package google_cloud_identity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// staticTokenSource yields a fixed bearer for tests.
type staticTokenSource struct{ tok string }

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.tok, Expiry: time.Now().Add(time.Hour)}, nil
}

func newTestClient(t *testing.T, h http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	c := NewClient(
		context.Background(),
		staticTokenSource{tok: "test-bearer"},
		WithCloudIdentityBase(srv.URL),
		WithDirectoryBase(srv.URL+"/admin/directory/v1"),
	)
	return c, srv
}

// expectBearer asserts the request carries our bearer token (i.e. the
// oauth2.NewClient layer is doing its job).
func expectBearer(t *testing.T, r *http.Request) {
	t.Helper()
	got := r.Header.Get("Authorization")
	assert.Equal(t, "Bearer test-bearer", got, "auth header")
}

func TestGetCustomer(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBearer(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/admin/directory/v1/customers/my_customer", r.URL.Path)
		_ = json.NewEncoder(w).Encode(Customer{ID: "C0xxxxxxx", CustomerDomain: "example.com"})
	}))

	cust, err := c.GetCustomer(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "C0xxxxxxx", cust.ID)
	assert.Equal(t, "example.com", cust.CustomerDomain)
}

func TestLookupDeviceUserByRawResourceID(t *testing.T) {
	const resID = "f60acecb-c136-4965-9b1b-ba089f75eede"

	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBearer(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1beta1/devices/-/deviceUsers:lookup", r.URL.Path)
		assert.Equal(t, resID, r.URL.Query().Get("rawResourceId"))
		_ = json.NewEncoder(w).Encode(DeviceUserLookupResponse{
			Names: []string{"devices/dev-1/deviceUsers/user-1"},
		})
	}))

	resp, err := c.LookupDeviceUserByRawResourceID(context.Background(), resID)
	require.NoError(t, err)
	require.Len(t, resp.Names, 1)
	assert.Equal(t, "devices/dev-1/deviceUsers/user-1", resp.Names[0])
}

func TestLookupDeviceUserByEmail(t *testing.T) {
	const email = "user+alias@example.com"

	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1beta1/devices/-/deviceUsers:lookup", r.URL.Path)
		// The query parameter is named `userId` in the lookup-by-email shape.
		assert.Equal(t, email, r.URL.Query().Get("userId"))
		_ = json.NewEncoder(w).Encode(DeviceUserLookupResponse{
			Names: []string{
				"devices/dev-1/deviceUsers/user-1",
				"devices/dev-2/deviceUsers/user-2",
			},
		})
	}))

	resp, err := c.LookupDeviceUserByEmail(context.Background(), email)
	require.NoError(t, err)
	assert.Len(t, resp.Names, 2)
}

func TestPatchClientState(t *testing.T) {
	want := ClientState{
		CustomID:        "host-uuid-1",
		ComplianceState: ComplianceStateNonCompliant,
		Managed:         ManagedStateManaged,
		ScoreReason:     "Failing Fleet policies: policy_42",
	}

	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectBearer(t, r)
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t,
			"/v1beta1/devices/dev-1/deviceUsers/user-1/clientState/fleet-0xxxxxxx",
			r.URL.Path,
		)
		assert.Equal(t, "customers/my_customer", r.URL.Query().Get("customer"))
		assert.Equal(t, "complianceState,managed,scoreReason", r.URL.Query().Get("updateMask"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var got ClientState
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, want.CustomID, got.CustomID)
		assert.Equal(t, want.ComplianceState, got.ComplianceState)
		assert.Equal(t, want.Managed, got.Managed)
		assert.Equal(t, want.ScoreReason, got.ScoreReason)

		// Echo back an updated etag.
		resp := ClientState{
			Name:            r.URL.Path[len("/v1beta1/"):],
			Etag:            "new-etag",
			ComplianceState: got.ComplianceState,
			Managed:         got.Managed,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))

	got, err := c.PatchClientState(context.Background(), PatchClientStateRequest{
		DeviceUserResource: "devices/dev-1/deviceUsers/user-1",
		Partner:            "fleet-0xxxxxxx",
		Customer:           "customers/my_customer",
		State:              &want,
		UpdateMask:         "complianceState,managed,scoreReason",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-etag", got.Etag)
}

func TestPatchClientStateValidation(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when validation fails")
	}))

	cases := map[string]PatchClientStateRequest{
		"missing DeviceUserResource": {
			Partner:    "fleet-0xxxxxxx",
			Customer:   "customers/my_customer",
			State:      &ClientState{},
			UpdateMask: "complianceState",
		},
		"missing Partner": {
			DeviceUserResource: "devices/d/deviceUsers/u",
			Customer:           "customers/my_customer",
			State:              &ClientState{},
			UpdateMask:         "complianceState",
		},
		"missing Customer": {
			DeviceUserResource: "devices/d/deviceUsers/u",
			Partner:            "fleet-0xxxxxxx",
			State:              &ClientState{},
			UpdateMask:         "complianceState",
		},
		"missing State": {
			DeviceUserResource: "devices/d/deviceUsers/u",
			Partner:            "fleet-0xxxxxxx",
			Customer:           "customers/my_customer",
			UpdateMask:         "complianceState",
		},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := c.PatchClientState(context.Background(), in)
			require.Error(t, err)
		})
	}
}

func TestPermissionDeniedSurfacing(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"PERMISSION_DENIED","status":"PERMISSION_DENIED"}}`))
	}))

	_, err := c.PatchClientState(context.Background(), PatchClientStateRequest{
		DeviceUserResource: "devices/d/deviceUsers/u",
		Partner:            "fleet-0xxxxxxx",
		Customer:           "customers/my_customer",
		State:              &ClientState{ComplianceState: ComplianceStateCompliant},
		UpdateMask:         "complianceState",
	})
	require.Error(t, err)
	assert.True(t, IsPermissionDenied(err), "IsPermissionDenied should match 403 response")

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.Contains(t, apiErr.Body, "PERMISSION_DENIED")
}

func TestNon200Surfacing(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	_, err := c.GetCustomer(context.Background())
	require.Error(t, err)
	assert.False(t, IsPermissionDenied(err), "500 must not be flagged as permission-denied")

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	assert.Contains(t, strings.ToLower(apiErr.Body), "upstream error")
}
