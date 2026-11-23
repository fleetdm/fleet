package http

import "net/http"

// PathVar returns the named path variable of the route that matched r.
func PathVar(r *http.Request, name string) (string, bool) {
	v := r.PathValue(name)
	return v, v != ""
}
