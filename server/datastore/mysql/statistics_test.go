package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatistics(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ShouldSend", testStatisticsShouldSend},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testStatisticsShouldSend(t *testing.T, ds *Datastore) {
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "M",
	})
	require.NoError(t, err)

	// First time running, we send statistics
	stats, shouldSend, err := ds.ShouldSendStatistics(context.Background(), fleet.StatisticsFrequency)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.NotEmpty(t, stats.AnonymousIdentifier)
	assert.Equal(t, stats.NumHostsEnrolled, 1)
	firstIdentifier := stats.AnonymousIdentifier

	err = ds.RecordStatisticsSent(context.Background())
	require.NoError(t, err)

	// If we try right away, it shouldn't ask to send
	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), fleet.StatisticsFrequency)
	require.NoError(t, err)
	assert.False(t, shouldSend)

	time.Sleep(2)

	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   "S",
	})
	require.NoError(t, err)

	// Lower the frequency to trigger an "outdated" sent
	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), time.Millisecond)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, stats.NumHostsEnrolled, 2)
}
