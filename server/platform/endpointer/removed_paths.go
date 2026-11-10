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

// RemovedPath is a URL path that no longer exists. A request to it with any usual method (or one of Methods, when set) gets 410 Gone and
// Message, without authentication, because the path is gone for everyone and the response says nothing about the caller's data.
type RemovedPath struct {
	// Path may contain the _version_ placeholder and mux variables, like the paths of a DeprecatedPathAlias.
	Path string
	// Methods limits the 410 to these methods, for a path that shares its shape with routes that are still served: a 410 on every method
	// would answer wrong-method requests to those routes, which should get 405. Empty means every method in removedPathMethods.
	Methods []string
	// Overlap is for a Path that a stdlib ServeMux cannot register next to a live route.
	Overlap RemovedPathOverlap
	// Message is the response message for a request that used the given HTTP method.
	Message func(method string) string
}

// RemovedPathOverlap registers RouteTemplate instead of the removed path for the one method that overlaps a live route. RouteTemplate
// writes the overlapping literal segment as a wildcard, which makes it less specific than the live route, and a request without the
// literals of the removed path gets a 404. The other methods keep the removed path, so a wrong method on the live route still gets a 405.
type RemovedPathOverlap struct {
	// Method and RouteTemplate are the wildcard route registered in place of the removed path.
	Method        string
	RouteTemplate string
	// WrongMethodRoutes are live routes on another method that the wildcard also catches, mapped to the methods they serve. A request to
	// one gets a 405 with those methods in its Allow header.
	WrongMethodRoutes map[string][]string
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
		versioned := func(path string) string {
			return strings.Replace(path, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", versionRegex), 1)
		}
		handler := removedPathHandler(removed, logger)
		methods := slices.Clone(removedPathMethods)
		if len(removed.Methods) > 0 {
			methods = slices.Clone(removed.Methods)
		}
		// The overlap only applies to a method that was removed.
		if overlap := removed.Overlap; overlap.RouteTemplate != "" && slices.Contains(methods, overlap.Method) {
			methods = slices.DeleteFunc(methods, func(method string) bool { return method == overlap.Method })
			// Register a more generic path template so it doesn't conflict with the still supported path template
			r.Handle(versioned(overlap.RouteTemplate), handler).Methods(overlap.Method).
				Name(getNameFromPathAndVerb(overlap.Method, removed.Path, ""))
		}
		if len(methods) == 0 {
			continue // the overlap route took the only removed method
		}
		verb := "any"
		if len(removed.Methods) > 0 {
			verb = strings.Join(methods, "_")
		}
		r.Handle(versioned(removed.Path), handler).Methods(methods...).Name(getNameFromPathAndVerb(verb, removed.Path, ""))
	}
}

func removedPathHandler(removed RemovedPath, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if removed.Overlap.RouteTemplate != "" && !literalSegmentsMatch(removed.Path, r.URL.Path) {
			for live, allow := range removed.Overlap.WrongMethodRoutes {
				if literalSegmentsMatch(live, r.URL.Path) {
					w.Header().Set("Allow", strings.Join(allow, ", "))
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
			}
			http.NotFound(w, r)
			return
		}
		logger.WarnContext(r.Context(), "request to a removed path", "method", r.Method, "uri", r.URL.Path, "removed_path", removed.Path,
			"user_agent", r.UserAgent(), "ip_addr", r.RemoteAddr, "x_for_ip_addr", r.Header.Get("X-Forwarded-For"))
		reason := removed.Message(r.Method)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(JsonError{Message: "Gone", Errors: baseError(reason)})
	})
}

// literalSegmentsMatch reports whether requestPath ends with the literal segments of template. They are aligned from the end because the
// request path can also have a URL prefix.
func literalSegmentsMatch(template, requestPath string) bool {
	templateSegments := strings.Split(strings.TrimPrefix(template, "/"), "/")
	requestSegments := strings.Split(requestPath, "/")
	if len(requestSegments) < len(templateSegments) {
		return false
	}
	requestSegments = requestSegments[len(requestSegments)-len(templateSegments):]
	for i, segment := range templateSegments {
		if segment != "_version_" && !strings.HasPrefix(segment, "{") && segment != requestSegments[i] {
			return false
		}
	}
	return true
}
