package main

import (
	"crypto/tls"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
)

// Tunables mirror orbit's wstransport manager defaults (see
// orbit/pkg/wstransport.Options.applyDefaults), so load tests exercise the
// same connect/reconnect behavior as real fleetd.
const (
	wsNotificationsPath  = "/api/fleet/orbit/notifications"
	wsReconnectJitterMax = 30 * time.Second
	wsBackoffBase        = 5 * time.Second
	wsBackoffCap         = 5 * time.Minute
	wsServerPingInterval = 5 * time.Minute
	wsHandshakeTimeout   = 30 * time.Second
)

// wsTransport simulates orbit's WebSocket notification transport
// (orbit/pkg/wstransport.Manager): one distributed read+write cycle per
// "check now" notification, coalescing triggers arriving mid-cycle into a
// single queued follow-up. The simulator runs the whole iteration inline (no
// separate osquery process), so orbit's three iteration states collapse into
// one busy flag with the same external behavior: at most one cycle in flight,
// at most one queued.
type wsTransport struct {
	agent *agent

	// connected is true while the WebSocket is up; the agent's distributed
	// poll ticks are skipped while it is set (see agent.wsPollTick).
	connected atomic.Bool

	// mu guards the iteration state: busy while a read+write cycle runs,
	// pending when a trigger arrived mid-cycle.
	mu      sync.Mutex
	busy    bool
	pending bool

	done     chan struct{}
	stopOnce sync.Once
}

func newWSTransport(a *agent) *wsTransport {
	return &wsTransport{agent: a, done: make(chan struct{})}
}

func (w *wsTransport) stop() {
	w.stopOnce.Do(func() { close(w.done) })
}

func (w *wsTransport) stopped() bool {
	select {
	case <-w.done:
		return true
	default:
		return false
	}
}

// run dials, services and re-dials the WebSocket until stopped, mirroring
// orbit's connection loop: backoff on failed dials, jitter before re-dials so
// a server restart doesn't produce a thundering herd.
func (w *wsTransport) run() {
	backoff := wsBackoffBase
	for {
		if w.stopped() {
			return
		}
		conn, err := w.dial()
		if err != nil {
			w.agent.stats.IncrementWebSocketErrors()
			if !w.sleep(backoff) {
				return
			}
			backoff = min(backoff*2, wsBackoffCap)
			continue
		}
		backoff = wsBackoffBase
		w.agent.stats.IncrementWebSocketConnects()
		w.agent.stats.UpdateWebSocketConnected(1)
		w.connected.Store(true)
		w.readLoop(conn)
		w.connected.Store(false)
		w.agent.stats.UpdateWebSocketConnected(-1)
		_ = conn.Close()
		if w.stopped() {
			return
		}
		if !w.sleep(time.Duration(rand.Int64N(int64(wsReconnectJitterMax)))) { //nolint:gosec // jitter does not need cryptographic randomness
			return
		}
	}
}

// readLoop consumes notifications until the connection breaks or the
// transport is stopped. As in orbit, the read deadline doubles as connection
// liveness: the server pings every wsServerPingInterval and every ping
// refreshes the deadline.
func (w *wsTransport) readLoop(conn *websocket.Conn) {
	// Unblock the blocking read when the transport is stopped.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-w.done:
			_ = conn.Close()
		case <-stop:
		}
	}()

	deadline := 2 * wsServerPingInterval
	_ = conn.SetReadDeadline(time.Now().Add(deadline))
	defaultPingHandler := conn.PingHandler()
	conn.SetPingHandler(func(message string) error {
		_ = conn.SetReadDeadline(time.Now().Add(deadline))
		return defaultPingHandler(message)
	})

	for {
		var msg fleet.AgentWSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			// A stopped transport closes the connection itself; the resulting
			// read error is not worth logging.
			if !w.stopped() {
				log.Printf("agent %d: websocket read failed: %s", w.agent.agentIndex, err)
			}
			return
		}
		// Unknown notification types are ignored for forward compatibility,
		// like orbit.
		if msg.Type == fleet.AgentWSMessageTypeDistributedRead {
			w.agent.stats.IncrementWebSocketNotifications()
			w.trigger()
		}
	}
}

// trigger starts one distributed read+write cycle, or queues one if a cycle
// is already in flight. It is the single entry point for notifications and
// poll ticks, coalescing them under the same rule as orbit's Manager.trigger:
// at most one cycle in flight, at most one follow-up queued.
func (w *wsTransport) trigger() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.busy {
		w.pending = true
		return
	}
	w.busy = true
	go w.runCycle()
}

// runCycle performs distributed read+write cycles until none is queued. A
// failed read drops the queued trigger, like orbit: retrying immediately
// would tight-loop against a failing server, and the server's per-tick
// re-notify (or the polling fallback) re-triggers while work remains due.
func (w *wsTransport) runCycle() {
	for {
		resp, err := w.agent.DistributedRead()
		if err == nil && len(resp.Queries) > 0 {
			_ = w.agent.DistributedWrite(resp.Queries)
		}
		w.mu.Lock()
		fire := w.pending && err == nil && !w.stopped()
		w.pending = false
		if !fire {
			w.busy = false
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
	}
}

// dial performs the authenticated WebSocket upgrade with the orbit node key,
// like orbit.
func (w *wsTransport) dial() (*websocket.Conn, error) {
	wsURL, err := url.Parse(w.agent.serverAddress)
	if err != nil {
		return nil, err
	}
	switch wsURL.Scheme {
	case "https":
		wsURL.Scheme = "wss"
	case "http":
		wsURL.Scheme = "ws"
	}
	wsURL.Path = wsNotificationsPath

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: wsHandshakeTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // nolint:gosec // load-test tool, matches the simulator's HTTP client
		},
	}
	header := http.Header{}
	header.Set("Authorization", "Node key "+*w.agent.orbitNodeKey)

	conn, resp, err := dialer.Dial(wsURL.String(), header)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	return conn, err
}

// sleep waits for d or until stopped; it reports false when stopped.
func (w *wsTransport) sleep(d time.Duration) bool {
	if d <= 0 {
		return !w.stopped()
	}
	select {
	case <-time.After(d):
		return true
	case <-w.done:
		return false
	}
}

// syncWSTransport starts or stops the simulated WebSocket transport to match
// the server's orbit config directive. Real fleetd restarts itself when the
// directive flips (see orbit/pkg/wstransport.ToggleReceiver); the simulator
// just starts or stops the loop.
func (a *agent) syncWSTransport(directive *fleet.OrbitWebSocketTransportConfig) {
	enabled := a.wsSupported && directive != nil && directive.Enabled
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	switch {
	case enabled && a.ws == nil:
		a.ws = newWSTransport(a)
		go a.ws.run()
	case !enabled && a.ws != nil:
		a.ws.stop()
		a.ws = nil
	}
}

// wsPollTick reports whether the WebSocket transport owns this distributed
// poll tick. While connected the tick is skipped (notifications drive the
// reads); while disconnected it routes through the coalescing trigger,
// mirroring orbit's polling fallback.
func (a *agent) wsPollTick() bool {
	a.wsMu.Lock()
	ws := a.ws
	a.wsMu.Unlock()
	if ws == nil {
		return false
	}
	if !ws.connected.Load() {
		ws.trigger()
	}
	return true
}

// wsTransportActive reports whether the WebSocket transport is running.
func (a *agent) wsTransportActive() bool {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	return a.ws != nil
}

// distributedNodeKey is the node key sent on distributed read/write calls.
// With the WebSocket transport active the calls go through orbit in real
// fleetd (osquery's tls distributed plugin is replaced by orbit's), so they
// authenticate with the orbit node key via the server's fallback lookup.
func (a *agent) distributedNodeKey() string {
	if a.wsTransportActive() && a.orbitNodeKey != nil {
		return *a.orbitNodeKey
	}
	return a.nodeKey
}

// distributedAPIPrefix is the URL prefix for distributed read/write calls.
// osquery's tls distributed plugin is configured with the versioned
// /api/v1/osquery endpoints, while orbit's WebSocket transport calls the
// unversioned /api/osquery aliases.
func (a *agent) distributedAPIPrefix() string {
	if a.wsTransportActive() {
		return "/api/osquery"
	}
	return "/api/v1/osquery"
}
