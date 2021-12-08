package test

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func NewQuery(t *testing.T, ds fleet.Datastore, name, q string, authorID uint, saved bool) *fleet.Query {
	authorPtr := &authorID
	if authorID == 0 {
		authorPtr = nil
	}
	query, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:     name,
		Query:    q,
		AuthorID: authorPtr,
		Saved:    saved,
	})
	require.NoError(t, err)

	// Loading gives us the timestamps
	query, err = ds.Query(context.Background(), query.ID)
	require.NoError(t, err)

	return query
}

func NewPack(t *testing.T, ds fleet.Datastore, name string) *fleet.Pack {
	err := ds.ApplyPackSpecs(context.Background(), []*fleet.PackSpec{{Name: name}})
	require.Nil(t, err)

	// Loading gives us the timestamps
	pack, ok, err := ds.PackByName(context.Background(), name)
	require.True(t, ok)
	require.NoError(t, err)

	return pack
}

func NewCampaign(t *testing.T, ds fleet.Datastore, queryID uint, status fleet.DistributedQueryStatus, now time.Time) *fleet.DistributedQueryCampaign {
	campaign, err := ds.NewDistributedQueryCampaign(context.Background(), &fleet.DistributedQueryCampaign{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{
				CreatedAt: now,
			},
		},
		QueryID: queryID,
		Status:  status,
	})
	require.NoError(t, err)

	// Loading gives us the timestamps
	campaign, err = ds.DistributedQueryCampaign(context.Background(), campaign.ID)
	require.NoError(t, err)

	return campaign
}

func AddHostToCampaign(t *testing.T, ds fleet.Datastore, campaignID, hostID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		context.Background(),
		&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetHost,
			TargetID:                   hostID,
			DistributedQueryCampaignID: campaignID,
		})
	require.NoError(t, err)
}

func AddLabelToCampaign(t *testing.T, ds fleet.Datastore, campaignID, labelID uint) {
	_, err := ds.NewDistributedQueryCampaignTarget(
		context.Background(),
		&fleet.DistributedQueryCampaignTarget{
			Type:                       fleet.TargetLabel,
			TargetID:                   labelID,
			DistributedQueryCampaignID: campaignID,
		})
	require.NoError(t, err)
}

func AddAllHostsLabel(t *testing.T, ds fleet.Datastore) {
	_, err := ds.NewLabel(
		context.Background(),
		&fleet.Label{
			Name:                "All Hosts",
			Query:               "select 1",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
	)
	require.NoError(t, err)
}

func NewHost(t *testing.T, ds fleet.Datastore, name, ip, key, uuid string, now time.Time) *fleet.Host {
	osqueryHostID, _ := server.GenerateRandomText(10)
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		Hostname:        name,
		NodeKey:         key,
		UUID:            uuid,
		DetailUpdatedAt: now,
		LabelUpdatedAt:  now,
		PolicyUpdatedAt: now,
		SeenTime:        now,
		OsqueryHostID:   osqueryHostID,
		Platform:        "darwin",
	})

	require.NoError(t, err)
	require.NotZero(t, h.ID)
	require.NoError(t, ds.MarkHostsSeen(context.Background(), []uint{h.ID}, now))

	return h
}

func NewUser(t *testing.T, ds fleet.Datastore, name, email string, admin bool) *fleet.User {
	role := fleet.RoleObserver
	if admin {
		role = fleet.RoleAdmin
	}
	u, err := ds.NewUser(context.Background(), &fleet.User{
		Password:   []byte("garbage"),
		Salt:       "garbage",
		Name:       name,
		Email:      email,
		GlobalRole: &role,
	})

	require.NoError(t, err)
	require.NotZero(t, u.ID)

	return u
}

func NewScheduledQuery(t *testing.T, ds fleet.Datastore, pid, qid, interval uint, snapshot, removed bool, name string) *fleet.ScheduledQuery {
	sq, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     name,
		PackID:   pid,
		QueryID:  qid,
		Interval: interval,
		Snapshot: &snapshot,
		Removed:  &removed,
		Platform: ptr.String("darwin"),
	})
	require.NoError(t, err)
	require.NotZero(t, sq.ID)

	return sq
}
