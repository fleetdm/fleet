// OPENFRAME(mysql-multitenancy): tests for the shared-mode per-request tenant pin.
// The shared-mode gates (WithOpenframeTenant / openframePinHostTeam) read a cached env
// decision, so the tests exercise the mode-independent bodies (openframeTenantHandler,
// openframePinHostTeamShared) directly — same pattern as the pure-function tests in
// server/fleet/openframe_test.go.
package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTeamEnsurer struct {
	teamID uint
	err    error
	calls  int
}

func (f *fakeTeamEnsurer) EnsureOpenframeTeamID(_ context.Context, _ string) (uint, error) {
	f.calls++
	return f.teamID, f.err
}

func testTenantHandler(t *testing.T, ensurer *fakeTeamEnsurer) http.Handler {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return openframeTenantHandler(ensurer, noOpenframeSuperuser, noOpenframeSuperuserEmail, slog.New(slog.DiscardHandler), next)
}

func TestOpenframeTenantHandlerPinsFromHeader(t *testing.T) {
	ensurer := &fakeTeamEnsurer{teamID: 42}
	var gotTeam uint
	var gotOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTeam, gotOK = fleet.OpenframeTeamID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := openframeTenantHandler(ensurer, noOpenframeSuperuser, noOpenframeSuperuserEmail, slog.New(slog.DiscardHandler), next)

	const tenantUUID = "3f1a9b2c-0000-4d5e-8f00-000000000001"
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		req.Header.Set("X-Tenant-Id", tenantUUID)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		require.True(t, gotOK, "request ctx must carry the tenant team pin")
		require.Equal(t, uint(42), gotTeam)
	}
	// uuid → team id never changes: resolved once, then served from the cache.
	assert.Equal(t, 1, ensurer.calls)
}

func TestOpenframeTenantHandlerFailClosed(t *testing.T) {
	t.Run("missing header on a control-plane path is rejected", func(t *testing.T) {
		ensurer := &fakeTeamEnsurer{teamID: 42}
		h := testTenantHandler(t, ensurer)
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Zero(t, ensurer.calls)
	})

	t.Run("malformed tenant uuid is rejected and mints no team", func(t *testing.T) {
		ensurer := &fakeTeamEnsurer{teamID: 42}
		h := testTenantHandler(t, ensurer)
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		req.Header.Set("X-Tenant-Id", "not-a-uuid")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Zero(t, ensurer.calls, "a malformed uuid must never reach EnsureOpenframeTeamID")
	})

	t.Run("resolver error does not pass the request through", func(t *testing.T) {
		ensurer := &fakeTeamEnsurer{err: context.DeadlineExceeded}
		reached := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })
		h := openframeTenantHandler(ensurer, noOpenframeSuperuser, noOpenframeSuperuserEmail, slog.New(slog.DiscardHandler), next)
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		req.Header.Set("X-Tenant-Id", "3f1a9b2c-0000-4d5e-8f00-000000000001")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.NotEqual(t, http.StatusOK, rr.Code)
		assert.False(t, reached, "a request whose tenant cannot be resolved must not run")
	})
}

func TestOpenframeTenantHandlerAgentPathsExempt(t *testing.T) {
	ensurer := &fakeTeamEnsurer{teamID: 42}
	var pinnedOK bool
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		_, pinnedOK = fleet.OpenframeTeamID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	h := openframeTenantHandler(ensurer, noOpenframeSuperuser, noOpenframeSuperuserEmail, slog.New(slog.DiscardHandler), next)

	// No header, agent-plane path → passes through unpinned; the tenant is derived later
	// from the authenticated host / enroll secret.
	req := httptest.NewRequest("POST", "/api/v1/osquery/config", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.True(t, reached)
	assert.False(t, pinnedOK)
	assert.Zero(t, ensurer.calls)
}

func TestOpenframeTenantExemptPath(t *testing.T) {
	exempt := []string{
		"/api/v1/osquery/enroll",
		"/api/osquery/log",
		"/api/osquery/carve/block",
		"/api/fleet/orbit/enroll",
		"/api/fleet/orbit/ping",
		"/api/latest/fleet/orbit/config",
		"/api/latest/fleet/device/token123/desktop",
		"/api/fleet/device/ping",
		"/api/mdm/apple/enroll",
		"/api/mdm/apple/installer",
		"/api/mdm/apple/account_driven_enroll",
		"/api/mdm/microsoft/discovery",
		"/api/latest/fleet/ota_enrollment",
	}
	for _, p := range exempt {
		assert.True(t, openframeTenantExemptPath(p), "expected exempt: %s", p)
	}

	notExempt := []string{
		"/api/latest/fleet/hosts",
		"/api/latest/fleet/hosts/1",
		"/api/latest/fleet/config",
		"/api/latest/fleet/software",
		"/api/latest/fleet/login",
		"/api/latest/fleet/queries",
		// User-authenticated admin MDM APIs — must still require X-Tenant-Id in shared mode
		// (these are under /api/{v}/fleet/mdm/, not /api/mdm/).
		"/api/latest/fleet/mdm/apple/commands",
		"/api/latest/fleet/mdm/apple/enrollment_profile",
		"/api/v1/fleet/mdm/commands",
		// Device-facing DEP paths that share their path with admin variants (method-only difference);
		// not exempt by path alone. Acceptable — OpenFrame does not use Apple/Windows MDM.
		"/api/latest/fleet/mdm/bootstrap",
		"/api/latest/fleet/mdm/setup/eula/token123",
	}
	for _, p := range notExempt {
		assert.False(t, openframeTenantExemptPath(p), "expected NOT exempt: %s", p)
	}
}

func TestOpenframePinHostTeamShared(t *testing.T) {
	ctx := context.Background()

	t.Run("host with a team pins the ctx", func(t *testing.T) {
		teamID := uint(7)
		got, err := openframePinHostTeamShared(ctx, &fleet.Host{TeamID: &teamID})
		require.NoError(t, err)
		id, ok := fleet.OpenframeTeamID(got)
		require.True(t, ok)
		assert.Equal(t, uint(7), id)
	})

	t.Run("host without a team fails auth", func(t *testing.T) {
		_, err := openframePinHostTeamShared(ctx, &fleet.Host{})
		require.Error(t, err)
		var authFailed *fleet.AuthFailedError
		assert.ErrorAs(t, err, &authFailed)
	})

	t.Run("nil host fails auth", func(t *testing.T) {
		_, err := openframePinHostTeamShared(ctx, nil)
		require.Error(t, err)
	})
}

// --- superuser escape hatch -------------------------------------------------

// noOpenframeSuperuser is the resolver the pre-existing tests run with: no caller is ever
// recognised, so they keep asserting the fail-closed behavior they were written for.
func noOpenframeSuperuser(_ context.Context, _ string) (*viewer.Viewer, error) {
	return nil, errors.New("no session")
}

// noOpenframeSuperuserEmail is the allowlist the pre-existing tests run with: empty, so nobody is
// recognised and they keep asserting the fail-closed behavior they were written for.
func noOpenframeSuperuserEmail(_ string) bool { return false }

func allowOnly(allowed string) openframeSuperuserPredicate {
	return func(email string) bool { return email == allowed }
}

func viewerFor(email string) *viewer.Viewer {
	return &viewer.Viewer{
		User:    &fleet.User{ID: 7, Email: email},
		Session: &fleet.Session{ID: 1},
	}
}

// resolverFor recognises exactly one session key, the way a live Fleet session would.
func resolverFor(sessionKey, email string) openframeViewerResolver {
	return func(_ context.Context, key string) (*viewer.Viewer, error) {
		if key != sessionKey {
			return nil, errors.New("no session")
		}
		return viewerFor(email), nil
	}
}

func superuserHandler(t *testing.T, resolve openframeViewerResolver, isSuperuser openframeSuperuserPredicate) http.Handler {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pinned := fleet.OpenframeTeamID(r.Context()); pinned {
			t.Error("superuser request must reach the handler unpinned")
		}
		w.WriteHeader(http.StatusOK)
	})
	return openframeTenantHandler(&fakeTeamEnsurer{teamID: 42}, resolve, isSuperuser, slog.New(slog.DiscardHandler), next)
}

func TestOpenframeSuperuserServedUnpinned(t *testing.T) {
	const (
		sessionKey = "a-live-session-key"
		email      = "research@flamingo.cx"
	)
	t.Run("bearer token", func(t *testing.T) {
		h := superuserHandler(t, resolverFor(sessionKey, email), allowOnly(email))
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+sessionKey)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	// The live-query result stream is a WebSocket upgrade: a browser attaches cookies to it but
	// cannot attach an Authorization header, so the cookie is the only identity it can carry.
	for _, name := range []string{"token", "__Host-token"} {
		t.Run("session cookie "+name, func(t *testing.T) {
			h := superuserHandler(t, resolverFor(sessionKey, email), allowOnly(email))
			req := httptest.NewRequest("GET", "/api/v1/fleet/results/1/abc/websocket", nil)
			req.AddCookie(&http.Cookie{Name: name, Value: sessionKey})
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestOpenframeSuperuserStaysFailClosed(t *testing.T) {
	const (
		sessionKey = "a-live-session-key"
		email      = "research@flamingo.cx"
	)

	t.Run("a session that is not on the allowlist is still rejected", func(t *testing.T) {
		ensurer := &fakeTeamEnsurer{teamID: 42}
		h := openframeTenantHandler(ensurer, resolverFor(sessionKey, "someone.else@flamingo.cx"),
			allowOnly(email), slog.New(slog.DiscardHandler),
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+sessionKey)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("an unknown session key is rejected even for an allowlisted address", func(t *testing.T) {
		h := superuserHandler(t, resolverFor(sessionKey, email), allowOnly(email))
		req := httptest.NewRequest("GET", "/api/latest/fleet/hosts", nil)
		req.Header.Set("Authorization", "Bearer not-the-session-key")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestOpenframeSessionBootstrapPathsNeedNoTenant(t *testing.T) {
	// Without these a shared-mode Fleet cannot be logged into at all: the login page reads the SSO
	// settings and posts credentials before anyone holds a token.
	bootstrap := []string{
		"/api/v1/fleet/login",
		"/api/latest/fleet/logout",
		"/api/v1/fleet/sessions",
		"/api/v1/fleet/sso",
		"/api/v1/fleet/sso/callback",
	}
	for _, path := range bootstrap {
		assert.True(t, openframeSessionBootstrapPath.MatchString(path), path)
	}

	// Anchored: the admin endpoint that merely ends in /sessions must keep requiring a tenant.
	pinned := []string{
		"/api/v1/fleet/users/3/sessions",
		"/api/latest/fleet/hosts",
		"/api/v1/fleet/login/extra",
	}
	for _, path := range pinned {
		assert.False(t, openframeSessionBootstrapPath.MatchString(path), path)
	}
}
