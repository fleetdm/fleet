package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

const defaultEndpointsPerPage = 50

// SetupMCPServer creates and configures the MCP server with all available tools
func SetupMCPServer(config *Config, fleetClient *FleetClient) *server.MCPServer {
	// Create MCP server
	s := server.NewMCPServer("fleet-mcp", "1.0.0", server.WithLogging())

	// ==========================================
	// 1. Get Endpoints Tool
	// ==========================================
	getEndpointsTool := mcp.NewTool("get_endpoints",
		mcp.WithDescription("Get a list of hosts/endpoints enrolled in Fleet, including labels. Use get_host for full details on a specific host. Use get_total_system_count for total counts and get_aggregate_platforms for platform breakdowns — do NOT call this tool repeatedly with per_page=1 just to count hosts."),
		mcp.WithString("fleet", mcp.Description("Optional fleet name to filter by (e.g. '💻 Workstations')")),
		mcp.WithString("platform", mcp.Description("Optional platform to filter by (e.g. 'macos', 'windows', 'linux')")),
		mcp.WithString("status", mcp.Description("Optional host status filter (e.g. 'online', 'offline', 'new', 'mia')")),
		mcp.WithString("per_page", mcp.Description("Max number of hosts to return (default 50, max 200)")),
	)
	s.AddTool(getEndpointsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_endpoints")

		var fleet, platform, status string
		perPage := defaultEndpointsPerPage
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if t, ok := args["fleet"].(string); ok {
				fleet = t
			}
			if p, ok := args["platform"].(string); ok {
				platform = p
			}
			if s, ok := args["status"].(string); ok {
				status = s
			}
			if pp, ok := args["per_page"].(string); ok && pp != "" {
				if n, err := strconv.Atoi(pp); err == nil && n > 0 {
					perPage = n
					if perPage > 200 {
						perPage = 200
					}
				}
			}
		}

		var endpoints []Endpoint
		var err error
		if fleet != "" || platform != "" || status != "" {
			endpoints, err = fleetClient.GetEndpointsWithFilters(fleet, platform, status, perPage)
		} else {
			endpoints, err = fleetClient.GetEndpoints(perPage)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get endpoints: %v", err)), nil
		}

		totalCount, err := fleetClient.GetHostCount()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get host count: %v", err)), nil
		}

		result := struct {
			Total     int        `json:"total"`
			Returned  int        `json:"returned"`
			Endpoints []Endpoint `json:"endpoints"`
		}{
			Total:     totalCount,
			Returned:  len(endpoints),
			Endpoints: endpoints,
		}

		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 1b. Get Host Tool (with full label data)
	// ==========================================
	getHostTool := mcp.NewTool("get_host",
		mcp.WithDescription("Get full details for a single host including its labels, fleet, and platform info. Use this when you need complete data for one host; use get_endpoints when you need to list or filter many hosts."),
		mcp.WithString("identifier", mcp.Required(), mcp.Description("The host's hostname, display name, UUID, or serial number (e.g. 'dhruvs-macbook-pro.local' or 'Dhruv\\'s MacBook Pro')")),
	)
	s.AddTool(getHostTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_host")
		identifier, err := request.RequireString("identifier")
		if err != nil || identifier == "" {
			return mcp.NewToolResultError("identifier is required"), nil
		}

		host, err := fleetClient.GetHostByIdentifier(identifier)
		if err != nil {
			// If lookup by identifier fails, try searching by display_name from the endpoints list
			endpoints, epErr := fleetClient.GetEndpoints(0)
			if epErr == nil {
				lowerID := strings.ToLower(identifier)
				for _, ep := range endpoints {
					if strings.ToLower(ep.DisplayName) == lowerID ||
						strings.ToLower(ep.ComputerName) == lowerID ||
						strings.ToLower(ep.Name) == lowerID {
						// Found a match — fetch by numeric ID via hostname
						host, err = fleetClient.GetHostByIdentifier(ep.Name)
						break
					}
				}
			}
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Host not found: %v", err)), nil
			}
		}

		b, err := json.MarshalIndent(host, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 1c. Get Fleets Tool
	// ==========================================
	getFleetsTool := mcp.NewTool("get_fleets",
		mcp.WithDescription("Get all fleets with their IDs and names. Use this to discover the exact fleet names before filtering by fleet in other tools."),
	)
	s.AddTool(getFleetsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_fleets")
		fleets, err := fleetClient.GetTeams()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get fleets: %v", err)), nil
		}
		b, err := json.MarshalIndent(fleets, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	getQueriesTool := mcp.NewTool("get_queries", mcp.WithDescription("Get a list of all saved queries in Fleet"))
	s.AddTool(getQueriesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_queries")
		queries, err := fleetClient.GetQueries()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get queries: %v", err)), nil
		}

		b, err := json.MarshalIndent(queries, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 3. Get Policies Tool
	// ==========================================
	getPoliciesTool := mcp.NewTool("get_policies", mcp.WithDescription("Get all Fleet policies with their pass/fail host counts. Focus on passing_host_count and failing_host_count to assess compliance — that is what matters. Do not comment on enforcement status."))
	s.AddTool(getPoliciesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_policies")
		policies, err := fleetClient.GetPolicies()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get policies: %v", err)), nil
		}

		b, err := json.MarshalIndent(policies, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 4. Get Labels Tool
	// ==========================================
	getLabelsTool := mcp.NewTool("get_labels", mcp.WithDescription("Get a list of all labels in Fleet"))
	s.AddTool(getLabelsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_labels")
		labels, err := fleetClient.GetLabels()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get labels: %v", err)), nil
		}

		b, err := json.MarshalIndent(labels, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 5. Get Aggregate Platforms Tool
	// ==========================================
	getPlatformsTool := mcp.NewTool("get_aggregate_platforms", mcp.WithDescription("Get the count of systems aggregated by platform (macOS, Windows, Linux)"))
	s.AddTool(getPlatformsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_aggregate_platforms")
		aggregate, err := fleetClient.GetEndpointsWithAggregations()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get aggregate platforms: %v", err)), nil
		}

		// Retrieve platform_breakdown
		dataMap, ok := aggregate.Data.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Unexpected data format returned from Fleet"), nil
		}

		platformBreakdown, ok := dataMap["platform_breakdown"]
		if !ok {
			return mcp.NewToolResultError("platform_breakdown key missing from Fleet response"), nil
		}

		b, err := json.MarshalIndent(platformBreakdown, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 6. Get Aggregate Total Tool
	// ==========================================
	getTotalTool := mcp.NewTool("get_total_system_count", mcp.WithDescription("Get the total count of active systems enrolled in Fleet"))
	s.AddTool(getTotalTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_total_system_count")
		count, err := fleetClient.GetHostCount()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get total count: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Total Enrolled Systems: %d", count)), nil
	})

	// ==========================================
	// 7. Get Policy Compliance Tool
	// ==========================================
	getPolicyComplianceTool := mcp.NewTool("get_policy_compliance",
		mcp.WithDescription("Check the compliance status for a specific global policy ID"),
		mcp.WithString("policy_id", mcp.Required(), mcp.Description("The numeric ID of the global policy to check (e.g. '1', '42')")),
	)
	s.AddTool(getPolicyComplianceTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_policy_compliance")
		policyId, err := request.RequireString("policy_id")
		if err != nil || policyId == "" {
			return mcp.NewToolResultError("policy_id is required"), nil
		}

		compliance, err := fleetClient.GetPolicyCompliance(policyId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get policy compliance for ID %s: %v", policyId, err)), nil
		}

		b, err := json.MarshalIndent(compliance, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 8. Get Vulnerability Impact Tool
	// ==========================================
	getVulnerabilityImpactTool := mcp.NewTool("get_vulnerability_impact",
		mcp.WithDescription("Check the amount of systems impacted by a specific vulnerability (CVE)"),
		mcp.WithString("cve_id", mcp.Required(), mcp.Description("The CVE ID to check (e.g. 'CVE-2022-40898')")),
	)
	s.AddTool(getVulnerabilityImpactTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_vulnerability_impact")
		cveId, err := request.RequireString("cve_id")
		if err != nil || cveId == "" {
			return mcp.NewToolResultError("cve_id is required"), nil
		}

		impact, err := fleetClient.GetVulnerabilityImpact(cveId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get vulnerability impact for CVE %s: %v", cveId, err)), nil
		}

		b, err := json.MarshalIndent(impact, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 9. Prepare Live Query Tool
	// ==========================================
	prepareLiveQueryTool := mcp.NewTool("prepare_live_query",
		mcp.WithDescription("Step 1 of 2 for running a live query. Validates targets and returns the exact OSQuery schemas so you can write a valid SQL query."),
		mcp.WithString("hostnames", mcp.Description("Optional comma-separated list of hostnames to target (e.g. 'mac-1, ubuntu-server')")),
		mcp.WithString("labels", mcp.Description("Optional comma-separated list of Fleet label names to target (e.g. 'macos, engineering')")),
		mcp.WithString("platforms", mcp.Description("Optional comma-separated list of platforms to target (e.g. 'macos, windows, linux')")),
		mcp.WithString("fleets", mcp.Description("Optional comma-separated list of fleet names to target (e.g. 'engineering, security')")),
	)
	s.AddTool(prepareLiveQueryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: prepare_live_query")

		var hostnames, labels, platforms, fleets []string
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if hStr, ok := args["hostnames"].(string); ok && hStr != "" {
				for _, h := range strings.Split(hStr, ",") {
					hostnames = append(hostnames, strings.TrimSpace(h))
				}
			}
			if lStr, ok := args["labels"].(string); ok && lStr != "" {
				for _, l := range strings.Split(lStr, ",") {
					labels = append(labels, strings.TrimSpace(l))
				}
			}
			if pStr, ok := args["platforms"].(string); ok && pStr != "" {
				for _, p := range strings.Split(pStr, ",") {
					platforms = append(platforms, strings.TrimSpace(p))
				}
			}
			if tStr, ok := args["fleets"].(string); ok && tStr != "" {
				for _, t := range strings.Split(tStr, ",") {
					fleets = append(fleets, strings.TrimSpace(t))
				}
			}
		}

		if len(hostnames) == 0 && len(labels) == 0 && len(platforms) == 0 && len(fleets) == 0 {
			return mcp.NewToolResultError("You must provide at least one target in hostnames, labels, platforms, or fleets. 'all' is not supported directly, please specify a platform, fleet, or label if you want a wide blast radius."), nil
		}

		// Resolve Schema context. If they targeted a specific platform, get that schema, otherwise return all.
		schemaPlatform := "all"
		if len(platforms) == 1 {
			schemaPlatform = platforms[0]
		}

		schema, err := GetOsquerySchema(schemaPlatform)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get schema for context: %v", err)), nil
		}

		b, err := json.MarshalIndent(map[string]interface{}{
			"message": "Targets accepted. Use this OSQuery schema to write your SQL query.",
			"schema":  schema,
		}, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 10. Run Live Query Tool
	// ==========================================
	runLiveQueryTool := mcp.NewTool("run_live_query",
		mcp.WithDescription("Step 2 of 2. Run an arbitrary live OSQuery SQL statement against Fleet devices. Wait for results. Ensure you used prepare_live_query first."),
		mcp.WithString("sql", mcp.Required(), mcp.Description("The OSQuery SQL statement to run (e.g. 'SELECT * FROM os_version;')")),
		mcp.WithString("hostnames", mcp.Description("Optional comma-separated list of hostnames to target (e.g. 'mac-1, ubuntu-server')")),
		mcp.WithString("labels", mcp.Description("Optional comma-separated list of Fleet label names to target (e.g. 'macos, engineering')")),
		mcp.WithString("platforms", mcp.Description("Optional comma-separated list of platforms to target (e.g. 'macos, windows, linux')")),
		mcp.WithString("fleets", mcp.Description("Optional comma-separated list of fleet names to target (e.g. 'engineering, security')")),
	)
	s.AddTool(runLiveQueryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: run_live_query")
		sql, err := request.RequireString("sql")
		if err != nil || sql == "" {
			return mcp.NewToolResultError("sql is required"), nil
		}

		var hostnames, labels, platforms, fleets []string
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if hStr, ok := args["hostnames"].(string); ok && hStr != "" {
				for _, h := range strings.Split(hStr, ",") {
					hostnames = append(hostnames, strings.TrimSpace(h))
				}
			}
			if lStr, ok := args["labels"].(string); ok && lStr != "" {
				for _, l := range strings.Split(lStr, ",") {
					labels = append(labels, strings.TrimSpace(l))
				}
			}
			if pStr, ok := args["platforms"].(string); ok && pStr != "" {
				for _, p := range strings.Split(pStr, ",") {
					platforms = append(platforms, strings.TrimSpace(p))
				}
			}
			if tStr, ok := args["fleets"].(string); ok && tStr != "" {
				for _, t := range strings.Split(tStr, ",") {
					fleets = append(fleets, strings.TrimSpace(t))
				}
			}
		}

		if len(hostnames) == 0 && len(labels) == 0 && len(platforms) == 0 && len(fleets) == 0 {
			return mcp.NewToolResultError("You must provide at least one target in hostnames, labels, platforms, or fleets to run the query."), nil
		}

		// Pre-flight: validate SQL table compatibility for the specified platforms
		if len(platforms) > 0 {
			if valErr := ValidateSQLForPlatforms(sql, platforms); valErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("SQL platform validation failed: %v", valErr)), nil
			}
		}

		results, err := fleetClient.RunLiveQuery(sql, hostnames, labels, platforms, fleets)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to run live query: %v", err)), nil
		}

		b, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 10. Create Saved Query Tool
	// ==========================================
	createSavedQueryTool := mcp.NewTool("create_saved_query",
		mcp.WithDescription("Create a new saved query in Fleet"),
		mcp.WithString("name", mcp.Required(), mcp.Description("The name of the query")),
		mcp.WithString("sql", mcp.Required(), mcp.Description("The OSQuery SQL statement")),
		mcp.WithString("description", mcp.Description("Description of what the query does")),
		mcp.WithString("platform", mcp.Description("Target platform (e.g., 'darwin,windows,linux'). Leave empty for all.")),
	)
	s.AddTool(createSavedQueryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: create_saved_query")

		name, err := request.RequireString("name")
		if err != nil || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		sql, err := request.RequireString("sql")
		if err != nil || sql == "" {
			return mcp.NewToolResultError("sql is required"), nil
		}

		desc := ""
		platform := ""
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if d, ok := args["description"].(string); ok {
				desc = d
			}
			if p, ok := args["platform"].(string); ok {
				platform = p
			}
		}

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

		query, err := fleetClient.CreateSavedQuery(name, desc, sql, platform)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create saved query: %v", err)), nil
		}

		b, err := json.MarshalIndent(query, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 11. Get Osquery Schema Tool
	// ==========================================
	getOsquerySchemaTool := mcp.NewTool("get_osquery_schema",
		mcp.WithDescription("Get the hardcoded, 100% accurate schema for the most important Fleet/Osquery tables and their columns. Use this before writing SQL queries."),
		mcp.WithString("platform", mcp.Description("Target platform to filter tables for (e.g. 'macos', 'windows', 'linux', 'all'). Defaults to 'all'.")),
	)
	s.AddTool(getOsquerySchemaTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_osquery_schema")

		platform := "all"
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if p, ok := args["platform"].(string); ok && p != "" {
				platform = p
			}
		}

		schema, err := GetOsquerySchema(platform)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get schema: %v", err)), nil
		}

		b, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	// ==========================================
	// 12. Get Vetted Queries Tool
	// ==========================================
	getVettedQueriesTool := mcp.NewTool("get_vetted_queries",
		mcp.WithDescription("Get the library of 100% vetted, production-safe CIS-8.1 policy queries for macOS, Windows, and Linux. Always use these as a reference or starting point for creating new policies — they have been tested and use the correct table schemas for each platform."),
		mcp.WithString("platform", mcp.Description("Filter by platform: 'darwin' or 'macos' for macOS, 'windows' for Windows, 'linux' for Linux, 'all' for everything. Defaults to 'all'.")),
	)
	s.AddTool(getVettedQueriesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_vetted_queries")

		platform := "all"
		if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if p, ok := args["platform"].(string); ok && p != "" {
				platform = p
			}
		}

		queries := GetVettedQueries(platform)
		if len(queries) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("No vetted queries found for platform: %s", platform)), nil
		}

		b, err := json.MarshalIndent(queries, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
		}

		return mcp.NewToolResultText(string(b)), nil
	})

	return s
}
