package datastore

import (
	"testing"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/require"
)

func newQuery(t *testing.T, ds kolide.Datastore, name, q string, authorID uint, saved bool) *kolide.Query {
	query, err := ds.NewQuery(&kolide.Query{
		Name:     name,
		Query:    q,
		AuthorID: authorID,
		Saved:    saved,
	})
	require.Nil(t, err)

	// Loading gives us the timestamps
	query, err = ds.Query(query.ID)
	require.Nil(t, err)

	return query
}

func newPack(t *testing.T, ds kolide.Datastore, name string) *kolide.Pack {
	pack, err := ds.NewPack(&kolide.Pack{
		Name: name,
	})
	require.Nil(t, err)

	// Loading gives us the timestamps
	pack, err = ds.Pack(pack.ID)
	require.Nil(t, err)

	return pack
}

func addQueryToPack(t *testing.T, ds kolide.Datastore, queryID, packID uint) {
	err := ds.AddQueryToPack(queryID, packID)
	require.Nil(t, err)
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

	// Loading gives us the timestamps
	campaign, err = ds.DistributedQueryCampaign(campaign.ID)
	require.Nil(t, err)

	return campaign
}

func addHostToCampaign(t *testing.T, ds kolide.Datastore, campaignID, hostID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetHost,
			TargetID:                   hostID,
			DistributedQueryCampaignID: campaignID,
		})
	require.Nil(t, err)
}

func addLabelToCampaign(t *testing.T, ds kolide.Datastore, campaignID, labelID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		&kolide.DistributedQueryCampaignTarget{
			Type:                       kolide.TargetLabel,
			TargetID:                   labelID,
			DistributedQueryCampaignID: campaignID,
		})
	require.Nil(t, err)
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
	osqueryHostID, _ := kolide.RandomText(10)
	h, err := ds.NewHost(&kolide.Host{
		HostName:         name,
		NodeKey:          key,
		UUID:             uuid,
		DetailUpdateTime: now,
		OsqueryHostID:    osqueryHostID,
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

func newUser(t *testing.T, ds kolide.Datastore, name, username, email string, admin bool) *kolide.User {
	u, err := ds.NewUser(&kolide.User{
		Password: []byte("garbage"),
		Salt:     "garbage",
		Name:     name,
		Username: username,
		Email:    email,
		Admin:    admin,
	})

	require.Nil(t, err)
	require.NotZero(t, u.ID)

	return u
}
