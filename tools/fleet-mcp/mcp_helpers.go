package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// getOptionalString reads an optional string argument from an MCP tool request.
// Returns empty string if the argument is absent, non-string, or the request has
// no arguments map.
func getOptionalString(req mcp.CallToolRequest, key string) string {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return ""
	}
	v, _ := args[key].(string)
	return v
}

// parseCSVArg reads an optional comma-separated string argument and returns
// each non-empty segment trimmed of surrounding whitespace. Empty segments
// (e.g. "foo,,bar" or " , , ") are dropped — otherwise they propagate as
// zero-value strings into downstream filter logic, and a leading empty
// segment can silently disable filters that pull only `parts[0]`. Returns
// nil when the argument is absent, empty, or contained only whitespace/commas.
func parseCSVArg(req mcp.CallToolRequest, key string) []string {
	raw := getOptionalString(req, key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// jsonResult marshals v as indented JSON and wraps it in an MCP text tool
// result. On marshal failure, returns an MCP error result rather than a Go
// error so the client receives a structured failure (matching the existing
// handler convention).
func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format output: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// defaultPerPageMax is the documented hard cap on the `per_page` MCP arg
// across host-listing tools. Callers that ask for more are silently clamped
// to keep result sets bounded — Fleet inventories of 50k hosts × ~2KB per
// Endpoint = ~100MB which would OOM the MCP if we let any caller fetch the
// world.
const defaultPerPageMax = 200

// parsePerPageArg reads the `per_page` MCP arg and returns a clamped value:
//   - missing or unparseable → fallback (intended to be the handler's default).
//   - n <= 0 → fallback (negative or zero is a misuse; treat as default).
//   - n > defaultPerPageMax → clamped to defaultPerPageMax.
//
// Centralized so every host-listing tool enforces the same contract that
// the tool descriptions advertise ("default 50, max 200").
func parsePerPageArg(req mcp.CallToolRequest, fallback int) int {
	raw := getOptionalString(req, "per_page")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	if n > defaultPerPageMax {
		return defaultPerPageMax
	}
	return n
}

// cveIDPattern matches the canonical CVE-YYYY-NNNN[N…] identifier shape
// (CVE prefix, four-digit year, dash, four-or-more-digit sequence). Anchored
// so that only the full-string form is accepted. Used by validateCVEID.
var cveIDPattern = regexp.MustCompile(`^CVE-\d{4}-\d{4,}$`)

// validateCVEID rejects CVE identifiers that don't match the canonical shape
// before we send them to Fleet. Stops malformed inputs from becoming opaque
// Fleet API errors and prevents weird inputs (URL injection attempts,
// unicode, etc.) from reaching the upstream.
func validateCVEID(cveID string) error {
	cveID = strings.TrimSpace(cveID)
	if cveID == "" {
		return fmt.Errorf("cve_id is required (expected shape CVE-YYYY-NNNN, e.g. CVE-2026-31431)")
	}
	if !cveIDPattern.MatchString(cveID) {
		return fmt.Errorf("cve_id %q is not a valid CVE identifier (expected shape CVE-YYYY-NNNN, e.g. CVE-2026-31431)", cveID)
	}
	return nil
}

// parsePositiveUintString parses a string like "42" as a positive integer.
// Used to validate numeric path-segment params (policy_id, team_id) before
// they're interpolated into Fleet API URLs. Returns an error naming the
// field so the AI client gets a usable hint about what failed.
func parsePositiveUintString(field, val string) (uint64, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	n, err := strconv.ParseUint(val, 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("%s must be a positive integer (got %q)", field, val)
	}
	return n, nil
}

// parseCSVUintArg reads an optional comma-separated list of unsigned integers.
// Returns nil when absent / empty. Returns an error naming the bad token if
// any segment is not a positive integer — surfaced verbatim to the caller.
func parseCSVUintArg(req mcp.CallToolRequest, key string) ([]uint, error) {
	raw := getOptionalString(req, key)
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		n, err := strconv.ParseUint(t, 10, strconv.IntSize)
		if err != nil || n == 0 {
			return nil, fmt.Errorf("%s: %q is not a positive integer", key, t)
		}
		out = append(out, uint(n))
	}
	return out, nil
}

// buildLiveQuerySpecFromRequest reads every supported targeting argument off
// an MCP request and returns a LiveQueryTargetSpec ready to hand to
// FleetClient.ResolveLiveQueryTargets. Centralizes argument parsing so the
// preview path (prepare_live_query) and the execution path (run_live_query)
// stay byte-for-byte aligned on what the user actually targeted.
func buildLiveQuerySpecFromRequest(req mcp.CallToolRequest) (LiveQueryTargetSpec, error) {
	hostIDs, err := parseCSVUintArg(req, "host_ids")
	if err != nil {
		return LiveQueryTargetSpec{}, err
	}
	policyResp := getOptionalString(req, "policy_response")
	policyID := getOptionalString(req, "policy_id")
	if policyResp != "" && policyID == "" {
		return LiveQueryTargetSpec{}, fmt.Errorf("policy_response is only valid when policy_id is also set")
	}
	if policyResp != "" && policyResp != "passing" && policyResp != "failing" {
		return LiveQueryTargetSpec{}, fmt.Errorf("policy_response must be 'passing' or 'failing', got %q", policyResp)
	}
	if policyID != "" {
		if _, err := parsePositiveUintString("policy_id", policyID); err != nil {
			return LiveQueryTargetSpec{}, err
		}
	}
	cveID := getOptionalString(req, "cve_id")
	if cveID != "" {
		if err := validateCVEID(cveID); err != nil {
			return LiveQueryTargetSpec{}, err
		}
	}
	return LiveQueryTargetSpec{
		Fleet:           getOptionalString(req, "fleet"),
		Platform:        getOptionalString(req, "platform"),
		Label:           getOptionalString(req, "label"),
		Status:          getOptionalString(req, "status"),
		Query:           getOptionalString(req, "query"),
		PolicyID:        policyID,
		PolicyResponse:  policyResp,
		CVEID:           cveID,
		Hostnames:       parseCSVArg(req, "hostnames"),
		HostIDs:         hostIDs,
		LegacyFleets:    parseCSVArg(req, "fleets"),
		LegacyPlatforms: parseCSVArg(req, "platforms"),
		LegacyLabels:    parseCSVArg(req, "labels"),
	}, nil
}
