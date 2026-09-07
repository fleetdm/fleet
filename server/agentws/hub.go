package agentws

import (
	"cmp"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
)

// Hub is the per-instance registry of open agent WebSocket connections, keyed
// by host ID. Each server instance holds only its own connections: wake-ups
// for hosts connected elsewhere are simply skipped here and handled by the
// instance that holds them.
type Hub struct {
	logger       *slog.Logger
	pingInterval time.Duration
	pongTimeout  time.Duration

	mu     sync.RWMutex
	conns  map[uint]*conn
	closed bool

	// reads counts distributed/read requests per host by path (see ReadStats),
	// reported by the debug endpoint alongside the connections.
	reads readStatsRegistry

	// nextCheckNano is the interval check job's next tick (unix nanos), for
	// the /debug/agentws "next sync" countdown; zero until the job first runs.
	nextCheckNano atomic.Int64

	// InstanceID identifies the Fleet server process owning this hub, so
	// /debug/agentws consumers behind a load balancer can tell instances apart.
	InstanceID string
}

// RecordNextCheck stores when the interval check job runs next.
func (h *Hub) RecordNextCheck(t time.Time) {
	h.nextCheckNano.Store(t.UnixNano())
}

// NextCheck returns when the interval check job runs next, or the zero time
// if it hasn't recorded a tick yet.
func (h *Hub) NextCheck() time.Time {
	nano := h.nextCheckNano.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

func NewHub(logger *slog.Logger, pingInterval, pongTimeout time.Duration) *Hub {
	return &Hub{
		logger:       logger,
		pingInterval: pingInterval,
		pongTimeout:  pongTimeout,
		conns:        make(map[uint]*conn),
	}
}

// ServeConn registers ws as the connection for hostID and services it until
// the peer disconnects or fails a keepalive check. It blocks (running the read
// loop) and is meant to be called from the upgrade handler's goroutine.
func (h *Hub) ServeConn(hostID uint, hostname, platform string, ws *websocket.Conn) {
	c := newConn(hostID, hostname, platform, ws)
	h.register(hostID, c)
	go c.writeLoop(h.pingInterval, h.pongTimeout)
	c.readLoop(h, h.pingInterval, h.pongTimeout)
}

// register stores c as the connection for hostID, evicting and closing any
// previous connection for the same host (e.g. an agent that reconnected
// before the server noticed the old connection died). An upgrade that races
// Shutdown registers after the hub drained its map; such late connections are
// closed instead of stored, since no other cleanup path would ever see them.
func (h *Hub) register(hostID uint, c *conn) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		c.close()
		return
	}
	old := h.conns[hostID]
	h.conns[hostID] = c
	h.mu.Unlock()

	if old != nil {
		old.close()
	}
}

// unregister removes c from the registry. It is a no-op if hostID has already
// been re-registered with a newer connection.
func (h *Hub) unregister(hostID uint, c *conn) {
	h.mu.Lock()
	if h.conns[hostID] == c {
		delete(h.conns, hostID)
	}
	h.mu.Unlock()
}

// Disconnect closes and removes the connections held for hostIDs, and returns
// how many were closed. Used when a host no longer exists: the agent
// reconnects (re-enrolled, under its new host ID) instead of lingering under
// a stale one.
func (h *Hub) Disconnect(hostIDs []uint) int {
	h.mu.Lock()
	conns := make([]*conn, 0, len(hostIDs))
	for _, id := range hostIDs {
		if c, ok := h.conns[id]; ok {
			delete(h.conns, id)
			conns = append(conns, c)
		}
	}
	h.mu.Unlock()

	for _, c := range conns {
		c.close()
	}
	return len(conns)
}

// Notify enqueues a notification of the given type and reason to each of
// hostIDs whose connection this instance holds, and returns how many were
// notified. Hosts connected to other instances (or not connected at all) are
// skipped: delivery is best-effort by design, with the interval check job and
// the agent's polling fallback as the safety nets.
func (h *Hub) Notify(msgType, reason string, hostIDs []uint) int {
	msg := fleet.AgentWSMessage{Type: msgType, Reason: reason}

	h.mu.RLock()
	defer h.mu.RUnlock()
	sent := 0
	for _, id := range hostIDs {
		if c, ok := h.conns[id]; ok {
			c.enqueue(msg)
			sent++
		}
	}
	return sent
}

// HeldHostIDs returns the host IDs of the connections this instance currently
// holds.
func (h *Hub) HeldHostIDs() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uint, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	return ids
}

// connCount returns the number of connections this instance holds.
func (h *Hub) connCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// ConnectionInfo describes one held connection, for observability (the
// /debug/agentws endpoint).
type ConnectionInfo struct {
	HostID         uint       `json:"host_id"`
	Hostname       string     `json:"hostname"`
	Platform       string     `json:"platform"`
	RemoteAddr     string     `json:"remote_addr"`
	ConnectedAt    time.Time  `json:"connected_at"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
	// LastNotifyReason is the reason of the last notification enqueued for
	// this connection (see the fleet.AgentWSReason constants).
	LastNotifyReason string `json:"last_notify_reason,omitempty"`
	NotifiedCount    int64  `json:"notified_count"`
	DroppedCount     int64  `json:"dropped_count"`
	// BytesIn/BytesOut are raw bytes on the underlying connection (post-TLS),
	// including WebSocket framing and ping/pong control frames.
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`
}

// Snapshot returns a point-in-time view of the connections this instance
// holds, sorted by host ID.
func (h *Hub) Snapshot() []ConnectionInfo {
	h.mu.RLock()
	infos := make([]ConnectionInfo, 0, len(h.conns))
	for _, c := range h.conns {
		info := ConnectionInfo{
			HostID:        c.hostID,
			Hostname:      c.hostname,
			Platform:      c.platform,
			RemoteAddr:    c.remoteAddr,
			ConnectedAt:   c.connectedAt,
			NotifiedCount: c.notified.Load(),
			DroppedCount:  c.dropped.Load(),
		}
		info.LastNotifyReason = c.lastNotifyReasonLoad()
		info.BytesIn, info.BytesOut = c.bytesInOut()
		if last := c.lastNotified(); !last.IsZero() {
			info.LastNotifiedAt = &last
		}
		infos = append(infos, info)
	}
	h.mu.RUnlock()

	slices.SortFunc(infos, func(a, b ConnectionInfo) int {
		return cmp.Compare(a.HostID, b.HostID)
	})
	return infos
}

// Shutdown closes all held connections and rejects registrations from then
// on. It is terminal: the hub cannot be reused afterwards.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	h.closed = true
	conns := make([]*conn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.conns = make(map[uint]*conn)
	h.mu.Unlock()

	for _, c := range conns {
		c.close()
	}
}
