package service

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/fleet"

	kitlog "github.com/go-kit/kit/log"
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

// MakeDebugHandler creates an HTTP handler for the Fleet debug endpoints.
func MakeDebugHandler(svc fleet.Service, config config.FleetConfig, logger kitlog.Logger) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.PathPrefix("/debug/pprof/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		pprof.Index(rw, req)
	})

	mw := &debugAuthenticationMiddleware{
		service: svc,
	}
	r.Use(mw.Middleware)

	return r
}
