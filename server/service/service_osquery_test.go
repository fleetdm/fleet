package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// One of these queries is the disk space, only one of the two works in a platform
var expectedDetailQueries = len(osquery_utils.GetDetailQueries(
	&fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true}})) - 1

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
	ds.EnrollHostFunc = func(ctx context.Context, osqueryHostId, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
		assert.Equal(t, ptr.Uint(3), teamID)
		return &fleet.Host{
			OsqueryHostID: osqueryHostId, NodeKey: nodeKey,
		}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	svc := newTestService(ds, nil, nil)

	nodeKey, err := svc.EnrollAgent(context.Background(), "valid_secret", "host123", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, nodeKey)
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

	svc := newTestService(ds, nil, nil)

	nodeKey, err := svc.EnrollAgent(context.Background(), "not_correct", "host123", nil)
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)
}

func TestEnrollAgentDetails(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{}, nil
	}
	ds.EnrollHostFunc = func(ctx context.Context, osqueryHostId, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
		return &fleet.Host{
			OsqueryHostID: osqueryHostId, NodeKey: nodeKey,
		}, nil
	}
	var gotHost *fleet.Host
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		gotHost = host
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	svc := newTestService(ds, nil, nil)

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
	nodeKey, err := svc.EnrollAgent(context.Background(), "", "host123", details)
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
	svc := newTestService(ds, nil, nil)

	var gotKey string
	host := fleet.Host{ID: 1, Hostname: "foobar"}
	ds.AuthenticateHostFunc = func(ctx context.Context, key string) (*fleet.Host, error) {
		gotKey = key
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

	_, _, err := svc.AuthenticateHost(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "test", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)

	host = fleet.Host{ID: 7, Hostname: "foobar"}
	_, _, err = svc.AuthenticateHost(context.Background(), "floobar")
	require.NoError(t, err)
	assert.Equal(t, "floobar", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)
	// Host checks in twice
	host = fleet.Host{ID: 7, Hostname: "foobar"}
	_, _, err = svc.AuthenticateHost(context.Background(), "floobar")
	require.NoError(t, err)
	assert.Equal(t, "floobar", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)

	err = svc.FlushSeenHosts(context.Background())
	require.NoError(t, err)
	assert.True(t, ds.MarkHostsSeenFuncInvoked)
	assert.ElementsMatch(t, []uint{1, 7}, gotHostIDs)

	err = svc.FlushSeenHosts(context.Background())
	require.NoError(t, err)
	assert.True(t, ds.MarkHostsSeenFuncInvoked)
	require.Len(t, gotHostIDs, 0)
}

func TestAuthenticateHostFailure(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.AuthenticateHostFunc = func(ctx context.Context, key string) (*fleet.Host, error) {
		return nil, errors.New("not found")
	}

	_, _, err := svc.AuthenticateHost(context.Background(), "test")
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
	svc := newTestService(ds, nil, nil)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*Service)

	testLogger := &testJSONLogger{}
	serv.osqueryLogWriter = &logging.OsqueryLogger{Status: testLogger}

	logs := []string{
		`{"severity":"0","filename":"tls.cpp","line":"216","message":"some message","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
		`{"severity":"1","filename":"buffered.cpp","line":"122","message":"warning!","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var status []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &status)
	require.NoError(t, err)

	host := fleet.Host{}
	ctx := hostctx.NewContext(context.Background(), host)
	err = serv.SubmitStatusLogs(ctx, status)
	require.NoError(t, err)

	assert.Equal(t, status, testLogger.logs)
}

func TestSubmitResultLogs(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*Service)

	testLogger := &testJSONLogger{}
	serv.osqueryLogWriter = &logging.OsqueryLogger{Result: testLogger}

	logs := []string{
		`{"name":"system_info","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 17:55:15 2016 UTC","unixTime":"1475258115","decorations":{"host_uuid":"some_uuid","username":"zwass"},"columns":{"cpu_brand":"Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz","hostname":"hostimus","physical_memory":"17179869184"},"action":"added"}`,
		`{"name":"encrypted","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 21:19:15 2016 UTC","unixTime":"1475270355","decorations":{"host_uuid":"4740D59F-699E-5B29-960B-979AAF9BBEEB","username":"zwass"},"columns":{"encrypted":"1","name":"\/dev\/disk1","type":"AES-XTS","uid":"","user_uuid":"","uuid":"some_uuid"},"action":"added"}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"time","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":"1484078931","decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"diffResults":{"removed":[{"address":"127.0.0.1","hostnames":"kl.groob.io"}],"added":""},"name":"pack\/test\/hosts","hostIdentifier":"FA01680E-98CA-5557-8F59-7716ECFEE964","calendarTime":"Sun Nov 19 00:02:08 2017 UTC","unixTime":"1511049728","epoch":"0","counter":"10","decorations":{"host_uuid":"FA01680E-98CA-5557-8F59-7716ECFEE964","hostname":"kl.groob.io"}}`,
		// fleet will accept anything in the "data" field of a log request.
		`{"unknown":{"foo": [] }}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var results []json.RawMessage
	err := json.Unmarshal([]byte(logJSON), &results)
	require.NoError(t, err)

	host := fleet.Host{}
	ctx := hostctx.NewContext(context.Background(), host)
	err = serv.SubmitResultLogs(ctx, results)
	require.NoError(t, err)

	assert.Equal(t, results, testLogger.logs)
}

func TestHostDetailQueries(t *testing.T) {
	ds := new(mock.Store)
	additional := json.RawMessage(`{"foobar": "select foo", "bim": "bam"}`)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{HostSettings: fleet.HostSettings{AdditionalQueries: &additional, EnableHostUsers: true}}, nil
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

		Platform:        "rhel",
		DetailUpdatedAt: mockClock.Now(),
		NodeKey:         "test_key",
		Hostname:        "test_hostname",
		UUID:            "test_uuid",
	}

	svc := &Service{clock: mockClock, config: config.TestConfig(), ds: ds}

	queries, err := svc.detailQueriesForHost(context.Background(), host)
	require.NoError(t, err)
	assert.Empty(t, queries)

	// With refetch requested detail queries should be returned
	host.RefetchRequested = true
	queries, err = svc.detailQueriesForHost(context.Background(), host)
	require.NoError(t, err)
	assert.NotEmpty(t, queries)
	host.RefetchRequested = false

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, err = svc.detailQueriesForHost(context.Background(), host)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+2)
	for name := range queries {
		assert.True(t,
			strings.HasPrefix(name, hostDetailQueryPrefix) || strings.HasPrefix(name, hostAdditionalQueryPrefix),
		)
	}
	assert.Equal(t, "bam", queries[hostAdditionalQueryPrefix+"bim"])
	assert.Equal(t, "select foo", queries[hostAdditionalQueryPrefix+"foobar"])
}

func TestGetDistributedQueriesMissingHost(t *testing.T) {
	svc := newTestService(&mock.Store{}, nil, nil)

	_, _, err := svc.GetDistributedQueries(context.Background())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestLabelQueries(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := new(live_query.MockLiveQuery)
	svc := newTestServiceWithClock(ds, nil, lq, mockClock)

	host := &fleet.Host{
		Platform: "darwin",
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostFunc = func(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, gotHost *fleet.Host) error {
		host = gotHost
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true}}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	lq.On("QueriesForHost", uint(0)).Return(map[string]string{}, nil)

	ctx := hostctx.NewContext(context.Background(), *host)

	// With a new host, we should get the detail queries (and accelerate
	// should be turned on so that we can quickly fill labels)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries)
	assert.NotZero(t, acc)

	// Simulate the detail queries being added.
	host.DetailUpdatedAt = mockClock.Now().Add(-1 * time.Minute)
	host.Hostname = "zwass.local"
	ctx = hostctx.NewContext(ctx, *host)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 0)
	assert.Zero(t, acc)

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{
			"label1": "query1",
			"label2": "query2",
			"label3": "query3",
		}, nil
	}

	// Now we should get the label queries
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 3)
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
	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 0)
	assert.Zero(t, acc)

	// With refetch requested details+label queries should be returned.
	host.RefetchRequested = true
	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+3)
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
	ctx = hostctx.NewContext(context.Background(), *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 0)
	assert.Zero(t, acc)
}

func TestGetClientConfig(t *testing.T) {
	ds := new(mock.Store)
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, pid uint, opt fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
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
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":{"options":{"baz":"bar"}}}`))}, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}

	svc := newTestService(ds, nil, nil)

	ctx1 := hostctx.NewContext(context.Background(), fleet.Host{ID: 1})
	ctx2 := hostctx.NewContext(context.Background(), fleet.Host{ID: 2})

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
}

func TestDetailQueriesWithEmptyStrings(t *testing.T) {
	ds := new(mock.Store)
	mockClock := clock.NewMockClock()
	lq := new(live_query.MockLiveQuery)
	svc := newTestServiceWithClock(ds, nil, lq, mockClock)

	host := fleet.Host{Platform: "windows"}
	ctx := hostctx.NewContext(context.Background(), host)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true}}, nil
	}
	ds.LabelQueriesForHostFunc = func(context.Context, *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries)
	assert.NotZero(t, acc)

	resultJSON := `
{
"fleet_detail_query_network_interface": [
		{
				"address": "192.168.0.1",
				"broadcast": "192.168.0.255",
				"ibytes": "",
				"ierrors": "",
				"interface": "en0",
				"ipackets": "25698094",
				"last_change": "1474233476",
				"mac": "5f:3d:4b:10:25:82",
				"mask": "255.255.255.0",
				"metric": "",
				"mtu": "",
				"obytes": "",
				"oerrors": "",
				"opackets": "",
				"point_to_point": "",
				"type": ""
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
				"hardware_serial": "ABCDEFGH",
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
			"value":""
		},
		{
			"name":"distributed_interval",
			"value":""
		},
		{
			"name":"logger_tls_period",
			"value":""
		}
]
}
`

	var results fleet.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.NoError(t, err)

	var gotHost *fleet.Host
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		gotHost = host
		gotHost.Additional = host.Additional
		return nil
	}

	ds.HostFunc = func(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
		return &host, nil
	}

	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results, map[string]fleet.OsqueryStatus{}, map[string]string{})

	// osquery_info
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "1.8.2", gotHost.OsqueryVersion)

	// system_info
	assert.Equal(t, int64(17179869184), gotHost.Memory)
	assert.Equal(t, "computer.local", gotHost.Hostname)
	assert.Equal(t, "uuid", gotHost.UUID)

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
	ctx = hostctx.NewContext(context.Background(), host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 0)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries)
	assert.Zero(t, acc)
}

func TestDetailQueries(t *testing.T) {
	ds := new(mock.Store)
	mockClock := clock.NewMockClock()
	lq := new(live_query.MockLiveQuery)
	svc := newTestServiceWithClock(ds, nil, lq, mockClock)

	host := fleet.Host{Platform: "linux"}
	ctx := hostctx.NewContext(context.Background(), host)

	lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true, EnableSoftwareInventory: true}}, nil
	}
	ds.LabelQueriesForHostFunc = func(context.Context, *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+1)
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
        "hardware_serial": "ABCDEFGH",
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
		"gigs_disk_space_available": "277.0"
	}
]
}
`

	var results fleet.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.NoError(t, err)

	var gotHost *fleet.Host
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		gotHost = host
		gotHost.Additional = host.Additional
		return nil
	}

	ds.HostFunc = func(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
		return &host, nil
	}

	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results, map[string]fleet.OsqueryStatus{}, map[string]string{})

	// osquery_info
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "1.8.2", gotHost.OsqueryVersion)

	// system_info
	assert.Equal(t, int64(17179869184), gotHost.Memory)
	assert.Equal(t, "computer.local", gotHost.Hostname)
	assert.Equal(t, "uuid", gotHost.UUID)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", gotHost.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, gotHost.Uptime)

	// osquery_flags
	assert.Equal(t, uint(10), gotHost.ConfigTLSRefresh)
	assert.Equal(t, uint(5), gotHost.DistributedInterval)
	assert.Equal(t, uint(60), gotHost.LoggerTLSPeriod)

	// users
	require.Len(t, gotHost.Users, 2)
	assert.Equal(t, fleet.HostUser{
		Uid:       1234,
		Username:  "user1",
		Type:      "sometype",
		GroupName: "somegroup",
		Shell:     "someloginshell",
	}, gotHost.Users[0])
	assert.Equal(t, fleet.HostUser{
		Uid:       5678,
		Username:  "user2",
		Type:      "sometype",
		GroupName: "somegroup",
		Shell:     "",
	}, gotHost.Users[1])

	// software
	require.Len(t, gotHost.HostSoftware.Software, 2)
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
	}, gotHost.HostSoftware.Software)

	assert.Equal(t, 56.0, gotHost.PercentDiskSpaceAvailable)
	assert.Equal(t, 277.0, gotHost.GigsDiskSpaceAvailable)

	host.Hostname = "computer.local"
	host.Platform = "darwin"
	host.DetailUpdatedAt = mockClock.Now()
	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	ctx = hostctx.NewContext(ctx, host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, 0)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+1)
	assert.Zero(t, acc)
}

func TestNewDistributedQueryCampaign(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	rs := &mock.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := &live_query.MockLiveQuery{}
	mockClock := clock.NewMockClock()
	svc := newTestServiceWithClock(ds, rs, lq, mockClock)

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
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
	viewerCtx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{
			ID:         0,
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	})
	q := "select year, month, day, hour, minutes, seconds from time"
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}
	campaign, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)
	assert.Equal(t, gotQuery.ID, gotCampaign.QueryID)
	assert.True(t, ds.NewActivityFuncInvoked)
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
	lq := new(live_query.MockLiveQuery)
	svc := newTestServiceWithClock(ds, rs, lq, mockClock)

	campaign := &fleet.DistributedQueryCampaign{ID: 42}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.PolicyQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true}}, nil
	}

	host := &fleet.Host{ID: 1, Platform: "windows"}
	hostCtx := hostctx.NewContext(context.Background(), *host)

	lq.On("QueriesForHost", uint(1)).Return(
		map[string]string{
			strconv.Itoa(int(campaign.ID)): "select * from time",
		},
		nil,
	)
	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(nil)

	// Now we should get the active distributed query
	queries, acc, err := svc.GetDistributedQueries(hostCtx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+1)
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
				assert.Equal(t, *host, res.Host)
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

	err = svc.SubmitDistributedQueryResults(hostCtx, results, map[string]fleet.OsqueryStatus{}, map[string]string{})
	require.NoError(t, err)
}

func TestIngestDistributedQueryParseIdError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	host := fleet.Host{ID: 1}
	err := svc.ingestDistributedQuery(context.Background(), host, "bad_name", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse campaign")
}

func TestIngestDistributedQueryOrphanedCampaignLoadError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
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

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading orphaned campaign")
}

func TestIngestDistributedQueryOrphanedCampaignWaitListener(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
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

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign waiting for listener")
}

func TestIngestDistributedQueryOrphanedCloseError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
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

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing orphaned campaign")
}

func TestIngestDistributedQueryOrphanedStopError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
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
	lq.On("StopQuery", strconv.Itoa(int(campaign.ID))).Return(errors.New("failed"))

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopping orphaned campaign")
}

func TestIngestDistributedQueryOrphanedStop(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
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
	lq.On("StopQuery", strconv.Itoa(int(campaign.ID))).Return(nil)

	host := fleet.Host{ID: 1}

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign stopped")
	lq.AssertExpectations(t)
}

func TestIngestDistributedQueryRecordCompletionError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{ID: 42}
	host := fleet.Host{ID: 1}

	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(errors.New("fail"))

	go func() {
		ch, err := rs.ReadChannel(context.Background(), *campaign)
		require.NoError(t, err)
		<-ch
	}()
	time.Sleep(10 * time.Millisecond)

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "record query completion")
	lq.AssertExpectations(t)
}

func TestIngestDistributedQuery(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := &Service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &fleet.DistributedQueryCampaign{ID: 42}
	host := fleet.Host{ID: 1}

	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(nil)

	go func() {
		ch, err := rs.ReadChannel(context.Background(), *campaign)
		require.NoError(t, err)
		<-ch
	}()
	time.Sleep(10 * time.Millisecond)

	err := svc.ingestDistributedQuery(context.Background(), host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.NoError(t, err)
	lq.AssertExpectations(t)
}

func TestUpdateHostIntervals(t *testing.T) {
	ds := new(mock.Store)

	svc := newTestService(ds, nil, nil)

	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{}, nil
	}

	testCases := []struct {
		initHost       fleet.Host
		finalHost      fleet.Host
		configOptions  json.RawMessage
		saveHostCalled bool
	}{
		// Both updated
		{
			fleet.Host{
				ConfigTLSRefresh: 60,
			},
			fleet.Host{
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
		// Only logger_tls_period updated
		{
			fleet.Host{
				DistributedInterval: 11,
				ConfigTLSRefresh:    60,
			},
			fleet.Host{
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
		// Only distributed_interval updated
		{
			fleet.Host{
				ConfigTLSRefresh: 60,
				LoggerTLSPeriod:  33,
			},
			fleet.Host{
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
		// Fleet not managing distributed_interval
		{
			fleet.Host{
				ConfigTLSRefresh:    60,
				DistributedInterval: 11,
			},
			fleet.Host{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			json.RawMessage(`{"options":{
				"logger_tls_period": 33
			}}`),
			true,
		},
		// config_refresh should also cause an update
		{
			fleet.Host{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			fleet.Host{
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
		// SaveHost should not be called with no changes
		{
			fleet.Host{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			fleet.Host{
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
		t.Run("", func(t *testing.T) {
			ctx := hostctx.NewContext(context.Background(), tt.initHost)

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":` + string(tt.configOptions) + `}`))}, nil
			}

			saveHostCalled := false
			ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
				saveHostCalled = true
				assert.Equal(t, tt.finalHost, *host)
				return nil
			}

			_, err := svc.GetClientConfig(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.saveHostCalled, saveHostCalled)
		})
	}
}

type notFoundError struct{}

func (e notFoundError) Error() string {
	return "not found"
}

func (e notFoundError) IsNotFound() bool {
	return true
}

func TestAuthenticationErrors(t *testing.T) {
	ms := new(mock.Store)
	ms.AuthenticateHostFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, nil
	}

	svc := newTestService(ms, nil, nil)
	ctx := context.Background()

	_, _, err := svc.AuthenticateHost(ctx, "")
	require.Error(t, err)
	require.True(t, err.(osqueryError).NodeInvalid())

	ms.AuthenticateHostFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return &fleet.Host{ID: 1}, nil
	}
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	_, _, err = svc.AuthenticateHost(ctx, "foo")
	require.NoError(t, err)

	// return not found error
	ms.AuthenticateHostFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, notFoundError{}
	}

	_, _, err = svc.AuthenticateHost(ctx, "foo")
	require.Error(t, err)
	require.True(t, err.(osqueryError).NodeInvalid())

	// return other error
	ms.AuthenticateHostFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		return nil, errors.New("foo")
	}

	_, _, err = svc.AuthenticateHost(ctx, "foo")
	require.NotNil(t, err)
	require.False(t, err.(osqueryError).NodeInvalid())
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
	svc := newTestService(ds, nil, nil)

	host := &fleet.Host{Platform: "darwin"}

	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return authz.CheckMissingWithResponse(nil)
	}
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferred bool) error {
		return errors.New("something went wrong")
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	lCtx := &fleetLogging.LoggingContext{}
	ctx := fleetLogging.NewContext(context.Background(), lCtx)
	ctx = hostctx.NewContext(ctx, *host)

	err := svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "1":      {{"col1": "val1"}},
			hostAdditionalQueryPrefix + "1": {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
	)
	require.NoError(t, err)

	lCtx.Log(ctx, logger)

	logs := buf.String()
	parts := strings.Split(strings.TrimSpace(logs), "\n")
	require.Len(t, parts, 1)
	logData := make(map[string]json.RawMessage)
	err = json.Unmarshal([]byte(parts[0]), &logData)
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`"something went wrong"`), logData["err"])
	assert.Equal(t, json.RawMessage(`"Missing authorization check"`), logData["internal"])
}

func TestDistributedQueriesReloadsHostIfDetailsAreIn(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	host := &fleet.Host{ID: 42, Platform: "darwin"}
	ip := "1.1.1.1"

	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		assert.Equal(t, ip, host.PrimaryIP)
		return nil
	}
	ds.HostFunc = func(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
		require.Equal(t, uint(42), id)
		return &fleet.Host{ID: 42, Platform: "darwin", PrimaryIP: ip}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ctx := hostctx.NewContext(context.Background(), *host)

	err := svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostDetailQueryPrefix + "1": {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{},
		map[string]string{},
	)
	require.NoError(t, err)
	assert.True(t, ds.HostFuncInvoked)
}

func TestObserversCanOnlyRunDistributedCampaigns(t *testing.T) {
	ds := new(mock.Store)
	rs := &mock.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := &live_query.MockLiveQuery{}
	mockClock := clock.NewMockClock()
	svc := newTestServiceWithClock(ds, rs, lq, mockClock)

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
	viewerCtx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{ID: 0, GlobalRole: ptr.String(fleet.RoleObserver)},
	})

	q := "select year, month, day, hour, minutes, seconds from time"
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
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
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error { return nil }
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
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}
	lq.On("RunQuery", "21", "select 1;", []uint{1, 3, 5}).Return(nil)
	_, err = svc.NewDistributedQueryCampaign(viewerCtx, "", ptr.Uint(42), fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)
}

func TestTeamMaintainerCanRunNewDistributedCampaigns(t *testing.T) {
	ds := new(mock.Store)
	rs := &mock.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := &live_query.MockLiveQuery{}
	mockClock := clock.NewMockClock()
	svc := newTestServiceWithClock(ds, rs, lq, mockClock)

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
	viewerCtx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{ID: 0, Teams: []fleet.UserTeam{{Role: fleet.RoleMaintainer}}},
	})

	q := "select year, month, day, hour, minutes, seconds from time"
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
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
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}
	lq.On("RunQuery", "0", "select year, month, day, hour, minutes, seconds from time", []uint{1, 3, 5}).Return(nil)
	_, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)
}

func TestPolicyQueries(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := new(live_query.MockLiveQuery)
	svc := newTestServiceWithClock(ds, nil, lq, mockClock)

	host := &fleet.Host{
		Platform: "darwin",
	}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.HostFunc = func(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, gotHost *fleet.Host) error {
		host = gotHost
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{HostSettings: fleet.HostSettings{EnableHostUsers: true}}, nil
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

	ctx := hostctx.NewContext(context.Background(), *host)

	queries, _, err := svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+2)

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
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1,
		},
		map[string]string{},
	)
	require.NoError(t, err)
	require.NotNil(t, recordedResults[1])
	require.Equal(t, true, *recordedResults[1])
	require.Nil(t, recordedResults[2])

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
	ctx = hostctx.NewContext(context.Background(), *host)
	queries, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries)
	noPolicyResults(queries)

	// Let's move time forward, there should be policies now.
	mockClock.AddTime(2 * time.Hour)

	queries, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+2)
	checkPolicyResults(queries)

	// Record another query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1,
		},
		map[string]string{},
	)
	require.NoError(t, err)
	require.NotNil(t, recordedResults[1])
	require.Equal(t, true, *recordedResults[1])
	require.Nil(t, recordedResults[2])

	// There shouldn't be any policies now.
	ctx = hostctx.NewContext(context.Background(), *host)
	queries, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries)
	noPolicyResults(queries)

	// With refetch requested policy queries should be returned.
	host.RefetchRequested = true
	ctx = hostctx.NewContext(context.Background(), *host)
	queries, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries+2)
	checkPolicyResults(queries)

	// Record another query execution.
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostPolicyQueryPrefix + "1": {{"col1": "val1"}},
		},
		map[string]fleet.OsqueryStatus{
			hostPolicyQueryPrefix + "2": 1,
		},
		map[string]string{},
	)
	require.NoError(t, err)
	require.NotNil(t, recordedResults[1])
	require.Equal(t, true, *recordedResults[1])
	require.Nil(t, recordedResults[2])

	// SubmitDistributedQueryResults will set RefetchRequested to false.
	require.False(t, host.RefetchRequested)

	// There shouldn't be any policies now.
	ctx = hostctx.NewContext(context.Background(), *host)
	queries, _, err = svc.GetDistributedQueries(ctx)
	require.NoError(t, err)
	require.Len(t, queries, expectedDetailQueries)
	noPolicyResults(queries)
}
