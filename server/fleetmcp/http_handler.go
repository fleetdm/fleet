package fleetmcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

// maxRequestBodyBytes bounds the body of any incoming MCP/SSE request. JSON-RPC
// payloads are tiny (kilobytes); 1 MiB is a generous ceiling that defeats
// memory-exhaustion attacks via oversized POST bodies.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// limitBodyMiddleware caps r.Body so handlers downstream cannot accidentally
// buffer arbitrarily large payloads from a hostile client.
func limitBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func NewHTTPHandler(mcpServer *server.MCPServer, authToken string, logger *slog.Logger) http.Handler {
	transports := http.NewServeMux()
	// No tool sends server-initiated messages, so neither sessions nor the
	// optional GET stream (405, allowed by the spec) buy anything. mcp-go only
	// frees a session on an explicit client DELETE, and stateless requests can be
	// served by any replica.
	transports.Handle("/mcp", server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true),
		server.WithDisableStreaming(true),
		server.WithLogger(mcpGoLogger{logger: logger}),
	))
	sseServer := server.NewSSEServer(mcpServer)
	transports.Handle("/sse", sseServer)
	transports.Handle("/message", sseServer)

	var handler http.Handler = transports
	handler = bearerAuthMiddleware(authToken, handler)
	handler = mcpRouteGuard(handler)
	handler = limitBodyMiddleware(handler)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.Handle("/", handler)
	return mux
}

// mcp-go's logger interface is printf-style and carries no context.
type mcpGoLogger struct {
	logger *slog.Logger
}

func (l mcpGoLogger) Infof(format string, v ...any) {
	l.logger.InfoContext(context.Background(), fmt.Sprintf(format, v...))
}

func (l mcpGoLogger) Errorf(format string, v ...any) {
	l.logger.ErrorContext(context.Background(), fmt.Sprintf(format, v...))
}
