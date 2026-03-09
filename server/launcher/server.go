// Package launcher provides a gRPC server to handle launcher requests.
package launcher

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/kitlogadapter"
	kithttp "github.com/go-kit/kit/transport/http"
	launcher "github.com/kolide/launcher/pkg/service"
	grpc "google.golang.org/grpc"
)

// Handler extends the grpc.Server, providing Handler that allows us to serve
// both gRPC and http traffic.
type Handler struct {
	*grpc.Server
}

// New creates a gRPC server to handle remote requests from launcher.
func New(
	tls fleet.OsqueryService,
	logger *slog.Logger,
	grpcServer *grpc.Server,
	healthCheckers map[string]health.Checker,
) *Handler {
	kitLogger := kitlogadapter.NewLogger(logger)
	var svc launcher.KolideService
	{
		svc = &launcherWrapper{
			tls:            tls,
			logger:         logger,
			healthCheckers: healthCheckers,
		}
		svc = launcher.LoggingMiddleware(kitLogger)(svc)
	}
	endpoints := launcher.MakeServerEndpoints(svc)
	server := launcher.NewGRPCServer(endpoints, kitLogger)
	launcher.RegisterGRPCServer(grpcServer, server)
	return &Handler{grpcServer}
}

// Handler will route gRPC traffic to the gRPC server, other http traffic
// will be routed to normal http handler functions.
func (hgrpc *Handler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			ctx := r.Context()
			ctx = kithttp.PopulateRequestContext(ctx, r)
			hgrpc.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
