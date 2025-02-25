package service

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	ws "github.com/fleetdm/fleet/v4/server/websocket"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamCampaignResultsClosesReditOnWSClose(t *testing.T) {
	t.Skip("Seems to be a bit problematic in CI")

	store := pubsub.SetupRedisForTest(t, false, false)

	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := live_query_mock.New(t)
	svc, ctx := newTestServiceWithClock(t, ds, store, lq, mockClock)

	campaign := &fleet.DistributedQueryCampaign{ID: 42}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{TotalHosts: 1}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              42,
			AccessedAt:      time.Now(),
			UserID:          999,
			Key:             "asd",
		}, nil
	}

	host := &fleet.Host{ID: 1, Platform: "windows"}

	lq.On("QueriesForHost", uint(1)).Return(
		map[string]string{
			strconv.Itoa(int(campaign.ID)): "select * from time",
		},
		nil,
	)
	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(nil)
	lq.On("RunQuery", "0", "select year, month, day, hour, minutes, seconds from time", []uint{1}).Return(nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{
		User: &fleet.User{
			ID:         0,
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	})
	q := "select year, month, day, hour, minutes, seconds from time"
	_, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)

	pathHandler := makeStreamDistributedQueryCampaignResultsHandler(config.TestConfig().Server, svc, kitlog.NewNopLogger())
	s := httptest.NewServer(pathHandler("/api/latest/fleet/results/"))
	defer s.Close()
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/api/latest/fleet/results/websocket"

	// Connect to the server
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}

	conn, _, err := dialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer conn.Close()

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "auth",
		Data: map[string]interface{}{"token": "asd"},
	})
	require.NoError(t, err)

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "select_campaign",
		Data: map[string]interface{}{"campaign_id": campaign.ID},
	})
	require.NoError(t, err)

	ds.MarkSessionAccessedFunc = func(context.Context, *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}
	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) error {
		return nil
	}
	ds.DistributedQueryCampaignTargetIDsFunc = func(ctx context.Context, id uint) (targets *fleet.HostTargets, err error) {
		return &fleet.HostTargets{HostIDs: []uint{1}}, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}

	/*****************************************************************************************/
	/* THE ACTUAL TEST BEGINS HERE                                                           */
	/*****************************************************************************************/
	prevActiveConn := 0
	for prevActiveConn < 3 {
		time.Sleep(2 * time.Second)

		for _, s := range store.Pool().Stats() {
			prevActiveConn = s.ActiveCount
		}
	}

	conn.Close()
	time.Sleep(10 * time.Second)

	newActiveConn := prevActiveConn
	for _, s := range store.Pool().Stats() {
		newActiveConn = s.ActiveCount
	}
	require.Equal(t, prevActiveConn-1, newActiveConn)
}

func testUpdateStats(t *testing.T, ds *mysql.Datastore, usingReplica bool) {
	t.Cleanup(
		func() {
			overwriteLastExecuted = false
		},
	)
	s, ctx := newTestService(t, ds, nil, nil)
	svc := s.(validationMiddleware).Service.(*Service)

	tracker := statsTracker{}
	// NOOP cases
	svc.updateStats(ctx, 0, svc.logger, nil, false)
	svc.updateStats(ctx, 0, svc.logger, &tracker, false)

	// More NOOP cases
	tracker.saveStats = true
	svc.updateStats(ctx, 0, svc.logger, nil, false)
	assert.True(t, tracker.saveStats)
	svc.updateStats(ctx, 0, svc.logger, nil, true)
	assert.True(t, tracker.saveStats)

	// Populate a batch of data
	hostIDs := []uint{}
	queryID := uint(1)
	myHostID := uint(10000)
	myWallTime := uint64(5)
	myUserTime := uint64(6)
	mySystemTime := uint64(7)
	myMemory := uint64(8)
	myOutputSize := uint64(9)
	tracker.stats = append(
		tracker.stats, statsToSave{
			hostID: myHostID,
			Stats: &fleet.Stats{
				WallTimeMs: myWallTime,
				UserTime:   myUserTime,
				SystemTime: mySystemTime,
				Memory:     myMemory,
			},
			outputSize: myOutputSize,
		},
	)
	hostIDs = append(hostIDs, myHostID)

	for i := uint(1); i < statsBatchSize; i++ {
		tracker.stats = append(
			tracker.stats, statsToSave{
				hostID: i,
				Stats: &fleet.Stats{
					WallTimeMs: rand.Uint64(),
					UserTime:   rand.Uint64(),
					SystemTime: rand.Uint64(),
					Memory:     rand.Uint64(),
				},
				outputSize: rand.Uint64(),
			},
		)
		hostIDs = append(hostIDs, i)
	}
	tracker.saveStats = true
	// We overwrite the last executed time to ensure that these stats have a different timestamp than later stats
	overwriteLastExecuted = true
	overwriteLastExecutedTime = time.Now().Add(-2 * time.Second).Round(time.Second)
	svc.updateStats(ctx, queryID, svc.logger, &tracker, false)
	assert.True(t, tracker.saveStats)
	assert.Equal(t, 0, len(tracker.stats))
	assert.True(t, tracker.aggregationNeeded)
	assert.Equal(t, overwriteLastExecutedTime, tracker.lastStatsEntry.LastExecuted)

	// Aggregate stats
	svc.updateStats(ctx, queryID, svc.logger, &tracker, true)
	overwriteLastExecuted = false

	// Check that aggregated stats were created. Since we read aggregated stats from the replica, we may need to wait for it to catch up.
	var err error
	var aggStats fleet.AggregatedStats
	done := make(chan struct{}, 1)
	go func() {
		for {
			aggStats, err = mysql.GetAggregatedStats(ctx, svc.ds.(*mysql.Datastore), fleet.AggregatedStatsTypeScheduledQuery, queryID)
			if usingReplica && err != nil {
				time.Sleep(30 * time.Millisecond)
			} else {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-time.After(2 * time.Second):
		// We fail the test. Gather information for debug.
		// Grab stats from primary DB (master)
		lastErr := ""
		if err != nil {
			lastErr = err.Error()
		}
		aggStats, err = mysql.GetAggregatedStats(
			ctxdb.RequirePrimary(ctx, true), svc.ds.(*mysql.Datastore), fleet.AggregatedStatsTypeScheduledQuery, queryID,
		)
		if err != nil {
			t.Logf("Error getting aggregated stats from primary DB: %s", err.Error())
		} else {
			t.Logf("Aggregated stats from primary DB: totalExecutions=%f %#v", *aggStats.TotalExecutions, aggStats)
		}
		replicaStatus, err := ds.ReplicaStatus(ctx)
		assert.NoError(t, err)
		t.Logf("Replica status: %v", replicaStatus)
		t.Fatalf("Timeout waiting for aggregated stats. Last error: %s", lastErr)
	case <-done:
		// Continue
	}
	require.NoError(t, err)
	assert.Equal(t, statsBatchSize, int(*aggStats.TotalExecutions))
	// Sanity checks. Complete testing done in aggregated_stats_test.go
	assert.True(t, *aggStats.SystemTimeP50 > 0)
	assert.True(t, *aggStats.SystemTimeP95 > 0)
	assert.True(t, *aggStats.UserTimeP50 > 0)
	assert.True(t, *aggStats.UserTimeP95 > 0)

	// Get the stats from DB and make sure they match
	currentStats, err := svc.ds.GetLiveQueryStats(ctx, queryID, hostIDs)
	assert.NoError(t, err)
	assert.Equal(t, statsBatchSize, len(currentStats))
	currentStats, err = svc.ds.GetLiveQueryStats(ctx, queryID, []uint{myHostID})
	assert.NoError(t, err)
	require.Equal(t, 1, len(currentStats))
	myStat := currentStats[0]
	assert.Equal(t, myHostID, myStat.HostID)
	assert.Equal(t, uint64(1), myStat.Executions)
	assert.Equal(t, myWallTime, myStat.WallTime)
	assert.Equal(t, myUserTime, myStat.UserTime)
	assert.Equal(t, mySystemTime, myStat.SystemTime)
	assert.Equal(t, myMemory, myStat.AverageMemory)
	assert.Equal(t, myOutputSize, myStat.OutputSize)

	// Write new stats (update) for the same query/hosts
	myNewWallTime := uint64(15)
	myNewUserTime := uint64(16)
	myNewSystemTime := uint64(17)
	myNewMemory := uint64(18)
	myNewOutputSize := uint64(19)
	tracker.stats = append(
		tracker.stats, statsToSave{
			hostID: myHostID,
			Stats: &fleet.Stats{
				WallTimeMs: myNewWallTime,
				UserTime:   myNewUserTime,
				SystemTime: myNewSystemTime,
				Memory:     myNewMemory,
			},
			outputSize: myNewOutputSize,
		},
	)

	for i := uint(1); i < statsBatchSize; i++ {
		tracker.stats = append(
			tracker.stats, statsToSave{
				hostID: i,
				Stats: &fleet.Stats{
					WallTimeMs: rand.Uint64(),
					UserTime:   rand.Uint64() % 100, // Keep these values small to ensure the update will be noticeable
					SystemTime: rand.Uint64() % 100, // Keep these values small to ensure the update will be noticeable
					Memory:     rand.Uint64(),
				},
				outputSize: rand.Uint64(),
			},
		)
	}
	tracker.saveStats = true
	svc.updateStats(ctx, queryID, svc.logger, &tracker, true)
	assert.True(t, tracker.saveStats)
	assert.Equal(t, 0, len(tracker.stats))
	assert.False(t, tracker.aggregationNeeded)

	// Check that aggregated stats were updated. Since we read aggregated stats from the replica, we may need to wait for it to catch up.
	var newAggStats fleet.AggregatedStats
	done = make(chan struct{}, 1)
	go func() {
		for {
			newAggStats, err = mysql.GetAggregatedStats(ctx, svc.ds.(*mysql.Datastore), fleet.AggregatedStatsTypeScheduledQuery, queryID)
			if usingReplica && (*aggStats.SystemTimeP50 == *newAggStats.SystemTimeP50 ||
				*aggStats.SystemTimeP95 == *newAggStats.SystemTimeP95 ||
				*aggStats.UserTimeP50 == *newAggStats.UserTimeP50 ||
				*aggStats.UserTimeP95 == *newAggStats.UserTimeP95) {
				time.Sleep(30 * time.Millisecond)
			} else {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for aggregated stats")
	case <-done:
		// Continue
	}

	require.NoError(t, err)
	assert.Equal(t, statsBatchSize*2, int(*newAggStats.TotalExecutions))
	// Sanity checks. Complete testing done in aggregated_stats_test.go
	assert.True(t, *newAggStats.SystemTimeP50 > 0)
	assert.True(t, *newAggStats.SystemTimeP95 > 0)
	assert.True(t, *newAggStats.UserTimeP50 > 0)
	assert.True(t, *newAggStats.UserTimeP95 > 0)
	assert.NotEqual(t, *aggStats.SystemTimeP50, *newAggStats.SystemTimeP50)
	assert.NotEqual(t, *aggStats.SystemTimeP95, *newAggStats.SystemTimeP95)
	assert.NotEqual(t, *aggStats.UserTimeP50, *newAggStats.UserTimeP50)
	assert.NotEqual(t, *aggStats.UserTimeP95, *newAggStats.UserTimeP95)

	// Check that stats were updated
	currentStats, err = svc.ds.GetLiveQueryStats(ctx, queryID, []uint{myHostID})
	assert.NoError(t, err)
	require.Equal(t, 1, len(currentStats))
	myStat = currentStats[0]
	assert.Equal(t, myHostID, myStat.HostID)
	assert.Equal(t, uint64(2), myStat.Executions)
	assert.Equal(t, myWallTime+myNewWallTime, myStat.WallTime)
	assert.Equal(t, myUserTime+myNewUserTime, myStat.UserTime)
	assert.Equal(t, mySystemTime+myNewSystemTime, myStat.SystemTime)
	assert.Equal(t, (myMemory+myNewMemory)/2, myStat.AverageMemory)
	assert.Equal(t, myOutputSize+myNewOutputSize, myStat.OutputSize)
}

func TestUpdateStats(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer mysql.TruncateTables(t, ds)
	testUpdateStats(t, ds, false)
}

func TestIntegrationsUpdateStatsOnReplica(t *testing.T) {
	ds := mysql.CreateMySQLDSWithReplica(t, nil)
	defer mysql.TruncateTables(t, ds)
	testUpdateStats(t, ds, true)
}

func TestCalculateOutputSize(t *testing.T) {
	createResult := func() *fleet.DistributedQueryResult {
		result := fleet.DistributedQueryResult{}
		result.Rows = append(result.Rows, nil)
		result.Rows = append(result.Rows, map[string]string{})
		result.Rows = append(result.Rows, map[string]string{"a": "b", "a1": "b1"})
		result.Rows = append(result.Rows, map[string]string{"c": "d"})
		result.Stats = &fleet.Stats{}
		return &result
	}
	t.Run(
		"output size save disabled", func(t *testing.T) {
			tracker := statsTracker{saveStats: false}
			size := calculateOutputSize(&tracker, createResult())
			require.Equal(t, uint64(0), size)
		},
	)
	t.Run(
		"output size empty", func(t *testing.T) {
			tracker := statsTracker{saveStats: true}
			size := calculateOutputSize(&tracker, &fleet.DistributedQueryResult{})
			require.Equal(t, uint64(0), size)
		},
	)
	t.Run(
		"output size calculate", func(t *testing.T) {
			tracker := statsTracker{saveStats: true}
			size := calculateOutputSize(&tracker, createResult())
			expected := uint64(8) // manually calculated
			require.Equal(t, expected, size)
		},
	)
}
