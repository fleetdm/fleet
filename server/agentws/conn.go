// Package agentws implements the server side of the agent WebSocket
// notification transport (ADR-0011): a per-instance registry of agent
// connections over which the server pushes tiny "check now" notifications.
// The channel is strictly server-to-agent and carries no query content,
// config, or results.
package agentws

import (
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
)

// maxInboundMessageSize caps inbound data messages, which readLoop buffers
// (ReadMessage) before discarding. The channel is server-to-agent only, so
// agents send nothing but control frames; without a cap, a misbehaving agent
// with a valid node key could force large allocations. Exceeding it closes
// the connection (gorilla sends close code 1009 and errors the read).
const maxInboundMessageSize = 1024

type conn struct {
	hostID      uint
	hostname    string
	platform    string
	ws          *websocket.Conn
	connectedAt time.Time
	remoteAddr  string

	closeOnce sync.Once
	done      chan struct{}

	// pending holds the notifications not yet written, as message type →
	// latest reason. Notifications carry no data, so a notification of a type
	// already pending is merged into it: nothing is ever dropped, and the
	// buffer is bounded by the number of message types. wake (size 1) tells
	// writeLoop that pending is not empty.
	mu      sync.Mutex
	pending map[string]string
	wake    chan struct{}
	// notifiedByType counts enqueued notifications per message type, for
	// observability (see Hub.Snapshot). Protected by mu.
	notifiedByType map[string]int64

	// lastNotifiedNano/lastNotifyReason record the last enqueued notification
	// for observability (see Hub.Snapshot). They are updated independently, so
	// a snapshot may pair a timestamp with a concurrent enqueue's reason.
	lastNotifiedNano atomic.Int64
	lastNotifyReason atomic.Pointer[string]
	// notified/coalesced count enqueued notifications and those merged into a
	// pending one of the same type, for observability (see Hub.Snapshot).
	notified  atomic.Int64
	coalesced atomic.Int64
	// counting is the byte-counting wrapper around the underlying net.Conn,
	// installed at upgrade time; nil when the connection wasn't wrapped (e.g.
	// tests that dial the hub directly).
	counting *countingConn
}

func newConn(hostID uint, hostname, platform string, ws *websocket.Conn) *conn {
	c := &conn{
		hostID:      hostID,
		hostname:    hostname,
		platform:    platform,
		ws:          ws,
		connectedAt: time.Now(),
		remoteAddr:  ws.RemoteAddr().String(),
		done:        make(chan struct{}),

		pending:        make(map[string]string),
		wake:           make(chan struct{}, 1),
		notifiedByType: make(map[string]int64),
	}
	if cc, ok := ws.NetConn().(*countingConn); ok {
		c.counting = cc
	}
	return c
}

// bytesInOut returns the raw bytes received/sent on the connection, or zeros
// when byte counting isn't installed.
func (c *conn) bytesInOut() (in, out int64) {
	if c.counting == nil {
		return 0, 0
	}
	return c.counting.bytesIn.Load(), c.counting.bytesOut.Load()
}

// enqueue queues msg for delivery, merging it into a pending notification of
// the same type (keeping the latest reason).
func (c *conn) enqueue(msg fleet.AgentWSMessage) {
	c.lastNotifiedNano.Store(time.Now().UnixNano())
	c.lastNotifyReason.Store(&msg.Reason)
	c.notified.Add(1)

	c.mu.Lock()
	if _, ok := c.pending[msg.Type]; ok {
		c.coalesced.Add(1)
	}
	c.pending[msg.Type] = msg.Reason
	c.notifiedByType[msg.Type]++
	c.mu.Unlock()

	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// takePending returns the pending notifications, sorted by type, and clears
// them.
func (c *conn) takePending() []fleet.AgentWSMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgs := make([]fleet.AgentWSMessage, 0, len(c.pending))
	for _, msgType := range slices.Sorted(maps.Keys(c.pending)) {
		msgs = append(msgs, fleet.AgentWSMessage{Type: msgType, Reason: c.pending[msgType]})
	}
	clear(c.pending)
	return msgs
}

// notifiedByTypeSnapshot returns a copy of the per-type notification counts.
func (c *conn) notifiedByTypeSnapshot() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.notifiedByType)
}

// lastNotifyReasonLoad returns the reason of the last enqueued notification,
// or "" when none was enqueued yet.
func (c *conn) lastNotifyReasonLoad() string {
	if reason := c.lastNotifyReason.Load(); reason != nil {
		return *reason
	}
	return ""
}

func (c *conn) lastNotified() time.Time {
	nano := c.lastNotifiedNano.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// close is idempotent and unblocks both loops.
func (c *conn) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.ws.Close()
	})
}

// writeLoop is the single writer for the underlying WebSocket: it owns both
// JSON notification writes and keepalive pings. gorilla/websocket does not
// allow concurrent writers, so all writes MUST funnel through here.
func (c *conn) writeLoop(pingInterval, pongTimeout time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.wake:
			for _, msg := range c.takePending() {
				_ = c.ws.SetWriteDeadline(time.Now().Add(pongTimeout))
				if err := c.ws.WriteJSON(msg); err != nil {
					c.close()
					return
				}
			}
		case <-ticker.C:
			if err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(pongTimeout)); err != nil {
				c.close()
				return
			}
		case <-c.done:
			return
		}
	}
}

// readLoop consumes control frames (pongs) and discards any data frames; the
// channel is server-to-agent only. The read deadline doubles as the liveness
// check: it is refreshed on every pong, so a peer that stops answering pings
// times out and the connection is torn down.
func (c *conn) readLoop(hub *Hub, pingInterval, pongTimeout time.Duration) {
	defer func() {
		hub.unregister(c.hostID, c)
		c.close()
	}()

	c.ws.SetReadLimit(maxInboundMessageSize)
	deadline := pingInterval + pongTimeout
	_ = c.ws.SetReadDeadline(time.Now().Add(deadline))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(deadline))
	})
	for {
		if _, _, err := c.ws.ReadMessage(); err != nil {
			return
		}
	}
}
