package httpsig

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// MiddlewareFunc is a function which receives an http.Handler and returns another http.Handler.
// Typically, the returned handler is a closure which does something with the http.ResponseWriter and http.Request passed
// to it, and then calls the handler passed as parameter to the MiddlewareFunc.
type MiddlewareFunc func(http.Handler) http.Handler

func Middleware(ds fleet.Datastore, logger kitlog.Logger) (MiddlewareFunc, error) {
	// Initialize HTTP signature verifier
	httpSig := NewHTTPSig(ds, logger)
	verifier, err := httpSig.Verifier()
	if err != nil {
		return nil, fmt.Errorf("setup httpsig verifier: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if (strings.Contains(req.URL.Path, "/api/fleet/orbit/") && !strings.HasSuffix(req.URL.Path, "/api/fleet/orbit/ping")) ||
				strings.Contains(req.URL.Path, "/api/v1/osquery/") {
				result, err := verifier.Verify(req)
				if err != nil {
					level.Error(logger).Log("msg", "failed to verify request signature", "path", req.URL.Path, "err", err)
					// http.Error(rw, err.Error(), http.StatusUnauthorized)
					// return
				} else if !result.Verified {
					level.Error(logger).Log("msg", "request not verified", "path", req.URL.Path)
					// http.Error(rw, "request not verified", http.StatusUnauthorized)
					// return
				}
			}
			next.ServeHTTP(w, req)
		})
	}, nil
}
