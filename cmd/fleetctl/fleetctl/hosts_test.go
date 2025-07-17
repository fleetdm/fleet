package fleetctl

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostTransferFlagChecks(t *testing.T) {
	testing_utils.RunServerWithMockedDS(t)

	RunAppCheckErr(t,
		[]string{"hosts", "transfer", "--team", "team1", "--hosts", "host1", "--label", "AAA"},
		"--hosts cannot be used along side any other flag",
	)
	RunAppCheckErr(t,
		[]string{"hosts", "transfer", "--team", "team1"},
		"You need to define either --hosts, or one or more of --label, --status, --search_query",
	)
}

func TestHostsTransferByHosts(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		require.Equal(t, "host1", identifier)
		return &fleet.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		require.Equal(t, "team1", name)
		return &fleet.Team{ID: 99, Name: "team1"}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
		require.NotNil(t, params.TeamID)
		require.Equal(t, uint(99), *params.TeamID)
		require.Equal(t, []uint{42}, params.HostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, fleet.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", RunAppForTest(t, []string{"hosts", "transfer", "--team", "team1", "--hosts", "host1"}))
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)

	// Now, transfer out of the team.
	ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
		assert.Nil(t, params.TeamID)
		assert.Equal(t, []uint{42}, params.HostIDs)
		return nil
	}
	ds.NewActivityFuncInvoked = false
	ds.AddHostsToTeamFuncInvoked = false
	assert.Equal(t, "", RunAppForTest(t, []string{"hosts", "transfer", "--team", "", "--hosts", "host1"}))
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
}

func TestHostsTransferByLabel(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		require.Equal(t, "host1", identifier)
		return &fleet.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		require.Equal(t, "team1", name)
		return &fleet.Team{ID: 99, Name: "team1"}, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Equal(t, []string{"label1"}, labels)
		return map[string]uint{"label1": uint(11)}, nil
	}

	ds.ListHostsInLabelFunc = func(ctx context.Context, filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		require.Equal(t, fleet.HostStatus(""), opt.StatusFilter)
		require.Equal(t, uint(11), lid)
		return []*fleet.Host{{ID: 32}, {ID: 12}}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
		require.NotNil(t, params.TeamID)
		require.Equal(t, uint(99), *params.TeamID)
		require.Equal(t, []uint{32, 12}, params.HostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, fleet.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", RunAppForTest(t, []string{"hosts", "transfer", "--team", "team1", "--label", "label1"}))
	require.True(t, ds.NewActivityFuncInvoked)
	assert.True(t, ds.AddHostsToTeamFuncInvoked)

	// Now, transfer out of the team.
	ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
		assert.Nil(t, params.TeamID)
		require.Equal(t, []uint{32, 12}, params.HostIDs)
		return nil
	}
	ds.NewActivityFuncInvoked = false
	ds.AddHostsToTeamFuncInvoked = false
	assert.Equal(t, "", RunAppForTest(t, []string{"hosts", "transfer", "--team", "", "--label", "label1"}))
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
}

func TestHostsTransferByStatus(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		require.Equal(t, "host1", identifier)
		return &fleet.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		require.Equal(t, "team1", name)
		return &fleet.Team{ID: 99, Name: "team1"}, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Equal(t, []string{"label1"}, labels)
		return map[string]uint{"label1": uint(11)}, nil
	}

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		require.Equal(t, fleet.StatusOnline, opt.StatusFilter)
		return []*fleet.Host{{ID: 32}, {ID: 12}}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
		require.NotNil(t, params.TeamID)
		require.Equal(t, uint(99), *params.TeamID)
		require.Equal(t, []uint{32, 12}, params.HostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, fleet.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", RunAppForTest(t,
		[]string{"hosts", "transfer", "--team", "team1", "--status", "online"}))
	require.True(t, ds.NewActivityFuncInvoked)
}

func TestHostsTransferByStatusAndSearchQuery(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		require.Equal(t, "host1", identifier)
		return &fleet.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		require.Equal(t, "team1", name)
		return &fleet.Team{ID: 99, Name: "team1"}, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Equal(t, []string{"label1"}, labels)
		return map[string]uint{"label1": uint(11)}, nil
	}

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		require.Equal(t, fleet.StatusOnline, opt.StatusFilter)
		require.Equal(t, "somequery", opt.MatchQuery)
		return []*fleet.Host{{ID: 32}, {ID: 12}}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
		require.NotNil(t, params.TeamID)
		require.Equal(t, uint(99), *params.TeamID)
		require.Equal(t, []uint{32, 12}, params.HostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, fleet.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", RunAppForTest(t,
		[]string{"hosts", "transfer", "--team", "team1", "--status", "online", "--search_query", "somequery"}))
	require.True(t, ds.NewActivityFuncInvoked)
}
