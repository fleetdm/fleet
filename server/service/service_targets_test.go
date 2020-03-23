package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/datastore/inmem"
	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchTargets(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName: "foo.local",
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

func TestSearchWithOmit(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName: "foo.local",
		NodeKey:  "1",
		UUID:     "1",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		HostName: "foobar.local",
		NodeKey:  "2",
		UUID:     "2",
	})
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})
	require.Nil(t, err)

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
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ctx := context.Background()

	h1, err := ds.NewHost(&kolide.Host{
		HostName: "foo.local",
		NodeKey:  "1",
		UUID:     "1",
	})
	require.Nil(t, err)

	h2, err := ds.NewHost(&kolide.Host{
		HostName: "bar.local",
		NodeKey:  "2",
		UUID:     "2",
	})
	require.Nil(t, err)

	h3, err := ds.NewHost(&kolide.Host{
		HostName: "baz.local",
		NodeKey:  "3",
		UUID:     "3",
	})
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})
	require.Nil(t, err)
	require.NotZero(t, l1.ID)

	for _, h := range []*kolide.Host{h1, h2, h3} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, time.Now())
		assert.Nil(t, err)
	}

	results, err := svc.SearchTargets(ctx, "baz", nil, nil)
	require.Nil(t, err)

	require.Len(t, results.Hosts, 1)
	assert.Equal(t, h3.HostName, results.Hosts[0].HostName)
}

func TestSearchResultsLimit(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	ctx := context.Background()

	for i := 0; i < 15; i++ {
		_, err := ds.NewHost(&kolide.Host{
			HostName: fmt.Sprintf("foo.%d.local", i),
			NodeKey:  fmt.Sprintf("%d", i+1),
			UUID:     fmt.Sprintf("%d", i+1),
		})
		require.Nil(t, err)
	}
	targets, err := svc.SearchTargets(ctx, "foo", nil, nil)
	require.Nil(t, err)
	assert.Len(t, targets.Hosts, 10)
}
