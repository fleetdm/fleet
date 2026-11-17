package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

// PathVar returns the named path variable of the route that matched r. The stdlib ServeMux fast path sets it as a path value and
// the gorilla/mux router sets it in mux.Vars, so callers that read it through PathVar work under both.
func PathVar(r *http.Request, name string) (string, bool) {
	if v := r.PathValue(name); v != "" {
		return v, true
	}
	v, ok := mux.Vars(r)[name]
	return v, ok
}
