package mysql

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"Save", testLabelsSave},
		{"QueriesForCentOSHost", testLabelsQueriesForCentOSHost},
		{"RecordNonExistentQueryLabelExecution", testLabelsRecordNonexistentQueryLabelExecution},
		{"LabelMembershipCleanup", testLabelMembershipCleanup},
	}
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
		host, err = db.EnrollHost(context.Background(), fmt.Sprint(i), fmt.Sprint(i), nil, 0)
		require.Nil(t, err, "enrollment should succeed")
		hosts = append(hosts, *host)
	}
	host.Platform = "darwin"
	require.NoError(t, db.SaveHost(context.Background(), host))

	// No labels to check
	queries, err := db.LabelQueriesForHost(context.Background(), host)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	// Only 'All Hosts' label should be returned
	labels, err := db.ListLabelsForHost(context.Background(), host.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)

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

	host, err = db.Host(context.Background(), host.ID, false)
	require.NoError(t, err)
	host.LabelUpdatedAt = baseTime

	// A new label targeting another platform should not effect the labels for
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

	all, err := db.Label(context.Background(), specs[len(specs)-1].ID)
	require.Nil(t, err)
	l3, err := db.Label(context.Background(), specs[2].ID)
	require.Nil(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

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
		OsqueryHostID:   "1",
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "2",
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "3",
		NodeKey:         "3",
		UUID:            "3",
		Hostname:        "baz.local",
	})
	require.Nil(t, err)

	l1 := &fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{l1})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{}, 0)
		assert.Len(t, hosts, 0)
	}

	for _, h := range []*fleet.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{}, 3)
		assert.Len(t, hosts, 3)
	}
}

func listHostsInLabelCheckCount(
	t *testing.T, db *Datastore, filter fleet.TeamFilter, labelID uint, opt fleet.HostListOptions, expectedCount int,
) []*fleet.Host {
	hosts, err := db.ListHostsInLabel(context.Background(), filter, labelID, opt)
	require.NoError(t, err)
	count, err := db.CountHostsInLabel(context.Background(), filter, labelID, opt)
	require.NoError(t, err)
	require.Equal(t, expectedCount, count)
	return hosts
}

func testLabelsListHostsInLabelAndStatus(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "1",
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	lastSeenTime := time.Now().Add(-1000 * time.Hour)
	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
		PolicyUpdatedAt: lastSeenTime,
		SeenTime:        lastSeenTime,
		OsqueryHostID:   "2",
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)
	h3, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
		PolicyUpdatedAt: lastSeenTime,
		SeenTime:        lastSeenTime,
		OsqueryHostID:   "3",
		NodeKey:         "3",
		UUID:            "3",
		Hostname:        "baz.local",
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
		require.Len(t, hosts, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusMIA}, 2)
		require.Len(t, hosts, 2)
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
		OsqueryHostID:   "1",
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.Nil(t, err)

	lastSeenTime := time.Now().Add(-1000 * time.Hour)
	h2, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
		PolicyUpdatedAt: lastSeenTime,
		SeenTime:        lastSeenTime,
		OsqueryHostID:   "2",
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
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

	filter := fleet.TeamFilter{User: test.UserAdmin}
	for _, h := range []*fleet.Host{h1, h2} {
		err = db.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{l1.ID: ptr.Bool(true)}, time.Now(), deferred)
		assert.Nil(t, err)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusOnline}, 1)
		require.Len(t, hosts, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusMIA}, 1)
		require.Len(t, hosts, 1)
		assert.Equal(t, "bar.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{TeamFilter: &team1.ID}, 1)
		require.Len(t, hosts, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts := listHostsInLabelCheckCount(t, db, filter, l1.ID, fleet.HostListOptions{TeamFilter: &team2.ID}, 0)
		require.Len(t, hosts, 0)
	}
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
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         strconv.Itoa(i),
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

	saved, err := db.Label(context.Background(), label.ID)
	require.Nil(t, err)
	assert.Equal(t, label.Name, saved.Name)
}

func setupLabelSpecsTest(t *testing.T, ds fleet.Datastore) []*fleet.LabelSpec {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         strconv.Itoa(i),
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
		assert.Equal(t, s, spec)
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
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
	assert.Equal(t, []uint{1, 2, 3}, labels)
}

func testLabelsSave(t *testing.T, db *Datastore) {
	h1, err := db.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "1",
		NodeKey:         "1",
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
	label.Name = "changed name"
	label.Description = "changed description"

	require.NoError(t, db.RecordLabelQueryExecutions(context.Background(), h1, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	_, err = db.SaveLabel(context.Background(), label)
	require.NoError(t, err)
	saved, err := db.Label(context.Background(), label.ID)
	require.NoError(t, err)
	assert.Equal(t, label.Name, saved.Name)
	assert.Equal(t, label.Description, saved.Description)
	assert.Equal(t, 1, saved.HostCount)
}

func testLabelsQueriesForCentOSHost(t *testing.T, db *Datastore) {
	host, err := db.EnrollHost(context.Background(), "0", "0", nil, 0)
	require.Nil(t, err, "enrollment should succeed")
	host.Platform = "rhel"
	host.OSVersion = "CentOS 6"
	require.NoError(t, db.SaveHost(context.Background(), host))

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
		OsqueryHostID:   "1",
		NodeKey:         "1",
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

func testLabelMembershipCleanup(t *testing.T, ds *Datastore) {
	setupTest := func() (*fleet.Host, *fleet.Label) {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         "1",
			UUID:            "1",
			Hostname:        "foo.local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
			OsqueryHostID:   "1",
		})
		require.NoError(t, err)
		require.NotNil(t, host)

		label := &fleet.Label{
			Name:  "foo",
			Query: "select * from foo;",
		}
		label, err = ds.NewLabel(context.Background(), label)
		require.NoError(t, err)

		require.NoError(t, ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))
		return host, label
	}

	checkCount := func(before, after int) {
		var count int
		require.NoError(t, ds.writer.Get(&count, `SELECT count(*) FROM label_membership`))
		assert.Equal(t, before, count)

		require.NoError(t, ds.CleanupOrphanLabelMembership(context.Background()))

		require.NoError(t, ds.writer.Get(&count, `SELECT count(*) FROM label_membership`))
		assert.Equal(t, after, count)
	}

	t.Run("none gone", func(t *testing.T) {
		host, label := setupTest()
		checkCount(1, 1)
		require.NoError(t, ds.DeleteHost(context.Background(), host.ID))
		require.NoError(t, ds.DeleteLabel(context.Background(), label.Name))
		require.NoError(t, ds.CleanupOrphanLabelMembership(context.Background()))
	})
	t.Run("label gone", func(t *testing.T) {
		host, label := setupTest()
		require.NoError(t, ds.DeleteLabel(context.Background(), label.Name))
		checkCount(1, 0)
		require.NoError(t, ds.DeleteHost(context.Background(), host.ID))
	})
	t.Run("host gone", func(t *testing.T) {
		host, label := setupTest()
		require.NoError(t, ds.DeleteHost(context.Background(), host.ID))
		checkCount(1, 0)
		require.NoError(t, ds.DeleteLabel(context.Background(), label.Name))
	})
	t.Run("both gone", func(t *testing.T) {
		host, label := setupTest()
		require.NoError(t, ds.DeleteHost(context.Background(), host.ID))
		require.NoError(t, ds.DeleteLabel(context.Background(), label.Name))
		checkCount(1, 0)
	})
}
