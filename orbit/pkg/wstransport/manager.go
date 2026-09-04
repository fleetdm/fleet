package wstransport

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/backoff"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// notificationsPath is the server's agent WebSocket endpoint.
const notificationsPath = "/api/fleet/orbit/notifications"

// Options configures a Manager. ServerURL, NodeKeyFunc, Client and Cache are
// required; zero-valued tunables get the defaults below.
type Options struct {
	// ServerURL is the base URL of the Fleet server (http(s) scheme; it is
	// rewritten to ws(s) for the WebSocket dial).
	ServerURL *url.URL
	// RootCA is the path of a PEM file with server CA certificates (optional).
	RootCA string
	// InsecureSkipVerify disables TLS certificate verification.
	InsecureSkipVerify bool
	// ClientCertificate is the TLS client certificate for mTLS (optional).
	ClientCertificate *tls.Certificate
	// NodeKeyFunc returns the orbit node key used to authenticate the upgrade.
	NodeKeyFunc func() (string, error)
	// Client performs the HTTP distributed read/write calls.
	Client distributedClient
	// Cache receives the queries fetched on "check now" notifications.
	Cache *QueryCache

	// PollInterval is the fallback polling cadence (default 10s, matching
	// osquery's distributed interval).
	PollInterval time.Duration
	// ReconnectJitterMax is the random delay before reconnecting after a drop,
	// so a server restart doesn't produce a thundering herd (default 30s).
	ReconnectJitterMax time.Duration
	// BackoffBase/BackoffCap bound the reconnection backoff (default 5s/5m).
	BackoffBase time.Duration
	BackoffCap  time.Duration
	// ServerPingInterval is the server's keepalive ping cadence, used to size
	// the read deadline (default 5m; the deadline is twice this).
	ServerPingInterval time.Duration
	// HandshakeTimeout bounds the WebSocket upgrade (default 30s).
	HandshakeTimeout time.Duration
}

func (o *Options) applyDefaults() {
	if o.PollInterval == 0 {
		o.PollInterval = 10 * time.Second
	}
	if o.ReconnectJitterMax == 0 {
		o.ReconnectJitterMax = 30 * time.Second
	}
	if o.BackoffBase == 0 {
		o.BackoffBase = 5 * time.Second
	}
	if o.BackoffCap == 0 {
		o.BackoffCap = 5 * time.Minute
	}
	if o.ServerPingInterval == 0 {
		o.ServerPingInterval = 5 * time.Minute
	}
	if o.HandshakeTimeout == 0 {
		o.HandshakeTimeout = 30 * time.Second
	}
}

// iterState tracks one distributed read iteration: the HTTP distributed/read,
// osquery picking up the cached queries on its local poll, and the resulting
// distributed/write. Exactly one iteration runs at a time; triggers arriving
// mid-iteration coalesce into a single queued follow-up (see Manager.trigger).
type iterState int

const (
	iterIdle iterState = iota
	// iterReading: the HTTP distributed/read is in flight.
	iterReading
	// iterDelivered: queries cached, waiting for osquery's next local poll.
	iterDelivered
	// iterAwaitingWrite: osquery took the queries; the iteration ends when
	// their distributed/write completes — or, for a pass that never writes
	// (osquery crashed mid-run, or every query discovery-filtered), on
	// osquery's next local poll (see Manager.takeQueries).
	iterAwaitingWrite
)

// Manager holds the WebSocket connection to the Fleet server, polling on
// PollInterval while it is disconnected. It is an oklog/run-compatible
// subsystem (Execute/Interrupt).
//
// A "check now" notification — or a poll tick — triggers one distributed read
// iteration (see iterState) whose queries go into the cache for osquery's
// next local poll. There is deliberately no catch-up read on (re)connect: a
// notification lost around a connect is recovered by the server's interval
// check job, which re-notifies until the work is done. A half-open connection
// is bounded by the keepalive read deadline.
type Manager struct {
	opts Options

	// connected is true while the WebSocket is up; poll ticks are skipped
	// while it is set.
	connected atomic.Bool

	// mu guards the iteration state machine. All triggers and all osquery
	// pass boundaries funnel through it, so at most one iteration is in
	// flight and at most one follow-up is queued.
	mu      sync.Mutex
	state   iterState
	pending bool

	cancel context.CancelFunc
	ctx    context.Context
}

// NewManager creates the WebSocket transport subsystem.
func NewManager(opts Options) *Manager {
	opts.applyDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		opts:   opts,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Execute runs the connection and polling loops until Interrupt is called.
func (m *Manager) Execute() error {
	log.Info().Str("url", m.opts.ServerURL.String()).Msg("starting websocket transport")

	pollDone := make(chan struct{})
	go func() {
		defer close(pollDone)
		m.pollLoop()
	}()
	m.connectionLoop()
	<-pollDone
	return nil
}

// Interrupt stops the subsystem.
func (m *Manager) Interrupt(err error) {
	m.cancel()
}

// connectionLoop dials, services and re-dials the WebSocket until interrupted.
func (m *Manager) connectionLoop() {
	tracker := backoff.New(m.opts.BackoffBase, m.opts.BackoffCap)

	for {
		if m.ctx.Err() != nil {
			return
		}

		conn, err := m.dial()
		if err != nil {
			tracker.RecordFailure()
			log.Debug().Err(err).Dur("retry_in", tracker.Interval()).Msg("websocket connect failed")
			if !m.sleep(tracker.Interval()) {
				return
			}
			continue
		}
		tracker.RecordSuccess()
		log.Debug().Msg("websocket connected, polling paused")

		m.connected.Store(true)
		m.readLoop(conn)
		m.connected.Store(false)
		_ = conn.Close()
		if m.ctx.Err() != nil {
			return
		}
		log.Debug().Msg("websocket disconnected, polling resumed")

		// Jitter re-dials so a server restart doesn't produce a thundering herd.
		if !m.sleep(time.Duration(rand.Int64N(int64(m.opts.ReconnectJitterMax)))) { //nolint:gosec // jitter does not need cryptographic randomness
			return
		}
	}
}

// readLoop consumes notifications until the connection breaks or the manager
// is interrupted.
func (m *Manager) readLoop(conn *websocket.Conn) {
	// Unblock the blocking read when interrupted.
	stop := context.AfterFunc(m.ctx, func() { _ = conn.Close() })
	defer stop()

	// The read deadline doubles as liveness: the server pings every
	// ServerPingInterval and every ping refreshes the deadline, so no traffic
	// for two intervals means a dead (e.g. half-open) connection.
	deadline := 2 * m.opts.ServerPingInterval
	_ = conn.SetReadDeadline(time.Now().Add(deadline))
	defaultPingHandler := conn.PingHandler()
	conn.SetPingHandler(func(message string) error {
		_ = conn.SetReadDeadline(time.Now().Add(deadline))
		return defaultPingHandler(message)
	})

	for {
		var msg fleet.AgentWSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if m.ctx.Err() == nil {
				log.Debug().Err(err).Msg("websocket read failed")
			}
			return
		}
		log.Debug().Str("type", msg.Type).Str("reason", msg.Reason).Msg("check-now notification received")
		switch msg.Type {
		case fleet.AgentWSMessageTypeDistributedRead:
			m.trigger()
		default:
			// Ignored for forward compatibility.
			log.Debug().Str("type", msg.Type).Msg("ignoring unknown websocket notification type")
		}
	}
}

// pollLoop is the fallback: it triggers a distributed read iteration on every
// tick while the WebSocket is down.
func (m *Manager) pollLoop() {
	ticker := time.NewTicker(m.opts.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if m.connected.Load() {
				continue
			}
			m.trigger()
		case <-m.ctx.Done():
			return
		}
	}
}

// trigger starts a distributed read iteration, or queues one if an iteration
// is already in flight. It is the single entry point for notifications and
// poll ticks, so both coalesce: at most one iteration in flight, at most one
// follow-up queued. Non-blocking.
func (m *Manager) trigger() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state != iterIdle {
		m.pending = true
		return
	}
	m.state = iterReading
	go m.runRead()
}

// runRead performs the iteration's HTTP distributed/read and caches the
// queries for osquery's next local poll. Called with state == iterReading.
func (m *Manager) runRead() {
	resp, err := m.opts.Client.DistributedRead()

	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		log.Debug().Err(err).Msg("distributed read failed")
		// Drop any queued trigger too: firing it now would tight-loop against
		// a failing server, and the server re-notifies while work is due.
		m.state = iterIdle
		m.pending = false
		return
	}
	m.opts.Cache.Set(resp)
	if len(resp.Queries) == 0 {
		// No work handed out, so no write will follow: the iteration ends here.
		m.state = iterIdle
		m.firePendingLocked()
		return
	}
	m.state = iterDelivered
}

// firePendingLocked starts the queued iteration, if any. Callers must hold
// m.mu, with state just returned to iterIdle.
func (m *Manager) firePendingLocked() {
	if !m.pending || m.ctx.Err() != nil {
		return
	}
	m.pending = false
	m.state = iterReading
	go m.runRead()
}

// takeQueries hands the cached queries to osquery's local distributed poll
// and advances the iteration state. The poll doubles as the recovery signal
// for a pass that will never write: osquery's distributed passes are
// sequential, so an empty poll arriving while a write is still expected
// proves the pass that took the queries died (osquery crashed mid-run) or
// finished without writing (every query discovery-filtered) — either way the
// iteration is over.
func (m *Manager) takeQueries() (queries, discovery map[string]string, accelerate uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	queries, discovery, accelerate = m.opts.Cache.Take()
	switch {
	case len(queries) > 0:
		m.state = iterAwaitingWrite
	case m.state == iterAwaitingWrite:
		m.state = iterIdle
		m.firePendingLocked()
	}
	return queries, discovery, accelerate
}

// writeDone closes the iteration when a pass's distributed/write completed,
// even a failed one: the pass is over either way, and on failure the host
// stays due in the server's view, which re-notifies.
func (m *Manager) writeDone() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == iterAwaitingWrite || m.state == iterDelivered {
		m.state = iterIdle
		m.firePendingLocked()
	}
}

// dial performs the authenticated WebSocket upgrade.
func (m *Manager) dial() (*websocket.Conn, error) {
	nodeKey, err := m.opts.NodeKeyFunc()
	if err != nil {
		return nil, fmt.Errorf("get node key: %w", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: m.opts.InsecureSkipVerify, //nolint:gosec // set from the --insecure orbit flag, same as the HTTP client
	}
	if m.opts.RootCA != "" {
		pool, err := certificate.LoadPEM(m.opts.RootCA)
		if err != nil {
			return nil, fmt.Errorf("load root CA: %w", err)
		}
		tlsConfig.RootCAs = pool
	}
	if m.opts.ClientCertificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*m.opts.ClientCertificate}
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: m.opts.HandshakeTimeout,
		TLSClientConfig:  tlsConfig,
	}

	wsURL := *m.opts.ServerURL
	switch wsURL.Scheme {
	case "https":
		wsURL.Scheme = "wss"
	case "http":
		wsURL.Scheme = "ws"
	}
	wsURL.Path = notificationsPath

	header := http.Header{}
	header.Set("Authorization", "Node key "+nodeKey)

	conn, resp, err := dialer.DialContext(m.ctx, wsURL.String(), header)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket dial (HTTP %d): %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial: %w", err)
	}
	return conn, nil
}

// sleep waits for d or until interrupted; it reports false when interrupted.
func (m *Manager) sleep(d time.Duration) bool {
	if d <= 0 {
		return m.ctx.Err() == nil
	}
	select {
	case <-time.After(d):
		return true
	case <-m.ctx.Done():
		return false
	}
}
