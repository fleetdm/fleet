package fleetmcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Route struct {
	Method string
	Path   string
}

func (r Route) String() string { return r.Method + " " + r.Path }

// ToolRoutes are the Fleet API routes each tool can call across all of its
// arguments, in the API endpoint catalog's notation. Inside the Fleet server, an
// API-only user with an endpoint allowlist only gets the tools whose routes it
// was granted. TestToolRoutesMatchTools keeps the table in sync with the
// tools.
var ToolRoutes = map[string][]Route{
	"get_aggregate_platforms": {{"GET", "/api/v1/fleet/host_summary"}},
	"get_endpoints": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/count"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/labels"},
		{"GET", "/api/v1/fleet/labels/:id/hosts"},
	},
	"get_fleets": {{"GET", "/api/v1/fleet/fleets"}},
	"get_host": {
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
	},
	"get_host_policies": {
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
	},
	"get_host_users": {
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
	},
	"get_labels":         {{"GET", "/api/v1/fleet/labels"}},
	"get_osquery_schema": {},
	"get_policies": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/fleets/:id/policies"},
		{"GET", "/api/v1/fleet/global/policies"},
	},
	"get_policy_compliance": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/fleets/:fleet_id/policies/:policy_id"},
		{"GET", "/api/v1/fleet/global/policies/:id"},
	},
	"get_policy_hosts": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/labels"},
		{"GET", "/api/v1/fleet/labels/:id/hosts"},
	},
	"get_queries": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/reports"},
	},
	"get_software": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/hosts/:id/software"},
		{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
		{"GET", "/api/v1/fleet/software/titles"},
	},
	"get_total_system_count": {{"GET", "/api/v1/fleet/hosts/count"}},
	"get_vetted_queries":     {},
	"get_vulnerability_hosts": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/software/titles"},
		{"GET", "/api/v1/fleet/software/titles/:id"},
	},
	"get_vulnerability_impact": {
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/count"},
		{"GET", "/api/v1/fleet/software/titles"},
		{"GET", "/api/v1/fleet/software/titles/:id"},
	},
	"prepare_live_query": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
		{"GET", "/api/v1/fleet/labels"},
		{"GET", "/api/v1/fleet/labels/:id/hosts"},
		{"GET", "/api/v1/fleet/software/titles"},
		{"GET", "/api/v1/fleet/software/titles/:id"},
	},
	"run_live_query": {
		{"GET", "/api/v1/fleet/fleets"},
		{"GET", "/api/v1/fleet/hosts"},
		{"GET", "/api/v1/fleet/hosts/:id"},
		{"GET", "/api/v1/fleet/hosts/identifier/:identifier"},
		{"GET", "/api/v1/fleet/labels"},
		{"GET", "/api/v1/fleet/labels/:id/hosts"},
		{"GET", "/api/v1/fleet/software/titles"},
		{"GET", "/api/v1/fleet/software/titles/:id"},
		{"POST", "/api/v1/fleet/hosts/:id/query"},
		{"POST", "/api/v1/fleet/reports/run"},
	},
}

type routeFilterKey struct{}

// ContextWithRouteFilter limits the tools for a request to those whose routes
// all pass allowed.
func ContextWithRouteFilter(ctx context.Context, allowed func(Route) bool) context.Context {
	return context.WithValue(ctx, routeFilterKey{}, allowed)
}

// toolAllowed reports whether ctx's filter lets tool run, and if not, which of
// its routes the filter rejects.
func toolAllowed(ctx context.Context, tool string) (bool, []Route) {
	allowed, _ := ctx.Value(routeFilterKey{}).(func(Route) bool)
	if allowed == nil {
		return true, nil
	}
	routes, ok := ToolRoutes[tool]
	if !ok {
		return false, nil
	}
	var missing []Route
	for _, r := range routes {
		if !allowed(r) {
			missing = append(missing, r)
		}
	}
	return len(missing) == 0, missing
}

func routeFilterOptions(fc *FleetClient) []server.ServerOption {
	return []server.ServerOption{
		server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
			visible := make([]mcp.Tool, 0, len(tools))
			for _, t := range tools {
				if ok, _ := toolAllowed(ctx, t.Name); ok {
					visible = append(visible, t)
				}
			}
			return visible
		}),
		// Clients may call a tool they didn't get from tools/list.
		server.WithToolHandlerMiddleware(func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
			return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				ok, missing := toolAllowed(ctx, req.Params.Name)
				if ok {
					return next(ctx, req)
				}
				names := make([]string, 0, len(missing))
				for _, r := range missing {
					names = append(names, r.String())
				}
				fc.logger.InfoContext(ctx, "tool not allowed for this API-only user", "tool", req.Params.Name, "missing_endpoints", names)
				if len(missing) == 0 {
					return mcp.NewToolResultError(fmt.Sprintf("%s isn't available to API-only users with endpoint restrictions.", req.Params.Name)), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("%s needs Fleet API endpoints this API-only user isn't allowed to call: %s. An admin can add them to the user's API endpoints.",
					req.Params.Name, strings.Join(names, ", "))), nil
			}
		}),
	}
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
