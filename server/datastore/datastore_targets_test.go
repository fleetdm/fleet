package datastore

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCountHostsInTargets(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	mockClock := clock.NewMockClock()

	h1, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "1",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		HostName:         "foo.local",
		NodeKey:          "1",
		UUID:             "1",
	})
	require.Nil(t, err)
	require.Nil(t, ds.MarkHostSeen(h1, mockClock.Now()))

	h2, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "2",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		HostName:         "bar.local",
		NodeKey:          "2",
		UUID:             "2",
	})
	require.Nil(t, err)
	// make this host "offline"
	require.Nil(t, ds.MarkHostSeen(h2, mockClock.Now().Add(-1*time.Hour)))

	h3, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "3",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		HostName:         "baz.local",
		NodeKey:          "3",
		UUID:             "3",
	})
	require.Nil(t, err)
	require.Nil(t, ds.MarkHostSeen(h3, mockClock.Now().Add(-5*time.Second)))

	h4, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "4",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		HostName:         "xxx.local",
		NodeKey:          "4",
		UUID:             "4",
	})
	require.Nil(t, err)
	require.Nil(t, ds.MarkHostSeen(h4, mockClock.Now()))

	h5, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "5",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		HostName:         "yyy.local",
		NodeKey:          "5",
		UUID:             "5",
	})
	require.Nil(t, err)
	require.Nil(t, ds.MarkHostSeen(h5, mockClock.Now()))

	h6, err := ds.NewHost(&kolide.Host{
		OsqueryHostID:    "6",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
		HostName:         "zzz.local",
		NodeKey:          "6",
		UUID:             "6",
	})
	require.Nil(t, err)
	const thirtyDaysAndAMinuteAgo = -1 * (30*24*60 + 1)
	require.Nil(t, ds.MarkHostSeen(h6, mockClock.Now().Add(thirtyDaysAndAMinuteAgo*time.Minute)))

	l1, err := ds.NewLabel(&kolide.Label{
		Name:  "label foo",
		Query: "query foo",
	})
	require.Nil(t, err)
	require.NotZero(t, l1.ID)

	l2, err := ds.NewLabel(&kolide.Label{
		Name:  "label bar",
		Query: "query foo",
	})
	require.Nil(t, err)
	require.NotZero(t, l2.ID)

	for _, h := range []*kolide.Host{h1, h2, h3, h6} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l1.ID: true}, mockClock.Now())
		assert.Nil(t, err)
	}

	for _, h := range []*kolide.Host{h3, h4, h5} {
		err = ds.RecordLabelQueryExecutions(h, map[uint]bool{l2.ID: true}, mockClock.Now())
		assert.Nil(t, err)
	}

	metrics, err := ds.CountHostsInTargets(nil, []uint{l1.ID, l2.ID}, mockClock.Now(), 30*time.Minute)
	require.Nil(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(4), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets([]uint{h1.ID, h2.ID}, []uint{l1.ID, l2.ID}, mockClock.Now(), 30*time.Minute)
	require.Nil(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(4), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets([]uint{h1.ID, h2.ID}, nil, mockClock.Now(), 30*time.Minute)
	require.Nil(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, uint(2), metrics.TotalHosts)
	assert.Equal(t, uint(1), metrics.OnlineHosts)
	assert.Equal(t, uint(1), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets([]uint{h1.ID}, []uint{l2.ID}, mockClock.Now(), 30*time.Minute)
	require.Nil(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, uint(4), metrics.TotalHosts)
	assert.Equal(t, uint(4), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	metrics, err = ds.CountHostsInTargets(nil, nil, mockClock.Now(), 30*time.Minute)
	require.Nil(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, uint(0), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(0), metrics.OfflineHosts)
	assert.Equal(t, uint(0), metrics.MissingInActionHosts)

	// Advance clock so all hosts are offline
	mockClock.AddTime(1 * time.Hour)
	metrics, err = ds.CountHostsInTargets(nil, []uint{l1.ID, l2.ID}, mockClock.Now(), 30*time.Minute)
	require.Nil(t, err)
	require.NotNil(t, metrics)
	assert.Equal(t, uint(6), metrics.TotalHosts)
	assert.Equal(t, uint(0), metrics.OnlineHosts)
	assert.Equal(t, uint(5), metrics.OfflineHosts)
	assert.Equal(t, uint(1), metrics.MissingInActionHosts)

}
