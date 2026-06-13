package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// registerQueryTools attaches query- and schema-domain MCP tools to s.
// Tools registered: get_queries, create_saved_query, get_vetted_queries,
// prepare_live_query, run_live_query, get_osquery_schema, refresh_osquery_schema.
//
// Annotation policy in this group:
//   - get_queries: read-only, idempotent, openWorld (Fleet API).
//   - create_saved_query: NOT read-only, NOT idempotent (creates new resource);
//     destructiveHint stays false because creating a saved query does not
//     mutate or remove existing data.
//   - get_vetted_queries / get_osquery_schema / prepare_live_query: read-only,
//     idempotent, openWorldHint=false (consults the in-memory canonical schema,
//     refreshed periodically by the background loop in schema.go).
//   - refresh_osquery_schema: read-only on Fleet (no API call), but openWorld=true
//     because it talks to raw.githubusercontent.com. Not idempotent in the sense
//     that the upstream JSON can change between calls.
//   - run_live_query: read-only on devices (osquery SELECT only), but NOT
//     idempotent because each invocation spawns a new live distribution.
func registerQueryTools(s *server.MCPServer, fleetClient *FleetClient) {
	registerGetQueries(s, fleetClient)
	registerCreateSavedQuery(s, fleetClient)
	registerGetVettedQueries(s)
	registerPrepareLiveQuery(s, fleetClient)
	registerRunLiveQuery(s, fleetClient)
	registerGetOsquerySchema(s)
	registerRefreshOsquerySchema(s)
}

func registerGetQueries(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_queries",
		mcp.WithDescription("Get a list of all saved queries in Fleet"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_queries")
		queries, err := fleetClient.GetQueries(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get queries: %v", err)), nil
		}
		return jsonResult(queries)
	})
}

func registerCreateSavedQuery(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("create_saved_query",
		mcp.WithDescription("Create a new saved query in Fleet. MUST call get_osquery_schema(platform=...) (or get_osquery_schema(tables=...) for tables outside the curated list) BEFORE writing the sql argument — column types and enum values must match the canonical schema. Assumed types (e.g. assuming windows_update_history.result_code is integer when it is text 'Succeeded'/'Failed') are the #1 cause of silent zero-row queries.\n\nTeam scoping: when `fleet` is provided, the query is created under that team (Fleet) — it appears under that team in the Fleet UI, inherits its RBAC, and is listed by per-team enumeration. Omit `fleet` only when you explicitly want the query at the Global scope. If the user mentioned a team in the conversation (e.g. 'Workstations'), pass it as `fleet`."),
		mcp.WithString("name", mcp.Required(), mcp.Description("The name of the query")),
		mcp.WithString("sql", mcp.Required(), mcp.Description("The OSQuery SQL statement")),
		mcp.WithString("description", mcp.Description("Description of what the query does")),
		mcp.WithString("platform", mcp.Description("Target platform (e.g., 'darwin,windows,linux'). Leave empty for all.")),
		mcp.WithString("fleet", mcp.Description("Fleet (team) name to scope the query to, e.g. 'Workstations'. When set, the query is created under that team — visible only in that team's query list and inheriting its RBAC. Leave empty for Global scope.")),
		// Writes new state to Fleet (a saved query that can later be scheduled / run
		// across every device). Treat as destructive so MCP clients (Claude Desktop)
		// surface explicit user approval rather than auto-approving as "safe."
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: create_saved_query")

		name, err := request.RequireString("name")
		if err != nil || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		sql, err := request.RequireString("sql")
		if err != nil || sql == "" {
			return mcp.NewToolResultError("sql is required"), nil
		}

		desc := getOptionalString(request, "description")
		platform := getOptionalString(request, "platform")
		fleet := strings.TrimSpace(getOptionalString(request, "fleet"))

		// Pre-flight: validate SQL table compatibility for the declared platform
		if platform != "" {
			platformTargets := strings.Split(platform, ",")
			for i, pt := range platformTargets {
				platformTargets[i] = strings.TrimSpace(pt)
			}
			if valErr := ValidateSQLForPlatforms(sql, platformTargets); valErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("SQL platform validation failed: %v", valErr)), nil
			}
		}

		// Resolve fleet name → team_id so the query is created under the
		// requested team rather than Global. Empty fleet stays nil = Global.
		var teamID *uint
		if fleet != "" {
			ids, terr := fleetClient.resolveTeamNames(ctx, []string{fleet})
			if terr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve fleet %q: %v", fleet, terr)), nil
			}
			if len(ids) == 0 {
				return mcp.NewToolResultError(fmt.Sprintf("Fleet %q resolved to no team IDs", fleet)), nil
			}
			id := ids[0]
			teamID = &id
		}

		query, err := fleetClient.CreateSavedQuery(ctx, name, desc, sql, platform, teamID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create saved query: %v", err)), nil
		}
		return jsonResult(query)
	})
}

func registerGetVettedQueries(s *server.MCPServer) {
	tool := mcp.NewTool("get_vetted_queries",
		mcp.WithDescription("Get the library of 100% vetted, production-safe CIS-8.1 policy queries for macOS, Windows, and Linux. Always use these as a reference or starting point for creating new policies — they have been tested and use the correct table schemas for each platform."),
		mcp.WithString("platform", mcp.Description("Filter by platform: 'darwin' or 'macos' for macOS, 'windows' for Windows, 'linux' for Linux, 'all' for everything. Defaults to 'all'.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_vetted_queries")

		platform := getOptionalString(request, "platform")
		if platform == "" {
			platform = "all"
		}

		queries := GetVettedQueries(platform)
		if len(queries) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("No vetted queries found for platform: %s", platform)), nil
		}
		return jsonResult(queries)
	})
}

func registerPrepareLiveQuery(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("prepare_live_query",
		mcp.WithDescription("Step 1 of 2 for running a live query. RESOLVES THE EXACT TARGET HOST SET using the same intersection semantics as get_endpoints — every dimension you set is AND-ed: fleet AND platform/label AND status AND query AND policy AND cve. Returns (a) the resolved target list (id, hostname, display_name, platform, team) so you can verify scope before firing, and (b) the OSQuery schema for the targeted platform. Explicit hostnames / host_ids combine with filter dimensions as an intersection — 'these named hosts that ALSO match the filters'.\n\nUse this — NOT a wide live-query — to pinpoint exactly what's in scope. Example: fleet='Workstations' + platform='linux' resolves to ONLY the Linux Workstations hosts (e.g. 2 hosts), not all 100 Workstations hosts. Example: cve_id='CVE-2025-12345' + fleet='Workstations' resolves to the host(s) actually impacted by that CVE in the team."),
		mcp.WithString("fleet", mcp.Description("Fleet (team) name, e.g. 'Workstations'")),
		mcp.WithString("platform", mcp.Description("Platform: 'macos' / 'windows' / 'linux' / 'chromeos'. Resolved server-side via the matching built-in label.")),
		mcp.WithString("label", mcp.Description("Custom Fleet label name. Takes precedence over platform when both set.")),
		mcp.WithString("status", mcp.Description("Host status filter: 'online' / 'offline' / 'new' / 'mia'.")),
		mcp.WithString("query", mcp.Description("Substring matched against hostname / serial / IP / model / user inventory.")),
		mcp.WithString("policy_id", mcp.Description("Numeric policy ID. Combine with policy_response to scope to hosts that pass/fail it.")),
		mcp.WithString("policy_response", mcp.Description("'passing' or 'failing'. Requires policy_id.")),
		mcp.WithString("cve_id", mcp.Description("CVE ID, e.g. 'CVE-2025-12345'. Resolves to hosts running affected software versions.")),
		mcp.WithString("host_ids", mcp.Description("Optional comma-separated numeric host IDs to target explicitly (unambiguous; use this to disambiguate hostname collisions).")),
		mcp.WithString("hostnames", mcp.Description("Optional comma-separated hostnames. Falls back to display name / computer name match. Multiple matches return an error — use host_ids to disambiguate.")),
		mcp.WithString("labels", mcp.Description("LEGACY — comma-separated label names (only first item used; prefer 'label').")),
		mcp.WithString("platforms", mcp.Description("LEGACY — comma-separated platforms (only first item used; prefer 'platform').")),
		mcp.WithString("fleets", mcp.Description("LEGACY — comma-separated fleet names (only first item used; prefer 'fleet').")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: prepare_live_query")

		spec, err := buildLiveQuerySpecFromRequest(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		targets, err := fleetClient.ResolveLiveQueryTargets(ctx, spec)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Target resolution failed: %v", err)), nil
		}
		if len(targets) == 0 {
			return mcp.NewToolResultError("Targets resolved to 0 hosts — refine your filters."), nil
		}

		// Decide schema context: platform (or first legacy platform) wins;
		// otherwise infer from the targets if they're homogeneous.
		schemaPlatform := strings.TrimSpace(spec.Platform)
		if schemaPlatform == "" && len(spec.LegacyPlatforms) == 1 {
			schemaPlatform = spec.LegacyPlatforms[0]
		}
		if schemaPlatform == "" {
			schemaPlatform = inferPlatformFromTargets(targets)
		}

		schema, err := GetOsquerySchema(schemaPlatform)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get schema for context: %v", err)), nil
		}

		// Build a compact target preview — full list capped at 100 so the
		// response stays AI-context-friendly. Always report the full count.
		const previewCap = 100
		preview := targets
		truncated := false
		if len(preview) > previewCap {
			preview = preview[:previewCap]
			truncated = true
		}
		previewItems := make([]map[string]interface{}, 0, len(preview))
		for _, h := range preview {
			previewItems = append(previewItems, map[string]interface{}{
				"id":           h.ID,
				"hostname":     h.Name,
				"display_name": h.DisplayName,
				"platform":     h.Platform,
				"team_name":    h.TeamName,
				"status":       h.Status,
			})
		}

		return jsonResult(map[string]interface{}{
			"message":         "Targets resolved. Review the host list, then call run_live_query with the SAME filter args to fire against exactly these hosts.",
			"targeted_count":  len(targets),
			"targets":         previewItems,
			"truncated":       truncated,
			"schema_platform": schemaPlatform,
			"schema":          schema,
		})
	})
}

// inferPlatformFromTargets returns the dominant osquery platform string
// across a target list, or "all" if mixed. Used to pick the schema context
// when the caller didn't pin a specific platform.
func inferPlatformFromTargets(targets []Endpoint) string {
	counts := make(map[string]int)
	for _, t := range targets {
		switch strings.ToLower(t.Platform) {
		case "darwin":
			counts["macos"]++
		case "windows":
			counts["windows"]++
		case "ubuntu", "centos", "rhel", "debian", "fedora", "amzn", "linux", "opensuse-leap":
			counts["linux"]++
		case "chrome":
			counts["chromeos"]++
		default:
			counts["other"]++
		}
	}
	if len(counts) == 1 {
		for k := range counts {
			if k != "other" {
				return k
			}
		}
	}
	return "all"
}

func registerRunLiveQuery(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("run_live_query",
		mcp.WithDescription("Step 2 of 2. MUST call get_osquery_schema(platform=<target>) (or prepare_live_query, which embeds the schema response) BEFORE writing the sql argument. This verifies column NAMES and TYPES against the canonical schema — many osquery columns are TEXT despite numeric-looking values (e.g. windows_update_history.result_code is TEXT 'Succeeded'/'Failed', not an integer). Skipping the schema check produces queries that run but silently return zero rows.\n\nResolve targets and run an OSQuery SQL statement against Fleet devices. Accepts the SAME filter dimensions as prepare_live_query (intersection across fleet, platform, label, status, query, policy, CVE, hostnames, host_ids). Resolved target set is included in the response so the caller sees exactly which hosts were queried.\n\nTeam scoping: when `fleet` is set, the transient saved query that this tool creates internally is also scoped to that team — visible only under that team in the Fleet UI / audit log, with that team's RBAC. When the user mentions a team (e.g. 'Workstations'), pass it as `fleet`; do not run queries Globally and rely on host filters alone.\n\nUse the smallest target set that answers the question. Example: a CVE remediation check should target only hosts impacted by that CVE — pass cve_id + fleet, not platform=all."),
		mcp.WithString("sql", mcp.Required(), mcp.Description("The OSQuery SQL statement to run (e.g. 'SELECT * FROM os_version;')")),
		mcp.WithString("fleet", mcp.Description("Fleet (team) name.")),
		mcp.WithString("platform", mcp.Description("Platform: 'macos' / 'windows' / 'linux' / 'chromeos'.")),
		mcp.WithString("label", mcp.Description("Custom Fleet label name. Takes precedence over platform when both set.")),
		mcp.WithString("status", mcp.Description("Host status: 'online' / 'offline' / 'new' / 'mia'.")),
		mcp.WithString("query", mcp.Description("Substring matched against hostname / serial / IP / model / user inventory.")),
		mcp.WithString("policy_id", mcp.Description("Numeric policy ID.")),
		mcp.WithString("policy_response", mcp.Description("'passing' or 'failing'. Requires policy_id.")),
		mcp.WithString("cve_id", mcp.Description("CVE ID. Targets hosts running affected software versions.")),
		mcp.WithString("host_ids", mcp.Description("Optional comma-separated numeric host IDs (unambiguous).")),
		mcp.WithString("hostnames", mcp.Description("Optional comma-separated hostnames. Errors on collision — use host_ids instead.")),
		mcp.WithString("labels", mcp.Description("LEGACY — first item only; prefer 'label'.")),
		mcp.WithString("platforms", mcp.Description("LEGACY — first item only; prefer 'platform'.")),
		mcp.WithString("fleets", mcp.Description("LEGACY — first item only; prefer 'fleet'.")),
		// run_live_query fires osquery against every targeted device and creates
		// (then deletes) a transient saved query on the Fleet server. Even when the
		// SQL itself is a SELECT, it consumes device CPU, surfaces in EDR telemetry,
		// and writes/deletes Fleet state. NOT read-only, IS destructive: MCP clients
		// must prompt for explicit user approval.
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: run_live_query")
		sql, err := request.RequireString("sql")
		if err != nil || sql == "" {
			return mcp.NewToolResultError("sql is required"), nil
		}

		spec, err := buildLiveQuerySpecFromRequest(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Pre-flight: validate SQL table compatibility for the declared platform.
		// Build a list from singular + legacy plural so existing callers keep
		// working while new callers use the singular field.
		validatePlatforms := []string{}
		if spec.Platform != "" {
			validatePlatforms = append(validatePlatforms, spec.Platform)
		}
		validatePlatforms = append(validatePlatforms, spec.LegacyPlatforms...)
		if len(validatePlatforms) > 0 {
			if valErr := ValidateSQLForPlatforms(sql, validatePlatforms); valErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("SQL platform validation failed: %v", valErr)), nil
			}
		}

		// Resolve targets ourselves (rather than letting RunLiveQueryWithSpec
		// do it internally) so we can include the host list in the response —
		// the caller sees exactly which hosts were queried.
		targets, rErr := fleetClient.ResolveLiveQueryTargets(ctx, spec)
		if rErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Target resolution failed: %v", rErr)), nil
		}
		if len(targets) == 0 {
			return mcp.NewToolResultError("Targets resolved to 0 hosts — refine your filters."), nil
		}

		// When no explicit platform filter was given (e.g. caller filtered by
		// hostname / label / CVE), the pre-flight above ran with an empty
		// platform list and validated nothing. Now that targets are resolved,
		// validate SQL against their actual platforms so a darwin-only table
		// doesn't fan out to Windows hosts.
		if len(validatePlatforms) == 0 {
			seen := make(map[string]struct{})
			targetPlatforms := make([]string, 0, 4)
			for _, t := range targets {
				if t.Platform == "" {
					continue
				}
				if _, ok := seen[t.Platform]; ok {
					continue
				}
				seen[t.Platform] = struct{}{}
				targetPlatforms = append(targetPlatforms, t.Platform)
			}
			if len(targetPlatforms) > 0 {
				if valErr := ValidateSQLForPlatforms(sql, targetPlatforms); valErr != nil {
					return mcp.NewToolResultError(fmt.Sprintf("SQL platform validation failed: %v", valErr)), nil
				}
			}
		}

		// Resolve team scoping for the transient saved query that
		// runMultiHostQuery creates. When `fleet` is set, the query lives
		// under that team in Fleet's UI / RBAC instead of Global.
		teamID, tErr := fleetClient.resolveLiveQueryTeamID(ctx, spec)
		if tErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Team scoping failed: %v", tErr)), nil
		}

		hostIDs := make([]uint, 0, len(targets))
		nameByID := make(map[uint]Endpoint, len(targets))
		for _, t := range targets {
			hostIDs = append(hostIDs, t.ID)
			nameByID[t.ID] = t
		}

		var results *LiveQueryResult
		if len(hostIDs) == 1 {
			results, err = fleetClient.runAdHocSingleHost(ctx, hostIDs[0], sql, nameByID)
		} else {
			results, err = fleetClient.runMultiHostQuery(ctx, hostIDs, sql, nameByID, teamID)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to run live query: %v", err)), nil
		}

		// Include target preview so the caller knows the scope it ran on.
		const previewCap = 100
		preview := targets
		truncated := false
		if len(preview) > previewCap {
			preview = preview[:previewCap]
			truncated = true
		}
		previewItems := make([]map[string]interface{}, 0, len(preview))
		for _, h := range preview {
			previewItems = append(previewItems, map[string]interface{}{
				"id":           h.ID,
				"hostname":     h.Name,
				"display_name": h.DisplayName,
				"platform":     h.Platform,
				"team_name":    h.TeamName,
				"status":       h.Status,
			})
		}

		return jsonResult(map[string]interface{}{
			"targeted_count":    len(targets),
			"targets":           previewItems,
			"targets_truncated": truncated,
			"results":           results,
		})
	})
}

func registerGetOsquerySchema(s *server.MCPServer) {
	tool := mcp.NewTool("get_osquery_schema",
		mcp.WithDescription("Returns the canonical, source-of-truth schema for Fleet/osquery tables. The data is sourced from https://fleetdm.com/tables (refreshed periodically from the canonical JSON in the fleetdm/fleet repo) and includes per-column TYPES and DESCRIPTIONS — call refresh_osquery_schema if you suspect the response is stale.\n\nDefaults to a curated short list of common security-ops tables filtered by platform. Pass `tables` (comma-separated) to fetch the full canonical schema for specific tables — use this for any table not in the curated default. ALWAYS call this before writing SQL — column TYPES vary per table, and assumed types (e.g. assuming `result_code` is integer when it is text) cause silent zero-row queries."),
		mcp.WithString("platform", mcp.Description("Target platform: 'darwin'/'macos', 'windows', 'linux', 'chrome'/'chromeos', or 'all'. Mirrors the platform tabs on https://fleetdm.com/tables. Defaults to 'all'.")),
		mcp.WithString("tables", mcp.Description("Optional comma-separated list of specific table names (e.g. 'windows_update_history,programs'). When set, returns the full canonical schema for those tables (every column, ignores `platform`). When unset, returns the curated short list filtered by platform.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_osquery_schema")

		platform := getOptionalString(request, "platform")
		if platform == "" {
			platform = "all"
		}
		tablesArg := strings.TrimSpace(getOptionalString(request, "tables"))

		var (
			tables []SchemaTable
			err    error
			warn   string
		)
		if tablesArg != "" {
			parts := strings.Split(tablesArg, ",")
			tables, err = GetOsquerySchemaForTables(parts)
			if err != nil {
				// Partial-success: when some names matched, schema returns the
				// known tables AND a non-nil "unknown tables" error. Surface
				// the warning but still return the matched tables so the LLM
				// has something to work with.
				if len(tables) == 0 {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to get schema: %v", err)), nil
				}
				warn = err.Error()
			}
		} else {
			tables, err = GetOsquerySchema(platform)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get schema: %v", err)), nil
			}
		}

		out := map[string]interface{}{
			"source": SchemaSource(),
			"tables": tables,
		}
		if warn != "" {
			out["warning"] = warn
		}
		return jsonResult(out)
	})
}

func registerRefreshOsquerySchema(s *server.MCPServer) {
	tool := mcp.NewTool("refresh_osquery_schema",
		mcp.WithDescription("Force-refresh the in-memory osquery/Fleet schema from the canonical JSON at https://raw.githubusercontent.com/fleetdm/fleet/main/schema/osquery_fleet_schema.json (the same source that powers https://fleetdm.com/tables). Use when get_osquery_schema returns data that conflicts with the live docs, or after Fleet upstream releases a new osquery version. The schema also auto-refreshes in the background; manual refresh is for the rare 'I just saw a new column on fleetdm.com that the response is missing' case."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: refresh_osquery_schema")
		if err := RefreshSchemaNow(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Schema refresh failed (previous schema retained): %v", err)), nil
		}
		return jsonResult(map[string]interface{}{
			"refreshed": true,
			"source":    SchemaSource(),
		})
	})
}
