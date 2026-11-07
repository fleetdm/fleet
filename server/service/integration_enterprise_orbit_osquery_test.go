package service

// Agent (orbit and osquery) tests for the enterprise (premium) suite.
//
// Belongs here: orbit enrollment and re-enrollment (IdP device mapping, end user auth tokens,
// fleet assignment), orbit config (extensions), and osquery distributed reads and detail queries
// that depend on premium features.
//
// Does not belong here: setup experience status (integration_enterprise_setup_experience_test.go),
// fleet label distributed reads/writes (integration_enterprise_labels_test.go), and Fleet Desktop
// device endpoints (integration_enterprise_desktop_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestDistributedReadWithFeatures() {
	t := s.T()

	// Global config has both features enabled
	spec := []byte(`
  features:
    additional_queries: null
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)

	// Team config has only additional queries enabled
	a := json.RawMessage(`{"time": "SELECT * FROM time"}`)
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          8324,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: false,
				AdditionalQueries:       &a,
			},
		},
	})
	require.NoError(t, err)

	// Create a host without a team
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "select 1 from osquery;"}, nil)

	// ensure we can read distributed queries for the host
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// get distributed queries for the host
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_ai_tools")
	require.Contains(t, dqResp.Discovery["fleet_detail_query_software_ai_tools"], "name = 'ai_tools'")
	require.NotContains(t, dqResp.Queries, "fleet_additional_query_time")

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_software_ai_tools")
	require.Contains(t, dqResp.Queries, "fleet_additional_query_time")
}

func (s *integrationEnterpriseTestSuite) TestOrbitConfigExtensions() {
	t := s.T()
	ctx := context.Background()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	defer func() {
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	foobarLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  "Foobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	zoobarLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  "Zoobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	allHostsLabel, err := s.ds.GetLabelSpec(ctx, fleet.TeamFilter{}, "All hosts")
	require.NoError(t, err)

	orbitDarwinClient := createOrbitEnrolledHost(t, "darwin", "foobar1", s.ds)
	orbitLinuxClient := createOrbitEnrolledHost(t, "linux", "foobar2", s.ds)
	orbitWindowsClient := createOrbitEnrolledHost(t, "windows", "foobar3", s.ds)

	// orbitDarwinClient is member of 'All hosts' and 'Zoobar' labels.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitDarwinClient, map[uint]*bool{
		allHostsLabel.ID: new(true),
		zoobarLabel.ID:   new(true),
	}, time.Now(), false)
	require.NoError(t, err)
	// orbitLinuxClient is member of 'All hosts' and 'Foobar' labels.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitLinuxClient, map[uint]*bool{
		allHostsLabel.ID: new(true),
		foobarLabel.ID:   new(true),
	}, time.Now(), false)
	require.NoError(t, err)
	// orbitWindowsClient is member of the 'All hosts' label only.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitWindowsClient, map[uint]*bool{
		allHostsLabel.ID: new(true),
	}, time.Now(), false)
	require.NoError(t, err)

	// Attempt to add labels to extensions.
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"labels": [
				"All hosts",
				"Foobar"
			],
			"channel": "stable",
			"platform": "linux"
		},
		"hello_world_macos": {
			"labels": [
				"All hosts",
				"Foobar"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_mars_macos": {
			"labels": [
				"All hosts",
				"Zoobar"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_world_windows": {
			"labels": [
				"Zoobar"
			],
			"channel": "stable",
			"platform": "windows"
		},
		"hello_mars_windows": {
			"labels": [
				"Foobar"
			],
			"channel": "stable",
			"platform": "windows"
		}
	}
  }
}`), http.StatusOK)

	resp := fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_mars_macos": {
		"channel": "stable",
		"platform": "macos"
	}
  }`, string(resp.Extensions))

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_linux": {
		"channel": "stable",
		"platform": "linux"
	}
  }`, string(resp.Extensions))

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitWindowsClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, string(resp.Extensions))

	// orbitDarwinClient is now also a member of the 'Foobar' label.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitDarwinClient, map[uint]*bool{
		foobarLabel.ID: new(true),
	}, time.Now(), false)
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_macos": {
		"channel": "stable",
		"platform": "macos"
	},
	"hello_mars_macos": {
		"channel": "stable",
		"platform": "macos"
	}
  }`, string(resp.Extensions))

	// orbitLinuxClient is no longer a member of the 'Foobar' label.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitLinuxClient, map[uint]*bool{
		foobarLabel.ID: new(false),
	}, time.Now(), false)
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, string(resp.Extensions))

	// Attempt to set non-existent labels in the config.
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"labels": [
				"All hosts",
				"Doesn't exist"
			],
			"channel": "stable",
			"platform": "linux"
		}
	}
  }
}`), http.StatusBadRequest)
}

// aiSessionsTestHost enrolls a host and returns how to read its detail queries and submit results as fleetd does.
func (s *integrationEnterpriseTestSuite) aiSessionsTestHost() (*fleet.Host, func() getDistributedQueriesResponse, func(name, rows string)) {
	t := s.T()
	ctx := t.Context()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	readQueries := func() getDistributedQueriesResponse {
		require.NoError(t, s.ds.UpdateHostRefetchRequested(ctx, host.ID, true))
		var dqResp getDistributedQueriesResponse
		s.DoJSON("POST", "/api/osquery/distributed/read", getDistributedQueriesRequest{NodeKey: *host.NodeKey}, http.StatusOK, &dqResp)
		return dqResp
	}
	submit := func(name, rows string) {
		var resp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write", submitDistributedQueryResultsRequestShim{
			NodeKey:  *host.NodeKey,
			Results:  map[string]json.RawMessage{name: json.RawMessage(rows)},
			Statuses: map[string]any{},
			Messages: map[string]string{},
			Stats:    map[string]*fleet.Stats{},
		}, http.StatusOK, &resp)
	}
	return host, readQueries, submit
}

func (s *integrationEnterpriseTestSuite) TestAIAgentSessionsDetailQueries() {
	t := s.T()
	ctx := t.Context()
	host, readQueries, submit := s.aiSessionsTestHost()

	// No sessions stored yet: sessions are asked for, those that started in the last 30 days by the host's clock;
	// events aren't.
	dqResp := readQueries()
	require.Contains(t, dqResp.Queries, "fleet_detail_query_ai_agent_sessions")
	require.Contains(t, dqResp.Queries["fleet_detail_query_ai_agent_sessions"], "WHERE started_at >= (SELECT unix_time FROM time) - 2592000")
	require.Contains(t, dqResp.Discovery["fleet_detail_query_ai_agent_sessions"], "name = 'ai_agent_sessions'")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_ai_agent_session_events")

	// Every session is collected from its first call.
	submit("fleet_detail_query_ai_agent_sessions", `[{
		"agent": "claude_code", "session_id": "s1", "username": "alice", "account_identifier": "alice@example.com",
		"started_at": "1800000000", "ended_at": "1800007200", "tool_calls": "4",
		"transcript_path": "/Users/alice/.claude/projects/app/s1.jsonl"
	}, {
		"agent": "claude_code", "session_id": "s2", "username": "alice", "account_identifier": "alice@example.com",
		"started_at": "1800005000", "ended_at": "1800007100", "tool_calls": "1",
		"transcript_path": "/Users/alice/.claude/projects/app/s2.jsonl"
	}]`)
	dqResp = readQueries()
	require.Contains(t, dqResp.Queries, "fleet_detail_query_ai_agent_session_events")
	eventsQuery := dqResp.Queries["fleet_detail_query_ai_agent_session_events"]
	require.Contains(t, eventsQuery, "transcript_path = '/Users/alice/.claude/projects/app/s1.jsonl' AND block_type = 'tool_use' AND tool_call_index >= 0\n")
	require.Contains(t, eventsQuery, "transcript_path = '/Users/alice/.claude/projects/app/s2.jsonl' AND block_type = 'tool_use' AND tool_call_index >= 0")
	require.Contains(t, eventsQuery, "1800007200 AS session_ended_at")
	require.Contains(t, dqResp.Discovery["fleet_detail_query_ai_agent_session_events"], "name = 'ai_agent_session_messages'")

	// s1's call 3 was still running when read.
	submit("fleet_detail_query_ai_agent_session_events", `[
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "0", "tool_name": "Read",
		 "tool_use_id": "toolu_0", "action": "read", "target": "go.mod", "decision": "allowed", "timestamp": "1800001000",
		 "content": "{\"file_path\":\"go.mod\"}", "content_length": "22", "truncated": "0", "session_ended_at": "1800007200"},
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "1", "tool_name": "Bash",
		 "tool_use_id": "toolu_1", "action": "execute", "target": "ls", "decision": "allowed", "timestamp": "1800005000",
		 "content": "{\"command\":\"ls\"}", "content_length": "16", "truncated": "0", "session_ended_at": "1800007200"},
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "2", "tool_name": "Bash",
		 "tool_use_id": "toolu_2", "action": "execute", "target": "cat big.txt", "decision": "allowed", "timestamp": "1800006000",
		 "content": "{\"command\":\"cat big.txt\"}", "content_length": "90000", "truncated": "1", "session_ended_at": "1800007200"},
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "3", "tool_name": "Agent",
		 "tool_use_id": "toolu_3", "action": "other", "target": "", "decision": "", "timestamp": "1800007000",
		 "content": "{\"prompt\":\"explore\"}", "session_ended_at": "1800007200"},
		{"agent": "claude_code", "session_id": "s2", "tool_call_index": "0", "tool_name": "Read",
		 "tool_use_id": "toolu_4", "action": "read", "target": "main.go", "decision": "allowed", "timestamp": "1800007000",
		 "content": "{\"file_path\":\"main.go\"}", "session_ended_at": "1800007100"}
	]`)

	// Both sessions are caught up, and neither has been active since: nothing to ask for.
	dqResp = readQueries()
	require.Contains(t, dqResp.Queries, "fleet_detail_query_ai_agent_sessions")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_ai_agent_session_events")

	// One write on the host recorded call 3's result along with call 4: catching up comes first.
	submit("fleet_detail_query_ai_agent_sessions", `[{
		"agent": "claude_code", "session_id": "s1", "username": "alice", "account_identifier": "alice@example.com",
		"started_at": "1800000000", "ended_at": "1800010000", "tool_calls": "5",
		"transcript_path": "/Users/alice/.claude/projects/app/s1.jsonl"
	}]`)
	dqResp = readQueries()
	eventsQuery = dqResp.Queries["fleet_detail_query_ai_agent_session_events"]
	require.Contains(t, eventsQuery, "transcript_path = '/Users/alice/.claude/projects/app/s1.jsonl' AND block_type = 'tool_use' AND tool_call_index >= 4")
	require.NotContains(t, eventsQuery, "s2.jsonl")
	submit("fleet_detail_query_ai_agent_session_events", `[
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "4", "tool_name": "Bash",
		 "tool_use_id": "toolu_5", "action": "execute", "target": "go test", "decision": "allowed", "timestamp": "1800009000",
		 "content": "{\"command\":\"go test\"}", "session_ended_at": "1800010000"}
	]`)

	// Caught up, the write that may hold call 3's decision is read again from that call.
	dqResp = readQueries()
	eventsQuery = dqResp.Queries["fleet_detail_query_ai_agent_session_events"]
	require.Contains(t, eventsQuery, "transcript_path = '/Users/alice/.claude/projects/app/s1.jsonl' AND block_type = 'tool_use' AND tool_call_index >= 3")
	submit("fleet_detail_query_ai_agent_session_events", `[
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "3", "tool_name": "Agent",
		 "tool_use_id": "toolu_3", "action": "other", "target": "", "decision": "allowed", "timestamp": "1800007000",
		 "content": "{\"prompt\":\"explore\"}", "session_ended_at": "1800010000"},
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "4", "tool_name": "Bash",
		 "tool_use_id": "toolu_5", "action": "execute", "target": "go test", "decision": "allowed", "timestamp": "1800009000",
		 "content": "{\"command\":\"go test\"}", "session_ended_at": "1800010000"}
	]`)
	dqResp = readQueries()
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_ai_agent_session_events")

	var stored []struct {
		AgentSessionID string `db:"agent_session_id"`
		Backfill       bool   `db:"backfill"`
		EventCount     uint   `db:"event_count"`
		Actions        string `db:"actions"`
		Decisions      string `db:"decisions"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &stored, `
			SELECT s.agent_session_id, s.backfill, s.event_count, s.actions,
				(SELECT GROUP_CONCAT(CONCAT(e.tool_call_index, ':', e.decision) ORDER BY e.tool_call_index)
					FROM host_ai_session_events e WHERE e.ai_session_id = s.id) AS decisions
			FROM host_ai_sessions s WHERE s.host_id = ? ORDER BY s.agent_session_id`, host.ID)
	})
	require.Len(t, stored, 2)
	require.Equal(t, "s1", stored[0].AgentSessionID)
	require.False(t, stored[0].Backfill)
	require.EqualValues(t, 5, stored[0].EventCount)
	require.Equal(t, "read,execute,other", stored[0].Actions)
	require.Equal(t, "0:allowed,1:allowed,2:allowed,3:allowed,4:allowed", stored[0].Decisions)
	require.Equal(t, "s2", stored[1].AgentSessionID)
	require.EqualValues(t, 1, stored[1].EventCount)
	require.Equal(t, "read", stored[1].Actions)
	require.Equal(t, "0:allowed", stored[1].Decisions)

	var truncated []struct {
		ToolCallIndex uint `db:"tool_call_index"`
		ContentLength uint `db:"content_length"`
		Truncated     bool `db:"truncated"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &truncated, `
			SELECT e.tool_call_index, e.content_length, e.truncated FROM host_ai_session_events e
			JOIN host_ai_sessions s ON s.id = e.ai_session_id
			WHERE s.host_id = ? AND s.agent_session_id = 's1' AND e.tool_call_index <= 2 ORDER BY e.tool_call_index`, host.ID)
	})
	require.Len(t, truncated, 3)
	require.EqualValues(t, 22, truncated[0].ContentLength)
	require.False(t, truncated[0].Truncated)
	require.EqualValues(t, 90000, truncated[2].ContentLength)
	require.True(t, truncated[2].Truncated)
}

// aiSessionEventsArm is one session's arm of the events query, as fleetd's table sees it.
var aiSessionEventsArm = regexp.MustCompile(
	`SELECT '([^']*)' AS agent, '([^']*)' AS session_id, 0 AS tool_call_index, [^\n]* (\d+) AS session_ended_at, 1 AS requested\n` +
		`UNION ALL\n[^\n]*\nFROM ai_agent_session_messages\nWHERE transcript_path = '([^']*)' AND block_type = 'tool_use' AND tool_call_index >= (\d+)(?: AND tool_call_index < (\d+))?`)

// aiSessionsFleetdResultBudget is roughly how many bytes of results fleetd's table returns for one query.
const aiSessionsFleetdResultBudget = 1 << 20

// aiSessionsFleetdResult answers the events query the way fleetd's table does, for sessions with the given number of
// calls: one row per call, the arms sharing fleetd's result budget, with a call's first row admitted once the
// budget is spent. It returns the result and each session's bounds as the query asked for them, its upper bound -1
// when there is none.
func aiSessionsFleetdResult(t *testing.T, query string, calls map[string]int) (string, map[string][2]int) {
	arms := aiSessionEventsArm.FindAllStringSubmatch(query, -1)
	require.NotEmpty(t, arms, query)
	used := 0
	var rows []map[string]string
	bounds := make(map[string][2]int)
	for _, arm := range arms {
		agent, sessionID, endedAt, path, from, upTo := arm[1], arm[2], arm[3], arm[4], arm[5], arm[6]
		require.Equal(t, "/Users/alice/.claude/projects/app/"+sessionID+".jsonl", path)
		rows = append(rows, map[string]string{"agent": agent, "session_id": sessionID, "tool_call_index": "0", "session_ended_at": endedAt, "requested": "1"})
		start, err := strconv.Atoi(from)
		require.NoError(t, err)
		end := -1
		if upTo != "" {
			end, err = strconv.Atoi(upTo)
			require.NoError(t, err)
		}
		bounds[sessionID] = [2]int{start, end}
		last := calls[sessionID]
		if end >= 0 {
			last = min(last, end)
		}
		for i := start; i < last; i++ {
			row := map[string]string{
				"agent": agent, "session_id": sessionID, "tool_call_index": strconv.Itoa(i), "tool_name": "Bash",
				"tool_use_id": fmt.Sprintf("toolu_%d", i), "action": "execute", "target": "echo " + strconv.Itoa(i),
				"decision": "allowed", "timestamp": endedAt,
				"content": fmt.Sprintf(`{"command":"echo %d %s"}`, i, strings.Repeat("x", 200)), "session_ended_at": endedAt,
			}
			b, err := json.Marshal(row)
			require.NoError(t, err)
			if used+len(b) > aiSessionsFleetdResultBudget && i > start {
				break
			}
			used += len(b)
			rows = append(rows, row)
		}
	}
	b, err := json.Marshal(rows)
	require.NoError(t, err)
	return string(b), bounds
}

// TestAIAgentSessionEventsPaging collects three 3,000-call sessions through the events query, answering it the way
// fleetd's table does: one call per arm, sharing a ~1 MiB result budget, with a call's first row admitted once the
// budget is spent. The sessions are read newest first, so the older ones get one row per refresh until the newer
// are done; resuming after the calls stored, every read makes progress.
func (s *integrationEnterpriseTestSuite) TestAIAgentSessionEventsPaging() {
	t := s.T()
	ctx := t.Context()
	host, readQueries, submit := s.aiSessionsTestHost()

	const calls, sessions = 3000, 3
	callsBySession := make(map[string]int, sessions)
	var rows []string
	for i := range sessions {
		callsBySession[fmt.Sprintf("s%d", i)] = calls
		rows = append(rows, fmt.Sprintf(`{"agent": "claude_code", "session_id": "s%d", "username": "alice",
			"started_at": "1800000000", "ended_at": "%d", "tool_calls": "%d", "transcript_path": "/Users/alice/.claude/projects/app/s%d.jsonl"}`,
			i, 1800003600+i, calls, i))
	}
	readQueries()
	submit("fleet_detail_query_ai_agent_sessions", "["+strings.Join(rows, ",")+"]")

	refreshes := 0
	for {
		dqResp := readQueries()
		query, ok := dqResp.Queries["fleet_detail_query_ai_agent_session_events"]
		if !ok {
			break
		}
		refreshes++
		require.LessOrEqual(t, refreshes, 10, "the sessions never finish")
		result, _ := aiSessionsFleetdResult(t, query, callsBySession)
		submit("fleet_detail_query_ai_agent_session_events", result)
	}
	require.GreaterOrEqual(t, refreshes, 3, "the result budget was never shared")

	var stored []struct {
		AgentSessionID   string `db:"agent_session_id"`
		EventCount       uint   `db:"event_count"`
		EventsEmptyReads uint   `db:"events_empty_reads"`
		Events           uint   `db:"events"`
		LastIndex        uint   `db:"last_index"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &stored, `
			SELECT s.agent_session_id, s.event_count, s.events_empty_reads,
				(SELECT COUNT(*) FROM host_ai_session_events e WHERE e.ai_session_id = s.id) AS events,
				(SELECT MAX(tool_call_index) FROM host_ai_session_events e WHERE e.ai_session_id = s.id) AS last_index
			FROM host_ai_sessions s WHERE s.host_id = ? ORDER BY s.agent_session_id`, host.ID)
	})
	require.Len(t, stored, sessions)
	for _, st := range stored {
		require.EqualValues(t, calls, st.EventCount, st.AgentSessionID)
		require.EqualValues(t, calls, st.Events, st.AgentSessionID)
		require.EqualValues(t, calls-1, st.LastIndex, st.AgentSessionID)
		require.Zero(t, st.EventsEmptyReads, st.AgentSessionID)
	}
}

// TestAIAgentSessionEventsBackfill checks that a session that had been running for days when the host first reported
// it is read 100 calls at a time, alongside the live sessions, until it's caught up.
func (s *integrationEnterpriseTestSuite) TestAIAgentSessionEventsBackfill() {
	t := s.T()
	ctx := t.Context()
	host, readQueries, submit := s.aiSessionsTestHost()

	now := time.Now().Unix()
	sessionRows := func(liveCalls int) string {
		return fmt.Sprintf(`[{
			"agent": "claude_code", "session_id": "old", "username": "alice", "started_at": "%d", "ended_at": "%d",
			"tool_calls": "250", "transcript_path": "/Users/alice/.claude/projects/app/old.jsonl"
		}, {
			"agent": "claude_code", "session_id": "live", "username": "alice", "started_at": "%d", "ended_at": "%d",
			"tool_calls": "%d", "transcript_path": "/Users/alice/.claude/projects/app/live.jsonl"
		}]`, now-10*24*3600, now-9*24*3600, now-600, now-60+int64(liveCalls), liveCalls)
	}
	calls := map[string]int{"old": 250, "live": 3}
	readQueries()
	submit("fleet_detail_query_ai_agent_sessions", sessionRows(calls["live"]))

	read := func() map[string][2]int {
		dqResp := readQueries()
		query, ok := dqResp.Queries["fleet_detail_query_ai_agent_session_events"]
		if !ok {
			return nil
		}
		result, bounds := aiSessionsFleetdResult(t, query, calls)
		submit("fleet_detail_query_ai_agent_session_events", result)
		return bounds
	}
	require.Equal(t, map[string][2]int{"live": {0, -1}, "old": {0, 100}}, read())

	// The live session moves on while the backfill is still behind.
	calls["live"] = 5
	submit("fleet_detail_query_ai_agent_sessions", sessionRows(calls["live"]))
	require.Equal(t, map[string][2]int{"live": {3, -1}, "old": {100, 200}}, read())
	require.Equal(t, map[string][2]int{"old": {200, 300}}, read())
	require.Nil(t, read())

	var stored []struct {
		AgentSessionID string `db:"agent_session_id"`
		Backfill       bool   `db:"backfill"`
		EventCount     uint   `db:"event_count"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &stored, `
			SELECT agent_session_id, backfill, event_count FROM host_ai_sessions WHERE host_id = ? ORDER BY agent_session_id`, host.ID)
	})
	require.Len(t, stored, 2)
	require.Equal(t, "live", stored[0].AgentSessionID)
	require.False(t, stored[0].Backfill)
	require.EqualValues(t, 5, stored[0].EventCount)
	require.Equal(t, "old", stored[1].AgentSessionID)
	require.False(t, stored[1].Backfill, "the backfill flag clears once the session is caught up")
	require.EqualValues(t, 250, stored[1].EventCount)
}

func genDistributedReqWithEntraJoinUser(host *fleet.Host, userEmail string) submitDistributedQueryResultsRequestShim {
	results := make(map[string]json.RawMessage)
	rows := "[]"
	if userEmail != "" {
		rows = fmt.Sprintf(`[{"user_email": "%s"}]`, userEmail)
	}
	results["fleet_detail_query_entra_join_user_windows"] = json.RawMessage(rows)
	return submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: make(map[string]any),
		Messages: make(map[string]string),
		Stats:    map[string]*fleet.Stats{},
	}
}

// TestOrbitOsqueryReEnrollDoesNotChangeTeam verifies that re-enrolling a host
// via orbit or osquery with an enroll secret belonging to a different team than
// the host's current team does not change the host's team_id. This protects
// against unintended team transfers when a host re-enrolls using a different
// enroll secret than the one originally used to onboard it.
func (s *integrationEnterpriseTestSuite) TestOrbitOsqueryReEnrollDoesNotChangeTeam() {
	t := s.T()
	ctx := context.Background()

	teamASecret := "team-a-reenroll-secret" //nolint:gosec // G101: test value only
	teamBSecret := "team-b-reenroll-secret" //nolint:gosec // G101: test value only

	teamA, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:    t.Name() + "_teamA",
		Secrets: []*fleet.EnrollSecret{{Secret: teamASecret}},
	})
	require.NoError(t, err)

	// Team B exists only to own teamBSecret. The host should never end up in it.
	_, err = s.ds.NewTeam(ctx, &fleet.Team{
		Name:    t.Name() + "_teamB",
		Secrets: []*fleet.EnrollSecret{{Secret: teamBSecret}},
	})
	require.NoError(t, err)

	t.Run("orbit", func(t *testing.T) {
		hostUUID := uuid.New().String()
		hostSerial := uuid.New().String()
		hostName := t.Name() + ".local"

		// Initial orbit enroll with team A's secret — host should be created in team A.
		var orbitResp enrollOrbitResponse
		s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
			EnrollSecret:   teamASecret,
			HardwareUUID:   hostUUID,
			HardwareSerial: hostSerial,
			Hostname:       hostName,
			Platform:       "darwin",
			HardwareModel:  "MacBookPro16,1",
		}, http.StatusOK, &orbitResp)
		require.NotEmpty(t, orbitResp.OrbitNodeKey)

		hostLite, err := s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, hostLite.TeamID)
		require.Equal(t, teamA.ID, *hostLite.TeamID)

		// Re-enroll the same host using team B's secret. team_id must not change.
		var reorbitResp enrollOrbitResponse
		s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
			EnrollSecret:   teamBSecret,
			HardwareUUID:   hostUUID,
			HardwareSerial: hostSerial,
			Hostname:       hostName,
			Platform:       "darwin",
			HardwareModel:  "MacBookPro16,1",
		}, http.StatusOK, &reorbitResp)
		require.NotEmpty(t, reorbitResp.OrbitNodeKey)

		hostLite, err = s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, hostLite.TeamID)
		require.Equal(t, teamA.ID, *hostLite.TeamID)
	})

	t.Run("osquery", func(t *testing.T) {
		hostUUID := uuid.New().String()
		hostSerial := uuid.New().String()

		// Initial osquery enroll with team A's secret — host should be created in team A.
		var osqueryResp contract.EnrollOsqueryAgentResponse
		s.DoJSON("POST", "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
			EnrollSecret:   teamASecret,
			HostIdentifier: hostUUID,
			HostDetails: map[string]map[string]string{
				"osquery_info": {
					"instance_id": hostUUID,
				},
				"system_info": {
					"hardware_serial": hostSerial,
					"uuid":            hostUUID,
				},
			},
		}, http.StatusOK, &osqueryResp)
		require.NotEmpty(t, osqueryResp.NodeKey)

		hostLite, err := s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, hostLite.TeamID)
		require.Equal(t, teamA.ID, *hostLite.TeamID)

		// Age last_enrolled_at to bypass the osquery enroll cooldown (default 42m in tests).
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				ctx,
				"UPDATE hosts SET last_enrolled_at = DATE_SUB(NOW(), INTERVAL '1' HOUR) WHERE id = ?",
				hostLite.ID,
			)
			return err
		})

		// Re-enroll the same host using team B's secret. team_id must not change.
		var reosqueryResp contract.EnrollOsqueryAgentResponse
		s.DoJSON("POST", "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
			EnrollSecret:   teamBSecret,
			HostIdentifier: hostUUID,
			HostDetails: map[string]map[string]string{
				"osquery_info": {
					"instance_id": hostUUID,
				},
				"system_info": {
					"hardware_serial": hostSerial,
					"uuid":            hostUUID,
				},
			},
		}, http.StatusOK, &reosqueryResp)
		require.NotEmpty(t, reosqueryResp.NodeKey)

		hostLite, err = s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, hostLite.TeamID)
		require.Equal(t, teamA.ID, *hostLite.TeamID)
	})
}

// TestOrbitEnrollWithIdPPopulatesDeviceMapping covers issue #45066: orbit
// enrolling a Linux or Windows host through the End User Authentication flow
// must populate host_emails so the hosts list endpoint returns
// device_mapping. The flow being exercised is:
//
//  1. Orbit calls /api/fleet/orbit/enroll. End User Auth is required, there
//     is no IdP association for this host UUID yet, so the server returns
//     END_USER_AUTH_REQUIRED.
//  2. Orbit opens a browser, the user authenticates, and the SSO callback
//     in ee/server/service/mdm.go writes the mdm_idp_accounts +
//     host_mdm_idp_accounts rows via AssociateHostMDMIdPAccountFromSSO
//     (the hosts row does not exist yet at this point — only the IdP-side
//     tables are populated). We simulate that callback by inserting the
//     same rows directly.
//  3. Orbit retries /api/fleet/orbit/enroll. EnrollOrbit creates the hosts
//     row, and the post-EnrollOrbit reconcile in server/service/orbit.go
//     calls AssociateHostMDMIdPAccount to populate host_emails.
//
// Pre-fix behavior: host_emails stayed empty, so ListHostDeviceMapping
// returned nothing and the hosts list endpoint returned device_mapping=null
// even though the single-host endpoint showed the IdP user via SCIM.
//
// Steps 1 and 3 also settle the end-user-auth marker that gates the
// unauthenticated setup experience SSO flow. Unit tests drive those branches
// against an injected store; this is the only place the store is the one the
// running server built for itself, so it is what proves the marker is wired at
// all.
func (s *integrationEnterpriseTestSuite) TestOrbitEnrollWithIdPPopulatesDeviceMapping() {
	t := s.T()
	ctx := t.Context()

	// Create a team with End User Authentication enabled, then use that
	// team's enroll secret. We use a team-level flag rather than the
	// app-config level because the enterprise suite enables EnableCachedDS,
	// and writes via s.ds bypass the running server's cached AppConfig
	// wrapper. Team config is read via TeamLite (not in the cached layer),
	// so a direct s.ds write is visible to the server immediately.
	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: t.Name(),
		Config: fleet.TeamConfig{
			MDM: fleet.TeamMDM{
				MacOSSetup: fleet.MacOSSetup{
					EnableEndUserAuthentication: true,
				},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		// Use a fresh background context — t.Context() has already been
		// cancelled by the time cleanup runs, and DeleteTeam opens its own
		// transaction.
		require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID))
	})

	enrollSecret := t.Name() + "-secret"
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &team.ID, []*fleet.EnrollSecret{{Secret: enrollSecret}}))

	// Orbit client capabilities — Linux and Windows orbit builds advertise
	// CapabilityEndUserAuth. The X-Fleet-Capabilities header is an
	// informational hint only: EUA gating must hold regardless of what
	// the client advertises.
	var caps fleet.CapabilityMap
	caps.PopulateFromString(string(fleet.CapabilityEndUserAuth))
	capsHeaders := map[string]string{fleet.CapabilitiesHeader: caps.String()}

	runFlow := func(t *testing.T, platform, platformLike string) {
		hostUUID := uuid.New().String()
		hostSerial := uuid.New().String()
		hostName := strings.ReplaceAll(t.Name(), "/", "-") + ".local"
		idpEmail := strings.ReplaceAll(uuid.NewString(), "-", "") + "@example.com"

		enrollBody, err := json.Marshal(fleet.EnrollOrbitRequest{
			EnrollSecret:   enrollSecret,
			HardwareUUID:   hostUUID,
			HardwareSerial: hostSerial,
			Hostname:       hostName,
			Platform:       platform,
			PlatformLike:   platformLike,
			HardwareModel:  "TestModel",
		})
		require.NoError(t, err)

		// Step 1: First orbit enrollment attempt — no IdP linked yet, so the
		// server returns END_USER_AUTH_REQUIRED (HTTP 401).
		res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/enroll", enrollBody, http.StatusUnauthorized, capsHeaders)
		respBody, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)
		require.Contains(t, string(respBody), "END_USER_AUTH_REQUIRED",
			"expected EUA gating to block enrollment when no IdP is linked")

		// The hosts row must not exist yet — the orbit enroll path returned
		// before reaching EnrollOrbit.
		_, err = s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))

		// The 401 also records the marker, without which the setup experience
		// SSO flow below would refuse to start for this host UUID.
		kv := redis_key_value.New(s.redisPool)
		pending, err := shared_mdm.HasEndUserAuthPrompt(ctx, kv, hostUUID, time.Now())
		require.NoError(t, err)
		require.True(t, pending)

		// Step 2: Simulate the Orbit Setup Experience SSO callback writing the
		// IdP account and linking it to the host UUID. This matches what
		// ee/server/service/mdm.go's mdmSSOHandleCallbackAuth does after the
		// browser-based SSO completes (the user authenticates against the IdP,
		// the SAML response lands at /api/v1/fleet/mdm/sso/callback, and the
		// initiator is SSOInitiatorOrbitSetupExperience).
		require.NoError(t, s.ds.InsertMDMIdPAccount(ctx, &fleet.MDMIdPAccount{
			Username: idpEmail,
			Fullname: "Test User",
			Email:    idpEmail,
		}))
		idpAcct, err := s.ds.GetMDMIdPAccountByEmail(ctx, idpEmail)
		require.NoError(t, err)
		require.NotNil(t, idpAcct)
		// This is the function the SSO callback uses; the hosts row does not
		// exist yet so the reconciling variant would error on the host-id lookup.
		_, err = s.ds.AssociateHostMDMIdPAccountFromSSO(ctx, hostUUID, idpAcct.UUID, true)
		require.NoError(t, err)

		// host_mdm_idp_accounts is now populated, but host_emails is not.
		var preCount int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &preCount,
				`SELECT COUNT(*) FROM host_mdm_idp_accounts WHERE host_uuid = ?`, hostUUID)
		})
		require.Equal(t, 1, preCount)

		// Step 3: Orbit retries the enrollment — EnrollOrbit creates the hosts
		// row and the post-EnrollOrbit reconcile in orbit.go calls
		// AssociateHostMDMIdPAccount, which populates host_emails.
		res = s.DoRawWithHeaders("POST", "/api/fleet/orbit/enroll", enrollBody, http.StatusOK, capsHeaders)
		var orbitResp enrollOrbitResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&orbitResp))
		res.Body.Close()
		require.NotEmpty(t, orbitResp.OrbitNodeKey)

		// Enrolling ends the prompt, so a sign-in completing later can no longer
		// take this host's IdP binding over.
		pending, err = shared_mdm.HasEndUserAuthPrompt(ctx, kv, hostUUID, time.Now())
		require.NoError(t, err)
		require.False(t, pending)

		// Step 4: Verify host_emails got populated — this is what fixes the
		// hosts list endpoint returning device_mapping=null (issue #45066).
		hostLite, err := s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.NotZero(t, hostLite.ID)

		dms, err := s.ds.ListHostDeviceMapping(ctx, hostLite.ID)
		require.NoError(t, err)
		require.Len(t, dms, 1,
			"host_emails must be populated after the post-EnrollOrbit reconcile (issue #45066)")
		require.Equal(t, idpEmail, dms[0].Email)
		require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, dms[0].Source)
	}

	t.Run("linux", func(t *testing.T) {
		runFlow(t, "ubuntu", "debian")
	})

	t.Run("windows", func(t *testing.T) {
		runFlow(t, "windows", "")
	})
}

// TestOrbitReEnrollSkipsEndUserAuth covers issue #46300: once a host has orbit-enrolled, a subsequent re-enrollment (e.g. after a
// service restart, node key file loss, or osquery DB rebuild) must not prompt for end user authentication again, even if End User
// Authentication is enabled on the host's team and the host has no IdP association. This grandfathers hosts that enrolled before
// EUA was enabled. A genuinely new device is still gated (covered by TestOrbitEnrollWithIdPPopulatesDeviceMapping).
func (s *integrationEnterpriseTestSuite) TestOrbitReEnrollSkipsEndUserAuth() {
	t := s.T()
	ctx := t.Context()

	// Create a team with End User Authentication DISABLED, so the first enrollment succeeds without an SSO prompt and the
	// host gets an orbit node key (the "previously enrolled" state).
	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: t.Name(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID))
	})

	enrollSecret := t.Name() + "-secret"
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &team.ID, []*fleet.EnrollSecret{{Secret: enrollSecret}}))

	var caps fleet.CapabilityMap
	caps.PopulateFromString(string(fleet.CapabilityEndUserAuth))
	capsHeaders := map[string]string{fleet.CapabilitiesHeader: caps.String()}

	runFlow := func(t *testing.T, platform, platformLike string) {
		hostUUID := uuid.New().String()
		enrollBody, err := json.Marshal(fleet.EnrollOrbitRequest{
			EnrollSecret:   enrollSecret,
			HardwareUUID:   hostUUID,
			HardwareSerial: uuid.New().String(),
			Hostname:       strings.ReplaceAll(t.Name(), "/", "-") + ".local",
			Platform:       platform,
			PlatformLike:   platformLike,
			HardwareModel:  "TestModel",
		})
		require.NoError(t, err)

		// First enrollment with EUA disabled: succeeds and stores an orbit node key.
		res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/enroll", enrollBody, http.StatusOK, capsHeaders)
		var orbitResp enrollOrbitResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&orbitResp))
		res.Body.Close()
		require.NotEmpty(t, orbitResp.OrbitNodeKey)

		hostLite, err := s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.NotZero(t, hostLite.ID)

		// Now enable End User Authentication on the team. EnrollOrbit reads team config via TeamLite (not in the cached
		// layer), so this direct write is visible to the running server immediately.
		team.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
		_, err = s.ds.SaveTeam(ctx, team)
		require.NoError(t, err)

		// Re-enroll the same host. There is no IdP association, so before the fix this returned END_USER_AUTH_REQUIRED. The
		// host is already orbit-enrolled, so enrollment must now succeed without prompting.
		res = s.DoRawWithHeaders("POST", "/api/fleet/orbit/enroll", enrollBody, http.StatusOK, capsHeaders)
		orbitResp = enrollOrbitResponse{}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&orbitResp))
		res.Body.Close()
		require.NotEmpty(t, orbitResp.OrbitNodeKey, "re-enrollment of an already-enrolled host must not be gated by EUA")

		// The re-enrollment must reuse the existing host row, not create a duplicate.
		reEnrolledHost, err := s.ds.HostLiteByIdentifier(ctx, hostUUID)
		require.NoError(t, err)
		require.Equal(t, hostLite.ID, reEnrolledHost.ID, "re-enrollment must reuse the existing host row, not create a duplicate")

		// Reset team EUA for the next subtest's first enrollment.
		team.Config.MDM.MacOSSetup.EnableEndUserAuthentication = false
		_, err = s.ds.SaveTeam(ctx, team)
		require.NoError(t, err)
	}

	t.Run("linux", func(t *testing.T) {
		runFlow(t, "ubuntu", "debian")
	})

	t.Run("windows", func(t *testing.T) {
		runFlow(t, "windows", "")
	})
}

// TestOrbitEnrollWithEUAToken covers the Windows MSI EUA-token branch in
// EnrollOrbit (the `case platform == "windows" && euaToken != ""` arm of the
// End User Auth switch in server/service/orbit.go). This is the flow Fleet
// uses when it enqueues a fleetd MSI install command on a Windows host that
// already enrolled in Windows MDM with an Azure UPN: the MSI carries a
// Fleet-signed EUA JWT so orbit can finish enrollment without prompting the
// user for SSO again.
//
// The test exercises the full server-side flow:
//
//  1. An mdm_windows_enrollments row exists for some MDM device, with a
//     valid Azure UPN captured during MDM enrollment.
//  2. The orbit request to /api/fleet/orbit/enroll carries a Fleet-signed
//     EUA token (issued in production by svc.generateWindowsEUAToken).
//  3. processWindowsEUAToken validates the JWT and resolves the mdm IdP
//     account for the UPN (creating one if needed).
//  4. EnrollOrbit creates the hosts row, then the post-EnrollOrbit reconcile
//     in orbit.go calls AssociateHostMDMIdPAccount with that IdP account
//     UUID, populating host_emails so the hosts list endpoint returns
//     device_mapping (issue #45066).
func (s *integrationEnterpriseTestSuite) TestOrbitEnrollWithEUAToken() {
	t := s.T()
	ctx := t.Context()

	// Build our own wstepCertManager using the same cert/key the test server
	// was configured with (SetupSuite points WindowsWSTEPIdentityCert/Key at
	// testdata/server.pem and server.key). The server's wstepCertManager
	// validates the token; ours signs it.
	certPEM, err := os.ReadFile("testdata/server.pem")
	require.NoError(t, err)
	keyPEM, err := os.ReadFile("testdata/server.key")
	require.NoError(t, err)
	certManager, err := microsoft_mdm.NewCertManager(nil, certPEM, keyPEM)
	require.NoError(t, err)

	// Team with End User Authentication enabled so the orbit handler reaches
	// the EUA switch. Team config is read via TeamLite which is not in the
	// caching layer, so direct s.ds writes are visible to the server.
	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: t.Name(),
		Config: fleet.TeamConfig{
			MDM: fleet.TeamMDM{
				MacOSSetup: fleet.MacOSSetup{
					EnableEndUserAuthentication: true,
				},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID))
	})

	enrollSecret := t.Name() + "-secret"
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &team.ID, []*fleet.EnrollSecret{{Secret: enrollSecret}}))

	// Pre-create the Windows MDM enrollment that the EUA token will reference.
	// In production this row is written when the device enrolls in Windows
	// MDM (Autopilot/Azure-joined). processWindowsEUAToken looks it up by
	// mdm_device_id to verify the token came from a real enrolled device.
	hostUUID := uuid.New().String()
	hostSerial := uuid.New().String()
	mdmDeviceID := uuid.New().String()
	idpEmail := strings.ReplaceAll(uuid.NewString(), "-", "") + "@example.com"
	require.NoError(t, s.ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            mdmDeviceID,
		MDMHardwareID:          uuid.New().String(),
		MDMDeviceState:         "enrolled",
		MDMDeviceType:          "Windows",
		MDMDeviceName:          "win-eua-test",
		MDMEnrollType:          "DeviceEnrollmentManager",
		MDMEnrollUserID:        idpEmail,
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045",
		MDMNotInOOBE:           true,
		HostUUID:               "",
	}))

	// Sign an EUA token the same way svc.generateWindowsEUAToken would.
	euaToken, err := certManager.NewEUAToken(idpEmail, mdmDeviceID)
	require.NoError(t, err)
	require.NotEmpty(t, euaToken)

	// Capabilities header — the EUA branch requires the orbit client to
	// advertise CapabilityEndUserAuth, otherwise the switch falls through
	// with a logged warning and enrollment is allowed without EUA.
	var caps fleet.CapabilityMap
	caps.PopulateFromString(string(fleet.CapabilityEndUserAuth))
	capsHeaders := map[string]string{fleet.CapabilitiesHeader: caps.String()}

	enrollBody, err := json.Marshal(fleet.EnrollOrbitRequest{
		EnrollSecret:   enrollSecret,
		HardwareUUID:   hostUUID,
		HardwareSerial: hostSerial,
		Hostname:       strings.ReplaceAll(t.Name(), "/", "-") + ".win.local",
		Platform:       "windows",
		HardwareModel:  "TestModel",
		EUAToken:       euaToken,
	})
	require.NoError(t, err)

	// Single orbit enroll call — the EUA token authenticates the user, so
	// no second retry is required (unlike the browser-SSO flow).
	res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/enroll", enrollBody, http.StatusOK, capsHeaders)
	var orbitResp enrollOrbitResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&orbitResp))
	res.Body.Close()
	require.NotEmpty(t, orbitResp.OrbitNodeKey)

	// host_emails must be populated — issue #45066. Without the
	// post-EnrollOrbit reconcile, processWindowsEUAToken's write would have
	// landed in host_mdm_idp_accounts but never made it into host_emails,
	// so the hosts list endpoint would still return device_mapping=null.
	hostLite, err := s.ds.HostLiteByIdentifier(ctx, hostUUID)
	require.NoError(t, err)
	require.NotZero(t, hostLite.ID)

	dms, err := s.ds.ListHostDeviceMapping(ctx, hostLite.ID)
	require.NoError(t, err)
	require.Len(t, dms, 1, "host_emails must be populated after EUA-token orbit enrollment (issue #45066)")
	require.Equal(t, idpEmail, dms[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, dms[0].Source)
}

func (s *integrationEnterpriseTestSuite) TestEntraJoinUserDetailQueryPopulatesIdPVitals() {
	t := s.T()
	ctx := t.Context()

	scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{
		UserName:   "join.user@example.com",
		GivenName:  new("Join"),
		FamilyName: new("User"),
		Department: new("Engineering"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := s.ds.DeleteScimUser(context.Background(), scimUserID)
		require.NoError(t, err)
	})

	host := createOrbitEnrolledHost(t, "windows", "entra-join", s.ds)

	getEndUsers := func() []fleet.HostEndUser {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		return hostResp.Host.EndUsers
	}
	submit := func(userEmail string) {
		var resp submitDistributedQueryResultsResponse
		s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithEntraJoinUser(host, userEmail), http.StatusOK, &resp)
	}

	// nothing mapped yet
	require.Empty(t, getEndUsers())

	// the device reports the join user (any casing): vitals come from SCIM
	submit("Join.User@Example.com")
	endUsers := getEndUsers()
	require.Len(t, endUsers, 1)
	require.Equal(t, "join.user@example.com", endUsers[0].IdpUserName)
	require.Equal(t, "Join User", endUsers[0].IdpFullName)
	require.Equal(t, "Engineering", endUsers[0].Department)

	// the mapping is reported under the mdm_idp_accounts source, like the manual one,
	// on the single-host endpoint and on the hosts list
	var mappingResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID), nil, http.StatusOK, &mappingResp)
	require.Len(t, mappingResp.DeviceMapping, 1)
	require.Equal(t, "join.user@example.com", mappingResp.DeviceMapping[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, mappingResp.DeviceMapping[0].Source)
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "device_mapping", "true", "query", host.Hostname)
	require.Len(t, listResp.Hosts, 1)
	require.NotNil(t, listResp.Hosts[0].DeviceMapping)
	var listMappings []fleet.HostDeviceMapping
	require.NoError(t, json.Unmarshal(*listResp.Hosts[0].DeviceMapping, &listMappings))
	require.Len(t, listMappings, 1)
	require.Equal(t, "join.user@example.com", listMappings[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, listMappings[0].Source)

	// no rows (left Entra, or joined without a user): the mapping is removed
	submit("")
	require.Empty(t, getEndUsers())

	// a join user that is not provisioned in SCIM never maps
	submit("stranger@example.com")
	require.Empty(t, getEndUsers())

	// mapped again, then an admin clears it and the next refresh refills it
	submit("join.user@example.com")
	require.Len(t, getEndUsers(), 1)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping/idp", host.ID), deleteHostIDPRequest{}, http.StatusNoContent)
	require.Empty(t, getEndUsers())
	submit("join.user@example.com")
	require.Len(t, getEndUsers(), 1)

	// a manually set IdP username wins over what the device reports
	var putResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "manual.user@example.com", Source: "idp"}, http.StatusOK, &putResp)
	submit("join.user@example.com")
	endUsers = getEndUsers()
	require.Len(t, endUsers, 1)
	require.Equal(t, "manual.user@example.com", endUsers[0].IdpUserName)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID), nil, http.StatusOK, &mappingResp)
	require.Len(t, mappingResp.DeviceMapping, 1)
}
