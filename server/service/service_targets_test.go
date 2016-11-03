package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestSearchTargets(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)

	svc, err := newTestService(ds)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName:  "foo.local",
		PrimaryIP: "192.168.1.10",
	})
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})
	require.Nil(t, err)

	results, err := svc.SearchTargets(ctx, "foo", nil, nil)
	require.Nil(t, err)

	require.Len(t, results.Hosts, 1)
	assert.Equal(t, h1.HostName, results.Hosts[0].HostName)

	require.Len(t, results.Labels, 1)
	assert.Equal(t, l1.Name, results.Labels[0].Name)
}

func TestCountHostsInTargets(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)

	svc, err := newTestService(ds)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName:  "foo.local",
		PrimaryIP: "192.168.1.10",
		NodeKey:   "1",
		UUID:      "1",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		HostName:  "bar.local",
		PrimaryIP: "192.168.1.11",
		NodeKey:   "2",
		UUID:      "2",
	})
	require.Nil(t, err)

	h3, err := ds.NewHost(&kolide.Host{
		HostName:  "baz.local",
		PrimaryIP: "192.168.1.12",
		NodeKey:   "3",
		UUID:      "3",
	})
	require.Nil(t, err)

	h4, err := ds.NewHost(&kolide.Host{
		HostName:  "xxx.local",
		PrimaryIP: "192.168.1.13",
		NodeKey:   "4",
		UUID:      "4",
	})
	require.Nil(t, err)

	h5, err := ds.NewHost(&kolide.Host{
		HostName:  "yyy.local",
		PrimaryIP: "192.168.1.14",
		NodeKey:   "5",
		UUID:      "5",
	})
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})
	require.Nil(t, err)
	require.NotZero(t, l1.ID)
	l1ID := fmt.Sprintf("%d", l1.ID)

	l2, err := ds.NewLabel(&kolide.Label{
		Name:  "label bar",
		Query: "query foo",
	})
	require.Nil(t, err)
	require.NotZero(t, l2.ID)
	l2ID := fmt.Sprintf("%d", l2.ID)

	for _, h := range []*kolide.Host{h1, h2, h3} {
		err = ds.RecordLabelQueryExecutions(h, map[string]bool{l1ID: true}, time.Now())
		assert.Nil(t, err)
	}

	for _, h := range []*kolide.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(h, map[string]bool{l2ID: true}, time.Now())
		assert.Nil(t, err)
	}

	count, err := svc.CountHostsInTargets(ctx, nil, []uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, uint(5), count)

	count, err = svc.CountHostsInTargets(ctx, []uint{h1.ID, h2.ID}, []uint{l1.ID, l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, uint(5), count)

	count, err = svc.CountHostsInTargets(ctx, []uint{h1.ID, h2.ID}, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint(2), count)

	count, err = svc.CountHostsInTargets(ctx, []uint{h1.ID}, []uint{l2.ID})
	assert.Nil(t, err)
	assert.Equal(t, uint(4), count)

	count, err = svc.CountHostsInTargets(ctx, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, uint(0), count)
}

func TestSearchWithOmit(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)

	svc, err := newTestService(ds)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName:  "foo.local",
		PrimaryIP: "192.168.1.10",
		NodeKey:   "1",
		UUID:      "1",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		HostName:  "foobar.local",
		PrimaryIP: "192.168.1.11",
		NodeKey:   "2",
		UUID:      "2",
	})
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})

	{
		results, err := svc.SearchTargets(ctx, "foo", nil, nil)
		require.Nil(t, err)

		require.Len(t, results.Hosts, 2)

		require.Len(t, results.Labels, 1)
		assert.Equal(t, l1.Name, results.Labels[0].Name)
	}

	{
		results, err := svc.SearchTargets(ctx, "foo", []uint{h2.ID}, nil)
		require.Nil(t, err)

		require.Len(t, results.Hosts, 1)
		assert.Equal(t, h1.HostName, results.Hosts[0].HostName)

		require.Len(t, results.Labels, 1)
		assert.Equal(t, l1.Name, results.Labels[0].Name)
	}
}

func TestSearchHostsInLabels(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)

	svc, err := newTestService(ds)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName:  "foo.local",
		PrimaryIP: "192.168.1.10",
		NodeKey:   "1",
		UUID:      "1",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		HostName:  "bar.local",
		PrimaryIP: "192.168.1.11",
		NodeKey:   "2",
		UUID:      "2",
	})
	require.Nil(t, err)

	h3, err := ds.NewHost(&kolide.Host{
		HostName:  "baz.local",
		PrimaryIP: "192.168.1.12",
		NodeKey:   "3",
		UUID:      "3",
	})
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})
	require.Nil(t, err)
	require.NotZero(t, l1.ID)
	l1ID := fmt.Sprintf("%d", l1.ID)

	for _, h := range []*kolide.Host{h1, h2, h3} {
		err = ds.RecordLabelQueryExecutions(h, map[string]bool{l1ID: true}, time.Now())
		assert.Nil(t, err)
	}

	results, err := svc.SearchTargets(ctx, "baz", nil, nil)
	require.Nil(t, err)

	require.Len(t, results.Hosts, 1)
	assert.Equal(t, h3.HostName, results.Hosts[0].HostName)
}

func TestSearchResultsLimit(t *testing.T) {
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)

	svc, err := newTestService(ds)
	require.Nil(t, err)

	ctx := context.Background()

	for i := 0; i < 15; i++ {
		_, err := ds.NewHost(&kolide.Host{
			HostName:  fmt.Sprintf("foo.%d.local", i),
			PrimaryIP: fmt.Sprintf("192.168.1.%d", i+1),
			NodeKey:   fmt.Sprintf("%d", i+1),
			UUID:      fmt.Sprintf("%d", i+1),
		})
		require.Nil(t, err)
	}
	targets, err := svc.SearchTargets(ctx, "foo", nil, nil)
	require.Nil(t, err)
	assert.Len(t, targets.Hosts, 10)
}
