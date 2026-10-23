package service

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/fleetmcp"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	kithttp "github.com/go-kit/kit/transport/http"
)

// The embedded MCP's client never resolves this host: it dials MCPConn.
const mcpInternalBaseURL = "http://fleet.internal"

// MCPConn is the in-memory connection the embedded MCP uses to call this
// server's own API, so every tool call goes through the same router, auth and
// authorization as an external API call, without a network hop.
type MCPConn struct {
	listener *pipeListener
	srv      *http.Server
	started  atomic.Bool
}

func NewMCPConn() *MCPConn {
	return &MCPConn{listener: newPipeListener()}
}

// Start serves handler to the embedded MCP. baseCtx should be the context the
// public server derives requests from (it carries the license).
func (c *MCPConn) Start(baseCtx context.Context, handler http.Handler) {
	c.srv = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
	}
	c.started.Store(true)
	go func() { _ = c.srv.Serve(c.listener) }()
}

// Shutdown must run after the public server has drained, since in-flight MCP
// requests depend on this connection.
func (c *MCPConn) Shutdown(ctx context.Context) error {
	if !c.started.Load() {
		return nil
	}
	return c.srv.Shutdown(ctx)
}

func (c *MCPConn) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	// Test handlers built without Start would otherwise block until the
	// caller's context ends.
	if !c.started.Load() {
		return nil, errors.New("embedded MCP is not connected to the Fleet API")
	}
	return c.listener.dial(ctx)
}

type pipeListener struct {
	conns     chan net.Conn
	done      chan struct{}
	closeOnce sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{conns: make(chan net.Conn), done: make(chan struct{})}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.closeOnce.Do(func() { close(l.done) })
	return nil
}

func (l *pipeListener) Addr() net.Addr { return pipeAddr{} }

func (l *pipeListener) dial(ctx context.Context) (net.Conn, error) {
	server, client := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-ctx.Done():
	case <-l.done:
	}
	_ = server.Close()
	_ = client.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, net.ErrClosed
}

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "embedded-mcp" }

// mcpUserLogHandler adds the caller to fleetmcp's log lines, which carry the
// tool call's context but know nothing about Fleet users.
type mcpUserLogHandler struct{ slog.Handler }

func (h mcpUserLogHandler) Handle(ctx context.Context, r slog.Record) error {
	if emailer, ok := logging.UserEmailerFromContext(ctx); ok {
		r.AddAttrs(slog.String("user", emailer.Email()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h mcpUserLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return mcpUserLogHandler{h.Handler.WithAttrs(attrs)}
}

func (h mcpUserLogHandler) WithGroup(name string) slog.Handler {
	return mcpUserLogHandler{h.Handler.WithGroup(name)}
}

func newEmbeddedFleetClient(conn *MCPConn, urlPrefix string, logger *slog.Logger) (*fleetmcp.FleetClient, error) {
	ctx := context.Background()
	// Both live query paths wait the period this server's own live query
	// endpoints use, which operators already keep under their proxies' timeouts.
	period, err := LiveQueryRestPeriod()
	if err != nil {
		logger.WarnContext(ctx, "invalid FLEET_LIVE_QUERY_REST_PERIOD, the embedded MCP uses the default", "err", err)
	}
	// The internal listener serves the same root handler as the public one,
	// which only answers under the URL prefix.
	return fleetmcp.NewFleetClient(ctx, mcpInternalBaseURL+urlPrefix, "", false, "", logger,
		fleetmcp.WithDialContext(conn.DialContext),
		fleetmcp.WithLiveQueryWaits(period, period),
	)
}

// grantedMCPRoutes returns which tool routes an API-only user's endpoint
// allowlist grants, matched the way APIOnlyEndpointCheck matches each call the
// tools make, or nil when the user has no allowlist.
func grantedMCPRoutes(u *fleet.User) func(fleetmcp.Route) bool {
	if !u.APIOnly || len(u.APIEndpoints) == 0 {
		return nil
	}
	granted := make(map[string]struct{}, len(u.APIEndpoints))
	for _, ep := range u.APIEndpoints {
		granted[fleet.NewAPIEndpointFromTpl(ep.Method, ep.Path).Fingerprint()] = struct{}{}
	}
	return func(r fleetmcp.Route) bool {
		_, ok := granted[fleet.NewAPIEndpointFromTpl(r.Method, r.Path).Fingerprint()]
		return ok
	}
}

// sameOrigin reports whether a browser Origin header names the Fleet server
// itself, comparing scheme, host and port.
func sameOrigin(origin, serverURL string) bool {
	o, err := url.Parse(origin)
	if err != nil || o.Host == "" {
		return false
	}
	s, err := url.Parse(serverURL)
	if err != nil || s.Host == "" {
		return false
	}
	port := func(u *url.URL) string {
		if p := u.Port(); p != "" {
			return p
		}
		if strings.EqualFold(u.Scheme, "https") {
			return "443"
		}
		return "80"
	}
	return strings.EqualFold(o.Scheme, s.Scheme) && strings.EqualFold(o.Hostname(), s.Hostname()) && port(o) == port(s)
}

func newMCPHandler(svc fleet.Service, logger *slog.Logger, conn *MCPConn, urlPrefix string) (http.Handler, error) {
	toolLogger := slog.New(mcpUserLogHandler{logger.Handler()})
	fleetClient, err := newEmbeddedFleetClient(conn, urlPrefix, toolLogger)
	if err != nil {
		return nil, err
	}
	mcpHandler := fleetmcp.NewEmbeddedHTTPHandler(fleetmcp.NewEmbeddedMCPServer(fleetClient), toolLogger)
	noop := func(context.Context, any) (any, error) { return nil, nil }

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reject := func(level slog.Level, reason string, err error, attrs ...any) {
			logger.Log(ctx, level, "rejected embedded MCP request", append([]any{"reason", reason}, attrs...)...)
			encodeError(ctx, err, w)
		}

		// Unauthenticated rejections are anyone's noise, so they stay at debug.
		bearer := token.FromHTTPRequest(r)
		if bearer == "" {
			reject(slog.LevelDebug, "no auth token", fleet.NewAuthHeaderRequiredError("no auth token"))
			return
		}
		v, err := auth.AuthViewer(ctx, string(bearer), svc)
		if err != nil {
			reject(slog.LevelDebug, "invalid auth token", err)
			return
		}
		user := []any{"user", v.Email()}
		if !v.User.APIOnly {
			reject(slog.LevelInfo, "not an API-only user", fleet.NewPermissionError("Fleet's MCP server is only available to API-only users"), user...)
			return
		}
		ctx = viewer.NewContext(ctx, *v)
		ctx = logging.WithUserEmailer(ctx, v)
		// Raw handlers skip the API-only endpoint allowlist, so run it here for
		// this route, together with the checks that follow authentication.
		probeCtx := kithttp.PopulateRequestContext(auth.RouteTemplateRequestFunc(ctx, r), r)
		if _, err := auth.AuthenticatedUser(svc, auth.APIOnlyEndpointCheck(noop))(probeCtx, nil); err != nil {
			reject(slog.LevelInfo, "route not allowed for this user", err, user...)
			return
		}
		appConfig, err := svc.AppConfigObfuscated(ctx)
		if err != nil {
			encodeError(ctx, err, w)
			return
		}
		// Browsers always send Origin and MCP clients outside a browser don't;
		// a page on another origin must not drive this endpoint (DNS rebinding).
		if origin := r.Header.Get("Origin"); origin != "" && !sameOrigin(origin, appConfig.ServerSettings.ServerURL) {
			reject(slog.LevelInfo, "Origin doesn't match server_settings.server_url", fleet.NewPermissionError("Origin not allowed"), append(user, "origin", origin[:min(len(origin), 256)])...)
			return
		}
		if !license.IsPremium(ctx) {
			reject(slog.LevelInfo, "Fleet Premium required", fleet.ErrMissingLicense, user...)
			return
		}
		if appConfig.ServerSettings.MCPDisabled {
			reject(slog.LevelInfo, "server_settings.mcp_disabled is on", &fleet.BadRequestError{Message: "Fleet's MCP server is disabled (server_settings.mcp_disabled)"}, user...)
			return
		}
		if granted := grantedMCPRoutes(v.User); granted != nil {
			ctx = fleetmcp.ContextWithRouteFilter(ctx, granted)
		}
		mcpHandler.ServeHTTP(w, r.WithContext(fleetmcp.ContextWithFleetToken(ctx, string(bearer))))
	}), nil
}
