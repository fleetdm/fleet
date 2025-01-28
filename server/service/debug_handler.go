package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/pprof"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"

	kithttp "github.com/go-kit/kit/transport/http"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
		v, err := authViewer(ctx, string(bearer), m.service)
		if err != nil {
			http.Error(w, "Invalid authentication", http.StatusUnauthorized)
			return
		}

		if !v.CanPerformActions() {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func jsonHandler(
	logger kitlog.Logger,
	jsonGenerator func(ctx context.Context) (interface{}, error),
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
			level.Error(logger).Log("err", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Write(b) //nolint:errcheck
	}
}

// MakeDebugHandler creates an HTTP handler for the Fleet debug endpoints.
func MakeDebugHandler(svc fleet.Service, config config.FleetConfig, logger kitlog.Logger, eh *errorstore.Handler, ds fleet.Datastore) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.Handle("/debug/errors", eh)
	r.PathPrefix("/debug/pprof/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) { pprof.Index(rw, req) })
	r.HandleFunc("/debug/migrations", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.MigrationStatus(ctx) }))
	// TODO: Add handler for feature migrations
	r.HandleFunc("/debug/db/locks", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.DBLocks(ctx) }))
	r.HandleFunc("/debug/db/innodb-status", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.InnoDBStatus(ctx) }))
	r.HandleFunc("/debug/db/process-list", jsonHandler(logger, func(ctx context.Context) (interface{}, error) { return ds.ProcessList(ctx) }))

	mw := &debugAuthenticationMiddleware{
		service: svc,
	}
	r.Use(mw.Middleware)

	return r
}
