package test

import (
	"github.com/fleetdm/fleet/v4/server"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func NewQuery(t *testing.T, ds fleet.Datastore, name, q string, authorID uint, saved bool) *fleet.Query {
	authorPtr := &authorID
	if authorID == 0 {
		authorPtr = nil
	}
	query, err := ds.NewQuery(&fleet.Query{
		Name:     name,
		Query:    q,
		AuthorID: authorPtr,
		Saved:    saved,
	})
	require.Nil(t, err)

	// Loading gives us the timestamps
	query, err = ds.Query(query.ID)
	require.Nil(t, err)

	return query
}

func NewPack(t *testing.T, ds fleet.Datastore, name string) *fleet.Pack {
	err := ds.ApplyPackSpecs([]*fleet.PackSpec{&fleet.PackSpec{Name: name}})
	require.Nil(t, err)

	// Loading gives us the timestamps
	pack, ok, err := ds.PackByName(name)
	require.True(t, ok)
	require.Nil(t, err)

	return pack
}

func NewCampaign(t *testing.T, ds fleet.Datastore, queryID uint, status fleet.DistributedQueryStatus, now time.Time) *fleet.DistributedQueryCampaign {
	campaign, err := ds.NewDistributedQueryCampaign(&fleet.DistributedQueryCampaign{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
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

func AddHostToCampaign(t *testing.T, ds fleet.Datastore, campaignID, hostID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetHost,
			TargetID:                   hostID,
			DistributedQueryCampaignID: campaignID,
		})
	require.Nil(t, err)
}

func AddLabelToCampaign(t *testing.T, ds fleet.Datastore, campaignID, labelID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetLabel,
			TargetID:                   labelID,
			DistributedQueryCampaignID: campaignID,
		})
	require.Nil(t, err)
}

func AddAllHostsLabel(t *testing.T, ds fleet.Datastore) {
	_, err := ds.NewLabel(
		&fleet.Label{
			Name:                "All Hosts",
			Query:               "select 1",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
	)
	require.Nil(t, err)
}

func NewHost(t *testing.T, ds fleet.Datastore, name, ip, key, uuid string, now time.Time) *fleet.Host {
	osqueryHostID, _ := server.GenerateRandomText(10)
	h, err := ds.NewHost(&fleet.Host{
		Hostname:        name,
		NodeKey:         key,
		UUID:            uuid,
		DetailUpdatedAt: now,
		LabelUpdatedAt:  now,
		SeenTime:        now,
		OsqueryHostID:   osqueryHostID,
	})

	require.Nil(t, err)
	require.NotZero(t, h.ID)
	require.Nil(t, ds.MarkHostSeen(h, now))

	return h
}

func NewUser(t *testing.T, ds fleet.Datastore, name, email string, admin bool) *fleet.User {
	role := fleet.RoleObserver
	if admin {
		role = fleet.RoleAdmin
	}
	u, err := ds.NewUser(&fleet.User{
		Password:   []byte("garbage"),
		Salt:       "garbage",
		Name:       name,
		Email:      email,
		GlobalRole: &role,
	})

	require.Nil(t, err)
	require.NotZero(t, u.ID)

	return u
}

func NewScheduledQuery(t *testing.T, ds fleet.Datastore, pid, qid, interval uint, snapshot, removed bool, name string) *fleet.ScheduledQuery {
	sq, err := ds.NewScheduledQuery(&fleet.ScheduledQuery{
		Name:     name,
		PackID:   pid,
		QueryID:  qid,
		Interval: interval,
		Snapshot: &snapshot,
		Removed:  &removed,
	})
	require.Nil(t, err)
	require.NotZero(t, sq.ID)

	return sq
}
