package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/osqueryauth"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bodyTracker is a ReadCloser that records whether Read was ever called,
// so tests can assert that the request body is never consumed when the
// header pre-auth short-circuits.
type bodyTracker struct {
	io.Reader
	read int32
}

func (b *bodyTracker) Read(p []byte) (int, error) {
	atomic.StoreInt32(&b.read, 1)
	return b.Reader.Read(p)
}

func (b *bodyTracker) Close() error { return nil }

func (b *bodyTracker) wasRead() bool {
	return atomic.LoadInt32(&b.read) == 1
}

func TestExtractNodeKeyFromHeader(t *testing.T) {
	tests := []struct {
		authHeader string
		want       string
	}{
		{"", ""},
		{"NodeKey abc123", "abc123"},
		{"NodeKey  abc123  ", "abc123"},
		{"NodeKey ", ""},
		{"NodeKey", ""},
		{"Bearer abc123", ""},
		{"Node key abc123", ""},      // Orbit's scheme, must not match
		{"nodekey abc123", "abc123"}, // case-insensitive scheme per RFC 7235
		{"NODEKEY abc123", "abc123"},
		{"NoDeKeY abc123", "abc123"},
		{"NodeKeyabc123", ""},    // missing space
		{"NodeKey\tabc123", ""},  // tab is not a space
		{"NodeKey abc def", ""},  // embedded space rejected (defense against auth-params)
		{"NodeKey abc\tdef", ""}, // embedded tab rejected
		{"NodeKey " + strings.Repeat("A", 4096), strings.Repeat("A", 4096)}, // long token
	}
	for _, tt := range tests {
		t.Run(tt.authHeader, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/osquery/log", strings.NewReader(""))
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}
			got := extractNodeKeyFromHeader(r)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOsqueryHeaderPreAuth(t *testing.T) {
	const goodNodeKey = "valid-node-key"
	host := &fleet.Host{ID: 42, Hostname: "test-host", HasHostIdentityCert: new(false)}

	newSvc := func(t *testing.T) (fleet.Service, *mock.Store) {
		ds := new(mock.Store)
		svc, _ := newTestService(t, ds, nil, nil)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
			if nodeKey == goodNodeKey {
				return host, nil
			}
			return nil, newNotFoundError()
		}
		return svc, ds
	}

	type testCase struct {
		name               string
		authHeader         string
		wantNextCalled     bool
		wantPreAuthedInCtx bool
		wantStatus         int
		wantBodyRead       bool
	}

	// The pre-auth middleware is only installed when allow_body_auth_fallback
	// is false (handler.go gates the .WithHTTPPreAuth(...) call). In that
	// strict-mode all non-valid headers reject; valid ones populate ctx.
	cases := []testCase{
		{
			name:               "valid header",
			authHeader:         "NodeKey " + goodNodeKey,
			wantNextCalled:     true,
			wantPreAuthedInCtx: true,
			wantStatus:         http.StatusOK,
		},
		{
			name:           "invalid header",
			authHeader:     "NodeKey bogus",
			wantNextCalled: false,
			wantStatus:     http.StatusUnauthorized,
		},
		{
			name:           "absent header",
			authHeader:     "",
			wantNextCalled: false,
			wantStatus:     http.StatusUnauthorized,
		},
		{
			name:           "wrong scheme",
			authHeader:     "Bearer " + goodNodeKey,
			wantNextCalled: false,
			wantStatus:     http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newSvc(t)

			var nextCalled bool
			var ctxFromNext context.Context
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				ctxFromNext = r.Context()
				w.WriteHeader(http.StatusOK)
			})

			mw := osqueryHeaderPreAuth(svc, slog.New(slog.DiscardHandler))
			h := mw(next)

			tracker := &bodyTracker{Reader: strings.NewReader(`{"node_key":"some-body-content"}`)}
			req := httptest.NewRequest(http.MethodPost, "/api/osquery/log", nil)
			req.Body = tracker
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantNextCalled, nextCalled, "next-called")
			assert.Equal(t, tc.wantStatus, rec.Code, "status")
			assert.Equal(t, tc.wantBodyRead, tracker.wasRead(), "body-read")

			if tc.wantPreAuthedInCtx {
				require.NotNil(t, ctxFromNext)
				assert.True(t, osqueryauth.IsPreAuthed(ctxFromNext))
				gotHost, ok := hostctx.FromContext(ctxFromNext)
				assert.True(t, ok, "host should be in ctx")
				if ok {
					assert.Equal(t, host.ID, gotHost.ID)
				}
			}
		})
	}
}

// TestOsqueryHeaderPreAuthHostIdentityCert verifies that when the host has
// HasHostIdentityCert=true but the request lacks a valid HTTP message
// signature (httpsig.FromContext returns false), the pre-auth rejects with
// 401. This guards against a future refactor that accidentally bypasses
// VerifyHostIdentity on the header-auth path.
func TestOsqueryHeaderPreAuthHostIdentityCert(t *testing.T) {
	const goodNodeKey = "valid-node-key"
	// Host has an identity cert, so AuthenticateHost MUST verify the
	// httpsig — and without a cert in ctx that verification fails.
	host := &fleet.Host{
		ID:                  42,
		Hostname:            "tpm-host",
		HasHostIdentityCert: new(true),
		OsqueryHostID:       new("tpm-host-uuid"),
	}

	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
	ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		if nodeKey == goodNodeKey {
			return host, nil
		}
		return nil, newNotFoundError()
	}

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := osqueryHeaderPreAuth(svc, slog.New(slog.DiscardHandler))
	h := mw(next)

	req := httptest.NewRequest(http.MethodPost, "/api/osquery/log", strings.NewReader("{}"))
	req.Header.Set("Authorization", "NodeKey "+goodNodeKey)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "downstream handler must not run when httpsig verification fails")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	// Pre-auth normalizes every authentication failure (missing header,
	// wrong scheme, invalid token, httpsig failure, etc.) into a uniform
	// node_invalid:true response so the underlying reason is not exposed
	// to clients.
	assert.Contains(t, rec.Body.String(), `"node_invalid": true`)
}

// TestOsqueryCarveBlockHeaderPreAuth covers the strict-mode behavior of
// /api/osquery/carve/block: the middleware is only registered when
// allow_body_auth_fallback is false, in which case all non-valid headers
// reject and valid headers populate hostctx so CarveBlock can enforce the
// ownership check.
func TestOsqueryCarveBlockHeaderPreAuth(t *testing.T) {
	const goodNodeKey = "valid-node-key"
	host := &fleet.Host{ID: 42, Hostname: "test-host", HasHostIdentityCert: new(false)}

	newSvc := func(t *testing.T) fleet.Service {
		ds := new(mock.Store)
		svc, _ := newTestService(t, ds, nil, nil)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
			if nodeKey == goodNodeKey {
				return host, nil
			}
			return nil, newNotFoundError()
		}
		return svc
	}

	cases := []struct {
		name           string
		authHeader     string
		wantNextCalled bool
		wantStatus     int
		wantBodyRead   bool
		wantHostInCtx  bool
	}{
		{"absent header rejects", "", false, http.StatusUnauthorized, false, false},
		{"wrong scheme rejects", "Bearer " + goodNodeKey, false, http.StatusUnauthorized, false, false},
		{"malformed header rejects", "NodeKey", false, http.StatusUnauthorized, false, false},
		{"valid header passes through with host in ctx", "NodeKey " + goodNodeKey, true, http.StatusOK, false, true},
		{"invalid token short-circuits", "NodeKey bogus", false, http.StatusUnauthorized, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(t)

			var nextCalled bool
			var ctxFromNext context.Context
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				ctxFromNext = r.Context()
				w.WriteHeader(http.StatusOK)
			})

			h := osqueryCarveBlockHeaderPreAuth(svc, slog.New(slog.DiscardHandler))(next)

			tracker := &bodyTracker{Reader: strings.NewReader(`{"session_id":"s","request_id":"r","data":""}`)}
			req := httptest.NewRequest(http.MethodPost, "/api/osquery/carve/block", nil)
			req.Body = tracker
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantNextCalled, nextCalled, "next-called")
			assert.Equal(t, tc.wantStatus, rec.Code, "status")
			assert.Equal(t, tc.wantBodyRead, tracker.wasRead(), "body-read")

			if tc.wantNextCalled {
				require.NotNil(t, ctxFromNext)
				// Pre-authed node key ctx marker is never set on the
				// carve/block path — we don't want authenticatedHost (which
				// runs on other routes) to passthrough if wiring ever changes.
				ok := osqueryauth.IsPreAuthed(ctxFromNext)
				assert.False(t, ok, "pre-authed node key must not be set on carve/block path")

				gotHost, hostOk := hostctx.FromContext(ctxFromNext)
				assert.Equal(t, tc.wantHostInCtx, hostOk, "host in ctx")
				if hostOk {
					assert.Equal(t, host.ID, gotHost.ID)
				}
			}
		})
	}
}

// TestAuthenticatedHostPreAuthedPassthrough verifies that when the HTTP
// pre-auth middleware has set the pre-auth marker AND the host in ctx, the
// endpoint-layer authenticatedHost middleware skips body-based auth entirely
// and does not call svc.AuthenticateHost again.
func TestAuthenticatedHostPreAuthedPassthrough(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	var loadCalled int32
	ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		atomic.AddInt32(&loadCalled, 1)
		return nil, errors.New("should not be called on pre-authed path")
	}

	var nextCalled bool
	endpoint := authenticatedHost(
		svc,
		slog.New(slog.DiscardHandler),
		func(ctx context.Context, request any) (any, error) {
			nextCalled = true
			return nil, nil
		},
	)

	preCtx := hostctx.NewContext(ctx, &fleet.Host{ID: 7})
	preCtx = osqueryauth.NewPreAuthedContext(preCtx)
	_, err := endpoint(preCtx, &testNodeKeyRequest{NodeKey: ""}) // empty body key is fine
	require.NoError(t, err)
	assert.True(t, nextCalled, "next should be called")
	assert.Equal(t, int32(0), atomic.LoadInt32(&loadCalled), "LoadHostByNodeKey must not be called when pre-authed")
}

// TestAuthenticatedHostPreAuthedWithoutHostFails verifies the invariant that
// the pre-auth marker must always be set together with a host in ctx. If a
// future bug stamps the marker without populating hostctx, the endpoint-layer
// passthrough fails loudly instead of degrading silently.
func TestAuthenticatedHostPreAuthedWithoutHostFails(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }

	var nextCalled bool
	endpoint := authenticatedHost(
		svc,
		slog.New(slog.DiscardHandler),
		func(ctx context.Context, request any) (any, error) {
			nextCalled = true
			return nil, nil
		},
	)

	// Marker without host — programmer error.
	preCtx := osqueryauth.NewPreAuthedContext(ctx)
	_, err := endpoint(preCtx, &testNodeKeyRequest{NodeKey: ""})
	require.Error(t, err)
	assert.False(t, nextCalled, "next must not be called when invariant violated")
	assert.Contains(t, err.Error(), "pre-auth marker")
}
