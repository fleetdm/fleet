package scim

import (
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// CAVEAT — keep this in sync with scim.go. The route list below mirrors
// two things that cannot be introspected from the github.com/elimity-com/scim
// library (its Server fields are unexported and routing is a hardcoded switch in
// ServeHTTP): (1) the resource Endpoints registered in RegisterSCIM, and (2) the
// library's discovery endpoints and per-resource method matrix. Adding or
// renaming a SCIM resource type in scim.go, or a library upgrade that changes
// its routing, requires a matching edit here. The coupling is by convention, not
// enforced by the compiler — but it is enforced at runtime: apiendpoints.Validate
// fails whenever a catalog SCIM endpoint isn't covered here, so drift cannot
// ship silently.

// scimRootPath is the path prefix the SCIM handler is mounted under (see
// RegisterSCIM). The /_version_/ placeholder is expanded to the concrete API
// version when comparing against the api_endpoints catalog.
const scimRootPath = "/api/_version_/fleet/scim"

// SCIM resource endpoints, matching the Endpoint of each resource type
// registered in RegisterSCIM. Keep these in sync with that list.
const (
	usersEndpoint  = "/Users"
	groupsEndpoint = "/Groups"
)

// servedRoute is a (method, path-template) pair served by the SCIM handler.
type servedRoute struct {
	method string
	tpl    string
}

// servedRoutes returns the routes served by the prefix-mounted SCIM handler.
// The github.com/elimity-com/scim library routes these internally in
// (scim.Server).ServeHTTP, so they never reach gorilla/mux and cannot be
// discovered by walking the router. We reconstruct them from the resource
// endpoints the server is configured with plus the discovery endpoints the
// library hardcodes per RFC 7644. Keep this in sync with RegisterSCIM's
// resource types and the library's ServeHTTP switch.
func servedRoutes() []servedRoute {
	var routes []servedRoute
	add := func(method, tpl string) {
		routes = append(routes, servedRoute{method: method, tpl: tpl})
	}

	// Discovery endpoints (fixed by the SCIM library / RFC 7644).
	add(http.MethodGet, scimRootPath+"/Schemas")
	add(http.MethodGet, scimRootPath+"/ServiceProviderConfig")
	add(http.MethodGet, scimRootPath+"/ResourceTypes")

	// CRUD endpoints, one set per registered resource type.
	for _, endpoint := range []string{usersEndpoint, groupsEndpoint} {
		base := scimRootPath + endpoint
		add(http.MethodGet, base)
		add(http.MethodPost, base)
		add(http.MethodGet, base+"/{id}")
		add(http.MethodPut, base+"/{id}")
		add(http.MethodPatch, base+"/{id}")
		add(http.MethodDelete, base+"/{id}")
	}
	return routes
}

// RegisterValidationRoutes registers stub routes for every endpoint served by
// the prefix-mounted SCIM handler (see RegisterSCIM) onto r. It exists so
// apiendpoints.Validate can confirm the api_endpoints catalog stays in sync
// with what SCIM actually serves; the handlers are never invoked, only their
// path templates and methods are inspected.
func RegisterValidationRoutes(r *mux.Router, _ []kithttp.ServerOption) {
	for _, rt := range servedRoutes() {
		r.Handle(rt.tpl, http.NotFoundHandler()).Methods(rt.method)
	}
}
