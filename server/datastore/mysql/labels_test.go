package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchHostnamesSmall(t *testing.T) {
	small := []string{"foo", "bar", "baz"}
	batched := batchHostnames(small)
	require.Equal(t, 1, len(batched))
	assert.Equal(t, small, batched[0])
}

func TestBatchHostnamesLarge(t *testing.T) {
	large := []string{}
	for i := range 110_000 {
		large = append(large, strconv.Itoa(i))
	}
	batched := batchHostnames(large)
	require.Equal(t, 8, len(batched))
	assert.Equal(t, large[:15_000], batched[0])
	assert.Equal(t, large[15_000:30_000], batched[1])
	assert.Equal(t, large[30_000:45_000], batched[2])
	assert.Equal(t, large[45_000:60_000], batched[3])
	assert.Equal(t, large[60_000:75_000], batched[4])
	assert.Equal(t, large[75_000:90_000], batched[5])
	assert.Equal(t, large[90_000:105_000], batched[6])
	assert.Equal(t, large[105_000:110_000], batched[7])
}

func TestBatchHostIdsSmall(t *testing.T) {
	small := []uint{1, 2, 3}
	batched := batchHostIds(small)
	require.Equal(t, 1, len(batched))
	assert.Equal(t, small, batched[0])
}

func TestBatchHostIdsLarge(t *testing.T) {
	large := []uint{}
	for i := 0; i < 230000; i++ {
		large = append(large, uint(i)) //nolint:gosec // dismiss G115
	}
	batched := batchHostIds(large)
	require.Equal(t, 5, len(batched))
	assert.Equal(t, large[:50000], batched[0])
	assert.Equal(t, large[50000:100000], batched[1])
	assert.Equal(t, large[100000:150000], batched[2])
	assert.Equal(t, large[150000:200000], batched[3])
	assert.Equal(t, large[200000:230000], batched[4])
}

func TestLabels(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"AddAllHostsDeferred", func(t *testing.T, ds *Datastore) { testLabelsAddAllHosts(true, t, ds) }},
		{"AddAllHostsNotDeferred", func(t *testing.T, ds *Datastore) { testLabelsAddAllHosts(false, t, ds) }},
		{"Search", testLabelsSearch},
		{"ListHostsInLabel", testLabelsListHostsInLabel},
		{"ListHostsInLabelAndStatus", testLabelsListHostsInLabelAndStatus},
		{"ListHostsInLabelAndTeamFilterDeferred", func(t *testing.T, ds *Datastore) { testLabelsListHostsInLabelAndTeamFilter(true, t, ds) }},
		{"ListHostsInLabelAndTeamFilterNotDeferred", func(t *testing.T, ds *Datastore) { testLabelsListHostsInLabelAndTeamFilter(false, t, ds) }},
		{"BuiltIn", testLabelsBuiltIn},
		{"ChangeDetails", testLabelsChangeDetails},
		{"GetSpec", testLabelsGetSpec},
		{"ApplySpecsRoundtrip", testLabelsApplySpecsRoundtrip},
		{"UpdateLabelMembershipByHostIDs", testUpdateLabelMembershipByHostIDs},
		{"IDsByName", testLabelsIDsByName},
		{"ByName", testLabelsByName},
		{"SingleByName", testLabelByName},
		{"Save", testLabelsSave},
		{"QueriesForCentOSHost", testLabelsQueriesForCentOSHost},
		{"RecordNonExistentQueryLabelExecution", testLabelsRecordNonexistentQueryLabelExecution},
		{"DeleteLabel", testDeleteLabel},
		{"LabelsSummaryAndListTeamFiltering", testLabelsSummaryAndListTeamFiltering},
		{"ListHostsInLabelIssues", testListHostsInLabelIssues},
		{"ListHostsInLabelDiskEncryptionStatus", testListHostsInLabelDiskEncryptionStatus},
		{"HostMemberOfAllLabels", testHostMemberOfAllLabels},
		{"ListHostsInLabelOSSettings", testLabelsListHostsInLabelOSSettings},
		{"AddDeleteLabelsToFromHost", testAddDeleteLabelsToFromHost},
		{"ApplyLabelSpecSerialUUID", testApplyLabelSpecsForSerialUUID},
		{"ApplyLabelSpecsWithPlatformChange", testApplyLabelSpecsWithPlatformChange},
		{"UpdateLabelMembershipByHostCriteria", testUpdateLabelMembershipByHostCriteria},
		{"TeamLabels", testTeamLabels},
		{"UpdateLabelMembershipForTransferredHost", testUpdateLabelMembershipForTransferredHost},
		{"SetAsideLabels", testSetAsideLabels},
		{"ApplyLabelSpecsWithManualTeamLabels", testApplyLabelSpecsWithManualTeamLabels},
	}
	// call TruncateTables first to remove migration-created labels
	TruncateTables(t, ds)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testLabelsAddAllHosts(deferred bool, t *testing.T, db *Datastore) {
	test.AddAllHostsLabel(t, db)
	hosts := []fleet.Host{}
	var host *fleet.Host
	var err error
	for i := 0; i < 10; i++ {
		host, err = db.EnrollOsquery(context.Background(),
			fleet.WithEnrollOsqueryHostID(fmt.Sprint(i)),
			fleet.WithEnrollOsqueryNodeKey(fmt.Sprint(i)),
		)
		require.Nil(t, err, "enrollment should succeed")
		hosts = append(hosts, *host)
	}

	host.Platform = "darwin"
	err = db.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	queries, err := db.LabelQueriesForHost(context.Background(), host)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	labels, err := db.ListLabelsForHost(context.Background(), host.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1) // all hosts only

	newLabels := []*fleet.LabelSpec{
		// Note these are intentionally out of order
		{
			Name:     "label3",
			Query:    "query3",
			Platform: "darwin",
		},
		{
			Name:  "label1",
			Query: "query1",
		},
		{
			Name:     "label2",
			Query:    "query2",
			Platform: "darwin",
		},
		{
			Name:     "label4",
			Query:    "query4",
			Platform: "darwin",
		},
	}
	err = db.ApplyLabelSpecs(context.Background(), newLabels)
	require.Nil(t, err)

	expectQueries := map[string]string{
		"2": "query3",
		"3": "query1",
		"4": "query2",
		"5": "query4",
	}

	host.Platform = "darwin"

	// Now queries should be returned
	queries, err = db.LabelQueriesForHost(context.Background(), host)
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// No labels should match with no results yet
	labels, err = db.ListLabelsForHost(context.Background(), host.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)

	baseTime := time.Now()

	// Record a query execution
	err = db.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{
		1: ptr.Bool(true), 2: ptr.Bool(false), 3: ptr.Bool(true), 4: ptr.Bool(false), 5: ptr.Bool(false),
	}, baseTime, deferred)
	assert.Nil(t, err)

	host, err = db.Host(context.Background(), host.ID)
	require.NoError(t, err)
	host.LabelUpdatedAt = baseTime

	// A new label targeting another platform should not affect the labels for
	// this host
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{
		{
			Name:     "label5",
			Platform: "not-matching",
			Query:    "query5",
		},
	})
	require.NoError(t, err)
	queries, err = db.LabelQueriesForHost(context.Background(), host)
	assert.Nil(t, err)
	assert.Len(t, queries, 4)

	// If a new label is added, all labels should be returned
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{
		{
			Name:     "label6",
			Platform: "",
			Query:    "query6",
		},
	})
	require.NoError(t, err)
	expectQueries["7"] = "query6"
	queries, err = db.LabelQueriesForHost(context.Background(), host)
	assert.Nil(t, err)
	assert.Len(t, queries, 5)

	// Only the 'All Hosts' label should apply for a host with no labels
	// executed.
	labels, err = db.ListLabelsForHost(context.Background(), hosts[0].ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)
}

func testLabelsSearch(t *testing.T, db *Datastore) {
	specs := []*fleet.LabelSpec{
		{ID: 1, Name: "foo"},
		{ID: 2, Name: "bar"},
		{ID: 3, Name: "foo-bar"},
		{ID: 4, Name: "bar2"},
		{ID: 5, Name: "bar3"},
		{ID: 6, Name: "bar4"},
		{ID: 7, Name: "bar5"},
		{ID: 8, Name: "bar6"},
		{ID: 9, Name: "bar7"},
		{ID: 10, Name: "bar8"},
		{ID: 11, Name: "bar9"},
		{
			ID:        12,
			Name:      "All Hosts",
			LabelType: fleet.LabelTypeBuiltIn,
		},
	}
	err := db.ApplyLabelSpecs(context.Background(), specs)
	require.Nil(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	all, _, err := db.Label(context.Background(), specs[len(specs)-1].ID, filter)
	require.Nil(t, err)
	l3, _, err := db.Label(context.Background(), specs[2].ID, filter)
	require.Nil(t, err)

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	labels, err := db.SearchLabels(context.Background(), filter, "")
	require.Nil(t, err)
	assert.Len(t, labels, 12)
	assert.Contains(t, labels, &all.Label)

	labels, err = db.SearchLabels(context.Background(), filter, "foo")
	require.Nil(t, err)
	assert.Len(t, labels, 3)
	assert.Contains(t, labels, &all.Label)

	labels, err = db.SearchLabels(context.Background(), filter, "foo", all.ID, l3.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "foo", labels[0].Name)

	labels, err = db.SearchLabels(context.Background(), filter, "xxx")
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Contains(t, labels, &all.Label)

	// Test team filtering
	team1, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// Create team labels
	team1Label, err := db.NewLabel(context.Background(), &fleet.Label{
		Name:                "team1-foo",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	team2Label, err := db.NewLabel(context.Background(), &fleet.Label{
		Name:                "team2-foo",
		Query:               "SELECT 2",
		TeamID:              &team2.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Global admin should see all labels (global + team labels), including the All Hosts label
	labels, err = db.SearchLabels(context.Background(), filter, "foo")
	require.NoError(t, err)
	assert.Len(t, labels, 5) // foo, foo-bar, All Hosts, team1-foo, team2-foo

	// Filter to team1 only - should see global labels + team1 labels
	team1Filter := fleet.TeamFilter{User: user, TeamID: &team1.ID}
	labels, err = db.SearchLabels(context.Background(), team1Filter, "foo")
	require.NoError(t, err)
	assert.Len(t, labels, 4) // foo, foo-bar, All Hosts, team1-foo
	foundTeam1Label := false
	foundTeam2Label := false
	for _, l := range labels {
		if l.ID == team1Label.ID {
			foundTeam1Label = true
		}
		if l.ID == team2Label.ID {
			foundTeam2Label = true
		}
	}
	assert.True(t, foundTeam1Label, "team1 label should be found")
	assert.False(t, foundTeam2Label, "team2 label should not be found")

	// Filter to global only (team_id = 0)
	globalOnlyFilter := fleet.TeamFilter{User: user, TeamID: ptr.Uint(0)}
	labels, err = db.SearchLabels(context.Background(), globalOnlyFilter, "foo")
	require.NoError(t, err)
	assert.Len(t, labels, 3) // foo, foo-bar, All Hosts
	for _, l := range labels {
		assert.Nil(t, l.TeamID, "should only have global labels")
	}

	// Team user can only see their team's labels + global labels
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserver}}}
	team1UserFilter := fleet.TeamFilter{User: team1User}
	labels, err = db.SearchLabels(context.Background(), team1UserFilter, "foo")
	require.NoError(t, err)
	assert.Len(t, labels, 4) // foo, foo-bar, All Hosts, team1-foo
	for _, l := range labels {
		if l.TeamID != nil {
			assert.Equal(t, team1.ID, *l.TeamID, "team user should only see their team's labels")
		}
	}

	// Team user trying to access another team's labels should fail
	team1UserTeam2Filter := fleet.TeamFilter{User: team1User, TeamID: &team2.ID}
	_, err = db.SearchLabels(context.Background(), team1UserTeam2Filter, "foo")
	require.ErrorContains(t, err, errInaccessibleTeam.Error()) // not ErrorIs due to UserError wrapping
}

func testLabelsListHostsInLabel(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		Platform:        "darwin",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
		Platform:        "darwin",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("3"),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		Hostname:        "baz.local",
		Platform:        "darwin",
	})
	require.Nil(t, err)
	require.NoError(t, db.SetOrUpdateHostDisksSpace(context.Background(), h1.ID, 10, 5, 200.0, nil))
	require.NoError(t, db.SetOrUpdateHostDisksSpace(context.Background(), h2.ID, 20, 10, 200.1, nil))
	require.NoError(t, db.SetOrUpdateHostDisksSpace(context.Background(), h3.ID, 30, 15, 200.2, nil))

	ctx := context.Background()
	const simpleMDM, kandji = "https://simplemdm.com", "https://kandji.io"
	err = db.SetOrUpdateMDMData(ctx, h1.ID, false, true, simpleMDM, true, fleet.WellKnownMDMSimpleMDM, "", false) // enrollment: automatic
	require.NoError(t, err)
	err = db.SetOrUpdateMDMData(ctx, h2.ID, false, true, kandji, true, fleet.WellKnownMDMKandji, "", false) // enrollment: automatic
	require.NoError(t, err)
	err = db.SetOrUpdateMDMData(ctx, h3.ID, false, false, simpleMDM, false, fleet.WellKnownMDMSimpleMDM, "", false) // enrollment: unenrolled
	require.NoError(t, err)

	var simpleMDMID uint
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &simpleMDMID, `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, simpleMDM)
	})
	var kandjiID uint
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &kandjiID, `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMKandji, kandji)
	})

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}

	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{}, 0)

	for _, h := range []*fleet.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{}, 3)

	hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(35), ListOptions: fleet.ListOptions{OrderKey: "id", After: "1"}}, 2)
	require.Equal(t, h2.ID, hosts[0].ID)
	require.Equal(t, h3.ID, hosts[1].ID)

	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(35)}, 3)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(25)}, 2)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(15)}, 1)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{LowDiskSpaceFilter: ptr.Int(5)}, 0)

	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{MDMIDFilter: ptr.Uint(99)}, 0)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{MDMIDFilter: ptr.Uint(simpleMDMID)}, 2)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{MDMIDFilter: ptr.Uint(kandjiID)}, 1)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMSimpleMDM)}, 2)
	listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{MDMNameFilter: ptr.String(fleet.WellKnownMDMSimpleMDM), MDMEnrollmentStatusFilter: fleet.MDMEnrollStatusEnrolled}, 1)

	// Test team label filtering
	team1, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team1_listhosts"})
	require.NoError(t, err)

	// Create a team label
	teamLabel, err := db.NewLabel(context.Background(), &fleet.Label{
		Name:                "team1-label",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Create a host on team1
	h4, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("4"),
		NodeKey:         ptr.String("4"),
		UUID:            "4",
		Hostname:        "team1host.local",
		Platform:        "darwin",
		TeamID:          &team1.ID,
	})
	require.NoError(t, err)

	// Add host to team label
	err = db.RecordLabelQueryExecutions(context.Background(), h4, map[uint]*bool{teamLabel.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)

	// Global admin can list hosts in team label
	listHostsInLabelCheckCount(t, db, filter, teamLabel.ID, fleet.HostListOptions{}, 1)

	// Team user can list hosts in their team's label
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserver}}}
	team1Filter := fleet.TeamFilter{User: team1User, IncludeObserver: true}
	listHostsInLabelCheckCount(t, db, team1Filter, teamLabel.ID, fleet.HostListOptions{}, 1)

	// Trying to list a team label that the user doesn't have access to returns empty
	team2, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team2_listhosts"})
	require.NoError(t, err)
	team2User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleObserver}}}
	team2Filter := fleet.TeamFilter{User: team2User, IncludeObserver: true}
	// Team2 user cannot see team1's label, so they get no results
	team2Hosts, err := db.ListHostsInLabel(context.Background(), team2Filter, teamLabel.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Nil(t, team2Hosts) // Returns nil when label is not accessible
}

func listHostsInLabelCheckCount(
	t *testing.T, db *Datastore, filter fleet.TeamFilter, labelID uint, opt fleet.HostListOptions, expectedCount int,
) []*fleet.Host {
	hosts, err := db.ListHostsInLabel(context.Background(), filter, labelID, opt)
	require.NoError(t, err)
	count, err := db.CountHostsInLabel(context.Background(), filter, labelID, opt)
	require.NoError(t, err)
	require.Equal(t, expectedCount, count)
	require.Len(t, hosts, expectedCount)
	return hosts
}

func testLabelsListHostsInLabelAndStatus(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	lastSeenTime := time.Now().Add(-1000 * time.Hour)
	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
		PolicyUpdatedAt: lastSeenTime,
		SeenTime:        lastSeenTime,
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
		Platform:        "darwin",
	})
	require.NoError(t, err)
	h3, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
		PolicyUpdatedAt: lastSeenTime,
		SeenTime:        lastSeenTime,
		OsqueryHostID:   ptr.String("3"),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		Hostname:        "baz.local",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	for _, h := range []*fleet.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		assert.Nil(t, err)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusOnline}, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusMIA}, 2)
		assert.Equal(t, "bar.local", hosts[0].Hostname)
		assert.Equal(t, "baz.local", hosts[1].Hostname)
	}
}

func testLabelsListHostsInLabelAndTeamFilter(deferred bool, t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		Platform:        "darwin",
	})
	require.Nil(t, err)

	lastSeenTime := time.Now().Add(-1000 * time.Hour)
	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
		PolicyUpdatedAt: lastSeenTime,
		SeenTime:        lastSeenTime,
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
		Platform:        "darwin",
	})
	require.Nil(t, err)

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	team1, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	team2, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	require.NoError(t, db.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{h1.ID})))

	userFilter := fleet.TeamFilter{User: test.UserAdmin}
	var teamIDFilterNil *uint                // "All teams" option should include all hosts regardless of team assignment
	var teamIDFilterZero *uint = ptr.Uint(0) // "No team" option should include only hosts that are not assigned to any team

	for _, h := range []*fleet.Host{h1, h2} {
		err = db.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), deferred)
		assert.Nil(t, err)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusOnline}, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusMIA}, 1)
		assert.Equal(t, "bar.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: &team1.ID}, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: &team2.ID}, 0)        // no hosts assigned to team 2
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: teamIDFilterZero}, 1) // h2 not assigned to any team
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: teamIDFilterNil}, 2)  // h1 and h2

	// test team filter in combination with macos settings filter
	nanoEnrollAndSetHostMDMData(t, db, h1, false)
	require.NoError(t, err)
	require.NoError(t, db.BulkUpsertMDMAppleHostProfiles(context.Background(), []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       "a" + uuid.NewString(),
			ProfileIdentifier: "identifier",
			HostUUID:          h1.UUID, // hosts[0] is assgined to team 1
			CommandUUID:       "command-uuid-1",
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryVerifying,
			Checksum:          []byte("csum"),
			Scope:             fleet.PayloadScopeSystem,
		},
	}))
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: &team1.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1) // h1
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: &team2.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0) // wrong team
	// macos settings filter does not support "all teams" so teamIDFilterNil acts the same as teamIDFilterZero
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: teamIDFilterZero, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0) // no team
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: teamIDFilterNil, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0)  // no team
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0)                               // no team

	nanoEnrollAndSetHostMDMData(t, db, h2, false)
	require.NoError(t, db.BulkUpsertMDMAppleHostProfiles(context.Background(), []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       "a" + uuid.NewString(),
			ProfileIdentifier: "identifier",
			HostUUID:          h2.UUID, // hosts[9] is assgined to no team
			CommandUUID:       "command-uuid-2",
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryVerifying,
			Checksum:          []byte("csum"),
			Scope:             fleet.PayloadScopeSystem,
		},
	}))
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: &team1.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1) // h1
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: &team2.ID, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 0) // wrong team
	// macos settings filter does not support "all teams" so both teamIDFilterNil acts the same as teamIDFilterZero
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: teamIDFilterZero, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1) // h2
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{TeamFilter: teamIDFilterNil, MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1)  // h2
	listHostsInLabelCheckCount(t, db, userFilter, l1.ID, fleet.HostListOptions{MacOSSettingsFilter: fleet.OSSettingsVerifying}, 1)                               // h2
}

func testLabelsBuiltIn(t *testing.T, db *Datastore) {
	require.Nil(t, db.MigrateData(context.Background()))

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	hits, err := db.SearchLabels(context.Background(), filter, "macOS")
	require.Nil(t, err)
	// Should get Mac OS X and All Hosts
	assert.Equal(t, 2, len(hits))
	assert.Equal(t, fleet.LabelTypeBuiltIn, hits[0].LabelType)
	assert.Equal(t, fleet.LabelTypeBuiltIn, hits[1].LabelType)
}

func testLabelsChangeDetails(t *testing.T, db *Datastore) {
	label := fleet.LabelSpec{
		ID:          1,
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes",
		Platform:    "darwin",
	}
	err := db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{&label})
	require.Nil(t, err)

	label.Description = "changed description"
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{&label})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	saved, _, err := db.Label(context.Background(), label.ID, filter)
	require.Nil(t, err)
	assert.Equal(t, label.Name, saved.Name)
	assert.Equal(t, label.Description, saved.Description)

	// Create an Apple config profile, which should reflect a change in label's name
	profA, err := db.NewMDMAppleConfigProfile(context.Background(), *generateAppleCP("a", "a", 0), nil)
	require.NoError(t, err)
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(),
			"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
			profA.ProfileUUID, label.Name, label.ID)
		return err
	})
	label.Name = "changed name"
	// ApplyLabelSpecs can't update the name -- it simply creates a new label, so we need to call SaveLabel.
	saved.Name = label.Name
	saved2, _, err := db.SaveLabel(context.Background(), &saved.Label, filter)
	require.NoError(t, err)
	assert.Equal(t, label.Name, saved2.Name)
	assert.Equal(t, label.Description, saved2.Description)

	var configProfileLabelName string
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &configProfileLabelName,
			"SELECT label_name FROM mdm_configuration_profile_labels WHERE label_id = ?", label.ID)
	})
	assert.Equal(t, label.Name, configProfileLabelName)
}

func setupLabelSpecsTest(t *testing.T, ds fleet.Datastore) []*fleet.LabelSpec {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(strconv.Itoa(i)),
			UUID:            fmt.Sprintf("uuid%s", strconv.Itoa(i)),
			Hostname:        fmt.Sprintf("host%s", strconv.Itoa(i)),
		})
		require.Nil(t, err)
	}

	expectedSpecs := []*fleet.LabelSpec{
		{
			Name:        "foo",
			Query:       "select * from foo",
			Description: "foo description",
			Platform:    "darwin",
		},
		{
			Name:  "bar",
			Query: "select * from bar",
		},
		{
			Name:  "bing",
			Query: "select * from bing",
		},
		{
			Name:                "All Hosts",
			Query:               "SELECT 1",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
		{
			Name:                "Manual Label",
			LabelMembershipType: fleet.LabelMembershipTypeManual,
			Hosts: []string{
				"1", "2", "3", "4",
			},
		},
	}
	err := ds.ApplyLabelSpecs(context.Background(), expectedSpecs)
	require.Nil(t, err)

	expectedSpecs[4].Hosts = []string{"1", "2", "3", "4"}
	return expectedSpecs
}

func testLabelsGetSpec(t *testing.T, ds *Datastore) {
	expectedSpecs := setupLabelSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetLabelSpec(context.Background(), fleet.TeamFilter{}, s.Name)
		require.Nil(t, err)

		require.True(t, cmp.Equal(s, spec, cmp.FilterPath(func(p cmp.Path) bool {
			return p.String() == "ID"
		}, cmp.Ignore())))
	}
}

func testLabelsApplySpecsRoundtrip(t *testing.T, ds *Datastore) {
	globalSpecs := setupLabelSpecsTest(t, ds)
	globalOnlyFilter := fleet.TeamFilter{}

	specs, err := ds.GetLabelSpecs(context.Background(), globalOnlyFilter)
	require.Nil(t, err)
	test.ElementsMatchSkipTimestampsID(t, globalSpecs, specs)

	// Should be idempotent
	err = ds.ApplyLabelSpecs(context.Background(), globalSpecs)
	require.Nil(t, err)
	specs, err = ds.GetLabelSpecs(context.Background(), globalOnlyFilter)
	require.Nil(t, err)
	test.ElementsMatchSkipTimestampsID(t, globalSpecs, specs)

	// Test team labels
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1_roundtrip"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2_roundtrip"})
	require.NoError(t, err)

	// Create team label specs; these wouldn't normally coexist in the same call but that gets handled upstream;
	// it doesn't hurt anything to set labels cross-team at the data store level (which assumes auth has already
	// happened).
	teamSpecs := []*fleet.LabelSpec{
		{Name: "team1-label", Query: "SELECT 1", TeamID: &team1.ID},
		{Name: "team2-label", Query: "SELECT 2", TeamID: &team2.ID},
	}
	err = ds.ApplyLabelSpecs(context.Background(), teamSpecs)
	require.NoError(t, err)

	// Global filter should still only return global labels
	specs, err = ds.GetLabelSpecs(context.Background(), globalOnlyFilter)
	require.NoError(t, err)
	test.ElementsMatchSkipTimestampsID(t, globalSpecs, specs)

	// Admin user filter should return all labels (global + team)
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	adminFilter := fleet.TeamFilter{User: user}
	specs, err = ds.GetLabelSpecs(context.Background(), adminFilter)
	require.NoError(t, err)
	require.Len(t, specs, len(globalSpecs)+2)

	// Team1 filter should return only the team1 label
	team1Filter := fleet.TeamFilter{User: user, TeamID: &team1.ID}
	specs, err = ds.GetLabelSpecs(context.Background(), team1Filter)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, "team1-label", specs[0].Name)
	require.Equal(t, team1.ID, *specs[0].TeamID)

	// Team user can only see their team's labels + global labels with no filter applied
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer}}}
	team1UserFilter := fleet.TeamFilter{User: team1User}
	specs, err = ds.GetLabelSpecs(context.Background(), team1UserFilter)
	require.NoError(t, err)
	require.Len(t, specs, len(globalSpecs)+1)
	foundTeam1Label := false
	for _, s := range specs {
		if s.Name == "team1-label" {
			foundTeam1Label = true
			require.Equal(t, team1.ID, *s.TeamID)
		}
		if s.Name == "team2-label" {
			t.Fatal("team2 label should not be in team1 filter results")
		}
	}
	require.True(t, foundTeam1Label, "team1 label should be found")
}

func testLabelsIDsByName(t *testing.T, ds *Datastore) {
	setupLabelSpecsTest(t, ds)

	labels, err := ds.LabelIDsByName(context.Background(), []string{"foo", "bar", "bing"}, fleet.TeamFilter{})
	require.Nil(t, err)
	assert.Equal(t, map[string]uint{"foo": 1, "bar": 2, "bing": 3}, labels)

	// Test team labels
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1_idsbyname"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2_idsbyname"})
	require.NoError(t, err)

	// Create team labels
	team1Label, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:                "team1-idsbyname",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	team2Label, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:                "team2-idsbyname",
		Query:               "SELECT 2",
		TeamID:              &team2.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Global admin can see all labels
	adminUser := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	adminFilter := fleet.TeamFilter{User: adminUser}
	labels, err = ds.LabelIDsByName(context.Background(), []string{"foo", "team1-idsbyname", "team2-idsbyname"}, adminFilter)
	require.NoError(t, err)
	require.Len(t, labels, 3)
	assert.Equal(t, team1Label.ID, labels["team1-idsbyname"])
	assert.Equal(t, team2Label.ID, labels["team2-idsbyname"])

	// Team1 filter should only return global + team1 labels
	team1Filter := fleet.TeamFilter{User: adminUser, TeamID: &team1.ID}
	labels, err = ds.LabelIDsByName(context.Background(), []string{"foo", "team1-idsbyname", "team2-idsbyname"}, team1Filter)
	require.NoError(t, err)
	require.Len(t, labels, 2) // foo and team1-idsbyname
	assert.Equal(t, team1Label.ID, labels["team1-idsbyname"])
	_, hasTeam2 := labels["team2-idsbyname"]
	assert.False(t, hasTeam2, "team2 label should not be visible with team1 filter")

	// Team user can only see their team's labels + global labels
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer}}}
	team1UserFilter := fleet.TeamFilter{User: team1User}
	labels, err = ds.LabelIDsByName(context.Background(), []string{"foo", "team1-idsbyname", "team2-idsbyname"}, team1UserFilter)
	require.NoError(t, err)
	require.Len(t, labels, 2) // foo and team1-idsbyname
	assert.Equal(t, team1Label.ID, labels["team1-idsbyname"])
}

func testLabelsByName(t *testing.T, ds *Datastore) {
	setupLabelSpecsTest(t, ds)

	names := []string{"foo", "bar", "bing"}
	labels, err := ds.LabelsByName(context.Background(), names, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, labels, 3)
	for _, name := range names {
		assert.Contains(t, labels, name)
		assert.Equal(t, name, labels[name].Name)
		switch name {
		case "foo":
			assert.Equal(t, uint(1), labels[name].ID)
			assert.Equal(t, "foo description", labels[name].Description)
		case "bar":
			assert.Equal(t, uint(2), labels[name].ID)
			assert.Empty(t, labels[name].Description)
		case "bing":
			assert.Equal(t, uint(3), labels[name].ID)
			assert.Empty(t, labels[name].Description)
		}
	}

	// Test team labels
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1_byname"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2_byname"})
	require.NoError(t, err)

	// Create team labels
	team1Label, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:                "team1-byname",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	team2Label, err := ds.NewLabel(context.Background(), &fleet.Label{
		Name:                "team2-byname",
		Query:               "SELECT 2",
		TeamID:              &team2.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Global admin can see all labels
	adminUser := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	adminFilter := fleet.TeamFilter{User: adminUser}
	labelNames := []string{"foo", "team1-byname", "team2-byname"}
	labels, err = ds.LabelsByName(context.Background(), labelNames, adminFilter)
	require.NoError(t, err)
	require.Len(t, labels, 3)
	assert.Equal(t, team1Label.ID, labels["team1-byname"].ID)
	assert.Equal(t, team2Label.ID, labels["team2-byname"].ID)

	// Team1 filter should only return global + team1 labels
	team1Filter := fleet.TeamFilter{User: adminUser, TeamID: &team1.ID}
	labels, err = ds.LabelsByName(context.Background(), labelNames, team1Filter)
	require.NoError(t, err)
	require.Len(t, labels, 2) // foo and team1-byname
	assert.Equal(t, team1Label.ID, labels["team1-byname"].ID)
	_, hasTeam2 := labels["team2-byname"]
	assert.False(t, hasTeam2, "team2 label should not be visible with team1 filter")

	// Team user can only see their team's labels + global labels
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer}}}
	team1UserFilter := fleet.TeamFilter{User: team1User}
	labels, err = ds.LabelsByName(context.Background(), labelNames, team1UserFilter)
	require.NoError(t, err)
	require.Len(t, labels, 2) // foo and team1-byname
	assert.Equal(t, team1Label.ID, labels["team1-byname"].ID)
	_, hasTeam2 = labels["team2-byname"]
	assert.False(t, hasTeam2, "team2 label should not be visible to team1 user")
}

func testLabelByName(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Setup: create global labels
	globalLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "global-label",
		Query:               "SELECT 1",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Create teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1_labelbyname"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2_labelbyname"})
	require.NoError(t, err)

	// Create team labels
	team1Label, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "team1-labelbyname",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	_, err = ds.NewLabel(ctx, &fleet.Label{ // should never be retrieved
		Name:                "team2-labelbyname",
		Query:               "SELECT 2",
		TeamID:              &team2.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	adminUser := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer}}}

	// Global admin can get global label
	label, err := ds.LabelByName(ctx, "global-label", fleet.TeamFilter{User: adminUser})
	require.NoError(t, err)
	assert.Equal(t, globalLabel.ID, label.ID)

	// Global admin can get team label
	label, err = ds.LabelByName(ctx, "team1-labelbyname", fleet.TeamFilter{User: adminUser})
	require.NoError(t, err)
	assert.Equal(t, team1Label.ID, label.ID)

	// Team1 user can get global label
	label, err = ds.LabelByName(ctx, "global-label", fleet.TeamFilter{User: team1User})
	require.NoError(t, err)
	assert.Equal(t, globalLabel.ID, label.ID)

	// Team1 user can get team1 label
	label, err = ds.LabelByName(ctx, "team1-labelbyname", fleet.TeamFilter{User: team1User})
	require.NoError(t, err)
	assert.Equal(t, team1Label.ID, label.ID)

	// Team1 user cannot get team2 label
	_, err = ds.LabelByName(ctx, "team2-labelbyname", fleet.TeamFilter{User: team1User})
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "expected not found error for inaccessible team label")

	// Filter to team - team1 filter can see team1 label
	team1Filter := fleet.TeamFilter{User: adminUser, TeamID: &team1.ID}
	label, err = ds.LabelByName(ctx, "team1-labelbyname", team1Filter)
	require.NoError(t, err)
	assert.Equal(t, team1Label.ID, label.ID)

	// Filter to team - team1 filter cannot see team2 label
	_, err = ds.LabelByName(ctx, "team2-labelbyname", team1Filter)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Global-only filter (team_id = 0) can see global label
	globalOnlyFilter := fleet.TeamFilter{User: adminUser, TeamID: ptr.Uint(0)}
	label, err = ds.LabelByName(ctx, "global-label", globalOnlyFilter)
	require.NoError(t, err)
	assert.Equal(t, globalLabel.ID, label.ID)

	// Global-only filter cannot see team label
	_, err = ds.LabelByName(ctx, "team1-labelbyname", globalOnlyFilter)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Non-existent label returns not found
	_, err = ds.LabelByName(ctx, "nonexistent-label", fleet.TeamFilter{User: adminUser})
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testLabelsSave(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	user, err := db.NewUser(context.Background(), &fleet.User{
		Name:       "Adminboi",
		Password:   []byte("p4ssw0rd.123"),
		Email:      "admin@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	label := &fleet.Label{
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes;",
		Platform:    "darwin",
	}
	label, err = db.NewLabel(context.Background(), label)
	require.NoError(t, err)
	require.Nil(t, label.AuthorID)

	label2 := &fleet.Label{
		Name:        "another label",
		Description: "a label",
		Query:       "select 1 from processes;",
		Platform:    "darwin",
		AuthorID:    ptr.Uint(user.ID),
	}
	label2, err = db.NewLabel(context.Background(), label2)
	require.NoError(t, err)
	require.Equal(t, user.ID, *label2.AuthorID)

	// Create an Apple config profile
	profA, err := db.NewMDMAppleConfigProfile(context.Background(), *generateAppleCP("a", "a", 0), nil)
	require.NoError(t, err)
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(),
			"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
			profA.ProfileUUID, label.Name, label.ID)
		return err
	})

	label.Name = "changed name"
	label.Description = "changed description"

	require.NoError(t, db.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	_, _, err = db.SaveLabel(context.Background(), label, filter)
	require.NoError(t, err)
	saved, _, err := db.Label(context.Background(), label.ID, filter)
	require.NoError(t, err)
	assert.Equal(t, label.Name, saved.Name)
	assert.Equal(t, label.Description, saved.Description)
	assert.Equal(t, 1, saved.HostCount)

	var configProfileLabelName string
	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &configProfileLabelName,
			"SELECT label_name FROM mdm_configuration_profile_labels WHERE label_id = ?", label.ID)
	})
	assert.Equal(t, label.Name, configProfileLabelName)
}

func testLabelsQueriesForCentOSHost(t *testing.T, db *Datastore) {
	host, err := db.EnrollOsquery(context.Background(),
		fleet.WithEnrollOsqueryHostID("0"),
		fleet.WithEnrollOsqueryNodeKey("0"),
	)
	require.NoError(t, err, "enrollment should succeed")

	host.Platform = "rhel"
	host.OSVersion = "CentOS 6"
	err = db.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	label, err := db.NewLabel(context.Background(), &fleet.Label{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:                  42,
		Name:                "centos labe",
		Query:               "select 1;",
		Platform:            "centos",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	queries, err := db.LabelQueriesForHost(context.Background(), host)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, "select 1;", queries[fmt.Sprint(label.ID)])
}

func testLabelsRecordNonexistentQueryLabelExecution(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.Nil(t, err)

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	require.NoError(t, db.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{99999: ptr.Bool(true)}, time.Now(), false))
}

func testDeleteLabel(t *testing.T, db *Datastore) {
	ctx := context.Background()
	l, err := db.NewLabel(ctx, &fleet.Label{
		Name:  t.Name(),
		Query: "query1",
	})
	require.NoError(t, err)

	p, err := db.NewPack(ctx, &fleet.Pack{
		Name:     t.Name(),
		LabelIDs: []uint{l.ID},
	})
	require.NoError(t, err)

	require.NoError(t, db.DeleteLabel(ctx, l.Name, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}))

	newP, err := db.Pack(ctx, p.ID)
	require.NoError(t, err)
	require.Empty(t, newP.Labels)

	require.NoError(t, db.DeletePack(ctx, newP.Name))

	// delete a non-existing label
	err = db.DeleteLabel(ctx, "no-such-label", fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// create a software installer and scope it via a label
	u := test.NewUser(t, db, "user1", "user1@example.com", false)
	installer, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	installerID, _, err := db.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install foo",
		InstallerFile:   installer,
		StorageID:       uuid.NewString(),
		Filename:        "foo.pkg",
		Title:           "foo",
		Source:          "apps",
		Version:         "0.0.1",
		UserID:          u.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	l2, err := db.NewLabel(ctx, &fleet.Label{
		Name:  t.Name() + "2",
		Query: "query2",
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, db, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO software_installer_labels (software_installer_id, label_id) VALUES (?, ?)`, installerID, l2.ID)
		return err
	})

	// try to delete that label referenced by software installer
	err = db.DeleteLabel(ctx, l2.Name, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	require.Error(t, err)
	require.True(t, fleet.IsForeignKey(err))

	// Test team label filtering
	team1, err := db.NewTeam(ctx, &fleet.Team{Name: "team1_delete"})
	require.NoError(t, err)
	team2, err := db.NewTeam(ctx, &fleet.Team{Name: "team2_delete"})
	require.NoError(t, err)

	// Create team labels
	team1Label, err := db.NewLabel(ctx, &fleet.Label{
		Name:                "team1-delete-label",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	team2Label, err := db.NewLabel(ctx, &fleet.Label{
		Name:                "team2-delete-label",
		Query:               "SELECT 2",
		TeamID:              &team2.ID,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	adminUser := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	team1User := &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer}}}

	// Global admin can delete team labels
	err = db.DeleteLabel(ctx, team1Label.Name, fleet.TeamFilter{User: adminUser})
	require.NoError(t, err)

	// Verify it's deleted
	_, err = db.LabelByName(ctx, team1Label.Name, fleet.TeamFilter{User: adminUser})
	require.True(t, fleet.IsNotFound(err))

	// Team user cannot delete label from another team (not found because not visible)
	err = db.DeleteLabel(ctx, team2Label.Name, fleet.TeamFilter{User: team1User})
	require.True(t, fleet.IsNotFound(err))

	// Verify team2 label still exists
	label, err := db.LabelByName(ctx, team2Label.Name, fleet.TeamFilter{User: adminUser})
	require.NoError(t, err)
	require.Equal(t, team2Label.ID, label.ID)

	// Admin with team filter can delete
	err = db.DeleteLabel(ctx, team2Label.Name, fleet.TeamFilter{User: adminUser, TeamID: &team2.ID})
	require.NoError(t, err)
}

func testLabelsSummaryAndListTeamFiltering(t *testing.T, db *Datastore) {
	test.AddAllHostsLabel(t, db)

	// Only 'All Hosts' label should be returned
	labels, err := db.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{}, false)
	require.NoError(t, err)
	require.Len(t, labels, 1)

	newLabels := []*fleet.LabelSpec{
		{
			Name:     "foo",
			Query:    "query foo",
			Platform: "platform",
		},
		{
			Name:     "bar",
			Query:    "query bar",
			Platform: "platform",
		},
		{
			Name:        "baz",
			Query:       "query baz",
			Description: "description baz",
			Platform:    "darwin",
		},
	}
	err = db.ApplyLabelSpecs(context.Background(), newLabels)
	require.Nil(t, err)

	team1, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	team1Label, err := db.NewLabel(context.Background(), &fleet.Label{
		Name:                "t1 label",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		TeamID:              &team1.ID,
	})
	require.NoError(t, err)
	team2Label, err := db.NewLabel(context.Background(), &fleet.Label{
		Name:                "t2 label",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		TeamID:              &team2.ID,
	})
	require.NoError(t, err)

	// should only show global labels
	labels, err = db.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{}, false)
	require.NoError(t, err)
	require.Len(t, labels, 4)
	labelsByID := make(map[uint]*fleet.Label)
	for _, l := range labels {
		labelsByID[l.ID] = l
	}

	// should show only global labels
	ls, err := db.LabelsSummary(context.Background(), fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, ls, 4)
	for _, l := range ls {
		assert.NotNil(t, labelsByID[l.ID])
		assert.Equal(t, labelsByID[l.ID].Name, l.Name)
		assert.Equal(t, labelsByID[l.ID].Description, l.Description)
		assert.Equal(t, labelsByID[l.ID].LabelType, l.LabelType)
	}

	_, err = db.NewLabel(context.Background(), &fleet.Label{
		Name:  "bing",
		Query: "query bing",
	})
	require.NoError(t, err)

	ls, err = db.LabelsSummary(context.Background(), fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, ls, 5)

	for _, tc := range []struct {
		name               string
		filter             fleet.TeamFilter
		expectedErr        error
		expectedTeamLabels map[*fleet.Team]*fleet.Label
	}{
		{
			name: "explicit global filter",
			filter: fleet.TeamFilter{
				User:   &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserver}}},
				TeamID: ptr.Uint(0),
			},
		},
		{
			name: "global role filtered to team",
			filter: fleet.TeamFilter{
				User:   &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				TeamID: &team1.ID,
			},
			expectedTeamLabels: map[*fleet.Team]*fleet.Label{team1: team1Label},
		},
		{
			name: "team role filtered to user-accessible team",
			filter: fleet.TeamFilter{
				User:   &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserverPlus}}},
				TeamID: &team1.ID,
			},
			expectedTeamLabels: map[*fleet.Team]*fleet.Label{team1: team1Label},
		},
		{
			name: "team role filtered to inaccessible team",
			filter: fleet.TeamFilter{
				User:   &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserverPlus}}},
				TeamID: &team2.ID,
			},
			expectedErr: errInaccessibleTeam,
		},
		{
			name: "global role with no team filter",
			filter: fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			},
			expectedTeamLabels: map[*fleet.Team]*fleet.Label{team1: team1Label, team2: team2Label},
		},
		{
			name: "single-team user with no team filter",
			filter: fleet.TeamFilter{
				User: &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserverPlus}}},
			},
			expectedTeamLabels: map[*fleet.Team]*fleet.Label{team1: team1Label},
		},
		{
			name: "multi-team user with no team filter, partial overlap with labels",
			filter: fleet.TeamFilter{
				User: &fleet.User{Teams: []fleet.UserTeam{
					{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserverPlus},
					{Team: fleet.Team{ID: team3.ID}, Role: fleet.RoleMaintainer},
				}},
			},
			expectedTeamLabels: map[*fleet.Team]*fleet.Label{team1: team1Label},
		},
		{
			name: "multi-team user with no team filter, full overlap with labels",
			filter: fleet.TeamFilter{
				User: &fleet.User{Teams: []fleet.UserTeam{
					{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserverPlus},
					{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleMaintainer},
				}},
			},
			expectedTeamLabels: map[*fleet.Team]*fleet.Label{team1: team1Label, team2: team2Label},
		},
	} {
		t.Run(tc.name+" summary", func(t *testing.T) {
			ls, err := db.LabelsSummary(context.Background(), tc.filter)
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				return
			}
			require.NoError(t, err)
			require.Len(t, ls, 5+len(tc.expectedTeamLabels))

			foundTeamLabels := make(map[uint]fleet.LabelSummary)
			for _, l := range ls {
				if l.TeamID != nil {
					foundTeamLabels[*l.TeamID] = *l
				}
			}
			for team, label := range tc.expectedTeamLabels {
				foundLabel, labelInMap := foundTeamLabels[team.ID]
				require.Truef(t, labelInMap, "%s label should have been found", team.Name)
				require.Equalf(t, label.ID, foundLabel.ID, "Found team label %s label did not match expected (%s)", foundLabel.Name, label.Name)
			}
		})
		t.Run(tc.name+" list", func(t *testing.T) {
			ls, err := db.ListLabels(context.Background(), tc.filter, fleet.ListOptions{}, false)
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				return
			}
			require.NoError(t, err)
			require.Len(t, ls, 5+len(tc.expectedTeamLabels))

			foundTeamLabels := make(map[uint]fleet.Label)
			for _, l := range ls {
				if l.TeamID != nil {
					foundTeamLabels[*l.TeamID] = *l
				}
			}
			for team, label := range tc.expectedTeamLabels {
				foundLabel, labelInMap := foundTeamLabels[team.ID]
				require.Truef(t, labelInMap, "%s label should have been found", team.Name)
				require.Equalf(t, label.ID, foundLabel.ID, "Found team label %s label did not match expected (%s)", foundLabel.Name, label.Name)
			}
		})
	}
}

func testListHostsInLabelIssues(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	q := test.NewQuery(t, ds, nil, "query1", "select 1", 0, true)
	q2 := test.NewQuery(t, ds, nil, "query2", "select 1", 0, true)
	p, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)
	p2, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q2.ID,
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, filter, fleet.HostListOptions{}, 10)
	require.Len(t, hosts, 10)

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	for _, h := range hosts {
		err = ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	hosts = listHostsInLabelCheckCount(t, ds, filter, l1.ID, fleet.HostListOptions{}, 10)

	h1 := hosts[0]
	h2 := hosts[1]

	assert.Zero(t, h1.HostIssues.FailingPoliciesCount)
	assert.Zero(t, *h1.HostIssues.CriticalVulnerabilitiesCount)
	assert.Zero(t, h1.HostIssues.TotalIssuesCount)
	assert.Zero(t, h2.HostIssues.FailingPoliciesCount)
	assert.Zero(t, *h2.HostIssues.CriticalVulnerabilitiesCount)
	assert.Zero(t, h2.HostIssues.TotalIssuesCount)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), false))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(false), p2.ID: ptr.Bool(false)}, time.Now(), false))
	checkLabelHostIssues(t, ds, l1.ID, filter, h2.ID, fleet.HostListOptions{}, 2, 0)

	// Add a critical vulnerability
	// seed software
	software := []fleet.Software{
		{Name: "foo0", Version: "0", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo1", Version: "1", Source: "chrome_extensions"},
		{Name: "foo2", Version: "2", Source: "chrome_extensions"},
		{Name: "foo3", Version: "3", Source: "chrome_extensions"},
		{Name: "foo4", Version: "4", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo5", Version: "5", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo6", Version: "6", Source: "chrome_extensions"}, // vulnerable
		{Name: "foo7", Version: "7", Source: "chrome_extensions"}, // vulnerable
	}

	for i := 0; i < len(software); i++ {
		_, err := ds.UpdateHostSoftware(context.Background(), hosts[i].ID, software[:i+1])
		require.NoError(t, err)
	}

	softwareItems := make([]fleet.Software, 0, len(software))
	ctx := context.Background()
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &softwareItems, "SELECT id, version FROM software"))
	require.Len(t, softwareItems, len(software))

	for _, sw := range softwareItems {
		_, err := ds.InsertSoftwareVulnerability(
			context.Background(), fleet.SoftwareVulnerability{
				CVE:        fmt.Sprintf("CVE-%s", sw.Version),
				SoftwareID: sw.ID,
			}, fleet.NVDSource,
		)
		require.NoError(t, err)
	}
	require.NoError(
		t, ds.InsertCVEMeta(
			ctx, []fleet.CVEMeta{
				{
					CVE:       "CVE-0",
					CVSSScore: ptr.Float64(2 * criticalCVSSScoreCutoff),
				},
				{
					CVE:       "CVE-3",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff), // not critical
				},
				{
					CVE:       "CVE-4",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 0.001),
				},
				{
					CVE:       "CVE-5",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 0.01),
				},
				{
					CVE:       "CVE-6",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 0.1),
				},
				{
					CVE:       "CVE-7",
					CVSSScore: ptr.Float64(criticalCVSSScoreCutoff + 1),
				},
			},
		),
	)
	// Populate critical vulnerabilities, which can be done with premium license.
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	assert.NoError(t, ds.UpdateHostIssuesVulnerabilities(ctx))
	checkLabelHostIssues(t, ds, l1.ID, filter, hosts[6].ID, fleet.HostListOptions{}, 0, 4)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(true), p2.ID: ptr.Bool(false)}, time.Now(), false))
	checkLabelHostIssues(t, ds, l1.ID, filter, h2.ID, fleet.HostListOptions{}, 1, 1)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h2, map[uint]*bool{p.ID: ptr.Bool(true), p2.ID: ptr.Bool(true)}, time.Now(), false))
	checkLabelHostIssues(t, ds, l1.ID, filter, h2.ID, fleet.HostListOptions{}, 0, 1)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), h1, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))
	checkLabelHostIssues(t, ds, l1.ID, filter, h1.ID, fleet.HostListOptions{}, 1, 1)

	checkLabelHostIssues(t, ds, l1.ID, filter, h1.ID, fleet.HostListOptions{DisableIssues: true}, 0, 0)
	checkLabelHostIssues(t, ds, l1.ID, filter, hosts[6].ID, fleet.HostListOptions{DisableIssues: true}, 0, 0)
}

func checkLabelHostIssues(
	t *testing.T, ds *Datastore, lid uint, filter fleet.TeamFilter, hid uint, opts fleet.HostListOptions,
	failingPoliciesExpected uint64, criticalVulnerabilitiesExpected uint64,
) {
	hosts := listHostsInLabelCheckCount(t, ds, filter, lid, opts, 10)
	foundH2 := false
	var foundHost *fleet.Host
	for _, host := range hosts {
		if host.ID == hid {
			foundH2 = true
			foundHost = host
			break
		}
	}
	require.True(t, foundH2)
	assert.Equal(t, failingPoliciesExpected, foundHost.HostIssues.FailingPoliciesCount)

	if opts.DisableIssues {
		assert.Nil(t, foundHost.HostIssues.CriticalVulnerabilitiesCount)
		assert.Zero(t, foundHost.HostIssues.TotalIssuesCount)
		return
	}
	assert.Equal(t, criticalVulnerabilitiesExpected, *foundHost.HostIssues.CriticalVulnerabilitiesCount)
	assert.Equal(t, failingPoliciesExpected+criticalVulnerabilitiesExpected, foundHost.HostIssues.TotalIssuesCount)

	hostById, err := ds.Host(context.Background(), hid)
	require.NoError(t, err)
	assert.Equal(t, failingPoliciesExpected, hostById.HostIssues.FailingPoliciesCount)
	assert.Equal(t, failingPoliciesExpected+criticalVulnerabilitiesExpected, hostById.HostIssues.TotalIssuesCount)
	assert.Equal(t, foundHost.HostIssues.CriticalVulnerabilitiesCount, hostById.HostIssues.CriticalVulnerabilitiesCount)
}

func testListHostsInLabelDiskEncryptionStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// seed hosts
	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(fmt.Sprintf("%d", i)),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
		nanoEnrollAndSetHostMDMData(t, ds, h, false)
	}

	// set up data
	noTeamFVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("filevault-1", "com.fleetdm.fleet.mdm.filevault", 0), nil)
	require.NoError(t, err)

	// verifying status
	upsertHostCPs([]*fleet.Host{hosts[0], hosts[1]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
	createDiskEncryptionRecord(ctx, ds, t, hosts[0], "key-1", true, oneMinuteAfterThreshold)
	createDiskEncryptionRecord(ctx, ds, t, hosts[1], "key-1", true, oneMinuteAfterThreshold)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// action required status
	upsertHostCPs(
		[]*fleet.Host{hosts[2], hosts[3]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryVerifying, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[2].ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[3].ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// enforcing status

	// host profile status is `pending`
	upsertHostCPs(
		[]*fleet.Host{hosts[4]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryPending, ctx, ds, t,
	)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// host profile status does not exist
	upsertHostCPs(
		[]*fleet.Host{hosts[5]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		nil, ctx, ds, t,
	)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// host profile status is verifying but decryptable key field does not exist
	upsertHostCPs(
		[]*fleet.Host{hosts[6]},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryPending, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[6].ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// failed status
	upsertHostCPs([]*fleet.Host{hosts[7], hosts[8]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, ctx, ds, t)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 0)

	// removing enforcement status
	upsertHostCPs([]*fleet.Host{hosts[9]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryPending, ctx, ds, t)

	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 0)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 1)

	// verified status
	upsertHostCPs([]*fleet.Host{hosts[0]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerifying}, 1)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 1)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 3)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionFailed}, 2)
	listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{MacOSSettingsDiskEncryptionFilter: fleet.DiskEncryptionRemovingEnforcement}, 1)
}

func testHostMemberOfAllLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	//
	// Setup test
	// - h1 member of 'All hosts', 'Foobar' and 'Zoobar'
	// - h2 member of 'All hosts' and 'Foobar'
	// - h3 member of 'All hosts' and 'Zoobar'
	// - h4 member of 'All hosts'
	// - h5 member of no labels
	//

	allHostsLabel, err := ds.NewLabel(ctx,
		&fleet.Label{
			Name:                "All hosts",
			Query:               "SELECT 1",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
	)
	require.NoError(t, err)
	foobarLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "Foobar",
		Query:               "SELECT 1;",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	zoobarLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "Zoobar",
		Query:               "SELECT 2;",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	newHostFunc := func(name string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(name),
			NodeKey:         ptr.String(name),
			UUID:            name,
			Hostname:        "foo.local" + name,
		})
		require.NoError(t, err)
		return h
	}

	h1 := newHostFunc("h1")
	h2 := newHostFunc("h2")
	h3 := newHostFunc("h3")
	h4 := newHostFunc("h4")
	h5 := newHostFunc("h5")
	_ = h5

	err = ds.RecordLabelQueryExecutions(ctx, h1, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		foobarLabel.ID:   ptr.Bool(true),
		zoobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(ctx, h2, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		foobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(ctx, h3, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		zoobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(ctx, h4, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)

	//
	// Run tests for HostMemberOfAllLabels
	//

	for _, tc := range []struct {
		name           string
		hostID         uint
		labelNames     []string
		expectedResult bool
	}{
		{
			name:           "nonexistent host",
			hostID:         999,
			labelNames:     []string{allHostsLabel.Name},
			expectedResult: false,
		},
		{
			name:           "h1 does not belong to nonexistent label",
			hostID:         h1.ID,
			labelNames:     []string{"Non existent label"},
			expectedResult: false,
		},
		{
			name:           "h1 does not belong to All hosts + nonexistent label",
			hostID:         h1.ID,
			labelNames:     []string{allHostsLabel.Name, "Non existent label"},
			expectedResult: false,
		},
		{
			name:           "h1 belongs to the given subset of labels",
			hostID:         h1.ID,
			labelNames:     []string{allHostsLabel.Name, foobarLabel.Name},
			expectedResult: true,
		},
		{
			name:           "h1 belongs to all the given labels",
			hostID:         h1.ID,
			labelNames:     []string{allHostsLabel.Name, foobarLabel.Name, zoobarLabel.Name},
			expectedResult: true,
		},
		{
			name:           "h1 member of empty label set",
			hostID:         h1.ID,
			labelNames:     []string{},
			expectedResult: true,
		},
		{
			name:           "h2 belongs to all the given labels",
			hostID:         h2.ID,
			labelNames:     []string{allHostsLabel.Name, foobarLabel.Name},
			expectedResult: true,
		},
		{
			name:           "h2 does not belongs to all the given labels",
			hostID:         h2.ID,
			labelNames:     []string{allHostsLabel.Name, foobarLabel.Name, zoobarLabel.Name},
			expectedResult: false,
		},
		{
			name:           "h2 belongs to the given label",
			hostID:         h2.ID,
			labelNames:     []string{foobarLabel.Name},
			expectedResult: true,
		},
		{
			name:           "h2 does not belong to the given label",
			hostID:         h2.ID,
			labelNames:     []string{zoobarLabel.Name},
			expectedResult: false,
		},
		{
			name:           "h3 belongs to all the given labels",
			hostID:         h3.ID,
			labelNames:     []string{allHostsLabel.Name, zoobarLabel.Name},
			expectedResult: true,
		},
		{
			name:           "h4 belongs to all the given labels",
			hostID:         h4.ID,
			labelNames:     []string{allHostsLabel.Name},
			expectedResult: true,
		},
		{
			name:           "h4 does not belong to the given labels",
			hostID:         h4.ID,
			labelNames:     []string{foobarLabel.Name},
			expectedResult: false,
		},
		{
			name:           "h5 does not belong to the given labels",
			hostID:         h5.ID,
			labelNames:     []string{allHostsLabel.Name},
			expectedResult: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := ds.HostMemberOfAllLabels(ctx, tc.hostID, tc.labelNames)
			require.NoError(t, err)
			require.Equal(t, tc.expectedResult, v)
		})
	}
}

func testLabelsListHostsInLabelOSSettings(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1"),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		Platform:        "windows",
	})
	require.NoError(t, err)

	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("2"),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
		Platform:        "windows",
	})
	require.NoError(t, err)
	h3, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("3"),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		Hostname:        "baz.local",
		Platform:        "centos",
	})
	require.NoError(t, err)

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	// add all hosts to label
	for _, h := range []*fleet.Host{h1, h2, h3} {
		require.NoError(t, db.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false))
	}

	// turn on disk encryption
	ac, err := db.AppConfig(context.Background())
	require.NoError(t, err)
	ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
	require.NoError(t, db.SaveAppConfig(context.Background(), ac))

	// add two hosts to MDM to enforce disk encryption, fleet doesn't enforce settings on centos so h3 is not included
	for _, h := range []*fleet.Host{h1, h2} {
		windowsEnroll(t, db, h)
		require.NoError(t, db.SetOrUpdateMDMData(context.Background(), h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))
	}
	// add disk encryption key for h1
	_, err = db.SetOrUpdateHostDiskEncryptionKey(context.Background(), h1, "test-key", "", ptr.Bool(true))
	require.NoError(t, err)
	// add disk encryption for h1
	require.NoError(t, db.SetOrUpdateHostDisksEncryption(context.Background(), h1.ID, true))

	checkHosts := func(t *testing.T, gotHosts []*fleet.Host, expectedIDs []uint) {
		require.Len(t, gotHosts, len(expectedIDs))
		for _, h := range gotHosts {
			require.Contains(t, expectedIDs, h.ID)
		}
	}

	// baseline no filter
	hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{}, 3)
	checkHosts(t, hosts, []uint{h1.ID, h2.ID, h3.ID})

	t.Run("os_settings_disk_encryption", func(t *testing.T) {
		hosts = listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 1)
		checkHosts(t, hosts, []uint{h1.ID})
		hosts = listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1)
		checkHosts(t, hosts, []uint{h2.ID})
	})

	t.Run("os_settings", func(t *testing.T) {
		hosts = listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{OSSettingsFilter: fleet.OSSettingsVerified}, 1)
		checkHosts(t, hosts, []uint{h1.ID})
		hosts = listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{OSSettingsFilter: fleet.OSSettingsPending}, 1)
		checkHosts(t, hosts, []uint{h2.ID})
	})
}

func testAddDeleteLabelsToFromHost(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("1"),
		NodeKey:       ptr.String("1"),
		UUID:          "1",
		Hostname:      "foo.local",
		Platform:      "darwin",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("2"),
		NodeKey:       ptr.String("2"),
		UUID:          "2",
		Hostname:      "bar.local",
		Platform:      "windows",
	})
	require.NoError(t, err)

	err = ds.AddLabelsToHost(ctx, host1.ID, nil)
	require.NoError(t, err)
	err = ds.RemoveLabelsFromHost(ctx, host1.ID, nil)
	require.NoError(t, err)

	label1, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "label1",
		Query:               "SELECT 1;",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	label2, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "label2",
		Query:               "SELECT 2;",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	// Removing a label and multiple labels that the host is not a member of.
	err = ds.RemoveLabelsFromHost(ctx, host1.ID, []uint{label1.ID})
	require.NoError(t, err)
	err = ds.RemoveLabelsFromHost(ctx, host1.ID, []uint{label1.ID, label2.ID})
	require.NoError(t, err)

	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}

	// Adding and removing labels.
	err = ds.AddLabelsToHost(ctx, host1.ID, []uint{label1.ID})
	require.NoError(t, err)
	lbl, hids, err := ds.Label(ctx, label1.ID, filter)
	require.NoError(t, err)
	require.Equal(t, label1.ID, lbl.ID)
	require.ElementsMatch(t, []uint{host1.ID}, hids)
	getLabelUpdatedAt := func(updatedAt *time.Time) func(q sqlx.ExtContext) error {
		return func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, updatedAt, `SELECT updated_at FROM label_membership WHERE host_id = ? AND label_id = ?`, host1.ID, label1.ID)
		}
	}
	var labelUpdatedAt1 time.Time
	ExecAdhocSQL(t, ds, getLabelUpdatedAt(&labelUpdatedAt1))
	time.Sleep(1 * time.Second)
	// Add a label that the host is already member of.
	err = ds.AddLabelsToHost(ctx, host1.ID, []uint{label1.ID})
	require.NoError(t, err)
	var labelUpdatedAt2 time.Time
	ExecAdhocSQL(t, ds, getLabelUpdatedAt(&labelUpdatedAt2))
	require.True(t, labelUpdatedAt2.After(labelUpdatedAt1))
	labels, err := ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)
	labels2, err := ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Empty(t, labels2)

	// Removing a label that the host is a member of
	// and one that the host is not a member of.
	err = ds.RemoveLabelsFromHost(ctx, host1.ID, []uint{label1.ID, label2.ID})
	require.NoError(t, err)
	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Empty(t, labels)

	// Add and remove multiple labels.
	err = ds.AddLabelsToHost(ctx, host1.ID, []uint{label1.ID, label2.ID})
	require.NoError(t, err)
	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 2)

	err = ds.AddLabelsToHost(ctx, host2.ID, []uint{label1.ID})
	require.NoError(t, err)
	labels, err = ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)

	lbl, hids, err = ds.Label(ctx, label1.ID, filter)
	require.NoError(t, err)
	require.Equal(t, label1.ID, lbl.ID)
	require.ElementsMatch(t, []uint{host1.ID, host2.ID}, hids)

	err = ds.RemoveLabelsFromHost(ctx, host1.ID, []uint{label1.ID, label2.ID})
	require.NoError(t, err)
	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Empty(t, labels)
}

func labelIDFromName(t *testing.T, ds fleet.Datastore, name string) uint {
	allLbls, err := ds.ListLabels(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{}, false)
	require.Nil(t, err)
	for _, lbl := range allLbls {
		if lbl.Name == name {
			return lbl.ID
		}
	}
	return 0
}

func testUpdateLabelMembershipByHostIDs(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	filter := fleet.TeamFilter{User: test.UserAdmin}

	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("1"),
		NodeKey:       ptr.String("1"),
		UUID:          "1",
		Hostname:      "foo.local",
		Platform:      "darwin",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("2"),
		NodeKey:       ptr.String("2"),
		UUID:          "2",
		Hostname:      "bar.local",
		Platform:      "windows",
	})
	require.NoError(t, err)
	// hosts 2 and 3 have the same hostname
	host3, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("3"),
		NodeKey:       ptr.String("3"),
		UUID:          "3",
		Hostname:      "bar.local",
		Platform:      "windows",
	})
	require.NoError(t, err)

	label1, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "label1",
		Query:               "",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	// add hosts 1 and 2 to the label
	label, hostIDs, err := ds.UpdateLabelMembershipByHostIDs(ctx, *label1, []uint{host1.ID, host2.ID}, filter)
	require.NoError(t, err)

	require.Equal(t, label.HostCount, 2)

	// expect hosts 1 and 2 to be in the label, but not 3
	require.NoError(t, err)
	// correct hosts were added to label
	require.Len(t, hostIDs, 2)
	require.Equal(t, host1.ID, hostIDs[0])
	require.Equal(t, host2.ID, hostIDs[1])

	labelSpec, err := ds.GetLabelSpec(ctx, fleet.TeamFilter{}, label1.Name) // only need global labels, so this works
	require.NoError(t, err)
	// label.Hosts contains hostnames
	require.Len(t, labelSpec.Hosts, 2)
	require.Equal(t, strconv.Itoa(int(host1.ID)), labelSpec.Hosts[0]) //nolint:gosec // dismiss G115
	require.Equal(t, strconv.Itoa(int(host2.ID)), labelSpec.Hosts[1]) //nolint:gosec // dismiss G115

	labels, err := ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	labels, err = ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	labels, err = ds.ListLabelsForHost(ctx, host3.ID)
	require.NoError(t, err)
	require.Len(t, labels, 0)

	// modify the label to contain hosts 1 and 3, confirm
	label, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, *label1, []uint{host1.ID, host3.ID}, filter)
	require.NoError(t, err)

	require.Equal(t, label.HostCount, 2)

	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	labels, err = ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 0)

	labels, err = ds.ListLabelsForHost(ctx, host3.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	// modify the label to contain hosts 2 and 3, confirm
	label, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, *label1, []uint{host2.ID, host3.ID}, filter)
	require.NoError(t, err)

	require.Equal(t, label.HostCount, 2)

	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 0)

	labels, err = ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	labels, err = ds.ListLabelsForHost(ctx, host3.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	// modify the label to contain no hosts, confirm
	label, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, *label1, []uint{}, filter)
	require.NoError(t, err)
	require.Equal(t, label.HostCount, 0)

	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 0)

	labels, err = ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 0)

	labels, err = ds.ListLabelsForHost(ctx, host3.ID)
	require.NoError(t, err)
	require.Len(t, labels, 0)

	// modify the label to contain all 3 hosts, confirm
	label, hostIDs, err = ds.UpdateLabelMembershipByHostIDs(ctx, *label1, []uint{host1.ID, host2.ID, host3.ID}, filter)
	require.NoError(t, err)

	require.Equal(t, label.HostCount, 3)

	labels, err = ds.ListLabelsForHost(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	labels, err = ds.ListLabelsForHost(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	labels, err = ds.ListLabelsForHost(ctx, host3.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "label1", labels[0].Name)

	require.NoError(t, err)
	require.Len(t, hostIDs, 3)
	require.Equal(t, host1.ID, hostIDs[0])
	// 2 and 3 have same name
	require.Equal(t, host2.ID, hostIDs[1])
	require.Equal(t, host3.ID, hostIDs[2])

	labelSpec, err = ds.GetLabelSpec(ctx, fleet.TeamFilter{}, label1.Name) // only need global labels, so this works
	require.NoError(t, err)

	// label.Hosts contains hostnames
	require.Len(t, labelSpec.Hosts, 3)
	require.Equal(t, strconv.Itoa(int(host1.ID)), labelSpec.Hosts[0]) //nolint:gosec // dismiss G115
	require.Equal(t, strconv.Itoa(int(host2.ID)), labelSpec.Hosts[1]) //nolint:gosec // dismiss G115
	require.Equal(t, strconv.Itoa(int(host3.ID)), labelSpec.Hosts[2]) //nolint:gosec // dismiss G115

	// Test team label host validation behavior
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1_membership"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2_membership"})
	require.NoError(t, err)

	// Create hosts on different teams
	hostTeam1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("team1host"),
		NodeKey:       ptr.String("team1host"),
		UUID:          "team1host",
		Hostname:      "team1host.local",
		Platform:      "darwin",
		TeamID:        &team1.ID,
	})
	require.NoError(t, err)
	hostTeam2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("team2host"),
		NodeKey:       ptr.String("team2host"),
		UUID:          "team2host",
		Hostname:      "team2host.local",
		Platform:      "darwin",
		TeamID:        &team2.ID,
	})
	require.NoError(t, err)
	hostNoTeam, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("noteamhost"),
		NodeKey:       ptr.String("noteamhost"),
		UUID:          "noteamhost",
		Hostname:      "noteamhost.local",
		Platform:      "darwin",
		TeamID:        nil,
	})
	require.NoError(t, err)

	// Create a team label
	teamLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "team1-membership-label",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		TeamID:              &team1.ID,
	})
	require.NoError(t, err)

	// Adding a team1 host to a team1 label should succeed
	teamLabelResult, teamHostIDs, err := ds.UpdateLabelMembershipByHostIDs(ctx, *teamLabel, []uint{hostTeam1.ID}, filter)
	require.NoError(t, err)
	require.Equal(t, 1, teamLabelResult.HostCount)
	require.Len(t, teamHostIDs, 1)
	require.Equal(t, hostTeam1.ID, teamHostIDs[0])

	// Adding a team2 host to a team1 label should fail
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, *teamLabel, []uint{hostTeam2.ID}, filter)
	require.Error(t, err)
	require.Contains(t, err.Error(), "supplied hosts are on a different team than the label")

	// Adding a no-team host to a team1 label should fail
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, *teamLabel, []uint{hostNoTeam.ID}, filter)
	require.Error(t, err)
	require.Contains(t, err.Error(), "supplied hosts are on a different team than the label")

	// Adding mixed hosts (team1 + team2) to a team1 label should fail
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, *teamLabel, []uint{hostTeam1.ID, hostTeam2.ID}, filter)
	require.Error(t, err)
	require.Contains(t, err.Error(), "supplied hosts are on a different team than the label")

	// Global label can have hosts from any team
	globalMembershipLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "global-membership-label",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		TeamID:              nil,
	})
	require.NoError(t, err)

	globalLabelResult, globalHostIDs, err := ds.UpdateLabelMembershipByHostIDs(ctx, *globalMembershipLabel, []uint{hostTeam1.ID, hostTeam2.ID, hostNoTeam.ID}, filter)
	require.NoError(t, err)
	require.Equal(t, 3, globalLabelResult.HostCount)
	require.Len(t, globalHostIDs, 3)
}

func testApplyLabelSpecsForSerialUUID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("1"),
		NodeKey:        ptr.String("1"),
		UUID:           "1",
		Hostname:       "foo.local",
		HardwareSerial: "hwd1",
		Platform:       "darwin",
	})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("2"),
		NodeKey:        ptr.String("2"),
		UUID:           "2",
		Hostname:       "bar.local",
		HardwareSerial: "hwd2",
		Platform:       "windows",
	})
	require.NoError(t, err)
	host3, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("3"),
		NodeKey:        ptr.String("3"),
		UUID:           "uuid3",
		Hostname:       "baz.local",
		HardwareSerial: "hwd3",
		Platform:       "windows",
	})
	require.NoError(t, err)
	host4, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("4"),
		NodeKey:        ptr.String("4"),
		UUID:           "uuid4",
		Hostname:       "boop.local",
		HardwareSerial: "hwd4",
		Platform:       "linux",
	})
	require.NoError(t, err)

	err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
		{
			Name:                "label1",
			LabelMembershipType: fleet.LabelMembershipTypeManual,
			Hosts: []string{
				"foo.local",
				"hwd2",
				"uuid3",
				strconv.Itoa(int(host4.ID)), //nolint:gosec // dismiss G115
			},
		},
	})
	require.NoError(t, err)

	hosts, err := ds.ListHostsInLabel(ctx, fleet.TeamFilter{User: test.UserAdmin}, 1, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 4)
	require.Equal(t, host1.ID, hosts[0].ID)
	require.Equal(t, host2.ID, hosts[1].ID)
	require.Equal(t, host3.ID, hosts[2].ID)
	require.Equal(t, host4.ID, hosts[3].ID)
}

func testApplyLabelSpecsWithPlatformChange(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create hosts with different platforms
	hostDarwin1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("darwin1"),
		NodeKey:        ptr.String("darwin1"),
		UUID:           "darwin-uuid-1",
		Hostname:       "darwin1.local",
		HardwareSerial: "darwin-serial-1",
		Platform:       "darwin",
	})
	require.NoError(t, err)

	hostDarwin2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("darwin2"),
		NodeKey:        ptr.String("darwin2"),
		UUID:           "darwin-uuid-2",
		Hostname:       "darwin2.local",
		HardwareSerial: "darwin-serial-2",
		Platform:       "darwin",
	})
	require.NoError(t, err)

	hostWindows1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("windows1"),
		NodeKey:        ptr.String("windows1"),
		UUID:           "windows-uuid-1",
		Hostname:       "windows1.local",
		HardwareSerial: "windows-serial-1",
		Platform:       "windows",
	})
	require.NoError(t, err)

	hostLinux1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("linux1"),
		NodeKey:        ptr.String("linux1"),
		UUID:           "linux-uuid-1",
		Hostname:       "linux1.local",
		HardwareSerial: "linux-serial-1",
		Platform:       "linux",
	})
	require.NoError(t, err)

	// Test 1: Create a dynamic label for darwin platform
	err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
		{
			Name:                "platform_test_label",
			Description:         "Test label for platform changes",
			Query:               "select 1",
			Platform:            "darwin",
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
	})
	require.NoError(t, err)

	// Get the label ID
	labels, err := ds.LabelsByName(ctx, []string{"platform_test_label"}, fleet.TeamFilter{})
	require.NoError(t, err)
	label := labels["platform_test_label"]
	require.NotNil(t, label)
	require.Equal(t, "darwin", label.Platform)

	// Add hosts to the label to simulate existing memberships
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostDarwin1, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostDarwin2, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostWindows1, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostLinux1, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	// Verify all hosts are in the label
	hosts, err := ds.ListHostsInLabel(ctx, fleet.TeamFilter{User: test.UserAdmin}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 4)

	// Test 2: Change platform to windows - all memberships are cleared
	err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
		{
			Name:                "platform_test_label",
			Description:         "Test label for platform changes",
			Query:               "select 1",
			Platform:            "windows",
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
	})
	require.NoError(t, err)

	// All memberships should be cleared (label query will repopulate with windows hosts on next execution)
	hosts, err = ds.ListHostsInLabel(ctx, fleet.TeamFilter{User: test.UserAdmin}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 0)
	// Re-seed membership to ensure this step exercises clearing again
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, hostWindows1, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	// Test 3: Change platform to empty (all platforms) - all memberships are cleared
	err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
		{
			Name:                "platform_test_label",
			Description:         "Test label for platform changes",
			Query:               "select 1",
			Platform:            "",
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
	})
	require.NoError(t, err)

	// All memberships should be cleared (label query will repopulate on next execution)
	hosts, err = ds.ListHostsInLabel(ctx, fleet.TeamFilter{User: test.UserAdmin}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 0)

	// Add all hosts back
	for _, h := range []*fleet.Host{hostDarwin1, hostDarwin2, hostLinux1} {
		require.NoError(t, ds.RecordLabelQueryExecutions(ctx, h, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))
	}

	// Test 4: Change from empty to specific platform (linux) - all memberships are cleared
	err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
		{
			Name:                "platform_test_label",
			Description:         "Test label for platform changes",
			Query:               "select 1",
			Platform:            "linux",
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
	})
	require.NoError(t, err)

	// All memberships should be cleared (label query will repopulate with linux hosts on next execution)
	hosts, err = ds.ListHostsInLabel(ctx, fleet.TeamFilter{User: test.UserAdmin}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 0)
}

type TestHostVitalsLabel struct {
	fleet.Label
}

func (t *TestHostVitalsLabel) CalculateHostVitalsQuery() (string, []interface{}, error) {
	return "SELECT %s FROM %s JOIN host_users ON (host_users.host_id = hosts.id) WHERE host_users.username = ?", []interface{}{"user1"}, nil
}

func (t *TestHostVitalsLabel) GetLabel() *fleet.Label {
	return &t.Label
}

func testUpdateLabelMembershipByHostCriteria(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	hosts := make([]*fleet.Host, 4)
	for i := 1; i <= 4; i++ {
		var teamID *uint
		if i == 1 || i == 2 {
			teamID = &team1.ID
		} else if i == 3 {
			teamID = &team2.ID
		}

		host, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:  ptr.String(fmt.Sprintf("%d", i)),
			NodeKey:        ptr.String(fmt.Sprintf("%d", i)),
			UUID:           fmt.Sprintf("uuid%d", i),
			Hostname:       fmt.Sprintf("host%d.local", i),
			HardwareSerial: fmt.Sprintf("hwd%d", i),
			Platform:       "darwin",
			TeamID:         teamID,
		})
		require.NoError(t, err)
		hosts[i-1] = host
	}
	// Add users to the hosts
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
		INSERT INTO host_users (host_id, uid, username) VALUES
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?)`,
			hosts[0].ID, 1, "user1",
			hosts[1].ID, 2, "user2",
			hosts[2].ID, 1, "user1",
			hosts[2].ID, 3, "user3",
			hosts[3].ID, 3, "user3")
		return err
	})

	criteria, err := json.Marshal(&fleet.HostVitalCriteria{
		Vital: ptr.String("username"),
		Value: ptr.String("user1"),
	})
	require.NoError(t, err)

	var ids []uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		for _, teamID := range []*uint{nil, &team1.ID, &team2.ID} {
			result, err := q.ExecContext(context.Background(),
				"INSERT INTO labels (name, description, platform, label_type, label_membership_type, query, team_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
				fmt.Sprintf("test host vitals label %d", teamID), "test", "", fleet.LabelTypeRegular, fleet.LabelMembershipTypeHostVitals, "", teamID)
			if err != nil {
				return err
			}
			id64, err := result.LastInsertId()
			if err != nil {
				return err
			}
			ids = append(ids, uint(id64)) // nolint:gosec
		}
		return nil
	})

	testCases := []struct {
		LabelID       uint
		TeamID        *uint
		BeforeHostIDs []uint
		AfterHostIDs  []uint
	}{
		{
			ids[0],
			nil,
			[]uint{hosts[0].ID, hosts[2].ID}, // Only hosts 1 and 3 should match the criteria (user1)
			[]uint{hosts[1].ID, hosts[2].ID, hosts[3].ID}, // Only hosts 2, 3 and 4 should match the criteria (user1)
		},
		{
			ids[1],
			&team1.ID,
			[]uint{hosts[0].ID}, // Only host 1 is on the team affected by the label
			[]uint{hosts[1].ID}, // Only host 2 is on the team affected by the label after vitals changes
		},
	}

	makeLabel := func(id uint, teamID *uint) *TestHostVitalsLabel {
		return &TestHostVitalsLabel{
			Label: fleet.Label{
				ID:                  id,
				TeamID:              teamID,
				Name:                fmt.Sprintf("Test Host Vitals Label %d", teamID),
				LabelType:           fleet.LabelTypeRegular,
				LabelMembershipType: fleet.LabelMembershipTypeHostVitals,
				HostVitalsCriteria:  ptr.RawMessage(criteria),
			},
		}
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	for _, tt := range testCases {
		updatedLabel, err := ds.UpdateLabelMembershipByHostCriteria(ctx, makeLabel(tt.LabelID, tt.TeamID))
		require.NoError(t, err)
		require.Equal(t, len(tt.BeforeHostIDs), updatedLabel.HostCount)

		// Check that the label has the correct hosts
		hostsInLabel, err := ds.ListHostsInLabel(ctx, filter, tt.LabelID, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Len(t, hostsInLabel, len(tt.BeforeHostIDs))
		labelHostIDs := make([]uint, 0, len(hostsInLabel))
		for _, host := range hostsInLabel {
			labelHostIDs = append(labelHostIDs, host.ID)
		}
		require.ElementsMatch(t, tt.BeforeHostIDs, labelHostIDs)
	}

	// Update host users.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
		INSERT INTO host_users (host_id, uid, username) VALUES
		(?, ?, ?),
		(?, ?, ?),
		(?, ?, ?) ON DUPLICATE KEY UPDATE username = VALUES(username), uid = VALUES(uid)`,
			hosts[0].ID, 2, "user2",
			hosts[1].ID, 1, "user1",
			hosts[3].ID, 1, "user1")
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
		DELETE FROM host_users WHERE host_id = ? AND uid = ?`,
			hosts[0].ID, 1) // Remove user1 from host 1
		return err
	})

	for _, tt := range testCases {
		updatedLabel, err := ds.UpdateLabelMembershipByHostCriteria(ctx, makeLabel(tt.LabelID, tt.TeamID))
		require.NoError(t, err)
		require.Equal(t, len(tt.AfterHostIDs), updatedLabel.HostCount)

		// Check that the label has the correct hosts
		hostsInLabel, err := ds.ListHostsInLabel(ctx, filter, tt.LabelID, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Len(t, hostsInLabel, len(tt.AfterHostIDs))
		labelHostIDs := make([]uint, 0, len(hostsInLabel))
		for _, host := range hostsInLabel {
			labelHostIDs = append(labelHostIDs, host.ID)
		}
		require.ElementsMatch(t, tt.AfterHostIDs, labelHostIDs)
	}
}

func testTeamLabels(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t1, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "t1",
	})
	require.NoError(t, err)
	t2, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "t2",
	})
	require.NoError(t, err)

	gl, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "g1",
		Query:               "SELECT 1;",
		TeamID:              nil,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		Platform:            "", // all platforms
	})
	require.NoError(t, err)

	l1t1, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "l1t1",
		Query:               "SELECT 2;",
		TeamID:              &t1.ID,
		Platform:            "darwin",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Manual label.
	l2t2, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "l2t2",
		TeamID:              &t2.ID,
		Platform:            "", // all platforms
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	windowsHostT1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("1"),
		NodeKey:       ptr.String("1"),
		UUID:          "1",
		Hostname:      "foo.local",
		Platform:      "windows",
		TeamID:        &t1.ID,
	})
	require.NoError(t, err)
	macOSHostT1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("2"),
		NodeKey:       ptr.String("2"),
		UUID:          "2",
		Hostname:      "foo2.local",
		Platform:      "darwin",
		TeamID:        &t1.ID,
	})
	require.NoError(t, err)
	linuxHostT2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("3"),
		NodeKey:       ptr.String("3"),
		UUID:          "3",
		Hostname:      "foo3.local",
		Platform:      "ubuntu",
		TeamID:        &t2.ID,
	})
	require.NoError(t, err)
	macOSHostGlobal, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("4"),
		NodeKey:       ptr.String("4"),
		UUID:          "4",
		Hostname:      "foo4.local",
		Platform:      "darwin",
		TeamID:        nil,
	})
	require.NoError(t, err)

	queries, err := ds.LabelQueriesForHost(ctx, macOSHostT1)
	require.NoError(t, err)
	require.Len(t, queries, 2)
	require.Equal(t, queries[fmt.Sprint(gl.ID)], "SELECT 1;")
	require.Equal(t, queries[fmt.Sprint(l1t1.ID)], "SELECT 2;")

	queries, err = ds.LabelQueriesForHost(ctx, windowsHostT1)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	require.Equal(t, queries[fmt.Sprint(gl.ID)], "SELECT 1;")

	queries, err = ds.LabelQueriesForHost(ctx, linuxHostT2)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	require.Equal(t, queries[fmt.Sprint(gl.ID)], "SELECT 1;")
	// l2t2 is not returned here because it's a manual label.

	// Add team (manual) label to host.
	err = ds.AddLabelsToHost(t.Context(), linuxHostT2.ID, []uint{l2t2.ID})
	require.NoError(t, err)
	hosts, err := ds.ListHostsInLabel(t.Context(), fleet.TeamFilter{User: test.UserAdmin}, l2t2.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, hosts[0].ID, linuxHostT2.ID)

	queries, err = ds.LabelQueriesForHost(ctx, macOSHostGlobal)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	require.Equal(t, queries[fmt.Sprint(gl.ID)], "SELECT 1;")
}

func testUpdateLabelMembershipForTransferredHost(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	t1, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "t1",
	})
	require.NoError(t, err)
	t2, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "t2",
	})
	require.NoError(t, err)

	macOSHostT1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("1"),
		NodeKey:       ptr.String("1"),
		UUID:          "1",
		Hostname:      "foo.local",
		Platform:      "darwin",
		TeamID:        &t1.ID,
	})
	require.NoError(t, err)
	windowsHostT2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: ptr.String("2"),
		NodeKey:       ptr.String("2"),
		UUID:          "2",
		Hostname:      "foo2.local",
		Platform:      "windows",
		TeamID:        &t2.ID,
	})
	require.NoError(t, err)

	globalLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "global",
		Query:               "SELECT 1;",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		TeamID:              nil,
	})
	require.NoError(t, err)
	l1t1, err := ds.NewLabel(ctx, &fleet.Label{
		Name:     "l1t1",
		Query:    "SELECT 2;",
		TeamID:   &t1.ID,
		Platform: "", // all platforms
	})
	require.NoError(t, err)
	l2t2, err := ds.NewLabel(ctx, &fleet.Label{
		Name:     "l2t2",
		Query:    "SELECT 3;",
		TeamID:   &t2.ID,
		Platform: "", // all platforms
	})
	require.NoError(t, err)

	err = ds.RecordLabelQueryExecutions(ctx, macOSHostT1, map[uint]*bool{
		globalLabel.ID: ptr.Bool(true),
		l1t1.ID:        ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordLabelQueryExecutions(ctx, windowsHostT2, map[uint]*bool{
		globalLabel.ID: ptr.Bool(true),
		l2t2.ID:        ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)

	// Move hosts to "No team".
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, []uint{macOSHostT1.ID, windowsHostT2.ID}))
	require.NoError(t, err)

	// Both hosts have their team label memberships erased, but the global label membership stays.
	labels, err := ds.ListLabelsForHost(ctx, macOSHostT1.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "global", labels[0].Name)
	labels, err = ds.ListLabelsForHost(ctx, windowsHostT2.ID)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	require.Equal(t, "global", labels[0].Name)
}

func testSetAsideLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1_setaside"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2_setaside"})
	require.NoError(t, err)

	i := 0
	newUser := func(u fleet.User) *fleet.User {
		i++
		u.Name = fmt.Sprintf("SetAsideUser%d", i)
		u.Email = fmt.Sprintf("%s@example.com", u.Name)
		u.Password = []byte("foobar")
		persisted, err := ds.NewUser(ctx, &u)
		require.NoError(t, err)
		return persisted
	}

	// Create users that are reused across tests
	globalAdmin := newUser(fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)})
	team1Admin := newUser(fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleAdmin}}})
	team1Maintainer := newUser(fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer}}})
	multiTeamAdmin := newUser(fleet.User{Teams: []fleet.UserTeam{
		{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleAdmin},
		{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleAdmin},
	}})
	multiTeamMaintainer := newUser(fleet.User{Teams: []fleet.UserTeam{
		{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer},
		{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleMaintainer},
	}})

	type labelSpec struct {
		name     string
		teamID   *uint
		authorID *uint
	}

	type testCase struct {
		name        string
		labels      []labelSpec // labels to create for this test
		notOnTeamID *uint       // team ID being applied to
		labelNames  []string    // names to pass to SetAsideLabels (if nil, uses labels[*].name)
		user        *fleet.User
		expectError bool
	}

	cases := []testCase{
		{
			name:        "empty names list is a no-op",
			labels:      nil,
			notOnTeamID: nil,
			labelNames:  []string{},
			user:        globalAdmin,
			expectError: false,
		},
		{
			name:        "global admin can set aside global labels when applying to a team",
			labels:      []labelSpec{{name: "global-setaside-1", teamID: nil, authorID: nil}},
			notOnTeamID: &team1.ID,
			user:        globalAdmin,
			expectError: false,
		},
		{
			name:        "global maintainer can set aside global labels",
			labels:      []labelSpec{{name: "global-setaside-2", teamID: nil, authorID: nil}},
			notOnTeamID: &team1.ID,
			user:        newUser(fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)}),
			expectError: false,
		},
		{
			name:        "global gitops can set aside global labels",
			labels:      []labelSpec{{name: "global-setaside-3", teamID: nil, authorID: nil}},
			notOnTeamID: &team1.ID,
			user:        newUser(fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)}),
			expectError: false,
		},
		{
			name:        "global observer cannot set aside global labels",
			labels:      []labelSpec{{name: "global-setaside-4", teamID: nil, authorID: nil}},
			notOnTeamID: &team1.ID,
			user:        newUser(fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}),
			expectError: true,
		},
		{
			name:        "team admin can't set aside their own team's labels if they can't edit the not-on team",
			labels:      []labelSpec{{name: "team1-setaside-1", teamID: &team1.ID, authorID: nil}},
			notOnTeamID: &team2.ID,
			user:        team1Admin,
			expectError: true,
		},
		{
			name:        "team admin can set aside their own team's labels if they can also edit the not-on team",
			labels:      []labelSpec{{name: "team1-setaside-admin", teamID: &team1.ID, authorID: nil}},
			notOnTeamID: &team2.ID,
			user:        multiTeamAdmin,
			expectError: false,
		},
		{
			name:        "team maintainer can set aside their own team's labels if they can also edit the not-on team",
			labels:      []labelSpec{{name: "team1-setaside-maintain", teamID: &team1.ID, authorID: nil}},
			notOnTeamID: &team2.ID,
			user:        multiTeamMaintainer,
			expectError: false,
		},
		{
			name:        "team gitops can set aside their own team's labels if they can also edit the not-on team",
			labels:      []labelSpec{{name: "team1-setaside-gitops", teamID: &team1.ID, authorID: nil}},
			notOnTeamID: &team2.ID,
			user: newUser(fleet.User{Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleGitOps},
				{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleGitOps},
			}}),
			expectError: false,
		},
		{
			name:        "team observer cannot set aside team labels",
			labels:      []labelSpec{{name: "team1-setaside-4", teamID: &team1.ID, authorID: nil}},
			notOnTeamID: &team2.ID,
			user:        newUser(fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleObserver}}}),
			expectError: true,
		},
		{
			name:        "cannot set aside labels from the same team we're applying to",
			labels:      []labelSpec{{name: "team1-setaside-5", teamID: &team1.ID, authorID: nil}},
			notOnTeamID: &team1.ID,
			user:        team1Admin,
			expectError: true,
		},
		{
			name:        "team admin cannot set aside labels when applying to the same team, even if that user authored",
			labels:      []labelSpec{{name: "team1-setaside-6", teamID: &team1.ID, authorID: &team1Admin.ID}},
			notOnTeamID: &team1.ID,
			user:        team1Admin,
			expectError: true,
		},
		{
			name:        "cannot set aside global labels when applying to global",
			labels:      []labelSpec{{name: "global-setaside-5", teamID: nil, authorID: nil}},
			notOnTeamID: nil,
			user:        globalAdmin,
			expectError: true,
		},
		{
			name:        "team user with write role can set aside their authored global labels",
			labels:      []labelSpec{{name: "global-authored-setaside", teamID: nil, authorID: &team1Maintainer.ID}},
			notOnTeamID: &team1.ID,
			user:        team1Maintainer,
			expectError: false,
		},
		{
			name:        "team user cannot set aside non-authored global labels",
			labels:      []labelSpec{{name: "global-nonauthored-setaside", teamID: nil, authorID: &globalAdmin.ID}},
			notOnTeamID: &team1.ID,
			user:        team1Maintainer,
			expectError: true,
		},
		{
			name:        "multi-team user can set aside labels from teams they have write access to",
			labels:      []labelSpec{{name: "team2-setaside-1", teamID: &team2.ID, authorID: nil}},
			notOnTeamID: &team1.ID,
			user: newUser(fleet.User{Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer},
				{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleAdmin},
			}}),
			expectError: false,
		},
		{
			name:        "multi-team user cannot set aside labels from teams they don't have write access to",
			labels:      []labelSpec{{name: "team2-setaside-2", teamID: &team2.ID, authorID: nil}},
			notOnTeamID: &team1.ID,
			user: newUser(fleet.User{Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team1.ID}, Role: fleet.RoleMaintainer},
				{Team: fleet.Team{ID: team2.ID}, Role: fleet.RoleObserver},
			}}),
			expectError: true,
		},
		{
			name:        "non-existent label should fail",
			labels:      nil,
			notOnTeamID: &team1.ID,
			labelNames:  []string{"nonexistent-label"},
			user:        globalAdmin,
			expectError: true,
		},
		{
			name: "multiple labels can be set aside at once",
			labels: []labelSpec{
				{name: "multi-setaside-1", teamID: nil, authorID: &multiTeamMaintainer.ID},
				{name: "multi-setaside-2", teamID: &team2.ID, authorID: nil},
			},
			notOnTeamID: &team1.ID,
			user:        multiTeamMaintainer,
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create labels for this test case
			var expectedLabelNames []string
			for _, spec := range tc.labels {
				// Build expected renamed name based on label's team ID
				var expectedSuffix string
				if spec.teamID == nil {
					expectedSuffix = "__team_0"
				} else {
					expectedSuffix = fmt.Sprintf("__team_%d", *spec.teamID)
				}
				expectedLabelNames = append(expectedLabelNames, spec.name+expectedSuffix)

				_, err := ds.NewLabel(ctx, &fleet.Label{
					Name:                spec.name,
					Query:               "SELECT 1",
					TeamID:              spec.teamID,
					AuthorID:            spec.authorID,
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				})
				require.NoError(t, err)
			}

			// Determine label names to use, and which to expect
			labelNames := tc.labelNames
			if labelNames == nil {
				for _, spec := range tc.labels {
					labelNames = append(labelNames, spec.name)
				}
			}

			err := ds.SetAsideLabels(ctx, tc.notOnTeamID, labelNames, *tc.user)

			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Original name should not exist
			oldLabels, _ := ds.LabelsByName(ctx, labelNames, fleet.TeamFilter{User: globalAdmin})
			require.Empty(t, oldLabels)

			// All renamed labels should exist
			renamedLabels, _ := ds.LabelsByName(ctx, expectedLabelNames, fleet.TeamFilter{User: globalAdmin})
			require.Len(t, renamedLabels, len(expectedLabelNames))
			for _, expected := range expectedLabelNames {
				_, ok := renamedLabels[expected]
				require.Truef(t, ok, "Missing renamed label %s", expected)
			}
		})
	}
}

func testApplyLabelSpecsWithManualTeamLabels(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	teamFilter := fleet.TeamFilter{User: test.UserAdmin}

	// Create teams.
	t1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	t2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	t3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team3"})
	require.NoError(t, err)
	t4, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team4"})
	require.NoError(t, err)

	// Create hosts on the teams.
	h1t1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("h1t1"),
		NodeKey:        ptr.String("h1t1"),
		Hostname:       "hostname-h1t1",
		HardwareSerial: "serial-h1t1",
		UUID:           "uuid-h1t1",
		Platform:       "darwin",
		TeamID:         &t1.ID,
	})
	require.NoError(t, err)
	h2t2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("h2t2"),
		NodeKey:        ptr.String("h2t2"),
		Hostname:       "hostname-h2t2",
		HardwareSerial: "serial-h2t2",
		UUID:           "uuid-h2t2",
		Platform:       "darwin",
		TeamID:         &t2.ID,
	})
	require.NoError(t, err)
	h3t3, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("h3t3"),
		NodeKey:        ptr.String("h3t3"),
		Hostname:       "hostname-h3t3",
		HardwareSerial: "serial-h3t3",
		UUID:           "uuid-h3t3",
		Platform:       "darwin",
		TeamID:         &t3.ID,
	})
	require.NoError(t, err)
	h4t4, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("h4t4"),
		NodeKey:        ptr.String("h4t4"),
		Hostname:       "hostname-h4t4",
		HardwareSerial: "serial-h4t4",
		UUID:           "uuid-h4t4",
		Platform:       "darwin",
		TeamID:         &t4.ID,
	})
	require.NoError(t, err)
	h5Global, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:  ptr.String("h5Global"),
		NodeKey:        ptr.String("h5Global"),
		Hostname:       "hostname-h5Global",
		HardwareSerial: "serial-h5Global",
		UUID:           "uuid-h5Global",
		Platform:       "darwin",
		TeamID:         nil,
	})
	require.NoError(t, err)

	// Create a global manual label, make sure you can add all.
	err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
		{
			Name:                "global1",
			LabelMembershipType: fleet.LabelMembershipTypeManual,
			Hosts: fleet.HostsSlice{
				h1t1.Hostname,
				h2t2.HardwareSerial,
				h3t3.UUID,
				fmt.Sprint(h4t4.ID),
				h5Global.Hostname,
			},
		},
	})
	require.NoError(t, err)

	global1, err := ds.LabelByName(ctx, "global1", teamFilter)
	require.NoError(t, err)
	hosts, err := ds.ListHostsInLabel(ctx, teamFilter, global1.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 5)

	// Attempt to create team label, make sure we can only add hosts on that team.
	for _, hostIdentifier := range []string{
		h2t2.Hostname,
		h3t3.UUID,
		h4t4.HardwareSerial,
		fmt.Sprint(h5Global.ID),
	} {
		err := ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
			{
				Name:                "l1t1",
				LabelMembershipType: fleet.LabelMembershipTypeManual,
				Hosts: fleet.HostsSlice{
					h1t1.Hostname,
					hostIdentifier, // conflicting host identifier.
				},
				TeamID: &t1.ID,
			},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, errLabelMismatchHostTeam)
	}
	// Create team label with team host identifiers should work.
	for _, hostIdentifier := range []string{
		h1t1.Hostname,
		h1t1.UUID,
		h1t1.HardwareSerial,
		fmt.Sprint(h1t1.ID),
	} {
		err = ds.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
			{
				Name:                "l1t1",
				LabelMembershipType: fleet.LabelMembershipTypeManual,
				Hosts: fleet.HostsSlice{
					hostIdentifier,
				},
				TeamID: &t1.ID,
			},
		})
		require.NoError(t, err)
		l1t1, err := ds.LabelByName(ctx, "l1t1", teamFilter)
		require.NoError(t, err)
		hosts, err := ds.ListHostsInLabel(ctx, teamFilter, l1t1.ID, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Len(t, hosts, 1)
		require.Equal(t, h1t1.ID, hosts[0].ID)
	}
}
