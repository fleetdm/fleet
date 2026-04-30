package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// teamFanOutConcurrency caps the number of in-flight per-team API calls during
// GetQueries / GetPolicies. Sequential per-team calls scale O(N teams) and
// dominate the latency on enterprise Fleets with 50+ teams; concurrency lets
// us amortize the round-trip count without overwhelming Fleet with thousands
// of simultaneous requests.
const teamFanOutConcurrency = 8

// tempQueryNamePrefix is the prefix used by all transient saved queries created
// by runMultiHostQuery. Sweeping leftover queries at startup uses this prefix
// to find them.
const tempQueryNamePrefix = "fleet-mcp-temp-"

// randomHexSuffix returns a hex-encoded random string for unique temp-query
// names. Falls back to time.Now().UnixNano() if crypto/rand is unavailable
// (extremely unlikely, but the fallback keeps runMultiHostQuery functional).
func randomHexSuffix(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

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
		// Hard gate: refuse to start when FLEET_TLS_SKIP_VERIFY is paired with a
		// non-loopback URL. Allowing this on a remote URL means an on-path attacker
		// can present any TLS cert and capture the admin Fleet API token in one
		// handshake. Localhost is the only safe context for skip-verify.
		if !isLoopbackURL(baseURL) {
			logrus.Fatalf("FLEET_TLS_SKIP_VERIFY=true is only allowed when FLEET_BASE_URL points at localhost (got %s); refuse to start. Remove FLEET_TLS_SKIP_VERIFY or set FLEET_CA_FILE to a trusted PEM instead.", baseURL)
		}
		logrus.Warn("TLS certificate verification is disabled — localhost only; do not use in production")
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
	ID             uint        `json:"id"`
	Name           string      `json:"hostname"`
	DisplayName    string      `json:"display_name"`
	ComputerName   string      `json:"computer_name"`
	Status         string      `json:"status"`
	LastSeen       int64       `json:"last_seen"`
	Platform       string      `json:"platform"`
	Version        string      `json:"osquery_version"`
	HardwareSerial string      `json:"hardware_serial"`
	PrimaryIP      string      `json:"primary_ip"`
	TeamID         *uint       `json:"team_id"`
	TeamName       string      `json:"team_name"`
	Labels         []HostLabel `json:"labels"`
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

// HostPolicyEntry represents one policy result attached to a host when the
// hosts endpoint is called with populate_policies=true. The Response field
// is the per-host pass/fail outcome ("pass", "fail", or "" for not-yet-run).
type HostPolicyEntry struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	Description string `json:"description"`
	Resolution  string `json:"resolution"`
	Platform    string `json:"platform"`
	Critical    bool   `json:"critical"`
	Response    string `json:"response"`
}

// HostWithPolicies is a host listing enriched with the per-host policy
// compliance array. Returned by GetHostByIdentifierWithPolicies.
//
// JSON shape preserves the existing Endpoint fields and adds a top-level
// "policies" array — backward compatible with consumers that only read
// Endpoint fields, additive for consumers that want the policies.
type HostWithPolicies struct {
	Endpoint
	Policies []HostPolicyEntry `json:"policies"`
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
func (fc *FleetClient) GetEndpoints(ctx context.Context, perPage int) ([]Endpoint, error) {
	params := url.Values{}
	params.Set("populate_labels", "true")
	if perPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", perPage))
	}
	endpoint := "/api/v1/fleet/hosts?" + params.Encode()
	resp, err := fc.makeFleetRequest(ctx, "GET", endpoint, nil)
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
func (fc *FleetClient) GetHostByIdentifier(ctx context.Context, identifier string) (*Endpoint, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/identifier/%s", url.PathEscape(identifier))
	resp, err := fc.makeFleetRequest(ctx, "GET", endpointPath, nil)
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

// GetHostByID fetches a host by its numeric Fleet ID. Wraps
// GET /api/v1/fleet/hosts/:host_id. Use this when the caller already has a
// concrete host_id (e.g. from a prior candidate list) — the identifier
// endpoint can silently return the wrong host when multiple hosts share a
// hostname, but :host_id is unambiguous.
func (fc *FleetClient) GetHostByID(ctx context.Context, hostID uint) (*Endpoint, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID)
	resp, err := fc.makeFleetRequest(ctx, "GET", endpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get host by id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("host not found: id=%d", hostID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get host by id: status code %d", resp.StatusCode)
	}

	var result struct {
		Host Endpoint `json:"host"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode host by id response: %w", err)
	}
	return &result.Host, nil
}

// GetHostByIDWithPolicies fetches a host by numeric ID together with the
// per-host policy compliance array. Same disambiguation guarantee as
// GetHostByID — :host_id never collides on shared hostnames.
func (fc *FleetClient) GetHostByIDWithPolicies(ctx context.Context, hostID uint) (*HostWithPolicies, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/%d?populate_policies=true", hostID)
	resp, err := fc.makeFleetRequest(ctx, "GET", endpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get host with policies by id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("host not found: id=%d", hostID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get host with policies by id: status code %d", resp.StatusCode)
	}

	var result struct {
		Host HostWithPolicies `json:"host"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode host with policies by id response: %w", err)
	}
	return &result.Host, nil
}

// GetHostByIdentifierWithPolicies fetches a host's full details together with
// every policy that applies to it (global + fleet-inherited), each entry
// carrying its pass/fail/empty Response. Wraps GET
// /api/v1/fleet/hosts/identifier/:identifier?populate_policies=true.
//
// This is the single API call behind the per-host "Policies" tab in the
// Fleet UI — answer "is this host compliant?" with one call instead of
// listing all policies and scanning by host.
func (fc *FleetClient) GetHostByIdentifierWithPolicies(ctx context.Context, identifier string) (*HostWithPolicies, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/identifier/%s?populate_policies=true", url.PathEscape(identifier))
	resp, err := fc.makeFleetRequest(ctx, "GET", endpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get host with policies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("host not found: %s", identifier)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get host with policies: status code %d", resp.StatusCode)
	}

	var result struct {
		Host HostWithPolicies `json:"host"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode host with policies response: %w", err)
	}
	return &result.Host, nil
}

// GetQueries retrieves global and all team-specific queries from Fleet.
func (fc *FleetClient) GetQueries(ctx context.Context) ([]Query, error) {
	resp, err := fc.makeFleetRequest(ctx, "GET", "/api/v1/fleet/reports", nil)
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

	teams, err := fc.GetTeams(ctx)
	if err != nil {
		logrus.Warnf("skipping team queries: %v", err)
		return result.Queries, nil
	}
	// Concurrent per-team fan-out (bounded by teamFanOutConcurrency). Each
	// goroutine writes its slice into a shared chunk that is merged after
	// all workers finish — no shared-slice mutation, no lock contention.
	type chunk struct {
		idx     int
		queries []Query
	}
	chunks := make(chan chunk, len(teams))
	sem := make(chan struct{}, teamFanOutConcurrency)
	var wg sync.WaitGroup
	for i, team := range teams {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, team Team) {
			defer wg.Done()
			defer func() { <-sem }()
			teamResp, err := fc.makeFleetRequest(ctx, "GET", fmt.Sprintf("/api/v1/fleet/reports?team_id=%d", team.ID), nil)
			if err != nil {
				logrus.Warnf("team %d queries error: %v", team.ID, err)
				return
			}
			defer teamResp.Body.Close()
			if teamResp.StatusCode != http.StatusOK {
				logrus.Warnf("team %d queries: status %d", team.ID, teamResp.StatusCode)
				return
			}
			var tr struct {
				Queries []Query `json:"queries"`
			}
			if derr := json.NewDecoder(teamResp.Body).Decode(&tr); derr != nil {
				logrus.Warnf("team %d queries decode failed: %v", team.ID, derr)
				return
			}
			for i := range tr.Queries {
				tr.Queries[i].Name = fmt.Sprintf("[%s] %s", team.Name, tr.Queries[i].Name)
			}
			chunks <- chunk{idx: idx, queries: tr.Queries}
		}(i, team)
	}
	wg.Wait()
	close(chunks)
	// Order-stable merge: collect chunks, then sort by team index so the
	// emitted slice has a deterministic shape per Fleet config.
	chunkSlice := make([]chunk, 0, len(teams))
	for c := range chunks {
		chunkSlice = append(chunkSlice, c)
	}
	sort.Slice(chunkSlice, func(i, j int) bool { return chunkSlice[i].idx < chunkSlice[j].idx })
	for _, c := range chunkSlice {
		result.Queries = append(result.Queries, c.queries...)
	}
	return result.Queries, nil
}

// GetPolicies retrieves global and all team-specific policies from Fleet.
func (fc *FleetClient) GetPolicies(ctx context.Context) ([]Policy, error) {
	resp, err := fc.makeFleetRequest(ctx, "GET", "/api/v1/fleet/global/policies", nil)
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

	teams, err := fc.GetTeams(ctx)
	if err != nil {
		logrus.Warnf("skipping team policies: %v", err)
		return result.Policies, nil
	}
	// Concurrent per-team fan-out (same pattern as GetQueries).
	type chunk struct {
		idx      int
		policies []Policy
	}
	chunks := make(chan chunk, len(teams))
	sem := make(chan struct{}, teamFanOutConcurrency)
	var wg sync.WaitGroup
	for i, team := range teams {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, team Team) {
			defer wg.Done()
			defer func() { <-sem }()
			teamResp, err := fc.makeFleetRequest(ctx, "GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team.ID), nil)
			if err != nil {
				logrus.Warnf("team %d policies error: %v", team.ID, err)
				return
			}
			defer teamResp.Body.Close()
			if teamResp.StatusCode != http.StatusOK {
				logrus.Warnf("team %d policies: status %d", team.ID, teamResp.StatusCode)
				return
			}
			var tr struct {
				Policies []Policy `json:"policies"`
			}
			if derr := json.NewDecoder(teamResp.Body).Decode(&tr); derr != nil {
				logrus.Warnf("team %d policies decode failed: %v", team.ID, derr)
				return
			}
			for i := range tr.Policies {
				tr.Policies[i].Name = fmt.Sprintf("[%s] %s", team.Name, tr.Policies[i].Name)
			}
			chunks <- chunk{idx: idx, policies: tr.Policies}
		}(i, team)
	}
	wg.Wait()
	close(chunks)
	chunkSlice := make([]chunk, 0, len(teams))
	for c := range chunks {
		chunkSlice = append(chunkSlice, c)
	}
	sort.Slice(chunkSlice, func(i, j int) bool { return chunkSlice[i].idx < chunkSlice[j].idx })
	for _, c := range chunkSlice {
		result.Policies = append(result.Policies, c.policies...)
	}
	return result.Policies, nil
}

// GetLabels retrieves all labels from Fleet
func (fc *FleetClient) GetLabels(ctx context.Context) ([]Label, error) {
	endpoint := "/api/v1/fleet/labels"
	resp, err := fc.makeFleetRequest(ctx, "GET", endpoint, nil)
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
func (fc *FleetClient) GetFleetConfig(ctx context.Context) (map[string]interface{}, error) {
	resp, err := fc.makeFleetRequest(ctx, "GET", "/api/v1/fleet/config", nil)
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

// GetEndpointsWithAggregations returns the platform breakdown for the entire
// Fleet using /api/v1/fleet/host_summary, which Fleet computes server-side
// over the full inventory. The previous implementation called GetEndpoints(0)
// which silently truncated to Fleet's default 100-host page — wrong on any
// Fleet larger than that. host_summary is the correct dedicated endpoint.
func (fc *FleetClient) GetEndpointsWithAggregations(ctx context.Context) (*AggregateResponse, error) {
	resp, err := fc.makeFleetRequest(ctx, "GET", "/api/v1/fleet/host_summary", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get host summary: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get host summary: status %d", resp.StatusCode)
	}
	var summary struct {
		TotalsHostsCount int `json:"totals_hosts_count"`
		Platforms        []struct {
			Platform   string `json:"platform"`
			HostsCount int    `json:"hosts_count"`
		} `json:"platforms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode host summary: %w", err)
	}

	platformBreakdown := PlatformBreakdown{}
	for _, p := range summary.Platforms {
		switch p.Platform {
		case "darwin":
			platformBreakdown.MacOS += p.HostsCount
		case "windows":
			platformBreakdown.Windows += p.HostsCount
		case "linux", "ubuntu", "centos", "rhel", "debian", "fedora", "amzn":
			platformBreakdown.Linux += p.HostsCount
		case "chrome":
			platformBreakdown.ChromeOS += p.HostsCount
		case "ios":
			platformBreakdown.IOS += p.HostsCount
		case "ipados":
			platformBreakdown.IPadOS += p.HostsCount
		case "android":
			platformBreakdown.Android += p.HostsCount
		default:
			platformBreakdown.Other += p.HostsCount
		}
	}
	platformBreakdown.Total = summary.TotalsHostsCount

	return &AggregateResponse{
		Count: summary.TotalsHostsCount,
		Data: map[string]interface{}{
			"platform_breakdown": platformBreakdown,
			"total_count":        summary.TotalsHostsCount,
		},
	}, nil
}

// GetTeams retrieves all teams from Fleet
func (fc *FleetClient) GetTeams(ctx context.Context) ([]Team, error) {
	endpoint := "/api/v1/fleet/teams"
	resp, err := fc.makeFleetRequest(ctx, "GET", endpoint, nil)
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
func (fc *FleetClient) GetHostCount(ctx context.Context) (int, error) {
	resp, err := fc.makeFleetRequest(ctx, "GET", "/api/v1/fleet/hosts/count", nil)
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

// resolveLabelName resolves a label name to its numeric ID using exact
// case-insensitive matching. On failure, lists available labels so the
// caller can retry. Mirrors resolveTeamNames — no caching, calls GetLabels()
// each invocation. Labels lists are small on dogfood so the cost is
// negligible and the code stays parallel with the team resolver.
func (fc *FleetClient) resolveLabelName(ctx context.Context, name string) (uint, error) {
	labels, err := fc.GetLabels(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get labels: %w", err)
	}
	target := strings.ToLower(strings.TrimSpace(name))
	availableNames := make([]string, 0, len(labels))
	for _, l := range labels {
		availableNames = append(availableNames, l.Name)
		if strings.ToLower(l.Name) == target {
			return l.ID, nil
		}
	}
	return 0, fmt.Errorf("label not found: %q (available labels: %s)", name, strings.Join(availableNames, ", "))
}

// resolveTeamNames resolves team names to team IDs using exact case-insensitive
// matching. On failure, lists available teams so the caller can retry.
func (fc *FleetClient) resolveTeamNames(ctx context.Context, teamNames []string) ([]uint, error) {
	teams, err := fc.GetTeams(ctx)
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

// fetchHostsHardCap is the safety ceiling on a single paginated fetch. Fleet
// inventories of 50k hosts × ~2KB per Endpoint = ~100MB in memory per call —
// without this cap a runaway filter (or a Fleet that ignores a filter and
// returns the full inventory) can OOM the MCP. Callers can tune via
// fetchHostsFromPathBounded.
const fetchHostsHardCap = 10000

// fetchHostsFromPath issues GETs against an arbitrary Fleet hosts-listing
// path (e.g. /api/v1/fleet/hosts?... or /api/v1/fleet/labels/:id/hosts?...)
// and decodes the {hosts: [...]} envelope. Paginates server-side via ?page=N
// until either the upstream returns a short page (last page) or the hard cap
// is reached. The path's existing per_page (if any) sets the page size; this
// function manages ?page= itself.
//
// ctx propagation: caller cancellation stops the fan-out between pages —
// long-running multi-page fetches (label intersection, CVE compose) honor
// MCP request cancellation rather than running every page to completion.
func (fc *FleetClient) fetchHostsFromPath(ctx context.Context, path string) ([]Endpoint, error) {
	return fc.fetchHostsFromPathBounded(ctx, path, fetchHostsHardCap)
}

// fetchHostsFromPathBounded is the paginating worker behind fetchHostsFromPath.
// hardCap <= 0 falls back to fetchHostsHardCap. When the cap is hit we log a
// warning so operators see truncation rather than silently returning a partial
// host list.
func (fc *FleetClient) fetchHostsFromPathBounded(ctx context.Context, path string, hardCap int) ([]Endpoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if hardCap <= 0 {
		hardCap = fetchHostsHardCap
	}
	base, query, _ := strings.Cut(path, "?")
	params, err := url.ParseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hosts path query: %w", err)
	}
	perPage := 500
	if v := params.Get("per_page"); v != "" {
		if n, perr := strconv.Atoi(v); perr == nil && n > 0 && n <= 1000 {
			perPage = n
		}
	}
	params.Set("per_page", strconv.Itoa(perPage))

	out := make([]Endpoint, 0, perPage)
	for page := 0; ; page++ {
		// Honor caller cancellation between paginated requests.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		params.Set("page", strconv.Itoa(page))
		resp, err := fc.makeFleetRequest(ctx, "GET", base+"?"+params.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch hosts: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch hosts: status %d", resp.StatusCode)
		}
		var result struct {
			Hosts []Endpoint `json:"hosts"`
		}
		decErr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if decErr != nil {
			return nil, fmt.Errorf("failed to decode hosts response: %w", decErr)
		}
		out = append(out, result.Hosts...)
		// Last page — Fleet returned fewer than the requested page size.
		if len(result.Hosts) < perPage {
			break
		}
		// Hard cap hit — log and stop. Truncation here is rare; when it
		// happens the operator should either tighten filters or raise the cap.
		if len(out) >= hardCap {
			logrus.Warnf("fleet host fetch hit hard cap %d (path=%s) — result truncated; tighten filters or raise fetchHostsHardCap", hardCap, base)
			break
		}
	}
	return out, nil
}

// resolvePlatformOrLabelToLabelID picks a single Fleet label_id from EITHER
// labelName or platform — used because Fleet's /hosts endpoint silently
// IGNORES both ?platform= and ?label_id= filter params, so the only way to
// scope by label or platform is to call /api/v1/fleet/labels/:id/hosts
// directly. labelName takes precedence when both are set.
//
// Returns (labelID, true, nil) when label resolution succeeds, (_, false, nil)
// when neither argument is set, or (_, false, err) on resolution failure.
func (fc *FleetClient) resolvePlatformOrLabelToLabelID(ctx context.Context, labelName, platform string) (uint, bool, error) {
	if labelName != "" {
		id, err := fc.resolveLabelName(ctx, labelName)
		if err != nil {
			return 0, false, err
		}
		return id, true, nil
	}
	if platform != "" {
		builtin := platformToBuiltinLabel(platform)
		if builtin == "" {
			return 0, false, fmt.Errorf("unsupported platform %q (use one of: macos, windows, linux, chromeos)", platform)
		}
		id, err := fc.resolveLabelName(ctx, builtin)
		if err != nil {
			return 0, false, fmt.Errorf("failed to resolve built-in label for platform %q: %w", platform, err)
		}
		return id, true, nil
	}
	return 0, false, nil
}

// GetEndpointsWithFilters retrieves endpoints from Fleet with optional
// server-side filters.
//
// IMPORTANT — Fleet API quirks this function works around:
//
//   - /api/v1/fleet/hosts SILENTLY IGNORES ?platform= and ?label_id= query
//     params (any value returns the unfiltered host list). To filter by
//     label or platform we must call /api/v1/fleet/labels/:label_id/hosts
//     instead — that endpoint actually scopes results.
//   - /api/v1/fleet/labels/:id/hosts in turn IGNORES ?policy_id= and
//     ?software_version_id= filters but DOES respect ?team_id= / ?status= /
//     ?query=. So when label/platform AND policy_id/policy_response are
//     combined, we fetch both sets and intersect by host ID.
//   - /api/v1/fleet/hosts respects ?team_id, ?status, ?query, ?policy_id,
//     ?policy_response, ?software_version_id — used for the no-label path
//     and as the policy side of the intersection.
//
// query is a free-text substring matched case-insensitively against
// hostname / hardware_serial / primary_ip / hardware_model / user inventory
// (username / email / IdP group). Empty to skip.
//
// labelName takes precedence over platform when both are set; platform is
// translated to its built-in label name (macOS, MS Windows, All Linux, etc.)
// and resolved to a label_id for the same routing.
//
// policyResponse without policyID is rejected here as a sanity check.
func (fc *FleetClient) GetEndpointsWithFilters(ctx context.Context, teamName, platform, status, query, labelName, policyID, policyResponse string, perPage int) ([]Endpoint, error) {
	if policyResponse != "" && policyID == "" {
		return nil, fmt.Errorf("policy_response is only valid when policy_id is also set")
	}
	if policyResponse != "" && policyResponse != "passing" && policyResponse != "failing" {
		return nil, fmt.Errorf("policy_response must be 'passing' or 'failing', got %q", policyResponse)
	}

	// Resolve team name → team ID once (used in every branch below).
	var teamIDStr string
	if teamName != "" {
		teamIDs, err := fc.resolveTeamNames(ctx, []string{teamName})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve fleet: %w", err)
		}
		teamIDStr = fmt.Sprintf("%d", teamIDs[0])
	}

	// Decide whether label-based routing is needed.
	labelID, viaLabel, err := fc.resolvePlatformOrLabelToLabelID(ctx, labelName, platform)
	if err != nil {
		return nil, err
	}

	// Path 1: no label/platform — single /hosts call with all filters server-side.
	if !viaLabel {
		params := url.Values{}
		params.Set("populate_labels", "true")
		if perPage > 0 {
			params.Set("per_page", fmt.Sprintf("%d", perPage))
		}
		if teamIDStr != "" {
			params.Set("team_id", teamIDStr)
		}
		if status != "" {
			params.Set("status", status)
		}
		if q := strings.TrimSpace(query); q != "" {
			params.Set("query", q)
		}
		if policyID != "" {
			params.Set("policy_id", policyID)
		}
		if policyResponse != "" {
			params.Set("policy_response", policyResponse)
		}
		hosts, err := fc.fetchHostsFromPath(ctx, "/api/v1/fleet/hosts?"+params.Encode())
		if err != nil {
			return nil, err
		}
		if perPage > 0 && len(hosts) > perPage {
			hosts = hosts[:perPage]
		}
		return hosts, nil
	}

	// Path 2: label/platform routing — call /labels/:id/hosts with the
	// filters that endpoint respects. Use a generous per_page so that
	// downstream client-side intersection (if any) has the full label set.
	labelParams := url.Values{}
	labelParams.Set("populate_labels", "true")
	if teamIDStr != "" {
		labelParams.Set("team_id", teamIDStr)
	}
	if status != "" {
		labelParams.Set("status", status)
	}
	if q := strings.TrimSpace(query); q != "" {
		labelParams.Set("query", q)
	}
	// Always pull a wide page from the label endpoint — intersection may
	// reduce the count, and the label endpoint doesn't honor most filters.
	labelParams.Set("per_page", "500")
	labelHosts, err := fc.fetchHostsFromPath(ctx, fmt.Sprintf("/api/v1/fleet/labels/%d/hosts?%s", labelID, labelParams.Encode()))
	if err != nil {
		return nil, err
	}

	// No policy filter → cap, enrich, return.
	//
	// Why enrich: Fleet's /labels/:id/hosts silently ignores populate_labels=true
	// (verified empirically) — every host in labelHosts has Labels=nil. The
	// MCP contract is to return hosts with their Labels populated, so we
	// hydrate via per-host /api/v1/fleet/hosts/:id calls (concurrent, bounded).
	if policyID == "" {
		if perPage > 0 && len(labelHosts) > perPage {
			labelHosts = labelHosts[:perPage]
		}
		fc.enrichHostLabels(ctx, labelHosts)
		return labelHosts, nil
	}

	// Path 3: label + policy combo — fetch policy side via /hosts (which DOES
	// honor populate_labels), then intersect against the label-side ID set.
	//
	// Why iterate policyHosts (not labelHosts): /hosts populates Labels;
	// /labels/:id/hosts does not. Picking from policyHosts means the result
	// already has Labels — no per-host enrichment needed for Path 3.
	policyParams := url.Values{}
	policyParams.Set("policy_id", policyID)
	policyParams.Set("populate_labels", "true")
	if policyResponse != "" {
		policyParams.Set("policy_response", policyResponse)
	}
	if teamIDStr != "" {
		policyParams.Set("team_id", teamIDStr)
	}
	if status != "" {
		policyParams.Set("status", status)
	}
	if q := strings.TrimSpace(query); q != "" {
		policyParams.Set("query", q)
	}
	policyParams.Set("per_page", "500")
	policyHosts, err := fc.fetchHostsFromPath(ctx, "/api/v1/fleet/hosts?"+policyParams.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch policy host set for intersection: %w", err)
	}

	labelIDs := make(map[uint]bool, len(labelHosts))
	for _, h := range labelHosts {
		labelIDs[h.ID] = true
	}

	intersected := make([]Endpoint, 0)
	for _, h := range policyHosts {
		if !labelIDs[h.ID] {
			continue
		}
		intersected = append(intersected, h)
		if perPage > 0 && len(intersected) >= perPage {
			break
		}
	}
	return intersected, nil
}

// enrichHostLabels populates each host's Labels field via per-host detail
// fetches when Labels is nil. Used after /labels/:id/hosts (which silently
// ignores populate_labels=true and leaves Labels unpopulated). Idempotent:
// hosts that already carry Labels are skipped, so callers can invoke
// liberally without re-fetching.
//
// Concurrency is bounded so a 200-host result doesn't fan out to 200
// in-flight Fleet API calls. ctx propagation means MCP-level cancellation
// stops in-flight enrichment promptly. Per-host failures are logged but do
// not abort the whole call — the caller still gets the original list, just
// with some hosts missing labels.
func (fc *FleetClient) enrichHostLabels(ctx context.Context, hosts []Endpoint) {
	const enrichConcurrency = 8
	sem := make(chan struct{}, enrichConcurrency)
	var wg sync.WaitGroup
	for i := range hosts {
		if hosts[i].Labels != nil {
			continue
		}
		if err := ctx.Err(); err != nil {
			return
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			full, err := fc.GetHostByID(ctx, hosts[idx].ID)
			if err != nil {
				logrus.Warnf("enrichHostLabels: failed to fetch host %d: %v", hosts[idx].ID, err)
				return
			}
			hosts[idx].Labels = full.Labels
		}(i)
	}
	wg.Wait()
}

// GetPolicyCompliance retrieves policy compliance data
func (fc *FleetClient) GetPolicyCompliance(ctx context.Context, policyID string) (*PolicyCompliance, error) {
	endpoint := fmt.Sprintf("/api/v1/fleet/global/policies/%s", url.PathEscape(policyID))
	resp, err := fc.makeFleetRequest(ctx, "GET", endpoint, nil)
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

// GetTeamPolicyCompliance retrieves policy compliance scoped to a single fleet
// (team). Wraps GET /api/v1/fleet/teams/:team_id/policies/:policy_id and
// returns the same PolicyCompliance shape as the global variant so callers
// can treat both uniformly. Use this — not GetPolicyCompliance — when the
// caller knows the policy belongs to a specific fleet, or when global counts
// would be misleading because the policy is fleet-scoped.
func (fc *FleetClient) GetTeamPolicyCompliance(ctx context.Context, teamID, policyID string) (*PolicyCompliance, error) {
	endpoint := fmt.Sprintf("/api/v1/fleet/teams/%s/policies/%s", url.PathEscape(teamID), url.PathEscape(policyID))
	resp, err := fc.makeFleetRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get team policy compliance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get team policy compliance: status code %d", resp.StatusCode)
	}

	var result struct {
		Policy struct {
			ID               uint   `json:"id"`
			Name             string `json:"name"`
			PassingHostCount int    `json:"passing_host_count"`
			FailingHostCount int    `json:"failing_host_count"`
		} `json:"policy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode team policy compliance response: %w", err)
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

// GetVulnerabilityImpact retrieves vulnerability impact data.
//
// The aggregate count is derived from the SAME fan-out used by
// GetHostsForCVE (software titles → versions → /hosts?software_version_id=N,
// deduped by host ID) so the count returned here matches the host list
// returned by get_vulnerability_hosts byte-for-byte.
//
// Why not Fleet's /api/v1/fleet/vulnerabilities/:cve.hosts_count? That field
// is populated by Fleet's vuln-aggregation cron, which runs less frequently
// than software inventory. In practice the aggregate trails the software
// inventory by minutes-to-hours, so the two values disagree on freshly
// vulnerable hosts. Sharing the fan-out path keeps impact and host listing
// numerically consistent — the price is N+1 extra HTTP calls per CVE, which
// is fine because impact is a low-frequency operation.
func (fc *FleetClient) GetVulnerabilityImpact(ctx context.Context, cveID string) (*VulnerabilityImpact, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cveID) == "" {
		return nil, fmt.Errorf("cve_id is required")
	}

	// Reuse GetHostsForCVE with no filters and no per-page cap so the count
	// is the full impacted set, not the truncated tool-response cap.
	hosts, err := fc.GetHostsForCVE(ctx, cveID, "", "", "", "", "", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to compute vulnerability impact: %w", err)
	}

	totalSystems := 0
	if count, err := fc.GetHostCount(ctx); err == nil {
		totalSystems = count
	}

	return &VulnerabilityImpact{
		CVEID:           cveID,
		TotalSystems:    totalSystems,
		ImpactedSystems: len(hosts),
	}, nil
}

// GetHostsForCVE returns the specific hosts impacted by a CVE, optionally
// narrowed by team / platform / status / query (substring) / label name.
//
// Fleet's /hosts endpoint silently IGNORES ?cve= so we can't filter hosts
// by CVE directly. This composes three steps server-side:
//
//  1. /api/v1/fleet/software/titles?vulnerable=true&query=CVE-X[&team_id=N]
//     → list of software titles affected by the CVE (in the team if scoped).
//  2. /api/v1/fleet/software/titles/:title_id[?team_id=N]
//     → version IDs of the title.
//  3. /api/v1/fleet/hosts?software_version_id=V[&team_id=N&status=...&query=...]
//     → the actual hosts with that vulnerable version. /hosts respects this
//     filter (unlike ?cve= or ?platform= or ?label_id=).
//
// platform / labelName trigger client-side post-filtering on the final host
// list using each host's populate_labels=true label array — Fleet's /hosts
// endpoint silently ignores these filter params, so we can't push them
// server-side and have to verify membership locally.
//
// Use this — NOT GetVulnerabilityImpact — when callers need the actual
// host list. GetVulnerabilityImpact only returns an aggregate count.
func (fc *FleetClient) GetHostsForCVE(ctx context.Context, cveID, teamName, platform, status, query, labelName string, perPage int) ([]Endpoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cveID) == "" {
		return nil, fmt.Errorf("cve_id is required")
	}

	// Resolve team once.
	var teamIDStr string
	if teamName != "" {
		teamIDs, err := fc.resolveTeamNames(ctx, []string{teamName})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve fleet: %w", err)
		}
		teamIDStr = fmt.Sprintf("%d", teamIDs[0])
	}

	// Step 1: software titles affected by this CVE (optionally team-scoped).
	titleParams := url.Values{}
	titleParams.Set("vulnerable", "true")
	titleParams.Set("query", strings.TrimSpace(cveID))
	titleParams.Set("per_page", "100")
	if teamIDStr != "" {
		titleParams.Set("team_id", teamIDStr)
	}
	titleResp, err := fc.makeFleetRequest(ctx, "GET", "/api/v1/fleet/software/titles?"+titleParams.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get software titles for CVE: %w", err)
	}
	defer titleResp.Body.Close()
	if titleResp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("CVE not found: %s", cveID)
	}
	if titleResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get software titles for CVE: status %d", titleResp.StatusCode)
	}
	var titlesResult struct {
		SoftwareTitles []struct {
			ID uint `json:"id"`
		} `json:"software_titles"`
	}
	if err := json.NewDecoder(titleResp.Body).Decode(&titlesResult); err != nil {
		return nil, fmt.Errorf("failed to decode software titles response: %w", err)
	}
	if len(titlesResult.SoftwareTitles) == 0 {
		return []Endpoint{}, nil
	}

	// Step 2: per title, fetch detail to get version IDs.
	versionIDs := make([]uint, 0)
	for _, t := range titlesResult.SoftwareTitles {
		detailURL := fmt.Sprintf("/api/v1/fleet/software/titles/%d", t.ID)
		if teamIDStr != "" {
			detailURL += "?team_id=" + teamIDStr
		}
		// Honor caller cancellation between fan-out iterations: a slow CVE with
		// many vulnerable titles can issue dozens of HTTP calls, so checking
		// ctx between each one means a cancelled MCP request stops promptly.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		detailResp, dErr := fc.makeFleetRequest(ctx, "GET", detailURL, nil)
		if dErr != nil {
			logrus.Warnf("failed to fetch software title %d detail: %v", t.ID, dErr)
			continue
		}
		if detailResp.StatusCode != http.StatusOK {
			detailResp.Body.Close()
			logrus.Warnf("failed to fetch software title %d detail: status %d", t.ID, detailResp.StatusCode)
			continue
		}
		var detailResult struct {
			SoftwareTitle struct {
				Versions []struct {
					ID uint `json:"id"`
				} `json:"versions"`
			} `json:"software_title"`
		}
		decErr := json.NewDecoder(detailResp.Body).Decode(&detailResult)
		detailResp.Body.Close()
		if decErr != nil {
			logrus.Warnf("failed to decode software title %d detail: %v", t.ID, decErr)
			continue
		}
		for _, v := range detailResult.SoftwareTitle.Versions {
			versionIDs = append(versionIDs, v.ID)
		}
	}
	if len(versionIDs) == 0 {
		return []Endpoint{}, nil
	}

	// Step 3: per version_id, fetch hosts with composing filters server-side.
	baseParams := url.Values{}
	baseParams.Set("populate_labels", "true")
	if teamIDStr != "" {
		baseParams.Set("team_id", teamIDStr)
	}
	if status != "" {
		baseParams.Set("status", status)
	}
	if q := strings.TrimSpace(query); q != "" {
		baseParams.Set("query", q)
	}
	// Generous per_page on each fan-out; we cap the merged result at perPage.
	baseParams.Set("per_page", "500")

	seen := make(map[uint]bool)
	hosts := make([]Endpoint, 0)
	for _, vid := range versionIDs {
		// Honor caller cancellation between version-id fan-outs.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		params := url.Values{}
		for k, v := range baseParams {
			params[k] = v
		}
		params.Set("software_version_id", fmt.Sprintf("%d", vid))
		page, fErr := fc.fetchHostsFromPath(ctx, "/api/v1/fleet/hosts?"+params.Encode())
		if fErr != nil {
			logrus.Warnf("failed to fetch hosts for software_version_id=%d: %v", vid, fErr)
			continue
		}
		for _, h := range page {
			if seen[h.ID] {
				continue
			}
			seen[h.ID] = true
			hosts = append(hosts, h)
		}
	}

	// Step 4: client-side platform / label post-filter (Fleet's /hosts endpoint
	// silently ignores ?platform= and ?label_id= so we can't push these
	// dimensions server-side; populate_labels=true gives us the data needed
	// to verify membership locally).
	if platform != "" || labelName != "" {
		hosts = filterHostsByPlatformOrLabel(hosts, platform, labelName)
	}

	if perPage > 0 && len(hosts) > perPage {
		hosts = hosts[:perPage]
	}
	return hosts, nil
}

// filterHostsByPlatformOrLabel narrows a host list to those matching either
// a normalized platform string (e.g. "linux" matches ubuntu/rhel/debian/etc.
// per matchesPlatform) AND/OR a label name (case-insensitive exact match
// against any of the host's labels). Empty arguments are skipped — passing
// "" for both is a no-op.
func filterHostsByPlatformOrLabel(hosts []Endpoint, platform, labelName string) []Endpoint {
	target := normalizePlatform(platform)
	wantLabel := strings.ToLower(strings.TrimSpace(labelName))
	out := make([]Endpoint, 0, len(hosts))
	for _, h := range hosts {
		if platform != "" && !matchesPlatform(h.Platform, target) {
			continue
		}
		if wantLabel != "" {
			has := false
			for _, l := range h.Labels {
				if strings.ToLower(l.Name) == wantLabel {
					has = true
					break
				}
			}
			if !has {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

// CreateSavedQuery creates a new saved query in Fleet
func (fc *FleetClient) CreateSavedQuery(ctx context.Context, name, description, sql, platform string) (*Query, error) {
	endpoint := "/api/v1/fleet/reports"

	reqBody := CreateQueryRequest{
		Name:        name,
		Description: description,
		Query:       sql,
		Platform:    platform,
	}

	resp, err := fc.makeFleetRequest(ctx, "POST", endpoint, reqBody)
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
// LiveQueryTargetSpec captures every dimension that scopes a live query.
//
// Filter dimensions (Fleet / Platform / Label / Status / Query / PolicyID /
// PolicyResponse / CVEID) are AND-ed together — the result is the intersection
// of every non-empty dimension. This mirrors the routing in
// GetEndpointsWithFilters and GetHostsForCVE so live-query target resolution
// is consistent with what the host-listing tools return.
//
// Hostnames + HostIDs form an additive "named host" set. When both filter
// dimensions and explicit names are provided, the final target is the
// INTERSECTION (named hosts that also pass the filter) — this lets a caller
// say "run on these specific hosts but only if they're Linux Workstations".
//
// Legacy plural args (Labels, Platforms, Fleets) are accepted for backward
// compatibility and processed as a per-dimension union of the FIRST item
// from each list — multi-label / multi-platform / multi-team intersection is
// not supported by Fleet's label/team endpoints in a single round trip and
// the union-then-intersect pattern was the source of the original
// "everyone gets queried" scope-bloat bug. Callers should prefer the
// singular Fleet / Platform / Label fields for new code.
type LiveQueryTargetSpec struct {
	Fleet          string
	Platform       string
	Label          string
	Status         string
	Query          string
	PolicyID       string
	PolicyResponse string
	CVEID          string
	Hostnames      []string
	HostIDs        []uint

	// Legacy / deprecated — first item only.
	LegacyFleets    []string
	LegacyPlatforms []string
	LegacyLabels    []string
}

// ResolveLiveQueryTargets returns the exact host set that would be targeted
// by a live query given the spec. Used by both prepare_live_query (preview)
// and run_live_query (execution) so the two stay in lockstep — what the
// user previews is exactly what gets queried.
func (fc *FleetClient) ResolveLiveQueryTargets(ctx context.Context, spec LiveQueryTargetSpec) ([]Endpoint, error) {
	// Apply legacy fallbacks (first item from each plural).
	if spec.Fleet == "" && len(spec.LegacyFleets) > 0 {
		spec.Fleet = strings.TrimSpace(spec.LegacyFleets[0])
		if len(spec.LegacyFleets) > 1 {
			logrus.Warnf("multi-fleet targeting not supported in single round-trip; using first: %q", spec.Fleet)
		}
	}
	if spec.Platform == "" && len(spec.LegacyPlatforms) > 0 {
		spec.Platform = strings.TrimSpace(spec.LegacyPlatforms[0])
		if len(spec.LegacyPlatforms) > 1 {
			logrus.Warnf("multi-platform targeting not supported in single round-trip; using first: %q", spec.Platform)
		}
	}
	if spec.Label == "" && len(spec.LegacyLabels) > 0 {
		spec.Label = strings.TrimSpace(spec.LegacyLabels[0])
		if len(spec.LegacyLabels) > 1 {
			logrus.Warnf("multi-label targeting not supported in single round-trip; using first: %q", spec.Label)
		}
	}

	hasFilter := spec.Fleet != "" || spec.Platform != "" || spec.Label != "" ||
		spec.Status != "" || spec.Query != "" || spec.PolicyID != "" ||
		spec.PolicyResponse != "" || spec.CVEID != ""
	hasExplicit := len(spec.Hostnames) > 0 || len(spec.HostIDs) > 0
	if !hasFilter && !hasExplicit {
		return nil, fmt.Errorf("at least one target dimension required (fleet, platform, label, status, query, policy_id, cve_id, hostnames, or host_ids)")
	}

	const livePerPage = 500

	// Build filter set.
	var filterSet []Endpoint
	if hasFilter {
		switch {
		case spec.CVEID != "":
			cveHosts, err := fc.GetHostsForCVE(ctx, spec.CVEID, spec.Fleet, spec.Platform, spec.Status, spec.Query, spec.Label, livePerPage)
			if err != nil {
				return nil, fmt.Errorf("CVE filter resolution failed: %w", err)
			}
			filterSet = cveHosts
			// Also intersect with policy filter when both are set.
			if spec.PolicyID != "" {
				policyHosts, pErr := fc.GetEndpointsWithFilters(ctx, spec.Fleet, "", spec.Status, spec.Query, "", spec.PolicyID, spec.PolicyResponse, livePerPage)
				if pErr != nil {
					return nil, fmt.Errorf("policy filter resolution failed: %w", pErr)
				}
				filterSet = intersectHostsByID(filterSet, policyHosts)
			}
		default:
			endpointHosts, err := fc.GetEndpointsWithFilters(ctx, spec.Fleet, spec.Platform, spec.Status, spec.Query, spec.Label, spec.PolicyID, spec.PolicyResponse, livePerPage)
			if err != nil {
				return nil, fmt.Errorf("endpoint filter resolution failed: %w", err)
			}
			filterSet = endpointHosts
		}
	}

	// Build explicit set from host_ids and hostnames.
	explicitSet := make([]Endpoint, 0)
	seen := make(map[uint]bool)
	for _, id := range spec.HostIDs {
		if id == 0 {
			continue
		}
		h, err := fc.GetHostByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("host_id %d not found: %w", id, err)
		}
		if seen[h.ID] {
			continue
		}
		seen[h.ID] = true
		explicitSet = append(explicitSet, *h)
	}
	for _, raw := range spec.Hostnames {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		// Query-first to detect hostname collisions before silently picking.
		candidates, qErr := fc.GetEndpointsWithFilters(ctx, "", "", "", name, "", "", "", 50)
		var resolved *Endpoint
		switch {
		case qErr == nil && len(candidates) == 1:
			full, fErr := fc.GetHostByID(ctx, candidates[0].ID)
			if fErr == nil {
				resolved = full
			} else {
				cand := candidates[0]
				resolved = &cand
			}
		case qErr == nil && len(candidates) > 1:
			return nil, fmt.Errorf("hostname %q matches %d hosts — disambiguate with host_ids (Fleet's substring search does not cover display_name; pass numeric IDs to be unambiguous)", name, len(candidates))
		default:
			// Fall back to /hosts/identifier/:id for UUID and computer_name matches.
			h, idErr := fc.GetHostByIdentifier(ctx, name)
			if idErr != nil {
				return nil, fmt.Errorf("hostname %q not found: %w", name, idErr)
			}
			resolved = h
		}
		if seen[resolved.ID] {
			continue
		}
		seen[resolved.ID] = true
		explicitSet = append(explicitSet, *resolved)
	}

	// Combine.
	switch {
	case hasFilter && hasExplicit:
		return intersectHostsByID(explicitSet, filterSet), nil
	case hasFilter:
		return filterSet, nil
	default:
		return explicitSet, nil
	}
}

// intersectHostsByID returns hosts present in both lists, preserving the
// order of the first list. Used for label+policy and CVE+policy intersection.
func intersectHostsByID(a, b []Endpoint) []Endpoint {
	bIDs := make(map[uint]bool, len(b))
	for _, h := range b {
		bIDs[h.ID] = true
	}
	out := make([]Endpoint, 0)
	seen := make(map[uint]bool)
	for _, h := range a {
		if !bIDs[h.ID] || seen[h.ID] {
			continue
		}
		seen[h.ID] = true
		out = append(out, h)
	}
	return out
}

func (fc *FleetClient) RunLiveQuery(ctx context.Context, sql string, hostnames, labels, platforms, teams []string) (*LiveQueryResult, error) {
	// Legacy entry point — preserved so existing callers keep working.
	// New code should use RunLiveQueryWithSpec for full filter dimensions.
	spec := LiveQueryTargetSpec{
		Hostnames:       hostnames,
		LegacyLabels:    labels,
		LegacyPlatforms: platforms,
		LegacyFleets:    teams,
	}
	return fc.RunLiveQueryWithSpec(ctx, sql, spec)
}

// RunLiveQueryWithSpec resolves the spec to an exact target host list using
// the same intersection semantics as ResolveLiveQueryTargets, then dispatches
// to single-host or multi-host osquery distribution.
func (fc *FleetClient) RunLiveQueryWithSpec(ctx context.Context, sql string, spec LiveQueryTargetSpec) (*LiveQueryResult, error) {
	targets, err := fc.ResolveLiveQueryTargets(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target hosts: %w", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no matching hosts found for the provided targets")
	}

	hostIDs := make([]uint, 0, len(targets))
	nameByID := make(map[uint]Endpoint, len(targets))
	for _, t := range targets {
		hostIDs = append(hostIDs, t.ID)
		nameByID[t.ID] = t
	}

	if len(hostIDs) == 1 {
		return fc.runAdHocSingleHost(ctx, hostIDs[0], sql, nameByID)
	}
	return fc.runMultiHostQuery(ctx, hostIDs, sql, nameByID)
}

// runAdHocSingleHost uses POST /api/v1/fleet/hosts/:id/query (Fleet 4.43+ synchronous REST).
func (fc *FleetClient) runAdHocSingleHost(ctx context.Context, hostID uint, sql string, endpointByID map[uint]Endpoint) (*LiveQueryResult, error) {
	endpointPath := fmt.Sprintf("/api/v1/fleet/hosts/%d/query", hostID)
	resp, err := fc.makeFleetRequest(ctx, "POST", endpointPath, AdHocQueryRequest{Query: sql})
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
//
// The temp query name pairs a millisecond timestamp with 8 random bytes — the
// timestamp keeps lexical order useful for log scans, the random suffix makes
// concurrent invocations from the same MCP process collision-proof. If the
// DELETE in the deferred cleanup fails (network blip, Fleet 5xx, MCP killed),
// the leftover is logged at error level so an operator can run the startup
// sweeper or clean it up by hand. SweepLeftoverTempQueries() also removes any
// such residue at next MCP boot.
func (fc *FleetClient) runMultiHostQuery(ctx context.Context, hostIDs []uint, sql string, endpointByID map[uint]Endpoint) (*LiveQueryResult, error) {
	tempName := fmt.Sprintf("%s%d-%s", tempQueryNamePrefix, time.Now().UnixMilli(), randomHexSuffix(8))
	savedQuery, err := fc.CreateSavedQuery(ctx, tempName, "Temporary MCP live query", sql, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary query: %w", err)
	}
	defer func() {
		delEndpoint := fmt.Sprintf("/api/v1/fleet/reports/id/%d", savedQuery.ID)
		r, delErr := fc.makeFleetRequest(ctx, "DELETE", delEndpoint, nil)
		if r != nil {
			r.Body.Close()
		}
		if delErr != nil {
			logrus.Errorf("failed to delete temp query %s (id=%d): %v — will be swept on next startup", tempName, savedQuery.ID, delErr)
		} else if r != nil && r.StatusCode != http.StatusOK && r.StatusCode != http.StatusNoContent {
			logrus.Errorf("temp query DELETE returned status %d for %s (id=%d) — will be swept on next startup", r.StatusCode, tempName, savedQuery.ID)
		}
	}()

	logrus.Infof("Created temp query ID=%d, running against %d hosts", savedQuery.ID, len(hostIDs))

	runEndpoint := fmt.Sprintf("/api/v1/fleet/reports/%d/run", savedQuery.ID)
	resp, err := fc.makeFleetRequest(ctx, "POST", runEndpoint, MultiQueryRunRequest{HostIDs: hostIDs})
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

// SweepLeftoverTempQueries deletes any saved queries whose name begins with
// tempQueryNamePrefix. Called once at MCP startup to clean up residue from
// previous runMultiHostQuery invocations whose deferred DELETE failed (process
// killed mid-run, Fleet 5xx, network partition). Best-effort: errors are
// logged but do not block startup.
func (fc *FleetClient) SweepLeftoverTempQueries(ctx context.Context) {
	queries, err := fc.GetQueries(ctx)
	if err != nil {
		logrus.Warnf("temp-query sweep: failed to list queries: %v", err)
		return
	}
	swept := 0
	for _, q := range queries {
		if !strings.HasPrefix(q.Name, tempQueryNamePrefix) {
			continue
		}
		delEndpoint := fmt.Sprintf("/api/v1/fleet/reports/id/%d", q.ID)
		r, err := fc.makeFleetRequest(ctx, "DELETE", delEndpoint, nil)
		if r != nil {
			r.Body.Close()
		}
		if err != nil {
			logrus.Warnf("temp-query sweep: failed to delete %s (id=%d): %v", q.Name, q.ID, err)
			continue
		}
		swept++
	}
	if swept > 0 {
		logrus.Infof("temp-query sweep: deleted %d leftover %s* queries", swept, tempQueryNamePrefix)
	}
}

// makeFleetRequest builds and executes a Fleet API request bound to ctx.
//
// ctx propagation: when the MCP caller cancels the request (client disconnect,
// deadline exceeded, transport-level cancellation), the in-flight Fleet HTTP
// call cancels too. Callers that do NOT have a useful ctx may pass
// context.Background(); long-running fan-outs (e.g. CVE compose, live-query
// resolve) should pass the MCP handler's ctx directly.
//
// The previous implementation logged "%s %s" with the full request path —
// that path includes user-supplied identifiers (hostnames, serials, IdP emails
// via ?query=, CVE IDs, etc.) which leaks PII to debug logs and any log
// shipper. We now log only the method and the path before the query string
// so the route shape is observable without exposing identifiers.
func (fc *FleetClient) makeFleetRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	url := fmt.Sprintf("%s%s", fc.baseURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fc.apiKey)

	// PII-safe debug: log the route shape (path before any query string)
	// rather than the full endpoint, which can carry hostnames / emails /
	// CVE IDs as filter values.
	pathOnly, _, _ := strings.Cut(endpoint, "?")
	logrus.Debugf("%s %s", method, pathOnly)

	resp, err := fc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to Fleet API: %w", err)
	}

	return resp, nil
}
