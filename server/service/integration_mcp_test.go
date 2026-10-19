package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const mcpPath = "/api/latest/fleet/mcp"

var mcpInitializeBody = []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func (ts *withServer) newMCPClient(t *testing.T, headers map[string]string) *client.Client {
	t.Helper()
	c, err := client.NewStreamableHttpClient(ts.server.URL+mcpPath, transport.WithHTTPHeaders(headers))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.Start(t.Context()))
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1"}
	_, err = c.Initialize(t.Context(), initReq)
	require.NoError(t, err)
	return c
}

func callMCPToolResult(t *testing.T, c *client.Client, name string, args map[string]any) (string, bool) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	res, err := c.CallTool(t.Context(), req)
	require.NoError(t, err)
	require.NotEmpty(t, res.Content)
	text, ok := mcp.AsTextContent(res.Content[0])
	require.True(t, ok)
	return text.Text, res.IsError
}

func callMCPTool(t *testing.T, c *client.Client, name string, args map[string]any) string {
	t.Helper()
	text, isError := callMCPToolResult(t, c, name, args)
	require.False(t, isError, "%s: %s", name, text)
	return text
}

func (s *integrationEnterpriseTestSuite) TestEmbeddedMCP() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	newHost := func(name string, teamID *uint, seen time.Time) *fleet.Host {
		id := strings.ReplaceAll(t.Name(), "/", "-") + name
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        seen,
			OsqueryHostID:   new(id),
			NodeKey:         new(id),
			UUID:            id,
			Hostname:        id,
			Platform:        "darwin",
		})
		require.NoError(t, err)
		if teamID != nil {
			require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(teamID, []uint{h.ID})))
		}
		return h
	}
	globalHost := newHost("global", nil, time.Now())
	newHost("team", &team.ID, time.Now())
	asleep := []*fleet.Host{newHost("asleep1", nil, time.Now().Add(-24*time.Hour)), newHost("asleep2", nil, time.Now().Add(-24*time.Hour))}

	type apiOnlyUser struct {
		User  struct{ ID uint } `json:"user"`
		Token string            `json:"token"`
	}
	createAPIOnly := func(body map[string]any) apiOnlyUser {
		var resp apiOnlyUser
		s.DoJSON("POST", "/api/latest/fleet/users/api_only", body, http.StatusOK, &resp)
		return resp
	}
	operator := createAPIOnly(map[string]any{"name": t.Name() + " operator", "global_role": fleet.RoleObserverPlus})
	// Restricted to the MCP route alone: every tool must still work, and nothing else.
	mcpOnly := createAPIOnly(map[string]any{
		"name":          t.Name() + " mcp only",
		"global_role":   fleet.RoleObserverPlus,
		"api_endpoints": []map[string]any{{"method": "POST", "path": "/api/v1/fleet/mcp"}},
	})

	totalHosts := func(c *client.Client) int {
		var n int
		_, err := fmt.Sscanf(callMCPTool(t, c, "get_total_system_count", nil), "Total Enrolled Systems: %d", &n)
		require.NoError(t, err)
		return n
	}
	readBody := func(resp *http.Response) string {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		_ = resp.Body.Close()
		return string(body)
	}

	t.Run("authentication", func(t *testing.T) {
		s.DoRawNoAuth("POST", mcpPath, mcpInitializeBody, http.StatusUnauthorized)
		s.DoRawWithHeaders("POST", mcpPath, mcpInitializeBody, http.StatusUnauthorized, bearer("not-a-token"))
		// The suite's own token belongs to a UI admin.
		s.DoRaw("POST", mcpPath, mcpInitializeBody, http.StatusForbidden)
		s.DoRawWithHeaders("GET", mcpPath, nil, http.StatusMethodNotAllowed, bearer(operator.Token))
	})

	t.Run("Origin must be the Fleet server's own", func(t *testing.T) {
		var cfg appConfigResponse
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &cfg)
		serverURL, err := url.Parse(cfg.ServerSettings.ServerURL)
		require.NoError(t, err)

		headers := bearer(operator.Token)
		headers["Origin"] = "https://evil.example.com"
		s.DoRawWithHeaders("POST", mcpPath, mcpInitializeBody, http.StatusForbidden, headers)

		headers["Origin"] = serverURL.Scheme + "://" + serverURL.Host
		assert.Positive(t, totalHosts(s.newMCPClient(t, headers)))
	})

	t.Run("tools run as the API-only user", func(t *testing.T) {
		c := s.newMCPClient(t, bearer(operator.Token))
		tools, err := c.ListTools(t.Context(), mcp.ListToolsRequest{})
		require.NoError(t, err)
		var names []string
		for _, tool := range tools.Tools {
			names = append(names, tool.Name)
		}
		assert.Contains(t, names, "run_live_query")
		assert.NotContains(t, names, "refresh_osquery_schema")
		assert.GreaterOrEqual(t, totalHosts(c), 4)
	})

	t.Run("a user restricted to the MCP route can use every tool and nothing else", func(t *testing.T) {
		admin := s.users["admin1@example.com"]
		label, err := s.ds.NewLabel(ctx, &fleet.Label{
			Name: t.Name(), Query: "select 1", LabelType: fleet.LabelTypeRegular, LabelMembershipType: fleet.LabelMembershipTypeManual,
		})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddLabelsToHost(ctx, globalHost.ID, []uint{label.ID}))
		globalPolicy, err := s.ds.NewGlobalPolicy(ctx, &admin.ID, fleet.PolicyPayload{Name: t.Name(), Query: "select 1"})
		require.NoError(t, err)
		teamPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, &admin.ID, fleet.PolicyPayload{Name: t.Name() + " team", Query: "select 1"})
		require.NoError(t, err)

		hostID := fmt.Sprint(globalHost.ID)
		c := s.newMCPClient(t, bearer(mcpOnly.Token))
		// A tool whose API call isn't covered by the MCP route set fails with
		// the endpoint restriction, so every call here must succeed.
		for _, call := range []struct {
			tool string
			args map[string]any
		}{
			{"get_endpoints", nil},
			{"get_endpoints", map[string]any{"fleet": team.Name}},
			{"get_endpoints", map[string]any{"label": label.Name}},
			{"get_endpoints", map[string]any{"policy_id": fmt.Sprint(globalPolicy.ID), "policy_response": "failing"}},
			{"get_host", map[string]any{"host_id": hostID}},
			{"get_host", map[string]any{"identifier": globalHost.UUID}},
			{"get_host_policies", map[string]any{"host_id": hostID}},
			{"get_host_policies", map[string]any{"identifier": globalHost.UUID}},
			{"get_host_users", map[string]any{"host_id": hostID}},
			{"get_total_system_count", nil},
			{"get_aggregate_platforms", nil},
			{"get_fleets", nil},
			{"get_labels", nil},
			{"get_queries", nil},
			{"get_vetted_queries", nil},
			{"get_osquery_schema", map[string]any{"tables": "os_version"}},
			{"prepare_live_query", map[string]any{"host_ids": hostID}},
			{"get_policies", nil},
			{"get_policy_compliance", map[string]any{"policy_id": fmt.Sprint(globalPolicy.ID)}},
			{"get_policy_compliance", map[string]any{"policy_id": fmt.Sprint(teamPolicy.ID), "fleet": team.Name}},
			{"get_policy_hosts", map[string]any{"policy_id": fmt.Sprint(globalPolicy.ID), "response": "failing"}},
			{"get_vulnerability_impact", map[string]any{"cve_id": "CVE-2024-0001"}},
			{"get_vulnerability_hosts", map[string]any{"cve_id": "CVE-2024-0001"}},
			{"get_software", nil},
			{"get_software", map[string]any{"host_id": hostID}},
		} {
			callMCPTool(t, c, call.tool, call.args)
		}

		// The same token can't call the API directly.
		s.DoRawWithHeaders("GET", "/api/latest/fleet/hosts/count", nil, http.StatusForbidden, bearer(mcpOnly.Token))

		noMCP := createAPIOnly(map[string]any{
			"name":          t.Name() + " no mcp",
			"global_role":   fleet.RoleObserver,
			"api_endpoints": []map[string]any{{"method": "GET", "path": "/api/v1/fleet/hosts/count"}},
		})
		s.DoRawWithHeaders("POST", mcpPath, mcpInitializeBody, http.StatusForbidden, bearer(noMCP.Token))
	})

	t.Run("a tool argument can't reach another route", func(t *testing.T) {
		c := s.newMCPClient(t, bearer(mcpOnly.Token))
		for _, identifier := range []string{
			fmt.Sprintf("../%d/recovery_lock_password", globalHost.ID),
			fmt.Sprintf("../%d/encryption_key", globalHost.ID),
			"..",
		} {
			text, isError := callMCPToolResult(t, c, "get_host", map[string]any{"identifier": identifier})
			assert.True(t, isError, identifier)
			assert.Contains(t, text, "not found", identifier)
		}

		var resp listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &resp, "order_key", "id", "order_direction", "desc", "per_page", "20")
		for _, a := range resp.Activities {
			assert.NotEqual(t, fleet.ActivityTypeViewedHostRecoveryLockPassword{}.ActivityName(), a.Type)
		}
	})

	t.Run("a fleet-scoped API-only user only sees its fleet", func(t *testing.T) {
		scoped := createAPIOnly(map[string]any{
			"name":   t.Name() + " team observer",
			"fleets": []map[string]any{{"id": team.ID, "role": fleet.RoleObserver}},
		})
		assert.Equal(t, 1, totalHosts(s.newMCPClient(t, bearer(scoped.Token))))
	})

	t.Run("the setting turns it off", func(t *testing.T) {
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"server_settings":{"mcp_enabled":false}}`), http.StatusOK)
		t.Cleanup(func() {
			s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"server_settings":{"mcp_enabled":true}}`), http.StatusOK)
		})
		resp := s.DoRawWithHeaders("POST", mcpPath, mcpInitializeBody, http.StatusBadRequest, bearer(operator.Token))
		assert.Contains(t, readBody(resp), "server_settings.mcp_enabled")
	})

	t.Run("a live query waits the server period and is attributed to the API-only user", func(t *testing.T) {
		// Read per request by the live query endpoint the tool calls.
		t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "2s")
		s.lq.On("RunQuery", mock.Anything, mock.Anything, []uint{globalHost.ID}).Return(nil)
		s.lq.On("StopQuery", mock.Anything).Return(nil)

		c := s.newMCPClient(t, bearer(mcpOnly.Token))
		start := time.Now()
		text := callMCPTool(t, c, "run_live_query", map[string]any{
			"sql":      "SELECT version FROM os_version;",
			"host_ids": fmt.Sprint(globalHost.ID),
		})
		elapsed := time.Since(start)
		assert.GreaterOrEqual(t, elapsed, 2*time.Second)
		assert.Less(t, elapsed, 20*time.Second, "the embedded MCP must not wait longer than the server's period plus margin")
		assert.Contains(t, text, "timeout waiting for results")

		liveQueryActivity := fleet.ActivityTypeLiveQuery{}.ActivityName()
		require.Eventually(t, func() bool {
			var resp listActivitiesResponse
			s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &resp, "order_key", "id", "order_direction", "desc", "per_page", "5")
			for _, a := range resp.Activities {
				if a.Type == liveQueryActivity && a.ActorID != nil && *a.ActorID == mcpOnly.User.ID {
					return true
				}
			}
			return false
		}, 10*time.Second, 200*time.Millisecond)
	})

	// Campaign creation and the results websocket both go through the in-memory
	// connection with the caller's token. Asleep hosts let the campaign finish
	// as soon as Fleet reports no online targets.
	t.Run("a multi-host live query streams results over the internal connection", func(t *testing.T) {
		ids := []uint{asleep[0].ID, asleep[1].ID}
		s.lq.On("RunQuery", mock.Anything, mock.Anything, mock.MatchedBy(func(got []uint) bool {
			return len(got) == 2 && ((got[0] == ids[0] && got[1] == ids[1]) || (got[0] == ids[1] && got[1] == ids[0]))
		})).Return(nil)
		s.lq.On("StopQuery", mock.Anything).Return(nil)

		c := s.newMCPClient(t, bearer(mcpOnly.Token))
		text := callMCPTool(t, c, "run_live_query", map[string]any{
			"sql":      "SELECT version FROM os_version;",
			"host_ids": fmt.Sprintf("%d,%d", ids[0], ids[1]),
		})
		var res struct {
			Results struct {
				Targeted  int `json:"targeted_host_count"`
				Responded int `json:"responded_host_count"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal([]byte(text), &res), text)
		assert.Equal(t, 2, res.Results.Targeted)
		assert.Zero(t, res.Results.Responded)
	})
}

func (s *integrationTestSuite) TestEmbeddedMCPRequiresPremium() {
	t := s.T()
	u := &fleet.User{
		Name:       t.Name(),
		Email:      strings.ToLower(t.Name()) + "@example.com",
		GlobalRole: new(fleet.RoleObserver),
		APIOnly:    true,
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	u, err := s.ds.NewUser(t.Context(), u)
	require.NoError(t, err)
	session, err := s.ds.NewSession(t.Context(), u.ID, 64)
	require.NoError(t, err)

	s.DoRawWithHeaders("POST", mcpPath, mcpInitializeBody, http.StatusPaymentRequired, bearer(session.Key))
}
