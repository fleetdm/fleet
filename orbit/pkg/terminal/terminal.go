// Package terminal connects the orbit agent to a Fleet server web terminal
// session.  When the Fleet server signals a pending terminal session via
// OrbitConfigNotifications.PendingTerminalSessionIDs, orbit calls Connect,
// which:
//
//  1. Dials back to the Fleet server over a WebSocket.
//  2. Spawns a local shell (bash on Linux/macOS, PowerShell on Windows) with
//     stdin/stdout/stderr piped through the WebSocket.
//  3. Relays keyboard input from the browser and PTY output back to the
//     browser until either side closes.
//
// Message protocol (JSON text frames):
//
//	{"type":"input",  "data":"<base64>"}          – browser keystrokes → shell stdin
//	{"type":"output", "data":"<base64>"}          – shell stdout/stderr → browser
//	{"type":"resize", "cols":<n>, "rows":<n>}     – browser resize event (best-effort)
package terminal

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	gws "github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// msg is the JSON envelope exchanged over the WebSocket.
type msg struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"` // base64-encoded for "input" / "output"
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// Connect dials the Fleet server terminal endpoint for sessionID, spawns a
// local shell, and proxies data between the two until the context is cancelled
// or either side closes the connection.
//
// fleetURL is the base URL of the Fleet server (e.g. "https://fleet.corp").
// nodeKey is the orbit node key used to authenticate the connection.
func Connect(ctx context.Context, fleetURL, sessionID, nodeKey string, tlsInsecure bool) error {
	wsURL := buildWSURL(fleetURL, sessionID)

	headers := http.Header{}
	headers.Set("Authorization", "FleetOrbitNodeKey "+nodeKey)

	log.Info().Str("session_id", sessionID).Str("url", wsURL).Msg("terminal: dialling Fleet server")

	dialer := gws.DefaultDialer
	if tlsInsecure {
		dialer = &gws.Dialer{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return fmt.Errorf("terminal: dial Fleet server: %w", err)
	}
	defer conn.Close()

	shell, err := startShell(ctx)
	if err != nil {
		return fmt.Errorf("terminal: start shell: %w", err)
	}

	// closeShell is safe to call from multiple goroutines; the Once ensures the
	// shell is closed exactly once.  Closing the shell's PTY file descriptor
	// unblocks any pending shell.read() call, so the PTY goroutine exits and
	// wg.Wait() can return even if the server/browser side disconnected first.
	var closeOnce sync.Once
	closeShell := func() { closeOnce.Do(shell.close) }
	defer closeShell()

	log.Info().Str("session_id", sessionID).Msg("terminal: shell started, relaying")

	var wg sync.WaitGroup

	// shell stdout/stderr → WebSocket (output messages).
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := shell.read(buf)
			if n > 0 {
				m := msg{
					Type: "output",
					Data: base64.StdEncoding.EncodeToString(buf[:n]),
				}
				b, _ := json.Marshal(m)
				if writeErr := conn.WriteMessage(gws.TextMessage, b); writeErr != nil {
					log.Debug().Err(writeErr).Msg("terminal: write to Fleet WebSocket failed")
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					log.Debug().Err(readErr).Msg("terminal: shell read error")
				}
				// Shell exited — send a close frame so the Fleet server propagates
				// the close to the browser, which then closes the tab.
				_ = conn.WriteMessage(gws.CloseMessage,
					gws.FormatCloseMessage(gws.CloseNormalClosure, "shell exited"))
				conn.Close() // unblock the ReadMessage goroutine below
				return
			}
		}
	}()

	// WebSocket → shell stdin (input / resize messages).
	wg.Add(1)
	go func() {
		defer wg.Done()
		// When the server closes the session (browser disconnect, auth failure,
		// TTL expiry), ReadMessage returns an error.  Closing the shell here
		// unblocks the PTY read goroutine above, allowing wg.Wait() to return
		// and the deferred conn.Close() / shell cleanup to run promptly.
		defer closeShell()
		for {
			_, raw, readErr := conn.ReadMessage()
			if readErr != nil {
				log.Debug().Err(readErr).Msg("terminal: read from Fleet WebSocket failed")
				return
			}
			var m msg
			if err := json.Unmarshal(raw, &m); err != nil {
				continue
			}
			switch m.Type {
			case "input":
				data, err := base64.StdEncoding.DecodeString(m.Data)
				if err != nil {
					continue
				}
				if _, err := shell.write(data); err != nil {
					log.Debug().Err(err).Msg("terminal: shell write error")
					return
				}
			case "resize":
				shell.resize(m.Cols, m.Rows) //nolint:errcheck
			}
		}
	}()

	wg.Wait()
	log.Info().Str("session_id", sessionID).Msg("terminal: session ended")
	return nil
}

// buildWSURL converts a Fleet http(s) base URL into a ws(s) terminal URL.
func buildWSURL(fleetURL, sessionID string) string {
	u := strings.TrimRight(fleetURL, "/")
	u = strings.Replace(u, "https://", "wss://", 1)
	u = strings.Replace(u, "http://", "ws://", 1)
	return u + "/api/fleet/orbit/terminal/" + sessionID
}
