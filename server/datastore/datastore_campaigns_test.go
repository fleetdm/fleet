package datastore

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/patrickmn/sortutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newQuery(t *testing.T, ds kolide.Datastore, name, q string) *kolide.Query {
	query, err := ds.NewQuery(&kolide.Query{
		Name:  name,
		Query: q,
	})
	require.Nil(t, err)

	return query
}

func newCampaign(t *testing.T, ds kolide.Datastore, queryID uint, status kolide.DistributedQueryStatus, now time.Time) *kolide.DistributedQueryCampaign {
	campaign, err := ds.NewDistributedQueryCampaign(&kolide.DistributedQueryCampaign{
		UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
			CreateTimestamp: kolide.CreateTimestamp{
				CreatedAt: now,
			},
		},
		QueryID: queryID,
		Status:  status,
	})
	require.Nil(t, err)

	return campaign
}

func newExecution(t *testing.T, ds kolide.Datastore, campaignID uint, hostID uint) *kolide.DistributedQueryExecution {
	execution, err := ds.NewDistributedQueryExecution(&kolide.DistributedQueryExecution{
		HostID: hostID,
		DistributedQueryCampaignID: campaignID,
	})
	require.Nil(t, err)

	return execution
}

func newHost(t *testing.T, ds kolide.Datastore, name, ip, key, uuid string, now time.Time) *kolide.Host {
	h, err := ds.NewHost(&kolide.Host{
		HostName:         name,
		NodeKey:          key,
		UUID:             uuid,
		DetailUpdateTime: now,
	})

	require.Nil(t, err)
	require.NotZero(t, h.ID)
	require.Nil(t, ds.MarkHostSeen(h, now))

	return h
}

func newLabel(t *testing.T, ds kolide.Datastore, name, query string) *kolide.Label {
	l, err := ds.NewLabel(&kolide.Label{Name: name, Query: query})

	require.Nil(t, err)
	require.NotZero(t, l.ID)

	return l
}

func addHost(t *testing.T, ds kolide.Datastore, campaignID, hostID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetHost,
			TargetID:                   hostID,
			DistributedQueryCampaignID: campaignID,
		})
	require.Nil(t, err)

}

func addLabel(t *testing.T, ds kolide.Datastore, campaignID, labelID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetLabel,
			TargetID:                   labelID,
			DistributedQueryCampaignID: campaignID,
		})
	require.Nil(t, err)
}

func checkTargets(t *testing.T, ds kolide.Datastore, campaignID uint, expectedHostIDs []uint, expectedLabelIDs []uint) {
	hostIDs, labelIDs, err := ds.DistributedQueryCampaignTargetIDs(campaignID)
	require.Nil(t, err)

	sortutil.Asc(expectedHostIDs)
	sortutil.Asc(hostIDs)
	assert.Equal(t, expectedHostIDs, hostIDs)

	sortutil.Asc(expectedLabelIDs)
	sortutil.Asc(labelIDs)
	assert.Equal(t, expectedLabelIDs, labelIDs)
}

func testDistributedQueryCampaign(t *testing.T, ds kolide.Datastore) {
	mockClock := clock.NewMockClock()

	query := newQuery(t, ds, "test", "select * from time")

	campaign := newCampaign(t, ds, query.ID, kolide.QueryRunning, mockClock.Now())

	{
		retrieved, err := ds.DistributedQueryCampaign(campaign.ID)
		require.Nil(t, err)
		assert.Equal(t, campaign.QueryID, retrieved.QueryID)
		assert.Equal(t, campaign.Status, retrieved.Status)
	}

	h1 := newHost(t, ds, "foo.local", "192.168.1.10", "1", "1", mockClock.Now())
	h2 := newHost(t, ds, "bar.local", "192.168.1.11", "2", "2", mockClock.Now().Add(-1*time.Hour))
	h3 := newHost(t, ds, "baz.local", "192.168.1.12", "3", "3", mockClock.Now().Add(-13*time.Minute))

	l1 := newLabel(t, ds, "label foo", "query foo")
	l2 := newLabel(t, ds, "label bar", "query foo")

	checkTargets(t, ds, campaign.ID, []uint{}, []uint{})

	addHost(t, ds, campaign.ID, h1.ID)
	checkTargets(t, ds, campaign.ID, []uint{h1.ID}, []uint{})

	addLabel(t, ds, campaign.ID, l1.ID)
	checkTargets(t, ds, campaign.ID, []uint{h1.ID}, []uint{l1.ID})

	addLabel(t, ds, campaign.ID, l2.ID)
	checkTargets(t, ds, campaign.ID, []uint{h1.ID}, []uint{l1.ID, l2.ID})

	addHost(t, ds, campaign.ID, h2.ID)
	addHost(t, ds, campaign.ID, h3.ID)

	checkTargets(t, ds, campaign.ID, []uint{h1.ID, h2.ID, h3.ID}, []uint{l1.ID, l2.ID})

}

func testCleanupDistributedQueryCampaigns(t *testing.T, ds kolide.Datastore) {
	mockClock := clock.NewMockClock()

	query := newQuery(t, ds, "test", "select * from time")

	c1 := newCampaign(t, ds, query.ID, kolide.QueryWaiting, mockClock.Now())
	c2 := newCampaign(t, ds, query.ID, kolide.QueryRunning, mockClock.Now())

	h1 := newHost(t, ds, "1", "", "1", "1", mockClock.Now())
	h2 := newHost(t, ds, "2", "", "2", "2", mockClock.Now())
	h3 := newHost(t, ds, "3", "", "3", "3", mockClock.Now())

	// Cleanup and verify that nothing changed (because time has not
	// advanced)
	expired, deleted, err := ds.CleanupDistributedQueryCampaigns(mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), expired)
	assert.Equal(t, uint(0), deleted)

	{
		retrieved, err := ds.DistributedQueryCampaign(c1.ID)
		require.Nil(t, err)
		assert.Equal(t, c1.QueryID, retrieved.QueryID)
		assert.Equal(t, c1.Status, retrieved.Status)
	}
	{
		retrieved, err := ds.DistributedQueryCampaign(c2.ID)
		require.Nil(t, err)
		assert.Equal(t, c2.QueryID, retrieved.QueryID)
		assert.Equal(t, c2.Status, retrieved.Status)
	}

	// Add some executions
	newExecution(t, ds, c1.ID, h1.ID)
	newExecution(t, ds, c1.ID, h2.ID)
	newExecution(t, ds, c2.ID, h1.ID)
	newExecution(t, ds, c2.ID, h2.ID)
	newExecution(t, ds, c2.ID, h3.ID)

	mockClock.AddTime(1*time.Minute + 1*time.Second)

	// Cleanup and verify that the campaign was expired and executions
	// deleted appropriately
	expired, deleted, err = ds.CleanupDistributedQueryCampaigns(mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), expired)
	assert.Equal(t, uint(2), deleted)
	{
		// c1 should now be complete
		retrieved, err := ds.DistributedQueryCampaign(c1.ID)
		require.Nil(t, err)
		assert.Equal(t, c1.QueryID, retrieved.QueryID)
		assert.Equal(t, kolide.QueryComplete, retrieved.Status)
	}
	{
		retrieved, err := ds.DistributedQueryCampaign(c2.ID)
		require.Nil(t, err)
		assert.Equal(t, c2.QueryID, retrieved.QueryID)
		assert.Equal(t, c2.Status, retrieved.Status)
	}

	mockClock.AddTime(24*time.Hour + 1*time.Second)

	// Cleanup and verify that the campaign was expired and executions
	// deleted appropriately
	expired, deleted, err = ds.CleanupDistributedQueryCampaigns(mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), expired)
	assert.Equal(t, uint(3), deleted)
	{
		retrieved, err := ds.DistributedQueryCampaign(c1.ID)
		require.Nil(t, err)
		assert.Equal(t, c1.QueryID, retrieved.QueryID)
		assert.Equal(t, kolide.QueryComplete, retrieved.Status)
	}
	{
		// c2 should now be complete
		retrieved, err := ds.DistributedQueryCampaign(c2.ID)
		require.Nil(t, err)
		assert.Equal(t, c2.QueryID, retrieved.QueryID)
		assert.Equal(t, kolide.QueryComplete, retrieved.Status)
	}

}
