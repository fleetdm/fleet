package main

import (
	"net/http"
	"strings"
)

// mcpRouteGuard passes MCP protocol paths (/sse, /message) through to the
// wrapped handler and returns a short plain-text response for everything else.
// This prevents web crawlers and health-check probes from hitting the auth
// middleware or SSE server and generating noisy log entries.
func mcpRouteGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimRight(r.URL.Path, "/")
		switch p {
		case "/sse", "/message":
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("fleet-mcp: use an MCP client to connect\n"))
		}
	})
}
