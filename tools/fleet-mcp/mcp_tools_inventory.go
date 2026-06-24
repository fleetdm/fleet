package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

func registerInventoryTools(s *server.MCPServer, fleetClient *FleetClient) {
	registerGetSoftware(s, fleetClient)
	registerGetHostUsers(s, fleetClient)
}

func validateGetSoftwareArgs(perHost bool, fleet, platform, vulnerable string) error {
	if perHost && (fleet != "" || platform != "") {
		return fmt.Errorf("host_id/host_identifier are mutually exclusive with fleet/platform — pick per-host or cross-host mode")
	}
	if vulnerable != "" && vulnerable != "true" && vulnerable != "false" {
		return fmt.Errorf("vulnerable must be 'true' or 'false', got %q", vulnerable)
	}
	if !perHost && platform != "" && fleet == "" {
		return fmt.Errorf("platform requires fleet in cross-host mode — Fleet's software/titles endpoint only filters by platform when a team is also set")
	}
	return nil
}

func registerGetSoftware(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_software",
		mcp.WithDescription("List software/packages from Fleet's stored host inventory (refreshed on each host check-in — works even when hosts are offline). Two modes, picked automatically:\n\n- PER-HOST mode (when host_id OR host_identifier is set): every package installed on that host, including version, source, install paths, and any matching CVEs. Use this for 'what's on host X?' questions.\n- CROSS-HOST mode (no host arg): software TITLES seen across hosts, optionally scoped by fleet/platform/vulnerability. Use this for 'do we have python on any Workstation?' or 'every npm package across the fleet'.\n\nThe `source` arg is the osquery source-table name (e.g. 'npm_packages', 'python_packages', 'apps', 'deb_packages', 'rpm_packages', 'chrome_extensions', 'vscode_extensions', 'homebrew_packages') and is matched client-side case-insensitively. Use `query` for a substring match on software name OR a CVE id ('CVE-2026-12345') — server-side, fast. Prefer this tool over run_live_query for inventory lookups: the cached data is always-available and doesn't burn host CPU."),
		mcp.WithString("host_id", mcp.Description("Numeric Fleet host ID. Switches to per-host mode. Mutually exclusive with fleet/platform.")),
		mcp.WithString("host_identifier", mcp.Description("Exact hostname / UUID / serial OR a substring (same disambiguation as get_host). Switches to per-host mode. Mutually exclusive with fleet/platform.")),
		mcp.WithString("fleet", mcp.Description("Fleet name (e.g. 'Workstations') — cross-host mode only. Resolved via get_fleets.")),
		mcp.WithString("platform", mcp.Description("Cross-host mode only, and REQUIRES `fleet` (Fleet's software/titles endpoint only filters by platform together with a team). One of: macos, windows, linux, chrome, ios, ipados.")),
		mcp.WithString("vulnerable", mcp.Description("'true' to show only software with known CVEs; 'false' or omitted shows all.")),
		mcp.WithString("source", mcp.Description("osquery source table (e.g. 'npm_packages', 'python_packages', 'apps', 'deb_packages', 'chrome_extensions'). Client-side case-insensitive filter — Fleet doesn't accept this server-side.")),
		mcp.WithString("query", mcp.Description("Substring (case-insensitive) matched against software name OR a CVE id. Server-side. Use for plain 'do we have X?' lookups.")),
		mcp.WithString("per_page", mcp.Description("Max rows in the merged result (default 50, max 200). Applied AFTER the source filter so the cap reflects the filtered set.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_software")

		hostIDArg := getOptionalString(request, "host_id")
		identifier := getOptionalString(request, "host_identifier")
		fleet := getOptionalString(request, "fleet")
		platform := getOptionalString(request, "platform")
		vulnerable := getOptionalString(request, "vulnerable")
		source := getOptionalString(request, "source")
		query := getOptionalString(request, "query")
		perPage := parsePerPageArg(request, defaultEndpointsPerPage)

		hostID, err := parseHostIDArg(hostIDArg)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		perHost := hostID != 0 || identifier != ""
		if err := validateGetSoftwareArgs(perHost, fleet, platform, vulnerable); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if perHost {
			host, candidates, ambiguous, rErr := resolveHost(ctx, fleetClient, hostID, identifier)
			if rErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve host: %v", rErr)), nil
			}
			if ambiguous {
				return jsonResult(map[string]interface{}{
					"message":    fmt.Sprintf("%d hosts match %q. Substring search does NOT cover display_name; pick the `id` from the candidates below and re-call with `host_id` set.", len(candidates), identifier),
					"candidates": candidates,
				})
			}

			software, truncated, err := fleetClient.GetHostSoftware(ctx, host.ID, query, vulnerable, source, perPage)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch host software: %v", err)), nil
			}

			return jsonResult(struct {
				Scope     string         `json:"scope"`
				HostID    uint           `json:"host_id"`
				HostName  string         `json:"host_name,omitempty"`
				Returned  int            `json:"returned"`
				Truncated bool           `json:"truncated,omitempty"`
				Software  []HostSoftware `json:"software"`
			}{
				Scope:     "host",
				HostID:    host.ID,
				HostName:  host.Name,
				Returned:  len(software),
				Truncated: truncated,
				Software:  software,
			})
		}

		titles, truncated, err := fleetClient.ListSoftwareTitles(ctx, fleet, platform, query, vulnerable, source, perPage)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list software titles: %v", err)), nil
		}

		return jsonResult(struct {
			Scope          string          `json:"scope"`
			Fleet          string          `json:"fleet,omitempty"`
			Platform       string          `json:"platform,omitempty"`
			Returned       int             `json:"returned"`
			Truncated      bool            `json:"truncated,omitempty"`
			SoftwareTitles []SoftwareTitle `json:"software_titles"`
		}{
			Scope:          "titles",
			Fleet:          fleet,
			Platform:       platform,
			Returned:       len(titles),
			Truncated:      truncated,
			SoftwareTitles: titles,
		})
	})
}

func registerGetHostUsers(s *server.MCPServer, fleetClient *FleetClient) {
	tool := mcp.NewTool("get_host_users",
		mcp.WithDescription("List OS-local user accounts on a single host as inventoried by osquery (uid, username, type, groupname, shell). Returned from Fleet's stored host detail — works even when the host is currently offline. Use this for 'which accounts exist on host X?', 'is there a user named X on this host?', or to enumerate service accounts.\n\nIDENTIFIER GUIDANCE: pass `host_id` (numeric) when known — unambiguous. `host_identifier` accepts an exact hostname / UUID / serial OR a substring (same disambiguation as get_host). On collision returns a candidate list — re-call with `host_id` from the candidate you want.\n\nOptional `query` substring filters the returned users array client-side against username / uid / groupname / shell."),
		mcp.WithString("host_id", mcp.Description("Numeric Fleet host ID. Preferred when known — unambiguous.")),
		mcp.WithString("host_identifier", mcp.Description("Exact hostname / UUID / serial OR a substring. Required if host_id is not set. Does NOT match display_name — use host_id for display-name-only hosts.")),
		mcp.WithString("query", mcp.Description("Optional case-insensitive substring filter on username / uid / groupname / shell. Client-side.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logrus.Info("Tool invoked: get_host_users")

		hostIDArg := getOptionalString(request, "host_id")
		identifier := getOptionalString(request, "host_identifier")
		query := getOptionalString(request, "query")

		hostID, err := parseHostIDArg(hostIDArg)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if hostID == 0 && identifier == "" {
			return mcp.NewToolResultError("either host_id or host_identifier is required"), nil
		}

		host, ambiguous, candidates, err := resolveHostWithUsers(ctx, fleetClient, hostID, identifier)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get host users: %v", err)), nil
		}
		if ambiguous {
			return jsonResult(map[string]interface{}{
				"message":    fmt.Sprintf("%d hosts match %q. Substring search does NOT cover display_name; pick the `id` from the candidates below and re-call with `host_id` set.", len(candidates), identifier),
				"candidates": candidates,
			})
		}

		users := host.Users
		if q := strings.TrimSpace(query); q != "" {
			users = filterHostUsers(users, q)
		}

		return jsonResult(struct {
			Host     Endpoint   `json:"host"`
			Returned int        `json:"returned"`
			Users    []HostUser `json:"users"`
		}{
			Host:     host.Endpoint,
			Returned: len(users),
			Users:    users,
		})
	})
}

// hostID is the validated host_id (0 means none — fall back to identifier).
// Query-first so hostname collisions surface as candidates before the
// identifier-endpoint fallback. Returns the resolved host so callers don't
// re-fetch it just for the hostname.
func resolveHost(ctx context.Context, fleetClient *FleetClient, hostID uint, identifier string) (host *Endpoint, candidates []Endpoint, ambiguous bool, err error) {
	if hostID != 0 {
		h, hErr := fleetClient.GetHostByID(ctx, hostID)
		if hErr != nil {
			return nil, nil, false, hErr
		}
		return h, nil, false, nil
	}

	const maxCandidates = 50
	cands, qErr := fleetClient.GetEndpointsWithFilters(ctx, "", "", "", identifier, "", "", "", maxCandidates)

	if qErr == nil && len(cands) == 1 {
		return &cands[0], nil, false, nil
	}
	if qErr == nil && len(cands) > 1 {
		return nil, cands, true, nil
	}

	// Identifier fallback catches UUIDs the substring index misses.
	h, idErr := fleetClient.GetHostByIdentifier(ctx, identifier)
	if idErr != nil {
		return nil, nil, false, fmt.Errorf("host not found by query or identifier: %s (substring search does NOT cover display_name — try host_id if you have it)", identifier)
	}
	return h, nil, false, nil
}

func resolveHostWithUsers(ctx context.Context, fleetClient *FleetClient, hostID uint, identifier string) (*HostWithUsers, bool, []Endpoint, error) {
	byIdentifier := func(ctx context.Context, ident string) (*HostWithUsers, error) {
		ep, err := fleetClient.GetHostByIdentifier(ctx, ident)
		if err != nil {
			return nil, err
		}

		// The identifier endpoint doesn't populate users, so the fallback resolves the
		// id there and refetches by id (which does).
		return fleetClient.GetHostByIDWithUsers(ctx, ep.ID)
	}
	return resolveHostDetail(ctx, fleetClient, hostID, identifier, fleetClient.GetHostByIDWithUsers, byIdentifier)
}

func filterHostUsers(users []HostUser, q string) []HostUser {
	needle := strings.ToLower(q)
	out := make([]HostUser, 0, len(users))
	for _, u := range users {
		uidStr := strconv.FormatUint(u.UID, 10)
		if strings.Contains(strings.ToLower(u.Username), needle) ||
			strings.Contains(uidStr, needle) ||
			strings.Contains(strings.ToLower(u.GroupName), needle) ||
			strings.Contains(strings.ToLower(u.Shell), needle) {
			out = append(out, u)
		}
	}
	return out
}
