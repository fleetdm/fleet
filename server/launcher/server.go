// Package launcher provides a gRPC server to handle launcher requests.
package launcher

import (
	"net/http"
	"strings"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	launcher "github.com/kolide/launcher/pkg/service"
	grpc "google.golang.org/grpc"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
)

// Handler extends the grpc.Server, providing Handler that allows us to serve
// both gRPC and http traffic.
type Handler struct {
	*grpc.Server
}

// New creates a gRPC server to handle remote requests from launcher.
func New(
	tls fleet.OsqueryService,
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
			ctx := r.Context()
			ctx = kithttp.PopulateRequestContext(ctx, r)
			hgprc.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
