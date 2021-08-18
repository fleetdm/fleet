package mysql

import (
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
	db := CreateMySQLDS(t)
	defer db.Close()

	test.AddAllHostsLabel(t, db)
	hosts := []fleet.Host{}
	var host *fleet.Host
	var err error
	for i := 0; i < 10; i++ {
		host, err = db.EnrollHost(fmt.Sprint(i), fmt.Sprint(i), nil, 0)
		require.Nil(t, err, "enrollment should succeed")
		hosts = append(hosts, *host)
	}
	host.Platform = "darwin"
	require.NoError(t, db.SaveHost(host))

	baseTime := time.Now()

	// No labels to check
	queries, err := db.LabelQueriesForHost(host, baseTime)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	// Only 'All Hosts' label should be returned
	labels, err := db.ListLabelsForHost(host.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)

	newLabels := []*fleet.LabelSpec{
		// Note these are intentionally out of order
		&fleet.LabelSpec{
			Name:     "label3",
			Query:    "query3",
			Platform: "darwin",
		},
		&fleet.LabelSpec{
			Name:  "label1",
			Query: "query1",
		},
		&fleet.LabelSpec{
			Name:     "label2",
			Query:    "query2",
			Platform: "darwin",
		},
		&fleet.LabelSpec{
			Name:     "label4",
			Query:    "query4",
			Platform: "darwin",
		},
	}
	err = db.ApplyLabelSpecs(newLabels)
	require.Nil(t, err)

	expectQueries := map[string]string{
		"2": "query3",
		"3": "query1",
		"4": "query2",
		"5": "query4",
	}

	host.Platform = "darwin"

	// Now queries should be returned
	queries, err = db.LabelQueriesForHost(host, baseTime)
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// No labels should match with no results yet
	labels, err = db.ListLabelsForHost(host.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)

	// Record a query execution
	err = db.RecordLabelQueryExecutions(host, map[uint]bool{1: true, 2: false, 3: true, 4: false, 5: false}, baseTime)
	assert.Nil(t, err)

	host, err = db.Host(host.ID)
	require.NoError(t, err)
	host.LabelUpdatedAt = baseTime

	// Now no queries should be returned
	queries, err = db.LabelQueriesForHost(host, baseTime.Add(-1*time.Minute))
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	// Ensure enough gap in created_at
	time.Sleep(2 * time.Second)

	// A new label targeting another platform should not effect the labels for
	// this host
	err = db.ApplyLabelSpecs([]*fleet.LabelSpec{
		&fleet.LabelSpec{
			Name:     "label5",
			Platform: "not-matching",
			Query:    "query5",
		},
	})
	require.NoError(t, err)
	queries, err = db.LabelQueriesForHost(host, baseTime.Add(-1*time.Minute))
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	// If a new label is added, all labels should be returned
	err = db.ApplyLabelSpecs([]*fleet.LabelSpec{
		&fleet.LabelSpec{
			Name:     "label6",
			Platform: "",
			Query:    "query6",
		},
	})
	require.NoError(t, err)
	expectQueries["7"] = "query6"
	queries, err = db.LabelQueriesForHost(host, baseTime.Add(-1*time.Minute))
	assert.Nil(t, err)
	assert.Len(t, queries, 5)

	// After expiration, all queries should be returned
	queries, err = db.LabelQueriesForHost(host, baseTime.Add((2 * time.Minute)))
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// Now the two matching labels should be returned
	labels, err = db.ListLabelsForHost(host.ID)
	assert.Nil(t, err)
	if assert.Len(t, labels, 2) {
		labelNames := []string{labels[0].Name, labels[1].Name}
		sort.Strings(labelNames)
		assert.Equal(t, "All Hosts", labelNames[0])
		assert.Equal(t, "label1", labelNames[1])
	}

	// A host that hasn't executed any label queries should still be asked
	// to execute those queries
	hosts[0].Platform = "darwin"
	queries, err = db.LabelQueriesForHost(&hosts[0], time.Now())
	assert.Nil(t, err)
	assert.Len(t, queries, 5)

	// Only the 'All Hosts' label should apply for a host with no labels
	// executed.
	labels, err = db.ListLabelsForHost(hosts[0].ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)
}

func TestSearchLabels(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	specs := []*fleet.LabelSpec{
		&fleet.LabelSpec{
			ID:   1,
			Name: "foo",
		},
		&fleet.LabelSpec{
			ID:   2,
			Name: "bar",
		},
		&fleet.LabelSpec{
			ID:   3,
			Name: "foo-bar",
		},
		&fleet.LabelSpec{
			ID:        4,
			Name:      "All Hosts",
			LabelType: fleet.LabelTypeBuiltIn,
		},
	}
	err := db.ApplyLabelSpecs(specs)
	require.Nil(t, err)

	all, err := db.Label(specs[3].ID)
	require.Nil(t, err)
	l3, err := db.Label(specs[2].ID)
	require.Nil(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	labels, err := db.SearchLabels(filter, "")
	require.Nil(t, err)
	assert.Contains(t, labels, all)

	labels, err = db.SearchLabels(filter, "foo")
	require.Nil(t, err)
	assert.Len(t, labels, 3)
	assert.Contains(t, labels, all)

	labels, err = db.SearchLabels(filter, "foo", all.ID, l3.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "foo", labels[0].Name)

	labels, err = db.SearchLabels(filter, "xxx")
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Contains(t, labels, all)
}

func TestSearchLabelsLimit(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	if db.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	all := &fleet.LabelSpec{
		Name:      "All Hosts",
		LabelType: fleet.LabelTypeBuiltIn,
	}
	err := db.ApplyLabelSpecs([]*fleet.LabelSpec{all})
	require.Nil(t, err)

	for i := 0; i < 15; i++ {
		l := &fleet.LabelSpec{
			Name: fmt.Sprintf("foo%d", i),
		}
		err := db.ApplyLabelSpecs([]*fleet.LabelSpec{l})
		require.Nil(t, err)
	}

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	labels, err := db.SearchLabels(filter, "foo")
	require.Nil(t, err)
	assert.Len(t, labels, 11)
}

func TestListHostsInLabel(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	h1, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "1",
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "2",
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
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
	err = db.ApplyLabelSpecs([]*fleet.LabelSpec{l1})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{})
		require.Nil(t, err)
		assert.Len(t, hosts, 0)
	}

	for _, h := range []*fleet.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{})
		require.Nil(t, err)
		assert.Len(t, hosts, 3)
	}
}

func TestListHostsInLabelAndStatus(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	h1, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "1",
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.Nil(t, err)

	lastSeenTime := time.Now().Add(-1000 * time.Hour)
	h2, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
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
	err = db.ApplyLabelSpecs([]*fleet.LabelSpec{l1})
	require.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	for _, h := range []*fleet.Host{h1, h2} {
		err = db.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusOnline})
		require.Nil(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusMIA})
		require.Nil(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "bar.local", hosts[0].Hostname)
	}
}

func TestListHostsInLabelAndTeamFilter(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	h1, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   "1",
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.Nil(t, err)

	lastSeenTime := time.Now().Add(-1000 * time.Hour)
	h2, err := db.NewHost(&fleet.Host{
		DetailUpdatedAt: lastSeenTime,
		LabelUpdatedAt:  lastSeenTime,
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
	err = db.ApplyLabelSpecs([]*fleet.LabelSpec{l1})
	require.Nil(t, err)

	team1, err := db.NewTeam(&fleet.Team{Name: "team1"})
	require.NoError(t, err)

	team2, err := db.NewTeam(&fleet.Team{Name: "team2"})
	require.NoError(t, err)

	db.AddHostsToTeam(&team1.ID, []uint{h1.ID})

	filter := fleet.TeamFilter{User: test.UserAdmin}
	for _, h := range []*fleet.Host{h1, h2} {
		err = db.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusOnline})
		require.Nil(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{StatusFilter: fleet.StatusMIA})
		require.Nil(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "bar.local", hosts[0].Hostname)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{TeamFilter: &team1.ID})
		require.Nil(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "foo.local", hosts[0].Hostname)
	}

	{
		hosts, err := db.ListHostsInLabel(filter, l1.ID, fleet.HostListOptions{TeamFilter: &team2.ID})
		require.Nil(t, err)
		require.Len(t, hosts, 0)
	}
}

func TestBuiltInLabels(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	require.Nil(t, db.MigrateData())

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	hits, err := db.SearchLabels(filter, "macOS")
	require.Nil(t, err)
	// Should get Mac OS X and All Hosts
	assert.Equal(t, 2, len(hits))
	assert.Equal(t, fleet.LabelTypeBuiltIn, hits[0].LabelType)
	assert.Equal(t, fleet.LabelTypeBuiltIn, hits[1].LabelType)
}

func TestListUniqueHostsInLabels(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	hosts := []*fleet.Host{}
	for i := 0; i < 4; i++ {
		h, err := db.NewHost(&fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         strconv.Itoa(i),
			UUID:            strconv.Itoa(i),
			Hostname:        fmt.Sprintf("host_%d", i),
		})
		require.Nil(t, err)
		require.NotNil(t, h)
		hosts = append(hosts, h)
	}

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
	err := db.ApplyLabelSpecs([]*fleet.LabelSpec{&l1, &l2})
	require.Nil(t, err)

	for i := 0; i < 3; i++ {
		err = db.RecordLabelQueryExecutions(hosts[i], map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}
	// host 2 executes twice
	for i := 2; i < len(hosts); i++ {
		err = db.RecordLabelQueryExecutions(hosts[i], map[uint]bool{l2.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	filter := fleet.TeamFilter{User: test.UserAdmin}

	uniqueHosts, err := db.ListUniqueHostsInLabels(filter, []uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, len(hosts), len(uniqueHosts))

	labels, err := db.ListLabels(filter, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, labels, 2)

}

func TestChangeLabelDetails(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	if db.Name() == "inmem" {
		t.Skip("inmem is being deprecated")
	}

	label := fleet.LabelSpec{
		ID:          1,
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes",
		Platform:    "darwin",
	}
	err := db.ApplyLabelSpecs([]*fleet.LabelSpec{&label})
	require.Nil(t, err)

	label.Description = "changed description"
	err = db.ApplyLabelSpecs([]*fleet.LabelSpec{&label})
	require.Nil(t, err)

	saved, err := db.Label(label.ID)
	require.Nil(t, err)
	assert.Equal(t, label.Name, saved.Name)
}

func setupLabelSpecsTest(t *testing.T, ds fleet.Datastore) []*fleet.LabelSpec {
	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(&fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
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
	err := ds.ApplyLabelSpecs(expectedSpecs)
	require.Nil(t, err)

	return expectedSpecs
}

func TestGetLabelSpec(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	expectedSpecs := setupLabelSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetLabelSpec(s.Name)
		require.Nil(t, err)
		assert.Equal(t, s, spec)
	}
}

func TestApplyLabelSpecsRoundtrip(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	expectedSpecs := setupLabelSpecsTest(t, ds)

	specs, err := ds.GetLabelSpecs()
	require.Nil(t, err)
	test.ElementsMatchSkipTimestampsID(t, expectedSpecs, specs)

	// Should be idempotent
	err = ds.ApplyLabelSpecs(expectedSpecs)
	require.Nil(t, err)
	specs, err = ds.GetLabelSpecs()
	require.Nil(t, err)
	test.ElementsMatchSkipTimestampsID(t, expectedSpecs, specs)
}

func TestLabelIDsByName(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	setupLabelSpecsTest(t, ds)

	labels, err := ds.LabelIDsByName([]string{"foo", "bar", "bing"})
	require.Nil(t, err)
	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
	assert.Equal(t, []uint{1, 2, 3}, labels)
}

func TestSaveLabel(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	label := &fleet.Label{
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes;",
		Platform:    "darwin",
	}
	label, err := db.NewLabel(label)
	require.Nil(t, err)
	label.Name = "changed name"
	label.Description = "changed description"
	_, err = db.SaveLabel(label)
	require.Nil(t, err)
	saved, err := db.Label(label.ID)
	require.Nil(t, err)
	assert.Equal(t, label.Name, saved.Name)
	assert.Equal(t, label.Description, saved.Description)
}

func TestLabelQueriesForCentOSHost(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	host, err := db.EnrollHost("0", "0", nil, 0)
	require.Nil(t, err, "enrollment should succeed")
	host.Platform = "rhel"
	host.OSVersion = "CentOS 6"
	require.NoError(t, db.SaveHost(host))

	label, err := db.NewLabel(&fleet.Label{
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

	baseTime := time.Now().Add(-5 * time.Minute)

	queries, err := db.LabelQueriesForHost(host, baseTime)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, "select 1;", queries[fmt.Sprint(label.ID)])
}
