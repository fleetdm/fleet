package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestHostAISessions() {
	t := s.T()
	ctx := t.Context()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)
	hosts := s.createHosts(t, "darwin", "windows", "ubuntu")
	require.Len(t, hosts, 3)
	teamHost, noTeamHost, busyHost := hosts[0], hosts[1], hosts[2]
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{teamHost.ID})))

	now := time.Now().UTC().Truncate(time.Second)
	newSession := func(agent, id string, endedAt time.Time, toolCalls uint, outputTokens uint64) *fleet.HostAISession {
		startedAt := endedAt.Add(-time.Hour)
		return &fleet.HostAISession{
			Agent:             agent,
			AgentSessionID:    id,
			AgentVersion:      "2.1.4",
			Username:          "anna",
			AccountIdentifier: "anna@example.com",
			Model:             "claude-opus-5-5",
			ProjectPath:       "/Users/anna/code/fleet",
			GitBranch:         "main",
			StartedAt:         &startedAt,
			EndedAt:           &endedAt,
			UserMessageCount:  2,
			ToolCallCount:     toolCalls,
			OutputTokenCount:  outputTokens,
			TranscriptPath:    "/Users/anna/.claude/projects/fleet/" + id + ".jsonl",
			TranscriptSize:    1024,
		}
	}
	claude := newSession("claude_code", "claude-1", now.Add(-2*time.Hour), 3, 300)
	cursor := newSession("cursor", "cursor-1", now.Add(-time.Hour), 1, 0)
	cursor.AgentVersion, cursor.Model, cursor.AccountIdentifier, cursor.StartedAt = "", "", "", nil
	codex := newSession("codex_cli", "codex-1", now.Add(-3*time.Hour), 0, 100)
	codex.Username, codex.Model, codex.GitRepository = "bob", "gpt-5", "fleetdm/website"
	require.NoError(t, s.ds.UpsertHostAISessions(ctx, teamHost.ID, []*fleet.HostAISession{claude, cursor, codex}))
	require.NoError(t, s.ds.UpsertHostAISessions(ctx, noTeamHost.ID, []*fleet.HostAISession{newSession("gemini_cli", "gemini-1", now, 0, 0)}))

	// Events carry fleetd's action, which the API maps to categories, and whether their arguments were cut.
	newEvent := func(s *fleet.HostAISession, toolCallIndex uint, decision string, target string, action fleet.AISessionAction) *fleet.HostAISessionEvent {
		return &fleet.HostAISessionEvent{
			Agent:          s.Agent,
			AgentSessionID: s.AgentSessionID,
			SessionEndedAt: s.EndedAt,
			ToolCallIndex:  toolCallIndex,
			Action:         action,
			Target:         target,
			Decision:       decision,
			ExecutedAt:     &now,
			Payload:        json.RawMessage(fmt.Sprintf(`{"name":"Bash","id":"toolu_%d","input":{"command":%q}}`, toolCallIndex, target)),
			ContentLength:  20000,
			Truncated:      true,
		}
	}
	require.NoError(t, s.ds.UpsertHostAISessionEvents(ctx, teamHost.ID, nil, []*fleet.HostAISessionEvent{
		newEvent(claude, 3, "", "WebFetch", fleet.AISessionActionNetwork),
		newEvent(claude, 2, "approved", "README.md", fleet.AISessionActionRead),
		newEvent(claude, 1, "denied", "rm -rf /", fleet.AISessionActionExecute),
		newEvent(cursor, 1, "", "main.go", fleet.AISessionActionWrite),
	}))

	sessionsURL := func(host *fleet.Host) string {
		return fmt.Sprintf("/api/latest/fleet/hosts/%d/ai_sessions", host.ID)
	}
	listSessions := func(wantStatus int, queryParams ...string) listHostAISessionsResponse {
		var resp listHostAISessionsResponse
		s.DoJSON("GET", sessionsURL(teamHost), nil, wantStatus, &resp, queryParams...)
		return resp
	}
	sessionIDs := func(resp listHostAISessionsResponse) []string {
		ids := []string{}
		for _, s := range resp.AISessions {
			ids = append(ids, s.AgentSessionID)
		}
		return ids
	}

	// Default order is ended_at descending.
	resp := listSessions(http.StatusOK)
	require.Equal(t, []string{"cursor-1", "claude-1", "codex-1"}, sessionIDs(resp))
	require.Equal(t, uint(3), resp.Count)
	require.Equal(t, &fleet.PaginationMetadata{}, resp.Meta)
	got := resp.AISessions[1]
	require.Equal(t, teamHost.ID, got.HostID)
	require.Equal(t, "anna@example.com", got.AccountIdentifier)
	require.Equal(t, uint(3), got.ToolCallCount)
	require.Equal(t, []fleet.AISessionCategory{fleet.AISessionCategoryCommand, fleet.AISessionCategoryFileAccess}, got.Categories)
	claudeID := got.ID

	// Unknown values are empty strings, zeros, nulls, and empty arrays.
	raw := s.Do("GET", sessionsURL(teamHost), nil, http.StatusOK, "agent", "codex_cli")
	body, err := io.ReadAll(raw.Body)
	require.NoError(t, err)
	var rawResp struct {
		AISessions []map[string]any `json:"ai_sessions"`
	}
	require.NoError(t, json.Unmarshal(body, &rawResp))
	require.Len(t, rawResp.AISessions, 1)
	require.Equal(t, []any{}, rawResp.AISessions[0]["categories"])
	require.NotContains(t, rawResp.AISessions[0], "actions")
	require.Equal(t, false, rawResp.AISessions[0]["backfill"])
	require.Zero(t, rawResp.AISessions[0]["tool_call_count"])
	resp = listSessions(http.StatusOK, "agent", "cursor")
	require.Nil(t, resp.AISessions[0].StartedAt)
	require.Empty(t, resp.AISessions[0].Model)

	// Filters, search, sorting and page-based pagination are covered by the datastore tests; the HTTP checks are the
	// query parameters reaching the datastore and an empty list being [].
	require.Equal(t, []string{"claude-1"}, sessionIDs(listSessions(http.StatusOK, "category", "command")))
	require.Equal(t, []string{"codex-1"}, sessionIDs(listSessions(http.StatusOK, "query", "bob")))
	resp = listSessions(http.StatusOK, "agent", "gemini_cli")
	require.Empty(t, resp.AISessions)
	require.NotNil(t, resp.AISessions)
	require.Zero(t, resp.Count)

	// The direction defaults to descending when only the key is given.
	require.Equal(t, []string{"claude-1", "codex-1", "cursor-1"}, sessionIDs(listSessions(http.StatusOK, "order_key", "output_token_count")))
	require.Equal(t, []string{"cursor-1", "codex-1", "claude-1"},
		sessionIDs(listSessions(http.StatusOK, "order_key", "output_token_count", "order_direction", "asc")))

	// Timestamp cursors are accepted as the API reports them, and supersede the page.
	resp = listSessions(http.StatusOK, "order_key", "ended_at", "after", claude.EndedAt.Format(time.RFC3339), "page", "3")
	require.Equal(t, []string{"codex-1"}, sessionIDs(resp))
	require.Equal(t, &fleet.PaginationMetadata{}, resp.Meta)

	// Pages default to 50 results, so a busy host or session doesn't come back whole.
	manySessions := make([]*fleet.HostAISession, 0, 51)
	for i := range 51 {
		manySessions = append(manySessions, newSession("claude_code", fmt.Sprintf("many-%d", i), now.Add(-time.Duration(i)*time.Minute), 0, 0))
	}
	require.NoError(t, s.ds.UpsertHostAISessions(ctx, busyHost.ID, manySessions))
	var busyResp listHostAISessionsResponse
	s.DoJSON("GET", sessionsURL(busyHost), nil, http.StatusOK, &busyResp)
	require.Len(t, busyResp.AISessions, 50)
	require.Equal(t, uint(51), busyResp.Count)
	require.Equal(t, &fleet.PaginationMetadata{HasNextResults: true}, busyResp.Meta)
	s.DoJSON("GET", sessionsURL(busyHost), nil, http.StatusOK, &busyResp, "per_page", "51")
	require.Len(t, busyResp.AISessions, 51)
	busySession := busyResp.AISessions[0]
	manyEvents := make([]*fleet.HostAISessionEvent, 0, 51)
	for i := range uint(51) {
		manyEvents = append(manyEvents, newEvent(busySession, i, "", "ls", fleet.AISessionActionExecute))
	}
	require.NoError(t, s.ds.UpsertHostAISessionEvents(ctx, busyHost.ID, nil, manyEvents))
	var busyEventsResp listHostAISessionEventsResponse
	s.DoJSON("GET", fmt.Sprintf("%s/%d/events", sessionsURL(busyHost), busySession.ID), nil, http.StatusOK, &busyEventsResp)
	require.Len(t, busyEventsResp.Events, 50)
	require.Equal(t, uint(51), busyEventsResp.Count)
	require.Equal(t, &fleet.PaginationMetadata{HasNextResults: true}, busyEventsResp.Meta)

	// Invalid values.
	for _, c := range []struct {
		params  []string
		wantErr string
	}{
		{[]string{"agent", "claude"}, "claude_code, copilot_cli"},
		{[]string{"category", "network"}, `"command" or "file_access"`},
		{[]string{"order_key", "id"}, "started_at, ended_at"},
		{[]string{"order_key", "tool_call_count"}, "started_at, ended_at"},
		{[]string{"order_direction", "up"}, `"asc" or "desc"`},
		{[]string{"order_key", "ended_at", "after", "yesterday"}, "timestamp"},
		{[]string{"after", "2026-13-45"}, "timestamp"},
		{[]string{"order_key", "output_token_count", "after", "many"}, "integer"},
	} {
		res := s.Do("GET", sessionsURL(teamHost), nil, http.StatusUnprocessableEntity, c.params...)
		require.Contains(t, extractServerErrorText(res.Body), c.wantErr)
	}
	s.Do("GET", "/api/latest/fleet/hosts/999999/ai_sessions", nil, http.StatusNotFound)

	// Get a session.
	var getResp getHostAISessionResponse
	s.DoJSON("GET", fmt.Sprintf("%s/%d", sessionsURL(teamHost), claudeID), nil, http.StatusOK, &getResp)
	require.Equal(t, claudeID, getResp.AISession.ID)
	require.Equal(t, teamHost.ID, getResp.AISession.HostID)
	require.Equal(t, "claude-1", getResp.AISession.AgentSessionID)
	require.Equal(t, "2.1.4", getResp.AISession.AgentVersion)
	require.Equal(t, uint64(300), getResp.AISession.OutputTokenCount)
	require.Equal(t, claude.EndedAt.Unix(), getResp.AISession.EndedAt.Unix())
	s.Do("GET", fmt.Sprintf("%s/%d", sessionsURL(noTeamHost), claudeID), nil, http.StatusNotFound)
	s.Do("GET", fmt.Sprintf("%s/%d", sessionsURL(teamHost), claudeID+1000), nil, http.StatusNotFound)

	// List a session's events, in the order the calls ran.
	eventsURL := fmt.Sprintf("%s/%d/events", sessionsURL(teamHost), claudeID)
	listEvents := func(wantStatus int, queryParams ...string) listHostAISessionEventsResponse {
		var resp listHostAISessionEventsResponse
		s.DoJSON("GET", eventsURL, nil, wantStatus, &resp, queryParams...)
		return resp
	}
	targets := func(resp listHostAISessionEventsResponse) []string {
		out := []string{}
		for _, e := range resp.Events {
			out = append(out, e.Target)
		}
		return out
	}
	eventsResp := listEvents(http.StatusOK)
	require.Equal(t, []string{"rm -rf /", "README.md", "WebFetch"}, targets(eventsResp))
	require.Equal(t, uint(3), eventsResp.Count)
	first := eventsResp.Events[0]
	require.NotZero(t, first.ID)
	require.Equal(t, "denied", first.Decision)
	require.Equal(t, []fleet.AISessionCategory{fleet.AISessionCategoryCommand}, first.Categories)
	require.Equal(t, now.Unix(), first.ExecutedAt.Unix())
	require.JSONEq(t, `{"name":"Bash","id":"toolu_1","input":{"command":"rm -rf /"}}`, string(first.Payload))
	require.Equal(t, []fleet.AISessionCategory{}, eventsResp.Events[2].Categories)

	raw = s.Do("GET", eventsURL, nil, http.StatusOK, "per_page", "1")
	body, err = io.ReadAll(raw.Body)
	require.NoError(t, err)
	require.JSONEq(t, fmt.Sprintf(`{
		"events": [{
			"id": %d,
			"categories": ["command"],
			"target": "rm -rf /",
			"decision": "denied",
			"executed_at": %q,
			"payload": {"name": "Bash", "id": "toolu_1", "input": {"command": "rm -rf /"}},
			"truncated": true
		}],
		"count": 3,
		"meta": {"has_next_results": true, "has_previous_results": false}
	}`, first.ID, now.Format(time.RFC3339)), string(body))

	require.Equal(t, []string{"README.md"}, targets(listEvents(http.StatusOK, "category", "file_access")))
	eventsResp = listEvents(http.StatusOK, "per_page", "2", "page", "1")
	require.Equal(t, []string{"WebFetch"}, targets(eventsResp))
	require.Equal(t, &fleet.PaginationMetadata{HasPreviousResults: true}, eventsResp.Meta)
	res := s.Do("GET", eventsURL, nil, http.StatusUnprocessableEntity, "category", "network")
	require.Contains(t, extractServerErrorText(res.Body), `"command" or "file_access"`)
	s.Do("GET", fmt.Sprintf("%s/%d/events", sessionsURL(noTeamHost), claudeID), nil, http.StatusNotFound)
	s.Do("GET", fmt.Sprintf("%s/%d/events", sessionsURL(teamHost), claudeID+1000), nil, http.StatusNotFound)

	// Only admins and maintainers, globally or of the host's fleet, can read sessions. The policy test has every role;
	// this checks the service authorizes on the host's fleet, and that gitops is stopped before it, at listing hosts.
	defer func() { s.token = s.getTestAdminToken() }()
	urls := func(host *fleet.Host, sessionID uint) []string {
		return []string{
			sessionsURL(host),
			fmt.Sprintf("%s/%d", sessionsURL(host), sessionID),
			fmt.Sprintf("%s/%d/events", sessionsURL(host), sessionID),
		}
	}
	var noTeamResp listHostAISessionsResponse
	s.DoJSON("GET", sessionsURL(noTeamHost), nil, http.StatusOK, &noTeamResp)
	require.Len(t, noTeamResp.AISessions, 1)
	noTeamSessionID := noTeamResp.AISessions[0].ID

	newUser := func(name string, globalRole *string, teams ...fleet.UserTeam) {
		u := &fleet.User{
			Name:       name,
			Email:      name + "-ai-sessions@example.com",
			GlobalRole: globalRole,
			Teams:      teams,
		}
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(ctx, u)
		require.NoError(t, err)
		s.token = s.getTestToken(u.Email, test.GoodPassword)
	}
	for _, c := range []struct {
		name       string
		globalRole string
		team       *fleet.Team
		teamRole   string
		teamHost   int
		noTeamHost int
	}{
		{name: "global-maintainer", globalRole: fleet.RoleMaintainer, teamHost: http.StatusOK, noTeamHost: http.StatusOK},
		{name: "global-observer", globalRole: fleet.RoleObserver, teamHost: http.StatusForbidden, noTeamHost: http.StatusForbidden},
		{name: "global-gitops", globalRole: fleet.RoleGitOps, teamHost: http.StatusForbidden, noTeamHost: http.StatusForbidden},
		{name: "team-admin", team: team1, teamRole: fleet.RoleAdmin, teamHost: http.StatusOK, noTeamHost: http.StatusForbidden},
		{name: "other-team-admin", team: team2, teamRole: fleet.RoleAdmin, teamHost: http.StatusForbidden, noTeamHost: http.StatusForbidden},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.team != nil {
				newUser(c.name, nil, fleet.UserTeam{Team: *c.team, Role: c.teamRole})
			} else {
				newUser(c.name, new(c.globalRole))
			}
			for _, u := range urls(teamHost, claudeID) {
				s.Do("GET", u, nil, c.teamHost)
			}
			for _, u := range urls(noTeamHost, noTeamSessionID) {
				s.Do("GET", u, nil, c.noTeamHost)
			}
		})
	}
}
