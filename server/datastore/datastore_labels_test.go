package datastore

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLabels(t *testing.T, db kolide.Datastore) {
	hosts := []kolide.Host{}
	var host *kolide.Host
	var err error
	for i := 0; i < 10; i++ {
		host, err = db.EnrollHost(string(i), 10)
		require.Nil(t, err, "enrollment should succeed")
		hosts = append(hosts, *host)
	}

	baseTime := time.Now()

	// No queries should be returned before labels or queries added
	queries, err := db.LabelQueriesForHost(host, baseTime)
	assert.Nil(t, err)
	assert.Empty(t, queries)

	// No labels should match
	labels, err := db.ListLabelsForHost(host.ID)
	assert.Nil(t, err)
	assert.Empty(t, labels)

	// No queries should be returned before labels added
	queries, err = db.LabelQueriesForHost(host, baseTime)
	assert.Nil(t, err)
	assert.Empty(t, queries)

	newLabels := []*kolide.LabelSpec{
		// Note these are intentionally out of order
		&kolide.LabelSpec{
			ID:       1,
			Name:     "label3",
			Query:    "query3",
			Platform: "darwin",
		},
		&kolide.LabelSpec{
			ID:    2,
			Name:  "label1",
			Query: "query1",
		},
		&kolide.LabelSpec{
			ID:       3,
			Name:     "label2",
			Query:    "query2",
			Platform: "darwin",
		},
		&kolide.LabelSpec{
			ID:       4,
			Name:     "label4",
			Query:    "query4",
			Platform: "darwin",
		},
	}
	err = db.ApplyLabelSpecs(newLabels)
	require.Nil(t, err)

	expectQueries := map[string]string{
		"1": "query3",
		"2": "query1",
		"3": "query2",
		"4": "query4",
	}

	host.Platform = "darwin"

	// Now queries should be returned
	queries, err = db.LabelQueriesForHost(host, baseTime)
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// No labels should match with no results yet
	labels, err = db.ListLabelsForHost(host.ID)
	assert.Nil(t, err)
	assert.Empty(t, labels)

	// Record a query execution
	err = db.RecordLabelQueryExecutions(host, map[uint]bool{1: true}, baseTime)
	assert.Nil(t, err)

	// Use a 10 minute interval, so the query we just added should show up
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.Nil(t, err)
	delete(expectQueries, "1")
	assert.Equal(t, expectQueries, queries)

	// Record an old query execution -- Shouldn't change the return
	err = db.RecordLabelQueryExecutions(host, map[uint]bool{2: true}, baseTime.Add(-1*time.Hour))
	assert.Nil(t, err)
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// Record a newer execution for that query and another
	err = db.RecordLabelQueryExecutions(host, map[uint]bool{2: false, 3: true}, baseTime)
	assert.Nil(t, err)

	// Now these should no longer show up in the necessary to run queries
	delete(expectQueries, "2")
	delete(expectQueries, "3")
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// Now the two matching labels should be returned
	labels, err = db.ListLabelsForHost(host.ID)
	assert.Nil(t, err)
	if assert.Len(t, labels, 2) {
		labelNames := []string{labels[0].Name, labels[1].Name}
		sort.Strings(labelNames)
		assert.Equal(t, "label2", labelNames[0])
		assert.Equal(t, "label3", labelNames[1])
	}

	// A host that hasn't executed any label queries should still be asked
	// to execute those queries
	hosts[0].Platform = "darwin"
	queries, err = db.LabelQueriesForHost(&hosts[0], time.Now())
	assert.Nil(t, err)
	assert.Len(t, queries, 4)

	// There should still be no labels returned for a host that never
	// executed any label queries
	labels, err = db.ListLabelsForHost(hosts[0].ID)
	assert.Nil(t, err)
	assert.Empty(t, labels)
}

func testManagingLabelsOnPacks(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.PackSpec{
		ID:   1,
		Name: "pack1",
	}
	err := ds.ApplyPackSpecs([]*kolide.PackSpec{pack})
	require.Nil(t, err)

	labels, err := ds.ListLabelsForPack(pack.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 0)

	mysqlLabel := &kolide.LabelSpec{
		ID:    1,
		Name:  "MySQL Monitoring",
		Query: "select pid from processes where name = 'mysqld';",
	}
	err = ds.ApplyLabelSpecs([]*kolide.LabelSpec{mysqlLabel})
	require.Nil(t, err)

	pack.Targets = kolide.PackSpecTargets{
		Labels: []string{
			mysqlLabel.Name,
		},
	}
	err = ds.ApplyPackSpecs([]*kolide.PackSpec{pack})
	require.Nil(t, err)

	labels, err = ds.ListLabelsForPack(pack.ID)
	require.Nil(t, err)
	if assert.Len(t, labels, 1) {
		assert.Equal(t, "MySQL Monitoring", labels[0].Name)
	}

	osqueryLabel := &kolide.LabelSpec{
		ID:    2,
		Name:  "Osquery Monitoring",
		Query: "select pid from processes where name = 'osqueryd';",
	}
	err = ds.ApplyLabelSpecs([]*kolide.LabelSpec{mysqlLabel, osqueryLabel})
	require.Nil(t, err)

	pack.Targets = kolide.PackSpecTargets{
		Labels: []string{
			mysqlLabel.Name,
			osqueryLabel.Name,
		},
	}
	err = ds.ApplyPackSpecs([]*kolide.PackSpec{pack})
	require.Nil(t, err)

	labels, err = ds.ListLabelsForPack(pack.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 2)
}

func testSearchLabels(t *testing.T, db kolide.Datastore) {
	specs := []*kolide.LabelSpec{
		&kolide.LabelSpec{
			ID:   1,
			Name: "foo",
		},
		&kolide.LabelSpec{
			ID:   2,
			Name: "bar",
		},
		&kolide.LabelSpec{
			ID:   3,
			Name: "foo-bar",
		},
		&kolide.LabelSpec{
			ID:        4,
			Name:      "All Hosts",
			LabelType: kolide.LabelTypeBuiltIn,
		},
	}
	err := db.ApplyLabelSpecs(specs)
	require.Nil(t, err)

	all, err := db.Label(specs[3].ID)
	require.Nil(t, err)
	l3, err := db.Label(specs[2].ID)
	require.Nil(t, err)

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	labels, err := db.SearchLabels("")
	require.Nil(t, err)
	assert.Contains(t, labels, *all)

	labels, err = db.SearchLabels("foo")
	require.Nil(t, err)
	assert.Len(t, labels, 3)
	assert.Contains(t, labels, *all)

	labels, err = db.SearchLabels("foo", all.ID, l3.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "foo", labels[0].Name)

	labels, err = db.SearchLabels("xxx")
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Contains(t, labels, *all)
}

func testSearchLabelsLimit(t *testing.T, db kolide.Datastore) {
	if db.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	all := &kolide.LabelSpec{
		Name:      "All Hosts",
		LabelType: kolide.LabelTypeBuiltIn,
	}
	err := db.ApplyLabelSpecs([]*kolide.LabelSpec{all})
	require.Nil(t, err)

	for i := 0; i < 15; i++ {
		l := &kolide.LabelSpec{
			Name: fmt.Sprintf("foo%d", i),
		}
		err := db.ApplyLabelSpecs([]*kolide.LabelSpec{l})
		require.Nil(t, err)
	}

	labels, err := db.SearchLabels("foo")
	require.Nil(t, err)
	assert.Len(t, labels, 11)
}

func testListHostsInLabel(t *testing.T, db kolide.Datastore) {
	h1, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		OsqueryHostID:    "1",
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		OsqueryHostID:    "2",
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		OsqueryHostID:    "3",
		NodeKey:          "3",
		UUID:             "3",
		HostName:         "baz.local",
	})
	require.Nil(t, err)

	l1 := &kolide.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	err = db.ApplyLabelSpecs([]*kolide.LabelSpec{l1})
	require.Nil(t, err)

	{

		hosts, err := db.ListHostsInLabel(l1.ID)
		require.Nil(t, err)
		assert.Len(t, hosts, 0)
	}

	for _, h := range []*kolide.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	{
		hosts, err := db.ListHostsInLabel(l1.ID)
		require.Nil(t, err)
		assert.Len(t, hosts, 3)
	}
}

func testBuiltInLabels(t *testing.T, db kolide.Datastore) {
	require.Nil(t, db.MigrateData())

	hits, err := db.SearchLabels("macOS")
	require.Nil(t, err)
	// Should get Mac OS X and All Hosts
	assert.Equal(t, 2, len(hits))
	assert.Equal(t, kolide.LabelTypeBuiltIn, hits[0].LabelType)
	assert.Equal(t, kolide.LabelTypeBuiltIn, hits[1].LabelType)
}

func testListUniqueHostsInLabels(t *testing.T, db kolide.Datastore) {
	hosts := []*kolide.Host{}
	for i := 0; i < 4; i++ {
		h, err := db.NewHost(&kolide.Host{
			DetailUpdateTime: time.Now(),
			SeenTime:         time.Now(),
			OsqueryHostID:    strconv.Itoa(i),
			NodeKey:          strconv.Itoa(i),
			UUID:             strconv.Itoa(i),
			HostName:         fmt.Sprintf("host_%d", i),
		})
		require.Nil(t, err)
		require.NotNil(t, h)
		hosts = append(hosts, h)
	}

	l1 := kolide.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query1",
	}
	l2 := kolide.LabelSpec{
		ID:    2,
		Name:  "label bar",
		Query: "query2",
	}
	err := db.ApplyLabelSpecs([]*kolide.LabelSpec{&l1, &l2})
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

	uniqueHosts, err := db.ListUniqueHostsInLabels([]uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, len(hosts), len(uniqueHosts))

	labels, err := db.ListLabels(kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, labels, 2)

}

func testChangeLabelDetails(t *testing.T, db kolide.Datastore) {
	if db.Name() == "inmem" {
		t.Skip("inmem is being deprecated")
	}

	label := kolide.LabelSpec{
		ID:          1,
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes",
		Platform:    "darwin",
	}
	err := db.ApplyLabelSpecs([]*kolide.LabelSpec{&label})
	require.Nil(t, err)

	label.Description = "changed description"
	err = db.ApplyLabelSpecs([]*kolide.LabelSpec{&label})
	require.Nil(t, err)

	saved, err := db.Label(label.ID)
	require.Nil(t, err)
	assert.Equal(t, label.Name, saved.Name)
}

func setupLabelSpecsTest(t *testing.T, ds kolide.Datastore) []*kolide.LabelSpec {
	expectedSpecs := []*kolide.LabelSpec{
		&kolide.LabelSpec{
			Name:        "foo",
			Query:       "select * from foo",
			Description: "foo description",
			Platform:    "darwin",
		},
		&kolide.LabelSpec{
			Name:  "bar",
			Query: "select * from bar",
		},
		&kolide.LabelSpec{
			Name:  "bing",
			Query: "select * from bing",
		},
		&kolide.LabelSpec{
			Name:      "All Hosts",
			Query:     "SELECT 1",
			LabelType: kolide.LabelTypeBuiltIn,
		},
	}
	err := ds.ApplyLabelSpecs(expectedSpecs)
	require.Nil(t, err)

	return expectedSpecs
}

func testGetLabelSpec(t *testing.T, ds kolide.Datastore) {
	expectedSpecs := setupLabelSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetLabelSpec(s.Name)
		require.Nil(t, err)
		assert.Equal(t, s, spec)
	}
}

func testApplyLabelSpecsRoundtrip(t *testing.T, ds kolide.Datastore) {
	expectedSpecs := setupLabelSpecsTest(t, ds)

	specs, err := ds.GetLabelSpecs()
	require.Nil(t, err)
	assert.Equal(t, expectedSpecs, specs)

	// Should be idempotent
	err = ds.ApplyLabelSpecs(expectedSpecs)
	require.Nil(t, err)
	specs, err = ds.GetLabelSpecs()
	require.Nil(t, err)
	assert.Equal(t, expectedSpecs, specs)
}
