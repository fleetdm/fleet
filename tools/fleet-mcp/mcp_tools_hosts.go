package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// registerHostTools attaches host- and inventory-domain MCP tools to s.
// Tools registered: get_endpoints, get_host, get_total_system_count,
// get_aggregate_platforms, get_fleets, get_labels.
//
// All tools in this group are read-only against the Fleet API, idempotent,
// and non-destructive. They are annotated as such so MCP clients (e.g.
// Claude Desktop) do not gate them behind destructive-action review.
func registerHostTools(s *server.MCPServer, fleetClient *FleetClient) {
	registerGetEndpoints(s, fleetClient)
	registerGetHost(s, fleetClient)
	registerGetHostPolicies(s, fleetClient)
	registerGetTotalSystemCount(s, fleetClient)
	registerGetAggregatePlatforms(s, fleetClient)
	registerGetFleets(s, fleetClient)
	registerGetLabels(s, fleetClient)
}

func registerGetEndpoints(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_endpoints",
		mcp.WithDescription("Get a list of hosts/endpoints enrolled in Fleet with full server-side filtering. All filters compose: combine fleet+platform+label+policy_id+policy_response+status+query in one call to narrow precisely instead of paginating client-side. The `query` parameter alone covers user / IP / hostname / serial / hardware model / IdP group as a case-insensitive substring — reach for it before paginating. Use get_host for full details on one host, get_host_policies for one host's compliance, get_policy_hosts for hosts grouped by policy result. Do NOT call this tool repeatedly with per_page=1 just to count — use get_total_system_count instead."),
		mcp.WithString("fleet", mcp.Description("Optional fleet name to filter by (e.g. 'Workstations')")),
		mcp.WithString("platform", mcp.Description("Optional platform to filter by (e.g. 'macos', 'windows', 'linux')")),
		mcp.WithString("status", mcp.Description("Optional host status filter (e.g. 'online', 'offline', 'new', 'mia')")),
		mcp.WithString("query", mcp.Description("Optional substring (case-insensitive) matched against hostname, hardware serial, primary IP, hardware model, AND user inventory (username / email / IdP group). Best way to narrow results when you have a partial identifier such as a person's name, email, or IP fragment.")),
		mcp.WithString("label", mcp.Description("Optional Fleet label name (e.g. 'macOS', 'engineering'). Resolved to a label_id server-side. Use get_labels to discover names.")),
		mcp.WithString("policy_id", mcp.Description("Optional numeric policy ID (from get_policies) to scope to hosts that have a result on that policy.")),
		mcp.WithString("policy_response", mcp.Description("Optional 'passing' or 'failing'. Requires policy_id — narrows to hosts that pass / fail that specific policy.")),
		mcp.WithString("per_page", mcp.Description("Max number of hosts to return (default 50, max 200)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_endpoints")

		fleet := getOptionalString(request, "fleet")
		platform := getOptionalString(request, "platform")
		status := getOptionalString(request, "status")
		query := getOptionalString(request, "query")
		label := getOptionalString(request, "label")
		policyID := getOptionalString(request, "policy_id")
		policyResponse := getOptionalString(request, "policy_response")

		// Validate at the MCP layer so the AI client gets a clear message
		// rather than a cryptic Fleet 400.
		if policyResponse != "" && policyID == "" {
			return mcp.NewToolResultError("policy_response is only valid when policy_id is also set"), nil
		}
		if policyResponse != "" && policyResponse != "passing" && policyResponse != "failing" {
			return mcp.NewToolResultError(fmt.Sprintf("policy_response must be 'passing' or 'failing', got %q", policyResponse)), nil
		}

		perPage := parsePerPageArg(request, defaultEndpointsPerPage)

		anyFilter := fleet != "" || platform != "" || status != "" || query != "" || label != "" || policyID != "" || policyResponse != ""

		var endpoints []Endpoint
		var err error
		if anyFilter {
			endpoints, err = fleetClient.GetEndpointsWithFilters(ctx, fleet, platform, status, query, label, policyID, policyResponse, perPage)
		} else {
			endpoints, err = fleetClient.GetEndpoints(ctx, perPage)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get endpoints: %v", err)), nil
		}

		// Total must reflect the *filter scope*, not the global inventory.
		// Otherwise the LLM trusts a 133-host global count as "Workstations
		// total" even when the returned slice is correctly team-scoped.
		var totalCount int
		if anyFilter {
			totalCount, err = fleetClient.GetHostCountWithFilters(ctx, fleet, platform, status, query, label, policyID, policyResponse)
		} else {
			totalCount, err = fleetClient.GetHostCount(ctx)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get host count: %v", err)), nil
		}

		return jsonResult(struct {
			Total     int        `json:"total"`
			Returned  int        `json:"returned"`
			Endpoints []Endpoint `json:"endpoints"`
		}{
			Total:     totalCount,
			Returned:  len(endpoints),
			Endpoints: endpoints,
		})
	})
}

func registerGetHost(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_host",
		mcp.WithDescription("Get full details for a single host including its labels, fleet, and platform info. Accepts a numeric `host_id` (most precise), or an `identifier` (exact hostname / UUID / hardware serial, OR a substring to fuzzy-match). If the substring matches exactly one host, full details are returned. If multiple match — for example two hosts share a hostname — a candidate list is returned with each host's id, hostname, display_name, hardware_serial, primary_ip, and team so you can pick the right one and re-call with `host_id`.\n\nIMPORTANT: substring matching covers hostname / serial / IP / model / user inventory but NOT display_name. If the host you want has only a custom display_name (a user-set computer name that does not appear in any indexed string field), use `host_id` from a candidate list. Use get_endpoints when you need many hosts; use get_host_policies when you need a host's policy compliance."),
		mcp.WithString("host_id", mcp.Description("Numeric Fleet host ID (e.g. '1309'). Unambiguous. Use whenever you have it — preferred over identifier when collisions are possible.")),
		mcp.WithString("identifier", mcp.Description("Optional. Exact hostname / UUID / serial OR a fuzzy substring (e.g. 'Dhruv' → 'Dhruvs-MacBook-Pro.local'). Required if host_id is not set. Does NOT match display_name.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_host")

		hostIDArg := getOptionalString(request, "host_id")
		identifier := getOptionalString(request, "identifier")

		if hostIDArg == "" && identifier == "" {
			return mcp.NewToolResultError("either host_id or identifier is required"), nil
		}

		// Case 1: explicit numeric host_id wins. Always exact.
		if hostIDArg != "" {
			id, parseErr := strconv.ParseUint(hostIDArg, 10, 64)
			if parseErr != nil || id == 0 || id > uint64(^uint(0)) {
				return mcp.NewToolResultError(fmt.Sprintf("host_id must be a positive integer, got %q", hostIDArg)), nil
			}
			host, err := fleetClient.GetHostByID(ctx, uint(id))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get host by id: %v", err)), nil
			}
			return jsonResult(host)
		}

		// Case 2: identifier path — query-first to detect collisions before
		// falling back to /hosts/identifier/:id (which silently picks one
		// when multiple hosts share a hostname). Cap at 50: Fleet's substring
		// matcher is permissive so we need room for collisions to surface.
		const maxCandidates = 50
		candidates, qErr := fleetClient.GetEndpointsWithFilters(ctx, "", "", "", identifier, "", "", "", maxCandidates)

		if qErr == nil && len(candidates) == 1 {
			// Single unambiguous match — fetch by ID for guaranteed
			// no-collision lookup.
			full, fErr := fleetClient.GetHostByID(ctx, candidates[0].ID)
			if fErr == nil {
				return jsonResult(full)
			}
			// API hiccup on the ID lookup — degrade to the candidate hit.
			return jsonResult(candidates[0])
		}
		if qErr == nil && len(candidates) > 1 {
			return jsonResult(map[string]interface{}{
				"message":    fmt.Sprintf("%d hosts match %q. Note that the substring search does NOT cover display_name; pick the `id` from the candidates below and re-call with `host_id` set.", len(candidates), identifier),
				"candidates": candidates,
			})
		}

		// Zero query matches OR query failed: fall back to identifier endpoint
		// (catches UUIDs and other identifiers Fleet's substring index misses).
		host, err := fleetClient.GetHostByIdentifier(ctx, identifier)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Host not found by query or identifier: %s (substring search does NOT cover display_name — try host_id if you have it)", identifier)), nil
		}
		return jsonResult(host)
	})
}

func registerGetTotalSystemCount(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_total_system_count",
		mcp.WithDescription("Get the total count of active systems enrolled in Fleet"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_total_system_count")
		count, err := fleetClient.GetHostCount(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get total count: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Total Enrolled Systems: %d", count)), nil
	})
}

func registerGetAggregatePlatforms(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_aggregate_platforms",
		mcp.WithDescription("Get the count of systems aggregated by platform (macOS, Windows, Linux)"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_aggregate_platforms")
		aggregate, err := fleetClient.GetEndpointsWithAggregations(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get aggregate platforms: %v", err)), nil
		}

		dataMap, ok := aggregate.Data.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Unexpected data format returned from Fleet"), nil
		}

		platformBreakdown, ok := dataMap["platform_breakdown"]
		if !ok {
			return mcp.NewToolResultError("platform_breakdown key missing from Fleet response"), nil
		}

		return jsonResult(platformBreakdown)
	})
}

func registerGetFleets(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_fleets",
		mcp.WithDescription("Get all fleets with their IDs and names. Use this to discover the exact fleet names before filtering by fleet in other tools."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_fleets")
		fleets, err := fleetClient.GetTeams(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get fleets: %v", err)), nil
		}
		return jsonResult(fleets)
	})
}

func registerGetLabels(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_labels",
		mcp.WithDescription("Get a list of all labels in Fleet"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_labels")
		labels, err := fleetClient.GetLabels(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get labels: %v", err)), nil
		}
		return jsonResult(labels)
	})
}

func registerGetHostPolicies(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_host_policies",
		mcp.WithDescription("Get the compliance status of every policy applied to a single host (global + fleet-inherited). Each policy entry has a `response` field: 'pass', 'fail', or '' (not yet run). The response also includes a summary block with `failing_count`, `passing_count`, `not_run_count`, and `total` so you can answer 'is this host compliant?' directly. Use this — NOT get_host — whenever the question is 'which policies is this host failing?', 'is host X compliant?', or 'what policies apply to this host?'. get_host returns base host info without policy data; this tool wraps a single API call (populate_policies=true) that includes them. Optionally filter by `response` ('passing'|'failing') to narrow the returned list.\n\nIDENTIFIER GUIDANCE: pass `host_id` (numeric) when known — it is unambiguous. `identifier` accepts an exact hostname / UUID / serial OR a substring; substring matching covers hostname / serial / primary IP / hardware model / user inventory but NOT display_name. When multiple hosts match the substring (e.g. shared hostname), this tool returns a candidate list with each host's id, hostname, display_name, serial, primary_ip, and team — re-call with `host_id` from the candidate you want."),
		mcp.WithString("host_id", mcp.Description("Numeric Fleet host ID (e.g. '1309'). When set, takes precedence over identifier and bypasses substring matching. Use this whenever you have a concrete ID from a candidate list or prior call.")),
		mcp.WithString("identifier", mcp.Description("Optional. Exact hostname / UUID / hardware serial, OR a substring matched against hostname / serial / IP / model / user inventory. Required if host_id is not set. Note: does NOT match display_name — use host_id for display-name-only hosts.")),
		mcp.WithString("response", mcp.Description("Optional filter: 'passing' (only `response=='pass'` entries) or 'failing' (only `response=='fail'` entries). Defaults to all.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_host_policies")

		hostIDArg := getOptionalString(request, "host_id")
		identifier := getOptionalString(request, "identifier")
		responseFilter := getOptionalString(request, "response")

		if hostIDArg == "" && identifier == "" {
			return mcp.NewToolResultError("either host_id or identifier is required"), nil
		}
		if responseFilter != "" && responseFilter != "passing" && responseFilter != "failing" {
			return mcp.NewToolResultError(fmt.Sprintf("response must be 'passing' or 'failing', got %q", responseFilter)), nil
		}

		host, ambiguous, candidates, err := resolveHostWithPolicies(ctx, fleetClient, hostIDArg, identifier)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get host policies: %v", err)), nil
		}
		if ambiguous {
			return jsonResult(map[string]interface{}{
				"message":    fmt.Sprintf("%d hosts match %q. Note that the substring search does NOT cover display_name; if the host you want shares a hostname with others, pick its `id` from the candidates below and re-call this tool with `host_id` set.", len(candidates), identifier),
				"candidates": candidates,
			})
		}

		// Compute summary counts — this mirrors the "This device is failing N
		// policies" headline shown in the Fleet UI per-host Policies tab and
		// gives the AI client a direct compliance answer without scanning.
		var failing, passing, notRun int
		for _, p := range host.Policies {
			switch p.Response {
			case "fail":
				failing++
			case "pass":
				passing++
			default:
				notRun++
			}
		}

		// Apply optional response filter to the policies list.
		filtered := host.Policies
		if responseFilter != "" {
			want := "pass"
			if responseFilter == "failing" {
				want = "fail"
			}
			filtered = make([]HostPolicyEntry, 0, len(host.Policies))
			for _, p := range host.Policies {
				if p.Response == want {
					filtered = append(filtered, p)
				}
			}
		}

		return jsonResult(struct {
			Host     Endpoint          `json:"host"`
			Summary  map[string]int    `json:"summary"`
			Policies []HostPolicyEntry `json:"policies"`
		}{
			Host: host.Endpoint,
			Summary: map[string]int{
				"failing_count": failing,
				"passing_count": passing,
				"not_run_count": notRun,
				"total":         len(host.Policies),
			},
			Policies: filtered,
		})
	})
}

// resolveHostWithPolicies turns a (host_id, identifier) pair into a single
// authoritative host with populated policies, OR a candidate list when the
// identifier is ambiguous.
//
// Resolution order:
//  1. host_id set (numeric) → /hosts/:host_id?populate_policies=true. Exact.
//  2. identifier non-numeric → query-first: /hosts?query=identifier
//     - 0 matches → fall back to /hosts/identifier/:id (catches UUIDs, which
//     Fleet's substring search doesn't index).
//     - 1 match → fetch the resolved host by ID (ID-path is the only way to
//     guarantee no silent collision when hostnames are duplicated).
//     - 2+ matches → return ambiguous=true with candidates so the caller can
//     re-call with host_id.
//
// Reasoning: Fleet's /hosts/identifier/:id endpoint silently returns ONE
// host when multiple share the same hostname — giving callers the wrong
// host with no warning. Going through the query endpoint first surfaces
// collisions, then the explicit /hosts/:id resolves the chosen one with
// no further ambiguity.
func resolveHostWithPolicies(ctx context.Context, fleetClient *FleetClient, hostIDArg, identifier string) (host *HostWithPolicies, ambiguous bool, candidates []Endpoint, err error) {
	// Case 1: explicit numeric host_id wins.
	if hostIDArg != "" {
		id, parseErr := strconv.ParseUint(hostIDArg, 10, strconv.IntSize)
		if parseErr != nil || id == 0 {
			return nil, false, nil, fmt.Errorf("host_id must be a positive integer, got %q", hostIDArg)
		}
		h, hErr := fleetClient.GetHostByIDWithPolicies(ctx, uint(id))
		if hErr != nil {
			return nil, false, nil, hErr
		}
		return h, false, nil, nil
	}

	// Case 2: identifier path — query first to detect collisions.
	// Cap at 50 candidates: Fleet's substring matcher is permissive (e.g.
	// "mac" hits hundreds of hosts) so we need headroom for true collisions
	// to surface. 50 keeps the disambiguation list bounded for the AI client.
	const maxCandidates = 50
	cands, qErr := fleetClient.GetEndpointsWithFilters(ctx, "", "", "", identifier, "", "", "", maxCandidates)

	if qErr == nil && len(cands) == 1 {
		// One unambiguous match. Fetch by ID for guaranteed no-collision and
		// to populate policies (the substring search doesn't return them).
		h, hErr := fleetClient.GetHostByIDWithPolicies(ctx, cands[0].ID)
		if hErr != nil {
			// API hiccup — the search did find the host but the ID lookup
			// failed. Return error rather than guess.
			return nil, false, nil, hErr
		}
		return h, false, nil, nil
	}
	if qErr == nil && len(cands) > 1 {
		// Multiple hosts match the substring — caller must disambiguate.
		return nil, true, cands, nil
	}

	// Zero query matches OR query failed: fall back to the identifier
	// endpoint for UUIDs and other identifiers Fleet's substring index
	// doesn't reach.
	h, idErr := fleetClient.GetHostByIdentifierWithPolicies(ctx, identifier)
	if idErr != nil {
		return nil, false, nil, fmt.Errorf("host not found by query or identifier: %s (substring search does NOT cover display_name — try host_id if you have it)", identifier)
	}
	return h, false, nil, nil
}
