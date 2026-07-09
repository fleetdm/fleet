package update

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/terminal"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// TerminalConfigReceiver implements fleet.OrbitConfigReceiver.  When the Fleet
// server includes pending terminal session IDs in the config notifications, it
// dials back to the server for each session and spawns a local shell.
//
// It is exported (capital T) so that orbit.go can also call StartFastPoll on
// the concrete type.
type TerminalConfigReceiver struct {
	fleetURL    string
	nodeKey     func() (string, error)
	tlsInsecure bool
	httpClient  *http.Client

	mu              sync.Mutex
	activeSessions  map[string]struct{} // session IDs currently being handled
	terminalEnabled bool                // updated by Run(); gates the long-poll loop
}

// ApplyTerminalConfigReceiverMiddleware creates a receiver that handles
// PendingTerminalSessionIDs notifications from the Fleet server.
//
// nodeKeyFn is called each time a new session is to be established; it should
// return the current orbit node key.
func ApplyTerminalConfigReceiverMiddleware(
	fleetURL string,
	nodeKeyFn func() (string, error),
	tlsInsecure bool,
) *TerminalConfigReceiver {
	//nolint:gosec
	tlsCfg := &tls.Config{InsecureSkipVerify: tlsInsecure}
	return &TerminalConfigReceiver{
		fleetURL:    fleetURL,
		nodeKey:     nodeKeyFn,
		tlsInsecure: tlsInsecure,
		// No client-level Timeout — per-request context (35 s) handles cancellation.
		httpClient:     fleethttp.NewClient(fleethttp.WithTLSClientConfig(tlsCfg)),
		activeSessions: make(map[string]struct{}),
	}
}

// Run implements fleet.OrbitConfigReceiver.  Called by the orbit config poll
// cycle (typically every 30 s).  For faster pickup, StartFastPoll runs an
// independent long-poll loop.
func (t *TerminalConfigReceiver) Run(cfg *fleet.OrbitConfig) error {
	t.mu.Lock()
	// Always update the enabled flag so longPollOnce learns about license
	// changes without waiting for the next config poll.
	t.terminalEnabled = cfg.Notifications.TerminalEnabled

	if !cfg.Notifications.TerminalEnabled || len(cfg.Notifications.PendingTerminalSessionIDs) == 0 {
		t.mu.Unlock()
		return nil
	}
	defer t.mu.Unlock()

	for _, sessionID := range cfg.Notifications.PendingTerminalSessionIDs {
		if _, already := t.activeSessions[sessionID]; already {
			continue // already handling this session
		}

		nodeKey, err := t.nodeKey()
		if err != nil {
			log.Error().Err(err).Str("session_id", sessionID).
				Msg("terminal: failed to get orbit node key, skipping session")
			continue
		}

		t.activeSessions[sessionID] = struct{}{}
		go func(id, key string) {
			defer func() {
				t.mu.Lock()
				delete(t.activeSessions, id)
				t.mu.Unlock()
			}()

			log.Info().Str("session_id", id).Msg("terminal: starting session")
			if err := terminal.Connect(context.Background(), t.fleetURL, id, key, t.tlsInsecure); err != nil {
				log.Error().Err(err).Str("session_id", id).Msg("terminal: session failed")
			}
		}(sessionID, nodeKey)
	}
	return nil
}

// StartFastPoll runs a long-poll loop against /api/fleet/orbit/terminal/notify.
// The server blocks each request until a terminal session is created for this
// host (or 30 s elapses), so orbit picks up new sessions in sub-100 ms.
// Call it as a goroutine after ctx is available:
//
//	go terminalReceiver.StartFastPoll(ctx)
func (t *TerminalConfigReceiver) StartFastPoll(ctx context.Context) {
	for ctx.Err() == nil {
		started := time.Now()
		t.longPollOnce(ctx)

		// A healthy server holds this request for up to 30 seconds. Keep a
		// minimum interval if an older server or proxy returns immediately.
		if wait := time.Second - time.Since(started); wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
			}
		}
	}
}

// longPollOnce calls the notify endpoint once.  It blocks until the server
// responds (≤30 s), then processes any returned session IDs.
func (t *TerminalConfigReceiver) longPollOnce(ctx context.Context) {
	// If the server has not signalled that terminal is available (e.g. Free
	// tier), skip the HTTP round-trip entirely and sleep until the next
	// regular config poll cycle may flip the flag.
	t.mu.Lock()
	enabled := t.terminalEnabled
	t.mu.Unlock()
	if !enabled {
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
		}
		return
	}

	nodeKey, err := t.nodeKey()
	if err != nil {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
		}
		return
	}

	// Use a slightly longer timeout than the server's 30 s so we don't race.
	reqCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	u := strings.TrimRight(t.fleetURL, "/") + "/api/fleet/orbit/terminal/notify"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "FleetOrbitNodeKey "+nodeKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
			}
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
		}
		return
	}

	var result struct {
		PendingTerminalSessionIDs []string `json:"pending_terminal_session_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	if len(result.PendingTerminalSessionIDs) == 0 {
		return // timed out on server side — loop immediately
	}

	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.TerminalEnabled = true // already in long-poll loop → terminal is enabled
	cfg.Notifications.PendingTerminalSessionIDs = result.PendingTerminalSessionIDs
	_ = t.Run(cfg) //nolint:errcheck
}
