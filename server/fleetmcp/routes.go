package fleetmcp

import (
	"fmt"
	"net/url"
	"strings"
)

type Route struct {
	Method string
	Path   string
}

// EmbeddedRoutes are the Fleet API routes the tools call, in the API endpoint
// catalog's notation. Inside the Fleet server, an API-only user granted the MCP
// route can reach these routes through the tools without granting each one;
// TestEmbeddedRoutesMatchTools keeps the list in sync with the tools.
var EmbeddedRoutes = []Route{
	{"GET", "/api/v1/fleet/fleets"},
	{"GET", "/api/v1/fleet/fleets/:id/policies"},
	{"GET", "/api/v1/fleet/fleets/:fleet_id/policies/:policy_id"},
	{"GET", "/api/v1/fleet/global/policies"},
	{"GET", "/api/v1/fleet/global/policies/:id"},
	{"GET", "/api/v1/fleet/host_summary"},
	{"GET", "/api/v1/fleet/hosts"},
	{"GET", "/api/v1/fleet/hosts/count"},
	{"GET", "/api/v1/fleet/hosts/:id"},
	{"GET", "/api/v1/fleet/hosts/:id/software"},
	{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
	{"GET", "/api/v1/fleet/labels"},
	{"GET", "/api/v1/fleet/labels/:id/hosts"},
	{"GET", "/api/v1/fleet/reports"},
	{"GET", "/api/v1/fleet/software/titles"},
	{"GET", "/api/v1/fleet/software/titles/:id"},
	{"POST", "/api/v1/fleet/hosts/:id/query"},
	{"POST", "/api/v1/fleet/reports/run"},
}

// pathSegment escapes a tool argument for use as one path segment. Values that
// would still change which route the call reaches are rejected: Fleet's router
// cleans "." and ".." segments, and escaping doesn't stop that.
func pathSegment(name, value string) (string, error) {
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\`) {
		return "", fmt.Errorf("invalid %s %q", name, value)
	}
	return url.PathEscape(value), nil
}

// routeForLog keeps the route shape of a request in debug logs without the
// identifiers callers pass in the path or query string.
func routeForLog(endpoint string) string {
	path, _, _ := strings.Cut(endpoint, "?")
	if prefix := "/api/v1/fleet/hosts/identifier/"; strings.HasPrefix(path, prefix) {
		return prefix + ":identifier"
	}
	return path
}
