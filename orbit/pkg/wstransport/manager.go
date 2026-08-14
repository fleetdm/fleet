package wstransport

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/backoff"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// notificationsPath is the server's agent WebSocket endpoint (ADR-0011).
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
	// StableAfter is how long a connection must stay up before polling stops
	// (default 1m).
	StableAfter time.Duration
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
	if o.StableAfter == 0 {
		o.StableAfter = 1 * time.Minute
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

// Manager holds the WebSocket connection to the Fleet server and falls back to
// polling when the connection is not stable. It is an oklog/run-compatible
// subsystem (Execute/Interrupt).
//
// State machine: polling runs on PollInterval whenever the WebSocket is not
// "stable" (up for at least StableAfter). A "check now" notification — or a
// polling tick — triggers one HTTP distributed/read whose queries go into the
// cache for osquery's next local poll. Polling is the guaranteed baseline: a
// dropped notification or connection can delay work by at most one polling or
// interval-check cycle, never lose it. When both the WebSocket and HTTP fail,
// the server (or the path to it) is down and both paths simply keep retrying —
// there is nothing to "downgrade" to.
type Manager struct {
	opts Options

	// wsStable is true once the current connection has been up for
	// StableAfter; polling ticks are skipped while it is set.
	wsStable atomic.Bool
	// checking singleflights checkNow: overlapping triggers (notification
	// burst, poll tick during a fetch) collapse into one distributed/read.
	checking atomic.Bool

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
		log.Info().Msg("websocket connected")

		// Catch up on anything that happened while disconnected (e.g. a live
		// query created before the connection was up).
		go m.checkNow()

		// The connection is considered stable (and polling stops) only after
		// it has stayed up for StableAfter.
		stableTimer := time.AfterFunc(m.opts.StableAfter, func() {
			m.wsStable.Store(true)
			log.Debug().Msg("websocket transport stable, polling paused")
		})
		m.readLoop(conn)
		stableTimer.Stop()
		m.wsStable.Store(false)
		_ = conn.Close()
		if m.ctx.Err() != nil {
			return
		}
		log.Info().Msg("websocket disconnected, polling resumed")

		// Jitter the reconnection so a server restart doesn't produce a
		// thundering herd of simultaneous re-dials.
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

	// The read deadline doubles as connection liveness: the server pings every
	// ServerPingInterval, and every ping refreshes the deadline. A connection
	// with no traffic for two ping intervals is dead (e.g. half-open TCP).
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
		switch msg.Type {
		case fleet.AgentWSMessageTypeDistributedRead:
			go m.checkNow()
		default:
			// Unknown notification types are ignored for forward
			// compatibility with future channels.
			log.Debug().Str("type", msg.Type).Msg("ignoring unknown websocket notification type")
		}
	}
}

// pollLoop is the fallback: it performs a distributed read on every tick while
// the WebSocket is not stable.
func (m *Manager) pollLoop() {
	ticker := time.NewTicker(m.opts.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if m.wsStable.Load() {
				continue
			}
			m.checkNow()
		case <-m.ctx.Done():
			return
		}
	}
}

// checkNow performs one HTTP distributed/read and caches the result for
// osquery's next local poll. Overlapping calls collapse into one.
func (m *Manager) checkNow() {
	if !m.checking.CompareAndSwap(false, true) {
		return
	}
	defer m.checking.Store(false)

	resp, err := m.opts.Client.DistributedRead()
	if err != nil {
		log.Debug().Err(err).Msg("distributed read failed")
		return
	}
	m.opts.Cache.Set(resp)
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
