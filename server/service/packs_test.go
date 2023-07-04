package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPack(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.PackFunc = func(ctx context.Context, id uint) (*fleet.Pack, error) {
		return &fleet.Pack{
			ID:      1,
			TeamIDs: []uint{1},
		}, nil
	}

	pack, err := svc.GetPack(test.UserContext(ctx, test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), pack.ID)

	_, err = svc.GetPack(test.UserContext(ctx, test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestNewPackSavesTargets(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewPackFunc = func(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
		return pack, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	packPayload := fleet.PackPayload{
		Name:     ptr.String("foo"),
		HostIDs:  &[]uint{123},
		LabelIDs: &[]uint{456},
		TeamIDs:  &[]uint{789},
	}
	pack, err := svc.NewPack(test.UserContext(ctx, test.UserAdmin), packPayload)
	require.NoError(t, err)

	require.Len(t, pack.HostIDs, 1)
	require.Len(t, pack.LabelIDs, 1)
	require.Len(t, pack.TeamIDs, 1)
	assert.Equal(t, uint(123), pack.HostIDs[0])
	assert.Equal(t, uint(456), pack.LabelIDs[0])
	assert.Equal(t, uint(789), pack.TeamIDs[0])
	assert.True(t, ds.NewPackFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
}

func TestPacksWithDS(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *mysql.Datastore)
	}{
		{"ModifyPack", testPacksModifyPack},
		{"ListPacks", testPacksListPacks},
		{"DeletePack", testPacksDeletePack},
		{"DeletePackByID", testPacksDeletePackByID},
		{"ApplyPackSpecs", testPacksApplyPackSpecs},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPacksModifyPack(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)
	test.AddAllHostsLabel(t, ds)
	users := createTestUsers(t, ds)

	globalPack, err := ds.EnsureGlobalPack(ctx)
	require.NoError(t, err)

	labelids := []uint{1, 2, 3}
	hostids := []uint{4, 5, 6}
	teamids := []uint{7, 8, 9}
	packPayload := fleet.PackPayload{
		Name:        ptr.String("foo"),
		Description: ptr.String("bar"),
		LabelIDs:    &labelids,
		HostIDs:     &hostids,
		TeamIDs:     &teamids,
	}

	user := users["admin1@example.com"]
	pack, _ := svc.ModifyPack(test.UserContext(ctx, &user), globalPack.ID, packPayload)

	require.Equal(t, "Global", pack.Name, "name for global pack should not change")
	require.Equal(t, "Global pack", pack.Description, "description for global pack should not change")
	require.Len(t, pack.LabelIDs, 1)
	require.Len(t, pack.HostIDs, 0)
	require.Len(t, pack.TeamIDs, 0)
}

func testPacksListPacks(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)

	queries, err := svc.ListPacks(test.UserContext(ctx, test.UserAdmin), fleet.PackListOptions{IncludeSystemPacks: false})
	require.NoError(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(ctx, &fleet.Pack{
		Name: "foo",
	})
	require.NoError(t, err)

	queries, err = svc.ListPacks(test.UserContext(ctx, test.UserAdmin), fleet.PackListOptions{IncludeSystemPacks: false})
	require.NoError(t, err)
	assert.Len(t, queries, 1)
}

func testPacksDeletePack(t *testing.T, ds *mysql.Datastore) {
	test.AddAllHostsLabel(t, ds)

	gp, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)

	users := createTestUsers(t, ds)
	user := users["admin1@example.com"]

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	tp, err := ds.EnsureTeamPack(context.Background(), team1.ID)
	require.NoError(t, err)

	type args struct {
		ctx  context.Context
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "cannot delete global pack",
			args: args{
				ctx:  test.UserContext(context.Background(), &user),
				name: gp.Name,
			},
			wantErr: true,
		},
		{
			name: "cannot delete team pack",
			args: args{
				ctx:  test.UserContext(context.Background(), &user),
				name: tp.Name,
			},
			wantErr: true,
		},
		{
			name: "delete pack that doesn't exist",
			args: args{
				ctx:  test.UserContext(context.Background(), &user),
				name: "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t, ds, nil, nil)
			if err := svc.DeletePack(tt.args.ctx, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("DeletePack() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testPacksDeletePackByID(t *testing.T, ds *mysql.Datastore) {
	test.AddAllHostsLabel(t, ds)

	globalPack, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)

	type args struct {
		ctx context.Context
		id  uint
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "cannot delete global pack",
			args: args{
				ctx: test.UserContext(context.Background(), test.UserAdmin),
				id:  globalPack.ID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t, ds, nil, nil)
			if err := svc.DeletePackByID(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("DeletePackByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testPacksApplyPackSpecs(t *testing.T, ds *mysql.Datastore) {
	test.AddAllHostsLabel(t, ds)

	global, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)

	users := createTestUsers(t, ds)
	user := users["admin1@example.com"]

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	teamPack, err := ds.EnsureTeamPack(context.Background(), team1.ID)
	require.NoError(t, err)

	type args struct {
		ctx   context.Context
		specs []*fleet.PackSpec
	}
	tests := []struct {
		name    string
		args    args
		want    []*fleet.PackSpec
		wantErr bool
	}{
		{
			name: "cannot modify global pack",
			args: args{
				ctx: test.UserContext(context.Background(), &user),
				specs: []*fleet.PackSpec{
					{Name: global.Name, Description: "bar", Platform: "baz"},
					{Name: "Foo Pack", Description: "Foo Desc", Platform: "MacOS"},
					{Name: "Bar Pack", Description: "Bar Desc", Platform: "MacOS"},
				},
			},
			want: []*fleet.PackSpec{
				{Name: "Foo Pack", Description: "Foo Desc", Platform: "MacOS"},
				{Name: "Bar Pack", Description: "Bar Desc", Platform: "MacOS"},
			},
			wantErr: false,
		},
		{
			name: "cannot modify team pack",
			args: args{
				ctx: test.UserContext(context.Background(), &user),
				specs: []*fleet.PackSpec{
					{Name: teamPack.Name, Description: "Desc", Platform: "windows"},
					{Name: "Test", Description: "Test Desc", Platform: "linux"},
				},
			},
			want: []*fleet.PackSpec{
				{Name: "Test", Description: "Test Desc", Platform: "linux"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t, ds, nil, nil)
			got, err := svc.ApplyPackSpecs(tt.args.ctx, tt.args.specs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPackSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUserIsGitOpsOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		user       *fleet.User
		expectedFn func(value bool, err error) bool
	}{
		{
			name: "missing user in context",
			user: nil,
			expectedFn: func(value bool, err error) bool {
				return err != nil && !value
			},
		},
		{
			name: "no roles",
			user: &fleet.User{},
			expectedFn: func(value bool, err error) bool {
				return err != nil && !value
			},
		},
		{
			name: "global gitops",
			user: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleGitOps),
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "global non-gitops",
			user: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleObserver),
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
		{
			name: "team gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "multiple team gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleGitOps,
					},
					{
						Team: fleet.Team{ID: 2},
						Role: fleet.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && value
			},
		},
		{
			name: "multiple teams, not all gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleObserver,
					},
					{
						Team: fleet.Team{ID: 2},
						Role: fleet.RoleGitOps,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
		{
			name: "multiple teams, none gitops",
			user: &fleet.User{
				GlobalRole: nil,
				Teams: []fleet.UserTeam{
					{
						Team: fleet.Team{ID: 1},
						Role: fleet.RoleObserver,
					},
					{
						Team: fleet.Team{ID: 2},
						Role: fleet.RoleMaintainer,
					},
				},
			},
			expectedFn: func(value bool, err error) bool {
				return err == nil && !value
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := userIsGitOpsOnly(viewer.NewContext(context.Background(), viewer.Viewer{User: tc.user}))
			require.True(t, tc.expectedFn(actual, err))
		})
	}
}
