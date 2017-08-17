// Package launcher provides a gRPC server to handle launcher requests.
package launcher

import (
	"net/http"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	pb "github.com/kolide/agent-api"
	"github.com/kolide/fleet/server/kolide"
	grpc "google.golang.org/grpc"
)

// Handler extends the grpc.Server, providing Handler that allows us to serve
// both gRPC and http traffic.
type Handler struct {
	*grpc.Server
}

// New creates a gRPC server to handler remote requests from launcher.
func New(svc kolide.OsqueryService, logger kitlog.Logger, opts ...grpc.ServerOption) *Handler {
	binding := newAgentBinding(svc)
	binding = newAuthMiddleware(svc)(binding)
	binding = newLoggingMiddleware(logger)(binding)

	server := grpc.NewServer(opts...)
	pb.RegisterApiServer(server, binding)
	return &Handler{server}
}

// Handler will route gRPC traffic to the gRPC server, other http traffic
// will be routed to normal http handler functions.
func (hgprc *Handler) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			hgprc.ServeHTTP(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
