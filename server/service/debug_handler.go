package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/fleetdm/fleet/v4/server/agentws"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

type debugAuthenticationMiddleware struct {
	service fleet.Service
}

// Authenticate the user and ensure the account is not disabled.
func (m *debugAuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := token.FromHTTPRequest(r)
		if bearer == "" {
			http.Error(w, "Please authenticate", http.StatusUnauthorized)
			return
		}
		ctx := token.NewContext(context.Background(), bearer)
		v, err := auth.AuthViewer(ctx, string(bearer), m.service)
		if err != nil {
			http.Error(w, "Invalid authentication", http.StatusUnauthorized)
			return
		}

		if !v.CanPerformActions() || v.User.GlobalRole == nil || *v.User.GlobalRole != fleet.RoleAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Debug routes are not part of the public API catalog, so they can never appear in an
		// API-only user's endpoint allowlist. A restricted API-only token (api_only with a
		// non-empty api_endpoints list) must therefore be denied here, matching the least-privilege
		// scoping that APIOnlyEndpointCheck enforces on the main API path.
		if v.User.APIOnly && len(v.User.APIEndpoints) > 0 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Attach the authenticated viewer to the request context so downstream debug handlers can record who triggered an
		// action (e.g. updating trace sampler settings).
		next.ServeHTTP(w, r.WithContext(viewer.NewContext(r.Context(), *v)))
	})
}

func jsonHandler(
	logger *slog.Logger,
	jsonGenerator func(ctx context.Context) (any, error),
) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		lc := &logging.LoggingContext{SkipUser: true} // The debug handler does not save the logged-in user.
		ctx := logging.NewContext(kithttp.PopulateRequestContext(r.Context(), r), lc)
		ctx = logging.WithStartTime(ctx)
		jsonData, err := jsonGenerator(ctx)
		if err != nil {
			lc.SetErrs(err)
			lc.Log(ctx, logger)
			var sce kithttp.StatusCoder
			if errors.As(err, &sce) {
				rw.WriteHeader(sce.StatusCode())
				_, _ = rw.Write([]byte(err.Error()))
				return
			}
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		b, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			lc.SetErrs(err)
			lc.Log(ctx, logger)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Write(b) //nolint:errcheck
	}
}

// MakeDebugHandler creates an HTTP handler for the Fleet debug endpoints.
func MakeDebugHandler(svc fleet.Service, config config.FleetConfig, logger *slog.Logger, eh *errorstore.Handler, ds fleet.Datastore, agentWSHub *agentws.Hub) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.Handle("/debug/errors", eh)
	r.PathPrefix("/debug/pprof/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { pprof.Index(rw, req) })
	r.HandleFunc("/debug/migrations", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.MigrationStatus(ctx) }))
	r.HandleFunc("/debug/db/locks", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.DBLocks(ctx) }))
	r.HandleFunc("/debug/db/innodb-status", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.InnoDBStatus(ctx) }))
	r.HandleFunc("/debug/db/process-list", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.ProcessList(ctx) }))
	r.HandleFunc("/debug/trace_sampler", jsonHandler(logger, func(ctx context.Context) (any, error) {
		return ds.GetTraceSamplerSettings(ctx)
	})).Methods(http.MethodGet)
	r.HandleFunc("/debug/trace_sampler", patchTraceSamplerHandler(logger, ds)).Methods(http.MethodPatch)
	// Agent WebSocket transport observability: a point-in-time view of the
	// connections held by THIS server instance; instance_id lets consumers
	// behind a load balancer tell instances apart. Read-path counters are
	// collected only with debug logging enabled (see metrics_enabled).
	r.HandleFunc("/debug/agentws", jsonHandler(logger, func(ctx context.Context) (any, error) {
		if agentWSHub == nil {
			return map[string]any{"enabled": false}, nil
		}
		connections := agentWSHub.Snapshot()
		readStats := agentWSHub.ReadStats()
		payload := map[string]any{
			"enabled":         true,
			"instance_id":     agentWSHub.InstanceID,
			"metrics_enabled": config.Logging.Debug,
			"connections":     connections,
			"read_stats":      readStats,
		}
		// Interval check job timing, for the dashboard's "next sync" countdown.
		// Remaining time is computed server-side so client clock skew doesn't
		// matter; zero until the job records its first tick.
		if next := agentWSHub.NextCheck(); !next.IsZero() {
			payload["check_interval_seconds"] = config.WebSocket.CheckInterval.Seconds()
			payload["next_check_in_ms"] = max(0, time.Until(next).Milliseconds())
		}

		return payload, nil
	})).Methods(http.MethodGet)

	mw := &debugAuthenticationMiddleware{
		service: svc,
	}
	r.Use(mw.Middleware)

	return r
}
