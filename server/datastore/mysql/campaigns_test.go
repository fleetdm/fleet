package mysql

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkTargets(t *testing.T, ds fleet.Datastore, campaignID uint, expectedTargets fleet.HostTargets) {
	targets, err := ds.DistributedQueryCampaignTargetIDs(campaignID)
	require.Nil(t, err)
	assert.ElementsMatch(t, expectedTargets.HostIDs, targets.HostIDs)
	assert.ElementsMatch(t, expectedTargets.LabelIDs, targets.LabelIDs)
	assert.ElementsMatch(t, expectedTargets.TeamIDs, targets.TeamIDs)
}

func TestDistributedQueryCampaign(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	mockClock := clock.NewMockClock()

	query := test.NewQuery(t, ds, "test", "select * from time", user.ID, false)

	campaign := test.NewCampaign(t, ds, query.ID, fleet.QueryRunning, mockClock.Now())

	{
		retrieved, err := ds.DistributedQueryCampaign(campaign.ID)
		require.Nil(t, err)
		assert.Equal(t, campaign.QueryID, retrieved.QueryID)
		assert.Equal(t, campaign.Status, retrieved.Status)
	}

	h1 := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", mockClock.Now())
	h2 := test.NewHost(t, ds, "bar.local", "192.168.1.11", "2", "2", mockClock.Now().Add(-1*time.Hour))
	h3 := test.NewHost(t, ds, "baz.local", "192.168.1.12", "3", "3", mockClock.Now().Add(-13*time.Minute))

	l1 := fleet.LabelSpec{
		ID:    1,
		Name:  "label foo",
		Query: "query foo",
	}
	l2 := fleet.LabelSpec{
		ID:    2,
		Name:  "label bar",
		Query: "query bar",
	}
	err := ds.ApplyLabelSpecs([]*fleet.LabelSpec{&l1, &l2})
	require.Nil(t, err)

	checkTargets(t, ds, campaign.ID, fleet.HostTargets{})

	test.AddHostToCampaign(t, ds, campaign.ID, h1.ID)
	checkTargets(t, ds, campaign.ID, fleet.HostTargets{HostIDs: []uint{h1.ID}})

	test.AddLabelToCampaign(t, ds, campaign.ID, l1.ID)
	checkTargets(t, ds, campaign.ID, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l1.ID}})

	test.AddLabelToCampaign(t, ds, campaign.ID, l2.ID)
	checkTargets(t, ds, campaign.ID, fleet.HostTargets{HostIDs: []uint{h1.ID}, LabelIDs: []uint{l1.ID, l2.ID}})

	test.AddHostToCampaign(t, ds, campaign.ID, h2.ID)
	test.AddHostToCampaign(t, ds, campaign.ID, h3.ID)

	checkTargets(t, ds, campaign.ID, fleet.HostTargets{HostIDs: []uint{h1.ID, h2.ID, h3.ID}, LabelIDs: []uint{l1.ID, l2.ID}})

}

func TestCleanupDistributedQueryCampaigns(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	mockClock := clock.NewMockClock()

	query := test.NewQuery(t, ds, "test", "select * from time", user.ID, false)

	c1 := test.NewCampaign(t, ds, query.ID, fleet.QueryWaiting, mockClock.Now())
	c2 := test.NewCampaign(t, ds, query.ID, fleet.QueryRunning, mockClock.Now())

	// Cleanup and verify that nothing changed (because time has not
	// advanced)
	expired, err := ds.CleanupDistributedQueryCampaigns(mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(0), expired)

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

	mockClock.AddTime(1*time.Minute + 1*time.Second)

	// Cleanup and verify that the campaign was expired and executions
	// deleted appropriately
	expired, err = ds.CleanupDistributedQueryCampaigns(mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), expired)
	{
		// c1 should now be complete
		retrieved, err := ds.DistributedQueryCampaign(c1.ID)
		require.Nil(t, err)
		assert.Equal(t, c1.QueryID, retrieved.QueryID)
		assert.Equal(t, fleet.QueryComplete, retrieved.Status)
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
	expired, err = ds.CleanupDistributedQueryCampaigns(mockClock.Now())
	require.Nil(t, err)
	assert.Equal(t, uint(1), expired)
	{
		retrieved, err := ds.DistributedQueryCampaign(c1.ID)
		require.Nil(t, err)
		assert.Equal(t, c1.QueryID, retrieved.QueryID)
		assert.Equal(t, fleet.QueryComplete, retrieved.Status)
	}
	{
		// c2 should now be complete
		retrieved, err := ds.DistributedQueryCampaign(c2.ID)
		require.Nil(t, err)
		assert.Equal(t, c2.QueryID, retrieved.QueryID)
		assert.Equal(t, fleet.QueryComplete, retrieved.Status)
	}

}
