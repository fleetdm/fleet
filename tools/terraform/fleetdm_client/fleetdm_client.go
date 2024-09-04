package fleetdm_client

// This file gives us a nice API to use to call FleetDM's API. It's focused
// only on teams.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const prefix = "/api/v1/fleet"
const teamPrefix = prefix + "/teams"

type Team struct {
	Name         string      `json:"name"`
	ID           int64       `json:"id"`
	Description  string      `json:"description"`
	AgentOptions interface{} `json:"agent_options"`
	Scripts      interface{} `json:"scripts"`
	Secrets      []struct {
		Secret  string    `json:"secret"`
		Created time.Time `json:"created_at"`
		TeamID  int       `json:"team_id"`
	} `json:"secrets"`
}

type TeamCreate struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type TeamGetResponse struct {
	Team Team `json:"team"`
}

type TeamQueryResponse struct {
	Teams []Team `json:"teams"`
}

// FleetDMClient is a FleetDM HTTP client that overrides http.DefaultClient.
type FleetDMClient struct {
	*http.Client
	URL    string
	APIKey string
}

// NewFleetDMClient creates a new instance of FleetDMClient with the provided
// URL and API key.
func NewFleetDMClient(url, apiKey string) *FleetDMClient {
	// Ensure the URL ends with a trailing slash for bizarre parsing reasons
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return &FleetDMClient{
		Client: http.DefaultClient,
		URL:    url,
		APIKey: apiKey,
	}
}

// Do will add necessary headers and call the http.Client.Do method.
func (c *FleetDMClient) do(req *http.Request, query *url.Values) (*http.Response, error) {
	// Add the API key to the request header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Add("Accept", `application/json`)
	// Set the request URL based on the client URL
	baseURL, err := url.Parse(c.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL %s: %w", c.URL, err)
	}
	req.URL = baseURL.JoinPath(req.URL.Path)
	if query != nil {
		req.URL.RawQuery = query.Encode()
	}
	// Send the request using the embedded http.Client
	return c.Client.Do(req)
}

// TeamNameToID will return the ID of a team given the name.
func (c *FleetDMClient) TeamNameToID(name string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, teamPrefix, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create GET request for %s: %w", teamPrefix, err)
	}
	vals := make(url.Values)
	vals.Add("query", name)
	resp, err := c.do(req, &vals)
	if err != nil {
		return 0, fmt.Errorf("failed to GET %s %s: %w", teamPrefix, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get team: %s %s", name, resp.Status)
	}

	var teamqry TeamQueryResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read team response: %w", err)
	}
	err = json.Unmarshal(b, &teamqry)
	if err != nil {
		return 0, fmt.Errorf("failed to decode get team response: %w", err)
	}

	for _, team := range teamqry.Teams {
		if team.Name == name {
			return team.ID, nil
		}
	}

	return 0, fmt.Errorf("team %s not found", name)
}

// CreateTeam creates a new team with the provided name and description.
func (c *FleetDMClient) CreateTeam(name string, description string) (*TeamGetResponse, error) {
	teamCreate := TeamCreate{
		Name:        name,
		Description: description,
	}
	nameJson, err := json.Marshal(teamCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create team body: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, teamPrefix, bytes.NewReader(nameJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request for %s name %s: %w",
			req.URL.String(), name, err)
	}
	resp, err := c.do(req, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to POST %s name %s: %w",
			req.URL.String(), name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create team %s: %s %s", name, resp.Status, string(b))
	}

	var newTeam TeamGetResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read team response: %w", err)
	}
	err = json.Unmarshal(b, &newTeam)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &newTeam, nil
}

// GetTeam returns the team with the provided ID.
func (c *FleetDMClient) GetTeam(id int64) (*TeamGetResponse, error) {
	url := teamPrefix + "/" + strconv.FormatInt(id, 10)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for %s: %w",
			url, err)
	}
	resp, err := c.do(req, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get team %d: %s %s", id, resp.Status, string(b))
	}

	var team TeamGetResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read team response: %w", err)
	}
	err = json.Unmarshal(b, &team)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &team, nil
}

// UpdateTeam updates the team with the provided ID with the provided name and description.
func (c *FleetDMClient) UpdateTeam(id int64, name, description *string) (*TeamGetResponse, error) {
	if name == nil && description == nil {
		return nil, fmt.Errorf("both name and description are nil")
	}

	url := teamPrefix + "/" + strconv.FormatInt(id, 10)
	var teamUpdate TeamCreate
	if name != nil {
		teamUpdate.Name = *name
	}
	if description != nil {
		teamUpdate.Description = *description
	}
	updateJson, err := json.Marshal(teamUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to update team body request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(updateJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create PATCH request for %s body %s: %w",
			url, updateJson, err)
	}
	resp, err := c.do(req, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to PATCH %s body %s: %w",
			url, updateJson, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed PATCH %s body %s: %s %s",
			url, updateJson, resp.Status, string(b))
	}

	var newTeam TeamGetResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read team response: %w", err)
	}
	err = json.Unmarshal(b, &newTeam)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &newTeam, nil
}

// UpdateAgentOptions pretends that the agent options are a string, when it's really actually json.
// Strangely the body that comes back is a team, not just the agent options.
func (c *FleetDMClient) UpdateAgentOptions(id int64, ao string) (*TeamGetResponse, error) {

	// First verify it's actually json.
	valid := json.Valid([]byte(ao))
	if !valid {
		return nil, fmt.Errorf("agent_options isn't json: %s", ao)
	}

	aoUrl := teamPrefix + "/" + strconv.FormatInt(id, 10) + "/" + "agent_options"
	req, err := http.NewRequest(http.MethodPost, aoUrl, strings.NewReader(ao))
	if err != nil {
		return nil, fmt.Errorf("failed to create agent_options POST request for %s id %d: %w",
			teamPrefix, id, err)
	}
	resp, err := c.do(req, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to POST agent_options %s id %d: %w",
			teamPrefix, id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to modify agent_options %d: %s %s", id, resp.Status, string(b))
	}

	var team TeamGetResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read team response: %w", err)
	}
	err = json.Unmarshal(b, &team)
	if err != nil {
		return nil, fmt.Errorf("failed to decode agent_options team response: %w", err)
	}
	return &team, nil
}

// DeleteTeam deletes the team with the provided ID.
func (c *FleetDMClient) DeleteTeam(id int64) error {
	url := teamPrefix + "/" + strconv.FormatInt(id, 10)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request for %s: %w", url, err)
	}
	resp, err := c.do(req, nil)
	if err != nil {
		return fmt.Errorf("failed to DELETE %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete team %d: %s", id, resp.Status)
	}

	return nil
}
