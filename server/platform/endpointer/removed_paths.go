package endpointer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// RemovedPath is a URL path that no longer exists. A request to it with any method gets 410 Gone and Message, without authentication,
// because the path is gone for everyone and the response says nothing about the caller's data.
type RemovedPath struct {
	// Path may contain the _version_ placeholder and mux variables, like the paths of a DeprecatedPathAlias.
	Path string
	// Message is the response message for a request that used the given API version (for example "v1" or "latest").
	Message func(version string) string
}

// RegisterRemovedPaths registers a 410 Gone handler for each removed path, for every API version.
func RegisterRemovedPaths(r *mux.Router, versions []string, paths []RemovedPath) {
	versionRegex := strings.Join(append(append([]string{}, versions...), "latest"), "|")
	for _, removed := range paths {
		pathForHandler := strings.Replace(removed.Path, "/_version_/", fmt.Sprintf("/{fleetversion:(?:%s)}/", versionRegex), 1)
		r.Handle(pathForHandler, removedPathHandler(removed)).Name(getNameFromPathAndVerb("any", removed.Path, ""))
	}
}

func removedPathHandler(removed RemovedPath) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reason := removed.Message(mux.Vars(r)["fleetversion"])
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(JsonError{Message: "Gone", Errors: baseError(reason)})
	})
}
