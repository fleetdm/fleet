package datastore

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
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

	newLabels := []kolide.Label{
		// Note these are intentionally out of order
		kolide.Label{
			Name:     "label3",
			Query:    "query3",
			Platform: "darwin",
		},
		kolide.Label{
			Name:  "label1",
			Query: "query1",
		},
		kolide.Label{
			Name:     "label2",
			Query:    "query2",
			Platform: "darwin",
		},
		kolide.Label{
			Name:     "label4",
			Query:    "query4",
			Platform: "darwin",
		},
	}

	for _, label := range newLabels {
		var newLabel *kolide.Label
		newLabel, err = db.NewLabel(&label)
		assert.Nil(t, err)
		assert.NotZero(t, newLabel.ID)
	}

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
	err = db.RecordLabelQueryExecutions(host, map[string]bool{"1": true}, baseTime)
	assert.Nil(t, err)

	// Use a 10 minute interval, so the query we just added should show up
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.Nil(t, err)
	delete(expectQueries, "1")
	assert.Equal(t, expectQueries, queries)

	// Record an old query execution -- Shouldn't change the return
	err = db.RecordLabelQueryExecutions(host, map[string]bool{"2": true}, baseTime.Add(-1*time.Hour))
	assert.Nil(t, err)
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.Nil(t, err)
	assert.Equal(t, expectQueries, queries)

	// Record a newer execution for that query and another
	err = db.RecordLabelQueryExecutions(host, map[string]bool{"2": false, "3": true}, baseTime)
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
	monitoringPack := &kolide.Pack{
		Name: "monitoring",
	}
	_, err := ds.NewPack(monitoringPack)
	require.Nil(t, err)

	mysqlLabel := &kolide.Label{
		Name:  "MySQL Monitoring",
		Query: "select pid from processes where name = 'mysqld';",
	}
	mysqlLabel, err = ds.NewLabel(mysqlLabel)
	require.Nil(t, err)

	err = ds.AddLabelToPack(mysqlLabel.ID, monitoringPack.ID)
	require.Nil(t, err)

	labels, err := ds.ListLabelsForPack(monitoringPack.ID)
	require.Nil(t, err)
	if assert.Len(t, labels, 1) {
		assert.Equal(t, "MySQL Monitoring", labels[0].Name)
	}

	osqueryLabel := &kolide.Label{
		Name:  "Osquery Monitoring",
		Query: "select pid from processes where name = 'osqueryd';",
	}
	osqueryLabel, err = ds.NewLabel(osqueryLabel)
	require.Nil(t, err)

	err = ds.AddLabelToPack(osqueryLabel.ID, monitoringPack.ID)
	require.Nil(t, err)

	labels, err = ds.ListLabelsForPack(monitoringPack.ID)
	require.Nil(t, err)
	assert.Len(t, labels, 2)
}

func testSearchLabels(t *testing.T, db kolide.Datastore) {
	_, err := db.NewLabel(&kolide.Label{
		Name: "foo",
	})
	require.Nil(t, err)

	_, err = db.NewLabel(&kolide.Label{
		Name: "bar",
	})
	require.Nil(t, err)

	l3, err := db.NewLabel(&kolide.Label{
		Name: "foo-bar",
	})
	require.Nil(t, err)

	all, err := db.NewLabel(&kolide.Label{
		Name:      "All Hosts",
		LabelType: kolide.LabelTypeBuiltIn,
	})
	require.Nil(t, err)
	all, err = db.Label(all.ID)
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
	for i := 0; i < 15; i++ {
		_, err := db.NewLabel(&kolide.Label{
			Name: fmt.Sprintf("foo-%d", i),
		})
		require.Nil(t, err)
	}

	labels, err := db.SearchLabels("foo")
	require.Nil(t, err)
	assert.Len(t, labels, 10)
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

	l1, err := db.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query1",
	})
	require.Nil(t, err)
	require.NotZero(t, l1.ID)
	l1ID := fmt.Sprintf("%d", l1.ID)

	{

		hosts, err := db.ListHostsInLabel(l1.ID)
		require.Nil(t, err)
		assert.Len(t, hosts, 0)
	}

	for _, h := range []*kolide.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(h, map[string]bool{l1ID: true}, time.Now())
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

	hits, err := db.SearchLabels("Mac OS X")
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

	l1, err := db.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query1",
	})
	require.Nil(t, err)
	require.NotZero(t, l1.ID)
	l1ID := fmt.Sprintf("%d", l1.ID)

	l2, err := db.NewLabel(&kolide.Label{
		Name:  "label bar",
		Query: "query2",
	})
	require.Nil(t, err)
	require.NotZero(t, l2.ID)
	l2ID := fmt.Sprintf("%d", l2.ID)

	for i := 0; i < 3; i++ {
		err = db.RecordLabelQueryExecutions(hosts[i], map[string]bool{l1ID: true}, time.Now())
		assert.Nil(t, err)
	}
	// host 2 executes twice
	for i := 2; i < len(hosts); i++ {
		err = db.RecordLabelQueryExecutions(hosts[i], map[string]bool{l2ID: true}, time.Now())
		assert.Nil(t, err)
	}

	uniqueHosts, err := db.ListUniqueHostsInLabels([]uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, len(hosts), len(uniqueHosts))
}
