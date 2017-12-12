package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/kolide/fleet/server/mock"
	"github.com/kolide/fleet/server/pubsub"
	"github.com/kolide/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollAgent(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
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

	nodeKey, err := svc.EnrollAgent(ctx, "not_correct", "host123")
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)

	hosts, err = ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)
}

func TestAuthenticateHost(t *testing.T) {
	ds, svc, mockClock := setupOsqueryTests(t)
	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
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

type nopCloserWriter struct {
	io.Writer
}

func (n *nopCloserWriter) Close() error { return nil }

func TestSubmitStatusLogs(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	_, err := svc.EnrollAgent(ctx, "", "host123")
	require.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]
	ctx = hostctx.NewContext(ctx, *host)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(service)

	var statusBuf bytes.Buffer
	serv.osqueryStatusLogWriter = &nopCloserWriter{&statusBuf}

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

	statusJSON := statusBuf.String()
	statusJSON = strings.TrimRight(statusJSON, "\n")
	statusLines := strings.Split(statusJSON, "\n")

	if assert.Equal(t, len(logs), len(statusLines)) {
		for i, line := range statusLines {
			assert.JSONEq(t, logs[i], line)
		}
	}
}

func TestSubmitResultLogs(t *testing.T) {
	ds, svc, _ := setupOsqueryTests(t)
	ctx := context.Background()

	_, err := svc.EnrollAgent(ctx, "", "host123")
	require.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]
	ctx = hostctx.NewContext(ctx, *host)

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(service)

	var resultBuf bytes.Buffer
	serv.osqueryResultLogWriter = &nopCloserWriter{&resultBuf}

	logs := []string{
		`{"name":"system_info","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 17:55:15 2016 UTC","unixTime":"1475258115","decorations":{"host_uuid":"some_uuid","username":"zwass"},"columns":{"cpu_brand":"Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz","hostname":"hostimus","physical_memory":"17179869184"},"action":"added"}`,
		`{"name":"encrypted","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 21:19:15 2016 UTC","unixTime":"1475270355","decorations":{"host_uuid":"4740D59F-699E-5B29-960B-979AAF9BBEEB","username":"zwass"},"columns":{"encrypted":"1","name":"\/dev\/disk1","type":"AES-XTS","uid":"","user_uuid":"","uuid":"some_uuid"},"action":"added"}`,
		`{"snapshot":[{"hour":"20","minutes":"8"}],"action":"snapshot","name":"time","hostIdentifier":"1379f59d98f4","calendarTime":"Tue Jan 10 20:08:51 2017 UTC","unixTime":"1484078931","decorations":{"host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA"}}`,
		`{"diffResults":{"removed":[{"address":"127.0.0.1","hostnames":"kl.groob.io"}],"added":""},"name":"pack\/test\/hosts","hostIdentifier":"FA01680E-98CA-5557-8F59-7716ECFEE964","calendarTime":"Sun Nov 19 00:02:08 2017 UTC","unixTime":"1511049728","epoch":"0","counter":"10","decorations":{"host_uuid":"FA01680E-98CA-5557-8F59-7716ECFEE964","hostname":"kl.groob.io"}}`,
		// fleet will accept anything in the "data" field of an log request.
		`{"unknown":{"foo": [] }}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var results []json.RawMessage
	err = json.Unmarshal([]byte(logJSON), &results)
	require.Nil(t, err)

	err = serv.SubmitResultLogs(ctx, results)
	assert.Nil(t, err)

	resultJSON := resultBuf.String()
	resultJSON = strings.TrimRight(resultJSON, "\n")
	resultLines := strings.Split(resultJSON, "\n")

	if assert.Equal(t, len(logs), len(resultLines)) {
		for i, line := range resultLines {
			assert.JSONEq(t, logs[i], line)
		}
	}
}

func TestHostDetailQueries(t *testing.T) {
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

	svc := service{clock: mockClock}

	queries := svc.hostDetailQueries(host)
	assert.Empty(t, queries)

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries = svc.hostDetailQueries(host)
	assert.Len(t, queries, len(detailQueries))
	for name, _ := range queries {
		assert.True(t,
			strings.HasPrefix(name, hostDetailQueryPrefix),
			fmt.Sprintf("%s not prefixed with %s", name, hostDetailQueryPrefix),
		)
	}
}

func TestLabelQueries(t *testing.T) {
	ds, svc, mockClock := setupOsqueryTests(t)
	ctx := context.Background()

	_, err := svc.EnrollAgent(ctx, "", "host123")
	require.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]

	ctx = hostctx.NewContext(ctx, *host)

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
	ds.SaveHost(host)
	ctx = hostctx.NewContext(ctx, *host)

	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Zero(t, acc)

	labels := []kolide.Label{
		kolide.Label{
			Name:     "label1",
			Query:    "query1",
			Platform: "darwin",
		},
		kolide.Label{
			Name:     "label2",
			Query:    "query2",
			Platform: "darwin",
		},
		kolide.Label{
			Name:     "label3",
			Query:    "query3",
			Platform: "darwin,linux",
		},
		kolide.Label{
			Name:     "label4",
			Query:    "query4",
			Platform: "linux",
		},
	}

	for _, label := range labels {
		_, err := ds.NewLabel(&label)
		assert.Nil(t, err)
	}

	// Now we should get the label queries
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)
	assert.Zero(t, acc)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "1": {{"col1": "val1"}},
		},
		map[string]string{},
	)
	assert.Nil(t, err)

	// Verify that labels are set appropriately
	hostLabels, err := ds.ListLabelsForHost(host.ID)
	assert.Len(t, hostLabels, 1)
	assert.Equal(t, "label1", hostLabels[0].Name)

	// Now that query should not be returned
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 2)
	assert.NotContains(t, queries, "kolide_label_query_1")
	assert.Zero(t, acc)

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	// Keep the host details fresh
	host.DetailUpdateTime = mockClock.Now().Add(-1 * time.Minute)
	ds.SaveHost(host)
	ctx = hostctx.NewContext(ctx, *host)

	// Now we should get all the label queries again
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)
	assert.Zero(t, acc)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "2": {{"col1": "val1"}},
			hostLabelQueryPrefix + "3": {},
		},
		map[string]string{},
	)
	assert.Nil(t, err)

	// Now these should no longer show up in the necessary to run queries
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Zero(t, acc)

	// Verify that labels are set appropriately
	hostLabels, err = ds.ListLabelsForHost(host.ID)
	assert.Len(t, hostLabels, 2)
	expectLabelNames := map[string]bool{"label1": true, "label2": true}
	for _, label := range hostLabels {
		assert.Contains(t, expectLabelNames, label.Name)
		delete(expectLabelNames, label.Name)
	}
}

func TestGetClientConfig(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	require.Nil(t, ds.MigrateData())

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, nil, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 0)

	_, err = svc.EnrollAgent(ctx, "", "user.local")
	assert.Nil(t, err)

	hosts, err = ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]

	ctx = hostctx.NewContext(ctx, *host)

	// with no queries, packs, labels, etc. verify the state of a fresh host
	// asking for a config
	config, err := svc.GetClientConfig(ctx)
	require.Nil(t, err)
	assert.NotNil(t, config)
	val, ok := config.Options["disable_distributed"]
	require.True(t, ok)
	disabled, ok := val.(bool)
	require.True(t, ok)
	assert.False(t, disabled)
	val, ok = config.Options["pack_delimiter"]
	require.True(t, ok)
	delim, ok := val.(string)
	require.True(t, ok)
	assert.Equal(t, "/", delim)

	// this will be greater than 0 if we ever start inserting an administration
	// pack
	assert.Len(t, config.Packs, 0)

	// let's populate the database with some info

	infoQuery := &kolide.Query{
		Name:  "Info",
		Query: "select * from osquery_info;",
	}
	infoQueryInterval := uint(60)
	infoQuery, err = ds.NewQuery(infoQuery)
	assert.Nil(t, err)

	monitoringPack := &kolide.Pack{
		Name: "monitoring",
	}
	_, err = ds.NewPack(monitoringPack)
	assert.Nil(t, err)

	test.NewScheduledQuery(t, ds, monitoringPack.ID, infoQuery.ID, infoQueryInterval, false, false)

	mysqlLabel := &kolide.Label{
		Name:  "MySQL Monitoring",
		Query: "select pid from processes where name = 'mysqld';",
	}
	mysqlLabel, err = ds.NewLabel(mysqlLabel)
	assert.Nil(t, err)

	err = ds.AddLabelToPack(mysqlLabel.ID, monitoringPack.ID)
	assert.Nil(t, err)

	err = ds.RecordLabelQueryExecutions(
		host,
		map[uint]bool{mysqlLabel.ID: true},
		mockClock.Now(),
	)
	assert.Nil(t, err)

	// with a minimal setup of packs, labels, and queries, will our host get the
	// pack
	config, err = svc.GetClientConfig(ctx)
	require.Nil(t, err)
	assert.Len(t, config.Packs, 1)
	assert.Len(t, config.Packs["monitoring"].Queries, 1)
}

func TestDetailQueriesWithEmptyStrings(t *testing.T) {
	ds, svc, mockClock := setupOsqueryTests(t)
	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
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
	svc.SubmitDistributedQueryResults(ctx, results, map[string]string{})

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

	err = svc.SubmitDistributedQueryResults(ctx, kolide.OsqueryDistributedQueryResults{}, map[string]string{})
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

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
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
	svc.SubmitDistributedQueryResults(ctx, results, map[string]string{})

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

	err = svc.SubmitDistributedQueryResults(ctx, kolide.OsqueryDistributedQueryResults{}, map[string]string{})
	require.Nil(t, err)
	host, err = ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = hostctx.NewContext(ctx, *host)
	queries, acc, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.Zero(t, acc)
}

func TestDistributedQueries(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	_, err = ds.NewAppConfig(&kolide.AppConfig{EnrollSecret: ""})
	require.Nil(t, err)

	mockClock := clock.NewMockClock()

	rs := pubsub.NewInmemQueryResults()

	svc, err := newTestServiceWithClock(ds, rs, mockClock)
	require.Nil(t, err)

	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
	require.Nil(t, err)

	host, err := ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	host.Platform = "centos"
	host.HostName = "zwass.local"
	require.Nil(t, ds.SaveHost(host))

	// Create label
	n := "foo"
	q := "select * from foo;"
	label, err := svc.NewLabel(ctx, kolide.LabelPayload{
		Name:  &n,
		Query: &q,
	})
	require.Nil(t, err)

	// Record match with label
	ctx = viewer.NewContext(ctx, viewer.Viewer{
		User: &kolide.User{
			ID: 0,
		},
	})
	err = ds.RecordLabelQueryExecutions(host, map[uint]bool{label.ID: true}, mockClock.Now())
	require.Nil(t, err)
	err = ds.MarkHostSeen(host, mockClock.Now())
	require.Nil(t, err)
	ctx = hostctx.NewContext(ctx, *host)

	q = "select year, month, day, hour, minutes, seconds from time"
	campaign, err := svc.NewDistributedQueryCampaign(ctx, q, []uint{}, []uint{label.ID})
	require.Nil(t, err)

	// Manually set the campaign to running (so that it shows up when
	// requesting queries)
	campaign.Status = kolide.QueryRunning
	err = ds.SaveDistributedQueryCampaign(campaign)
	require.Nil(t, err)

	queryKey := fmt.Sprintf("%s%d", hostDistributedQueryPrefix, campaign.ID)

	// Now we should get the active distributed query
	queries, acc, err := svc.GetDistributedQueries(ctx)
	require.Nil(t, err)
	assert.Len(t, queries, len(detailQueries)+1)
	assert.Equal(t, q, queries[queryKey])
	assert.Zero(t, acc)

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
	readChan, err := rs.ReadChannel(ctx, *campaign)
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

	err = svc.SubmitDistributedQueryResults(ctx, results, map[string]string{})
	require.Nil(t, err)

	// Now the distributed query should be completed and not returned
	queries, acc, err = svc.GetDistributedQueries(ctx)
	require.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.NotContains(t, queries, queryKey)
	assert.Zero(t, acc)

	waitComplete.Wait()
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

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
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
	err = svc.SubmitDistributedQueryResults(ctx, results, map[string]string{})
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

	ds.ListDecoratorsFunc = func(opt ...kolide.OptionalArg) ([]*kolide.Decorator, error) {
		return []*kolide.Decorator{}, nil
	}
	ds.ListPacksFunc = func(opt kolide.ListOptions) ([]*kolide.Pack, error) {
		return []*kolide.Pack{}, nil
	}
	ds.ListLabelsForHostFunc = func(hid uint) ([]kolide.Label, error) {
		return []kolide.Label{}, nil
	}
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{FIMInterval: 400}, nil
	}
	ds.FIMSectionsFunc = func() (kolide.FIMSections, error) {
		sections := kolide.FIMSections{
			"etc": []string{
				"/etc/%%",
			},
		}
		return sections, nil
	}

	var testCases = []struct {
		initHost       kolide.Host
		finalHost      kolide.Host
		configOptions  map[string]interface{}
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
			map[string]interface{}{
				"distributed_interval": 11,
				"logger_tls_period":    33,
				"logger_plugin":        "tls",
			},
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
			map[string]interface{}{
				"distributed_interval": 11,
				"logger_tls_period":    33,
			},
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
			map[string]interface{}{
				"distributed_interval": 11,
				"logger_tls_period":    33,
			},
			true,
		},
		// Kolide not managing distributed_interval
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
			map[string]interface{}{
				"logger_tls_period": 33,
			},
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
			map[string]interface{}{
				"distributed_interval": 11,
				"logger_tls_period":    33,
			},
			false,
		},
	}

	for _, tt := range testCases {
		ds.FIMSectionsFuncInvoked = false

		t.Run("", func(t *testing.T) {
			ctx := hostctx.NewContext(context.Background(), tt.initHost)

			ds.GetOsqueryConfigOptionsFunc = func() (map[string]interface{}, error) {
				return tt.configOptions, nil
			}

			saveHostCalled := false
			ds.SaveHostFunc = func(host *kolide.Host) error {
				saveHostCalled = true
				assert.Equal(t, tt.finalHost, *host)
				return nil
			}

			cfg, err := svc.GetClientConfig(ctx)
			require.Nil(t, err)
			assert.Equal(t, tt.saveHostCalled, saveHostCalled)
			require.True(t, ds.FIMSectionsFuncInvoked)
			require.Condition(t, func() bool {
				_, ok := cfg.Schedule["file_events"]
				return ok
			})
			assert.Equal(t, 400, int(cfg.Schedule["file_events"].Interval))
			assert.Equal(t, "SELECT * FROM file_events;", cfg.Schedule["file_events"].Query)
			require.NotNil(t, cfg.FilePaths)
			require.Condition(t, func() bool {
				_, ok := cfg.FilePaths["etc"]
				return ok
			})
			assert.Len(t, cfg.FilePaths["etc"], 1)
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
