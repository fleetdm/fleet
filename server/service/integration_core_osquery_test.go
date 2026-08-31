package service

// Osquery protocol tests for the core (no-license) suite.
//
// Belongs here: osquery enrollment and enroll-secret handling, the distributed
// read/write endpoints, generated osquery config, scheduled query stats ingest,
// and request body size limits.
//
// Does not belong here: orbit-specific config and enrollment
// (integration_core_orbit_test.go), and the report definitions whose results
// arrive over this protocol (integration_core_queries_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestSlowOsqueryHost() {
	t := s.T()
	_, server := RunServerForTestsWithDS(
		t,
		s.ds,
		&TestServerOpts{
			SkipCreateTestUsers: true,
			//nolint:gosec // G112: server is just run for testing this explicit config.
			HTTPServerConfig: &http.Server{ReadTimeout: 2 * time.Second},
			EnableCachedDS:   true,
		},
	)
	defer func() {
		server.Close()
	}()

	req, err := http.NewRequest("POST", server.URL+"/api/v1/osquery/distributed/write", &slowReader{})
	require.NoError(t, err)

	client := fleethttp.NewClient()

	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusRequestTimeout, resp.StatusCode)
}

func (s *integrationTestSuite) TestDistributedReadWithChangedQueries() {
	t := s.T()

	spec := []byte(`
  features:
    enable_software_inventory: true
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_macos: "SELECT * FROM foo;"
      unknown_query: "SELECT * FROM bar;"
`)
	s.applyConfig(spec)

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

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "SELECT 1 FROM osquery;"}, nil)

	// Ensure we can read distributed queries for the host.
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// Get distributed queries for the host.
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Equal(t, "SELECT * FROM foo;", dqResp.Queries["fleet_detail_query_software_macos"])

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	spec = []byte(`
  features:
    enable_software_inventory: true
    enable_host_users: true
    detail_query_overrides:
`)
	s.applyConfig(spec)

	// Get distributed queries for the host.
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries["fleet_detail_query_software_macos"], "FROM apps")
	require.Contains(t, dqResp.Queries["fleet_detail_query_users"], "FROM users")
}

func (s *integrationTestSuite) TestOsqueryConfig() {
	t := s.T()

	hosts := s.createHosts(t)
	req := getClientConfigRequest{NodeKey: *hosts[0].NodeKey}
	var resp getClientConfigResponse
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)

	// test with invalid node key
	var errRes map[string]any
	req.NodeKey += "zzzz"
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")
}

func (s *integrationTestSuite) TestOsqueryConfigETag() {
	t := s.T()

	hosts := s.createHosts(t)
	nodeKey := *hosts[0].NodeKey

	// Request bodies for the three etag states of the body-carried protocol:
	// field absent (legacy), field empty (opted in, no validator yet), and
	// field carrying an echoed validator.
	legacyRequestBody, err := json.Marshal(map[string]string{"node_key": nodeKey})
	require.NoError(t, err)
	etagRequestBody := func(etag string) []byte {
		b, err := json.Marshal(map[string]string{"node_key": nodeKey, "etag": etag})
		require.NoError(t, err)
		return b
	}
	readAll := func(resp *http.Response) []byte {
		t.Cleanup(func() { resp.Body.Close() })
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return b
	}
	etagOf := func(body []byte) string {
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(body, &decoded))
		etag, _ := decoded["etag"].(string)
		return etag
	}

	// 1. A legacy agent (no etag field) gets the config with no "etag" key —
	// byte-for-byte the pre-feature response.
	legacyBody := readAll(s.DoRaw("POST", "/api/osquery/config", legacyRequestBody, http.StatusOK))
	require.NotEmpty(t, legacyBody)
	require.NotContains(t, string(legacyBody), `"etag"`)

	// 2. An opted-in agent with no validator (empty etag) gets the full
	// config with the "etag" key added; the validator covers the etag-less
	// representation, i.e. the legacy body.
	optedInBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(""), http.StatusOK))
	expectedETag := clientConfigETag(legacyBody)
	require.Equal(t, expectedETag, etagOf(optedInBody))
	require.NotEqual(t, "ok", etagOf(optedInBody))

	// Stripping the etag key yields the same configuration. This is a
	// semantic (unmarshaled) comparison, not a byte comparison: pack content
	// is embedded as struct-marshaled json.RawMessage whose field order a
	// round-trip through map[string]any cannot preserve, so re-marshaling
	// here would alphabetize nested keys and spuriously mismatch whenever
	// earlier suite tests left scheduled reports behind. The byte-level
	// guarantee — the validator covers the exact legacy bytes — is already
	// pinned by the expectedETag assertion above.
	var legacyConfig, optedInConfig map[string]any
	require.NoError(t, json.Unmarshal(legacyBody, &legacyConfig))
	require.NoError(t, json.Unmarshal(optedInBody, &optedInConfig))
	delete(optedInConfig, "etag")
	require.Equal(t, legacyConfig, optedInConfig)

	// 3. Echoing the validator gets the constant unchanged body, HTTP 200.
	unchangedBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(expectedETag), http.StatusOK))
	require.JSONEq(t, configUnchangedBody, string(unchangedBody))

	// 4. A stale etag gets the full config again, current validator included.
	staleBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody("wrong-etag"), http.StatusOK))
	require.Equal(t, expectedETag, etagOf(staleBody))

	// 5. An empty etag is never answered "unchanged", even with the record
	// warm from the previous requests.
	emptyAgainBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(""), http.StatusOK))
	require.Equal(t, expectedETag, etagOf(emptyAgainBody))

	// 6. Invalid node key fails auth regardless of a matching etag.
	invalidBody, _ := json.Marshal(map[string]string{"node_key": "invalid-key", "etag": expectedETag})
	resp := s.DoRaw("POST", "/api/osquery/config", invalidBody, http.StatusUnauthorized)
	readAll(resp)

	// 7. The /api/v1/osquery/config alias speaks the same protocol.
	aliasBody := readAll(s.DoRaw("POST", "/api/v1/osquery/config", etagRequestBody(expectedETag), http.StatusOK))
	require.JSONEq(t, configUnchangedBody, string(aliasBody))

	// 8. A config change produces a new validator: the old etag downloads
	// the full new config, and the new etag is then answered "unchanged".
	//
	// This suite shares one server, so agent options may already hold anything
	// a previously executed test left behind. Derive the new value from the
	// current one so the PATCH cannot be a no-op, and restore the previous
	// options afterwards so this mutation does not leak into later tests.
	ctx := context.Background()
	appCfgBefore, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, appCfgBefore.AgentOptions)
	prevAgentOptions := append(json.RawMessage(nil), *appCfgBefore.AgentOptions...)
	t.Cleanup(func() {
		// Restored through the datastore rather than the API: the stored blob is
		// not necessarily accepted by the write validator (the suite's defaults
		// carry command-line flags that PATCH rejects inside config.options), and
		// the point here is to put back exactly what was there.
		appCfg, err := s.ds.AppConfig(ctx)
		require.NoError(t, err)
		appCfg.AgentOptions = &prevAgentOptions
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
	})

	// logger_tls_period is echoed into the rendered config, so bumping it past
	// whatever is there now guarantees the body — and therefore the validator —
	// changes.
	var currentOptions struct {
		Config struct {
			Options struct {
				LoggerTLSPeriod int `json:"logger_tls_period"`
			} `json:"options"`
		} `json:"config"`
	}
	require.NoError(t, json.Unmarshal(prevAgentOptions, &currentOptions))
	newPeriod := currentOptions.Config.Options.LoggerTLSPeriod + 7
	s.DoRaw("PATCH", "/api/latest/fleet/config",
		[]byte(fmt.Sprintf(`{"agent_options":{"config":{"options":{"logger_tls_period":%d}}}}`, newPeriod)),
		http.StatusOK)

	changedBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(expectedETag), http.StatusOK))
	require.Contains(t, string(changedBody), fmt.Sprintf(`"logger_tls_period": %d`, newPeriod),
		"the PATCH must actually change the rendered config, or the validator assertion below is vacuous")
	newETag := etagOf(changedBody)
	require.NotEmpty(t, newETag)
	require.NotEqual(t, "ok", newETag)
	require.NotEqual(t, expectedETag, newETag, "validator should have changed")

	unchangedAfterBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(newETag), http.StatusOK))
	require.JSONEq(t, configUnchangedBody, string(unchangedAfterBody))

	// 9. Legacy agents see the new config with no "etag" key.
	newLegacyBody := readAll(s.DoRaw("POST", "/api/osquery/config", legacyRequestBody, http.StatusOK))
	require.NotContains(t, string(newLegacyBody), `"etag"`)
	require.Equal(t, newETag, clientConfigETag(newLegacyBody))
}

func (s *integrationTestSuite) TestEnrollOsquery() {
	t := s.T()

	// set the enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// invalid enroll secret fails
	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   "nosuchsecret",
		HostIdentifier: "abcd",
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)

	// valid enroll secret succeeds
	j, err = json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: t.Name(),
	})
	require.NoError(t, err)

	var resp contract.EnrollOsqueryAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&resp))
	require.NotEmpty(t, resp.NodeKey)

	// A team may retain an empty enroll secret created before the create/update
	// validation existed. Simulate that by writing an empty secret directly via
	// the datastore, bypassing the service-layer validation.
	ctx := context.Background()
	emptyTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "empty"})
	require.NoError(t, err)
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &emptyTeam.ID, []*fleet.EnrollSecret{{Secret: "", TeamID: &emptyTeam.ID}}))

	// Enrolling with an empty or whitespace-only secret must be rejected as
	// node_invalid, even though an empty secret exists in storage.
	for _, badSecret := range []string{"", "   "} {
		j, err = json.Marshal(&contract.EnrollOsqueryAgentRequest{
			EnrollSecret:   badSecret,
			HostIdentifier: t.Name() + "empty-host",
		})
		require.NoError(t, err)
		badRes := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)
		var body map[string]any
		require.NoError(t, json.NewDecoder(badRes.Body).Decode(&body))
		badRes.Body.Close()
		require.Equal(t, true, body["node_invalid"])
	}
}

func (s *integrationTestSuite) TestTryingToEnrollWithTheWrongSecret() {
	t := s.T()
	ctx := context.Background()

	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   time.Now(),
		DetailUpdatedAt:  time.Now(),
		RefetchRequested: true,
	})
	require.NoError(t, err)

	var resp endpointer.JsonError
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   uuid.New().String(),
		HardwareUUID:   h.UUID,
		HardwareSerial: h.HardwareSerial,
	}, http.StatusUnauthorized, &resp)

	require.Equal(t, "Authentication failed", resp.Message)
}

func (s *integrationTestSuite) TestDirectIngestScheduledQueryStats() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "Foobar",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	team1Host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.team", t.Name()),
		Platform:        "darwin",
		TeamID:          &team1.ID,
	})
	require.NoError(t, err)
	scheduledGlobalQuery, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-global-query",
		TeamID:             nil,
		Interval:           10,
		Platform:           "darwin",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from time;",
		Saved:              true,
	})
	require.NoError(t, err)
	nonScheduledGlobalQuery, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "non-scheduled-global-query",
		TeamID:             nil,
		Interval:           0,
		Platform:           "darwin",
		AutomationsEnabled: false,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from osquery_info;",
		Saved:              true,
	})
	require.NoError(t, err)
	scheduledTeam1Query1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team1-query1",
		TeamID:             &team1.ID,
		Interval:           20,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	scheduledTeam1Query2, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team1-query2",
		TeamID:             &team1.ID,
		Interval:           90,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a non-scheduled query to test that we filter it out when providing
	// the queries in the osquery/config endpoint.
	_, err = s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "non-scheduled-team1-query",
		TeamID:             &team1.ID,
		Interval:           0,
		Platform:           "",
		AutomationsEnabled: false,
		Logging:            "snapshot",
		Description:        "foobar",
		Query:              "SELECT * from foobar;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a scheduled query but on another team to test that we filter it
	// out when providing the queries in the osquery/config endpoint.
	_, err = s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team2-query",
		TeamID:             &team2.ID,
		Interval:           40,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a legacy 2017 user pack with one query.
	userPack1TargetTeam1, err := s.ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "2017 Pack",
		Type:    nil,
		Teams:   []fleet.Target{{TargetID: team1.ID, Type: fleet.TargetTeam}},
		TeamIDs: []uint{team1.ID},
	})
	require.NoError(t, err)
	scheduledQueryOnPack1, err := s.ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "scheduled-query-pack1",
		PackID:   userPack1TargetTeam1.ID,
		QueryID:  nonScheduledGlobalQuery.ID,
		Interval: 60,
		Snapshot: new(true),
		Removed:  new(true),
	})
	require.NoError(t, err)

	// Simulate the osquery instance of the global host calling the osquery/config endpoint
	// and test the returned scheduled queries.
	req := getClientConfigRequest{NodeKey: *globalHost.NodeKey}
	var resp getClientConfigResponse
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
	packs := resp.Config["packs"].(map[string]any)
	require.Len(t, packs, 1)
	globalQueries := packs["Global"].(map[string]any)["queries"].(map[string]any)
	require.Len(t, globalQueries, 1)
	require.Contains(t, globalQueries, scheduledGlobalQuery.Name)

	// Simulate the osquery instance of the team host calling the osquery/config endpoint
	// and test the returned scheduled queries.
	req = getClientConfigRequest{NodeKey: *team1Host.NodeKey}
	resp = getClientConfigResponse{}
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
	packs = resp.Config["packs"].(map[string]any)
	require.Len(t, packs, 3)
	globalQueries = packs["Global"].(map[string]any)["queries"].(map[string]any)
	require.Len(t, globalQueries, 1)
	require.Contains(t, globalQueries, scheduledGlobalQuery.Name)
	team1Queries := packs[fmt.Sprintf("team-%d", team1.ID)].(map[string]any)["queries"].(map[string]any)
	require.Len(t, team1Queries, 2)
	require.Contains(t, team1Queries, scheduledTeam1Query1.Name)
	require.Contains(t, team1Queries, scheduledTeam1Query2.Name)
	userPack1Queries := packs[userPack1TargetTeam1.Name].(map[string]any)["queries"].(map[string]any)
	require.Len(t, userPack1Queries, 1)
	require.Contains(t, userPack1Queries, scheduledQueryOnPack1.Name)

	// Now let's simulate a osquery instance running in the team host returning the
	// stats in the distributed/write (osquery_schedule table)
	rows := []map[string]string{
		{
			"name":              "pack/Global/scheduled-global-query",
			"query":             "SELECT * FROM time;",
			"interval":          "10",
			"executions":        "2",
			"last_executed":     "1693476753",
			"denylisted":        "0",
			"output_size":       "576",
			"wall_time":         "1",
			"wall_time_ms":      "2",
			"last_wall_time_ms": "3",
			"user_time":         "4",
			"last_user_time":    "5",
			"system_time":       "6",
			"last_system_time":  "7",
			"average_memory":    "8",
			"last_memory":       "9",
			"delimiter":         "/",
		},
		{
			"name":              "pack/2017 Pack/scheduled-query-pack1",
			"query":             "SELECT * FROM osquery_info;",
			"interval":          "60",
			"executions":        "20",
			"last_executed":     "1693476842",
			"denylisted":        "0",
			"output_size":       "9620",
			"wall_time":         "9",
			"wall_time_ms":      "8",
			"last_wall_time_ms": "7",
			"user_time":         "6",
			"last_user_time":    "5",
			"system_time":       "4",
			"last_system_time":  "3",
			"average_memory":    "2",
			"last_memory":       "1",
			"delimiter":         "/",
		},
		{
			"name":              fmt.Sprintf("pack/team-%d/scheduled-team1-query1", team1.ID),
			"query":             "SELECT * FROM other;",
			"interval":          "20",
			"executions":        "1",
			"last_executed":     "1693476561",
			"denylisted":        "0",
			"output_size":       "10",
			"wall_time":         "11",
			"wall_time_ms":      "12",
			"last_wall_time_ms": "13",
			"user_time":         "14",
			"last_user_time":    "15",
			"system_time":       "16",
			"last_system_time":  "17",
			"average_memory":    "18",
			"last_memory":       "19",
			"delimiter":         "/",
		},
		{
			"name":              fmt.Sprintf("pack/team-%d/scheduled-team1-query2", team1.ID),
			"query":             "SELECT * FROM other;",
			"interval":          "90",
			"executions":        "5",
			"last_executed":     "1693476666",
			"denylisted":        "0",
			"output_size":       "20",
			"wall_time":         "21",
			"wall_time_ms":      "22",
			"last_wall_time_ms": "23",
			"user_time":         "24",
			"last_user_time":    "25",
			"system_time":       "26",
			"last_system_time":  "27",
			"average_memory":    "28",
			"last_memory":       "29",
			"delimiter":         "/",
		},
	}

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{
		App: config.AppConfig{
			EnableScheduledQueryStats: true,
		},
	}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	task := async.NewTask(s.ds, nil, clock.C, nil)
	err = detailQueries["scheduled_query_stats"].DirectTaskIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		team1Host,
		task,
		rows,
	)
	require.NoError(t, err)

	// Check that the received stats were stored in the DB as expected.
	var scheduledQueriesStats []fleet.ScheduledQueryStats
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(
			context.Background(), q, &scheduledQueriesStats,
			`SELECT
				scheduled_query_id, q.name AS scheduled_query_name, average_memory, denylisted,
				executions, q.schedule_interval, last_executed,
				output_size, system_time, user_time, wall_time
			FROM scheduled_query_stats sqs
			JOIN queries q ON sqs.scheduled_query_id = q.id
			WHERE host_id = ?;`,
			team1Host.ID,
		)
	})
	require.Len(t, scheduledQueriesStats, 4)
	rowsMap := make(map[string]map[string]string)
	for _, row := range rows {
		parts := strings.Split(row["name"], "/")
		queryName := parts[len(parts)-1]
		// we need to map this because 2017 packs send the name of the schedule and not
		// the name of the query.
		if queryName == "scheduled-query-pack1" {
			queryName = "non-scheduled-global-query"
		}
		rowsMap[queryName] = row
	}
	for _, sqs := range scheduledQueriesStats {
		row := rowsMap[sqs.ScheduledQueryName]
		require.Equal(t, fmt.Sprint(sqs.AverageMemory), row["average_memory"])
		require.Equal(t, fmt.Sprint(sqs.Executions), row["executions"])
		interval := row["interval"]
		if sqs.ScheduledQueryName == "non-scheduled-global-query" {
			interval = "0" // this query has metrics because it runs on a pack.
		}
		require.Equal(t, strconv.FormatInt(int64(sqs.Interval), 10), interval)
		lastExecuted, err := strconv.ParseInt(row["last_executed"], 10, 64)
		require.NoError(t, err)
		require.WithinDuration(t, sqs.LastExecuted, time.Unix(lastExecuted, 0), 1*time.Second)
		require.Equal(t, fmt.Sprint(sqs.OutputSize), row["output_size"])
		require.Equal(t, fmt.Sprint(sqs.SystemTime), row["system_time"])
		require.Equal(t, fmt.Sprint(sqs.UserTime), row["user_time"])
		assert.Equal(t, fmt.Sprint(sqs.WallTime), row["wall_time_ms"])
	}

	// Now let's simulate a osquery instance running in the global host returning the
	// stats in the distributed/write (osquery_schedule table)
	rows = []map[string]string{
		{
			"name":              "pack/Global/scheduled-global-query",
			"query":             "SELECT * FROM time;",
			"interval":          "10",
			"executions":        "2",
			"last_executed":     "1693476753",
			"denylisted":        "0",
			"output_size":       "576",
			"wall_time":         "1",
			"wall_time_ms":      "2",
			"last_wall_time_ms": "3",
			"user_time":         "4",
			"last_user_time":    "5",
			"system_time":       "6",
			"last_system_time":  "7",
			"average_memory":    "8",
			"last_memory":       "9",
			"delimiter":         "/",
		},
	}

	err = detailQueries["scheduled_query_stats"].DirectTaskIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		task,
		rows,
	)
	require.NoError(t, err)

	// Check that the received stats were stored in the DB as expected.
	scheduledQueriesStats = []fleet.ScheduledQueryStats{}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(
			context.Background(), q, &scheduledQueriesStats,
			`SELECT
				scheduled_query_id, q.name AS scheduled_query_name, average_memory, denylisted,
				executions, q.schedule_interval, last_executed,
				output_size, system_time, user_time, wall_time
			FROM scheduled_query_stats sqs
			JOIN queries q ON sqs.scheduled_query_id = q.id
			WHERE host_id = ?;`,
			globalHost.ID,
		)
	})
	require.Len(t, scheduledQueriesStats, 1)
	row := rows[0]
	parts := strings.Split(row["name"], "/")
	queryName := parts[len(parts)-1]
	sqs := scheduledQueriesStats[0]
	require.Equal(t, scheduledQueriesStats[0].ScheduledQueryName, queryName)
	require.Equal(t, fmt.Sprint(sqs.AverageMemory), row["average_memory"])
	require.Equal(t, fmt.Sprint(sqs.Executions), row["executions"])
	require.Equal(t, fmt.Sprint(sqs.Interval), row["interval"])
	lastExecuted, err := strconv.ParseInt(row["last_executed"], 10, 64)
	require.NoError(t, err)
	require.WithinDuration(t, sqs.LastExecuted, time.Unix(lastExecuted, 0), 1*time.Second)
	require.Equal(t, fmt.Sprint(sqs.OutputSize), row["output_size"])
	require.Equal(t, fmt.Sprint(sqs.SystemTime), row["system_time"])
	require.Equal(t, fmt.Sprint(sqs.UserTime), row["user_time"])
	require.Equal(t, fmt.Sprint(sqs.WallTime), row["wall_time_ms"])
}

func (s *integrationTestSuite) TestOsqueryBodySizeLimit() {
	t := s.T()

	host := createOrbitEnrolledHost(t, "linux", "body-limit", s.ds)

	logLimit := int(fleet.DefaultMaxOsqueryLogWriteSize)
	distLimit := int(fleet.DefaultMaxOsqueryDistributedWriteSize)

	// Body over the per-route default must be rejected with 413. The padding
	// is inside a JSON string value so the body is syntactically valid up to
	// the point where the reader is cut off.
	logPrefix := fmt.Sprintf(`{"node_key":%q,"log_type":"status","data":["`, *host.NodeKey)
	logSuffix := `"]}`
	logPadSize := logLimit + 1 - len(logPrefix) - len(logSuffix)
	require.Positive(t, logPadSize, "padding must be positive")
	overLimitLog := []byte(logPrefix + strings.Repeat("x", logPadSize) + logSuffix)
	s.DoRawNoAuth("POST", "/api/osquery/log", overLimitLog, http.StatusRequestEntityTooLarge)

	// A well-formed body within the limit is accepted.
	withinLimitLog, err := json.Marshal(submitLogsRequest{
		NodeKey: *host.NodeKey,
		LogType: "status",
		Data:    []json.RawMessage{},
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/log", withinLimitLog, http.StatusOK)

	// A truncated (malformed) body within the limit must NOT return 413.
	// Before the fix, io.ErrUnexpectedEOF from the JSON decoder was incorrectly
	// converted to PayloadTooLargeError even when the reader had not been exhausted.
	// The correct response is 400 Bad Request.
	truncatedLog := fmt.Appendf(nil, `{"node_key":%q,"log_type":"status","data":[`, *host.NodeKey) // missing closing ]}
	s.DoRawNoAuth("POST", "/api/osquery/log", truncatedLog, http.StatusBadRequest)

	// Body over the per-route default must be rejected with 413.
	distPrefix := fmt.Sprintf(`{"node_key":%q,"queries":{"q1":[{"data":"`, *host.NodeKey)
	distSuffix := `"}]},"statuses":{"q1":0},"messages":{},"stats":{}}`
	distPadSize := distLimit + 1 - len(distPrefix) - len(distSuffix)
	require.Positive(t, distPadSize, "padding must be positive")
	overLimitDist := []byte(distPrefix + strings.Repeat("x", distPadSize) + distSuffix)
	s.DoRawNoAuth("POST", "/api/osquery/distributed/write", overLimitDist, http.StatusRequestEntityTooLarge)

	// A well-formed body within the limit is accepted.
	withinLimitDist, err := json.Marshal(submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  map[string]json.RawMessage{},
		Statuses: map[string]any{},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/distributed/write", withinLimitDist, http.StatusOK)

	// A truncated body within the limit must NOT return 413 (same false-positive guard).
	// io.ErrUnexpectedEOF from the bodyDecoder path is now wrapped as BadRequestErr → 400.
	truncatedDist := fmt.Appendf(nil, `{"node_key":%q,"queries":{"q1":[`, *host.NodeKey) // missing closing
	s.DoRawNoAuth("POST", "/api/osquery/distributed/write", truncatedDist, http.StatusBadRequest)

	s.Run("config overrides take effect in body-auth mode", func() {
		// Spin up a second server with custom per-route limits and
		// confirm bodies above the override are rejected while bodies
		// below are accepted.
		const customLimit = 2 * units.MiB

		cfg := config.TestConfig()
		cfg.Osquery.MaxLogWriteBodySize = customLimit
		cfg.Osquery.MaxDistributedWriteBodySize = customLimit

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		logPad := customLimit + 1 - len(logPrefix) - len(logSuffix)
		s.Require().Positive(logPad)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", logPad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/log", withinLimitLog, http.StatusOK)

		distPad := customLimit + 1 - len(distPrefix) - len(distSuffix)
		s.Require().Positive(distPad)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(distPrefix+strings.Repeat("x", distPad)+distSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write", withinLimitDist, http.StatusOK)
	})

	s.Run("header-auth mode imposes no body size limit", func() {
		// In header-auth mode the per-route configs are intentionally
		// ignored AND no body size limit applies. A body well above the
		// global default (and well above any per-route default) must
		// succeed when authenticated via header.
		cfg := config.TestConfig()
		cfg.Osquery.AllowBodyAuthFallback = false
		cfg.Osquery.MaxLogWriteBodySize = 1 * units.MiB         // ignored
		cfg.Osquery.MaxDistributedWriteBodySize = 1 * units.MiB // ignored

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// 12 MiB body — over both the global limit (1 MiB) and any
		// per-route default. Must succeed because header-auth mode
		// applies no body size constraint.
		oversizedLog, err := json.Marshal(submitLogsRequest{
			NodeKey: *host.NodeKey,
			LogType: "status",
			Data:    []json.RawMessage{json.RawMessage(`"` + strings.Repeat("x", 12*1024*1024) + `"`)},
		})
		s.Require().NoError(err)
		ts.DoRawWithHeaders("POST", "/api/osquery/log", oversizedLog,
			http.StatusOK,
			map[string]string{"Authorization": "NodeKey " + *host.NodeKey})
	})

	s.Run("endpoint_request_size_overrides wins over the osquery per-route default in body-auth mode", func() {
		const overrideLimit = 15 * units.MiB

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// Above the osquery route's own default (10MiB) but within the override (15MiB).
		// It must succeed, proving the override raised the effective limit.
		betweenPad := logLimit + 2*int(units.MiB) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", betweenPad)+logSuffix),
			http.StatusOK)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", betweenPad)+logSuffix),
			http.StatusOK)

		// Above the override itself. It must be rejected.
		aboveOverridePad := int(overrideLimit) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
	})

	s.Run("endpoint_request_size_overrides wins over an explicitly configured osquery_max_*_body_size  when larger", func() {
		// osquery_max_log_write_body_size & osquery_max_distributed_write_body_size (deprecated but supported) are explicitly set.
		// The override must still win over it when larger, proving the two config sources compose via the same "largest wins" comparison.
		const (
			deprecatedLimit = 5 * units.MiB
			overrideLimit   = 15 * units.MiB
		)

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		cfg.Osquery.MaxLogWriteBodySize = deprecatedLimit
		cfg.Osquery.MaxDistributedWriteBodySize = deprecatedLimit

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// Above the explicit deprecated-config limit (5MiB) but within the override (15MiB)
		// It must succeed.
		aboveDeprecatedPad := int(deprecatedLimit) + int(units.MiB) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusOK)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusOK)

		// Above the override itself. It must be rejected.
		aboveOverridePad := int(overrideLimit) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
	})

	s.Run("explicitly configured osquery_max_*_body_size wins when larger than endpoint_request_size_overrides", func() {
		// Reverse of above test.
		// A smaller override must not shrink the effective limit below an explicitly configured (larger) deprecated setting for the same path.
		const (
			deprecatedLimit = 15 * units.MiB
			overrideLimit   = 5 * units.MiB
		)

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		cfg.Osquery.MaxLogWriteBodySize = deprecatedLimit
		cfg.Osquery.MaxDistributedWriteBodySize = deprecatedLimit

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// Above the (smaller) override but within the explicit deprecated limit.
		// It must succeed, proving the override didn't shrink it.
		aboveOverridePad := int(overrideLimit) + int(units.MiB) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusOK)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusOK)

		// Above the deprecated limit itself. It must be rejected.
		aboveDeprecatedPad := int(deprecatedLimit) + 1 - len(logPrefix) - len(logSuffix)
		s.Require().Positive(aboveDeprecatedPad)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusRequestEntityTooLarge)
	})

	s.Run("endpoint_request_size_overrides does not apply in header-auth mode", func() {
		// Header-auth routes opt out of the size-limiting mechanism entirely (SkipRequestBodySizeLimit),
		// so a configured override for the same path must not reintroduce a limit there.
		const overrideLimit = 2 * units.MiB // well below the 12MiB body sent below

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		cfg.Osquery.AllowBodyAuthFallback = false

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		oversizedLog, err := json.Marshal(submitLogsRequest{
			NodeKey: *host.NodeKey,
			LogType: "status",
			Data:    []json.RawMessage{json.RawMessage(`"` + strings.Repeat("x", 12*1024*1024) + `"`)},
		})
		s.Require().NoError(err)
		ts.DoRawWithHeaders("POST", "/api/osquery/log", oversizedLog,
			http.StatusOK,
			map[string]string{"Authorization": "NodeKey " + *host.NodeKey})
		ts.DoRawWithHeaders("POST", "/api/osquery/distributed/write", oversizedLog,
			http.StatusOK,
			map[string]string{"Authorization": "NodeKey " + *host.NodeKey})
	})
}
