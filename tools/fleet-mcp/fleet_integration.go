package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// FleetClient represents a client for interacting with Fleet API
type FleetClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// PlatformBreakdown represents platform distribution data
type PlatformBreakdown struct {
	MacOS    int `json:"macos"`
	Windows  int `json:"windows"`
	Linux    int `json:"linux"`
	ChromeOS int `json:"chromeos"`
	IOS      int `json:"ios"`
	IPadOS   int `json:"ipados"`
	Android  int `json:"android"`
	Other    int `json:"other"`
	Total    int `json:"total"`
}

// PolicyCompliance represents policy compliance data
type PolicyCompliance struct {
	PolicyID     string `json:"policy_id"`
	PolicyName   string `json:"policy_name"`
	Total        int    `json:"total"`
	Compliant    int    `json:"compliant"`
	NonCompliant int    `json:"non_compliant"`
}

// VulnerabilityImpact represents vulnerability impact data
type VulnerabilityImpact struct {
	CVEID           string `json:"cve_id"`
	TotalSystems    int    `json:"total_systems"`
	ImpactedSystems int    `json:"impacted_systems"`
}

// AggregateResponse represents a consistent response format for aggregations
type AggregateResponse struct {
	Count int         `json:"count"`
	Data  interface{} `json:"data"`
}

// NewFleetClient creates a new Fleet client.
// tlsSkipVerify disables certificate verification (unsafe; use only in dev/test).
// caFile, if non-empty, is a path to a PEM-encoded CA certificate to trust (for self-signed certs).
func NewFleetClient(baseURL, apiKey string, tlsSkipVerify bool, caFile string) *FleetClient {
	tlsCfg := &tls.Config{}

	if tlsSkipVerify && caFile != "" {
		logrus.Fatalf("conflicting TLS settings: tlsSkipVerify and caFile are mutually exclusive — use one or the other, not both")
	}

	if tlsSkipVerify {
		if !isLoopbackURL(baseURL) {
			logrus.Errorf("FLEET_TLS_SKIP_VERIFY is set but FLEET_BASE_URL (%s) does not point at localhost — this is unsafe for non-local deployments", baseURL)
		}
		logrus.Warn("TLS certificate verification is disabled — do not use in production")
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	} else if caFile != "" {
		pemData, err := os.ReadFile(caFile)
		if err != nil {
			logrus.Fatalf("failed to read CA certificate file %s: %v", caFile, err)
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(pemData) {
			logrus.Fatalf("failed to parse CA certificate from %s", caFile)
		}
		tlsCfg.RootCAs = certPool
		logrus.Infof("loaded custom CA certificate from %s", caFile)
	}

	transport := &http.Transport{TLSClientConfig: tlsCfg}
	return &FleetClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// isLoopbackURL parses a URL and returns true only if the hostname is exactly
// "localhost", "127.0.0.1", or "::1". This avoids prefix-matching pitfalls
// like "localhost.evil.com".
func isLoopbackURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname() // strips port if present
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// HostLabel represents a label attached to a host (Fleet returns objects, not plain strings)
type HostLabel struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// Endpoint represents a Fleet endpoint
type Endpoint struct {
	ID           uint        `json:"id"`
	Name         string      `json:"hostname"`
	DisplayName  string      `json:"display_name"`
	ComputerName string      `json:"computer_name"`
	Status       string      `json:"status"`
	LastSeen     int64       `json:"last_seen"`
	Platform     string      `json:"platform"`
	Version      string      `json:"osquery_version"`
	TeamID       *uint       `json:"team_id"`
	TeamName     string      `json:"team_name"`
	Labels       []HostLabel `json:"labels"`
}

// Query represents a Fleet query
type Query struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
	Platform    string `json:"platform"`
	Created     int64  `json:"created"`
}

// Policy represents a Fleet policy
type Policy struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Platform         string `json:"platform"`
	PassingHostCount int    `json:"passing_host_count"`
	FailingHostCount int    `json:"failing_host_count"`
}

// Label represents a Fleet label
type Label struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Created     int64  `json:"created"`
}

// Team represents a Fleet team
type Team struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AdHocQueryRequest is the body for single-host ad hoc queries
type AdHocQueryRequest struct {
	Query string `json:"query"`
}

// AdHocQueryResponse is the response from a single-host ad hoc query
type AdHocQueryResponse struct {
	HostID uint                     `json:"host_id"`
	Query  string                   `json:"query"`
	Status string                   `json:"status"`
	Error  *string                  `json:"error"`
	Rows   []map[string]interface{} `json:"rows"`
}

// MultiQueryRunRequest is the body for running a saved query against multiple hosts
type MultiQueryRunRequest struct {
	HostIDs []uint `json:"host_ids,omitempty"`
}

// LiveQueryHostResult is a single host's result from a multi-host query run
type LiveQueryHostResult struct {
	HostID uint                     `json:"host_id"`
	Rows   []map[string]interface{} `json:"rows"`
	Error  *string                  `json:"error"`
}

// MultiQueryRunResponse is the response from POST /api/v1/fleet/queries/:id/run
type MultiQueryRunResponse struct {
	QueryID            uint                  `json:"query_id"`
	TargetedHostCount  int                   `json:"targeted_host_count"`
	RespondedHostCount int                   `json:"responded_host_count"`
	Results            []LiveQueryHostResult `json:"results"`
}

// LiveQueryResult is a unified result returned from RunLiveQuery
type LiveQueryResult struct {
	TargetedHostCount  int                      `json:"targeted_host_count"`
	RespondedHostCount int                      `json:"responded_host_count"`
	Results            []map[string]interface{} `json:"results"`
}

// CreateQueryRequest represents the payload for creating a saved query
type CreateQueryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Query       string `json:"query"`
	Platform    string `json:"platform,omitempty"`
}

// normalizePlatform normalizes platform input to Fleet's canonical platform string.
func normalizePlatform(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "macos", "mac", "osx", "darwin":
		return "darwin"
	case "windows":
		return "windows"
	case "linux", "ubuntu", "centos", "rhel", "debian", "fedora", "amzn":
		return "linux"
	case "chromeos", "chrome":
		return "chrome"
	default:
		return strings.ToLower(p)
	}
}

// matchesPlatform checks if a host's platform matches the target platform.
func matchesPlatform(hostPlatform, targetPlatform string) bool {
	hp := strings.ToLower(hostPlatform)
	if targetPlatform == "linux" {
		return hp == "linux" || hp == "ubuntu" || hp == "centos" || hp == "rhel" || hp == "debian" || hp == "fedora" || hp == "amzn"
	}
	return hp == targetPlatform
}

// platformToBuiltinLabel maps user-facing platform names to Fleet's built-in label names.
func platformToBuiltinLabel(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "macos", "darwin", "mac", "osx":
		return "macOS"
	case "windows":
		return "MS Windows"
	case "linux":
		return "All Linux"
	case "chromeos", "chrome":
		return "chrome"
	default:
		return ""
	}
}

// GetEndpoints retrieves endpoints from Fleet with server-side pagination.
// Pass 0 for perPage to use the Fleet API default.
func (fc *FleetClient) GetEndpoints(perPage int) ([]Endpoint, error) {
	params := url.Values{}
	params.Set("populate_labels", "true")
	if perPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", perPage))
	}
	endpoint := "/api/v1/fleet/hosts?" + params.Encode()
	resp, err := fc.makeFleetRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get endpoints: status code %d", resp.StatusCode)
	}

	var result struct {
		Hosts []Endpoint `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode endpoints response: %w", err)
	}

	return result.Hosts, nil
}

// GetHostByIdentifier fetches full host details (including labels) by hostname, UUID, or serial.
// Uses GET /api/v1/fleet/hosts/identifier/:identifier which returns complete host and label data.
// Note: GetEndpoints already requests labels via populate_labels=true; this method is for targeted lookups of a single host.
func (fc *FleetClient) GetHostByIdentifier(identifier string) (*Endpoint, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/identifier/%s", url.PathEscape(identifier))
	resp, err := fc.makeFleetRequest("GET", endpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("host not found: %s", identifier)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get host: status code %d", resp.StatusCode)
	}

	var result struct {
		Host Endpoint `json:"host"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode host response: %w", err)
	}
	return &result.Host, nil
}

// GetQueries retrieves global and all team-specific queries from Fleet.
func (fc *FleetClient) GetQueries() ([]Query, error) {
	resp, err := fc.makeFleetRequest("GET", "/api/v1/fleet/reports", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get queries: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Queries []Query `json:"queries"`
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get queries: status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode queries: %w", err)
	}

	teams, err := fc.GetTeams()
	if err != nil {
		logrus.Warnf("skipping team queries: %v", err)
		return result.Queries, nil
	}
	for _, team := range teams {
		teamResp, err := fc.makeFleetRequest("GET", fmt.Sprintf("/api/v1/fleet/reports?team_id=%d", team.ID), nil)
		if err != nil {
			logrus.Warnf("team %d queries error: %v", team.ID, err)
			continue
		}
		if teamResp.StatusCode == http.StatusOK {
			var tr struct {
				Queries []Query `json:"queries"`
			}
			if err := json.NewDecoder(teamResp.Body).Decode(&tr); err == nil {
				for i := range tr.Queries {
					tr.Queries[i].Name = fmt.Sprintf("[%s] %s", team.Name, tr.Queries[i].Name)
				}
				result.Queries = append(result.Queries, tr.Queries...)
			}
		}
		teamResp.Body.Close()
	}
	return result.Queries, nil
}

// GetPolicies retrieves global and all team-specific policies from Fleet.
func (fc *FleetClient) GetPolicies() ([]Policy, error) {
	resp, err := fc.makeFleetRequest("GET", "/api/v1/fleet/global/policies", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Policies []Policy `json:"policies"`
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get policies: status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode policies: %w", err)
	}

	teams, err := fc.GetTeams()
	if err != nil {
		logrus.Warnf("skipping team policies: %v", err)
		return result.Policies, nil
	}
	for _, team := range teams {
		teamResp, err := fc.makeFleetRequest("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team.ID), nil)
		if err != nil {
			logrus.Warnf("team %d policies error: %v", team.ID, err)
			continue
		}
		if teamResp.StatusCode == http.StatusOK {
			var tr struct {
				Policies []Policy `json:"policies"`
			}
			if err := json.NewDecoder(teamResp.Body).Decode(&tr); err == nil {
				for i := range tr.Policies {
					tr.Policies[i].Name = fmt.Sprintf("[%s] %s", team.Name, tr.Policies[i].Name)
				}
				result.Policies = append(result.Policies, tr.Policies...)
			}
		}
		teamResp.Body.Close()
	}
	return result.Policies, nil
}

// GetLabels retrieves all labels from Fleet
func (fc *FleetClient) GetLabels() ([]Label, error) {
	endpoint := "/api/v1/fleet/labels"
	resp, err := fc.makeFleetRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get labels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get labels: status code %d", resp.StatusCode)
	}

	var result struct {
		Labels []Label `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode labels response: %w", err)
	}

	return result.Labels, nil
}

// GetFleetConfig retrieves the Fleet server configuration.
func (fc *FleetClient) GetFleetConfig() (map[string]interface{}, error) {
	resp, err := fc.makeFleetRequest("GET", "/api/v1/fleet/config", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get fleet config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get fleet config: status code %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode fleet config: %w", err)
	}
	return result, nil
}

// GetEndpointsWithAggregations retrieves all endpoints and performs aggregations
func (fc *FleetClient) GetEndpointsWithAggregations() (*AggregateResponse, error) {
	endpoints, err := fc.GetEndpoints(0)
	if err != nil {
		return nil, err
	}

	// Count by platform
	platformBreakdown := PlatformBreakdown{}
	for _, endpoint := range endpoints {
		switch endpoint.Platform {
		case "darwin":
			platformBreakdown.MacOS++
		case "windows":
			platformBreakdown.Windows++
		case "linux", "ubuntu", "centos", "rhel", "debian", "fedora", "amzn":
			platformBreakdown.Linux++
		case "chrome":
			platformBreakdown.ChromeOS++
		case "ios":
			platformBreakdown.IOS++
		case "ipados":
			platformBreakdown.IPadOS++
		case "android":
			platformBreakdown.Android++
		default:
			platformBreakdown.Other++
		}
	}
	platformBreakdown.Total = len(endpoints)

	return &AggregateResponse{
		Count: len(endpoints),
		Data: map[string]interface{}{
			"platform_breakdown": platformBreakdown,
			"total_count":        len(endpoints),
		},
	}, nil
}

// GetTeams retrieves all teams from Fleet
func (fc *FleetClient) GetTeams() ([]Team, error) {
	endpoint := "/api/v1/fleet/teams"
	resp, err := fc.makeFleetRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get teams: status code %d", resp.StatusCode)
	}

	var result struct {
		Teams []Team `json:"teams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode teams response: %w", err)
	}

	return result.Teams, nil
}

// GetHostCount retrieves the total host count without fetching all host data.
func (fc *FleetClient) GetHostCount() (int, error) {
	resp, err := fc.makeFleetRequest("GET", "/api/v1/fleet/hosts/count", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get host count: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get host count: status %d", resp.StatusCode)
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode host count response: %w", err)
	}
	return result.Count, nil
}

// getTeamHosts returns all hosts belonging to a specific team.
func (fc *FleetClient) getTeamHosts(teamID uint) ([]Endpoint, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts?team_id=%d", teamID)
	resp, err := fc.makeFleetRequest("GET", endpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get team hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get team hosts: status %d", resp.StatusCode)
	}

	var result struct {
		Hosts []Endpoint `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode team hosts: %w", err)
	}
	return result.Hosts, nil
}

// resolveTeamNames resolves team names to team IDs using exact case-insensitive
// matching. On failure, lists available teams so the caller can retry.
func (fc *FleetClient) resolveTeamNames(teamNames []string) ([]uint, error) {
	teams, err := fc.GetTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	teamMap := make(map[string]uint)
	var availableNames []string
	for _, t := range teams {
		teamMap[strings.ToLower(t.Name)] = t.ID
		availableNames = append(availableNames, t.Name)
	}

	var ids []uint
	for _, name := range teamNames {
		id, ok := teamMap[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			return nil, fmt.Errorf("fleet not found: %q (available fleets: %s)", name, strings.Join(availableNames, ", "))
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetEndpointsWithFilters retrieves endpoints from Fleet with optional server-side filters.
func (fc *FleetClient) GetEndpointsWithFilters(teamName, platform, status string, perPage int) ([]Endpoint, error) {
	params := url.Values{}
	params.Set("populate_labels", "true")
	if perPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", perPage))
	}

	if teamName != "" {
		teamIDs, err := fc.resolveTeamNames([]string{teamName})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve fleet: %w", err)
		}
		params.Set("team_id", fmt.Sprintf("%d", teamIDs[0]))
	}

	if platform != "" {
		params.Set("platform", normalizePlatform(platform))
	}

	if status != "" {
		params.Set("status", status)
	}

	endpoint := "/api/v1/fleet/hosts?" + params.Encode()

	resp, err := fc.makeFleetRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered endpoints: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get filtered endpoints: status %d", resp.StatusCode)
	}

	var result struct {
		Hosts []Endpoint `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode filtered endpoints response: %w", err)
	}

	return result.Hosts, nil
}

// GetPolicyCompliance retrieves policy compliance data
func (fc *FleetClient) GetPolicyCompliance(policyID string) (*PolicyCompliance, error) {
	endpoint := fmt.Sprintf("/api/v1/fleet/global/policies/%s", url.PathEscape(policyID))
	resp, err := fc.makeFleetRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy compliance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get policy compliance: status code %d", resp.StatusCode)
	}

	// The response format for policies typically includes hosts_count, passing_host_count, failing_host_count
	var result struct {
		Policy struct {
			ID               uint   `json:"id"`
			Name             string `json:"name"`
			PassingHostCount int    `json:"passing_host_count"`
			FailingHostCount int    `json:"failing_host_count"`
		} `json:"policy"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode policy compliance response: %w", err)
	}

	total := result.Policy.PassingHostCount + result.Policy.FailingHostCount

	return &PolicyCompliance{
		PolicyID:     fmt.Sprint(result.Policy.ID),
		PolicyName:   result.Policy.Name,
		Total:        total,
		Compliant:    result.Policy.PassingHostCount,
		NonCompliant: result.Policy.FailingHostCount,
	}, nil
}

// GetVulnerabilityImpact retrieves vulnerability impact data
func (fc *FleetClient) GetVulnerabilityImpact(cveID string) (*VulnerabilityImpact, error) {
	endpoint := fmt.Sprintf("/api/v1/fleet/vulnerabilities/%s", url.PathEscape(cveID))
	resp, err := fc.makeFleetRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get vulnerability impact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get vulnerability impact: status code %d", resp.StatusCode)
	}

	// Vulnerability response format has hosts_count
	var result struct {
		Vulnerability struct {
			CVE        string `json:"cve"`
			HostsCount int    `json:"hosts_count"`
		} `json:"vulnerability"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode vulnerability impact response: %w", err)
	}

	totalSystems := 0
	if count, err := fc.GetHostCount(); err == nil {
		totalSystems = count
	}

	return &VulnerabilityImpact{
		CVEID:           result.Vulnerability.CVE,
		TotalSystems:    totalSystems,
		ImpactedSystems: result.Vulnerability.HostsCount,
	}, nil
}

// CreateSavedQuery creates a new saved query in Fleet
func (fc *FleetClient) CreateSavedQuery(name, description, sql, platform string) (*Query, error) {
	endpoint := "/api/v1/fleet/reports"

	reqBody := CreateQueryRequest{
		Name:        name,
		Description: description,
		Query:       sql,
		Platform:    platform,
	}

	resp, err := fc.makeFleetRequest("POST", endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create saved query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create saved query: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Query Query `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode created query response: %w", err)
	}

	return &result.Query, nil
}

// RunLiveQuery executes a live query against the specified targets using Fleet's modern REST API.
// Uses targeted API calls per dimension to avoid fetching all hosts.
// For single hosts: uses per-host ad hoc endpoint (POST /api/v1/fleet/hosts/:id/query).
// For multiple hosts: creates a temp saved query → runs by ID → deletes it.
func (fc *FleetClient) RunLiveQuery(sql string, hostnames, labels, platforms, teams []string) (*LiveQueryResult, error) {
	hostIDs, nameByID, err := fc.resolveTargetHosts(hostnames, labels, platforms, teams)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target hosts: %w", err)
	}

	if len(hostIDs) == 0 {
		return nil, fmt.Errorf("no matching hosts found for the provided targets")
	}

	// Deduplicate
	seen := make(map[uint]bool)
	unique := hostIDs[:0]
	for _, id := range hostIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	hostIDs = unique

	if len(hostIDs) == 1 {
		return fc.runAdHocSingleHost(hostIDs[0], sql, nameByID)
	}
	return fc.runMultiHostQuery(hostIDs, sql, nameByID)
}

// resolveTargetHosts resolves targeting parameters into host IDs and a name lookup map.
// Uses targeted API calls to avoid fetching all hosts.
// Returns an error listing any selectors that could not be resolved.
func (fc *FleetClient) resolveTargetHosts(hostnames, labels, platforms, teams []string) ([]uint, map[uint]Endpoint, error) {
	var hostIDs []uint
	nameByID := make(map[uint]Endpoint)
	var unresolved []string

	// Hostname targeting: use GetHostByIdentifier per host (no bulk fetch)
	if len(hostnames) > 0 {
		for _, hostname := range hostnames {
			host, err := fc.GetHostByIdentifier(strings.TrimSpace(hostname))
			if err != nil {
				logrus.Warnf("host lookup failed for %q: %v", hostname, err)
				unresolved = append(unresolved, "hostname:"+strings.TrimSpace(hostname))
				continue
			}
			hostIDs = append(hostIDs, host.ID)
			nameByID[host.ID] = *host
		}
	}

	// Label targeting: use getLabelHosts per label (no bulk fetch)
	if len(labels) > 0 {
		fleetLabels, err := fc.GetLabels()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get labels: %w", err)
		}
		labelTargets := make(map[string]bool)
		for _, l := range labels {
			labelTargets[strings.ToLower(strings.TrimSpace(l))] = true
		}
		resolvedLabels := make(map[string]bool)
		for _, l := range fleetLabels {
			key := strings.ToLower(l.Name)
			if !labelTargets[key] {
				continue
			}
			members, err := fc.getLabelHosts(l.ID)
			if err != nil {
				logrus.Warnf("label host lookup failed for %q: %v", l.Name, err)
				continue
			}
			resolvedLabels[key] = true
			for _, m := range members {
				hostIDs = append(hostIDs, m.ID)
				nameByID[m.ID] = m
			}
		}
		for _, l := range labels {
			if !resolvedLabels[strings.ToLower(strings.TrimSpace(l))] {
				unresolved = append(unresolved, "label:"+strings.TrimSpace(l))
			}
		}
	}

	// Team targeting: use getTeamHosts per team (no bulk fetch)
	if len(teams) > 0 {
		teamIDs, err := fc.resolveTeamNames(teams)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve teams: %w", err)
		}
		for i, tid := range teamIDs {
			members, err := fc.getTeamHosts(tid)
			if err != nil {
				logrus.Warnf("team host lookup failed for %q (ID %d): %v", teams[i], tid, err)
				unresolved = append(unresolved, "team:"+strings.TrimSpace(teams[i]))
				continue
			}
			for _, m := range members {
				hostIDs = append(hostIDs, m.ID)
				nameByID[m.ID] = m
			}
		}
	}

	// Platform targeting: use Fleet's built-in platform labels via getLabelHosts
	if len(platforms) > 0 {
		fleetLabels, err := fc.GetLabels()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get labels for platform resolution: %w", err)
		}
		for _, p := range platforms {
			labelName := platformToBuiltinLabel(strings.TrimSpace(p))
			if labelName == "" {
				logrus.Warnf("unknown platform %q, skipping", p)
				unresolved = append(unresolved, "platform:"+strings.TrimSpace(p))
				continue
			}
			found := false
			for _, l := range fleetLabels {
				if strings.EqualFold(l.Name, labelName) {
					members, err := fc.getLabelHosts(l.ID)
					if err != nil {
						logrus.Warnf("platform label host lookup failed for %q: %v", labelName, err)
						unresolved = append(unresolved, "platform:"+strings.TrimSpace(p))
						continue
					}
					for _, m := range members {
						hostIDs = append(hostIDs, m.ID)
						nameByID[m.ID] = m
					}
					found = true
					break
				}
			}
			if !found {
				logrus.Warnf("built-in label %q not found in Fleet, skipping platform %q", labelName, p)
				unresolved = append(unresolved, "platform:"+strings.TrimSpace(p))
			}
		}
	}

	if len(unresolved) > 0 {
		return nil, nil, fmt.Errorf("unresolved selectors: %s", strings.Join(unresolved, ", "))
	}
	return hostIDs, nameByID, nil
}

// getLabelHosts returns the hosts belonging to a Fleet label.
func (fc *FleetClient) getLabelHosts(labelID uint) ([]Endpoint, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/labels/%d/hosts", labelID)
	resp, err := fc.makeFleetRequest("GET", endpointPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("label hosts status %d", resp.StatusCode)
	}
	var result struct {
		Hosts []Endpoint `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Hosts, nil
}

// runAdHocSingleHost uses POST /api/v1/fleet/hosts/:id/query (Fleet 4.43+ synchronous REST).
func (fc *FleetClient) runAdHocSingleHost(hostID uint, sql string, endpointByID map[uint]Endpoint) (*LiveQueryResult, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/%d/query", hostID)
	resp, err := fc.makeFleetRequest("POST", endpointPath, AdHocQueryRequest{Query: sql})
	if err != nil {
		return nil, fmt.Errorf("ad hoc query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ad hoc query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var adHoc AdHocQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&adHoc); err != nil {
		return nil, fmt.Errorf("failed to decode ad hoc query response: %w", err)
	}

	hostName := ""
	if ep, ok := endpointByID[hostID]; ok {
		hostName = ep.DisplayName
		if hostName == "" {
			hostName = ep.Name
		}
	}

	resultRow := map[string]interface{}{
		"host_id":   hostID,
		"host_name": hostName,
		"status":    adHoc.Status,
		"rows":      adHoc.Rows,
	}
	if adHoc.Error != nil {
		resultRow["error"] = *adHoc.Error
	}

	respondedCount := 0
	if adHoc.Status == "online" {
		respondedCount = 1
	}
	return &LiveQueryResult{
		TargetedHostCount:  1,
		RespondedHostCount: respondedCount,
		Results:            []map[string]interface{}{resultRow},
	}, nil
}

// runMultiHostQuery creates a temporary saved query, runs it by ID, then deletes it.
// Uses POST /api/v1/fleet/queries/:id/run (Fleet 4.43+ synchronous REST).
func (fc *FleetClient) runMultiHostQuery(hostIDs []uint, sql string, endpointByID map[uint]Endpoint) (*LiveQueryResult, error) {
	tempName := fmt.Sprintf("fleet-mcp-temp-%d", time.Now().UnixMilli())
	savedQuery, err := fc.CreateSavedQuery(tempName, "Temporary MCP live query", sql, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary query: %w", err)
	}
	defer func() {
		delEndpoint := fmt.Sprintf("/api/v1/fleet/reports/id/%d", savedQuery.ID)
		r, _ := fc.makeFleetRequest("DELETE", delEndpoint, nil)
		if r != nil {
			r.Body.Close()
		}
	}()

	logrus.Infof("Created temp query ID=%d, running against %d hosts", savedQuery.ID, len(hostIDs))

	runEndpoint := fmt.Sprintf("/api/v1/fleet/reports/%d/run", savedQuery.ID)
	resp, err := fc.makeFleetRequest("POST", runEndpoint, MultiQueryRunRequest{HostIDs: hostIDs})
	if err != nil {
		return nil, fmt.Errorf("failed to run live query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("live query run failed with status %d: %s", resp.StatusCode, string(body))
	}

	var runResp MultiQueryRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("failed to decode live query run response: %w", err)
	}

	var enriched []map[string]interface{}
	for _, r := range runResp.Results {
		row := map[string]interface{}{
			"host_id": r.HostID,
			"rows":    r.Rows,
		}
		if ep, ok := endpointByID[r.HostID]; ok {
			name := ep.DisplayName
			if name == "" {
				name = ep.Name
			}
			row["host_name"] = name
		}
		if r.Error != nil {
			row["error"] = *r.Error
		}
		enriched = append(enriched, row)
	}

	return &LiveQueryResult{
		TargetedHostCount:  runResp.TargetedHostCount,
		RespondedHostCount: runResp.RespondedHostCount,
		Results:            enriched,
	}, nil
}

func (fc *FleetClient) makeFleetRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", fc.baseURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fc.apiKey)

	logrus.Debugf("%s %s", method, endpoint)

	resp, err := fc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Fleet API: %w", err)
	}

	return resp, nil
}
