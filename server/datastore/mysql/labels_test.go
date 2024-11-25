package mysql

import (
	"context"
	"fmt"
	"strconv"
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
	for i := 0; i < 230000; i++ {
		large = append(large, strconv.Itoa(i))
	}
	batched := batchHostnames(large)
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
		{"ListUniqueHostsInLabels", testLabelsListUniqueHostsInLabels},
		{"ChangeDetails", testLabelsChangeDetails},
		{"GetSpec", testLabelsGetSpec},
		{"ApplySpecsRoundtrip", testLabelsApplySpecsRoundtrip},
		{"IDsByName", testLabelsIDsByName},
		{"ByName", testLabelsByName},
		{"Save", testLabelsSave},
		{"QueriesForCentOSHost", testLabelsQueriesForCentOSHost},
		{"RecordNonExistentQueryLabelExecution", testLabelsRecordNonexistentQueryLabelExecution},
		{"DeleteLabel", testDeleteLabel},
		{"LabelsSummary", testLabelsSummary},
		{"ListHostsInLabelIssues", testListHostsInLabelIssues},
		{"ListHostsInLabelDiskEncryptionStatus", testListHostsInLabelDiskEncryptionStatus},
		{"HostMemberOfAllLabels", testHostMemberOfAllLabels},
		{"ListHostsInLabelOSSettings", testLabelsListHostsInLabelOSSettings},
		{"AddDeleteLabelsToFromHost", testAddDeleteLabelsToFromHost},
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
		host, err = db.EnrollHost(context.Background(), false, fmt.Sprint(i), "", "", fmt.Sprint(i), nil, 0)
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
	assert.Contains(t, labels, all)

	labels, err = db.SearchLabels(context.Background(), filter, "foo")
	require.Nil(t, err)
	assert.Len(t, labels, 3)
	assert.Contains(t, labels, all)

	labels, err = db.SearchLabels(context.Background(), filter, "foo", all.ID, l3.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "foo", labels[0].Name)

	labels, err = db.SearchLabels(context.Background(), filter, "xxx")
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Contains(t, labels, all)
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
	require.NoError(t, db.SetOrUpdateHostDisksSpace(context.Background(), h1.ID, 10, 5, 200.0))
	require.NoError(t, db.SetOrUpdateHostDisksSpace(context.Background(), h2.ID, 20, 10, 200.1))
	require.NoError(t, db.SetOrUpdateHostDisksSpace(context.Background(), h3.ID, 30, 15, 200.2))

	ctx := context.Background()
	const simpleMDM, kandji = "https://simplemdm.com", "https://kandji.io"
	err = db.SetOrUpdateMDMData(ctx, h1.ID, false, true, simpleMDM, true, fleet.WellKnownMDMSimpleMDM, "") // enrollment: automatic
	require.NoError(t, err)
	err = db.SetOrUpdateMDMData(ctx, h2.ID, false, true, kandji, true, fleet.WellKnownMDMKandji, "") // enrollment: automatic
	require.NoError(t, err)
	err = db.SetOrUpdateMDMData(ctx, h3.ID, false, false, simpleMDM, false, fleet.WellKnownMDMSimpleMDM, "") // enrollment: unenrolled
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

	require.NoError(t, db.AddHostsToTeam(context.Background(), &team1.ID, []uint{h1.ID}))

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

func testLabelsListUniqueHostsInLabels(t *testing.T, db *Datastore) {
	hosts := make([]*fleet.Host, 4)
	for i := range hosts {
		h, err := db.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String(strconv.Itoa(i)),
			NodeKey:         ptr.String(strconv.Itoa(i)),
			UUID:            strconv.Itoa(i),
			Hostname:        fmt.Sprintf("host_%d", i),
		})
		require.Nil(t, err)
		hosts[i] = h
	}

	team1, err := db.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, db.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID}))

	l1 := fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	l2 := fleet.LabelSpec{
		ID:    2,
		Name:  "label bar",
		Query: "query2",
	}
	require.NoError(t, db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{&l1, &l2}))

	for i := 0; i < 3; i++ {
		err = db.RecordLabelQueryExecutions(context.Background(), hosts[i], map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		assert.Nil(t, err)
	}
	// host 2 executes twice
	for i := 2; i < len(hosts); i++ {
		err = db.RecordLabelQueryExecutions(context.Background(), hosts[i], map[uint]*bool{l2.ID: ptr.Bool(true)}, time.Now(), false)
		assert.Nil(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	uniqueHosts, err := db.ListUniqueHostsInLabels(context.Background(), filter, []uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, len(hosts), len(uniqueHosts))

	labels, err := db.ListLabels(context.Background(), filter, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, labels, 2)
	for _, l := range labels {
		assert.True(t, l.HostCount > 0)
	}

	userObs := &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)}
	filter = fleet.TeamFilter{User: userObs}

	// observer not included
	uniqueHosts, err = db.ListUniqueHostsInLabels(context.Background(), filter, []uint{l1.ID, l2.ID})
	require.Nil(t, err)
	assert.Len(t, uniqueHosts, 0)

	labels, err = db.ListLabels(context.Background(), filter, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, labels, 2)
	for _, l := range labels {
		assert.Equal(t, 0, l.HostCount)
	}

	// observer included
	filter.IncludeObserver = true
	uniqueHosts, err = db.ListUniqueHostsInLabels(context.Background(), filter, []uint{l1.ID, l2.ID})
	require.Nil(t, err)
	assert.Len(t, uniqueHosts, len(hosts))

	labels, err = db.ListLabels(context.Background(), filter, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, labels, 2)
	for _, l := range labels {
		assert.True(t, l.HostCount > 0)
	}

	userTeam1 := &fleet.User{Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}}}
	filter = fleet.TeamFilter{User: userTeam1}

	uniqueHosts, err = db.ListUniqueHostsInLabels(context.Background(), filter, []uint{l1.ID, l2.ID})
	require.Nil(t, err)
	require.Len(t, uniqueHosts, 1) // only host 0 associated with this team
	assert.Equal(t, hosts[0].ID, uniqueHosts[0].ID)

	labels, err = db.ListLabels(context.Background(), filter, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, labels, 2)
	for _, l := range labels {
		if l.ID == l1.ID {
			assert.Equal(t, 1, l.HostCount)
		} else {
			assert.Equal(t, 0, l.HostCount)
		}
	}
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
	profA, err := db.NewMDMAppleConfigProfile(context.Background(), *generateCP("a", "a", 0))
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
	saved2, _, err := db.SaveLabel(context.Background(), saved, filter)
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
			UUID:            strconv.Itoa(i),
			Hostname:        strconv.Itoa(i),
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

	return expectedSpecs
}

func testLabelsGetSpec(t *testing.T, ds *Datastore) {
	expectedSpecs := setupLabelSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetLabelSpec(context.Background(), s.Name)
		require.Nil(t, err)

		require.True(t, cmp.Equal(s, spec, cmp.FilterPath(func(p cmp.Path) bool {
			return p.String() == "ID"
		}, cmp.Ignore())))
	}
}

func testLabelsApplySpecsRoundtrip(t *testing.T, ds *Datastore) {
	expectedSpecs := setupLabelSpecsTest(t, ds)

	specs, err := ds.GetLabelSpecs(context.Background())
	require.Nil(t, err)
	test.ElementsMatchSkipTimestampsID(t, expectedSpecs, specs)

	// Should be idempotent
	err = ds.ApplyLabelSpecs(context.Background(), expectedSpecs)
	require.Nil(t, err)
	specs, err = ds.GetLabelSpecs(context.Background())
	require.Nil(t, err)
	test.ElementsMatchSkipTimestampsID(t, expectedSpecs, specs)
}

func testLabelsIDsByName(t *testing.T, ds *Datastore) {
	setupLabelSpecsTest(t, ds)

	labels, err := ds.LabelIDsByName(context.Background(), []string{"foo", "bar", "bing"})
	require.Nil(t, err)
	assert.Equal(t, map[string]uint{"foo": 1, "bar": 2, "bing": 3}, labels)
}

func testLabelsByName(t *testing.T, ds *Datastore) {
	setupLabelSpecsTest(t, ds)

	names := []string{"foo", "bar", "bing"}
	labels, err := ds.LabelsByName(context.Background(), names)
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

	label := &fleet.Label{
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes;",
		Platform:    "darwin",
	}
	label, err = db.NewLabel(context.Background(), label)
	require.NoError(t, err)

	// Create an Apple config profile
	profA, err := db.NewMDMAppleConfigProfile(context.Background(), *generateCP("a", "a", 0))
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
	host, err := db.EnrollHost(context.Background(), false, "0", "", "", "0", nil, 0)
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
	l, err := db.NewLabel(context.Background(), &fleet.Label{
		Name:  t.Name(),
		Query: "query1",
	})
	require.NoError(t, err)

	p, err := db.NewPack(context.Background(), &fleet.Pack{
		Name:     t.Name(),
		LabelIDs: []uint{l.ID},
	})
	require.NoError(t, err)

	require.NoError(t, db.DeleteLabel(context.Background(), l.Name))

	newP, err := db.Pack(context.Background(), p.ID)
	require.NoError(t, err)
	require.Empty(t, newP.Labels)

	require.NoError(t, db.DeletePack(context.Background(), newP.Name))
}

func testLabelsSummary(t *testing.T, db *Datastore) {
	test.AddAllHostsLabel(t, db)

	// Only 'All Hosts' label should be returned
	labels, err := db.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
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

	labels, err = db.ListLabels(context.Background(), fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, labels, 4)
	labelsByID := make(map[uint]*fleet.Label)
	for _, l := range labels {
		labelsByID[l.ID] = l
	}

	ls, err := db.LabelsSummary(context.Background())
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

	ls, err = db.LabelsSummary(context.Background())
	require.NoError(t, err)
	require.Len(t, ls, 5)
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
	noTeamFVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("filevault-1", "com.fleetdm.fleet.mdm.filevault", 0))
	require.NoError(t, err)

	// verifying status
	upsertHostCPs([]*fleet.Host{hosts[0], hosts[1]}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
	createDiskEncryptionRecord(ctx, ds, t, hosts[0].ID, "key-1", true, oneMinuteAfterThreshold)
	createDiskEncryptionRecord(ctx, ds, t, hosts[1].ID, "key-1", true, oneMinuteAfterThreshold)

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
		nanoEnrollAndSetHostMDMData(t, db, h, false)
	}
	// add disk encryption key for h1
	require.NoError(t, db.SetOrUpdateHostDiskEncryptionKey(context.Background(), h1.ID, "test-key", "", ptr.Bool(true)))
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

	t.Run("os_settings", func(t *testing.T) {
		hosts = listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionVerified}, 1)
		checkHosts(t, hosts, []uint{h1.ID})
		hosts = listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionEnforcing}, 1)
		checkHosts(t, hosts, []uint{h2.ID})
	})

	t.Run("os_settings_disk_encryption", func(t *testing.T) {
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
	allLbls, err := ds.ListLabels(context.Background(), fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
	require.Nil(t, err)
	for _, lbl := range allLbls {
		if lbl.Name == name {
			return lbl.ID
		}
	}
	return 0
}
