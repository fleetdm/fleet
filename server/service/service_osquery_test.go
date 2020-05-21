package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/fleet/server/config"
	hostctx "github.com/kolide/fleet/server/contexts/host"
	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/datastore/inmem"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/logging"
	"github.com/kolide/fleet/server/mock"
	"github.com/kolide/fleet/server/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollAgent(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123", nil)
	require.Nil(t, err)
	assert.NotEmpty(t, nodeKey)

	hosts, err = ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestEnrollAgentIncorrectEnrollSecret(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "not_correct", "host123", nil)
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)

	hosts, err = ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)
}

func TestEnrollAgentDetails(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

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
	require.Nil(t, err)
	assert.NotEmpty(t, nodeKey)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)

	h := hosts[0]
	assert.Equal(t, "Mac OS X 10.14.5", h.OSVersion)
	assert.Equal(t, "darwin", h.Platform)
	assert.Equal(t, "2.12.0", h.OsqueryVersion)
	assert.Equal(t, "zwass.local", h.HostName)
	assert.Equal(t, "froobling_uuid", h.UUID)
}

func TestAuthenticateHost(t *testing.T) {
	ds, svc, mockClock := setupOsqueryTests(t)
	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123", nil)
	require.Nil(t, err)

	mockClock.AddTime(1 * time.Minute)

	host, err := svc.AuthenticateHost(ctx, nodeKey)
	require.Nil(t, err)

	// Verify that the update time is set appropriately
	checkHost, err := ds.Host(host.ID)
	require.Nil(t, err)
	assert.Equal(t, mockClock.Now(), checkHost.UpdatedAt)

	// Advance clock time and check that seen time is updated
	mockClock.AddTime(1*time.Minute + 27*time.Second)

	_, err = svc.AuthenticateHost(ctx, nodeKey)
	require.Nil(t, err)

	checkHost, err = ds.Host(host.ID)
	require.Nil(t, err)
	assert.Equal(t, mockClock.Now(), checkHost.UpdatedAt)
}

type testJSONLogger struct {
	logs []json.RawMessage
}

func (n *testJSONLogger) Write(ctx context.Context, logs []json.RawMessage) error {
	n.logs = logs
	return nil
}

func TestSubmitStatusLogs(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	_, err := svc.EnrollAgent(ctx, "", "host123", nil)
	require.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]
	ctx = hostctx.NewContext(ctx, *host)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(service)

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

	err = serv.SubmitStatusLogs(ctx, status)
	assert.Nil(t, err)

	assert.Equal(t, status, testLogger.logs)
}

func TestSubmitResultLogs(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	_, err := svc.EnrollAgent(ctx, "", "host123", nil)
	require.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]
	ctx = hostctx.NewContext(ctx, *host)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(service)

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
	svc, err := newTestService(&mock.Store{}, nil)
	require.Nil(t, err)

	_, _, err = svc.GetDistributedQueries(context.Background())
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestLabelQueries(t *testing.T) {
	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	svc, err := newTestServiceWithClock(ds, nil, mockClock)
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
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}

	host := &kolide.Host{}
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
	)
	assert.Nil(t, err)
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
	)
	assert.Nil(t, err)
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

	svc, err := newTestService(ds, nil)
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
	ds, svc, mockClock := setupOsqueryTests(t)
	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123", nil)
	assert.Nil(t, err)

	host, err := ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = hostctx.NewContext(ctx, *host)

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.NotZero(t, acc)

	resultJSON := `
{
"kolide_detail_query_network_interface": [
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
"kolide_detail_query_os_version": [
		{
				"platform": "darwin",
				"build": "15G1004",
				"major": "10",
				"minor": "10",
				"name": "Mac OS X",
				"patch": "6"
		}
],
"kolide_detail_query_osquery_info": [
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
"kolide_detail_query_system_info": [
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
"kolide_detail_query_uptime": [
		{
				"days": "20",
				"hours": "0",
				"minutes": "48",
				"seconds": "13",
				"total_seconds": "1730893"
		}
],
"kolide_detail_query_osquery_flags": [
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

	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results, map[string]kolide.OsqueryStatus{})

	// Make sure the result saved to the datastore
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	// osquery_info
	assert.Equal(t, "darwin", host.Platform)
	assert.Equal(t, "1.8.2", host.OsqueryVersion)

	// system_info
	assert.Equal(t, 17179869184, host.PhysicalMemory)
	assert.Equal(t, "computer.local", host.HostName)
	assert.Equal(t, "uuid", host.UUID)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", host.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, host.Uptime)

	// osquery_flags
	assert.Equal(t, uint(0), host.ConfigTLSRefresh)
	assert.Equal(t, uint(0), host.DistributedInterval)
	assert.Equal(t, uint(0), host.LoggerTLSPeriod)

	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)
	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	err = svc.SubmitDistributedQueryResults(ctx, kolide.OsqueryDistributedQueryResults{}, map[string]kolide.OsqueryStatus{})
	require.Nil(t, err)
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.Zero(t, acc)
}

func TestDetailQueries(t *testing.T) {
	ds, svc, mockClock := setupOsqueryTests(t)
	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123", nil)
	assert.Nil(t, err)

	host, err := ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = hostctx.NewContext(ctx, *host)

	// With a new host, we should get the detail queries (and accelerated
	// queries)
	queries, acc, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.NotZero(t, acc)

	resultJSON := `
{
"kolide_detail_query_network_interface": [
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
"kolide_detail_query_os_version": [
    {
        "platform": "darwin",
        "build": "15G1004",
        "major": "10",
        "minor": "10",
        "name": "Mac OS X",
        "patch": "6"
    }
],
"kolide_detail_query_osquery_info": [
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
"kolide_detail_query_system_info": [
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
"kolide_detail_query_uptime": [
    {
        "days": "20",
        "hours": "0",
        "minutes": "48",
        "seconds": "13",
        "total_seconds": "1730893"
    }
],
"kolide_detail_query_osquery_flags": [
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

	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results, map[string]kolide.OsqueryStatus{})

	// Make sure the result saved to the datastore
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	// osquery_info
	assert.Equal(t, "darwin", host.Platform)
	assert.Equal(t, "1.8.2", host.OsqueryVersion)

	// system_info
	assert.Equal(t, 17179869184, host.PhysicalMemory)
	assert.Equal(t, "computer.local", host.HostName)
	assert.Equal(t, "uuid", host.UUID)

	// os_version
	assert.Equal(t, "Mac OS X 10.10.6", host.OSVersion)

	// uptime
	assert.Equal(t, 1730893*time.Second, host.Uptime)

	// osquery_flags
	assert.Equal(t, uint(10), host.ConfigTLSRefresh)
	assert.Equal(t, uint(5), host.DistributedInterval)
	assert.Equal(t, uint(60), host.LoggerTLSPeriod)

	mockClock.AddTime(1 * time.Minute)

	// Now no detail queries should be required
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)
	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Zero(t, acc)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	err = svc.SubmitDistributedQueryResults(ctx, kolide.OsqueryDistributedQueryResults{}, map[string]kolide.OsqueryStatus{})
	require.Nil(t, err)
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.Zero(t, acc)
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
	mockClock := clock.NewMockClock()
	svc, err := newTestServiceWithClock(ds, rs, mockClock)
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
	svc, err := newTestServiceWithClock(ds, rs, mockClock)
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
	var gotExecution *kolide.DistributedQueryExecution
	ds.NewDistributedQueryExecutionFunc = func(exec *kolide.DistributedQueryExecution) (*kolide.DistributedQueryExecution, error) {
		gotExecution = exec
		return exec, nil
	}
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{}, nil
	}

	host := &kolide.Host{ID: 1}
	hostCtx := hostctx.NewContext(context.Background(), *host)

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

	err = svc.SubmitDistributedQueryResults(hostCtx, results, map[string]kolide.OsqueryStatus{})
	require.Nil(t, err)
	assert.Equal(t, campaign.ID, gotExecution.DistributedQueryCampaignID)
	assert.Equal(t, host.ID, gotExecution.HostID)
	assert.Equal(t, kolide.ExecutionSucceeded, gotExecution.Status)
}

func TestOrphanedQueryCampaign(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	_, err = ds.NewAppConfig(&kolide.AppConfig{EnrollSecret: ""})
	require.Nil(t, err)

	rs := pubsub.NewInmemQueryResults()

	svc, err := newTestService(ds, rs)
	require.Nil(t, err)

	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123", nil)
	require.Nil(t, err)

	host, err := ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = viewer.NewContext(context.Background(), viewer.Viewer{
		User: &kolide.User{
			ID: 0,
		},
	})
	q := "select year, month, day, hour, minutes, seconds from time"
	campaign, err := svc.NewDistributedQueryCampaign(ctx, q, []uint{}, []uint{})
	require.Nil(t, err)

	campaign.Status = kolide.QueryRunning
	err = ds.SaveDistributedQueryCampaign(campaign)
	require.Nil(t, err)

	queryKey := fmt.Sprintf("%s%d", hostDistributedQueryPrefix, campaign.ID)

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

	// Submit results
	ctx = hostctx.NewContext(context.Background(), *host)
	err = svc.SubmitDistributedQueryResults(ctx, results, map[string]kolide.OsqueryStatus{})
	require.Nil(t, err)

	// The campaign should be set to completed because it is orphaned
	campaign, err = ds.DistributedQueryCampaign(campaign.ID)
	require.Nil(t, err)
	assert.Equal(t, kolide.QueryComplete, campaign.Status)
}

func TestUpdateHostIntervals(t *testing.T) {
	ds := new(mock.Store)

	svc, err := newTestService(ds, nil)
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

func setupOsqueryTests(t *testing.T) (kolide.Datastore, kolide.Service, *clock.MockClock) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	_, err = ds.NewAppConfig(&kolide.AppConfig{EnrollSecret: ""})
	require.Nil(t, err)

	mockClock := clock.NewMockClock()
	svc, err := newTestServiceWithClock(ds, nil, mockClock)
	require.Nil(t, err)

	return ds, svc, mockClock
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

	svc, err := newTestService(ms, nil)
	require.Nil(t, err)
	ctx := context.Background()

	_, err = svc.AuthenticateHost(ctx, "")
	require.NotNil(t, err)
	require.True(t, err.(osqueryError).NodeInvalid())

	_, err = svc.AuthenticateHost(ctx, "foo")
	require.Nil(t, err)

	// return not found error
	ms.AuthenticateHostFunc = func(nodeKey string) (*kolide.Host, error) {
		return nil, notFoundError{}
	}

	_, err = svc.AuthenticateHost(ctx, "foo")
	require.NotNil(t, err)
	require.True(t, err.(osqueryError).NodeInvalid())

	// return other error
	ms.AuthenticateHostFunc = func(nodeKey string) (*kolide.Host, error) {
		return nil, errors.New("foo")
	}

	_, err = svc.AuthenticateHost(ctx, "foo")
	require.NotNil(t, err)
	require.False(t, err.(osqueryError).NodeInvalid())
}
