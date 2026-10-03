// OPENFRAME(mysql-multitenancy): tests for the live-query websocket tenant re-pin — the sockjs
// handler rebuilds its context from Background, so it must copy the team pin from the upgrade
// request's context (set by the tenant middleware) onto the stream context. The fail-closed
// shared-mode guard reads the cached mode env and is not exercisable here (same limitation as
// openframe_middleware_test.go); these tests cover the re-pin and the unpinned pass-through.
package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	ws "github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type observedPin struct {
	teamID uint
	ok     bool
}

// streamCampaignAndObservePin drives the websocket handler (optionally wrapped in the tenant
// middleware body, as WithOpenframeTenant wraps it in shared mode) through auth + select_campaign
// and returns the team pin observed inside the campaign datastore lookup.
func streamCampaignAndObservePin(t *testing.T, withTenantMiddleware bool) observedPin {
	t.Helper()

	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)

	// Auth plumbing for AuthViewer (token sent as the first websocket message).
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              1,
			AccessedAt:      time.Now(),
			UserID:          1,
			Key:             key,
		}, nil
	}
	ds.MarkSessionAccessedFunc = func(context.Context, *fleet.Session) error { return nil }
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id, GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}

	// Capture the pin at the first datastore touch of the stream, then short-circuit the
	// handler with a non-ErrNoRows error (ErrNoRows would make it poll for 5s).
	pinCh := make(chan observedPin, 1)
	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		teamID, ok := fleet.OpenframeTeamID(ctx)
		select {
		case pinCh <- observedPin{teamID: teamID, ok: ok}:
		default:
		}
		return nil, errors.New("pin captured, stop the stream")
	}

	pathHandler := makeStreamDistributedQueryCampaignResultsHandler(
		config.TestConfig().Server, svc, slog.New(slog.DiscardHandler))
	handler := pathHandler("/api/{fleetversion:(?:latest)}/fleet/results/")
	if withTenantMiddleware {
		handler = openframeTenantHandler(&fakeTeamEnsurer{teamID: 42}, noOpenframeSuperuser, noOpenframeSuperuserEmail, slog.New(slog.DiscardHandler), handler)
	}
	s := httptest.NewServer(handler)
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/api/latest/fleet/results/websocket"
	dialer := &websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	var header http.Header
	if withTenantMiddleware {
		header = http.Header{"X-Tenant-Id": []string{"3f1a9b2c-0000-4d5e-8f00-000000000001"}}
	}
	conn, _, err := dialer.Dial(u, header)
	require.NoError(t, err)
	defer conn.Close()

	require.NoError(t, conn.WriteJSON(ws.JSONMessage{
		Type: "auth",
		Data: map[string]interface{}{"token": "test-token"},
	}))
	require.NoError(t, conn.WriteJSON(ws.JSONMessage{
		Type: "select_campaign",
		Data: map[string]interface{}{"campaign_id": 1},
	}))

	select {
	case pin := <-pinCh:
		return pin
	case <-time.After(10 * time.Second):
		t.Fatal("the campaign datastore lookup was never reached")
		return observedPin{}
	}
}

// TestOpenframeCampaignStreamRePinsFromUpgradeRequest verifies that the tenant pin placed on the
// websocket upgrade request by the tenant middleware survives into the campaign stream's context
// (and from there into the datastore fences and the live_query activity team stamping).
func TestOpenframeCampaignStreamRePinsFromUpgradeRequest(t *testing.T) {
	pin := streamCampaignAndObservePin(t, true)
	require.True(t, pin.ok, "stream ctx must carry the tenant pin from the upgrade request")
	require.Equal(t, uint(42), pin.teamID)
}

// TestOpenframeCampaignStreamUnpinnedPassThrough verifies upstream behavior is preserved when no
// pin is present on the upgrade request (flag off / single-tenant): no pin is invented.
func TestOpenframeCampaignStreamUnpinnedPassThrough(t *testing.T) {
	pin := streamCampaignAndObservePin(t, false)
	require.False(t, pin.ok, "an unpinned upgrade request must leave the stream ctx unpinned")
}
