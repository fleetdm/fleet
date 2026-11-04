package endpointer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/gorilla/mux"
)

// RemovedPath is a URL path that no longer exists. A request to it with any usual method gets 410 Gone and Message, without authentication,
// because the path is gone for everyone and the response says nothing about the caller's data.
type RemovedPath struct {
	// Path may contain the _version_ placeholder and mux variables, like the paths of a DeprecatedPathAlias.
	Path string
	// Message is the response message for a request that used the given HTTP method.
	Message func(method string) string
}

// removedPathMethods are the methods that get 410. They are listed, not left open, because every route in the API has methods, and
// the route tests (and metrics) rely on that.
var removedPathMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions,
}

// RegisterRemovedPaths registers a 410 Gone handler for each removed path, for every API version. Each request is logged as a warning,
// because it comes from a client that still uses the removed path and the log is the only way to find it.
func RegisterRemovedPaths(r *mux.Router, versions []string, logger *slog.Logger, paths []RemovedPath) {
	versionRegex := strings.Join(append(append([]string{}, versions...), "latest"), "|")
	for _, removed := range paths {
		pathForHandler := strings.Replace(removed.Path, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", versionRegex), 1)
		r.Handle(pathForHandler, removedPathHandler(removed, logger)).Methods(slices.Clone(removedPathMethods)...).
			Name(getNameFromPathAndVerb("any", removed.Path, ""))
	}
}

func removedPathHandler(removed RemovedPath, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.WarnContext(r.Context(), "request to a removed path", "method", r.Method, "uri", r.URL.Path, "removed_path", removed.Path,
			"user_agent", r.UserAgent(), "ip_addr", r.RemoteAddr, "x_for_ip_addr", r.Header.Get("X-Forwarded-For"))
		reason := removed.Message(r.Method)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(JsonError{Message: "Gone", Errors: baseError(reason)})
	})
}
