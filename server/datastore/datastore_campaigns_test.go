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
	user := newUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	mockClock := clock.NewMockClock()

	query := newQuery(t, ds, "test", "select * from time", user.ID, false)

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

	addHostToCampaign(t, ds, campaign.ID, h1.ID)
	checkTargets(t, ds, campaign.ID, []uint{h1.ID}, []uint{})

	addLabelToCampaign(t, ds, campaign.ID, l1.ID)
	checkTargets(t, ds, campaign.ID, []uint{h1.ID}, []uint{l1.ID})

	addLabelToCampaign(t, ds, campaign.ID, l2.ID)
	checkTargets(t, ds, campaign.ID, []uint{h1.ID}, []uint{l1.ID, l2.ID})

	addHostToCampaign(t, ds, campaign.ID, h2.ID)
	addHostToCampaign(t, ds, campaign.ID, h3.ID)

	checkTargets(t, ds, campaign.ID, []uint{h1.ID, h2.ID, h3.ID}, []uint{l1.ID, l2.ID})

}

func testCleanupDistributedQueryCampaigns(t *testing.T, ds kolide.Datastore) {
	user := newUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	mockClock := clock.NewMockClock()

	query := newQuery(t, ds, "test", "select * from time", user.ID, false)

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
