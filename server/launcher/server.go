// Package launcher provides a gRPC server to handle launcher requests.
package launcher

import (
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	launcher "github.com/kolide/launcher/service"
	grpc "google.golang.org/grpc"

	"github.com/kolide/fleet/server/health"
	"github.com/kolide/fleet/server/kolide"
)

// Handler extends the grpc.Server, providing Handler that allows us to serve
// both gRPC and http traffic.
type Handler struct {
	*grpc.Server
}

// New creates a gRPC server to handle remote requests from launcher.
func New(
	tls kolide.OsqueryService,
	logger log.Logger,
	grpcServer *grpc.Server,
	healthCheckers map[string]health.Checker,
) *Handler {
	var svc launcher.KolideService
	{
		svc = &launcherWrapper{
			tls:            tls,
			logger:         logger,
			healthCheckers: healthCheckers,
		}
		svc = launcher.LoggingMiddleware(logger)(svc)
	}
	endpoints := launcher.MakeServerEndpoints(svc)
	server := launcher.NewGRPCServer(endpoints, logger)
	launcher.RegisterGRPCServer(grpcServer, server)
	return &Handler{grpcServer}
}

// Handler will route gRPC traffic to the gRPC server, other http traffic
// will be routed to normal http handler functions.
func (hgprc *Handler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			hgprc.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
