package service

import (
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
	"github.com/fleetdm/fleet/server/config"
	hostctx "github.com/fleetdm/fleet/server/contexts/host"
	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/live_query"
	"github.com/fleetdm/fleet/server/logging"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/fleetdm/fleet/server/pubsub"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollAgent(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(secret string) (string, error) {
		switch secret {
		case "valid_secret":
			return "valid", nil
		default:
			return "", errors.New("not found")
		}
	}
	ds.EnrollHostFunc = func(osqueryHostId, nodeKey, secretName string, cooldown time.Duration) (*kolide.Host, error) {
		return &kolide.Host{
			OsqueryHostID: osqueryHostId, NodeKey: nodeKey, EnrollSecretName: secretName,
		}, nil
	}

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	nodeKey, err := svc.EnrollAgent(context.Background(), "valid_secret", "host123", nil)
	require.Nil(t, err)
	assert.NotEmpty(t, nodeKey)
}

func TestEnrollAgentIncorrectEnrollSecret(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(secret string) (string, error) {
		switch secret {
		case "valid_secret":
			return "valid", nil
		default:
			return "", errors.New("not found")
		}
	}

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	nodeKey, err := svc.EnrollAgent(context.Background(), "not_correct", "host123", nil)
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)
}

func TestEnrollAgentDetails(t *testing.T) {
	ds := new(mock.Store)
	ds.VerifyEnrollSecretFunc = func(secret string) (string, error) {
		return "valid", nil
	}
	ds.EnrollHostFunc = func(osqueryHostId, nodeKey, secretName string, cooldown time.Duration) (*kolide.Host, error) {
		return &kolide.Host{
			OsqueryHostID: osqueryHostId, NodeKey: nodeKey, EnrollSecretName: secretName,
		}, nil
	}
	var gotHost *kolide.Host
	ds.SaveHostFunc = func(host *kolide.Host) error {
		gotHost = host
		return nil
	}

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

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
	require.Nil(t, err)
	assert.NotEmpty(t, nodeKey)

	assert.Equal(t, "Mac OS X 10.14.5", gotHost.OSVersion)
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "2.12.0", gotHost.OsqueryVersion)
	assert.Equal(t, "zwass.local", gotHost.HostName)
	assert.Equal(t, "froobling_uuid", gotHost.UUID)
	assert.Equal(t, "valid", gotHost.EnrollSecretName)
}

func TestAuthenticateHost(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.NoError(t, err)

	var gotKey string
	host := kolide.Host{ID: 1, HostName: "foobar"}
	ds.AuthenticateHostFunc = func(key string) (*kolide.Host, error) {
		gotKey = key
		return &host, nil
	}
	var gotHostIDs []uint
	ds.MarkHostsSeenFunc = func(hostIDs []uint, t time.Time) error {
		gotHostIDs = hostIDs
		return nil
	}

	_, err = svc.AuthenticateHost(context.Background(), "test")
	require.Nil(t, err)
	assert.Equal(t, "test", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)

	host = kolide.Host{ID: 7, HostName: "foobar"}
	_, err = svc.AuthenticateHost(context.Background(), "floobar")
	require.Nil(t, err)
	assert.Equal(t, "floobar", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)
	// Host checks in twice
	host = kolide.Host{ID: 7, HostName: "foobar"}
	_, err = svc.AuthenticateHost(context.Background(), "floobar")
	require.Nil(t, err)
	assert.Equal(t, "floobar", gotKey)
	assert.False(t, ds.MarkHostsSeenFuncInvoked)

	err = svc.FlushSeenHosts(context.Background())
	require.NoError(t, err)
	assert.True(t, ds.MarkHostsSeenFuncInvoked)
	assert.ElementsMatch(t, []uint{1, 7}, gotHostIDs)

	err = svc.FlushSeenHosts(context.Background())
	require.NoError(t, err)
	assert.True(t, ds.MarkHostsSeenFuncInvoked)
	assert.Len(t, gotHostIDs, 0)
}

func TestAuthenticateHostFailure(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ds.AuthenticateHostFunc = func(key string) (*kolide.Host, error) {
		return nil, errors.New("not found")
	}

	_, err = svc.AuthenticateHost(context.Background(), "test")
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
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*service)

	testLogger := &testJSONLogger{}
	serv.osqueryLogWriter = &logging.OsqueryLogger{Status: testLogger}

	logs := []string{
		`{"severity":"0","filename":"tls.cpp","line":"216","message":"some message","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
		`{"severity":"1","filename":"buffered.cpp","line":"122","message":"warning!","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var status []json.RawMessage
	err = json.Unmarshal([]byte(logJSON), &status)
	require.Nil(t, err)

	host := kolide.Host{}
	ctx := hostctx.NewContext(context.Background(), host)
	err = serv.SubmitStatusLogs(ctx, status)
	assert.Nil(t, err)

	assert.Equal(t, status, testLogger.logs)
}

func TestSubmitResultLogs(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(*service)

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
	err = json.Unmarshal([]byte(logJSON), &results)
	require.Nil(t, err)

	host := kolide.Host{}
	ctx := hostctx.NewContext(context.Background(), host)
	err = serv.SubmitResultLogs(ctx, results)
	assert.Nil(t, err)

	assert.Equal(t, results, testLogger.logs)
}

func TestHostDetailQueries(t *testing.T) {
	ds := new(mock.Store)
	additional := json.RawMessage(`{"foobar": "select foo", "bim": "bam"}`)
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{AdditionalQueries: &additional}, nil
	}

	mockClock := clock.NewMockClock()
	host := kolide.Host{
		ID: 1,
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			UpdateTimestamp: kolide.UpdateTimestamp{
				UpdatedAt: mockClock.Now(),
			},
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: mockClock.Now(),
			},
		},

		DetailUpdateTime: mockClock.Now(),
		NodeKey:          "test_key",
		HostName:         "test_hostname",
		UUID:             "test_uuid",
	}

	svc := service{clock: mockClock, config: config.TestConfig(), ds: ds}

	queries, err := svc.hostDetailQueries(host)
	assert.Nil(t, err)
	assert.Empty(t, queries, 0)

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, err = svc.hostDetailQueries(host)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries)+2)
	for name, _ := range queries {
		assert.True(t,
			strings.HasPrefix(name, hostDetailQueryPrefix) || strings.HasPrefix(name, hostAdditionalQueryPrefix),
		)
	}
	assert.Equal(t, "bam", queries[hostAdditionalQueryPrefix+"bim"])
	assert.Equal(t, "select foo", queries[hostAdditionalQueryPrefix+"foobar"])
}

func TestGetDistributedQueriesMissingHost(t *testing.T) {
	svc, err := newTestService(&mock.Store{}, nil, nil)
	require.Nil(t, err)

	_, _, err = svc.GetDistributedQueries(context.Background())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestLabelQueries(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := new(live_query.MockLiveQuery)
	svc, err := newTestServiceWithClock(ds, nil, lq, mockClock)
	require.Nil(t, err)

	host := &kolide.Host{}

	ds.LabelQueriesForHostFunc = func(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.DistributedQueriesForHostFunc = func(host *kolide.Host) (map[uint]string, error) {
		return map[uint]string{}, nil
	}
	ds.HostFunc = func(id uint) (*kolide.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(host *kolide.Host) error {
		return nil
	}
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}

	lq.On("QueriesForHost", uint(0)).Return(map[string]string{}, nil)

	ctx := hostctx.NewContext(context.Background(), *host)

	// With a new host, we should get the detail queries (and accelerate
	// should be turned on so that we can quickly fill labels)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.NotZero(t, acc)

	// Simulate the detail queries being added
	host.DetailUpdateTime = mockClock.Now().Add(-1 * time.Minute)
	host.Platform = "darwin"
	host.HostName = "zwass.local"
	ctx = hostctx.NewContext(ctx, *host)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Zero(t, acc)

	ds.LabelQueriesForHostFunc = func(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
		return map[string]string{
			"label1": "query1",
			"label2": "query2",
			"label3": "query3",
		}, nil
	}

	// Now we should get the label queries
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)
	assert.Zero(t, acc)

	var gotHost *kolide.Host
	var gotResults map[uint]bool
	var gotTime time.Time
	ds.RecordLabelQueryExecutionsFunc = func(host *kolide.Host, results map[uint]bool, t time.Time) error {
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
		map[string]kolide.OsqueryStatus{},
		map[string]string{},
	)
	assert.Nil(t, err)
	host.LabelUpdateTime = mockClock.Now()
	assert.Equal(t, host, gotHost)
	assert.Equal(t, mockClock.Now(), gotTime)
	if assert.Len(t, gotResults, 1) {
		assert.Equal(t, true, gotResults[1])
	}

	mockClock.AddTime(1 * time.Second)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "2": {{"col1": "val1"}},
			hostLabelQueryPrefix + "3": {},
		},
		map[string]kolide.OsqueryStatus{},
		map[string]string{},
	)
	assert.Nil(t, err)
	host.LabelUpdateTime = mockClock.Now()
	assert.Equal(t, host, gotHost)
	assert.Equal(t, mockClock.Now(), gotTime)
	if assert.Len(t, gotResults, 2) {
		assert.Equal(t, true, gotResults[2])
		assert.Equal(t, false, gotResults[3])
	}
}

func TestGetClientConfig(t *testing.T) {
	ds := new(mock.Store)
	ds.ListPacksForHostFunc = func(hid uint) ([]*kolide.Pack, error) {
		return []*kolide.Pack{}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(pid uint, opt kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
		tru := true
		fals := false
		fortytwo := uint(42)
		switch pid {
		case 1:
			return []*kolide.ScheduledQuery{
				{Name: "time", Query: "select * from time", Interval: 30, Removed: &fals},
			}, nil
		case 4:
			return []*kolide.ScheduledQuery{
				{Name: "foobar", Query: "select 3", Interval: 20, Shard: &fortytwo},
				{Name: "froobing", Query: "select 'guacamole'", Interval: 60, Snapshot: &tru},
			}, nil
		default:
			return []*kolide.ScheduledQuery{}, nil
		}
	}
	ds.OptionsForPlatformFunc = func(platform string) (json.RawMessage, error) {
		return json.RawMessage(`
{
  "options":{
    "distributed_interval":11,
    "logger_tls_period":33
  },
  "decorators":{
    "load":[
      "SELECT version FROM osquery_info;",
      "SELECT uuid AS host_uuid FROM system_info;"
    ],
    "always":[
      "SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;"
    ],
    "interval":{
      "3600":[
        "SELECT total_seconds AS uptime FROM uptime;"
      ]
    }
  },
  "foo": "bar"
}
`), nil
	}
	ds.SaveHostFunc = func(host *kolide.Host) error {
		return nil
	}

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ctx1 := hostctx.NewContext(context.Background(), kolide.Host{ID: 1})
	ctx2 := hostctx.NewContext(context.Background(), kolide.Host{ID: 2})

	expectedOptions := map[string]interface{}{
		"distributed_interval": float64(11),
		"logger_tls_period":    float64(33),
	}

	expectedDecorators := map[string]interface{}{
		"load": []interface{}{
			"SELECT version FROM osquery_info;",
			"SELECT uuid AS host_uuid FROM system_info;",
		},
		"always": []interface{}{
			"SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1;",
		},
		"interval": map[string]interface{}{
			"3600": []interface{}{"SELECT total_seconds AS uptime FROM uptime;"},
		},
	}

	expectedConfig := map[string]interface{}{
		"options":    expectedOptions,
		"decorators": expectedDecorators,
		"foo":        "bar",
	}

	// No packs loaded yet
	conf, err := svc.GetClientConfig(ctx1)
	require.Nil(t, err)
	assert.Equal(t, expectedConfig, conf)

	conf, err = svc.GetClientConfig(ctx2)
	require.Nil(t, err)
	assert.Equal(t, expectedConfig, conf)

	// Now add packs
	ds.ListPacksForHostFunc = func(hid uint) ([]*kolide.Pack, error) {
		switch hid {
		case 1:
			return []*kolide.Pack{
				{ID: 1, Name: "pack_by_label"},
				{ID: 4, Name: "pack_by_other_label"},
			}, nil

		case 2:
			return []*kolide.Pack{
				{ID: 1, Name: "pack_by_label"},
			}, nil
		}
		return []*kolide.Pack{}, nil
	}

	conf, err = svc.GetClientConfig(ctx1)
	require.Nil(t, err)
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
	require.Nil(t, err)
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
	svc, err := newTestServiceWithClock(ds, nil, lq, mockClock)
	require.Nil(t, err)

	host := kolide.Host{}
	ctx := hostctx.NewContext(context.Background(), host)

	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}
	ds.LabelQueriesForHostFunc = func(*kolide.Host, time.Time) (map[string]string, error) {
		return map[string]string{}, nil
	}

	lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
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

	var results kolide.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.Nil(t, err)

	var gotHost *kolide.Host
	ds.SaveHostFunc = func(host *kolide.Host) error {
		gotHost = host
		return nil
	}

	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results, map[string]kolide.OsqueryStatus{}, map[string]string{})

	// osquery_info
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "1.8.2", gotHost.OsqueryVersion)

	// system_info
	assert.Equal(t, int64(17179869184), gotHost.PhysicalMemory)
	assert.Equal(t, "computer.local", gotHost.HostName)
	assert.Equal(t, "uuid", gotHost.UUID)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", gotHost.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, gotHost.Uptime)

	// osquery_flags
	assert.Equal(t, uint(0), gotHost.ConfigTLSRefresh)
	assert.Equal(t, uint(0), gotHost.DistributedInterval)
	assert.Equal(t, uint(0), gotHost.LoggerTLSPeriod)

	host.HostName = "computer.local"
	host.DetailUpdateTime = mockClock.Now()
	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	ctx = hostctx.NewContext(context.Background(), host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.Zero(t, acc)
}

func TestDetailQueries(t *testing.T) {
	ds := new(mock.Store)
	mockClock := clock.NewMockClock()
	lq := new(live_query.MockLiveQuery)
	svc, err := newTestServiceWithClock(ds, nil, lq, mockClock)
	require.Nil(t, err)

	host := kolide.Host{}
	ctx := hostctx.NewContext(context.Background(), host)

	lq.On("QueriesForHost", host.ID).Return(map[string]string{}, nil)

	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}
	ds.LabelQueriesForHostFunc = func(*kolide.Host, time.Time) (map[string]string, error) {
		return map[string]string{}, nil
	}

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
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
]
}
`

	var results kolide.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.Nil(t, err)

	var gotHost *kolide.Host
	ds.SaveHostFunc = func(host *kolide.Host) error {
		gotHost = host
		return nil
	}
	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results, map[string]kolide.OsqueryStatus{}, map[string]string{})

	// osquery_info
	assert.Equal(t, "darwin", gotHost.Platform)
	assert.Equal(t, "1.8.2", gotHost.OsqueryVersion)

	// system_info
	assert.Equal(t, int64(17179869184), gotHost.PhysicalMemory)
	assert.Equal(t, "computer.local", gotHost.HostName)
	assert.Equal(t, "uuid", gotHost.UUID)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", gotHost.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, gotHost.Uptime)

	// osquery_flags
	assert.Equal(t, uint(10), gotHost.ConfigTLSRefresh)
	assert.Equal(t, uint(5), gotHost.DistributedInterval)
	assert.Equal(t, uint(60), gotHost.LoggerTLSPeriod)

	host.HostName = "computer.local"
	host.DetailUpdateTime = mockClock.Now()
	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	ctx = hostctx.NewContext(ctx, host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.Zero(t, acc)
}

func TestDetailQueryNetworkInterfaces(t *testing.T) {
	var initialHost kolide.Host
	host := initialHost

	ingest := detailQueries["network_interface"].IngestFunc

	assert.NoError(t, ingest(log.NewNopLogger(), &host, nil))
	assert.Equal(t, initialHost, host)

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"192.168.1.3","mac":"f4:5d:79:93:58:5b"},
  {"address":"fe80::241a:9aff:fe60:d80a%awdl0","mac":"27:1b:aa:60:e8:0a"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "192.168.1.3", host.PrimaryIP)
	assert.Equal(t, "f4:5d:79:93:58:5b", host.PrimaryMac)

	// Only IPv6
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"2604:3f08:1337:9411:cbe:814f:51a6:e4e3","mac":"27:1b:aa:60:e8:0a"},
  {"address":"3333:3f08:1337:9411:cbe:814f:51a6:e4e3","mac":"bb:1b:aa:60:e8:bb"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "2604:3f08:1337:9411:cbe:814f:51a6:e4e3", host.PrimaryIP)
	assert.Equal(t, "27:1b:aa:60:e8:0a", host.PrimaryMac)

	// IPv6 appears before IPv4 (v4 should be prioritized)
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"2604:3f08:1337:9411:cbe:814f:51a6:e4e3","mac":"27:1b:aa:60:e8:0a"},
  {"address":"205.111.43.79","mac":"ab:1b:aa:60:e8:0a"},
  {"address":"205.111.44.80","mac":"bb:bb:aa:60:e8:0a"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "205.111.43.79", host.PrimaryIP)
	assert.Equal(t, "ab:1b:aa:60:e8:0a", host.PrimaryMac)

	// Only link-local/loopback
	require.NoError(t, json.Unmarshal([]byte(`
[
  {"address":"127.0.0.1","mac":"00:00:00:00:00:00"},
  {"address":"::1","mac":"00:00:00:00:00:00"},
  {"address":"fe80::1%lo0","mac":"00:00:00:00:00:00"},
  {"address":"fe80::df:429b:971c:d051%en0","mac":"f4:5c:89:92:57:5b"},
  {"address":"fe80::241a:9aff:fe60:d80a%awdl0","mac":"27:1b:aa:60:e8:0a"},
  {"address":"fe80::3a6f:582f:86c5:8296%utun0","mac":"00:00:00:00:00:00"}
]`),
		&rows,
	))

	assert.NoError(t, ingest(log.NewNopLogger(), &host, rows))
	assert.Equal(t, "127.0.0.1", host.PrimaryIP)
	assert.Equal(t, "00:00:00:00:00:00", host.PrimaryMac)
}

func TestNewDistributedQueryCampaign(t *testing.T) {
	ds := &mock.Store{
		AppConfigStore: mock.AppConfigStore{
			AppConfigFunc: func() (*kolide.AppConfig, error) {
				config := &kolide.AppConfig{}
				return config, nil
			},
		},
	}
	rs := &mock.QueryResultStore{
		HealthCheckFunc: func() error {
			return nil
		},
	}
	lq := &live_query.MockLiveQuery{}
	mockClock := clock.NewMockClock()
	svc, err := newTestServiceWithClock(ds, rs, lq, mockClock)
	require.Nil(t, err)

	ds.LabelQueriesForHostFunc = func(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.DistributedQueriesForHostFunc = func(host *kolide.Host) (map[uint]string, error) {
		return map[uint]string{}, nil
	}
	ds.SaveHostFunc = func(host *kolide.Host) error {
		return nil
	}
	var gotQuery *kolide.Query
	ds.NewQueryFunc = func(query *kolide.Query, opts ...kolide.OptionalArg) (*kolide.Query, error) {
		gotQuery = query
		query.ID = 42
		return query, nil
	}
	var gotCampaign *kolide.DistributedQueryCampaign
	ds.NewDistributedQueryCampaignFunc = func(camp *kolide.DistributedQueryCampaign) (*kolide.DistributedQueryCampaign, error) {
		gotCampaign = camp
		camp.ID = 21
		return camp, nil
	}
	var gotTargets []*kolide.DistributedQueryCampaignTarget
	ds.NewDistributedQueryCampaignTargetFunc = func(target *kolide.DistributedQueryCampaignTarget) (*kolide.DistributedQueryCampaignTarget, error) {
		gotTargets = append(gotTargets, target)
		return target, nil
	}

	ds.CountHostsInTargetsFunc = func(hostIDs, labelIDs []uint, now time.Time) (kolide.TargetMetrics, error) {
		return kolide.TargetMetrics{}, nil
	}
	ds.HostIDsInTargetsFunc = func(hostIDs, labelIDs []uint) ([]uint, error) {
		return []uint{1, 3, 5}, nil
	}
	lq.On("RunQuery", "21", "select year, month, day, hour, minutes, seconds from time", []uint{1, 3, 5}).Return(nil)
	viewerCtx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &kolide.User{
			ID: 0,
		},
	})
	q := "select year, month, day, hour, minutes, seconds from time"
	campaign, err := svc.NewDistributedQueryCampaign(viewerCtx, q, []uint{2}, []uint{1})
	require.Nil(t, err)
	assert.Equal(t, gotQuery.ID, gotCampaign.QueryID)
	assert.Equal(t, []*kolide.DistributedQueryCampaignTarget{
		&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   2,
		},
		&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetLabel,
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
	svc, err := newTestServiceWithClock(ds, rs, lq, mockClock)
	require.Nil(t, err)

	campaign := &kolide.DistributedQueryCampaign{ID: 42}

	ds.LabelQueriesForHostFunc = func(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.SaveHostFunc = func(host *kolide.Host) error {
		return nil
	}
	ds.DistributedQueriesForHostFunc = func(host *kolide.Host) (map[uint]string, error) {
		return map[uint]string{campaign.ID: "select * from time"}, nil
	}
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}

	host := &kolide.Host{ID: 1}
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
	require.Nil(t, err)
	assert.Len(t, queries, len(detailQueries)+1)
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
	require.Nil(t, err)

	// We need to listen for the result in a separate thread to prevent the
	// write to the result channel from failing
	var waitSetup, waitComplete sync.WaitGroup
	waitSetup.Add(1)
	waitComplete.Add(1)
	go func() {
		waitSetup.Done()
		select {
		case val := <-readChan:
			if res, ok := val.(kolide.DistributedQueryResult); ok {
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

	err = svc.SubmitDistributedQueryResults(hostCtx, results, map[string]kolide.OsqueryStatus{}, map[string]string{})
	require.Nil(t, err)
}

func TestIngestDistributedQueryParseIdError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	host := kolide.Host{ID: 1}
	err := svc.ingestDistributedQuery(host, "bad_name", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse campaign")
}

func TestIngestDistributedQueryOrphanedCampaignLoadError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	ds.DistributedQueryCampaignFunc = func(id uint) (*kolide.DistributedQueryCampaign, error) {
		return nil, fmt.Errorf("missing campaign")
	}

	host := kolide.Host{ID: 1}

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading orphaned campaign")
}

func TestIngestDistributedQueryOrphanedCampaignWaitListener(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &kolide.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-1 * time.Second),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(id uint) (*kolide.DistributedQueryCampaign, error) {
		return campaign, nil
	}

	host := kolide.Host{ID: 1}

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign waiting for listener")
}

func TestIngestDistributedQueryOrphanedCloseError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &kolide.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-30 * time.Second),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(id uint) (*kolide.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(campaign *kolide.DistributedQueryCampaign) error {
		return fmt.Errorf("failed save")
	}

	host := kolide.Host{ID: 1}

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing orphaned campaign")
}

func TestIngestDistributedQueryOrphanedStopError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &kolide.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-30 * time.Second),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(id uint) (*kolide.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(campaign *kolide.DistributedQueryCampaign) error {
		return nil
	}
	lq.On("StopQuery", strconv.Itoa(int(campaign.ID))).Return(fmt.Errorf("failed"))

	host := kolide.Host{ID: 1}

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopping orphaned campaign")
}

func TestIngestDistributedQueryOrphanedStop(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &kolide.DistributedQueryCampaign{
		ID: 42,
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: mockClock.Now().Add(-30 * time.Second),
			},
		},
	}

	ds.DistributedQueryCampaignFunc = func(id uint) (*kolide.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(campaign *kolide.DistributedQueryCampaign) error {
		return nil
	}
	lq.On("StopQuery", strconv.Itoa(int(campaign.ID))).Return(nil)

	host := kolide.Host{ID: 1}

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.NoError(t, err)
	lq.AssertExpectations(t)
}

func TestIngestDistributedQueryRecordCompletionError(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &kolide.DistributedQueryCampaign{ID: 42}
	host := kolide.Host{ID: 1}

	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(fmt.Errorf("fail"))

	go func() {
		ch, err := rs.ReadChannel(context.Background(), *campaign)
		require.NoError(t, err)
		<-ch
	}()
	time.Sleep(10 * time.Millisecond)

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "record query completion")
	lq.AssertExpectations(t)
}

func TestIngestDistributedQuery(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	svc := service{
		ds:             ds,
		resultStore:    rs,
		liveQueryStore: lq,
		logger:         log.NewNopLogger(),
		clock:          mockClock,
	}

	campaign := &kolide.DistributedQueryCampaign{ID: 42}
	host := kolide.Host{ID: 1}

	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(nil)

	go func() {
		ch, err := rs.ReadChannel(context.Background(), *campaign)
		require.NoError(t, err)
		<-ch
	}()
	time.Sleep(10 * time.Millisecond)

	err := svc.ingestDistributedQuery(host, "fleet_distributed_query_42", []map[string]string{}, false, "")
	require.NoError(t, err)
	lq.AssertExpectations(t)
}

func TestUpdateHostIntervals(t *testing.T) {
	ds := new(mock.Store)

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ds.ListPacksForHostFunc = func(hid uint) ([]*kolide.Pack, error) {
		return []*kolide.Pack{}, nil
	}

	var testCases = []struct {
		initHost       kolide.Host
		finalHost      kolide.Host
		configOptions  json.RawMessage
		saveHostCalled bool
	}{
		// Both updated
		{
			kolide.Host{
				ConfigTLSRefresh: 60,
			},
			kolide.Host{
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
			kolide.Host{
				DistributedInterval: 11,
				ConfigTLSRefresh:    60,
			},
			kolide.Host{
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
			kolide.Host{
				ConfigTLSRefresh: 60,
				LoggerTLSPeriod:  33,
			},
			kolide.Host{
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
			kolide.Host{
				ConfigTLSRefresh:    60,
				DistributedInterval: 11,
			},
			kolide.Host{
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
			kolide.Host{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			kolide.Host{
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
			kolide.Host{
				DistributedInterval: 11,
				LoggerTLSPeriod:     33,
				ConfigTLSRefresh:    60,
			},
			kolide.Host{
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

			ds.OptionsForPlatformFunc = func(platform string) (json.RawMessage, error) {
				return tt.configOptions, nil
			}

			saveHostCalled := false
			ds.SaveHostFunc = func(host *kolide.Host) error {
				saveHostCalled = true
				assert.Equal(t, tt.finalHost, *host)
				return nil
			}

			_, err := svc.GetClientConfig(ctx)
			require.Nil(t, err)
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
	ms.MarkHostSeenFunc = func(*kolide.Host, time.Time) error {
		return nil
	}
	ms.AuthenticateHostFunc = func(nodeKey string) (*kolide.Host, error) {
		return nil, nil
	}

	svc, err := newTestService(ms, nil, nil)
	require.NoError(t, err)
	ctx := context.Background()

	_, err = svc.AuthenticateHost(ctx, "")
	require.Error(t, err)
	require.True(t, err.(osqueryError).NodeInvalid())

	ms.AuthenticateHostFunc = func(nodeKey string) (*kolide.Host, error) {
		return &kolide.Host{ID: 1}, nil
	}
	_, err = svc.AuthenticateHost(ctx, "foo")
	require.NoError(t, err)

	// return not found error
	ms.AuthenticateHostFunc = func(nodeKey string) (*kolide.Host, error) {
		return nil, notFoundError{}
	}

	_, err = svc.AuthenticateHost(ctx, "foo")
	require.Error(t, err)
	require.True(t, err.(osqueryError).NodeInvalid())

	// return other error
	ms.AuthenticateHostFunc = func(nodeKey string) (*kolide.Host, error) {
		return nil, errors.New("foo")
	}

	_, err = svc.AuthenticateHost(ctx, "foo")
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
