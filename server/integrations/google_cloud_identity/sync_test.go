package google_cloud_identity

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudidentity "google.golang.org/api/cloudidentity/v1beta1"
	"google.golang.org/api/option"
)

func boolPtr(b bool) *bool     { return &b }
func strPtr(s string) *string  { return &s }
func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})) }

// recordingHandler captures the requests an httptest server received.
type recordingHandler struct {
	mu       sync.Mutex
	requests []*recordedRequest
	handler  func(w http.ResponseWriter, r *http.Request, body []byte)
}

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   []byte
}

func (h *recordingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	h.mu.Lock()
	h.requests = append(h.requests, &recordedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
		Body:   body,
	})
	h.mu.Unlock()
	if h.handler != nil {
		h.handler(w, r, body)
	}
}

func (h *recordingHandler) recordedRequests() []*recordedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*recordedRequest, len(h.requests))
	copy(out, h.requests)
	return out
}

func newRecordingSyncer(t *testing.T, ds fleet.Datastore, handler func(w http.ResponseWriter, r *http.Request, body []byte)) (*Syncer, *recordingHandler) {
	t.Helper()
	h := &recordingHandler{handler: handler}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	client, err := NewClient(
		context.Background(),
		option.WithEndpoint(srv.URL),
		option.WithoutAuthentication(),
	)
	require.NoError(t, err)

	cfg := config.GoogleCloudIdentityConfig{
		ServiceAccountJSON: "ignored",
		ImpersonatedAdmin:  "admin@example.com",
		CustomerID:         "C0xxxxxxx",
		PartnerSuffix:      "fleet",
		WorkspaceDomains:   "example.com",
	}
	return NewSyncer(ds, client, cfg, testLogger()), h
}

// encodeOperation wraps a ClientState (with etag) in an Operation, mirroring
// how Cloud Identity returns PATCH results.
func encodeOperation(w http.ResponseWriter, etag string) {
	resp, _ := json.Marshal(cloudidentity.ClientState{Etag: etag})
	_ = json.NewEncoder(w).Encode(cloudidentity.Operation{Done: true, Response: resp})
}

func TestSyncHost_NoRows_NoOp(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return nil, nil
	}
	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		t.Fatalf("no HTTP request expected; got %s %s", r.Method, r.URL.Path)
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid", HardwareSerial: "H176YH"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))
	assert.Empty(t, h.recordedRequests())
}

func TestSyncHost_FirstSync_ResolvesAndPatches(t *testing.T) {
	const (
		serial     = "H176YH"
		deviceName = "devices/abc-encoded%3D"
		userName   = "devices/abc-encoded%3D/deviceUsers/user-1"
	)

	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{
				HostID:             hostID,
				WorkspaceEmail:     "robbie@example.com",
				PartnerSuffix:      "fleet",
				DeviceUserResource: nil,
			},
		}, nil
	}
	var resolvedCalls int
	ds.SetHostGoogleCloudIdentityResolvedDeviceUserFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix, deviceUserResource string) error {
		resolvedCalls++
		assert.Equal(t, "robbie@example.com", workspaceEmail)
		assert.Equal(t, "fleet", partnerSuffix)
		assert.Equal(t, userName, deviceUserResource)
		return nil
	}
	var setStateCalls int
	ds.SetHostGoogleCloudIdentityClientStateFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string, managed, compliant bool, scoreReason, etag string) error {
		setStateCalls++
		assert.Equal(t, "robbie@example.com", workspaceEmail)
		assert.True(t, managed)
		assert.True(t, compliant)
		assert.Equal(t, "", scoreReason)
		assert.Equal(t, "etag-after-patch", etag)
		return nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1beta1/devices":
			assert.Contains(t, r.URL.Query().Get("filter"), serial)
			_ = json.NewEncoder(w).Encode(cloudidentity.ListDevicesResponse{
				Devices: []*cloudidentity.Device{{Name: deviceName, SerialNumber: serial, LastSyncTime: "2026-05-29T00:00:00Z"}},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/deviceUsers"):
			_ = json.NewEncoder(w).Encode(cloudidentity.ListDeviceUsersResponse{
				DeviceUsers: []*cloudidentity.DeviceUser{
					{Name: "devices/abc-encoded%3D/deviceUsers/other", UserEmail: "other@example.com"},
					{Name: userName, UserEmail: "robbie@example.com"},
				},
			})
		case r.Method == http.MethodPatch:
			// Partner segment is customer-id-first per Google's REST docs
			// (verified empirically; suffix-first returns 403).
			assert.Contains(t, r.URL.Path, "/clientStates/0xxxxxxx-fleet")
			var got cloudidentity.ClientState
			require.NoError(t, json.Unmarshal(body, &got))
			assert.Equal(t, "MANAGED", got.Managed)
			assert.Equal(t, "COMPLIANT", got.ComplianceState)
			assert.Equal(t, "host-uuid", got.CustomId)
			encodeOperation(w, "etag-after-patch")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	host := &fleet.Host{ID: 7, UUID: "host-uuid", HardwareSerial: serial}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))

	assert.Equal(t, 1, resolvedCalls, "deviceUser cached once")
	assert.Equal(t, 1, setStateCalls, "last-known state recorded once")

	reqs := h.recordedRequests()
	require.Len(t, reqs, 3, "devices.list + deviceUsers.list + clientStates.patch")
}

func TestSyncHost_NoChangeNoPatch(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{
				HostID:             hostID,
				WorkspaceEmail:     "user@example.com",
				PartnerSuffix:      "fleet",
				DeviceUserResource: strPtr("devices/d/deviceUsers/u"),
				LastCompliant:      boolPtr(true),
				LastManaged:        boolPtr(true),
				LastScoreReason:    strPtr(""),
				LastEtag:           strPtr("etag-old"),
			},
		}, nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		t.Fatalf("no HTTP expected; state unchanged: %s %s", r.Method, r.URL.Path)
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid", HardwareSerial: "X"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))
	assert.Empty(t, h.recordedRequests())
}

func TestSyncHost_StateChangedPatches(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{
				HostID:             hostID,
				WorkspaceEmail:     "user@example.com",
				PartnerSuffix:      "fleet",
				DeviceUserResource: strPtr("devices/d/deviceUsers/u"),
				LastCompliant:      boolPtr(true),
				LastManaged:        boolPtr(true),
				LastScoreReason:    strPtr(""),
				LastEtag:           strPtr("etag-old"),
			},
		}, nil
	}
	var setCalls int
	ds.SetHostGoogleCloudIdentityClientStateFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string, managed, compliant bool, scoreReason, etag string) error {
		setCalls++
		assert.False(t, compliant)
		assert.Equal(t, "policy failed", scoreReason)
		return nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		assert.Equal(t, http.MethodPatch, r.Method)
		var got cloudidentity.ClientState
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, "NON_COMPLIANT", got.ComplianceState)
		assert.Equal(t, "policy failed", got.ScoreReason)
		assert.Equal(t, "etag-old", got.Etag)
		encodeOperation(w, "etag-new")
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid", HardwareSerial: "X"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, false, "policy failed"))

	assert.Equal(t, 1, setCalls)
	assert.Len(t, h.recordedRequests(), 1, "single PATCH; deviceUser already resolved")
}

func TestSyncHost_DeviceNotInCloudIdentity_SkipsPatch(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{HostID: hostID, WorkspaceEmail: "u@example.com", PartnerSuffix: "fleet"},
		}, nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		_ = json.NewEncoder(w).Encode(cloudidentity.ListDevicesResponse{Devices: nil})
	})

	host := &fleet.Host{ID: 1, UUID: "u", HardwareSerial: "UNKNOWN"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))

	reqs := h.recordedRequests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "/v1beta1/devices", reqs[0].Path)
}

func TestSyncHost_UserNotOnDevice_SkipsPatch(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{HostID: hostID, WorkspaceEmail: "robbie@example.com", PartnerSuffix: "fleet"},
		}, nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		switch {
		case r.URL.Path == "/v1beta1/devices":
			_ = json.NewEncoder(w).Encode(cloudidentity.ListDevicesResponse{
				Devices: []*cloudidentity.Device{{Name: "devices/d", SerialNumber: "S", LastSyncTime: "2026-01-01T00:00:00Z"}},
			})
		case strings.HasSuffix(r.URL.Path, "/deviceUsers"):
			_ = json.NewEncoder(w).Encode(cloudidentity.ListDeviceUsersResponse{
				DeviceUsers: []*cloudidentity.DeviceUser{{Name: "devices/d/deviceUsers/other", UserEmail: "someone@example.com"}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	host := &fleet.Host{ID: 1, UUID: "u", HardwareSerial: "S"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))

	reqs := h.recordedRequests()
	require.Len(t, reqs, 2, "list + listDeviceUsers; no PATCH because no email match")
}

func TestSyncHost_PerRowFailureDoesNotDropOthers(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{HostID: hostID, WorkspaceEmail: "good@example.com", PartnerSuffix: "fleet", DeviceUserResource: strPtr("devices/d/deviceUsers/good")},
			{HostID: hostID, WorkspaceEmail: "bad@example.com", PartnerSuffix: "fleet", DeviceUserResource: strPtr("devices/d/deviceUsers/bad")},
		}, nil
	}
	var goodSet int
	ds.SetHostGoogleCloudIdentityClientStateFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string, managed, compliant bool, scoreReason, etag string) error {
		if workspaceEmail == "good@example.com" {
			goodSet++
		}
		return nil
	}

	s, _ := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		if strings.Contains(r.URL.Path, "/bad/clientStates/") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		encodeOperation(w, "e")
	})

	host := &fleet.Host{ID: 1, UUID: "u", HardwareSerial: "S"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))
	assert.Equal(t, 1, goodSet, "good row patched despite bad row failing")
}
