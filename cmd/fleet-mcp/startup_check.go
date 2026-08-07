package main

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// startupCheckTimeout bounds the whole endpoint self-check. Fleet
// reachability was already proven by requireAPIOnlyUser, so a slow or
// wedged Fleet should degrade to a logged warning, not a hung startup.
const startupCheckTimeout = 30 * time.Second

// requiredEndpoint is one Fleet API route the MCP toolset depends on.
type requiredEndpoint struct {
	method string
	// route is the route shape with path params as :name — the form an
	// operator puts in an API-only user's endpoint allowlist, and the form
	// documented in README.md ("Required Fleet API endpoints").
	route string
	// probe is a concrete, side-effect-free instantiation of route. Path
	// params use ID 0 or a sentinel identifier that never exists, so GET
	// probes return 404 and POST probes fail Fleet-side validation before
	// anything is created or executed. Any response other than 403 proves
	// the route is reachable with the configured token.
	probe string
	// body is sent on POST probes. It is intentionally invalid (empty
	// query, nonexistent host) so Fleet rejects the request during
	// validation: POST /reports/run with no query/query_id errors before a
	// campaign or query row is created, and POST /hosts/0/query fails the
	// host lookup before the query is examined.
	body interface{}
}

// requiredEndpoints is the source of truth for which Fleet API routes the
// MCP needs. Deployments that lock the API-only user down with endpoint
// restrictions must allowlist exactly these routes; verifyRequiredEndpoints
// probes each at startup so a stale allowlist is caught at deploy time
// instead of by end users hitting opaque 403s.
//
// GET /api/v1/fleet/results/websocket is also used (live query result
// streaming) but is a raw handler not subject to endpoint restrictions, so
// it is not probed here.
//
// Keep this list, the README section, and the MCP toolset in sync — the
// TestRequiredEndpointsCoverSourcePaths test fails when a Fleet API path
// used by this package is missing from the list.
var requiredEndpoints = []requiredEndpoint{
	// /me can never be observed as blocked here (requireAPIOnlyUser already
	// fatals if it fails), but it stays in the list because it belongs in
	// the operator's allowlist all the same.
	{method: "GET", route: "/api/v1/fleet/me", probe: "/api/v1/fleet/me"},
	{method: "GET", route: "/api/v1/fleet/hosts", probe: "/api/v1/fleet/hosts?per_page=1"},
	{method: "GET", route: "/api/v1/fleet/hosts/count", probe: "/api/v1/fleet/hosts/count"},
	{method: "GET", route: "/api/v1/fleet/hosts/:id", probe: "/api/v1/fleet/hosts/0"},
	{method: "GET", route: "/api/v1/fleet/hosts/:id/software", probe: "/api/v1/fleet/hosts/0/software?per_page=1"},
	{method: "GET", route: "/api/v1/fleet/hosts/identifier/:identifier", probe: "/api/v1/fleet/hosts/identifier/fleet-mcp-startup-probe"},
	{method: "GET", route: "/api/v1/fleet/host_summary", probe: "/api/v1/fleet/host_summary"},
	{method: "GET", route: "/api/v1/fleet/labels", probe: "/api/v1/fleet/labels"},
	{method: "GET", route: "/api/v1/fleet/labels/:id/hosts", probe: "/api/v1/fleet/labels/0/hosts?per_page=1"},
	{method: "GET", route: "/api/v1/fleet/fleets", probe: "/api/v1/fleet/fleets"},
	{method: "GET", route: "/api/v1/fleet/fleets/:id/policies", probe: "/api/v1/fleet/fleets/0/policies"},
	{method: "GET", route: "/api/v1/fleet/fleets/:fleet_id/policies/:policy_id", probe: "/api/v1/fleet/fleets/0/policies/0"},
	{method: "GET", route: "/api/v1/fleet/global/policies", probe: "/api/v1/fleet/global/policies"},
	{method: "GET", route: "/api/v1/fleet/global/policies/:id", probe: "/api/v1/fleet/global/policies/0"},
	{method: "GET", route: "/api/v1/fleet/reports", probe: "/api/v1/fleet/reports?per_page=1"},
	{method: "POST", route: "/api/v1/fleet/reports/run", probe: "/api/v1/fleet/reports/run", body: struct{}{}},
	{method: "GET", route: "/api/v1/fleet/software/titles", probe: "/api/v1/fleet/software/titles?per_page=1"},
	{method: "GET", route: "/api/v1/fleet/software/titles/:id", probe: "/api/v1/fleet/software/titles/0"},
	{method: "POST", route: "/api/v1/fleet/hosts/:id/query", probe: "/api/v1/fleet/hosts/0/query", body: struct{}{}},
}

// verifyEndpointAccess probes every required endpoint with the configured
// token. It returns the "METHOD route" strings that responded 403 (blocked
// by the API-only user's endpoint restrictions or by the token's role) and
// the count of endpoints whose probe failed at the transport level — those
// are unverifiable, not blocked, so the caller must not report an all-clear
// for them.
func verifyEndpointAccess(ctx context.Context, fc *FleetClient) (blocked []string, unverified int) {
	ctx, cancel := context.WithTimeout(ctx, startupCheckTimeout)
	defer cancel()

	for i, e := range requiredEndpoints {
		if ctx.Err() != nil {
			logrus.Warnf("startup self-check: aborted (%v); skipping the remaining %d endpoints", ctx.Err(), len(requiredEndpoints)-i)
			unverified += len(requiredEndpoints) - i
			break
		}
		resp, err := fc.makeFleetRequest(ctx, e.method, e.probe, e.body)
		if err != nil {
			logrus.Warnf("startup self-check: could not probe %s %s: %v", e.method, e.route, err)
			unverified++
			continue
		}
		// Drain a bounded amount so the connection can be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden {
			blocked = append(blocked, e.method+" "+e.route)
			continue
		}
		logrus.Debugf("startup self-check: %s %s reachable (HTTP %d)", e.method, e.route, resp.StatusCode)
	}
	return blocked, unverified
}

// verifyRequiredEndpoints runs the startup endpoint self-check and logs the
// outcome. Non-fatal by design: a 403 usually means the API-only user's
// endpoint allowlist drifted behind the MCP toolset (or the token's role is
// below what a tool needs, e.g. observer vs run_live_query), and the
// remaining tools still work — so surface it loudly at deploy time and keep
// serving. Runs from a goroutine in main; it only logs and touches no shared
// state beyond the concurrency-safe FleetClient.
func verifyRequiredEndpoints(ctx context.Context, fc *FleetClient) {
	blocked, unverified := verifyEndpointAccess(ctx, fc)
	for _, b := range blocked {
		logrus.Warnf("startup self-check: %s returned HTTP 403 — blocked by the API-only user's endpoint restrictions or the token's role; MCP tools that call it will fail", b)
	}
	if len(blocked) > 0 {
		logrus.Warnf("startup self-check: %d of %d required Fleet API endpoints blocked for this token; update the API-only user's endpoint allowlist to match the \"Required Fleet API endpoints\" section of cmd/fleet-mcp/README.md", len(blocked), len(requiredEndpoints))
	}
	if unverified > 0 {
		logrus.Warnf("startup self-check: incomplete — %d of %d required Fleet API endpoints could not be probed (see warnings above); their allowlist status is unknown", unverified, len(requiredEndpoints))
	}
	if len(blocked) == 0 && unverified == 0 {
		logrus.Infof("startup self-check: all %d required Fleet API endpoints are reachable with the configured token", len(requiredEndpoints))
	}
}
