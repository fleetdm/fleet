package mysql

import (
	"context"
	"fmt"
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
			OsqueryHostID:       ptr.String(strconv.Itoa(hostCount)),
			DetailUpdatedAt:     mockClock.Now(),
			LabelUpdatedAt:      mockClock.Now(),
			PolicyUpdatedAt:     mockClock.Now(),
			SeenTime:            mockClock.Now(),
			NodeKey:             ptr.String(strconv.Itoa(hostCount)),
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
		Name:  "label foo",
		Query: "query foo",
	}
	l2 := fleet.LabelSpec{
		Name:  "label bar",
		Query: "query bar",
	}
	require.NoError(t, ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{&l1, &l2}))
	l1.ID = labelIDFromName(t, ds, l1.Name)
	l2.ID = labelIDFromName(t, ds, l2.Name)

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
	assert.Equal(t, uint(3), metrics.OfflineHosts) // metrics.MissingInActionHosts are also included in offline hosts as of Fleet 4.15
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(3), metrics.OfflineHosts) // metrics.MissingInActionHosts are also included in offline hosts as of Fleet 4.15
	assert.Equal(t, uint(3), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID}, LabelIDs: []uint{l1.ID, l2.ID}, TeamIDs: []uint{team1.ID, team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts) // metrics.MissingInActionHosts are also included in offline hosts as of Fleet 4.15
	assert.Equal(t, uint(2), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

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

	// Get 'No team' hosts
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{0}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(2), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(
		context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team1.ID, team3.ID, 0}}, mockClock.Now(),
	)
	require.Nil(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

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

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID}, TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(2), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team3.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}, TeamIDs: []uint{team3.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID}, TeamIDs: []uint{team1.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l2.ID}, TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(2), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l2.ID}, TeamIDs: []uint{team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(3), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{TeamIDs: []uint{team1.ID, team2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(2), metrics.OnlineHosts)
	assert.Equal(t, uint(2), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	// Advance clock so all hosts are offline
	mockClock.AddTime(2 * time.Minute)
	metrics, err = ds.CountHostsInTargets(context.Background(), filter, fleet.HostTargets{LabelIDs: []uint{l1.ID, l2.ID}}, mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(6), metrics.OfflineHosts) // metrics.MissingInActionHosts are also included in offline hosts as of Fleet 4.15
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

	h, err := ds.EnrollHost(context.Background(), false, "1", "", "", "key1", nil, 0)
	require.NoError(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	// Make host no longer appear new
	mockClock.AddTime(36 * time.Hour)

	expectOnline := fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}
	expectOffline := fleet.TargetMetrics{TotalHosts: 1, OfflineHosts: 1}
	expectMIA := fleet.TargetMetrics{
		TotalHosts:           1,
		OfflineHosts:         1, // MissingInActionHosts are also included in offline hosts as of Fleet 4.15
		MissingInActionHosts: 1,
	}

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
			require.NoError(t, ds.UpdateHost(context.Background(), h))

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
	initHost := func(teamID *uint, platform string) *fleet.Host {
		hostCount += 1
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			OsqueryHostID:   ptr.String(strconv.Itoa(hostCount)),
			NodeKey:         ptr.String(strconv.Itoa(hostCount)),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			TeamID:          teamID,
			Platform:        platform,
		})
		require.Nil(t, err)
		return h
	}

	// Run MigrateData to populate built-in labels.
	err := ds.MigrateData(context.Background())
	require.NoError(t, err)

	t1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	t2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	t3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	h1 := initHost(&t1.ID, "darwin")
	h2 := initHost(&t2.ID, "centos")
	h3 := initHost(&t2.ID, "ubuntu")
	h4 := initHost(&t2.ID, "windows")
	h5 := initHost(nil, "windows")
	h6 := initHost(nil, "darwin")

	// Load and record results for builtin labels.
	allHosts, _, err := ds.Label(context.Background(), 6, filter)
	require.NoError(t, err)
	macOS, _, err := ds.Label(context.Background(), 7, filter)
	require.NoError(t, err)
	ubuntuLinux, _, err := ds.Label(context.Background(), 8, filter)
	require.NoError(t, err)
	centOSLinux, _, err := ds.Label(context.Background(), 9, filter)
	require.NoError(t, err)
	msWindows, _, err := ds.Label(context.Background(), 10, filter)
	require.NoError(t, err)
	redHatLinux, _, err := ds.Label(context.Background(), 11, filter)
	require.NoError(t, err)
	allLinux, _, err := ds.Label(context.Background(), 12, filter)
	require.NoError(t, err)

	allBuiltIn := []*fleet.Label{
		allHosts, macOS, ubuntuLinux, centOSLinux, msWindows, redHatLinux, allLinux,
	}
	for _, item := range []struct {
		host   *fleet.Host
		labels map[*fleet.Label]struct{}
	}{
		{
			host: h1,
			labels: map[*fleet.Label]struct{}{
				allHosts: {},
				macOS:    {},
			},
		},
		{
			host: h2,
			labels: map[*fleet.Label]struct{}{
				allHosts:    {},
				centOSLinux: {},
				allLinux:    {},
			},
		},
		{
			host: h3,
			labels: map[*fleet.Label]struct{}{
				allHosts:    {},
				ubuntuLinux: {},
				allLinux:    {},
			},
		},
		{
			host: h4,
			labels: map[*fleet.Label]struct{}{
				allHosts:  {},
				msWindows: {},
			},
		},
		{
			host: h5,
			labels: map[*fleet.Label]struct{}{
				allHosts:  {},
				msWindows: {},
			},
		},
		{
			host: h6,
			labels: map[*fleet.Label]struct{}{
				allHosts: {},
				macOS:    {},
			},
		},
	} {
		for _, label := range allBuiltIn {
			value := false
			if _, ok := item.labels[label]; ok {
				value = true
			}
			err := ds.RecordLabelQueryExecutions(context.Background(), item.host, map[uint]*bool{label.ID: ptr.Bool(value)}, time.Now(), false)
			require.NoError(t, err)
		}
	}

	// Create and record results for custom labels.
	l1, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:      "label foo",
		Query:     "query foo",
		LabelType: fleet.LabelTypeRegular,
	})
	require.NoError(t, err)
	l2, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:      "label bar",
		Query:     "query bar",
		LabelType: fleet.LabelTypeRegular,
	})
	require.NoError(t, err)
	l3, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:      "label zoo",
		Query:     "query zoo",
		LabelType: fleet.LabelTypeRegular,
	})
	require.NoError(t, err)

	for _, h := range []*fleet.Host{h1, h2, h3, h6} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}
	for _, h := range []*fleet.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l2.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}
	for _, h := range []*fleet.Host{h1, h2, h3, h4, h5, h6} {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l3.ID: ptr.Bool(false)}, time.Now(), false)
		require.NoError(t, err)
	}

	// Scenario:
	//
	// Expected behavior:
	// Any target selected within one of the two categories "labels" and "teams" is understood as a union.
	// But the combination of the categories is understood as an intersection.
	//
	// Builtin labels (aka Platforms):
	//	- allHosts: { h1, h2, h3, h4, h5, h6 }
	//	- macOS: { h1, h6 }
	//	- ubuntuLinux: { h3 }
	//	- centOSLinux: { h2 }
	//	- msWindows: { h4, h5 }
	//	- redHatLinux: { }
	//	- allLinux: { h2, h3 }
	//
	// Custom labels (non-builtin labels):
	// 	- Label l1: { h1, h2, h3, h6 }
	// 	- Label l2: { h3, h4, h5 }
	// 	- Label l3: { }
	//
	// Teams:
	// 	- Team t1: { h1 }
	// 	- Team t2: { h2, h3, h4 }
	// 	- Team t3: { }
	//
	// - Hosts h5, h6 are global.

	for i, tc := range []struct {
		name string

		targetHostIDs  []uint
		targetLabelIDs []uint
		targetTeamIDs  []uint

		expectedHostIDs []uint
	}{
		{
			name:           "All hosts only",
			targetLabelIDs: []uint{allHosts.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID, h5.ID, h6.ID},
		},
		{
			name:           "The two labels should return all hosts",
			targetLabelIDs: []uint{l1.ID, l2.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID, h5.ID, h6.ID},
		},
		{
			name:           "All hosts should always return all hosts",
			targetLabelIDs: []uint{l1.ID, allHosts.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID, h5.ID, h6.ID},
		},
		{
			name:           "All hosts should always return all hosts (with empty label)",
			targetLabelIDs: []uint{l3.ID, allHosts.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID, h5.ID, h6.ID},
		},
		{
			name:          "One host only",
			targetHostIDs: []uint{h1.ID},

			expectedHostIDs: []uint{h1.ID},
		},
		{
			name:           "One host and a label the host is member of",
			targetHostIDs:  []uint{h1.ID},
			targetLabelIDs: []uint{l1.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h6.ID},
		},
		{
			name:           "One host and a label the host is not member of",
			targetHostIDs:  []uint{h4.ID},
			targetLabelIDs: []uint{l1.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID, h6.ID},
		},
		{
			name:           "One host, a label the host is member of and a platform the host is member of",
			targetHostIDs:  []uint{h4.ID},
			targetLabelIDs: []uint{l2.ID, msWindows.ID},

			expectedHostIDs: []uint{h4.ID, h5.ID},
		},
		{
			name:           "Label 2 and a platform none of the label 2 hosts is",
			targetLabelIDs: []uint{l2.ID, macOS.ID},

			expectedHostIDs: nil,
		},
		{
			name:           "Host selection + custom label selection + platform selection",
			targetHostIDs:  []uint{h1.ID},
			targetLabelIDs: []uint{l1.ID, macOS.ID},

			expectedHostIDs: []uint{h1.ID, h6.ID},
		},
		{
			name:           "Host selection + custom label selection + platform selection + team selection",
			targetHostIDs:  []uint{h1.ID},
			targetLabelIDs: []uint{l1.ID, allLinux.ID},
			targetTeamIDs:  []uint{t1.ID, t2.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID},
		},
		{
			name:          "Host selection + team selection",
			targetHostIDs: []uint{h1.ID, h2.ID},
			targetTeamIDs: []uint{t1.ID, t2.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID},
		},
		{
			name:            "No selection",
			expectedHostIDs: []uint{},
		},
		{
			name:            "'No Team' team selection",
			targetTeamIDs:   []uint{0},
			expectedHostIDs: []uint{h5.ID, h6.ID},
		},
		{
			name:            "'No Team' team, one team, and an empty team selection",
			targetTeamIDs:   []uint{t1.ID, 0, t3.ID},
			expectedHostIDs: []uint{h1.ID, h5.ID, h6.ID},
		},
		{
			name:          "One team and an empty team selection",
			targetTeamIDs: []uint{t1.ID, t3.ID},

			expectedHostIDs: []uint{h1.ID},
		},
		{
			name:          "One team selection",
			targetTeamIDs: []uint{t2.ID},

			expectedHostIDs: []uint{h2.ID, h3.ID, h4.ID},
		},
		{
			name:           "One non-builtin label and team selection",
			targetLabelIDs: []uint{l1.ID},
			targetTeamIDs:  []uint{t2.ID},

			expectedHostIDs: []uint{h2.ID, h3.ID},
		},
		{
			name:          "Empty team selection",
			targetTeamIDs: []uint{t3.ID},

			expectedHostIDs: nil,
		},
		{
			name:           "Two labels and an empty team",
			targetLabelIDs: []uint{l1.ID, l2.ID},
			targetTeamIDs:  []uint{t3.ID},

			expectedHostIDs: nil,
		},
		{
			name:           "Empty label selection",
			targetLabelIDs: []uint{l3.ID},

			expectedHostIDs: nil,
		},
		{
			name:           "Empty label and two teams should select no hosts",
			targetLabelIDs: []uint{l3.ID},
			targetTeamIDs:  []uint{t1.ID, t2.ID},

			expectedHostIDs: nil,
		},
		{
			name:           "Label and team intersection",
			targetLabelIDs: []uint{l1.ID},
			targetTeamIDs:  []uint{t1.ID},

			expectedHostIDs: []uint{h1.ID},
		},
		{
			name:           "Another label and team intersection",
			targetLabelIDs: []uint{l2.ID},
			targetTeamIDs:  []uint{t2.ID},

			expectedHostIDs: []uint{h3.ID, h4.ID},
		},
		{
			name:           "Host selection + non-builtin label selection + team selection",
			targetHostIDs:  []uint{h1.ID},
			targetLabelIDs: []uint{l2.ID},
			targetTeamIDs:  []uint{t2.ID},

			expectedHostIDs: []uint{h1.ID, h3.ID, h4.ID},
		},
		{
			name:          "Two teams selection",
			targetTeamIDs: []uint{t1.ID, t2.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID},
		},
		{
			name:           "Two platform labels",
			targetLabelIDs: []uint{centOSLinux.ID, ubuntuLinux.ID},

			expectedHostIDs: []uint{h2.ID, h3.ID},
		},
		{
			name:           "Two platform labels and a custom label",
			targetLabelIDs: []uint{centOSLinux.ID, ubuntuLinux.ID, l2.ID},

			expectedHostIDs: []uint{h3.ID},
		},
		{
			name:           "All platforms and all labels should return all hosts",
			targetLabelIDs: []uint{macOS.ID, ubuntuLinux.ID, centOSLinux.ID, msWindows.ID, redHatLinux.ID, l1.ID, l2.ID},

			expectedHostIDs: []uint{h1.ID, h2.ID, h3.ID, h4.ID, h5.ID, h6.ID},
		},
		{
			name:           "All platforms but one and all labels should return all hosts but one",
			targetLabelIDs: []uint{macOS.ID, ubuntuLinux.ID, msWindows.ID, redHatLinux.ID, l1.ID, l2.ID},

			expectedHostIDs: []uint{h1.ID, h3.ID, h4.ID, h5.ID, h6.ID},
		},
	} {
		t.Run(fmt.Sprintf("%d.%s", i, tc.name), func(t *testing.T) {
			targets := fleet.HostTargets{
				HostIDs:  tc.targetHostIDs,
				LabelIDs: tc.targetLabelIDs,
				TeamIDs:  tc.targetTeamIDs,
			}
			ids, err := ds.HostIDsInTargets(context.Background(), filter, targets)
			require.NoError(t, err)
			require.Equal(t, tc.expectedHostIDs, ids)

			metrics, err := ds.CountHostsInTargets(context.Background(), filter, targets, time.Now())
			require.NoError(t, err)
			require.Len(t, tc.expectedHostIDs, int(metrics.TotalHosts)) //nolint:gosec // dismiss G115
		})
	}

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	// observer not included
	ids, err := ds.HostIDsInTargets(context.Background(), filter, fleet.HostTargets{HostIDs: []uint{1, 6}, LabelIDs: []uint{l2.ID}})
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
			OsqueryHostID:       ptr.String(strconv.Itoa(hostCount)),
			DetailUpdatedAt:     mockClock.Now(),
			LabelUpdatedAt:      mockClock.Now(),
			PolicyUpdatedAt:     mockClock.Now(),
			SeenTime:            mockClock.Now(),
			NodeKey:             ptr.String(strconv.Itoa(hostCount)),
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
