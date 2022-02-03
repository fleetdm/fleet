package mysql

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargets(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CountHosts", testTargetsCountHosts},
		{"HostStatus", testTargetsHostStatus},
		{"HostIDsInTargets", testTargetsHostIDsInTargets},
		{"HostIDsInTargetsTeam", testTargetsHostIDsInTargetsTeam},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testTargetsCountHosts(t *testing.T, ds *Datastore) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	mockClock := clock.NewMockClock()

	hostCount := 0
	initHost := func(seenTime time.Time, distributedInterval uint, configTLSRefresh uint, teamID *uint) *fleet.Host {
		hostCount += 1
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			OsqueryHostID:       strconv.Itoa(hostCount),
			DetailUpdatedAt:     mockClock.Now(),
			LabelUpdatedAt:      mockClock.Now(),
			PolicyUpdatedAt:     mockClock.Now(),
			SeenTime:            mockClock.Now(),
			NodeKey:             strconv.Itoa(hostCount),
			DistributedInterval: distributedInterval,
			ConfigTLSRefresh:    configTLSRefresh,
			TeamID:              teamID,
		})
		require.NoError(t, err)
		require.NoError(t, ds.MarkHostsSeen(context.Background(), []uint{h.ID}, seenTime))
		return h
	}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	h1 := initHost(mockClock.Now().Add(-1*time.Second), 10, 60, &team1.ID)
	h2 := initHost(mockClock.Now().Add(-1*time.Hour), 30, 7200, &team2.ID)
	h3 := initHost(mockClock.Now().Add(-5*time.Second), 20, 20, &team2.ID)
	h4 := initHost(mockClock.Now().Add(-127*time.Second), 10, 10, &team2.ID)
	h5 := initHost(mockClock.Now(), 5, 5, nil)
	const thirtyDaysAndAMinuteAgo = -1 * (30*24*60 + 1)
	h6 := initHost(mockClock.Now().Add(thirtyDaysAndAMinuteAgo*time.Minute), 3600, 3600, nil)

	l1 := fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query foo",
	}
	l2 := fleet.LabelSpec{
		ID:    2,
		Name:  "label bar",
		Query: "query bar",
	}
	require.NoError(t, ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{&l1, &l2}))

	for _, h := range []*fleet.Host{h1, h2, h3, h6} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, mockClock.Now(), false)
		assert.Nil(t, err)
	}

	for _, h := range []*fleet.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l2.ID: ptr.Bool(true)}, mockClock.Now(), false)
		assert.Nil(t, err)
	}

	metrics, err := ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, LabelIDs: []uint{l1.ID, l2.ID}, TeamIDs: []uint{team1.ID, team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(2), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OnlineHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{}, LabelIDs: []uint{}, TeamIDs: []uint{}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team1.ID, team3.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	// Advance clock so all hosts are offline
	mockClock.AddTime(2 * time.Minute)
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(5), metrics.OfflineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	// observer not included
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)

	// observer included
	filter.IncludeObserver = true
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)

	userTeam2 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam2}

	// user can see team 2 which is associated with 3 hosts
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)

	// request team1, user cannot see it
	filter.TeamID = &team1.ID
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)

	// request team2, ok
	filter.TeamID = &team2.ID
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)
}

func testTargetsHostStatus(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)

	mockClock := clock.NewMockClock()

	h, err := ds.EnrollHost(context.Background(), "1", "key1", nil, 0)
	require.Nil(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	// Make host no longer appear new
	mockClock.AddTime(36 * time.Hour)

	expectOnline := fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}
	expectOffline := fleet.TargetMetrics{TotalHosts: 1, OfflineHosts: 1}
	expectMIA := fleet.TargetMetrics{TotalHosts: 1, MissingInActionHosts: 1}

	testCases := []struct {
		seenTime            time.Time
		distributedInterval uint
		configTLSRefresh    uint
		metrics             fleet.TargetMetrics
	}{
		{mockClock.Now().Add(-30 * time.Second), 10, 3600, expectOnline},
		{mockClock.Now().Add(-125 * time.Second), 10, 3600, expectOffline},
		{mockClock.Now().Add(-30 * time.Second), 3600, 10, expectOnline},
		{mockClock.Now().Add(-125 * time.Second), 3600, 10, expectOffline},

		{mockClock.Now().Add(-70 * time.Second), 60, 60, expectOnline},
		{mockClock.Now().Add(-121 * time.Second), 60, 60, expectOffline},

		{mockClock.Now().Add(-1 * time.Second), 10, 10, expectOnline},
		{mockClock.Now().Add(-2 * time.Minute), 10, 10, expectOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 10, 10, expectMIA},

		// Ensure behavior is reasonable if we don't have the values
		{mockClock.Now().Add(-1 * time.Second), 0, 0, expectOnline},
		{mockClock.Now().Add(-2 * time.Minute), 0, 0, expectOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 0, 0, expectMIA},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			// Save interval values
			h.DistributedInterval = tt.distributedInterval
			h.ConfigTLSRefresh = tt.configTLSRefresh
			require.NoError(t, ds.SaveHost(context.Background(), h))

			// Mark seen
			require.NoError(t, ds.MarkHostsSeen(context.Background(), []uint{h.ID}, tt.seenTime))

			// Verify status
			metrics, err := ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h.ID}}, mockClock.Now())
			require.NoError(t, err)
			assert.Equal(t, tt.metrics, metrics)
		})
	}
}

func testTargetsHostIDsInTargets(t *testing.T, ds *Datastore) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	hostCount := 0
	initHost := func() *fleet.Host {
		hostCount += 1
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			OsqueryHostID:   strconv.Itoa(hostCount),
			NodeKey:         strconv.Itoa(hostCount),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.Nil(t, err)
		return h
	}

	h1 := initHost()
	h2 := initHost()
	h3 := initHost()
	h4 := initHost()
	h5 := initHost()
	h6 := initHost()

	l1 := fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query foo",
	}
	l2 := fleet.LabelSpec{
		ID:    2,
		Name:  "label bar",
		Query: "query bar",
	}
	err := ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{&l1, &l2})
	require.Nil(t, err)

	for _, h := range []*fleet.Host{h1, h2, h3, h6} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		assert.Nil(t, err)
	}

	for _, h := range []*fleet.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l2.ID: ptr.Bool(true)}, time.Now(), false)
		assert.Nil(t, err)
	}

	ids, err := ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 2, 3, 4, 5, 6}, ids)

	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1}, ids)

	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l1.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 2, 3, 6}, ids)

	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{4}, LabelIDs: []uint{l1.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 2, 3, 4, 6}, ids)

	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{4}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{3, 4, 5}, ids)

	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{3, 4, 5}, ids)

	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{1, 6}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 3, 4, 5, 6}, ids)

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	// observer not included
	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{1, 6}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Len(t, ids, 0)

	// observer included
	filter.IncludeObserver = true
	ids, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{1, 6}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Len(t, ids, 5)
}

func testTargetsHostIDsInTargetsTeam(t *testing.T, ds *Datastore) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	mockClock := clock.NewMockClock()

	hostCount := 0
	initHost := func(seenTime time.Time, distributedInterval uint, configTLSRefresh uint, teamID *uint) *fleet.Host {
		hostCount += 1
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			OsqueryHostID:       strconv.Itoa(hostCount),
			DetailUpdatedAt:     mockClock.Now(),
			LabelUpdatedAt:      mockClock.Now(),
			PolicyUpdatedAt:     mockClock.Now(),
			SeenTime:            mockClock.Now(),
			NodeKey:             strconv.Itoa(hostCount),
			DistributedInterval: distributedInterval,
			ConfigTLSRefresh:    configTLSRefresh,
			TeamID:              teamID,
		})
		require.NoError(t, err)
		require.NoError(t, ds.MarkHostsSeen(context.Background(), []uint{h.ID}, seenTime))
		return h
	}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	h1 := initHost(mockClock.Now().Add(-1*time.Second), 10, 60, &team1.ID)
	initHost(mockClock.Now().Add(-1*time.Second), 10, 60, &team2.ID)

	targets, err := ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team1.ID}})
	require.NoError(t, err)
	assert.Equal(t, []uint{h1.ID}, targets)

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}

	// user can only see team1
	targets, err = ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team1.ID, team2.ID}})
	require.NoError(t, err)
	assert.Equal(t, []uint{h1.ID}, targets)
}
