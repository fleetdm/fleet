package service

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/contexts/token"
	"github.com/fleetdm/fleet/server/kolide"
	kitlog "github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
)

// MakeDebugHandler creates an HTTP handler for the Fleet debug endpoints.
func MakeDebugHandler(svc kolide.Service, config config.KolideConfig, logger kitlog.Logger) http.Handler {
	// kolideAPIOptions := []kithttp.ServerOption{
	// 	kithttp.ServerBefore(
	// 		kithttp.PopulateRequestContext, // populate the request context with common fields
	// 		setRequestsContexts(svc, config.Auth.JwtKey),
	// 	),
	// 	kithttp.ServerErrorLogger(logger),
	// 	kithttp.ServerErrorEncoder(encodeError),
	// }

	r := mux.NewRouter()
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.PathPrefix("/debug/pprof/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		token := token.FromHTTPRequest(req)
		fmt.Fprintf(os.Stderr, "%v -- %+v\n", token, req)
		pprof.Index(rw, req)
	})

	return r
}
