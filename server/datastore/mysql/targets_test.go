package mysql

import (
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

func TestCountHostsInTargets(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	mockClock := clock.NewMockClock()

	hostCount := 0
	initHost := func(seenTime time.Time, distributedInterval uint, configTLSRefresh uint, teamID *uint) *fleet.Host {
		hostCount += 1
		h, err := ds.NewHost(&fleet.Host{
			OsqueryHostID:       strconv.Itoa(hostCount),
			DetailUpdatedAt:     mockClock.Now(),
			LabelUpdatedAt:      mockClock.Now(),
			SeenTime:            mockClock.Now(),
			NodeKey:             strconv.Itoa(hostCount),
			DistributedInterval: distributedInterval,
			ConfigTLSRefresh:    configTLSRefresh,
			TeamID:              teamID,
		})
		require.Nil(t, err)
		require.Nil(t, ds.MarkHostSeen(h, seenTime))
		return h
	}

	team1, err := ds.NewTeam(&fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(&fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(&fleet.Team{Name: "team3"})
	require.NoError(t, err)

	h1 := initHost(mockClock.Now().Add(-1*time.Second), 10, 60, &team1.ID)
	h2 := initHost(mockClock.Now().Add(-1*time.Hour), 30, 7200, &team2.ID)
	h3 := initHost(mockClock.Now().Add(-5*time.Second), 20, 20, &team2.ID)
	h4 := initHost(mockClock.Now().Add(-47*time.Second), 10, 10, &team2.ID)
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
	require.NoError(t, ds.ApplyLabelSpecs([]*fleet.LabelSpec{&l1, &l2}))

	for _, h := range []*fleet.Host{h1, h2, h3, h6} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, mockClock.Now())
		assert.Nil(t, err)
	}

	for _, h := range []*fleet.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l2.ID: true}, mockClock.Now())
		assert.Nil(t, err)
	}

	metrics, err := ds.CountHostsInTargets(filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, LabelIDs: []uint{l1.ID, l2.ID}, TeamIDs: []uint{team1.ID, team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(2), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OnlineHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{}, LabelIDs: []uint{}, TeamIDs: []uint{}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{TeamIDs: []uint{team1.ID, team3.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	// Advance clock so all hosts are offline
	mockClock.AddTime(2 * time.Minute)
	metrics, err = ds.CountHostsInTargets(filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(5), metrics.OfflineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

}

func TestHostStatus(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	test.AddAllHostsLabel(t, ds)

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	mockClock := clock.NewMockClock()

	h, err := ds.EnrollHost("1", "key1", nil, 0)
	require.Nil(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	// Make host no longer appear new
	mockClock.AddTime(36 * time.Hour)

	expectOnline := fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}
	expectOffline := fleet.TargetMetrics{TotalHosts: 1, OfflineHosts: 1}
	expectMIA := fleet.TargetMetrics{TotalHosts: 1, MissingInActionHosts: 1}

	var testCases = []struct {
		seenTime            time.Time
		distributedInterval uint
		configTLSRefresh    uint
		metrics             fleet.TargetMetrics
	}{
		{mockClock.Now().Add(-30 * time.Second), 10, 3600, expectOnline},
		{mockClock.Now().Add(-45 * time.Second), 10, 3600, expectOffline},
		{mockClock.Now().Add(-30 * time.Second), 3600, 10, expectOnline},
		{mockClock.Now().Add(-45 * time.Second), 3600, 10, expectOffline},

		{mockClock.Now().Add(-70 * time.Second), 60, 60, expectOnline},
		{mockClock.Now().Add(-91 * time.Second), 60, 60, expectOffline},

		{mockClock.Now().Add(-1 * time.Second), 10, 10, expectOnline},
		{mockClock.Now().Add(-1 * time.Minute), 10, 10, expectOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 10, 10, expectMIA},

		// Ensure behavior is reasonable if we don't have the values
		{mockClock.Now().Add(-1 * time.Second), 0, 0, expectOnline},
		{mockClock.Now().Add(-1 * time.Minute), 0, 0, expectOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 0, 0, expectMIA},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			// Save interval values
			h.DistributedInterval = tt.distributedInterval
			h.ConfigTLSRefresh = tt.configTLSRefresh
			require.Nil(t, ds.SaveHost(h))

			// Mark seen
			require.Nil(t, ds.MarkHostSeen(h, tt.seenTime))

			// Verify status
			metrics, err := ds.CountHostsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h.ID}}, mockClock.Now())
			require.Nil(t, err)
			assert.Equal(t, tt.metrics, metrics)
		})
	}
}

func TestHostIDsInTargets(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	hostCount := 0
	initHost := func() *fleet.Host {
		hostCount += 1
		h, err := ds.NewHost(&fleet.Host{
			OsqueryHostID:   strconv.Itoa(hostCount),
			NodeKey:         strconv.Itoa(hostCount),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
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
	err := ds.ApplyLabelSpecs([]*fleet.LabelSpec{&l1, &l2})
	require.Nil(t, err)

	for _, h := range []*fleet.Host{h1, h2, h3, h6} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	for _, h := range []*fleet.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l2.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	ids, err := ds.HostIDsInTargets(filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 2, 3, 4, 5, 6}, ids)

	ids, err = ds.HostIDsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1}, ids)

	ids, err = ds.HostIDsInTargets(filter, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l1.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 2, 3, 6}, ids)

	ids, err = ds.HostIDsInTargets(filter, fleet.HostTargets{HostIDs: []uint{4}, LabelIDs: []uint{l1.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 2, 3, 4, 6}, ids)

	ids, err = ds.HostIDsInTargets(filter, fleet.HostTargets{HostIDs: []uint{4}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{3, 4, 5}, ids)

	ids, err = ds.HostIDsInTargets(filter, fleet.HostTargets{HostIDs: []uint{}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{3, 4, 5}, ids)

	ids, err = ds.HostIDsInTargets(filter, fleet.HostTargets{HostIDs: []uint{1, 6}, LabelIDs: []uint{l2.ID}})
	require.Nil(t, err)
	assert.Equal(t, []uint{1, 3, 4, 5, 6}, ids)
}
