package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// registerPolicyTools attaches policy- and vulnerability-domain MCP tools to s.
// Tools registered: get_policies, get_policy_compliance, get_policy_hosts,
// get_vulnerability_impact, get_vulnerability_hosts.
//
// All tools in this group are read-only against the Fleet API, idempotent,
// and non-destructive.
func registerPolicyTools(s *server.MCPServer, fleetClient *FleetClient) {
	registerGetPolicies(s, fleetClient)
	registerGetPolicyCompliance(s, fleetClient)
	registerGetPolicyHosts(s, fleetClient)
	registerGetVulnerabilityImpact(s, fleetClient)
	registerGetVulnerabilityHosts(s, fleetClient)
}

func registerGetPolicies(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_policies",
		mcp.WithDescription("Get all Fleet policies with their pass/fail host counts. Focus on passing_host_count and failing_host_count to assess compliance — that is what matters. Do not comment on enforcement status."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_policies")
		policies, err := fleetClient.GetPolicies(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get policies: %v", err)), nil
		}
		return jsonResult(policies)
	})
}

func registerGetPolicyCompliance(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_policy_compliance",
		mcp.WithDescription("Get pass/fail counts for a specific policy. By default returns global counts. Pass `fleet` to scope counts to a single fleet (e.g. compliance numbers shown on the fleet's Policies tab in the Fleet UI). For host-level breakdowns, use get_policy_hosts."),
		mcp.WithString("policy_id", mcp.Required(), mcp.Description("The numeric ID of the policy to check (e.g. '1', '42')")),
		mcp.WithString("fleet", mcp.Description("Optional fleet name to scope compliance counts to one fleet (e.g. '💻 Workstations'). Omit for global aggregate.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_policy_compliance")
		policyID, err := request.RequireString("policy_id")
		if err != nil || policyID == "" {
			return mcp.NewToolResultError("policy_id is required (use get_policies to list available policy IDs)"), nil
		}
		if _, perr := parsePositiveUintString("policy_id", policyID); perr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("%v (use get_policies to list valid IDs)", perr)), nil
		}

		fleet := getOptionalString(request, "fleet")
		var compliance *PolicyCompliance
		if fleet != "" {
			teamIDs, terr := fleetClient.resolveTeamNames(ctx, []string{fleet})
			if terr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve fleet %q: %v", fleet, terr)), nil
			}
			compliance, err = fleetClient.GetTeamPolicyCompliance(ctx, fmt.Sprintf("%d", teamIDs[0]), policyID)
		} else {
			compliance, err = fleetClient.GetPolicyCompliance(ctx, policyID)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get policy compliance for ID %s: %v", policyID, err)), nil
		}
		return jsonResult(compliance)
	})
}

func registerGetVulnerabilityImpact(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_vulnerability_impact",
		mcp.WithDescription("Check the amount of systems impacted by a specific vulnerability (CVE). Returns an aggregate count only — for the actual list of affected hosts (with optional filtering), use get_vulnerability_hosts."),
		mcp.WithString("cve_id", mcp.Required(), mcp.Description("The CVE ID to check (e.g. 'CVE-2022-40898')")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_vulnerability_impact")
		cveID, err := request.RequireString("cve_id")
		if err != nil || cveID == "" {
			return mcp.NewToolResultError("cve_id is required (expected shape CVE-YYYY-NNNN, e.g. CVE-2026-31431)"), nil
		}
		if verr := validateCVEID(cveID); verr != nil {
			return mcp.NewToolResultError(verr.Error()), nil
		}

		impact, err := fleetClient.GetVulnerabilityImpact(ctx, cveID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get vulnerability impact for CVE %s: %v", cveID, err)), nil
		}
		return jsonResult(impact)
	})
}

func registerGetPolicyHosts(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_policy_hosts",
		mcp.WithDescription("List the hosts that pass or fail a given policy, optionally narrowed by fleet / platform / label / status / substring. Use this to answer 'which Linux hosts are failing policy 42' or 'which hosts in the engineering fleet are non-compliant with policy 17'. Use get_policies first to discover the numeric policy_id. All filter dimensions compose server-side."),
		mcp.WithString("policy_id", mcp.Required(), mcp.Description("The numeric ID of the policy (from get_policies)")),
		mcp.WithString("response", mcp.Description("Optional 'passing' or 'failing'. Defaults to both — pass it to narrow to one side.")),
		mcp.WithString("fleet", mcp.Description("Optional fleet name (e.g. '💻 Workstations')")),
		mcp.WithString("platform", mcp.Description("Optional platform (e.g. 'macos', 'windows', 'linux')")),
		mcp.WithString("label", mcp.Description("Optional label name. Resolved server-side.")),
		mcp.WithString("status", mcp.Description("Optional host status ('online', 'offline', 'new', 'mia')")),
		mcp.WithString("query", mcp.Description("Optional substring matched against hostname / serial / IP / model / user inventory.")),
		mcp.WithString("per_page", mcp.Description("Max number of hosts to return (default 50, max 200)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_policy_hosts")
		policyID, err := request.RequireString("policy_id")
		if err != nil || policyID == "" {
			return mcp.NewToolResultError("policy_id is required (use get_policies to list available policy IDs)"), nil
		}
		if _, perr := parsePositiveUintString("policy_id", policyID); perr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("%v (use get_policies to list valid IDs)", perr)), nil
		}

		responseFilter := getOptionalString(request, "response")
		if responseFilter != "" && responseFilter != "passing" && responseFilter != "failing" {
			return mcp.NewToolResultError(fmt.Sprintf("response must be 'passing' or 'failing', got %q", responseFilter)), nil
		}

		fleet := getOptionalString(request, "fleet")
		platform := getOptionalString(request, "platform")
		label := getOptionalString(request, "label")
		status := getOptionalString(request, "status")
		query := getOptionalString(request, "query")

		perPage := parsePerPageArg(request, defaultEndpointsPerPage)

		hosts, err := fleetClient.GetEndpointsWithFilters(ctx, fleet, platform, status, query, label, policyID, responseFilter, perPage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get hosts for policy %s: %v", policyID, err)), nil
		}

		return jsonResult(struct {
			PolicyID string     `json:"policy_id"`
			Response string     `json:"response_filter"`
			Returned int        `json:"returned"`
			Hosts    []Endpoint `json:"hosts"`
		}{
			PolicyID: policyID,
			Response: responseFilter,
			Returned: len(hosts),
			Hosts:    hosts,
		})
	})
}

func registerGetVulnerabilityHosts(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_vulnerability_hosts",
		mcp.WithDescription("List the specific hosts impacted by a CVE, optionally narrowed by fleet / platform / label / status / substring. Use this — NOT get_vulnerability_impact — when the question is 'which of my hosts are affected by CVE-X' or 'are any prod servers vulnerable to CVE-Y'. get_vulnerability_impact returns only an aggregate count; this tool returns the actual host list. Composes server-side via Fleet's affected-software lookup."),
		mcp.WithString("cve_id", mcp.Required(), mcp.Description("The CVE ID (e.g. 'CVE-2022-40898')")),
		mcp.WithString("fleet", mcp.Description("Optional fleet name (e.g. '💻 Workstations')")),
		mcp.WithString("platform", mcp.Description("Optional platform (e.g. 'macos', 'windows', 'linux')")),
		mcp.WithString("label", mcp.Description("Optional label name. Resolved server-side.")),
		mcp.WithString("status", mcp.Description("Optional host status ('online', 'offline', 'new', 'mia')")),
		mcp.WithString("query", mcp.Description("Optional substring matched against hostname / serial / IP / model / user inventory.")),
		mcp.WithString("per_page", mcp.Description("Max number of hosts to return (default 50, max 200)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_vulnerability_hosts")
		cveID, err := request.RequireString("cve_id")
		if err != nil || cveID == "" {
			return mcp.NewToolResultError("cve_id is required (expected shape CVE-YYYY-NNNN, e.g. CVE-2026-31431)"), nil
		}
		if verr := validateCVEID(cveID); verr != nil {
			return mcp.NewToolResultError(verr.Error()), nil
		}

		fleet := getOptionalString(request, "fleet")
		platform := getOptionalString(request, "platform")
		label := getOptionalString(request, "label")
		status := getOptionalString(request, "status")
		query := getOptionalString(request, "query")

		perPage := parsePerPageArg(request, defaultEndpointsPerPage)

		hosts, err := fleetClient.GetHostsForCVE(ctx, cveID, fleet, platform, status, query, label, perPage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get hosts for CVE %s: %v", cveID, err)), nil
		}

		return jsonResult(struct {
			CVEID    string     `json:"cve_id"`
			Returned int        `json:"returned"`
			Hosts    []Endpoint `json:"hosts"`
		}{
			CVEID:    cveID,
			Returned: len(hosts),
			Hosts:    hosts,
		})
	})
}
