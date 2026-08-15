package agentws

import (
	"cmp"
	"log/slog"
	"slices"
	"sync"
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

	mu    sync.RWMutex
	conns map[uint]*conn

	// reads counts distributed/read requests per host, split by path (see
	// ReadStats). Kept on the hub so the debug endpoint can report them
	// alongside the connections.
	reads readStatsRegistry
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
// platform is the host's platform (e.g. "darwin"), carried for observability.
func (h *Hub) ServeConn(hostID uint, platform string, ws *websocket.Conn) {
	c := newConn(hostID, platform, ws)
	h.register(hostID, c)
	go c.writeLoop(h.pingInterval, h.pongTimeout)
	c.readLoop(h, h.pingInterval, h.pongTimeout)
}

// register stores c as the connection for hostID, evicting and closing any
// previous connection for the same host (e.g. an agent that reconnected
// before the server noticed the old connection died).
func (h *Hub) register(hostID uint, c *conn) {
	h.mu.Lock()
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

// Notify enqueues a notification of the given type to each of hostIDs whose
// connection this instance holds, and returns how many were notified. Hosts
// connected to other instances (or not connected at all) are skipped:
// delivery is best-effort by design, with the interval check job and the
// agent's polling fallback as the safety nets.
func (h *Hub) Notify(msgType string, hostIDs []uint) int {
	msg := fleet.AgentWSMessage{Type: msgType}

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

// FilterDueForRenotify returns the subset of hostIDs that were not notified
// within the last grace period (and are still connected). It keeps slow
// responders from being re-notified on every interval check tick.
func (h *Hub) FilterDueForRenotify(hostIDs []uint, grace time.Duration) []uint {
	cutoff := time.Now().Add(-grace)

	h.mu.RLock()
	defer h.mu.RUnlock()
	due := make([]uint, 0, len(hostIDs))
	for _, id := range hostIDs {
		c, ok := h.conns[id]
		if !ok {
			continue
		}
		if last := c.lastNotified(); last.IsZero() || last.Before(cutoff) {
			due = append(due, id)
		}
	}
	return due
}

// ConnCount returns the number of connections this instance holds. It is the
// hook for future per-instance connection caps and rebalancing (deferred from
// the ADR-0011 POC).
func (h *Hub) ConnCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

// ConnectionInfo describes one held connection, for observability (the
// /debug/agentws endpoint).
type ConnectionInfo struct {
	HostID         uint       `json:"host_id"`
	Platform       string     `json:"platform"`
	RemoteAddr     string     `json:"remote_addr"`
	ConnectedAt    time.Time  `json:"connected_at"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
	NotifiedCount  int64      `json:"notified_count"`
	DroppedCount   int64      `json:"dropped_count"`
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
			Platform:      c.platform,
			RemoteAddr:    c.remoteAddr,
			ConnectedAt:   c.connectedAt,
			NotifiedCount: c.notified.Load(),
			DroppedCount:  c.dropped.Load(),
		}
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

// Shutdown closes all held connections.
func (h *Hub) Shutdown() {
	h.mu.Lock()
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
