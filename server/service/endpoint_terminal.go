// Package service — web terminal relay.
//
// Architecture:
//
//	Browser ──WS──► Fleet server ──WS──► Orbit agent ──pipes──► bash / powershell
//
// The Fleet server acts as a pure relay: it holds one gorilla WebSocket
// connection to the browser and one to the orbit agent, then copies bytes
// between them.  Sessions are identified by a UUID created when the admin
// clicks "Open Terminal".  The session ID is surfaced to the orbit agent
// through the normal OrbitConfigNotifications polling mechanism (field
// PendingTerminalSessionIDs); orbit dials back and opens the PTY.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/gorilla/mux"
	gws "github.com/gorilla/websocket"
)

// ─── WebSocket heartbeat constants ───────────────────────────────────────────

const (
	// terminalPingPeriod is how often the Fleet server sends a WS ping to the
	// orbit agent to verify the connection is still alive.
	terminalPingPeriod = 30 * time.Second

	// terminalPongWait is how long the server waits for a pong reply before
	// treating the connection as dead and tearing down the session.
	// Must be greater than terminalPingPeriod.
	terminalPongWait = 60 * time.Second

	// terminalIdleTimeout tears down a session when no data has flowed in
	// either direction for this long.  This caps how long a forgotten but
	// healthy shell can stay open without an absolute-timeout sweep.
	terminalIdleTimeout = 30 * time.Minute
)

// ─── message types ────────────────────────────────────────────────────────────

// terminalMsg is the JSON envelope used on both the browser-facing and
// orbit-facing WebSocket connections.
//
//   - type "input"  – keyboard data from the browser (base64-encoded)
//   - type "output" – PTY output from orbit (base64-encoded)
//   - type "resize" – terminal resize from the browser (cols / rows)
//   - type "error"  – diagnostic message from the server
type terminalMsg struct {
	Type string `json:"type"`
	// Data carries base64-encoded bytes for "input" and "output" messages.
	Data string `json:"data,omitempty"`
	// Cols / Rows are set for "resize" messages.
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// ─── WebSocket upgraders ──────────────────────────────────────────────────────

// newBrowserUpgrader builds a WebSocket upgrader for browser connections.
// When websocketsAllowUnsafeOrigin is true (mirrors server.websockets_allow_unsafe_origin
// used by the live-query WS), all Origins are accepted — useful for reverse-proxy
// or dev setups where Origin and Host differ.  Otherwise the upgrader enforces
// same-origin to prevent cross-site WebSocket hijacking.
func newBrowserUpgrader(websocketsAllowUnsafeOrigin bool) gws.Upgrader {
	checkOrigin := func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		// Allow only requests whose Origin host matches the Host header.
		return u.Host == r.Host
	}
	if websocketsAllowUnsafeOrigin {
		checkOrigin = func(r *http.Request) bool { return true }
	}
	return gws.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   64 * 1024,
		WriteBufferSize:  64 * 1024,
		CheckOrigin:      checkOrigin,
	}
}

// orbitUpgrader is used for orbit agent connections.  Orbit is not a browser
// and does not send an Origin header, so origin checking is skipped.
var orbitUpgrader = gws.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   64 * 1024,
	WriteBufferSize:  64 * 1024,
	CheckOrigin:      func(r *http.Request) bool { return true },
}

// ─── Long-poll notification store ─────────────────────────────────────────────

// terminalNotifyStore lets the session-create path wake up any orbit agent
// that is blocked in a long-poll on /api/fleet/orbit/terminal/notify.
// When orbit is waiting in that endpoint, a new session will be delivered
// in sub-100 ms instead of waiting for the next config poll cycle.
var terminalNotifyStore = newTerminalNotifyStore()

type terminalNotifyStoreT struct {
	mu      sync.Mutex
	waiters map[uint][]chan struct{} // hostID → waiting channels
}

func newTerminalNotifyStore() *terminalNotifyStoreT {
	return &terminalNotifyStoreT{waiters: make(map[uint][]chan struct{})}
}

// subscribe registers a channel that will be closed when a terminal session
// arrives for hostID.  Call unsubscribe when done (deferred is fine).
func (s *terminalNotifyStoreT) subscribe(hostID uint) (ch chan struct{}, unsubscribe func()) {
	ch = make(chan struct{}, 1)
	s.mu.Lock()
	s.waiters[hostID] = append(s.waiters[hostID], ch)
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		ws := s.waiters[hostID]
		for i, w := range ws {
			if w == ch {
				s.waiters[hostID] = append(ws[:i], ws[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}
}

// notifyHost wakes all orbit agents waiting on a long-poll for hostID.
func (s *terminalNotifyStoreT) notifyHost(hostID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.waiters[hostID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ─── HTTP: create terminal session ───────────────────────────────────────────

// createTerminalSessionRequest is decoded from
//
//	POST /api/v1/fleet/hosts/{id}/terminal
type createTerminalSessionRequest struct {
	ID uint `url:"id"`
}

type createTerminalSessionResponse struct {
	SessionID string `json:"session_id"`
	Err       error  `json:"error,omitempty"`
}

func (r createTerminalSessionResponse) Error() error { return r.Err }

func createTerminalSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createTerminalSessionRequest)
	sessionID, err := svc.CreateTerminalSession(ctx, req.ID)
	if err != nil {
		return createTerminalSessionResponse{Err: err}, nil
	}
	return createTerminalSessionResponse{SessionID: sessionID}, nil
}

// ─── Browser-side WebSocket ───────────────────────────────────────────────────

// makeTerminalBrowserHandler returns an http.HandlerFunc for
//
//	GET /api/v1/fleet/hosts/{id}/terminal/{session_id}/ws
//
// Authentication: the browser sends its Fleet session token as the first
// JSON text message: {"token":"<fleet-session-token>"}
func makeTerminalBrowserHandler(svc fleet.Service, logger *slog.Logger, cfg config.ServerConfig) http.HandlerFunc {
	upgrader := newBrowserUpgrader(cfg.WebsocketsAllowUnsafeOrigin)
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sessionID := vars["session_id"]
		if sessionID == "" {
			http.Error(w, "missing session_id", http.StatusBadRequest)
			return
		}

		sess, ok := terminalStore.get(sessionID)
		if !ok {
			http.Error(w, "terminal session not found", http.StatusNotFound)
			return
		}

		// Verify the host ID in the URL matches the session's bound host.
		// This is a belt-and-suspenders check; auth already scopes the session,
		// but an explicit mismatch rejection avoids confusion and information leak.
		if rawID := vars["id"]; rawID != "" {
			if id, err := strconv.ParseUint(rawID, 10, 64); err == nil {
				if uint(id) != sess.hostID {
					http.Error(w, "session belongs to a different host", http.StatusForbidden)
					return
				}
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("terminal browser: WebSocket upgrade failed", "err", err)
			return
		}
		defer conn.Close()

		// Limit message size to prevent memory exhaustion.
		conn.SetReadLimit(256 * 1024)

		// First message must carry a valid Fleet session token.
		user, err := authenticateTerminalBrowser(r.Context(), conn, svc)
		if err != nil {
			logger.Warn("terminal browser: authentication failed", "err", err)
			writeTerminalError(conn, "unauthorized")
			terminalStore.remove(sessionID)
			return
		}

		// Mark the session as browser-claimed and notify orbit.  Orbit only
		// sees session IDs after this point, so no shell is started without a
		// verified browser connection.  markBrowserClaimed also enforces:
		//   • single-use: a second browser tab for the same session is rejected.
		//   • creator-bound: only the user who created the session can attach.
		if !terminalStore.markBrowserClaimed(sessionID, user.ID) {
			// Session was swept, already claimed, or caller is not the creator.
			writeTerminalError(conn, "session expired or not authorized")
			return
		}
		terminalNotifyStore.notifyHost(sess.hostID)

		// Wait up to 30 s for the orbit agent to dial in.
		select {
		case <-sess.orbitConnected:
			// Shell is live — record the activity now.
			if err := svc.NewActivity(r.Context(), user, fleet.ActivityTypeConnectedToHost{
				HostID:          sess.hostID,
				HostDisplayName: sess.hostDisplayName,
			}); err != nil {
				logger.Error("terminal browser: failed to record activity", "err", err)
			}
			// Tell the browser the shell is ready.  The frontend only
			// transitions to "connected" state on this frame, so errors
			// before it (orbit timeout, auth failure) surface as an error
			// banner instead of silently closing the tab.
			readyMsg, _ := json.Marshal(terminalMsg{Type: "ready"})
			if err := conn.WriteMessage(gws.TextMessage, readyMsg); err != nil {
				terminalStore.remove(sessionID)
				return
			}
		case <-time.After(30 * time.Second):
			writeTerminalError(conn, "timed out waiting for host agent to connect")
			terminalStore.remove(sessionID)
			return
		case <-sess.done:
			writeTerminalError(conn, "session closed before agent connected")
			return
		}

		// Apply the same absolute wall-clock limit used by the orbit handler and
		// the TTL sweeper.  When the timeout fires, ctx.Done() unblocks the
		// relay loops and the deferred conn.Close() tears down the browser WS.
		ctx, cancel := context.WithTimeout(r.Context(), maxSessionDuration)
		defer cancel()

		// Ping/pong: keep the browser WebSocket alive through idle-connection
		// timeouts on load balancers (e.g. AWS ALB default is 60 s).  The
		// pong handler resets the read deadline on each reply; if the browser
		// stops responding the read goroutine below returns a deadline error
		// and cancel() tears down the session.
		conn.SetReadDeadline(time.Now().Add(terminalPongWait)) //nolint:errcheck
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(terminalPongWait))
		})

		go func() {
			t := time.NewTicker(terminalPingPeriod)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					if err := conn.WriteControl(gws.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
						cancel()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// Browser → fromBrowser (keyboard input, resize events).
		go func() {
			defer cancel()
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				select {
				case sess.fromBrowser <- msg:
				case <-sess.done:
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		// toBrowser → browser (PTY output).
		for {
			select {
			case data := <-sess.toBrowser:
				if err := conn.WriteMessage(gws.TextMessage, data); err != nil {
					terminalStore.remove(sessionID)
					return
				}
			case <-sess.done:
				return
			case <-ctx.Done():
				terminalStore.remove(sessionID)
				return
			}
		}
	}
}

// authenticateTerminalBrowser reads the first WebSocket message, verifies it
// contains a valid Fleet session token for a global admin, and returns the
// authenticated user.
func authenticateTerminalBrowser(ctx context.Context, conn *gws.Conn, svc fleet.Service) (*fleet.User, error) {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read auth message: %w", err)
	}
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		return nil, fmt.Errorf("clear read deadline: %w", err)
	}

	var msg struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal auth message: %w", err)
	}

	vc, err := auth.AuthViewer(ctx, msg.Token, svc)
	if err != nil {
		return nil, fleet.NewAuthFailedError(err.Error())
	}
	// Terminal sessions are restricted to global admins only.
	if vc.User == nil || vc.User.GlobalRole == nil || *vc.User.GlobalRole != fleet.RoleAdmin {
		return nil, fleet.NewAuthFailedError("terminal sessions require global admin role")
	}
	// Reject accounts that must reset their password before taking any action.
	if vc.User.IsAdminForcedPasswordReset() {
		return nil, fleet.ErrPasswordResetRequired
	}
	return vc.User, nil
}

// ─── Orbit-side WebSocket ─────────────────────────────────────────────────────

// makeTerminalOrbitHandler returns an http.HandlerFunc for
//
//	GET /api/fleet/orbit/terminal/{session_id}
//
// The orbit node key is read from the Authorization header
// ("FleetOrbitNodeKey <key>") and validated via svc.AuthenticateOrbitHost.
// This mirrors what the orbit auth middleware does for normal POST endpoints,
// but is done inline here because WebSocket upgrades must be GET requests.
func makeTerminalOrbitHandler(svc fleet.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sessionID := vars["session_id"]
		if sessionID == "" {
			http.Error(w, "missing session_id", http.StatusBadRequest)
			return
		}

		// Authenticate the orbit node key from the Authorization header.
		// Orbit agents send "FleetOrbitNodeKey <key>" (same scheme used by all
		// orbit HTTP endpoints, extracted here manually because WebSocket
		// upgrades are GET requests and can't use the normal POST JSON body).
		nodeKey := strings.TrimPrefix(r.Header.Get("Authorization"), "FleetOrbitNodeKey ")
		if nodeKey == "" {
			http.Error(w, "missing orbit node key", http.StatusUnauthorized)
			return
		}
		host, _, err := svc.AuthenticateOrbitHost(r.Context(), nodeKey)
		if err != nil {
			http.Error(w, "unauthorized orbit node key", http.StatusUnauthorized)
			return
		}

		// claim atomically validates host ownership, checks for duplicates, and
		// marks the session connected — all under the store's mutex.  This
		// prevents two concurrent orbit goroutines from both passing the
		// duplicate check and then both calling close(sess.orbitConnected),
		// which would panic.  claim also enforces that the browser has already
		// authenticated (browserClaimed == true) before orbit can start a shell.
		sess, ok := terminalStore.claim(sessionID, host.ID)
		if !ok {
			http.Error(w, "session not found, wrong host, already connected, or browser not ready", http.StatusConflict)
			return
		}

		conn, err := orbitUpgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("terminal orbit: WebSocket upgrade failed", "err", err)
			// Upgrade failed; tear down the session we already claimed.
			terminalStore.remove(sessionID)
			return
		}
		defer func() {
			conn.Close()
			terminalStore.remove(sessionID)
		}()

		// Limit message size to prevent memory exhaustion.
		conn.SetReadLimit(256 * 1024)

		// Signal the browser side that the agent is ready.
		close(sess.orbitConnected)

		// Enforce an absolute wall-clock limit and enable WebSocket heartbeats
		// to detect half-open TCP connections before the OS keepalive fires.
		ctx, cancel := context.WithTimeout(r.Context(), maxSessionDuration)
		defer cancel()

		// Ping/pong: set an initial read deadline; the pong handler resets it
		// on each pong.  If orbit stops responding (dead network, hung agent),
		// ReadMessage returns a deadline error and the session is torn down.
		conn.SetReadDeadline(time.Now().Add(terminalPongWait)) //nolint:errcheck
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(terminalPongWait))
		})

		// Send periodic pings to the orbit agent.
		go func() {
			t := time.NewTicker(terminalPingPeriod)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					if err := conn.WriteControl(gws.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
						cancel()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// When the session is torn down (browser disconnect, TTL, auth failure,
		// absolute timeout), close the WS so blocking ReadMessage returns.
		go func() {
			select {
			case <-sess.done:
				conn.Close()
			case <-ctx.Done():
				conn.Close()
			}
		}()

		// Idle timeout: tear down if no data flows in either direction for
		// terminalIdleTimeout.  A buffered channel (capacity 1) means the
		// resetIdle call in the hot relay paths is always non-blocking.
		idleActivity := make(chan struct{}, 1)
		resetIdle := func() {
			select {
			case idleActivity <- struct{}{}:
			default:
			}
		}
		go func() {
			t := time.NewTimer(terminalIdleTimeout)
			defer t.Stop()
			for {
				select {
				case <-idleActivity:
					if !t.Stop() {
						select {
						case <-t.C:
						default:
						}
					}
					t.Reset(terminalIdleTimeout)
				case <-t.C:
					cancel() // idle timeout — no data for terminalIdleTimeout
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		// fromBrowser → orbit WebSocket (input / resize events).
		go func() {
			defer cancel()
			for {
				select {
				case data := <-sess.fromBrowser:
					if err := conn.WriteMessage(gws.TextMessage, data); err != nil {
						return
					}
					resetIdle()
				case <-sess.done:
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		// Orbit WebSocket → toBrowser (PTY output).
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			resetIdle()
			select {
			case sess.toBrowser <- msg:
			case <-sess.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}
}

// ─── Long-poll notify endpoint ────────────────────────────────────────────────

// makeTerminalNotifyOrbitHandler handles
//
//	GET /api/fleet/orbit/terminal/notify
//
// Orbit calls this in a tight loop.  The request blocks (up to 30 s) until a
// terminal session is created for this host, then returns immediately with the
// pending session IDs.  If no session arrives within 30 s the response has an
// empty list and orbit loops back.  This gives sub-100 ms session pickup.
func makeTerminalNotifyOrbitHandler(svc fleet.Service, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeKey := strings.TrimPrefix(r.Header.Get("Authorization"), "FleetOrbitNodeKey ")
		if nodeKey == "" {
			http.Error(w, "missing node key", http.StatusUnauthorized)
			return
		}
		host, _, err := svc.AuthenticateOrbitHost(r.Context(), nodeKey)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Respond immediately if sessions are already pending.
		if ids := pendingTerminalSessionIDsForHost(host.ID); len(ids) > 0 {
			writeNotifyResponse(w, ids)
			return
		}

		// Subscribe then re-check to avoid the race where a session is created
		// between the check above and the subscribe call.
		ch, unsubscribe := terminalNotifyStore.subscribe(host.ID)
		defer unsubscribe()
		if ids := pendingTerminalSessionIDsForHost(host.ID); len(ids) > 0 {
			writeNotifyResponse(w, ids)
			return
		}

		// Block until notified, timed out, or client disconnects.
		select {
		case <-ch:
		case <-time.After(30 * time.Second):
		case <-r.Context().Done():
			return
		}

		writeNotifyResponse(w, pendingTerminalSessionIDsForHost(host.ID))
	}
}

func writeNotifyResponse(w http.ResponseWriter, ids []string) {
	if ids == nil {
		ids = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		PendingTerminalSessionIDs []string `json:"pending_terminal_session_ids"`
	}{PendingTerminalSessionIDs: ids})
}

func writeTerminalError(conn *gws.Conn, message string) {
	b, _ := json.Marshal(terminalMsg{Type: "error", Data: message})
	_ = conn.WriteMessage(gws.TextMessage, b)
}
