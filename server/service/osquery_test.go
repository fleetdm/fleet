package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	fleetLogging "github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mock"
	mockresult "github.com/fleetdm/fleet/v4/server/mock/mockresult"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/fleetdm/fleet/v4/server/service/redis_policy_set"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClientConfig(t *testing.T) {
	ds := new(mock.Store)

	ds.TeamAgentOptionsFunc = func(ctx context.Context, teamID uint) (*json.RawMessage, error) {
		return nil, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, pid uint) (fleet.ScheduledQueryList, error) {
		tru := true
		fals := false
		fortytwo := uint(42)
		switch pid {
		case 1:
			return []*fleet.ScheduledQuery{
				{Name: "time", Query: "select * from time", Interval: 30, Removed: &fals},
			}, nil
		case 4:
			return []*fleet.ScheduledQuery{
				{Name: "foobar", Query: "select 3", Interval: 20, Shard: &fortytwo},
				{Name: "froobing", Query: "select 'guacamole'", Interval: 60, Snapshot: &tru},
			}, nil
		default:
			return []*fleet.ScheduledQuery{}, nil
		}
	}
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		if teamID == nil {
			return nil, nil
		}
		return []*fleet.Query{
			{
				Query:             "SELECT 1 FROM table_1",
				Name:              "Some strings carry more weight than others",
				Interval:          10,
				Platform:          "linux",
				MinOsqueryVersion: "5.12.2",
				Logging:           "snapshot",
				TeamID:            ptr.Uint(1),
			},
			{
				Query:    "SELECT 1 FROM table_2",
				Name:     "You shall not pass",
				Interval: 20,
				Platform: "macos",
				Logging:  "differential",
				TeamID:   ptr.Uint(1),
			},
		}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":{"options":{"baz":"bar"}}}`))}, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id != 1 && id != 2 {
			return nil, errors.New("not found")
		}
		return &fleet.Host{ID: id}, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	ctx1 := hostctx.NewContext(ctx, &fleet.Host{ID: 1})
	ctx2 := hostctx.NewContext(ctx, &fleet.Host{ID: 2})
	ctx3 := hostctx.NewContext(ctx, &fleet.Host{ID: 1, TeamID: ptr.Uint(1)})

	expectedOptions := map[string]interface{}{
		"baz": "bar",
	}

	expectedConfig := map[string]interface{}{
		"options": expectedOptions,
	}

	// No packs loaded yet
	conf, err := svc.GetClientConfig(ctx1)
	require.NoError(t, err)
	assert.Equal(t, expectedConfig, conf)

	conf, err = svc.GetClientConfig(ctx2)
	require.NoError(t, err)
	assert.Equal(t, expectedConfig, conf)

	// Now add packs
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		switch hid {
		case 1:
			return []*fleet.Pack{
				{ID: 1, Name: "pack_by_label"},
				{ID: 4, Name: "pack_by_other_label"},
			}, nil

		case 2:
			return []*fleet.Pack{
				{ID: 1, Name: "pack_by_label"},
			}, nil
		}
		return []*fleet.Pack{}, nil
	}

	conf, err = svc.GetClientConfig(ctx1)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, conf["options"])
	assert.JSONEq(t, `{
		"pack_by_other_label": {
			"queries": {
				"foobar":{"query":"select 3","interval":20,"shard":42},
				"froobing":{"query":"select 'guacamole'","interval":60,"snapshot":true}
			}
		},
		"pack_by_label": {
			"queries":{
				"time":{"query":"select * from time","interval":30,"removed":false}
			}
		}
	}`,
		string(conf["packs"].(json.RawMessage)),
	)

	conf, err = svc.GetClientConfig(ctx2)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, conf["options"])
	assert.JSONEq(t, `{
		"pack_by_label": {
			"queries":{
				"time":{"query":"select * from time","interval":30,"removed":false}
			}
		}
	}`,
		string(conf["packs"].(json.RawMessage)),
	)

	// Check scheduled queries are loaded properly
	conf, err = svc.GetClientConfig(ctx3)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"pack_by_label": {
			"queries":{
				"time":{"query":"select * from time","interval":30,"removed":false}
			}
		},
		"pack_by_other_label": {
			"queries": {
				"foobar":{"query":"select 3","interval":20,"shard":42},
				"froobing":{"query":"select 'guacamole'","interval":60,"snapshot":true}
			}
		},
		"team-1": {
			"queries": {
				"Some strings carry more weight than others": {
					"query": "SELECT 1 FROM table_1",
					"interval": 10,
					"platform": "linux",
					"version": "5.12.2",
					"snapshot": true
				},
				"You shall not pass": {
					"query": "SELECT 1 FROM table_2",
					"interval": 20,
					"platform": "macos",
					"removed": true,
					"version": ""
				}
			}
		}
	}`,
		string(conf["packs"].(json.RawMessage)),
	)
}

func TestAgentOptionsForHost(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	teamID := uint(1)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":{"baz":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override2"}}}}`)),
		}, nil
	}
	ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
		return ptr.RawMessage(json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)), nil
	}

	host := &fleet.Host{
		TeamID:   &teamID,
		Platform: "darwin",
	}

	opt, err := svc.AgentOptionsForHost(ctx, host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override"}`, string(opt))

	host.Platform = "windows"
	opt, err = svc.AgentOptionsForHost(ctx, host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"bar"}`, string(opt))

	// Should take gobal option with no team
	host.TeamID = nil
	opt, err = svc.AgentOptionsForHost(ctx, host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"baz":"bar"}`, string(opt))

	host.Platform = "darwin"
	opt, err = svc.AgentOptionsForHost(ctx, host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override2"}`, string(opt))
}

var allDetailQueries = osquery_utils.GetDetailQueries(
	context.Background(),
	config.FleetConfig{Vulnerabilities: config.VulnerabilitiesConfig{DisableWinOSVulnerabilities: true}},
	nil,
	&fleet.Features{
		EnableHostUsers:         true,
		EnableSoftwareInventory: true,
	},
)

func expectedDetailQueriesForPlatform(platform string) map[string]osquery_utils.DetailQuery {
	queries := make(map[string]osquery_utils.DetailQuery)
	for k, v := range allDetailQueries {
		if v.RunsForPlatform(platform) {
			queries[k] = v
		}
	}
	return queries
}

func TestEnrollAgent(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		switch secret {
		case "valid_secret":
			return &fleet.EnrollSecret{Secret: "valid_secret", TeamID: ptr.Uint(3)}, nil
		default:
			return nil, errors.New("not found")
		}
	}
	ds.EnrollHostFunc = func(ctx context.Context, isMDMEnabled bool, osqueryHostId, hUUID, hSerial, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
		assert.Equal(t, ptr.Uint(3), teamID)
		return &fleet.Host{
			OsqueryHostID: &osqueryHostId, NodeKey: &nodeKey,
		}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	nodeKey, err := svc.EnrollAgent(ctx, "valid_secret", "host123", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, nodeKey)
}

func TestEnrollAgentEnforceLimit(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		const maxHosts = 2

		var hostIDSeq uint
		ds := new(mock.Store)
		ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			switch secret {
			case "valid_secret":
				return &fleet.EnrollSecret{Secret: "valid_secret"}, nil
			default:
				return nil, errors.New("not found")
			}
		}
		ds.EnrollHostFunc = func(ctx context.Context, isMDMEnabled bool, osqueryHostId, hUUID, hSerial, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
			hostIDSeq++
			return &fleet.Host{
				ID: hostIDSeq, OsqueryHostID: &osqueryHostId, NodeKey: &nodeKey,
			}, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return &fleet.Host{ID: id}, nil
		}
		ds.DeleteHostFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		redisWrapDS := mysqlredis.New(ds, pool, mysqlredis.WithEnforcedHostLimit(maxHosts))
		svc, ctx := newTestService(t, redisWrapDS, nil, nil, &TestServerOpts{
			EnrollHostLimiter: redisWrapDS,
			License:           &fleet.LicenseInfo{DeviceCount: maxHosts},
		})
		ctx = viewer.NewContext(ctx, viewer.Viewer{
			User: &fleet.User{
				ID:         0,
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
		})

		nodeKey, err := svc.EnrollAgent(ctx, "valid_secret", "host001", nil)
		require.NoError(t, err)
		assert.NotEmpty(t, nodeKey)

		nodeKey, err = svc.EnrollAgent(ctx, "valid_secret", "host002", nil)
		require.NoError(t, err)
		assert.NotEmpty(t, nodeKey)

		_, err = svc.EnrollAgent(ctx, "valid_secret", "host003", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("maximum number of hosts reached: %d", maxHosts))

		// delete a host with id 1
		err = svc.DeleteHost(ctx, 1)
		require.NoError(t, err)

		// now host 003 can be enrolled
		nodeKey, err = svc.EnrollAgent(ctx, "valid_secret", "host003", nil)
		require.NoError(t, err)
		assert.NotEmpty(t, nodeKey)
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, "enrolled_hosts:*", false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, "enrolled_hosts:*", true, true, false)
		runTest(t, pool)
	})
}

func TestEnrollAgentIncorrectEnrollSecret(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		switch secret {
		case "valid_secret":
			return &fleet.EnrollSecret{Secret: "valid_secret", TeamID: ptr.Uint(3)}, nil
		default:
			return nil, errors.New("not found")
		}
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	nodeKey, err := svc.EnrollAgent(ctx, "not_correct", "host123", nil)
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)
}

func TestEnrollAgentDetails(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{}, nil
	}
	ds.EnrollHostFunc = func(ctx context.Context, isMDMEnabled bool, osqueryHostId, hUUID, hSerial, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
		return &fleet.Host{
			OsqueryHostID: &osqueryHostId, NodeKey: &nodeKey,
		}, nil
	}
	var gotHost *fleet.Host
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		gotHost = host
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	details := map[string](map[string]string){
		"osquery_info": {"version": "2.12.0"},
		"system_info":  {"hostname": "zwass.local", "uuid": "froobling_uuid"},
		"os_version": {
			"name":     "Mac OS X",
			"major":    "10",
			"minor":    "14",
			"patch":    "5",
			"platform": "darwin",
		},
		"foo": {"foo": "bar"},
	}
	nodeKey, err := svc.EnrollAgent(ctx, "", "host123", details)
	require.NoError(t, err)
	assert.NotEmpty(t, nodeKey)

	assert.Equal(t, "Mac OS X 10.14.5", gotHost.OSVersion)
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "2.12.0", gotHost.OsqueryVersion)
	assert.Equal(t, "zwass.local", gotHost.Hostname)
	assert.Equal(t, "froobling_uuid", gotHost.UUID)
}

func TestAuthenticateHost(t *testing.T) {
	ds := new(mock.Store)
	task := async.NewTask(ds, nil, clock.C, config.OsqueryConfig{})
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{Task: task})

	var gotKey string
	host := fleet.Host{ID: 1, Hostname: "foobar"}
	ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		gotKey = nodeKey
		return &host, nil
	}
	var gotHostIDs []uint
	ds.MarkHostsSeenFunc = func(ctx context.Context, hostIDs []uint, t time.Time) error {
		gotHostIDs = hostIDs
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	_, _, err := svc.AuthenticateHost(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "test", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)

	host = fleet.Host{ID: 7, Hostname: "foobar"}
	_, _, err = svc.AuthenticateHost(ctx, "floobar")
	require.NoError(t, err)
	assert.Equal(t, "floobar", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)
	// Host checks in twice
	host = fleet.Host{ID: 7, Hostname: "foobar"}
	_, _, err = svc.AuthenticateHost(ctx, "floobar")
	require.NoError(t, err)
	assert.Equal(t, "floobar", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)

	err = task.FlushHostsLastSeen(ctx, time.Now())
	require.NoError(t, err)
	assert.True(t, ds.MarkHostsSeenFuncInvoked)
	ds.MarkHostsSeenFuncInvoked = false
	assert.ElementsMatch(t, []uint{1, 7}, gotHostIDs)

	err = task.FlushHostsLastSeen(ctx, time.Now())
	require.NoError(t, err)
	assert.True(t, ds.MarkHostsSeenFuncInvoked)
	require.Len(t, gotHostIDs, 0)
}

func TestAuthenticateHostFailure(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, errors.New("not found")
	}

	_, _, err := svc.AuthenticateHost(ctx, "test")
	require.NotNil(t, err)
}

type testJSONLogger struct {
	logs []json.RawMessage
}

func (n *testJSONLogger) Write(ctx context.Context, logs []json.RawMessage) error {
	n.logs = logs
	return nil
}

func TestSubmitStatusLogs(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*Service)

	testLogger := &testJSONLogger{}
	serv.osqueryLogWriter = &OsqueryLogger{Status: testLogger}

	logs := []string{
		`{"severity":"0","filename":"tls.cpp","line":"216","message":"some message","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
		`{"severity":"1","filename":"buffered.cpp","line":"122","message":"warning!","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var status []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &status)
	require.NoError(t, err)

	host := fleet.Host{}
	ctx = hostctx.NewContext(ctx, &host)
	err = serv.SubmitStatusLogs(ctx, status)
	require.NoError(t, err)

	assert.Equal(t, status, testLogger.logs)
}

func TestSubmitResultLogsToLogDestination(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{Logger: log.NewJSONLogger(os.Stdout)})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		switch {
		case teamID != nil && *teamID == 1:
			return &fleet.Query{
				ID:                 4242,
				Name:               name,
				AutomationsEnabled: true,
				TeamID:             ptr.Uint(1),
				Logging:            fleet.LoggingSnapshot,
			}, nil
		case teamID != nil && *teamID == 2:
			return &fleet.Query{
				ID:                 4343,
				Name:               name,
				AutomationsEnabled: true,
				TeamID:             ptr.Uint(2),
				Logging:            fleet.LoggingSnapshot,
			}, nil
		case teamID == nil && (name == "time" || name == "system_info" || name == "encrypted" || name == "hosts"):
			return &fleet.Query{
				Name:               name,
				AutomationsEnabled: true,
			}, nil
		case teamID != nil && *teamID == 1 && name == "hosts":
			return &fleet.Query{
				Name:               name,
				AutomationsEnabled: true,
				TeamID:             teamID,
			}, nil
		case teamID == nil && name == "query_not_automated":
			return &fleet.Query{
				Name:               name,
				AutomationsEnabled: false,
			}, nil
		case teamID == nil && name == "query_should_be_saved_and_submitted":
			return &fleet.Query{
				ID:                 123,
				Name:               name,
				AutomationsEnabled: true,
				Logging:            fleet.LoggingSnapshot,
			}, nil
		case teamID == nil && name == "query_should_be_saved_and_submitted_with_custom_pack_delimiter":
			return &fleet.Query{
				ID:                 1234,
				Name:               name,
				AutomationsEnabled: true,
				Logging:            fleet.LoggingSnapshot,
			}, nil
		case teamID == nil && name == "query_should_be_saved_but_not_submitted":
			return &fleet.Query{
				ID:                 444,
				Name:               name,
				AutomationsEnabled: false,
				Logging:            fleet.LoggingSnapshot,
			}, nil
		case teamID == nil && name == "query_no_rows":
			return &fleet.Query{
				ID:                 555,
				Name:               name,
				AutomationsEnabled: true,
				Logging:            fleet.LoggingSnapshot,
			}, nil
		default:
			return nil, newNotFoundError()
		}
	}
	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}
	teamQueryResultsStored := false
	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) error {
		if len(rows) == 0 {
			return nil
		}
		switch {
		case rows[0].QueryID == 4242:
			t.Fatal("should not happen, as query 4242 is a team query and host is global")
		case rows[0].QueryID == 4343:
			teamQueryResultsStored = true
		case rows[0].QueryID == 123:
			require.Len(t, rows, 1)
			require.Equal(t, uint(999), rows[0].HostID)
			require.NotZero(t, rows[0].LastFetched)
			require.JSONEq(t, `{"hour":"20","minutes":"8"}`, string(*rows[0].Data))
		case rows[0].QueryID == 444:
			require.Len(t, rows, 2)
			require.Equal(t, uint(999), rows[0].HostID)
			require.NotZero(t, rows[0].LastFetched)
			require.JSONEq(t, `{"hour":"20","minutes":"8"}`, string(*rows[0].Data))
			require.Equal(t, uint(999), rows[1].HostID)
			require.Equal(t, uint(444), rows[1].QueryID)
			require.NotZero(t, rows[1].LastFetched)
			require.JSONEq(t, `{"hour":"21","minutes":"9"}`, string(*rows[1].Data))
		}
		return nil
	}

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*Service)

	testLogger := &testJSONLogger{}
	serv.osqueryLogWriter = &OsqueryLogger{Result: testLogger}

	validLogResults := []string{
		`{"name":"pack/Global/system_info","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 17:55:15 2016 UTC","unixTime":1475258115,"decorations":{"host_uuid":"some_uuid","username":"zwass"},"columns":{"cpu_brand":"Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz","hostname":"hostimus","physical_memory":"17179869184"},"action":"added"}`,

		`{"name":"pack/SomePack/encrypted","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 21:19:15 2016 UTC","unixTime":1475270355,"decorations":{"host_uuid":"4740D59F-699E-5B29-960B-979AAF9BBEEB","username":"zwass"},"columns":{"encrypted":"1","name":"\/dev\/disk1","type":"AES-XTS","uid":"","user_uuid":"","uuid":"some_uuid"},"action":"added"}`,
		`{"name":"pack/SomePack/encrypted","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 21:19:14 2016 UTC","unixTime":1475270354,"decorations":{"host_uuid":"4740D59F-699E-5B29-960B-979AAF9BBEEB","username":"zwass"},"columns":{"encrypted":"1","name":"\/dev\/disk1","type":"AES-XTS","uid":"","user_uuid":"","uuid":"some_uuid"},"action":"added"}`,

		// These results belong to the same query but have 1 second difference.
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Global/time","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Global/time","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:50 2017 UTC","unixTime":1484078930,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Global/time","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:52 2017 UTC","unixTime":1484078932,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,

		`{"diffResults":{"removed":[{"address":"127.0.0.1","hostnames":"kl.groob.io"}],"added":""},"name":"pack\/team-1/hosts","hostIdentifier":"FA01680E-98CA-5557-8F59-7716ECFEE964","calendarTime":"Sun Nov 19 00:02:08 2017 UTC","unixTime":1511049728,"epoch":"0","counter":"10","decorations":{"host_uuid":"FA01680E-98CA-5557-8F59-7716ECFEE964","hostname":"kl.groob.io"}}`,

		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Global/query_should_be_saved_and_submitted","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack_Global_query_should_be_saved_and_submitted_with_custom_pack_delimiter","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:52 2017 UTC","unixTime":1484078932,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,

		// Fleet doesn't know of this query, so this result should be streamed as is (This is to support streaming results for osquery nodes that are configured outside of Fleet, e.g. `--config_plugin=filesystem`).
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Global/doesntexist","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,

		// If a global query belongs to a 2017/legacy pack, it should be automated even if the global query has automations turned off.
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Some Pack Name/query_not_automated","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,

		// The "name" field has invalid format, so this result will be streamed as is (This is to support streaming results for osquery nodes that are configured outside of Fleet, e.g. `--config_plugin=filesystem`).
		`{"name":"com.foo.bar","hostIdentifier":"52eb420a-2085-438a-abf0-5670e97588e2","calendarTime":"Thu Dec  7 15:15:20 2023 UTC","unixTime":1701962120,"epoch":0,"counter":0,"numerics":false,"columns":{"foo": "bar"},"action":"snapshot"}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"some_name","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/team-foo/bar","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/team-","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/PackName","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,

		// Query results of a query that belongs to a different team than the host's team (can happen when host is transferred from one team to another or no team).
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/team-1/Foobar","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(validLogResults, ","))

	resultWithInvalidJSON := []byte("foobar:\n\t123")
	resultWithInvalidJSONLong := []byte("foobar:\n\t1233333333333333333333333333333333333333333333333333333333")
	// The "name" field will be empty, so this result will be ignored.
	resultWithoutName := []byte(`{"unknown":{"foo": [] }}`)
	// The query was configured with automations disabled, so this result will be ignored.
	resultWithQueryNotAutomated := []byte(`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/Global/query_not_automated","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`)
	// The query is supposed to be saved but with automations disabled (and has two columns).
	resultWithQuerySavedNotAutomated := []byte(`{"snapshot":[{"hour":"20","minutes":"8"},{"hour":"21","minutes":"9"}],"action":"snapshot","name":"pack/Global/query_should_be_saved_but_not_submitted","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`)

	var validResults []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &validResults)
	require.NoError(t, err)

	host := fleet.Host{
		ID:     999,
		TeamID: nil, // Global host.
	}
	ctx = hostctx.NewContext(ctx, &host)

	// Submit valid, invalid and to-be-ignored log results mixed.
	validAndInvalidResults := make([]json.RawMessage, 0, len(validResults)+5)
	for i, result := range validResults {
		validAndInvalidResults = append(validAndInvalidResults, result)
		if i == 2 {
			validAndInvalidResults = append(validAndInvalidResults,
				resultWithInvalidJSON, resultWithInvalidJSONLong,
				resultWithoutName, resultWithQueryNotAutomated,
				resultWithQuerySavedNotAutomated,
			)
		}
	}
	err = serv.SubmitResultLogs(ctx, validAndInvalidResults)
	require.NoError(t, err)

	assert.Equal(t, validResults, testLogger.logs)

	//
	// Run a similar test but now with a team host.
	//
	host = fleet.Host{
		ID:     999,
		TeamID: ptr.Uint(2),
	}
	ctx = hostctx.NewContext(ctx, &host)
	results := []json.RawMessage{
		// This query should be ignored.
		json.RawMessage(`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/team-1/Foobar","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`),
		// This query should be stored.
		json.RawMessage(`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"pack/team-2/Zoobar","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`),
	}
	err = serv.SubmitResultLogs(ctx, results)
	require.NoError(t, err)

	require.True(t, teamQueryResultsStored)
}

func TestSaveResultLogsToQueryReports(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// Hack to get at the private methods
	serv := ((svc.(validationMiddleware)).Service).(*Service)

	host := fleet.Host{}
	ctx = hostctx.NewContext(ctx, &host)

	results := []*fleet.ScheduledQueryResult{
		{
			QueryName:     "pack/Global/Uptime",
			OsqueryHostID: "1379f59d98f4",
			Snapshot: []*json.RawMessage{
				ptr.RawMessage(json.RawMessage(`{"hour":"20","minutes":"8"}`)),
			},
			UnixTime: 1484078931,
		},
	}

	// Results not saved if DiscardData is true in Query
	discardDataFalse := map[string]*fleet.Query{
		"pack/Global/Uptime": {
			ID:          1,
			DiscardData: true,
			Logging:     fleet.LoggingSnapshot,
		},
	}
	serv.saveResultLogsToQueryReports(ctx, results, discardDataFalse, fleet.DefaultMaxQueryReportRows)
	assert.False(t, ds.OverwriteQueryResultRowsFuncInvoked)

	// Happy Path: Results saved
	discardDataTrue := map[string]*fleet.Query{
		"pack/Global/Uptime": {
			ID:          1,
			DiscardData: false,
			Logging:     fleet.LoggingSnapshot,
		},
	}
	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) error {
		return nil
	}
	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}
	serv.saveResultLogsToQueryReports(ctx, results, discardDataTrue, fleet.DefaultMaxQueryReportRows)
	require.True(t, ds.OverwriteQueryResultRowsFuncInvoked)
}

func TestSubmitResultLogsToQueryResultsWithEmptySnapShot(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	host := fleet.Host{
		ID: 999,
	}
	ctx = hostctx.NewContext(ctx, &host)

	logs := []string{
		`{"snapshot":[],"action":"snapshot","name":"pack/Global/query_no_rows","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
	}

	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))
	var results []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &results)
	require.NoError(t, err)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				QueryReportsDisabled: false,
			},
		}, nil
	}

	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return &fleet.Query{
			ID:          1,
			DiscardData: false,
			Logging:     fleet.LoggingSnapshot,
		}, nil
	}

	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}

	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) error {
		require.Len(t, rows, 1)
		require.Equal(t, uint(999), rows[0].HostID)
		require.NotZero(t, rows[0].LastFetched)
		require.Nil(t, rows[0].Data)
		return nil
	}

	err = svc.SubmitResultLogs(ctx, results)
	require.NoError(t, err)
	assert.True(t, ds.OverwriteQueryResultRowsFuncInvoked)
}

func TestSubmitResultLogsToQueryResultsDoesNotCountNullDataRows(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	host := fleet.Host{
		ID: 999,
	}
	ctx = hostctx.NewContext(ctx, &host)

	logs := []string{
		`{"snapshot":[],"action":"snapshot","name":"pack/Global/query_no_rows","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":1484078931,"decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
	}

	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))
	var results []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &results)
	require.NoError(t, err)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				QueryReportsDisabled: false,
			},
		}, nil
	}

	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return &fleet.Query{
			ID:          1,
			DiscardData: false,
			Logging:     fleet.LoggingSnapshot,
		}, nil
	}

	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}

	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) error {
		require.Len(t, rows, 1)
		require.Equal(t, uint(999), rows[0].HostID)
		require.NotZero(t, rows[0].LastFetched)
		require.Nil(t, rows[0].Data)
		return nil
	}

	err = svc.SubmitResultLogs(ctx, results)
	require.NoError(t, err)
	assert.True(t, ds.OverwriteQueryResultRowsFuncInvoked)
}

type failingLogger struct{}

func (n *failingLogger) Write(context.Context, []json.RawMessage) error {
	return errors.New("some error")
}

func TestSubmitResultLogsFail(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	host := fleet.Host{
		ID: 999,
	}
	ctx = hostctx.NewContext(ctx, &host)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*Service)

	testLogger := &failingLogger{}
	serv.osqueryLogWriter = &OsqueryLogger{Result: testLogger}

	logs := []string{
		`{"name":"pack/Global/system_info","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 17:55:15 2016 UTC","unixTime":1475258115,"decorations":{"host_uuid":"some_uuid","username":"zwass"},"columns":{"cpu_brand":"Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz","hostname":"hostimus","physical_memory":"17179869184"},"action":"added"}`,
	}

	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))
	var results []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &results)
	require.NoError(t, err)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return &fleet.Query{
			ID:                 1,
			DiscardData:        false,
			AutomationsEnabled: true,
			Name:               name,
		}, nil
	}
	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}
	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) error {
		return nil
	}

	// Expect an error when unable to write to logging destination.
	err = svc.SubmitResultLogs(ctx, results)
	require.Error(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, err.(*osqueryError).Status())
}

func TestGetQueryNameAndTeamIDFromResult(t *testing.T) {
	tests := []struct {
		input        string
		expectedID   *uint
		expectedName string
		hasErr       bool
	}{
		{"pack/Global/Query Name", nil, "Query Name", false},                       // valid global query
		{"pack/team-1/Query Name", ptr.Uint(1), "Query Name", false},               // valid team query
		{"pack/team-12345/Another Query", ptr.Uint(12345), "Another Query", false}, // valid team query
		{"pack/team-foo/Query", nil, "", true},                                     // missing team ID
		{"pack/Global/QueryWith/Slash", nil, "QueryWith/Slash", false},             // query name contains forward slash
		{"packGlobalGlobalGlobalGlobal", nil, "Global", false},                     // pack_delimiter=Global
		{"packXGlobalGlobalXGlobalQueryWith/Slash", nil, "QueryWith/Slash", false}, // pack_delimiter=XGlobal
		{"pack//Global//QueryWith/Slash", nil, "QueryWith/Slash", false},           // pack_delimiter=//
		{"pack/team-1/QueryWith/Slash", ptr.Uint(1), "QueryWith/Slash", false},
		{"pack_team-1_QueryWith/Slash", ptr.Uint(1), "QueryWith/Slash", false},
		{"packFOOBARteam-1FOOBARQueryWith/Slash", ptr.Uint(1), "QueryWith/Slash", false},   // pack_delimiter=FOOBAR
		{"pack123😁123team-1123😁123QueryWith/Slash", ptr.Uint(1), "QueryWith/Slash", false}, // pack_delimiter=123😁123
		{"pack(foo)team-1(foo)fo(o)bar", ptr.Uint(1), "fo(o)bar", false},                   // pack_delimiter=(foo)
		{"packteam-1team-1team-1team-1", ptr.Uint(1), "team-1", false},                     // pack_delimiter=team-1
		{"pack/Global/GlobalInQueryName", nil, "GlobalInQueryName", false},                 // query name contains Global
		{"pack/team-1/team-1InQueryName", ptr.Uint(1), "team-1InQueryName", false},         // query name contains team-1

		{"InvalidString", nil, "", true},
		{"Invalid/Query", nil, "", true},
		{"pac", nil, "", true},
		{"pack", nil, "", true},
		{"pack/", nil, "", true},
		{"pack/Global", nil, "", true},
		{"pack/Global/", nil, "", true},
		{"pack/team/foo", nil, "", true},
		{"pack/team-123", nil, "", true},
		{"pack/team-/foo", nil, "", true},
		{"pack/team-123/", nil, "", true},

		// Legacy 2017 packs should fail the parsing as they are separate
		// from global or team queries.
		{"pack/PackName/Query", nil, "", true},
		{"pack/PackName/QueryWith/Slash", nil, "", true},
		{"packFOOBARPackNameFOOBARQueryWith/Slash", nil, "", true}, // pack_delimiter=FOOBAR
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			id, str, err := getQueryNameAndTeamIDFromResult(tt.input)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedName, str)
			if tt.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMostRecentResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []*fleet.ScheduledQueryResult
		expected []*fleet.ScheduledQueryResult
	}{
		{
			name: "basic test",
			input: []*fleet.ScheduledQueryResult{
				{QueryName: "test1", UnixTime: 1},
				{QueryName: "test1", UnixTime: 2},
				{QueryName: "test1", UnixTime: 3},
				{QueryName: "test2", UnixTime: 1},
				{QueryName: "test2", UnixTime: 2},
				{QueryName: "test2", UnixTime: 3},
			},
			expected: []*fleet.ScheduledQueryResult{
				{QueryName: "test1", UnixTime: 3},
				{QueryName: "test2", UnixTime: 3},
			},
		},
		{
			name: "out of order test",
			input: []*fleet.ScheduledQueryResult{
				{QueryName: "test1", UnixTime: 2},
				{QueryName: "test1", UnixTime: 3},
				{QueryName: "test1", UnixTime: 1},
				{QueryName: "test2", UnixTime: 3},
				{QueryName: "test2", UnixTime: 2},
				{QueryName: "test2", UnixTime: 1},
			},
			expected: []*fleet.ScheduledQueryResult{
				{QueryName: "test1", UnixTime: 3},
				{QueryName: "test2", UnixTime: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := getMostRecentResults(tt.input)
			assert.ElementsMatch(t, tt.expected, results)
		})
	}
}

func verifyDiscovery(t *testing.T, queries, discovery map[string]string) {
	t.Helper()
	assert.Equal(t, len(queries), len(discovery))
	// discoveryUsed holds the queries where we know use the distributed discovery feature.
	discoveryUsed := map[string]struct{}{
		hostDetailQueryPrefix + "google_chrome_profiles":     {},
		hostDetailQueryPrefix + "mdm":                        {},
		hostDetailQueryPrefix + "munki_info":                 {},
		hostDetailQueryPrefix + "windows_update_history":     {},
		hostDetailQueryPrefix + "kubequery_info":             {},
		hostDetailQueryPrefix + "orbit_info":                 {},
		hostDetailQueryPrefix + "software_vscode_extensions": {},
		hostDetailQueryPrefix + "software_macos_firefox":     {},
		hostDetailQueryPrefix + "battery":                    {},
		hostDetailQueryPrefix + "software_macos_codesign":    {},
	}
	for name := range queries {
		require.NotEmpty(t, discovery[name])
		if _, ok := discoveryUsed[name]; ok {
			require.NotEqual(t, alwaysTrueQuery, discovery[name])
		} else {
			require.Equal(t, alwaysTrueQuery, discovery[name])
		}
	}
}

func TestHostDetailQueries(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	additional := json.RawMessage(`{"foobar": "select foo", "bim": "bam"}`)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			AdditionalQueries:       &additional,
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}

	mockClock := clock.NewMockClock()
	host := fleet.Host{
		ID: 1,
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			UpdateTimestamp: fleet.UpdateTimestamp{
				UpdatedAt: mockClock.Now(),
			},
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: mockClock.Now(),
			},
		},

		Platform:        "darwin",
		DetailUpdatedAt: mockClock.Now(),
		NodeKey:         ptr.String("test_key"),
		Hostname:        "test_hostname",
		UUID:            "test_uuid",
	}

	svc := &Service{
		clock:    mockClock,
		logger:   log.NewNopLogger(),
		config:   config.TestConfig(),
		ds:       ds,
		jitterMu: new(sync.Mutex),
		jitterH:  make(map[time.Duration]*jitterHashTable),
	}

	// detail_updated_at is now, so nothing gets returned by default
	queries, discovery, err := svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	assert.Empty(t, queries)
	verifyDiscovery(t, queries, discovery)

	// With refetch requested detail queries should be returned
	host.RefetchRequested = true
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	// +2: additional queries: bim, foobar
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+2, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	host.RefetchRequested = false

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	// all queries returned now that detail udpated at is in the past
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	// +2: additional queries: bim, foobar
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+2, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	for name := range queries {
		assert.True(t,
			strings.HasPrefix(name, hostDetailQueryPrefix) || strings.HasPrefix(name, hostAdditionalQueryPrefix),
		)
	}
	assert.Equal(t, "bam", queries[hostAdditionalQueryPrefix+"bim"])
	assert.Equal(t, "select foo", queries[hostAdditionalQueryPrefix+"foobar"])

	host.DetailUpdatedAt = mockClock.Now()

	// detail_updated_at is now, so nothing gets returned
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	assert.Empty(t, queries)
	verifyDiscovery(t, queries, discovery)

	// setting refetch_critical_queries_until in the past still returns nothing
	host.RefetchCriticalQueriesUntil = ptr.Time(mockClock.Now().Add(-1 * time.Minute))
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	assert.Empty(t, queries)
	verifyDiscovery(t, queries, discovery)

	// setting refetch_critical_queries_until in the future returns only the critical queries
	host.RefetchCriticalQueriesUntil = ptr.Time(mockClock.Now().Add(1 * time.Minute))
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	// host is darwin so it gets only the darwin critical query
	require.Equal(t, 1, len(queries), distQueriesMapKeys(queries))
	for name := range criticalDetailQueries {
		if strings.HasSuffix(name, "_windows") {
			continue
		}
		assert.Contains(t, queries, hostDetailQueryPrefix+name)
	}
	verifyDiscovery(t, queries, discovery)
}

func TestQueriesAndHostFeatures(t *testing.T) {
	ds := new(mock.Store)
	team1 := fleet.Team{
		ID: 1,
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         true,
				EnableSoftwareInventory: false,
			},
		},
	}

	team2 := fleet.Team{
		ID: 2,
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: true,
			},
		},
	}

	host := fleet.Host{
		ID:       1,
		Platform: "darwin",
		NodeKey:  ptr.String("test_key"),
		Hostname: "test_hostname",
		UUID:     "test_uuid",
		TeamID:   nil,
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: false,
			},
		}, nil
	}

	ds.TeamFeaturesFunc = func(ctx context.Context, id uint) (*fleet.Features, error) {
		switch id {
		case uint(1):
			return &team1.Config.Features, nil
		case uint(2):
			return &team2.Config.Features, nil
		default:
			return nil, errors.New("team not found")
		}
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	lq := live_query_mock.New(t)
	lq.On("QueriesForHost", uint(1)).Return(map[string]string{}, nil)
	lq.On("QueriesForHost", uint(2)).Return(map[string]string{}, nil)
	lq.On("QueriesForHost", nil).Return(map[string]string{}, nil)

	t.Run("free license", func(t *testing.T) {
		license := &fleet.LicenseInfo{Tier: fleet.TierFree}
		svc, ctx := newTestService(t, ds, nil, lq, &TestServerOpts{License: license})

		ctx = hostctx.NewContext(ctx, &host)
		queries, _, _, err := svc.GetDistributedQueries(ctx)
		require.NoError(t, err)
		require.NotContains(t, queries, "fleet_detail_query_users")
		require.NotContains(t, queries, "fleet_detail_query_software_macos")
		require.NotContains(t, queries, "fleet_detail_query_software_linux")
		require.NotContains(t, queries, "fleet_detail_query_software_windows")

		// assign team 1 to host
		host.TeamID = &team1.ID
		queries, _, _, err = svc.GetDistributedQueries(ctx)
		require.NoError(t, err)
		require.NotContains(t, queries, "fleet_detail_query_users")
		require.NotContains(t, queries, "fleet_detail_query_software_macos")
		require.NotContains(t, queries, "fleet_detail_query_software_linux")
		require.NotContains(t, queries, "fleet_detail_query_software_windows")

		// assign team 2 to host
		host.TeamID = &team2.ID
		queries, _, _, err = svc.GetDistributedQueries(ctx)
		require.NoError(t, err)
		require.NotContains(t, queries, "fleet_detail_query_users")
		require.NotContains(t, queries, "fleet_detail_query_software_macos")
		require.NotContains(t, queries, "fleet_detail_query_software_linux")
		require.NotContains(t, queries, "fleet_detail_query_software_windows")
	})

	t.Run("premium license", func(t *testing.T) {
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, lq, &TestServerOpts{License: license})

		host.TeamID = nil
		ctx = hostctx.NewContext(ctx, &host)
		queries, _, _, err := svc.GetDistributedQueries(ctx)
		require.NoError(t, err)
		require.NotContains(t, queries, "fleet_detail_query_users")
		require.NotContains(t, queries, "fleet_detail_query_software_macos")
		require.NotContains(t, queries, "fleet_detail_query_software_linux")
		require.NotContains(t, queries, "fleet_detail_query_software_windows")

		// assign team 1 to host
		host.TeamID = &team1.ID
		queries, _, _, err = svc.GetDistributedQueries(ctx)
		require.NoError(t, err)
		require.Contains(t, queries, "fleet_detail_query_users")
		require.NotContains(t, queries, "fleet_detail_query_software_macos")
		require.NotContains(t, queries, "fleet_detail_query_software_linux")
		require.NotContains(t, queries, "fleet_detail_query_software_windows")

		// assign team 2 to host
		host.TeamID = &team2.ID
		queries, _, _, err = svc.GetDistributedQueries(ctx)
		require.NoError(t, err)
		require.NotContains(t, queries, "fleet_detail_query_users")
		require.Contains(t, queries, "fleet_detail_query_software_macos")
	})
}

func TestGetDistributedQueriesMissingHost(t *testing.T) {
	svc, ctx := newTestService(t, &mock.Store{}, nil, nil)

	_, _, _, err := svc.GetDistributedQueries(ctx)
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestGetDistributedQueriesEmptyQuery(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, nil, lq, mockClock)

	host := &fleet.Host{
		Platform: "darwin",
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{"empty_label_query": ""}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, gotHost *fleet.Host) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{EnableHostUsers: true}}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{"empty_policy_query": ""}, nil
	}

	lq.On("QueriesForHost", uint(0)).Return(map[string]string{"empty_live_query": ""}, nil)

	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, _, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, queries)
	for n, q := range queries {
		require.NotEmpty(t, q, n)
	}
	for n, q := range discovery {
		require.NotEmpty(t, q, n)
	}
}

func TestLabelQueries(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, nil, lq, mockClock)

	host := &fleet.Host{
		Platform: "darwin",
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, gotHost *fleet.Host) error {
		host = gotHost
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	lq.On("QueriesForHost", uint(0)).Return(map[string]string{}, nil)

	ctx = hostctx.NewContext(ctx, host)

	// With a new host, we should get the detail queries (and accelerate
	// should be turned on so that we can quickly fill labels)
	queries, discovery, acc, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +1 for the fleet_no_policies_wildcard query.
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+1, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	assert.NotZero(t, acc)

	// Simulate the detail queries being added.
	host.DetailUpdatedAt = mockClock.Now().Add(-1 * time.Minute)
	host.Hostname = "zwass.local"
	ctx = hostctx.NewContext(ctx, host)

	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 1) // fleet_no_policies_wildcard query
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{
			"label1": "query1",
			"label2": "query2",
			"label3": "query3",
		}, nil
	}

	// Now we should get the label queries
	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +1 for the fleet_no_policies_wildcard query.
	require.Len(t, queries, 3+1)
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)

	var gotHost *fleet.Host
	var gotResults map[uint]*bool
	var gotTime time.Time
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferred bool) error {
		gotHost = host
		gotResults = results
		gotTime = t
		return nil
	}

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "1": {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	host.LabelUpdatedAt = mockClock.Now()
	assert.Equal(t, host, gotHost)
	assert.Equal(t, mockClock.Now(), gotTime)
	require.Len(t, gotResults, 1)
	assert.Equal(t, true, *gotResults[1])

	mockClock.AddTime(1 * time.Second)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "2": {{"col1": "val1"}},
			hostLabelQueryPrefix + "3": {},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	host.LabelUpdatedAt = mockClock.Now()
	assert.Equal(t, host, gotHost)
	assert.Equal(t, mockClock.Now(), gotTime)
	require.Len(t, gotResults, 2)
	assert.Equal(t, true, *gotResults[2])
	assert.Equal(t, false, *gotResults[3])

	// We should get no labels now.
	host.LabelUpdatedAt = mockClock.Now()
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 1) // fleet_no_policies_wildcard query
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)

	// With refetch requested details+label queries should be returned.
	host.RefetchRequested = true
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +3 for label queries, +1 for the fleet_no_policies_wildcard query.
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+3+1, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "2": {{"col1": "val1"}},
			hostLabelQueryPrefix + "3": {},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	host.LabelUpdatedAt = mockClock.Now()
	assert.Equal(t, host, gotHost)
	assert.Equal(t, mockClock.Now(), gotTime)
	require.Len(t, gotResults, 2)
	assert.Equal(t, true, *gotResults[2])
	assert.Equal(t, false, *gotResults[3])

	// SubmitDistributedQueryResults will set RefetchRequested to false.
	require.False(t, host.RefetchRequested)

	// There shouldn't be any labels now.
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 1) // fleet_no_policies_wildcard query
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)
}

func TestDetailQueriesWithEmptyStrings(t *testing.T) {
	ds := new(mock.Store)
	mockClock := clock.NewMockClock()
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, nil, lq, mockClock)

	host := &fleet.Host{
		ID:       1,
		Platform: "windows",
	}
	ctx = hostctx.NewContext(ctx, host)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}
	ds.LabelQueriesForHostFunc = func(context.Context, *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id != 1 {
			return nil, errors.New("not found")
		}
		return host, nil
	}

	lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, discovery, acc, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +1 due to 'windows_update_history', +1 due to fleet_no_policies_wildcard query.
	if expected := expectedDetailQueriesForPlatform(host.Platform); !assert.Equal(t, len(expected)+1+1, len(queries)) {
		// this is just to print the diff between the expected and actual query
		// keys when the count assertion fails, to help debugging - they are not
		// expected to match.
		require.ElementsMatch(t, osqueryMapKeys(expected), distQueriesMapKeys(queries))
	}
	verifyDiscovery(t, queries, discovery)
	assert.NotZero(t, acc)

	resultJSON := `
{
  "fleet_detail_query_network_interface_windows": [
    {
      "address": "192.168.0.1",
      "mac": "5f:3d:4b:10:25:82"
    }
  ],
  "fleet_detail_query_os_version": [
    {
      "platform": "darwin",
      "build": "15G1004",
      "major": "10",
      "minor": "10",
      "name": "Mac OS X",
      "patch": "6"
    }
  ],
  "fleet_detail_query_osquery_info": [
    {
      "build_distro": "10.10",
      "build_platform": "darwin",
      "config_hash": "3c6e4537c4d0eb71a7c6dda19d",
      "config_valid": "1",
      "extensions": "active",
      "pid": "38113",
      "start_time": "1475603155",
      "version": "1.8.2",
      "watcher": "38112"
    }
  ],
  "fleet_detail_query_system_info": [
    {
      "computer_name": "computer",
      "cpu_brand": "Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz",
      "cpu_logical_cores": "8",
      "cpu_physical_cores": "4",
      "cpu_subtype": "Intel x86-64h Haswell",
      "cpu_type": "x86_64h",
      "hardware_model": "MacBookPro11,4",
      "hardware_serial": "NEW_HW_SERIAL",
      "hardware_vendor": "Apple Inc.",
      "hardware_version": "1.0",
      "hostname": "computer.local",
      "physical_memory": "17179869184",
      "uuid": "uuid"
    }
  ],
  "fleet_detail_query_uptime": [
    {
      "days": "20",
      "hours": "0",
      "minutes": "48",
      "seconds": "13",
      "total_seconds": "1730893"
    }
  ],
  "fleet_detail_query_osquery_flags": [
    {
      "name": "config_tls_refresh",
      "value": ""
    },
    {
      "name": "distributed_interval",
      "value": ""
    },
    {
      "name": "logger_tls_period",
      "value": ""
    }
  ],
  "fleet_detail_query_orbit_info": [
    {
      "name": "version",
      "value": "42"
    },
    {
      "name": "device_auth_token",
      "value": "foo"
    }
  ]
}
`

	var results fleet.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.NoError(t, err)

	var gotHost *fleet.Host
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		gotHost = host
		return nil
	}

	// Verify that results are ingested properly
	require.NoError(
		t, svc.SubmitDistributedQueryResults(ctx, results, map[string]fleet.OsqueryStatus{}, map[string]string{}, map[string]*fleet.Stats{}),
	)

	// osquery_info
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "1.8.2", gotHost.OsqueryVersion)

	// system_info
	assert.Equal(t, int64(17179869184), gotHost.Memory)
	assert.Equal(t, "computer.local", gotHost.Hostname)
	assert.Equal(t, "uuid", gotHost.UUID)
	assert.Equal(t, "NEW_HW_SERIAL", gotHost.HardwareSerial)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", gotHost.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, gotHost.Uptime)

	// osquery_flags
	assert.Equal(t, uint(0), gotHost.ConfigTLSRefresh)
	assert.Equal(t, uint(0), gotHost.DistributedInterval)
	assert.Equal(t, uint(0), gotHost.LoggerTLSPeriod)

	host.Hostname = "computer.local"
	host.DetailUpdatedAt = mockClock.Now()
	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 1) // fleet_no_policies_wildcard query
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// somehow confusingly, the query response above changed the host's platform
	// from windows to darwin
	// +1 due to fleet_no_policies_wildcard query.
	require.Equal(t, len(expectedDetailQueriesForPlatform(gotHost.Platform))+1, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)
}

func TestDetailQueries(t *testing.T) {
	ds := new(mock.Store)
	mockClock := clock.NewMockClock()
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, nil, lq, mockClock)

	host := &fleet.Host{
		ID:             1,
		Platform:       "linux",
		HardwareSerial: "HW_SERIAL",
	}
	ctx = hostctx.NewContext(ctx, host)

	lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}
	ds.LabelQueriesForHostFunc = func(context.Context, *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.SetOrUpdateMDMDataFunc = func(ctx context.Context, hostID uint, isServer, enrolled bool, serverURL string, installedFromDep bool, name string, fleetEnrollmentRef string) error {
		require.True(t, enrolled)
		require.False(t, installedFromDep)
		require.Equal(t, "hi.com", serverURL)
		require.Empty(t, fleetEnrollmentRef)
		return nil
	}
	ds.SetOrUpdateMunkiInfoFunc = func(ctx context.Context, hostID uint, version string, errs, warns []string) error {
		require.Equal(t, "3.4.5", version)
		return nil
	}
	ds.SetOrUpdateHostOrbitInfoFunc = func(
		ctx context.Context, hostID uint, version string, desktopVersion sql.NullString, scriptsEnabled sql.NullBool,
	) error {
		require.Equal(t, "42", version)
		require.Equal(t, sql.NullString{String: "1.2.3", Valid: true}, desktopVersion)
		require.Equal(t, sql.NullBool{Bool: true, Valid: true}, scriptsEnabled)
		return nil
	}
	ds.SetOrUpdateDeviceAuthTokenFunc = func(ctx context.Context, hostID uint, authToken string) error {
		require.Equal(t, uint(1), hostID)
		require.Equal(t, "foo", authToken)
		return nil
	}
	ds.SetOrUpdateHostDisksSpaceFunc = func(ctx context.Context, hostID uint, gigsAvailable, percentAvailable, gigsTotal float64) error {
		require.Equal(t, 277.0, gigsAvailable)
		require.Equal(t, 56.0, percentAvailable)
		require.Equal(t, 500.1, gigsTotal)
		return nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id != 1 {
			return nil, errors.New("not found")
		}
		return host, nil
	}

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, discovery, acc, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +1 for fleet_no_policies_wildcard
	if expected := expectedDetailQueriesForPlatform(host.Platform); !assert.Equal(t, len(expected)+1, len(queries)) {
		// this is just to print the diff between the expected and actual query
		// keys when the count assertion fails, to help debugging - they are not
		// expected to match.
		require.ElementsMatch(t, osqueryMapKeys(expected), distQueriesMapKeys(queries))
	}
	verifyDiscovery(t, queries, discovery)
	assert.NotZero(t, acc)

	resultJSON := `
{
"fleet_detail_query_network_interface": [
    {
        "address": "192.168.0.1",
        "broadcast": "192.168.0.255",
        "ibytes": "1601207629",
        "ierrors": "314179",
        "interface": "en0",
        "ipackets": "25698094",
        "last_change": "1474233476",
        "mac": "5f:3d:4b:10:25:82",
        "mask": "255.255.255.0",
        "metric": "1",
        "mtu": "1453",
        "obytes": "2607283152",
        "oerrors": "101010",
        "opackets": "12264603",
        "point_to_point": "",
        "type": "6"
    }
],
"fleet_detail_query_os_version": [
    {
        "platform": "darwin",
        "build": "15G1004",
        "major": "10",
        "minor": "10",
        "name": "Mac OS X",
        "patch": "6"
    }
],
"fleet_detail_query_osquery_info": [
    {
        "build_distro": "10.10",
        "build_platform": "darwin",
        "config_hash": "3c6e4537c4d0eb71a7c6dda19d",
        "config_valid": "1",
        "extensions": "active",
        "pid": "38113",
        "start_time": "1475603155",
        "version": "1.8.2",
        "watcher": "38112"
    }
],
"fleet_detail_query_system_info": [
    {
        "computer_name": "computer",
        "cpu_brand": "Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz",
        "cpu_logical_cores": "8",
        "cpu_physical_cores": "4",
        "cpu_subtype": "Intel x86-64h Haswell",
        "cpu_type": "x86_64h",
        "hardware_model": "MacBookPro11,4",
        "hardware_serial": "-1",
        "hardware_vendor": "Apple Inc.",
        "hardware_version": "1.0",
        "hostname": "computer.local",
        "physical_memory": "17179869184",
        "uuid": "uuid"
    }
],
"fleet_detail_query_uptime": [
    {
        "days": "20",
        "hours": "0",
        "minutes": "48",
        "seconds": "13",
        "total_seconds": "1730893"
    }
],
"fleet_detail_query_osquery_flags": [
    {
      "name":"config_tls_refresh",
      "value":"10"
    },
    {
      "name":"config_refresh",
      "value":"9"
    },
    {
      "name":"distributed_interval",
      "value":"5"
    },
    {
      "name":"logger_tls_period",
      "value":"60"
    }
],
"fleet_detail_query_users": [
    {
      "uid": "1234",
      "username": "user1",
      "type": "sometype",
      "groupname": "somegroup",
	  "shell": "someloginshell"
    },
	{
      "uid": "5678",
      "username": "user2",
      "type": "sometype",
      "groupname": "somegroup"
    }
],
"fleet_detail_query_software_macos": [
    {
      "name": "app1",
      "version": "1.0.0",
      "source": "source1"
    },
    {
      "name": "app2",
      "version": "1.0.0",
      "source": "source2",
      "bundle_identifier": "somebundle"
    }
],
"fleet_detail_query_disk_space_unix": [
	{
		"percent_disk_space_available": "56",
		"gigs_disk_space_available": "277.0",
		"gigs_total_disk_space": "500.1"
	}
],
"fleet_detail_query_mdm": [
	{
		"enrolled": "true",
		"server_url": "hi.com",
		"installed_from_dep": "false"
	}
],
"fleet_detail_query_munki_info": [
	{
		"version": "3.4.5"
	}
],
"fleet_detail_query_orbit_info": [
	{
		"version": "42",
		"desktop_version": "1.2.3",
		"scripts_enabled": "1"
	}
]
}
`

	var results fleet.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.NoError(t, err)

	var gotHost *fleet.Host
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		gotHost = host
		return nil
	}
	var gotUsers []fleet.HostUser
	ds.SaveHostUsersFunc = func(ctx context.Context, hostID uint, users []fleet.HostUser) error {
		if hostID != 1 {
			return errors.New("not found")
		}
		gotUsers = users
		return nil
	}
	var gotSoftware []fleet.Software
	ds.UpdateHostSoftwareFunc = func(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
		if hostID != 1 {
			return nil, errors.New("not found")
		}
		gotSoftware = software
		return nil, nil
	}

	ds.UpdateHostSoftwareInstalledPathsFunc = func(ctx context.Context, hostID uint, paths map[string]struct{}, result *fleet.UpdateHostSoftwareDBResult) error {
		return nil
	}

	// Verify that results are ingested properly
	require.NoError(
		t, svc.SubmitDistributedQueryResults(ctx, results, map[string]fleet.OsqueryStatus{}, map[string]string{}, map[string]*fleet.Stats{}),
	)
	require.NotNil(t, gotHost)

	require.True(t, ds.SetOrUpdateMDMDataFuncInvoked)
	require.True(t, ds.SetOrUpdateMunkiInfoFuncInvoked)
	require.True(t, ds.SetOrUpdateHostDisksSpaceFuncInvoked)

	// osquery_info
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "1.8.2", gotHost.OsqueryVersion)

	// system_info
	assert.Equal(t, int64(17179869184), gotHost.Memory)
	assert.Equal(t, "computer.local", gotHost.Hostname)
	assert.Equal(t, "uuid", gotHost.UUID)
	// The hardware serial should not have updated because return value was -1. See: https://github.com/fleetdm/fleet/issues/19789
	assert.Equal(t, "HW_SERIAL", gotHost.HardwareSerial)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", gotHost.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, gotHost.Uptime)

	// osquery_flags
	assert.Equal(t, uint(10), gotHost.ConfigTLSRefresh)
	assert.Equal(t, uint(5), gotHost.DistributedInterval)
	assert.Equal(t, uint(60), gotHost.LoggerTLSPeriod)

	// users
	require.Len(t, gotUsers, 2)
	assert.Equal(t, fleet.HostUser{
		Uid:       1234,
		Username:  "user1",
		Type:      "sometype",
		GroupName: "somegroup",
		Shell:     "someloginshell",
	}, gotUsers[0])
	assert.Equal(t, fleet.HostUser{
		Uid:       5678,
		Username:  "user2",
		Type:      "sometype",
		GroupName: "somegroup",
		Shell:     "",
	}, gotUsers[1])

	// software
	require.Len(t, gotSoftware, 2)
	assert.Equal(t, []fleet.Software{
		{
			Name:    "app1",
			Version: "1.0.0",
			Source:  "source1",
		},
		{
			Name:             "app2",
			Version:          "1.0.0",
			BundleIdentifier: "somebundle",
			Source:           "source2",
		},
	}, gotSoftware)

	host.Hostname = "computer.local"
	host.Platform = "darwin"
	host.DetailUpdatedAt = mockClock.Now()
	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 1) // fleet_no_policies_wildcard query
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, discovery, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +1 fleet_no_policies_wildcard query
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+1, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	assert.Zero(t, acc)
}

func TestMDMQueries(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{
		clock:    clock.NewMockClock(),
		logger:   log.NewNopLogger(),
		config:   config.TestConfig(),
		ds:       ds,
		jitterMu: new(sync.Mutex),
		jitterH:  make(map[time.Duration]*jitterHashTable),
	}

	expectedMDMQueries := []struct {
		name           string
		discoveryTable string
	}{
		{"fleet_detail_query_mdm_config_profiles_darwin", "macos_profiles"},
		{"fleet_detail_query_mdm_disk_encryption_key_file_darwin", "filevault_prk"},
		{"fleet_detail_query_mdm_disk_encryption_key_file_lines_darwin", "file_lines"},
	}

	mdmEnabled := true
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: mdmEnabled}}, nil
	}

	host := fleet.Host{
		ID:       1,
		Platform: "darwin",
		NodeKey:  ptr.String("test_key"),
		Hostname: "test_hostname",
		UUID:     "test_uuid",
		TeamID:   nil,
	}
	ctx := hostctx.NewContext(context.Background(), &host)

	// MDM enabled, darwin
	queries, discovery, err := svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	for _, q := range expectedMDMQueries {
		require.Contains(t, queries, q.name)
		d, ok := discovery[q.name]
		require.True(t, ok)
		require.Contains(t, d, fmt.Sprintf("name = '%s'", q.discoveryTable))
	}

	// MDM disabled, darwin
	mdmEnabled = false
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	for _, q := range expectedMDMQueries {
		require.NotContains(t, queries, q.name)
		require.NotContains(t, discovery, q.name)
	}

	// MDM enabled, not darwin
	mdmEnabled = true
	host.Platform = "windows"
	ctx = hostctx.NewContext(context.Background(), &host)
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	for _, q := range expectedMDMQueries {
		require.NotContains(t, queries, q.name)
		require.NotContains(t, discovery, q.name)
	}

	// MDM disabled, not darwin
	mdmEnabled = false
	queries, discovery, err = svc.detailQueriesForHost(ctx, &host)
	require.NoError(t, err)
	for _, q := range expectedMDMQueries {
		require.NotContains(t, queries, q.name)
		require.NotContains(t, discovery, q.name)
	}
}

func TestNewDistributedQueryCampaign(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	rs := &mockresult.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := live_query_mock.New(t)
	mockClock := clock.NewMockClock()
	svc, ctx := newTestServiceWithClock(t, ds, rs, lq, mockClock)

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	var gotQuery *fleet.Query
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		gotQuery = query
		query.ID = 42
		return query, nil
	}
	var gotCampaign *fleet.DistributedQueryCampaign
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		gotCampaign = camp
		camp.ID = 21
		return camp, nil
	}
	var gotTargets []*fleet.DistributedQueryCampaignTarget
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		gotTargets = append(gotTargets, target)
		return target, nil
	}

	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1, 3, 5}, nil
	}
	lq.On("RunQuery", "21", "select year, month, day, hour, minutes, seconds from time", []uint{1, 3, 5}).Return(nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{
		User: &fleet.User{
			ID:         0,
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	})
	q := "select year, month, day, hour, minutes, seconds from time"
	campaign, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)
	assert.Equal(t, gotQuery.ID, gotCampaign.QueryID)
	assert.Equal(t, []*fleet.DistributedQueryCampaignTarget{
		{
			Type:                       fleet.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   2,
		},
		{
			Type:                       fleet.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   1,
		},
	}, gotTargets,
	)
}

func TestDistributedQueryResults(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, rs, lq, mockClock)

	campaign := &fleet.DistributedQueryCampaign{ID: 42}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	host := &fleet.Host{
		ID:       1,
		Platform: "windows",
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id != 1 {
			return nil, errors.New("not found")
		}
		return host, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		if host.ID != 1 {
			return errors.New("not found")
		}
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}

	hostCtx := hostctx.NewContext(ctx, host)

	lq.On("QueriesForHost", uint(1)).Return(
		map[string]string{
			fmt.Sprint(campaign.ID): "select * from time",
		},
		nil,
	)
	lq.On("QueryCompletedByHost", fmt.Sprint(campaign.ID), host.ID).Return(nil)

	// Now we should get the active distributed query
	queries, discovery, acc, err := svc.GetDistributedQueries(hostCtx)
	require.NoError(t, err)
	// +1 for the distributed query for campaign ID 42, +1 for windows update history, +1 for the fleet_no_policies_wildcard query.
	if expected := expectedDetailQueriesForPlatform(host.Platform); !assert.Equal(t, len(expected)+3, len(queries)) {
		// this is just to print the diff between the expected and actual query
		// keys when the count assertion fails, to help debugging - they are not
		// expected to match.
		require.ElementsMatch(t, osqueryMapKeys(expected), distQueriesMapKeys(queries))
	}
	verifyDiscovery(t, queries, discovery)
	queryKey := fmt.Sprintf("%s%d", hostDistributedQueryPrefix, campaign.ID)
	assert.Equal(t, "select * from time", queries[queryKey])
	assert.NotZero(t, acc)

	expectedRows := []map[string]string{
		{
			"year":    "2016",
			"month":   "11",
			"day":     "11",
			"hour":    "6",
			"minutes": "12",
			"seconds": "10",
		},
	}
	results := map[string][]map[string]string{
		queryKey: expectedRows,
	}
	expectedStats := fleet.Stats{
		UserTime: uint64(1),
	}
	stats := map[string]*fleet.Stats{
		queryKey: &expectedStats,
	}

	// TODO use service method
	readChan, err := rs.ReadChannel(context.Background(), *campaign)
	require.NoError(t, err)

	// We need to listen for the result in a separate thread to prevent the
	// write to the result channel from failing
	var waitSetup, waitComplete sync.WaitGroup
	waitSetup.Add(1)
	waitComplete.Add(1)
	go func() {
		waitSetup.Done()
		select {
		case val := <-readChan:
			if res, ok := val.(fleet.DistributedQueryResult); ok {
				assert.Equal(t, campaign.ID, res.DistributedQueryCampaignID)
				assert.Equal(t, expectedRows, res.Rows)
				assert.Equal(t, host.ID, res.Host.ID)
				assert.Equal(t, host.Hostname, res.Host.Hostname)
				assert.Equal(t, host.DisplayName(), res.Host.DisplayName)
				assert.Equal(t, &expectedStats, res.Stats)
			} else {
				t.Error("Wrong result type")
			}
			assert.NotNil(t, val)

		case <-time.After(1 * time.Second):
			t.Error("No result received")
		}
		waitComplete.Done()
	}()

	waitSetup.Wait()
	// Sleep a short time to ensure that the above goroutine is blocking on
	// the channel read (the waitSetup.Wait() is not necessarily sufficient
	// if there is a context switch immediately after waitSetup.Done() is
	// called). This should be a small price to pay to prevent flakiness in
	// this test.
	time.Sleep(10 * time.Millisecond)

	err = svc.SubmitDistributedQueryResults(
		hostCtx, results, map[string]fleet.OsqueryStatus{}, map[string]string{}, stats,
	)
	require.NoError(t, err)
	// Sleep to ensure checks in the goroutine are actually done.
	time.Sleep(10 * time.Millisecond)
}

func TestIngestDistributedQueryParseIdError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	host := fleet.Host{ID: 1}
	err := svc.ingestDistributedQuery(context.Background(), host, "bad_name", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse campaign")
}

func TestIngestDistributedQueryOrphanedCampaignLoadError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return nil, errors.New("missing campaign")
	}

	lq.On("StopQuery", "42").Return(nil)

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading orphaned campaign")
}

func TestIngestDistributedQueryOrphanedCampaignWaitListener(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-1 * time.Second),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return campaign, nil
	}

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaignID=42 waiting for listener")
}

func TestIngestDistributedQueryOrphanedCloseError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-2 * time.Minute),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, campaign *fleet.DistributedQueryCampaign) error {
		return errors.New("failed save")
	}

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing orphaned campaign")
}

func TestIngestDistributedQueryOrphanedStopError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-2 * time.Minute),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, campaign *fleet.DistributedQueryCampaign) error {
		return nil
	}
	lq.On("StopQuery", fmt.Sprint(campaign.ID)).Return(errors.New("failed"))

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopping orphaned campaign")
}

func TestIngestDistributedQueryOrphanedStop(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-2 * time.Minute),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, campaign *fleet.DistributedQueryCampaign) error {
		return nil
	}
	lq.On("StopQuery", fmt.Sprint(campaign.ID)).Return(nil)

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaignID=42 stopped")
	lq.AssertExpectations(t)
}

func TestIngestDistributedQueryRecordCompletionError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{ID: 42}
	host := fleet.Host{ID: 1}

	lq.On("QueryCompletedByHost", fmt.Sprint(campaign.ID), host.ID).Return(errors.New("fail"))

	go func() {
		ch, err := rs.ReadChannel(context.Background(), *campaign)
		require.NoError(t, err)
		<-ch
	}()
	time.Sleep(10 * time.Millisecond)

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "record query completion")
	lq.AssertExpectations(t)
}

func TestIngestDistributedQuery(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{ID: 42}
	host := fleet.Host{ID: 1}

	lq.On("QueryCompletedByHost", fmt.Sprint(campaign.ID), host.ID).Return(nil)

	go func() {
		ch, err := rs.ReadChannel(context.Background(), *campaign)
		require.NoError(t, err)
		<-ch
	}()
	time.Sleep(10 * time.Millisecond)

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, "", nil)
	require.NoError(t, err)
	lq.AssertExpectations(t)
}

func TestUpdateHostIntervals(t *testing.T) {
	ds := new(mock.Store)

	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		return nil, nil
	}

	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{}, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}

	testCases := []struct {
		name                  string
		initIntervals         fleet.HostOsqueryIntervals
		finalIntervals        fleet.HostOsqueryIntervals
		configOptions         json.RawMessage
		updateIntervalsCalled bool
	}{
		{
			"Both updated",
			fleet.HostOsqueryIntervals{
				ConfigTLSRefresh: 60,
			},
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			json.RawMessage(`{"options": {
				"distributed_interval": 11,
				"logger_tls_period":    33,
				"logger_plugin":        "tls"
			}}`),
			true,
		},
		{
			"Only logger_tls_period updated",
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				ConfigTLSRefresh:    60,
			},
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			json.RawMessage(`{"options": {
				"distributed_interval": 11,
				"logger_tls_period":    33
			}}`),
			true,
		},
		{
			"Only distributed_interval updated",
			fleet.HostOsqueryIntervals{
				ConfigTLSRefresh: 60,
				LoggerTLSPeriod:  33,
			},
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			json.RawMessage(`{"options": {
				"distributed_interval": 11,
				"logger_tls_period":    33
			}}`),
			true,
		},
		{
			"Fleet not managing distributed_interval",
			fleet.HostOsqueryIntervals{
				ConfigTLSRefresh:    60,
				DistributedInterval: 11,
			},
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			json.RawMessage(`{"options":{
				"logger_tls_period": 33
			}}`),
			true,
		},
		{
			"config_refresh should also cause an update",
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    42,
			},
			json.RawMessage(`{"options":{
				"distributed_interval": 11,
				"logger_tls_period":    33,
				"config_refresh":    42
			}}`),
			true,
		},
		{
			"update intervals should not be called with no changes",
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			fleet.HostOsqueryIntervals{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			json.RawMessage(`{"options":{
				"distributed_interval": 11,
				"logger_tls_period":    33
			}}`),
			false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := hostctx.NewContext(ctx, &fleet.Host{
				ID:                  1,
				NodeKey:             ptr.String("123456"),
				DistributedInterval: tt.initIntervals.DistributedInterval,
				ConfigTLSRefresh:    tt.initIntervals.ConfigTLSRefresh,
				LoggerTLSPeriod:     tt.initIntervals.LoggerTLSPeriod,
			})

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":` + string(tt.configOptions) + `}`))}, nil
			}

			updateIntervalsCalled := false
			ds.UpdateHostOsqueryIntervalsFunc = func(ctx context.Context, hostID uint, intervals fleet.HostOsqueryIntervals) error {
				if hostID != 1 {
					return errors.New("not found")
				}
				updateIntervalsCalled = true
				assert.Equal(t, tt.finalIntervals, intervals)
				return nil
			}

			_, err := svc.GetClientConfig(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.updateIntervalsCalled, updateIntervalsCalled)
		})
	}
}

func TestAuthenticationErrors(t *testing.T) {
	ms := new(mock.Store)
	ms.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, nil
	}

	svc, ctx := newTestService(t, ms, nil, nil)

	_, _, err := svc.AuthenticateHost(ctx, "")
	require.Error(t, err)
	require.True(t, err.(*osqueryError).NodeInvalid())

	ms.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return &fleet.Host{ID: 1}, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	_, _, err = svc.AuthenticateHost(ctx, "foo")
	require.NoError(t, err)

	// return not found error
	ms.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, newNotFoundError()
	}

	_, _, err = svc.AuthenticateHost(ctx, "foo")
	require.Error(t, err)
	require.True(t, err.(*osqueryError).NodeInvalid())

	// return other error
	ms.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, errors.New("foo")
	}

	_, _, err = svc.AuthenticateHost(ctx, "foo")
	require.NotNil(t, err)
	require.False(t, err.(*osqueryError).NodeInvalid())
}

func TestGetHostIdentifier(t *testing.T) {
	t.Parallel()

	details := map[string](map[string]string){
		"osquery_info": map[string]string{
			"uuid":        "foouuid",
			"instance_id": "fooinstance",
		},
		"system_info": map[string]string{
			"hostname": "foohost",
		},
	}

	emptyDetails := map[string](map[string]string){
		"osquery_info": map[string]string{
			"uuid":        "",
			"instance_id": "",
		},
		"system_info": map[string]string{
			"hostname": "",
		},
	}

	testCases := []struct {
		identifierOption   string
		providedIdentifier string
		details            map[string](map[string]string)
		expected           string
		shouldPanic        bool
	}{
		// Panix
		{identifierOption: "bad", shouldPanic: true},
		{identifierOption: "", shouldPanic: true},

		// Missing details
		{identifierOption: "instance", providedIdentifier: "foobar", expected: "foobar"},
		{identifierOption: "uuid", providedIdentifier: "foobar", expected: "foobar"},
		{identifierOption: "hostname", providedIdentifier: "foobar", expected: "foobar"},
		{identifierOption: "provided", providedIdentifier: "foobar", expected: "foobar"},

		// Empty details
		{identifierOption: "instance", providedIdentifier: "foobar", details: emptyDetails, expected: "foobar"},
		{identifierOption: "uuid", providedIdentifier: "foobar", details: emptyDetails, expected: "foobar"},
		{identifierOption: "hostname", providedIdentifier: "foobar", details: emptyDetails, expected: "foobar"},
		{identifierOption: "provided", providedIdentifier: "foobar", details: emptyDetails, expected: "foobar"},

		// Successes
		{identifierOption: "instance", providedIdentifier: "foobar", details: details, expected: "fooinstance"},
		{identifierOption: "uuid", providedIdentifier: "foobar", details: details, expected: "foouuid"},
		{identifierOption: "hostname", providedIdentifier: "foobar", details: details, expected: "foohost"},
		{identifierOption: "provided", providedIdentifier: "foobar", details: details, expected: "foobar"},
	}
	logger := log.NewNopLogger()

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(
					t,
					func() { getHostIdentifier(logger, tt.identifierOption, tt.providedIdentifier, tt.details) },
				)
				return
			}

			assert.Equal(
				t,
				tt.expected,
				getHostIdentifier(logger, tt.identifierOption, tt.providedIdentifier, tt.details),
			)
		})
	}
}

func TestDistributedQueriesLogsManyErrors(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	host := &fleet.Host{
		ID:       1,
		Platform: "darwin",
	}

	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return authz.CheckMissingWithResponse(nil)
	}
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferred bool) error {
		return errors.New("something went wrong")
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveHostAdditionalFunc = func(ctx context.Context, hostID uint, additional *json.RawMessage) error {
		return errors.New("something went wrong")
	}

	lCtx := &fleetLogging.LoggingContext{}
	ctx = fleetLogging.NewContext(ctx, lCtx)
	ctx = hostctx.NewContext(ctx, host)

	err := svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostDetailQueryPrefix + "network_interface_unix": {{"col1": "val1"}}, // we need one detail query that updates hosts.
			hostLabelQueryPrefix + "1":                       {{"col1": "val1"}},
			hostAdditionalQueryPrefix + "1":                  {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)

	lCtx.Log(ctx, logger)

	logs := buf.String()
	parts := strings.Split(strings.TrimSpace(logs), "\n")
	require.Len(t, parts, 1)

	var logData map[string]interface{}
	err = json.Unmarshal([]byte(parts[0]), &logData)
	require.NoError(t, err)
	assert.Equal(t, "something went wrong || something went wrong", logData["err"])
	assert.Equal(t, "Missing authorization check", logData["internal"])
}

func TestDistributedQueriesReloadsHostIfDetailsAreIn(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	host := &fleet.Host{
		ID:       42,
		Platform: "darwin",
	}

	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ctx = hostctx.NewContext(ctx, host)

	err := svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostDetailQueryPrefix + "network_interface_unix": {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	assert.True(t, ds.UpdateHostFuncInvoked)
}

func TestObserversCanOnlyRunDistributedCampaigns(t *testing.T) {
	ds := new(mock.Store)
	rs := &mockresult.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := live_query_mock.New(t)
	mockClock := clock.NewMockClock()
	svc, ctx := newTestServiceWithClock(t, ds, rs, lq, mockClock)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		return camp, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{
			ID:             42,
			Name:           "query",
			Query:          "select 1;",
			ObserverCanRun: false,
		}, nil
	}
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{
		User: &fleet.User{ID: 0, GlobalRole: ptr.String(fleet.RoleObserver)},
	})

	q := "select year, month, day, hour, minutes, seconds from time"
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	_, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.Error(t, err)

	_, err = svc.NewDistributedQueryCampaign(viewerCtx, "", ptr.Uint(42), fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.Error(t, err)

	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{
			ID:             42,
			Name:           "query",
			Query:          "select 1;",
			ObserverCanRun: true,
		}, nil
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		camp.ID = 21
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1, 3, 5}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	lq.On("RunQuery", "21", "select 1;", []uint{1, 3, 5}).Return(nil)
	_, err = svc.NewDistributedQueryCampaign(viewerCtx, "", ptr.Uint(42), fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)
}

func TestTeamMaintainerCanRunNewDistributedCampaigns(t *testing.T) {
	ds := new(mock.Store)
	rs := &mockresult.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := live_query_mock.New(t)
	mockClock := clock.NewMockClock()
	svc, ctx := newTestServiceWithClock(t, ds, rs, lq, mockClock)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		return camp, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{
			ID:             42,
			AuthorID:       ptr.Uint(99),
			Name:           "query",
			Query:          "select 1;",
			ObserverCanRun: false,
		}, nil
	}
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{
		User: &fleet.User{ID: 99, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 123}, Role: fleet.RoleMaintainer}}},
	})

	q := "select year, month, day, hour, minutes, seconds from time"
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	// var gotQuery *fleet.Query
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		// gotQuery = query
		query.ID = 42
		return query, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{}, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1, 3, 5}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	lq.On("RunQuery", "0", "select year, month, day, hour, minutes, seconds from time", []uint{1, 3, 5}).Return(nil)
	_, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}, TeamIDs: []uint{123}})
	require.NoError(t, err)
}

func TestPolicyQueries(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, nil, lq, mockClock)

	host := &fleet.Host{
		Platform: "darwin",
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, gotHost *fleet.Host) error {
		host = gotHost
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}

	lq.On("QueriesForHost", uint(0)).Return(map[string]string{}, nil)

	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{"1": "select 1", "2": "select 42;"}, nil
	}
	recordedResults := make(map[uint]*bool)
	ds.RecordPolicyQueryExecutionsFunc = func(ctx context.Context, gotHost *fleet.Host, results map[uint]*bool, updated time.Time, deferred bool) error {
		recordedResults = results
		host = gotHost
		return nil
	}
	ds.FlippingPoliciesForHostFunc = func(ctx context.Context, hostID uint, incomingResults map[uint]*bool) (newFailing []uint, newPassing []uint, err error) {
		return nil, nil, nil
	}

	ctx = hostctx.NewContext(ctx, host)

	queries, discovery, _, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +2 policy queries
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+2, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)

	checkPolicyResults := func(queries map[string]string) {
		hasPolicy1, hasPolicy2 := false, false
		for name := range queries {
			if strings.HasPrefix(name, hostPolicyQueryPrefix) {
				if name[len(hostPolicyQueryPrefix):] == "1" {
					hasPolicy1 = true
				}
				if name[len(hostPolicyQueryPrefix):] == "2" {
					hasPolicy2 = true
				}
			}
		}
		assert.True(t, hasPolicy1)
		assert.True(t, hasPolicy2)
	}

	checkPolicyResults(queries)

	// Record a query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {{"col1": "val1"}},
			hostPolicyQueryPrefix + "2": {},
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1,
		},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	require.Len(t, recordedResults, 2)
	require.NotNil(t, recordedResults[1])
	require.True(t, *recordedResults[1])
	result, ok := recordedResults[2]
	require.True(t, ok)
	require.Nil(t, result)

	noPolicyResults := func(queries map[string]string) {
		hasAnyPolicy := false
		for name := range queries {
			if strings.HasPrefix(name, hostPolicyQueryPrefix) {
				hasAnyPolicy = true
				break
			}
		}
		assert.False(t, hasAnyPolicy)
	}

	// After the first time we get policies and update the host, then there shouldn't be any policies.
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform)), len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	noPolicyResults(queries)

	// Let's move time forward, there should be policies now.
	mockClock.AddTime(2 * time.Hour)

	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +2 policy queries
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+2, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	checkPolicyResults(queries)

	// Record another query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {{"col1": "val1"}},
			hostPolicyQueryPrefix + "2": {},
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1,
		},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	require.Len(t, recordedResults, 2)
	require.NotNil(t, recordedResults[1])
	require.True(t, *recordedResults[1])
	result, ok = recordedResults[2]
	require.True(t, ok)
	require.Nil(t, result)

	// There shouldn't be any policies now.
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform)), len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	noPolicyResults(queries)

	// With refetch requested policy queries should be returned.
	host.RefetchRequested = true
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +2 policy queries
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+2, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	checkPolicyResults(queries)

	// Record another query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {{"col1": "val1"}},
			hostPolicyQueryPrefix + "2": {},
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1,
		},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	require.NotNil(t, recordedResults[1])
	require.True(t, *recordedResults[1])
	result, ok = recordedResults[2]
	require.True(t, ok)
	require.Nil(t, result)

	// SubmitDistributedQueryResults will set RefetchRequested to false.
	require.False(t, host.RefetchRequested)

	// There shouldn't be any policies now.
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform)), len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	noPolicyResults(queries)
}

func TestPolicyWebhooks(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := live_query_mock.New(t)
	pool := redistest.SetupRedis(t, t.Name(), false, false, false)
	failingPolicySet := redis_policy_set.NewFailingTest(t, pool)
	testConfig := config.TestConfig()
	svc, ctx := newTestServiceWithConfig(t, ds, testConfig, nil, lq, &TestServerOpts{
		FailingPolicySet: failingPolicySet,
		Clock:            mockClock,
	})

	host := &fleet.Host{
		ID:       5,
		Platform: "darwin",
		Hostname: "test.hostname",
	}

	lq.On("QueriesForHost", uint(5)).Return(map[string]string{}, nil)
	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, gotHost *fleet.Host) error {
		host = gotHost
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Features: fleet.Features{
				EnableHostUsers:         true,
				EnableSoftwareInventory: true,
			},
			WebhookSettings: fleet.WebhookSettings{
				FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
					Enable:    true,
					PolicyIDs: []uint{1, 2, 3},
				},
			},
		}, nil
	}

	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{
			"1": "select 1;",                       // passing policy
			"2": "select * from unexistent_table;", // policy that fails to execute (e.g. missing table)
			"3": "select 1 where 1 = 0;",           // failing policy
		}, nil
	}
	recordedResults := make(map[uint]*bool)
	ds.RecordPolicyQueryExecutionsFunc = func(ctx context.Context, gotHost *fleet.Host, results map[uint]*bool, updated time.Time, deferred bool) error {
		recordedResults = results
		host = gotHost
		return nil
	}
	ctx = hostctx.NewContext(ctx, host)

	queries, discovery, _, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +3 for policies
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+3, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)

	checkPolicyResults := func(queries map[string]string) {
		hasPolicy1, hasPolicy2, hasPolicy3 := false, false, false
		for name := range queries {
			if strings.HasPrefix(name, hostPolicyQueryPrefix) {
				switch name[len(hostPolicyQueryPrefix):] {
				case "1":
					hasPolicy1 = true
				case "2":
					hasPolicy2 = true
				case "3":
					hasPolicy3 = true
				}
			}
		}
		assert.True(t, hasPolicy1)
		assert.True(t, hasPolicy2)
		assert.True(t, hasPolicy3)
	}

	checkPolicyResults(queries)

	ds.FlippingPoliciesForHostFunc = func(ctx context.Context, hostID uint, incomingResults map[uint]*bool) (newFailing []uint, newPassing []uint, err error) {
		return []uint{3}, nil, nil
	}

	// Record a query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {{"col1": "val1"}}, // succeeds
			hostPolicyQueryPrefix + "2": {},                 // didn't execute
			hostPolicyQueryPrefix + "3": {},                 // fails
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1, // didn't execute
		},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	require.Len(t, recordedResults, 3)
	require.NotNil(t, recordedResults[1])
	require.True(t, *recordedResults[1])
	result, ok := recordedResults[2]
	require.True(t, ok)
	require.Nil(t, result)
	require.NotNil(t, recordedResults[3])
	require.False(t, *recordedResults[3])

	cmpSets := func(expSets map[uint][]fleet.PolicySetHost) error {
		actualSets, err := failingPolicySet.ListSets()
		if err != nil {
			return err
		}
		var expSets_ []uint
		for expSet := range expSets {
			expSets_ = append(expSets_, expSet)
		}
		sort.Slice(expSets_, func(i, j int) bool {
			return expSets_[i] < expSets_[j]
		})
		sort.Slice(actualSets, func(i, j int) bool {
			return actualSets[i] < actualSets[j]
		})
		if !reflect.DeepEqual(actualSets, expSets_) {
			return fmt.Errorf("sets mismatch: %+v vs %+v", actualSets, expSets_)
		}
		for expID, expHosts := range expSets {
			actualHosts, err := failingPolicySet.ListHosts(expID)
			if err != nil {
				return err
			}
			sort.Slice(actualHosts, func(i, j int) bool {
				return actualHosts[i].ID < actualHosts[j].ID
			})
			sort.Slice(expHosts, func(i, j int) bool {
				return expHosts[i].ID < expHosts[j].ID
			})
			if !reflect.DeepEqual(actualHosts, expHosts) {
				return fmt.Errorf("hosts mismatch %d: %+v vs %+v", expID, actualHosts, expHosts)
			}
		}
		return nil
	}

	assert.Eventually(t, func() bool {
		err = cmpSets(map[uint][]fleet.PolicySetHost{
			3: {{
				ID:       host.ID,
				Hostname: host.Hostname,
			}},
		})
		return err == nil
	}, 1*time.Minute, 250*time.Millisecond)
	require.NoError(t, err)

	noPolicyResults := func(queries map[string]string) {
		hasAnyPolicy := false
		for name := range queries {
			if strings.HasPrefix(name, hostPolicyQueryPrefix) {
				hasAnyPolicy = true
				break
			}
		}
		assert.False(t, hasAnyPolicy)
	}

	// After the first time we get policies and update the host, then there shouldn't be any policies.
	ctx = hostctx.NewContext(ctx, host)
	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform)), len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	noPolicyResults(queries)

	// Let's move time forward, there should be policies now.
	mockClock.AddTime(2 * time.Hour)

	queries, discovery, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +3 for policies
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+3, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)
	checkPolicyResults(queries)

	ds.FlippingPoliciesForHostFunc = func(ctx context.Context, hostID uint, incomingResults map[uint]*bool) (newFailing []uint, newPassing []uint, err error) {
		return []uint{1}, []uint{3}, nil
	}

	// Record another query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {},                 // 1 now fails
			hostPolicyQueryPrefix + "2": {},                 // didn't execute
			hostPolicyQueryPrefix + "3": {{"col1": "val1"}}, // 1 now succeeds
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1, // didn't execute
		},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	require.Len(t, recordedResults, 3)
	require.NotNil(t, recordedResults[1])
	require.False(t, *recordedResults[1])
	result, ok = recordedResults[2]
	require.True(t, ok)
	require.Nil(t, result)
	require.NotNil(t, recordedResults[3])
	require.True(t, *recordedResults[3])

	assert.Eventually(t, func() bool {
		err = cmpSets(map[uint][]fleet.PolicySetHost{
			1: {{
				ID:       host.ID,
				Hostname: host.Hostname,
			}},
			3: {},
		})
		return err == nil
	}, 1*time.Minute, 250*time.Millisecond)
	require.NoError(t, err)

	// Simulate webhook trigger by removing the hosts.
	err = failingPolicySet.RemoveHosts(1, []fleet.PolicySetHost{{
		ID:       host.ID,
		Hostname: host.Hostname,
	}})
	require.NoError(t, err)

	ds.FlippingPoliciesForHostFunc = func(ctx context.Context, hostID uint, incomingResults map[uint]*bool) (newFailing []uint, newPassing []uint, err error) {
		return []uint{}, []uint{2}, nil
	}

	// Record another query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {},                 // continues to fail
			hostPolicyQueryPrefix + "2": {{"col1": "val1"}}, // now passes
			hostPolicyQueryPrefix + "3": {{"col1": "val1"}}, // continues to succeed
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
		map[string]*fleet.Stats{},
	)
	require.NoError(t, err)
	require.Len(t, recordedResults, 3)
	require.NotNil(t, recordedResults[1])
	require.False(t, *recordedResults[1])
	require.NotNil(t, recordedResults[2])
	require.True(t, *recordedResults[2])
	require.NotNil(t, recordedResults[3])
	require.True(t, *recordedResults[3])

	assert.Eventually(t, func() bool {
		err = cmpSets(map[uint][]fleet.PolicySetHost{
			1: {},
			3: {},
		})
		return err == nil
	}, 1*time.Minute, 250*time.Millisecond)
	require.NoError(t, err)
}

// If the live query store (Redis) is down we still (see #3503)
// want hosts to get queries and continue to check in.
func TestLiveQueriesFailing(t *testing.T) {
	ds := new(mock.Store)
	lq := live_query_mock.New(t)
	cfg := config.TestConfig()
	buf := new(bytes.Buffer)
	logger := log.NewLogfmtLogger(buf)
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, lq, &TestServerOpts{
		Logger: logger,
	})

	hostID := uint(1)
	host := &fleet.Host{
		ID:       hostID,
		Platform: "darwin",
	}
	lq.On("QueriesForHost", hostID).Return(
		map[string]string{},
		errors.New("failed to get queries for host"),
	)

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		}}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	ctx = hostctx.NewContext(ctx, host)

	queries, discovery, _, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	// +1 to account for the fleet_no_policies_wildcard query.
	require.Equal(t, len(expectedDetailQueriesForPlatform(host.Platform))+1, len(queries), distQueriesMapKeys(queries))
	verifyDiscovery(t, queries, discovery)

	logs, err := io.ReadAll(buf)
	require.NoError(t, err)
	require.Contains(t, string(logs), "level=error")
	require.Contains(t, string(logs), "failed to get queries for host")
}

func distQueriesMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, strings.TrimPrefix(k, "fleet_detail_query_"))
	}
	sort.Strings(keys)
	return keys
}

func osqueryMapKeys(m map[string]osquery_utils.DetailQuery) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestPreProcessSoftwareResults(t *testing.T) {
	foobarApp := map[string]string{
		"name":              "Foobar.app",
		"version":           "1.2.3",
		"type":              "Application (macOS)",
		"bundle_identifier": "com.zoobar.foobar",
		"extension_id":      "",
		"browser":           "",
		"source":            "apps",
		"vendor":            "",
		"last_opened_at":    "0",
		"installed_path":    "/some/path",
	}
	zoobarApp := map[string]string{
		"name":              "Zoobar.app",
		"version":           "3.2.1",
		"type":              "Application (macOS)",
		"bundle_identifier": "com.acme.zoobar",
		"extension_id":      "",
		"browser":           "",
		"source":            "apps",
		"vendor":            "",
		"last_opened_at":    "0",
		"installed_path":    "/some/other/path",
	}
	foobarVSCodeExtension := map[string]string{
		"name":              "vendor-x.foobar",
		"version":           "2024.2.1",
		"type":              "IDE extension (VS Code)",
		"bundle_identifier": "",
		"extension_id":      "",
		"browser":           "",
		"source":            "vscode_extensions",
		"vendor":            "VendorX",
		"last_opened_at":    "",
		"installed_path":    "/some/foobar/path",
	}
	zoobarVSCodeExtension := map[string]string{
		"name":              "vendor-x.zoobar",
		"version":           "2023.2.1",
		"type":              "IDE extension (VS Code)",
		"bundle_identifier": "",
		"extension_id":      "",
		"browser":           "",
		"source":            "vscode_extensions",
		"vendor":            "VendorX",
		"last_opened_at":    "",
		"installed_path":    "/some/zoobar/path",
	}
	someRow := map[string]string{
		"1": "1",
	}
	appToOverride := map[string]string{
		"name":              "OverrideMe.app",
		"version":           "1.2.3",
		"type":              "Application (macOS)",
		"bundle_identifier": "com.zoobar.overrideme",
		"extension_id":      "",
		"browser":           "",
		"source":            "apps",
		"vendor":            "OverrideMe",
		"last_opened_at":    "0",
		"installed_path":    "/some/override/path",
	}

	appThatOverrides := map[string]string{
		"name":              "OverrideMeSuccess.app",
		"version":           "1.2.3",
		"type":              "Application (macOS)",
		"bundle_identifier": "com.zoobar.overrideme",
		"extension_id":      "",
		"browser":           "",
		"source":            "apps",
		"vendor":            "OverrideMe",
		"last_opened_at":    "0",
		"installed_path":    "/some/override/path",
	}

	for _, tc := range []struct {
		name       string
		host       *fleet.Host
		resultsIn  fleet.OsqueryDistributedQueryResults
		statusesIn map[string]fleet.OsqueryStatus
		messagesIn map[string]string
		overrides  map[string]osquery_utils.DetailQuery

		resultsOut fleet.OsqueryDistributedQueryResults
	}{
		{
			name: "software query works and there are vs code extensions in extra",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "other_detail_query":         fleet.StatusOK,
				hostDetailQueryPrefix + "software_macos":             fleet.StatusOK,
				hostDetailQueryPrefix + "software_vscode_extensions": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
				hostDetailQueryPrefix + "software_vscode_extensions": []map[string]string{
					foobarVSCodeExtension,
					zoobarVSCodeExtension,
				},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
					foobarVSCodeExtension,
					zoobarVSCodeExtension,
				},
			},
		},
		{
			name: "software query and extra works and there are no vscode extensions",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "other_detail_query":         fleet.StatusOK,
				hostDetailQueryPrefix + "software_macos":             fleet.StatusOK,
				hostDetailQueryPrefix + "software_vscode_extensions": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
				hostDetailQueryPrefix + "software_vscode_extensions": []map[string]string{},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
			},
		},
		{
			name: "software query works and the software extra status and results are not returned",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "other_detail_query": fleet.StatusOK,
				hostDetailQueryPrefix + "software_macos":     fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
			},
		},
		{
			name: "software doesn't return status or results but the software extra does",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_vscode_extensions": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_vscode_extensions": []map[string]string{
					foobarVSCodeExtension,
					zoobarVSCodeExtension,
				},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{},
		},
		{
			name: "software query works, but vscode_extensions table doesn't exist",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_macos":             fleet.StatusOK,
				hostDetailQueryPrefix + "software_vscode_extensions": fleet.OsqueryStatus(1),
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
				hostDetailQueryPrefix + "software_vscode_extensions": []map[string]string{},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					zoobarApp,
				},
			},
		},
		{
			name: "software query fails, vscode_extensions table returns results",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_macos":             fleet.OsqueryStatus(1),
				hostDetailQueryPrefix + "software_vscode_extensions": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{},
				hostDetailQueryPrefix + "software_vscode_extensions": []map[string]string{
					foobarVSCodeExtension,
					zoobarVSCodeExtension,
				},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{},
			},
		},
		{
			name: "software query fails, software extra query also fails",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_macos":             fleet.OsqueryStatus(1),
				hostDetailQueryPrefix + "software_vscode_extensions": fleet.OsqueryStatus(1),
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos":             []map[string]string{},
				hostDetailQueryPrefix + "software_vscode_extensions": []map[string]string{},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{},
			},
		},
		{
			name: "software inventory turned off",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "other_detail_query": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "other_detail_query": []map[string]string{
					someRow,
				},
			},
		},
		{
			name: "override works",

			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_macos":      fleet.StatusOK,
				hostDetailQueryPrefix + "software_overrideMe": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					appToOverride,
					foobarApp,
				},
				hostDetailQueryPrefix + "software_overrideMe": []map[string]string{
					appThatOverrides,
				},
			},

			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_macos": []map[string]string{
					foobarApp,
					appThatOverrides,
				},
			},
			overrides: map[string]osquery_utils.DetailQuery{
				"overrideMe": {
					SoftwareOverrideMatch: func(row map[string]string) bool {
						return row["name"] == "OverrideMe.app"
					},
				},
			},
		},
		{
			name: "ubuntu dpkg installed python packages are filtered out",
			host: &fleet.Host{ID: 1, Platform: "ubuntu"},
			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_linux": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_linux": []map[string]string{
					{
						"name":    "python3-twisted",
						"version": "20.3.0-2",
						"source":  "deb_packages",
					},
					{
						"name":    "Twisted", // duplicate of python3-twisted
						"version": "20.3.0-2",
						"source":  "python_packages",
					},
					{
						"name":    "python3-setuptools",
						"version": "50.3.2",
						"source":  "deb_packages",
					},
					{
						"name":    "setuptools",
						"version": "50.3.2",
						"source":  "python_packages",
					},
					{
						"name":    "pillow",
						"version": "8.1.0",
						"source":  "python_packages",
					},
					{
						"name":    "python3-urllib3",
						"version": "1.26.2-2",
						"source":  "deb_packages",
					},
				},
			},
			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_linux": []map[string]string{
					{
						"name":    "python3-twisted",
						"version": "20.3.0-2",
						"source":  "deb_packages",
					},
					{
						"name":    "python3-setuptools",
						"version": "50.3.2",
						"source":  "deb_packages",
					},
					{
						"name":    "python3-pillow", // renamed from pillow
						"version": "8.1.0",
						"source":  "python_packages",
					},
					{
						"name":    "python3-urllib3",
						"version": "1.26.2-2",
						"source":  "deb_packages",
					},
				},
			},
		},
		{
			name: "debian dpkg installed python packages are filtered out",
			host: &fleet.Host{ID: 1, Platform: "debian"},
			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_linux": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_linux": []map[string]string{
					{
						"name":    "python3-twisted",
						"version": "22.4.0-4",
						"source":  "deb_packages",
					},
					{
						"name":    "Twisted", // duplicate of python3-twisted
						"version": "22.4.0-4",
						"source":  "python_packages",
					},
					// known issue below: names don't match so we don't deduplicate
					{
						"name":    "python3-attr", // osquery source column is python-attrs
						"version": "22.2.0-1",
						"source":  "deb_packages",
					},
					{
						"name":    "Attrs",
						"version": "22.2.0",
						"source":  "python_packages",
					},
				},
			},
			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_linux": []map[string]string{
					{
						"name":    "python3-twisted",
						"version": "22.4.0-4",
						"source":  "deb_packages",
					},
					{
						"name":    "python3-attr",
						"version": "22.2.0-1",
						"source":  "deb_packages",
					},
					{
						"name":    "python3-attrs",
						"version": "22.2.0",
						"source":  "python_packages",
					},
				},
			},
		},
		{
			name: "non-ubuntu/debian installed python packages are NOT filtered out",
			host: &fleet.Host{ID: 1, Platform: "rhel"},
			statusesIn: map[string]fleet.OsqueryStatus{
				hostDetailQueryPrefix + "software_linux": fleet.StatusOK,
			},
			resultsIn: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_linux": []map[string]string{
					{
						"name":    "python3-twisted",
						"version": "20.3.0-2",
						"source":  "rpm_packages",
					},
					{
						"name":    "twisted", // duplicate of python3-twisted
						"version": "20.3.0-2",
						"source":  "python_packages",
					},
					{
						"name":    "pillow",
						"version": "8.1.0",
						"source":  "python_packages",
					},
					{
						"name":    "python3-urllib3",
						"version": "1.26.2-2",
						"source":  "rpm_packages",
					},
				},
			},
			resultsOut: fleet.OsqueryDistributedQueryResults{
				hostDetailQueryPrefix + "software_linux": []map[string]string{
					{
						"name":    "python3-twisted",
						"version": "20.3.0-2",
						"source":  "rpm_packages",
					},
					{
						"name":    "twisted", // duplicate of python3-twisted
						"version": "20.3.0-2",
						"source":  "python_packages",
					},
					{
						"name":    "pillow",
						"version": "8.1.0",
						"source":  "python_packages",
					},
					{
						"name":    "python3-urllib3",
						"version": "1.26.2-2",
						"source":  "rpm_packages",
					},
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			host := &fleet.Host{ID: 1}
			if tc.host != nil {
				host = tc.host
			}
			preProcessSoftwareResults(host, &tc.resultsIn, &tc.statusesIn, &tc.messagesIn, tc.overrides, log.NewNopLogger())
			require.Equal(t, tc.resultsOut, tc.resultsIn)
		})
	}
}

func TestDetailQueriesLinuxDistros(t *testing.T) {
	for _, linuxPlatform := range fleet.HostLinuxOSs {
		m := expectedDetailQueriesForPlatform(linuxPlatform)
		require.Contains(t, m, "users")
		require.Contains(t, m, "network_interface_unix")
		require.Contains(t, m, "disk_space_unix")
		require.Contains(t, m, "os_unix_like")
		require.Contains(t, m, "orbit_info")
		require.Contains(t, m, "disk_encryption_linux")
		require.Contains(t, m, "software_vscode_extensions")
		require.Contains(t, m, "software_linux")
	}
}

// Benchmark function
func BenchmarkFindPackDelimiterStringCommon(b *testing.B) {
	// Input data for benchmarking
	input := "pack/Global/Foo"

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findPackDelimiterString(input)
	}
}

func BenchmarkFindPackDelimiterStringTeamPack(b *testing.B) {
	// Input data for benchmarking
	input := "packGlobalGlobalGlobalGlobal" // global pack delimiter, global team, query name global

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findPackDelimiterString(input)
	}
}

func mockUbuntuResults() *fleet.OsqueryDistributedQueryResults {
	results := &fleet.OsqueryDistributedQueryResults{
		hostDetailQueryPrefix + "software_linux": make([]map[string]string, 0),
	}

	// Adding 40 python packages with matching deb packages
	// Adding 2 python packages without matching deb packages
	for i := 1; i <= 42; i++ {
		pythonPkg := fmt.Sprintf("package%d", i)
		(*results)[hostDetailQueryPrefix+"software_linux"] = append((*results)[hostDetailQueryPrefix+"software_linux"], map[string]string{
			"source": "python_packages",
			"name":   pythonPkg,
		})
	}

	// Adding 1500 deb packages, with the first 40 matching python packages
	for i := 1; i <= 1500; i++ {
		var debPkg string
		if i <= 38 { // Match first 38 python packages
			debPkg = fmt.Sprintf("python3-package%d", i)
		} else { // Non-python packages
			debPkg = fmt.Sprintf("unrelated_package%d", i)
		}
		(*results)[hostDetailQueryPrefix+"software_linux"] = append((*results)[hostDetailQueryPrefix+"software_linux"], map[string]string{
			"source": "deb_packages",
			"name":   debPkg,
		})
	}

	return results
}

func BenchmarkPreprocessUbuntuPythonPackageFilter(b *testing.B) {
	platform := "ubuntu"
	results := mockUbuntuResults()
	statuses := &map[string]fleet.OsqueryStatus{
		hostDetailQueryPrefix + "software_linux": fleet.StatusOK,
	}

	for i := 0; i < b.N; i++ {
		preProcessSoftwareResults(&fleet.Host{ID: 1, Platform: platform}, results, statuses, nil, nil, log.NewNopLogger())
	}
}
