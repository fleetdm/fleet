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

func NewQueryWithSchedule(t *testing.T, ds fleet.Datastore, teamID *uint, name, q string, authorID uint, saved bool, interval uint, automationsEnabled bool) *fleet.Query {
	authorPtr := &authorID
	if authorID == 0 {
		authorPtr = nil
	}
	query, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:               name,
		Query:              q,
		AuthorID:           authorPtr,
		Saved:              saved,
		TeamID:             teamID,
		Interval:           interval,
		AutomationsEnabled: automationsEnabled,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// Loading gives us the timestamps
	query, err = ds.Query(context.Background(), query.ID)
	require.NoError(t, err)

	return query
}

func NewQuery(t *testing.T, ds fleet.Datastore, teamID *uint, name, q string, authorID uint, saved bool) *fleet.Query {
	return NewQueryWithSchedule(t, ds, teamID, name, q, authorID, saved, 0, false)
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

func AddBuiltinLabels(t *testing.T, ds fleet.Datastore) {
	builtins := []*fleet.Label{
		{
			Name:                "All Hosts",
			Query:               "select 1",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "macOS",
			Query:               "select 1 from os_version where platform = 'darwin';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "Ubuntu Linux",
			Query:               "select 1 from os_version where platform = 'ubuntu';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "CentOS Linux",
			Query:               "select 1 from os_version where platform = 'centos' or name like '%centos%';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "MS Windows",
			Query:               "select 1 from os_version where platform = 'windows';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "Red Hat Linux",
			Query:               "SELECT 1 FROM os_version WHERE name LIKE '%red hat%'",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "All Linux",
			Query:               "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "chrome",
			Query:               "select 1 from os_version where platform = 'chrome';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                fleet.BuiltinLabelMacOS14Plus,
			Query:               "select 1 from os_version where platform = 'darwin' and major >= 14;",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
		{
			Name:                "iOS",
			Platform:            "ios",
			Query:               "",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
		{
			Name:                "iPadOS",
			Platform:            "ipados",
			Query:               "",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
		{
			Name:                "Fedora Linux",
			Platform:            "rhel",
			Query:               "select 1 from os_version where name = 'Fedora Linux';",
			LabelType:           fleet.LabelTypeBuiltIn,
			LabelMembershipType: fleet.LabelMembershipTypeDynamic,
		},
	}

	names := fleet.ReservedLabelNames()
	require.Equal(t, len(builtins), len(names))
	storedByName := map[string]*fleet.Label{}
	for _, b := range builtins {
		stored, err := ds.NewLabel(context.Background(), b)
		require.NoError(t, err)
		storedByName[stored.Name] = stored
	}
	require.Len(t, storedByName, len(builtins))

	for name := range names {
		_, ok := storedByName[name]
		require.True(t, ok, "expected label %s to be created", name)
	}
}

// NewHostOption is an Option for the NewHost function.
type NewHostOption func(*fleet.Host)

// WithComputerName sets the ComputerName in NewHost.
func WithComputerName(s string) NewHostOption {
	return func(h *fleet.Host) {
		h.ComputerName = s
	}
}

func WithPlatform(s string) NewHostOption {
	return func(h *fleet.Host) {
		h.Platform = s
	}
}

func WithOSVersion(s string) NewHostOption {
	return func(h *fleet.Host) {
		h.OSVersion = s
	}
}

func WithTeamID(teamID uint) NewHostOption {
	return func(h *fleet.Host) {
		h.TeamID = &teamID
	}
}

func NewHost(tb testing.TB, ds fleet.Datastore, name, ip, key, uuid string, now time.Time, options ...NewHostOption) *fleet.Host {
	osqueryHostID, _ := server.GenerateRandomText(10)
	h := &fleet.Host{
		Hostname:        name,
		NodeKey:         &key,
		UUID:            uuid,
		DetailUpdatedAt: now,
		LabelUpdatedAt:  now,
		PolicyUpdatedAt: now,
		SeenTime:        now,
		OsqueryHostID:   &osqueryHostID,
		Platform:        "darwin",
		PublicIP:        ip,
		PrimaryIP:       ip,
	}
	for _, o := range options {
		o(h)
	}
	h, err := ds.NewHost(context.Background(), h)
	require.NoError(tb, err)
	require.NotZero(tb, h.ID)
	require.NoError(tb, ds.MarkHostsSeen(context.Background(), []uint{h.ID}, now))

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
