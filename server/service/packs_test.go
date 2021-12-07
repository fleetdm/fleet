package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
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
	svc := newTestService(ds, nil, nil)

	ds.PackFunc = func(ctx context.Context, id uint) (*fleet.Pack, error) {
		return &fleet.Pack{
			ID:      1,
			TeamIDs: []uint{1},
		}, nil
	}

	pack, err := svc.GetPack(test.UserContext(test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), pack.ID)

	_, err = svc.GetPack(test.UserContext(test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPacksModifyPack(t *testing.T, ds *mysql.Datastore) {
	svc := newTestService(ds, nil, nil)
	test.AddAllHostsLabel(t, ds)
	users := createTestUsers(t, ds)

	globalPack, err := ds.EnsureGlobalPack(context.Background())
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
	pack, _ := svc.ModifyPack(test.UserContext(&user), globalPack.ID, packPayload)

	require.Equal(t, "Global", pack.Name, "name for global pack should not change")
	require.Equal(t, "Global pack", pack.Description, "description for global pack should not change")
	require.Len(t, pack.LabelIDs, 1)
	require.Len(t, pack.HostIDs, 0)
	require.Len(t, pack.TeamIDs, 0)
}

func testPacksListPacks(t *testing.T, ds *mysql.Datastore) {
	svc := newTestService(ds, nil, nil)

	queries, err := svc.ListPacks(test.UserContext(test.UserAdmin), fleet.PackListOptions{IncludeSystemPacks: false})
	require.NoError(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(context.Background(), &fleet.Pack{
		Name: "foo",
	})
	require.NoError(t, err)

	queries, err = svc.ListPacks(test.UserContext(test.UserAdmin), fleet.PackListOptions{IncludeSystemPacks: false})
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
				ctx:  test.UserContext(&user),
				name: gp.Name,
			},
			wantErr: true,
		},
		{
			name: "cannot delete team pack",
			args: args{
				ctx:  test.UserContext(&user),
				name: tp.Name,
			},
			wantErr: true,
		},
		{
			name: "delete pack that doesn't exist",
			args: args{
				ctx:  test.UserContext(&user),
				name: "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(ds, nil, nil)
			if err := svc.DeletePack(tt.args.ctx, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("DeletePack() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
