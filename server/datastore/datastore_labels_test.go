package datastore

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/kolide/kolide-ose/server/datastore/inmem"
	"github.com/kolide/kolide-ose/server/datastore/mysql"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLabels(t *testing.T, db kolide.Datastore) {
	hosts := []kolide.Host{}
	var host *kolide.Host
	var err error
	for i := 0; i < 10; i++ {
		host, err = db.EnrollHost(string(i), "foo", "", 10)
		assert.Nil(t, err, "enrollment should succeed")
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
			Name:     "label1",
			Query:    "query1",
			Platform: "darwin",
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

	// We once threw errors when the search query was empty. Verify that we
	// don't error.
	_, err = db.SearchLabels("")
	require.Nil(t, err)

	labels, err := db.SearchLabels("foo")
	assert.Nil(t, err)
	assert.Len(t, labels, 2)

	label, err := db.SearchLabels("foo", l3.ID)
	assert.Nil(t, err)
	assert.Len(t, label, 1)
	assert.Equal(t, "foo", label[0].Name)

	none, err := db.SearchLabels("xxx")
	assert.Nil(t, err)
	assert.Len(t, none, 0)
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
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
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
	if i, ok := db.(*mysql.Datastore); ok {
		err := i.Initialize()
		require.Nil(t, err)
	}
	if i, ok := db.(*inmem.Datastore); ok {
		err := i.Initialize()
		require.Nil(t, err)
	}

	hits, err := db.SearchLabels("Mac OS X")
	require.Nil(t, err)
	assert.Equal(t, 1, len(hits))
	assert.Equal(t, kolide.LabelTypeBuiltIn, hits[0].LabelType)
}

func testListUniqueHostsInLabels(t *testing.T, db kolide.Datastore) {
	h1, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "1",
		UUID:             "1",
		HostName:         "foo.local",
	})
	require.Nil(t, err)

	h2, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "2",
		UUID:             "2",
		HostName:         "bar.local",
	})
	require.Nil(t, err)

	h3, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "3",
		UUID:             "3",
		HostName:         "baz.local",
	})
	require.Nil(t, err)

	h4, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "4",
		UUID:             "4",
		HostName:         "xxx.local",
	})
	require.Nil(t, err)

	h5, err := db.NewHost(&kolide.Host{
		DetailUpdateTime: time.Now(),
		NodeKey:          "5",
		UUID:             "5",
		HostName:         "yyy.local",
	})
	require.Nil(t, err)

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

	for _, h := range []*kolide.Host{h1, h2, h3} {
		err = db.RecordLabelQueryExecutions(h, map[string]bool{l1ID: true}, time.Now())
		assert.Nil(t, err)
	}

	for _, h := range []*kolide.Host{h3, h4, h5} {
		err = db.RecordLabelQueryExecutions(h, map[string]bool{l2ID: true}, time.Now())
		assert.Nil(t, err)
	}

	hosts, err := db.ListUniqueHostsInLabels([]uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Len(t, hosts, 5)
}
