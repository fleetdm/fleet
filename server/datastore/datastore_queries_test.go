package datastore

import (
	"fmt"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/patrickmn/sortutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeleteQuery(t *testing.T, ds kolide.Datastore) {
	query := &kolide.Query{
		Name:     "foo",
		Query:    "bar",
		Interval: 123,
	}
	query, err := ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotEqual(t, query.ID, 0)

	err = ds.DeleteQuery(query)
	assert.Nil(t, err)

	assert.NotEqual(t, query.ID, 0)
	_, err = ds.Query(query.ID)
	assert.NotNil(t, err)
}

func testSaveQuery(t *testing.T, ds kolide.Datastore) {
	query := &kolide.Query{
		Name:  "foo",
		Query: "bar",
	}
	query, err := ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotEqual(t, 0, query.ID)

	query.Query = "baz"
	err = ds.SaveQuery(query)

	assert.Nil(t, err)

	queryVerify, err := ds.Query(query.ID)
	assert.Nil(t, err)
	assert.Equal(t, "baz", queryVerify.Query)
}

func testListQuery(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewQuery(&kolide.Query{
			Name:  fmt.Sprintf("name%02d", i),
			Query: fmt.Sprintf("query%02d", i),
		})
		assert.Nil(t, err)
	}

	opts := kolide.ListOptions{}
	results, err := ds.ListQueries(opts)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(results))
}

func newQuery(t *testing.T, ds kolide.Datastore, name, q string) *kolide.Query {
	query, err := ds.NewQuery(&kolide.Query{
		Name:  name,
		Query: q,
	})
	require.Nil(t, err)

	return query
}

func newCampaign(t *testing.T, ds kolide.Datastore, queryID uint, status kolide.DistributedQueryStatus) *kolide.DistributedQueryCampaign {
	campaign, err := ds.NewDistributedQueryCampaign(&kolide.DistributedQueryCampaign{
		QueryID: queryID,
		Status:  status,
	})
	require.Nil(t, err)

	return campaign
}

func newHost(t *testing.T, ds kolide.Datastore, name, ip, key, uuid string, tim time.Time) *kolide.Host {
	h, err := ds.NewHost(&kolide.Host{
		HostName:         name,
		PrimaryIP:        ip,
		NodeKey:          key,
		UUID:             uuid,
		DetailUpdateTime: tim,
	})

	require.Nil(t, err)
	require.NotZero(t, h.ID)
	require.Nil(t, ds.MarkHostSeen(h, tim))

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

	campaign := newCampaign(t, ds, query.ID, kolide.QueryRunning)

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
