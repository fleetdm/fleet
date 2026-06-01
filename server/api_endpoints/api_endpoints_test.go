package apiendpoints

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestValidateAPIEndpoints(t *testing.T) {
	originalYAML := apiEndpointsYAML
	t.Cleanup(func() {
		apiEndpointsYAML = originalYAML
	})

	endpoints := []fleet.APIEndpoint{
		fleet.NewAPIEndpointFromTpl("GET", "/api/v1/fleet/hosts"),
		fleet.NewAPIEndpointFromTpl("POST", "/api/v1/fleet/hosts/:id/refetch"),
	}
	apiEndpointsYAML = []byte(`
- method: "GET"
  path: "/api/v1/fleet/hosts"
  display_name: "Route 1"
- method: "POST"
  path: "/api/v1/fleet/hosts/:id/refetch"
  display_name: "Route 2"`)

	routerWithEndpoints := func(endpoints []fleet.APIEndpoint) *mux.Router {
		r := mux.NewRouter()
		for _, e := range endpoints {
			path := strings.Replace(e.Path, "/_version_/", "/{fleetversion:(?:v1|latest)}/", 1)
			r.Handle(path, http.NotFoundHandler()).Methods(e.Method)
		}
		return r
	}

	t.Run("all routes present", func(t *testing.T) {
		err := Init(routerWithEndpoints(endpoints))
		require.NoError(t, err)
	})

	t.Run("missing route returns error", func(t *testing.T) {
		err := Init(routerWithEndpoints(endpoints[:1]))
		require.ErrorContains(t, err, endpoints[1].Method+" "+endpoints[1].Path)
	})

	t.Run("no routes registered returns error listing all missing", func(t *testing.T) {
		err := Init(mux.NewRouter())
		require.ErrorContains(t, err, "the following API endpoints are unknown")
	})

	t.Run("non-mux handler returns error", func(t *testing.T) {
		err := Init(http.NewServeMux())
		require.ErrorContains(t, err, "expected *mux.Router")
	})

	t.Run("empty endpoint list always passes", func(t *testing.T) {
		apiEndpointsYAML = []byte(``)
		err := Init(mux.NewRouter())
		require.NoError(t, err)
	})
}

// catalogBlocklistRule is a rule used by TestCatalogBlocklist to forbid
// catalog entries matching (method, path).
type catalogBlocklistRule struct {
	method string
	path   *regexp.Regexp // matched against the catalog entry's Path; anchor with ^...$
	reason string
}

// catalogBlocklistRules guards against catalog drift: endpoints that would let
// an api_only user with a restrictive allowlist bypass their own restrictions
// must never be added to api_endpoints.yml.
//
// The allowlist is a narrowing layer over RBAC, so endpoints that are simply
// destructive (e.g. DELETE /sessions/:id, DELETE /users/:id) are still gated
// by RBAC and are NOT covered here. The patterns below cover cases where
// adding the endpoint to the catalog would let the holder circumvent the
// allowlist itself:
//
//   - User-creation endpoints that return a session token in the response —
//     a restricted api_only user could mint a clone of themselves with no
//     allowlist and use the returned token to operate without restrictions.
//   - User-modification endpoints that touch the api_endpoints field —
//     direct allowlist broadening (self-modify is blocked, but any future
//     code change that loosens that check would expose this).
//   - Invite creation/modification — same as user creation, mints a new
//     account with a chosen role.
//   - Bulk role-spec apply (POST /users/roles/spec) — has a coarser authz
//     check than the per-user modify path (no per-target ActionWriteRole gate).
//
// If you are intentionally exposing a new endpoint that matches one of these
// patterns, justify the change in a security review and update the rules.
var catalogBlocklistRules = []catalogBlocklistRule{
	{"POST", regexp.MustCompile(`^/api/v1/fleet/users/roles/spec$`), "bulk role-spec apply lacks per-target ActionWriteRole gate"},
	{"POST", regexp.MustCompile(`^/api/v1/fleet/users(?:/.*)?$`), "user creation can return a session token (allowlist bypass via clone)"},
	{"PATCH", regexp.MustCompile(`^/api/v1/fleet/users(?:/.*)?$`), "user modification can change the api_endpoints allowlist"},
	{"POST", regexp.MustCompile(`^/api/v1/fleet/invites(?:/.*)?$`), "invite creation = mint user with chosen role"},
	{"PATCH", regexp.MustCompile(`^/api/v1/fleet/invites(?:/.*)?$`), "invite modification = change role on a pending user"},
}

func findBlocklistViolations(endpoints []fleet.APIEndpoint) []string {
	var msgs []string
	for _, ep := range endpoints {
		for _, r := range catalogBlocklistRules {
			if strings.EqualFold(ep.Method, r.method) && r.path.MatchString(ep.Path) {
				msgs = append(msgs, "  "+ep.Method+" "+ep.Path+" — "+r.reason)
				break
			}
		}
	}
	return msgs
}

func TestCatalogBlocklist(t *testing.T) {
	t.Run("current catalog is clean", func(t *testing.T) {
		loaded, err := loadAPIEndpoints()
		require.NoError(t, err)
		violations := findBlocklistViolations(loaded)
		if len(violations) > 0 {
			t.Fatalf("api_endpoints.yml contains forbidden routes (would let an api_only user bypass their allowlist):\n%s\n\n"+
				"If this is intentional, justify the change in a security review and update catalogBlocklistRules.",
				strings.Join(violations, "\n"))
		}
	})

	t.Run("rules catch each forbidden pattern", func(t *testing.T) {
		// Fault-injection: every forbidden pattern must produce a violation.
		// Failure here means a rule was deleted or its regex no longer matches
		// the example path it was meant to cover.
		examples := []fleet.APIEndpoint{
			{Method: "POST", Path: "/api/v1/fleet/users/roles/spec"},
			{Method: "POST", Path: "/api/v1/fleet/users"},
			{Method: "POST", Path: "/api/v1/fleet/users/admin"},
			{Method: "POST", Path: "/api/v1/fleet/users/api_only"},
			{Method: "PATCH", Path: "/api/v1/fleet/users/:id"},
			{Method: "PATCH", Path: "/api/v1/fleet/users/api_only/:id"},
			{Method: "POST", Path: "/api/v1/fleet/invites"},
			{Method: "PATCH", Path: "/api/v1/fleet/invites/:id"},
		}
		for _, ex := range examples {
			t.Run(ex.Method+" "+ex.Path, func(t *testing.T) {
				violations := findBlocklistViolations([]fleet.APIEndpoint{ex})
				require.Len(t, violations, 1, "expected this endpoint to be blocked but the rules let it through")
			})
		}
	})

	t.Run("rules do not flag legitimate catalog entries", func(t *testing.T) {
		// Sanity check: read-only entries on the same resources stay allowed.
		ok := []fleet.APIEndpoint{
			{Method: "GET", Path: "/api/v1/fleet/users"},
			{Method: "GET", Path: "/api/v1/fleet/users/:id"},
			{Method: "GET", Path: "/api/v1/fleet/sessions/:id"},
			{Method: "DELETE", Path: "/api/v1/fleet/sessions/:id"},
		}
		violations := findBlocklistViolations(ok)
		require.Empty(t, violations, "expected these entries to pass: %v", violations)
	})
}
