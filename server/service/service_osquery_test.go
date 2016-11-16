package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/WatchBeam/clock"
	hostctx "github.com/kolide/kolide-ose/server/contexts/host"
	"github.com/kolide/kolide-ose/server/datastore/inmem"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollAgent(t *testing.T) {
	ds, err := inmem.New()
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)
	assert.NotEmpty(t, nodeKey)

	hosts, err = ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestEnrollAgentIncorrectEnrollSecret(t *testing.T) {
	ds, err := inmem.New()
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

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

func TestSubmitStatusLogs(t *testing.T) {
	ds, err := inmem.New()
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, nil, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	_, err = svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(service)

	// Error due to missing host
	err = serv.SubmitResultLogs(ctx, []kolide.OsqueryResultLog{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing host")

	// Add that host
	ctx = hostctx.NewContext(ctx, *host)

	var statusBuf bytes.Buffer
	serv.osqueryStatusLogWriter = &statusBuf

	logs := []string{
		`{"severity":"0","filename":"tls.cpp","line":"216","message":"some message","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
		`{"severity":"1","filename":"buffered.cpp","line":"122","message":"warning!","version":"1.8.2","decorations":{"host_uuid":"uuid_foobar","username":"zwass"}}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var status []kolide.OsqueryStatusLog
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

	// Verify that the update time is set appropriately
	checkHost, err := ds.Host(host.ID)
	assert.Nil(t, err)
	assert.Equal(t, mockClock.Now(), checkHost.UpdatedAt)

	// Advance clock time and check that time is updated on new logs
	mockClock.AddTime(1 * time.Minute)

	err = serv.SubmitStatusLogs(ctx, []kolide.OsqueryStatusLog{})
	assert.Nil(t, err)

	checkHost, err = ds.Host(host.ID)
	assert.Nil(t, err)
	assert.Equal(t, mockClock.Now(), checkHost.UpdatedAt)
}

func TestSubmitResultLogs(t *testing.T) {
	ds, err := inmem.New()
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, nil, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	_, err = svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]

	// Hack to get at the service internals and modify the writer
	serv := ((svc.(validationMiddleware)).Service).(service)

	// Error due to missing host
	err = serv.SubmitResultLogs(ctx, []kolide.OsqueryResultLog{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing host")

	ctx = hostctx.NewContext(ctx, *host)

	var resultBuf bytes.Buffer
	serv.osqueryResultLogWriter = &resultBuf

	logs := []string{
		`{"name":"system_info","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 17:55:15 2016 UTC","unixTime":"1475258115","decorations":{"host_uuid":"some_uuid","username":"zwass"},"columns":{"cpu_brand":"Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz","hostname":"hostimus","physical_memory":"17179869184"},"action":"added"}`,
		`{"name":"encrypted","hostIdentifier":"some_uuid","calendarTime":"Fri Sep 30 21:19:15 2016 UTC","unixTime":"1475270355","decorations":{"host_uuid":"4740D59F-699E-5B29-960B-979AAF9BBEEB","username":"zwass"},"columns":{"encrypted":"1","name":"\/dev\/disk1","type":"AES-XTS","uid":"","user_uuid":"","uuid":"some_uuid"},"action":"added"}`,
	}
	logJSON := fmt.Sprintf("[%s]", strings.Join(logs, ","))

	var results []kolide.OsqueryResultLog
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

	// Verify that the update time is set appropriately
	checkHost, err := ds.Host(host.ID)
	assert.Nil(t, err)
	assert.Equal(t, mockClock.Now(), checkHost.UpdatedAt)

	// Advance clock time and check that time is updated on new logs
	mockClock.AddTime(1 * time.Minute)

	err = serv.SubmitResultLogs(ctx, []kolide.OsqueryResultLog{})
	assert.Nil(t, err)

	checkHost, err = ds.Host(host.ID)
	assert.Nil(t, err)
	assert.Equal(t, mockClock.Now(), checkHost.UpdatedAt)
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
	ds, err := inmem.New()
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, nil, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	_, err = svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]

	ctx = hostctx.NewContext(ctx, *host)

	// With a new host, we should get the detail queries
	queries, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))

	// Simulate the detail queries being added
	host.DetailUpdateTime = mockClock.Now().Add(-1 * time.Minute)
	host.Platform = "darwin"
	ds.SaveHost(host)
	ctx = hostctx.NewContext(ctx, *host)

	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

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
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "1": {{"col1": "val1"}},
		},
	)
	assert.Nil(t, err)

	// Verify that labels are set appropriately
	hostLabels, err := ds.ListLabelsForHost(host.ID)
	assert.Len(t, hostLabels, 1)
	assert.Equal(t, "label1", hostLabels[0].Name)

	// Now that query should not be returned
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 2)
	assert.NotContains(t, queries, "kolide_label_query_1")

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	// Keep the host details fresh
	host.DetailUpdateTime = mockClock.Now().Add(-1 * time.Minute)
	ds.SaveHost(host)
	ctx = hostctx.NewContext(ctx, *host)

	// Now we should get all the label queries again
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)

	// Record a query execution
	err = svc.SubmitDistributedQueryResults(
		ctx,
		map[string][]map[string]string{
			hostLabelQueryPrefix + "2": {{"col1": "val1"}},
			hostLabelQueryPrefix + "3": {},
		},
	)
	assert.Nil(t, err)

	// Now these should no longer show up in the necessary to run queries
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 1)

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
	ds, err := inmem.New()
	assert.Nil(t, err)

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
	assert.False(t, config.Options.DisableDistributed)
	assert.Equal(t, "/", config.Options.PackDelimiter)

	// this will be greater than 0 if we ever start inserting an administration
	// pack
	assert.Len(t, config.Packs, 0)

	// let's populate the database with some info

	infoQuery := &kolide.Query{
		Name:     "Info",
		Query:    "select * from osquery_info;",
		Interval: 60,
	}
	infoQuery, err = ds.NewQuery(infoQuery)
	assert.Nil(t, err)

	monitoringPack := &kolide.Pack{
		Name: "monitoring",
	}
	_, err = ds.NewPack(monitoringPack)
	assert.Nil(t, err)

	err = ds.AddQueryToPack(infoQuery.ID, monitoringPack.ID)
	assert.Nil(t, err)

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
		map[string]bool{fmt.Sprintf("%d", mysqlLabel.ID): true},
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

func TestDetailQueries(t *testing.T) {
	ds, err := inmem.New()
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, nil, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	host, err := ds.AuthenticateHost(nodeKey)
	require.Nil(t, err)

	ctx = hostctx.NewContext(ctx, *host)

	// With a new host, we should get the detail queries
	queries, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))

	resultJSON := `
{
"kolide_detail_query_network_interface": [
    {
        "address": "192.168.0.1",
        "broadcast": "192.168.0.255",
        "ibytes": "1601207629",
        "ierrors": "0",
        "interface": "en0",
        "ipackets": "25698094",
        "last_change": "1474233476",
        "mac": "5f:3d:4b:10:25:82",
        "mask": "255.255.255.0",
        "metric": "0",
        "mtu": "1453",
        "obytes": "2607283152",
        "oerrors": "0",
        "opackets": "12264603",
        "point_to_point": "",
        "type": "6"
    }
],
"kolide_detail_query_os_version": [
    {
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
]
}
`

	var results kolide.OsqueryDistributedQueryResults
	err = json.Unmarshal([]byte(resultJSON), &results)
	require.Nil(t, err)

	// Verify that results are ingested properly
	svc.SubmitDistributedQueryResults(ctx, results)

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

	// network_interface
	assert.Equal(t, "5f:3d:4b:10:25:82", host.PrimaryMAC)
	assert.Equal(t, "192.168.0.1", host.PrimaryIP)

	ctx = hostctx.NewContext(ctx, *host)

	// Now no detail queries should be required
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	// Advance clock and queries should exist again
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
}

func TestDistributedQueries(t *testing.T) {
	ds, err := inmem.New()
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

	ctx = hostctx.NewContext(ctx, *host)

	// Create label
	n := "foo"
	q := "select * from foo;"
	label, err := svc.NewLabel(ctx, kolide.LabelPayload{
		Name:  &n,
		Query: &q,
	})
	require.Nil(t, err)
	labelId := strconv.Itoa(int(label.ID))

	// Record match with label
	err = ds.RecordLabelQueryExecutions(host, map[string]bool{labelId: true}, mockClock.Now())
	require.Nil(t, err)

	q = "select year, month, day, hour, minutes, seconds from time"
	campaign, err := svc.NewDistributedQueryCampaign(ctx, 0, q, []uint{}, []uint{label.ID})
	require.Nil(t, err)

	queryKey := fmt.Sprintf("%s%d", hostDistributedQueryPrefix, campaign.ID)

	// Now we should get the active distributed query
	queries, err := svc.GetDistributedQueries(ctx)
	require.Nil(t, err)
	assert.Len(t, queries, len(detailQueries)+1)
	assert.Equal(t, q, queries[queryKey])

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

	// Submit results (should error because no one is listening)
	err = svc.SubmitDistributedQueryResults(ctx, results)
	assert.NotNil(t, err)

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

	err = svc.SubmitDistributedQueryResults(ctx, results)
	require.Nil(t, err)

	// Now the distributed query should be completed and not returned
	queries, err = svc.GetDistributedQueries(ctx)
	require.Nil(t, err)
	assert.Len(t, queries, len(detailQueries))
	assert.NotContains(t, queries, queryKey)

	waitComplete.Wait()
}
