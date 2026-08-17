package wstransport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/client"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingClient struct {
	mu    sync.Mutex
	reads int
	resp  *client.DistributedReadResponse
	err   error
}

func (c *countingClient) DistributedRead() (*client.DistributedReadResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	if c.err != nil {
		return nil, c.err
	}
	if c.resp != nil {
		return c.resp, nil
	}
	return &client.DistributedReadResponse{}, nil
}

func (c *countingClient) DistributedWrite(*client.DistributedWriteRequest) error { return nil }

func (c *countingClient) readCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

// wsTestServer upgrades authenticated requests and tracks connections.
type wsTestServer struct {
	srv      *httptest.Server
	mu       sync.Mutex
	conns    []*websocket.Conn
	rejected atomic.Int64
	// reject makes the server refuse upgrades (simulates a down/unreachable
	// WebSocket endpoint while HTTP keeps working).
	reject atomic.Bool
}

func newWSTestServer(t *testing.T, nodeKey string) *wsTestServer {
	t.Helper()
	ws := &wsTestServer{}
	upgrader := websocket.Upgrader{}
	ws.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ws.reject.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Header.Get("Authorization") != "Node key "+nodeKey {
			ws.rejected.Add(1)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ws.mu.Lock()
		ws.conns = append(ws.conns, conn)
		ws.mu.Unlock()
		// Consume control frames so pings/pongs and close work.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	}))
	t.Cleanup(ws.srv.Close)
	return ws
}

func (ws *wsTestServer) connCount() int {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return len(ws.conns)
}

func (ws *wsTestServer) notify(t *testing.T, msg fleet.AgentWSMessage) {
	t.Helper()
	ws.mu.Lock()
	defer ws.mu.Unlock()
	require.NotEmpty(t, ws.conns)
	require.NoError(t, ws.conns[len(ws.conns)-1].WriteJSON(msg))
}

func (ws *wsTestServer) closeAll() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for _, c := range ws.conns {
		_ = c.Close()
	}
	ws.conns = nil
}

func newTestManager(t *testing.T, serverURL string, fc distributedClient, cache *QueryCache) *Manager {
	t.Helper()
	u, err := url.Parse(serverURL)
	require.NoError(t, err)
	m := NewManager(Options{
		ServerURL:          u,
		NodeKeyFunc:        func() (string, error) { return "orbit-key", nil },
		Client:             fc,
		Cache:              cache,
		PollInterval:       30 * time.Millisecond,
		ReconnectJitterMax: 10 * time.Millisecond,
		BackoffBase:        10 * time.Millisecond,
		BackoffCap:         50 * time.Millisecond,
		ServerPingInterval: time.Minute,
		HandshakeTimeout:   time.Second,
	})
	done := make(chan struct{})
	go func() {
		_ = m.Execute()
		close(done)
	}()
	t.Cleanup(func() {
		m.Interrupt(nil)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("manager did not stop")
		}
	})
	return m
}

func TestManagerConnectsAndReadsOnNotification(t *testing.T) {
	ws := newWSTestServer(t, "orbit-key")
	fc := &countingClient{resp: &client.DistributedReadResponse{Queries: map[string]string{"q1": "SELECT 1"}}}
	cache := NewQueryCache()
	m := newTestManager(t, ws.srv.URL, fc, cache)

	require.Eventually(t, func() bool { return ws.connCount() == 1 }, 5*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool { return m.connected.Load() }, 5*time.Second, 5*time.Millisecond)

	// A notification triggers a read and the queries land in the cache.
	before := fc.readCount()
	ws.notify(t, fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeDistributedRead})
	require.Eventually(t, func() bool { return fc.readCount() > before }, 5*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool {
		queries, _, _ := cache.Take()
		return queries["q1"] == "SELECT 1"
	}, 5*time.Second, 5*time.Millisecond)
}

func TestManagerIgnoresUnknownNotificationTypes(t *testing.T) {
	ws := newWSTestServer(t, "orbit-key")
	fc := &countingClient{}
	newTestManager(t, ws.srv.URL, fc, NewQueryCache())

	require.Eventually(t, func() bool { return ws.connCount() == 1 }, 5*time.Second, 5*time.Millisecond)
	ws.notify(t, fleet.AgentWSMessage{Type: "future/channel"})

	// The connection survives unknown types: a known one still works after.
	ws.notify(t, fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeDistributedRead})
	require.Eventually(t, func() bool { return fc.readCount() >= 1 }, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, 1, ws.connCount())
}

func TestManagerConnectedStopsPolling(t *testing.T) {
	ws := newWSTestServer(t, "orbit-key")
	fc := &countingClient{}
	m := newTestManager(t, ws.srv.URL, fc, NewQueryCache())

	require.Eventually(t, func() bool { return m.connected.Load() }, 5*time.Second, 5*time.Millisecond)

	// While connected, polling is off: the read count stays flat. There is no
	// catch-up read either, so with a prompt connect this stays at zero.
	// (Small settle wait so an in-flight pre-connect poll can't race the
	// snapshot below.)
	time.Sleep(2 * m.opts.PollInterval)
	count := fc.readCount()
	time.Sleep(10 * m.opts.PollInterval)
	assert.Equal(t, count, fc.readCount())
}

func TestManagerFallsBackToPollingAndReconnects(t *testing.T) {
	ws := newWSTestServer(t, "orbit-key")
	fc := &countingClient{}
	m := newTestManager(t, ws.srv.URL, fc, NewQueryCache())

	require.Eventually(t, func() bool { return m.connected.Load() }, 5*time.Second, 5*time.Millisecond)

	// The WebSocket endpoint goes down and drops all connections: polling
	// resumes while reconnection attempts fail.
	ws.reject.Store(true)
	ws.closeAll()
	require.Eventually(t, func() bool { return !m.connected.Load() }, 5*time.Second, 5*time.Millisecond)

	count := fc.readCount()
	require.Eventually(t, func() bool { return fc.readCount() > count }, 5*time.Second, 5*time.Millisecond,
		"polling should resume while disconnected")

	// The endpoint comes back: the manager reconnects and polling stops again.
	ws.reject.Store(false)
	require.Eventually(t, func() bool { return m.connected.Load() }, 5*time.Second, 5*time.Millisecond,
		"manager should reconnect")
}

func TestManagerPollsWhenServerRejectsWebSocket(t *testing.T) {
	// Simulates the server flag being off (404 on the WS endpoint): the
	// manager keeps polling and keeps retrying the connection with backoff.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	fc := &countingClient{}
	newTestManager(t, srv.URL, fc, NewQueryCache())

	require.Eventually(t, func() bool { return fc.readCount() >= 2 }, 5*time.Second, 5*time.Millisecond,
		"polling must keep the host serviced when the websocket is unavailable")
}

func TestManagerKeepsRetryingWhenEverythingIsDown(t *testing.T) {
	// Both the WebSocket and HTTP paths fail: the manager neither exits nor
	// panics; it keeps trying both.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	fc := &countingClient{err: assert.AnError}
	m := newTestManager(t, srv.URL, fc, NewQueryCache())
	srv.Close()

	require.Eventually(t, func() bool { return fc.readCount() >= 2 }, 5*time.Second, 5*time.Millisecond)
	assert.False(t, m.connected.Load())
}
