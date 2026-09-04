package agentws

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuthenticator struct {
	hosts map[string]uint // node key -> host ID
}

func (f *fakeAuthenticator) AuthenticateOrbitHost(ctx context.Context, nodeKey string) (*fleet.Host, bool, error) {
	id, ok := f.hosts[nodeKey]
	if !ok {
		return nil, false, errors.New("invalid node key")
	}
	return &fleet.Host{ID: id, Hostname: "host-" + nodeKey, Platform: "darwin"}, false, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newTestServer(t *testing.T, hub *Hub) *httptest.Server {
	t.Helper()
	auth := &fakeAuthenticator{hosts: map[string]uint{"key-1": 1, "key-2": 2}}
	srv := httptest.NewServer(NewHandler(hub, auth, discardLogger()))
	t.Cleanup(srv.Close)
	return srv
}

func dial(t *testing.T, srv *httptest.Server, nodeKey string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	header := http.Header{}
	if nodeKey != "" {
		header.Set("Authorization", nodeKeyAuthScheme+nodeKey)
	}
	ws, resp, err := websocket.DefaultDialer.Dial(url, header)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	t.Cleanup(func() { ws.Close() })
	return ws
}

func waitForConnCount(t *testing.T, hub *Hub, want int) {
	t.Helper()
	require.Eventually(t, func() bool { return hub.connCount() == want },
		2*time.Second, 5*time.Millisecond, "expected %d connections", want)
}

func TestHandlerRejectsBadAuth(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)
	url := "ws" + strings.TrimPrefix(srv.URL, "http")

	for _, tc := range []struct {
		name   string
		header http.Header
	}{
		{"missing header", http.Header{}},
		{"wrong scheme", http.Header{"Authorization": []string{"Bearer key-1"}}},
		{"invalid key", http.Header{"Authorization": []string{nodeKeyAuthScheme + "bogus"}}},
		{"empty key", http.Header{"Authorization": []string{nodeKeyAuthScheme}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ws, resp, err := websocket.DefaultDialer.Dial(url, tc.header)
			require.Error(t, err)
			require.Nil(t, ws)
			require.NotNil(t, resp)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			assert.Equal(t, 0, hub.connCount())
		})
	}
}

func TestHandlerNotifyDelivery(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	ws := dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	// Host 1 is connected, hosts 2 and 3 are not.
	sent := hub.Notify(fleet.AgentWSMessageTypeDistributedRead, fleet.AgentWSReasonDetail, []uint{1, 2, 3})
	assert.Equal(t, 1, sent)

	require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
	var msg fleet.AgentWSMessage
	require.NoError(t, ws.ReadJSON(&msg))
	assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, msg.Type)
	assert.Equal(t, fleet.AgentWSReasonDetail, msg.Reason)

	// Byte counters: the delivered notification (and the 101 handshake
	// response) count as bytes out; a client data frame counts as bytes in.
	require.NoError(t, ws.WriteMessage(websocket.TextMessage, []byte("client data")))
	require.Eventually(t, func() bool {
		snap := hub.Snapshot()
		return len(snap) == 1 && snap[0].BytesOut > 0 && snap[0].BytesIn > 0
	}, 2*time.Second, 5*time.Millisecond)

	snap := hub.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, uint(1), snap[0].HostID)
	assert.Equal(t, "host-key-1", snap[0].Hostname)
	assert.Equal(t, "darwin", snap[0].Platform)
	assert.Equal(t, int64(1), snap[0].NotifiedCount)
	assert.Equal(t, int64(0), snap[0].DroppedCount)
	assert.NotNil(t, snap[0].LastNotifiedAt)
	assert.Equal(t, fleet.AgentWSReasonDetail, snap[0].LastNotifyReason)
	assert.False(t, snap[0].ConnectedAt.IsZero())
}

func TestHandlerEvictsPreviousConnection(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	first := dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	second := dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	// The first connection is closed by the server.
	require.NoError(t, first.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := first.ReadMessage()
	require.Error(t, err)

	// The second connection still receives notifications.
	require.Equal(t, 1, hub.Notify(fleet.AgentWSMessageTypeDistributedRead, fleet.AgentWSReasonDetail, []uint{1}))
	require.NoError(t, second.SetReadDeadline(time.Now().Add(2*time.Second)))
	var msg fleet.AgentWSMessage
	require.NoError(t, second.ReadJSON(&msg))
	assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, msg.Type)
}

func TestHandlerDisconnectUnregisters(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	ws := dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	ws.Close()
	waitForConnCount(t, hub, 0)
	assert.Equal(t, 0, hub.Notify(fleet.AgentWSMessageTypeDistributedRead, fleet.AgentWSReasonDetail, []uint{1}))
}

func TestHandlerClosesOnOversizedInboundMessage(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	ws := dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	// A message within the limit is discarded; the connection stays up.
	require.NoError(t, ws.WriteMessage(websocket.TextMessage, make([]byte, maxInboundMessageSize)))
	require.Equal(t, 1, hub.Notify(fleet.AgentWSMessageTypeDistributedRead, fleet.AgentWSReasonDetail, []uint{1}))
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
	var msg fleet.AgentWSMessage
	require.NoError(t, ws.ReadJSON(&msg))

	// An oversized message gets the connection torn down.
	require.NoError(t, ws.WriteMessage(websocket.TextMessage, make([]byte, maxInboundMessageSize+1)))
	waitForConnCount(t, hub, 0)
}

func TestHandlerKeepaliveClosesDeadPeer(t *testing.T) {
	// Millisecond-scale keepalive: the client never reads, so it never answers
	// pings, and the server's read deadline (pingInterval+pongTimeout) expires.
	hub := NewHub(discardLogger(), 20*time.Millisecond, 20*time.Millisecond)
	srv := newTestServer(t, hub)

	dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)
	waitForConnCount(t, hub, 0)
}

func TestHubReadStats(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)

	assert.Empty(t, hub.ReadStats())

	// Counting is independent of held connections: host 7 has no WebSocket.
	hub.RecordDistributedRead(7, true)
	hub.RecordDistributedRead(7, true)
	hub.RecordDistributedRead(7, false)
	hub.RecordDistributedRead(3, false)

	stats := hub.ReadStats()
	require.Len(t, stats, 2)
	assert.Equal(t, ReadStats{HostID: 3, OrbitReads: 1, LegacyReads: 0}, stats[0])
	assert.Equal(t, ReadStats{HostID: 7, OrbitReads: 1, LegacyReads: 2}, stats[1])
}

func TestHubShutdown(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	ws := dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	hub.Shutdown()
	assert.Equal(t, 0, hub.connCount())

	require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := ws.ReadMessage()
	require.Error(t, err)
}

func TestHubShutdownRejectsLateRegistrations(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	hub.Shutdown()

	// An upgrade racing shutdown still completes at the HTTP layer, but the
	// closed hub must close the connection instead of holding it forever.
	ws := dial(t, srv, "key-1")
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := ws.ReadMessage()
	require.Error(t, err)

	assert.Equal(t, 0, hub.connCount())
	assert.Equal(t, 0, hub.Notify(fleet.AgentWSMessageTypeDistributedRead, fleet.AgentWSReasonDetail, []uint{1}))
}
