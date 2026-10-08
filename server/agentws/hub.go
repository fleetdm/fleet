package agentws

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/gorilla/websocket"
)

// HostLister loads hosts by ID, to resolve the fleet of held connections for
// scoped notifications. It is satisfied by fleet.Datastore.
type HostLister interface {
	ListHostsLiteByIDs(ctx context.Context, ids []uint) ([]*fleet.Host, error)
}

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

	// pacer and hostLister serve scoped notifications; nil until
	// EnableScopedNotifications.
	pacer          *pacer
	hostLister     HostLister
	scopeBatchSize int
	// scopeQueue holds the scoped notifications waiting for the scope worker
	// (see QueueScope); scopeWake (size 1) wakes it.
	scopeMu    sync.Mutex
	scopeQueue []ScopedNotification
	scopeWake  chan struct{}

	// notifyCounts counts the notifications enqueued per message type and
	// reason, for observability (see NotifyCounts).
	notifyCountsMu sync.Mutex
	notifyCounts   map[string]map[string]int64
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
		notifyCounts: make(map[string]map[string]int64),
	}
}

// EnableScopedNotifications starts the pacer that spreads scoped
// notifications (see NotifyScope) over spread, until ctx is done. Fleet
// scopes are resolved by loading the held hosts in batches of batchSize.
func (h *Hub) EnableScopedNotifications(ctx context.Context, lister HostLister, batchSize int, spread time.Duration) {
	h.pacer = newPacer(h, spread)
	h.hostLister = lister
	h.scopeBatchSize = batchSize
	h.scopeWake = make(chan struct{}, 1)
	go h.pacer.run(ctx)
	go h.runScopeWorker(ctx)
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
	sent := 0
	for _, id := range hostIDs {
		if c, ok := h.conns[id]; ok {
			c.enqueue(msg)
			sent++
		}
	}
	h.mu.RUnlock()

	if sent > 0 {
		h.countNotified(msgType, reason, sent)
	}
	return sent
}

// NotifyScope queues a notification for each held connection in scope and
// returns how many were queued. The queued notifications are released by the
// pacer over the spread window: a change to a whole fleet must not make every
// agent call back at once. It blocks while loading the held hosts of a fleet
// scope.
func (h *Hub) NotifyScope(ctx context.Context, msgType, reason string, scope fleet.AgentNotificationScope) (int, error) {
	return h.notifyScopes(ctx, []ScopedNotification{{MsgType: msgType, Reason: reason, Scope: scope}})
}

// ScopedNotification is a notification for every held connection in Scope.
type ScopedNotification struct {
	MsgType string
	Reason  string
	Scope   fleet.AgentNotificationScope
}

// QueueScope is NotifyScope without blocking: scoped notifications are
// resolved by a single worker, which loads the held hosts once for all the
// fleet scopes queued meanwhile (e.g. a GitOps run changing many fleets).
// Errors are logged.
func (h *Hub) QueueScope(n ScopedNotification) {
	h.scopeMu.Lock()
	h.scopeQueue = append(h.scopeQueue, n)
	h.scopeMu.Unlock()
	select {
	case h.scopeWake <- struct{}{}:
	default:
	}
}

func (h *Hub) runScopeWorker(ctx context.Context) {
	for {
		select {
		case <-h.scopeWake:
		case <-ctx.Done():
			return
		}
		h.scopeMu.Lock()
		queued := h.scopeQueue
		h.scopeQueue = nil
		h.scopeMu.Unlock()
		if _, err := h.notifyScopes(ctx, queued); err != nil {
			h.logger.ErrorContext(ctx, "notify agents in scope", "err", err)
		}
	}
}

// notifyScopes queues the notifications for the held connections in their
// scopes, loading the held hosts at most once.
func (h *Hub) notifyScopes(ctx context.Context, ns []ScopedNotification) (int, error) {
	if h.pacer == nil {
		return 0, errors.New("scoped notifications are not enabled")
	}

	hostIDs := h.HeldHostIDs()
	var teamOf map[uint]uint // host ID → fleet ID (0 for "No fleet"), loaded on demand
	queued := 0
	var errs []error
	for _, n := range ns {
		teamID, isTeam := n.Scope.TeamID()
		switch {
		case n.Scope == fleet.AgentNotificationScopeGlobal:
			queued += h.pacer.add(n.MsgType, n.Reason, hostIDs)
			continue
		case !isTeam:
			errs = append(errs, ctxerr.Errorf(ctx, "unknown agent notification scope %q", n.Scope))
			continue
		}
		if teamOf == nil {
			teamOf = make(map[uint]uint, len(hostIDs))
			for chunk := range slices.Chunk(hostIDs, h.scopeBatchSize) {
				hosts, err := h.hostLister.ListHostsLiteByIDs(ctx, chunk)
				if err != nil {
					return queued, ctxerr.Wrap(ctx, err, "list hosts for scoped notification")
				}
				for _, host := range hosts {
					// Fleet ID 0 is "No fleet".
					teamOf[host.ID] = ptr.ValOrZero(host.TeamID)
				}
			}
		}
		var inTeam []uint
		for id, tid := range teamOf {
			if tid == teamID {
				inTeam = append(inTeam, id)
			}
		}
		queued += h.pacer.add(n.MsgType, n.Reason, inTeam)
	}
	return queued, errors.Join(errs...)
}

// NotifyPaced is Notify through the pacer: a few hosts are notified on its
// next tick, a large set is spread over the spread window. Without scoped
// notifications enabled, it is Notify.
func (h *Hub) NotifyPaced(msgType, reason string, hostIDs []uint) {
	if h.pacer == nil {
		h.Notify(msgType, reason, hostIDs)
		return
	}
	held := make([]uint, 0, len(hostIDs))
	h.mu.RLock()
	for _, id := range hostIDs {
		if _, ok := h.conns[id]; ok {
			held = append(held, id)
		}
	}
	h.mu.RUnlock()
	h.pacer.add(msgType, reason, held)
}

// PacerQueueLen returns the number of scoped notifications waiting to be
// released, or 0 when scoped notifications are not enabled.
func (h *Hub) PacerQueueLen() int {
	if h.pacer == nil {
		return 0
	}
	return h.pacer.queueLen()
}

func (h *Hub) countNotified(msgType, reason string, n int) {
	// Live query reasons embed the campaign ID; count them together.
	if strings.HasPrefix(reason, fleet.AgentWSReasonLiveQueryName("")) {
		reason = fleet.AgentWSReasonLiveQueryName("*")
	}
	h.notifyCountsMu.Lock()
	defer h.notifyCountsMu.Unlock()
	byReason := h.notifyCounts[msgType]
	if byReason == nil {
		byReason = make(map[string]int64)
		h.notifyCounts[msgType] = byReason
	}
	byReason[reason] += int64(n)
}

// NotifyCounts returns the number of notifications enqueued since boot, by
// message type and reason.
func (h *Hub) NotifyCounts() map[string]map[string]int64 {
	h.notifyCountsMu.Lock()
	defer h.notifyCountsMu.Unlock()
	counts := make(map[string]map[string]int64, len(h.notifyCounts))
	for msgType, byReason := range h.notifyCounts {
		counts[msgType] = maps.Clone(byReason)
	}
	return counts
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
	// NotifiedByType breaks NotifiedCount down by message type.
	NotifiedByType map[string]int64 `json:"notified_by_type,omitempty"`
	// CoalescedCount counts notifications merged into a pending notification
	// of the same type, i.e. delivered as one message.
	CoalescedCount int64 `json:"coalesced_count"`
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
			HostID:         c.hostID,
			Hostname:       c.hostname,
			Platform:       c.platform,
			RemoteAddr:     c.remoteAddr,
			ConnectedAt:    c.connectedAt,
			NotifiedCount:  c.notified.Load(),
			NotifiedByType: c.notifiedByTypeSnapshot(),
			CoalescedCount: c.coalesced.Load(),
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
