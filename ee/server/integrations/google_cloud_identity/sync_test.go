package google_cloud_identity

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boolPtr returns a pointer to b. Helper because Fleet types use *bool for
// nullable last-known state fields.
func boolPtr(b bool) *bool { return &b }
func strPtr(s string) *string { return &s }

// testLogger returns a logger that drops output. Switch to os.Stderr for
// debugging a specific failing test.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}

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

// recordedRequests returns a snapshot of every request the handler observed.
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

	client := NewClient(
		context.Background(),
		staticTokenSource{tok: "test-bearer"},
		WithCloudIdentityBase(srv.URL),
		WithDirectoryBase(srv.URL+"/admin/directory/v1"),
	)
	cfg := config.GoogleCloudIdentityConfig{
		ServiceAccountJSON: "ignored",
		ImpersonatedAdmin:  "admin@example.com",
		CustomerID:         "C0xxxxxxx",
		PartnerSuffix:      "fleet",
		WorkspaceDomains:   "example.com",
	}
	return NewSyncer(ds, client, cfg, testLogger()), h
}

func TestSyncHost_NoRows_NoOp(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return nil, nil
	}
	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		t.Fatalf("no HTTP request expected; got %s %s", r.Method, r.URL.Path)
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))
	assert.Empty(t, h.recordedRequests(), "no rows = no PATCH")
}

func TestSyncHost_FirstSync_LazyResolveThenPatch(t *testing.T) {
	const (
		rawID      = "f60acecb-c136-4965-9b1b-ba089f75eede"
		deviceName = "devices/dev-1/deviceUsers/user-1"
	)

	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{
				HostID:             hostID,
				RawResourceID:      rawID,
				DeviceUserResource: nil, // not yet resolved
				WorkspaceEmail:     "user@example.com",
				PartnerSuffix:      "fleet",
			},
		}, nil
	}
	var resolvedCalls int
	ds.SetHostGoogleCloudIdentityResolvedDeviceUserFunc = func(ctx context.Context, hostID uint, rawResourceID, partnerSuffix, deviceUserResource string) error {
		resolvedCalls++
		assert.Equal(t, rawID, rawResourceID)
		assert.Equal(t, "fleet", partnerSuffix)
		assert.Equal(t, deviceName, deviceUserResource)
		return nil
	}
	var setStateCalls int
	ds.SetHostGoogleCloudIdentityClientStateFunc = func(ctx context.Context, hostID uint, rawResourceID, partnerSuffix string, managed, compliant bool, scoreReason, etag string) error {
		setStateCalls++
		assert.Equal(t, rawID, rawResourceID)
		assert.True(t, managed)
		assert.True(t, compliant)
		assert.Equal(t, "", scoreReason)
		assert.Equal(t, "etag-after-patch", etag)
		return nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		switch r.URL.Path {
		case "/v1beta1/devices/-/deviceUsers:lookup":
			_ = json.NewEncoder(w).Encode(DeviceUserLookupResponse{
				Names: []string{deviceName},
			})
		case "/v1beta1/" + deviceName + "/clientState/fleet-0xxxxxxx":
			// Verify the PATCH body has the expected enum values.
			var got ClientState
			require.NoError(t, json.Unmarshal(body, &got))
			assert.Equal(t, ManagedStateManaged, got.Managed)
			assert.Equal(t, ComplianceStateCompliant, got.ComplianceState)
			assert.Equal(t, "host-uuid", got.CustomID)

			resp := ClientState{Name: r.URL.Path[len("/v1beta1/"):], Etag: "etag-after-patch"}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	host := &fleet.Host{ID: 7, UUID: "host-uuid"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))

	assert.Equal(t, 1, resolvedCalls, "resolved deviceUser cached once")
	assert.Equal(t, 1, setStateCalls, "last-known state recorded once")

	reqs := h.recordedRequests()
	require.Len(t, reqs, 2, "lookup + patch = 2 calls")
	assert.Equal(t, http.MethodGet, reqs[0].Method, "lookup is GET")
	assert.Contains(t, reqs[0].Query, "rawResourceId="+rawID)
	assert.Equal(t, http.MethodPatch, reqs[1].Method, "patch is PATCH")
}

func TestSyncHost_DiffsAgainstLastKnown_NoChangeNoPatch(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{
				HostID:             hostID,
				RawResourceID:      "raw",
				DeviceUserResource: strPtr("devices/d/deviceUsers/u"),
				WorkspaceEmail:     "user@example.com",
				PartnerSuffix:      "fleet",
				LastCompliant:      boolPtr(true),
				LastManaged:        boolPtr(true),
				LastScoreReason:    strPtr(""),
				LastEtag:           strPtr("etag-old"),
			},
		}, nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		t.Fatalf("no HTTP request expected; state unchanged: got %s %s", r.Method, r.URL.Path)
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))
	assert.Empty(t, h.recordedRequests(), "no diff = no PATCH")
}

func TestSyncHost_DiffsAgainstLastKnown_StateChangedPatches(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{
				HostID:             hostID,
				RawResourceID:      "raw",
				DeviceUserResource: strPtr("devices/d/deviceUsers/u"),
				WorkspaceEmail:     "user@example.com",
				PartnerSuffix:      "fleet",
				LastCompliant:      boolPtr(true), // was compliant
				LastManaged:        boolPtr(true),
				LastScoreReason:    strPtr(""),
				LastEtag:           strPtr("etag-old"),
			},
		}, nil
	}
	var setCalls int
	ds.SetHostGoogleCloudIdentityClientStateFunc = func(ctx context.Context, hostID uint, rawResourceID, partnerSuffix string, managed, compliant bool, scoreReason, etag string) error {
		setCalls++
		assert.False(t, compliant, "now non-compliant")
		assert.Equal(t, "policy failed", scoreReason)
		return nil
	}

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		assert.Equal(t, http.MethodPatch, r.Method)
		var got ClientState
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, ComplianceStateNonCompliant, got.ComplianceState)
		assert.Equal(t, "policy failed", got.ScoreReason)
		// Etag from prior PATCH should round-trip on subsequent PATCH for
		// optimistic concurrency.
		assert.Equal(t, "etag-old", got.Etag)
		_ = json.NewEncoder(w).Encode(ClientState{Etag: "etag-new"})
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, false, "policy failed"))

	assert.Equal(t, 1, setCalls)
	assert.Len(t, h.recordedRequests(), 1, "single PATCH, no lookup (already resolved)")
}

func TestSyncHost_LookupReturnsNoNames_SkipsPatch(t *testing.T) {
	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{HostID: hostID, RawResourceID: "stale-raw", PartnerSuffix: "fleet"},
		}, nil
	}
	// SetHostGoogleCloudIdentityResolvedDeviceUserFunc/SetHostGoogleCloudIdentityClientStateFunc
	// intentionally NOT set; if they get called the mock store will panic.

	s, h := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		// Lookup returns empty.
		_ = json.NewEncoder(w).Encode(DeviceUserLookupResponse{Names: nil})
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))

	reqs := h.recordedRequests()
	require.Len(t, reqs, 1, "only the lookup happens")
	assert.Equal(t, http.MethodGet, reqs[0].Method)
}

func TestSyncHost_PerRowFailureDoesNotDropOthers(t *testing.T) {
	const goodRaw = "good-raw"
	const badRaw = "bad-raw"

	ds := new(mock.Store)
	ds.LoadHostGoogleCloudIdentityClientStatesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostGoogleCloudIdentityClientState, error) {
		return []*fleet.HostGoogleCloudIdentityClientState{
			{HostID: hostID, RawResourceID: badRaw, PartnerSuffix: "fleet"},
			{HostID: hostID, RawResourceID: goodRaw, PartnerSuffix: "fleet"},
		}, nil
	}
	ds.SetHostGoogleCloudIdentityResolvedDeviceUserFunc = func(ctx context.Context, hostID uint, rawResourceID, partnerSuffix, deviceUserResource string) error {
		return nil
	}
	var goodSet int
	ds.SetHostGoogleCloudIdentityClientStateFunc = func(ctx context.Context, hostID uint, rawResourceID, partnerSuffix string, managed, compliant bool, scoreReason, etag string) error {
		if rawResourceID == goodRaw {
			goodSet++
		}
		return nil
	}

	s, _ := newRecordingSyncer(t, ds, func(w http.ResponseWriter, r *http.Request, body []byte) {
		if r.URL.Query().Get("rawResourceId") == badRaw {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(DeviceUserLookupResponse{Names: []string{"devices/d/deviceUsers/u"}})
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(ClientState{Etag: "e"})
		}
	})

	host := &fleet.Host{ID: 1, UUID: "host-uuid"}
	require.NoError(t, s.SyncHost(context.Background(), host, true, true, ""))
	assert.Equal(t, 1, goodSet, "good row patched even though bad row failed")
}

// Sanity: silence the unused-os-import warning when other tests don't pull os.
var _ = os.Getenv
