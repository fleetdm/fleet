package agentws

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
)

// HostAuthenticator authenticates the WebSocket upgrade request. It is
// satisfied by fleet.Service.
type HostAuthenticator interface {
	AuthenticateOrbitHost(ctx context.Context, nodeKey string) (*fleet.Host, bool, error)
}

// nodeKeyAuthScheme is the Authorization scheme used by orbit-authenticated
// requests that carry the node key in a header (same scheme as the Android
// agent endpoints).
const nodeKeyAuthScheme = "Node key "

// NewHandler returns the HTTP handler for the agent WebSocket endpoint. The
// upgrade request is authenticated with the orbit node key BEFORE the
// connection is promoted to a WebSocket and any connection state is allocated:
// a failed auth returns 401 immediately.
//
// Note on hosts with an identity certificate (httpsig): AuthenticateOrbitHost
// verifies the HTTP message signature of the current request, a per-request
// model that a persistent socket cannot re-prove per message. The identity is
// therefore verified once, at upgrade time. Agents whose upgrade request is
// not signed fall back to polling (where each request is signed as usual).
func NewHandler(hub *Hub, auth HostAuthenticator, logger *slog.Logger) http.Handler {
	upgrader := websocket.Upgrader{
		// Notifications are tiny; keep per-connection buffers small since an
		// instance may hold tens of thousands of connections.
		ReadBufferSize:  512,
		WriteBufferSize: 512,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		nodeKey, ok := strings.CutPrefix(r.Header.Get("Authorization"), nodeKeyAuthScheme)
		nodeKey = strings.TrimSpace(nodeKey)
		if !ok || nodeKey == "" {
			http.Error(w, "authentication error: missing node key", http.StatusUnauthorized)
			return
		}

		host, _, err := auth.AuthenticateOrbitHost(ctx, nodeKey)
		if err != nil {
			logger.DebugContext(ctx, "agent websocket auth failed", "err", err)
			http.Error(w, "authentication error: invalid node key", http.StatusUnauthorized)
			return
		}

		// Wrap the hijacked connection so raw bytes in/out are counted (see
		// countingConn); the hub reaches the counters via ws.NetConn().
		upgradeWriter := &hijackCountingResponseWriter{ResponseWriter: w}
		ws, err := upgrader.Upgrade(upgradeWriter, r, nil)
		if err != nil {
			// Upgrade already replied to the client.
			logger.DebugContext(ctx, "agent websocket upgrade failed", "host_id", host.ID, "err", err)
			return
		}

		hub.ServeConn(host.ID, host.Hostname, host.Platform, ws)
	})
}
