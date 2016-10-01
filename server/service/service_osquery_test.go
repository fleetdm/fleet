package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	hostctx "github.com/kolide/kolide-ose/server/contexts/host"
	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollAgent(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := newTestService(ds)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)
	assert.NotEmpty(t, nodeKey)

	hosts, err = ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestEnrollAgentIncorrectEnrollSecret(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := newTestService(ds)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "not_correct", "host123")
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)

	hosts, err = ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)
}

func TestSubmitStatusLogs(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	_, err = svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	hosts, err := ds.Hosts()
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
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	_, err = svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	hosts, err := ds.Hosts()
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
	host := kolide.Host{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		NodeKey:   "test_key",
		HostName:  "test_hostname",
		UUID:      "test_uuid",
	}

	queries := hostDetailQueries(host)
	assert.Len(t, queries, 1)
	if assert.Contains(t, queries, "kolide_detail_query_platform") {
		assert.Equal(t,
			"select build_platform from osquery_info;",
			queries["kolide_detail_query_platform"],
		)
	}

	host.Platform = "test_platform"

	queries = hostDetailQueries(host)
	assert.Len(t, queries, 0)
}

func TestGetDistributedQueries(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestServiceWithClock(ds, mockClock)
	assert.Nil(t, err)

	ctx := context.Background()

	_, err = svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)

	hosts, err := ds.Hosts()
	require.Nil(t, err)
	require.Len(t, hosts, 1)
	host := hosts[0]

	ctx = hostctx.NewContext(ctx, *host)

	// With no platform set, we should get the details query
	queries, err := svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
	if assert.Contains(t, queries, "kolide_detail_query_platform") {
		assert.Equal(t,
			"select build_platform from osquery_info;",
			queries["kolide_detail_query_platform"],
		)
	}

	host.Platform = "darwin"
	ds.SaveHost(host)
	ctx = hostctx.NewContext(ctx, *host)

	// With the platform set, we should get the label queries (but none
	// exist yet)
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	// Add some queries and labels to ensure they are returned

	labelQueries := []*kolide.Query{
		&kolide.Query{
			ID:       1,
			Name:     "query1",
			Platform: "darwin",
			Query:    "query1",
		},
		&kolide.Query{
			ID:       2,
			Name:     "query2",
			Platform: "darwin",
			Query:    "query2",
		},
		&kolide.Query{
			ID:       3,
			Name:     "query3",
			Platform: "darwin",
			Query:    "query3",
		},
	}

	expectQueries := make(map[string]string)

	for _, query := range labelQueries {
		_, err := ds.NewQuery(query)
		assert.Nil(t, err)
		expectQueries[fmt.Sprintf("kolide_label_query_%d", query.ID)] = query.Query
	}
	// this one should not show up
	_, err = ds.NewQuery(&kolide.Query{
		ID:       4,
		Name:     "query4",
		Platform: "not_darwin",
		Query:    "query4",
	})
	assert.Nil(t, err)

	labels := []*kolide.Label{
		&kolide.Label{
			Name:    "label1",
			QueryID: 1,
		},
		&kolide.Label{
			Name:    "label2",
			QueryID: 2,
		},
		&kolide.Label{
			Name:    "label3",
			QueryID: 3,
		},
		&kolide.Label{
			Name:    "label4",
			QueryID: 4,
		},
	}

	for _, label := range labels {
		_, err := ds.NewLabel(label)
		assert.Nil(t, err)
	}

	// Now we should get the label queries
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)
	assert.Equal(t, expectQueries, queries)

	// Record a query execution
	err = ds.RecordLabelQueryExecutions(host, map[string]bool{"1": true}, mockClock.Now())
	assert.Nil(t, err)

	// Now that query should not be returned
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 2)
	assert.NotContains(t, queries, "kolide_label_query_1")

	// Advance the time
	mockClock.AddTime(1*time.Hour + 1*time.Minute)

	// Now we should get all the label queries again
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 3)
	assert.Equal(t, expectQueries, queries)

	// Record an old query execution -- Shouldn't change the return
	err = ds.RecordLabelQueryExecutions(host, map[string]bool{"2": true}, mockClock.Now().Add(-10*time.Hour))
	assert.Nil(t, err)
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// Record a newer execution for that query and another
	err = ds.RecordLabelQueryExecutions(host, map[string]bool{"2": true, "3": false}, mockClock.Now().Add(-1*time.Minute))
	assert.Nil(t, err)

	// Now these should no longer show up in the necessary to run queries
	delete(expectQueries, "kolide_label_query_2")
	delete(expectQueries, "kolide_label_query_3")
	queries, err = svc.GetDistributedQueries(ctx)
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)
}
