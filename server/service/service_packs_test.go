package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceListPacks(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)

	queries, err := svc.ListPacks(test.UserContext(test.UserAdmin), fleet.PackListOptions{IncludeSystemPacks: false})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(context.Background(), &fleet.Pack{
		Name: "foo",
	})
	assert.Nil(t, err)

	queries, err = svc.ListPacks(test.UserContext(test.UserAdmin), fleet.PackListOptions{IncludeSystemPacks: false})
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestNewSavesTargets(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.NewPackFunc = func(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
		return pack, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	packPayload := fleet.PackPayload{
		Name:     ptr.String("foo"),
		HostIDs:  &[]uint{123},
		LabelIDs: &[]uint{456},
		TeamIDs:  &[]uint{789},
	}
	pack, _ := svc.NewPack(test.UserContext(test.UserAdmin), packPayload)

	require.Len(t, pack.HostIDs, 1)
	require.Len(t, pack.LabelIDs, 1)
	require.Len(t, pack.TeamIDs, 1)
	assert.Equal(t, uint(123), pack.HostIDs[0])
	assert.Equal(t, uint(456), pack.LabelIDs[0])
	assert.Equal(t, uint(789), pack.TeamIDs[0])
}

func TestService_ModifyPack_GlobalPack(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

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

func TestService_DeletePackByID_GlobalPack(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()
	test.AddAllHostsLabel(t, ds)

	globalPack, err := ds.EnsureGlobalPack(context.Background())
	require.NoError(t, err)

	type fields struct {
		ds fleet.Datastore
	}
	type args struct {
		ctx context.Context
		id  uint
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "cannot delete global pack",
			fields: fields{
				ds,
			},
			args: args{
				ctx: test.UserContext(test.UserAdmin),
				id:  globalPack.ID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.fields.ds, nil, nil)
			if err := svc.DeletePackByID(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("DeletePackByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_ApplyPackSpecs(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()
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

	type fields struct {
		ds fleet.Datastore
	}
	type args struct {
		ctx   context.Context
		specs []*fleet.PackSpec
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*fleet.PackSpec
		wantErr bool
	}{
		{
			name: "cannot modify global pack",
			fields: fields{
				ds,
			},
			args: args{
				ctx: test.UserContext(&user),
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
			fields: fields{
				ds,
			},
			args: args{
				ctx: test.UserContext(&user),
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
			svc := newTestService(tt.fields.ds, nil, nil)
			got, err := svc.ApplyPackSpecs(tt.args.ctx, tt.args.specs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyPackSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestService_DeletePack(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()
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

	type fields struct {
		ds fleet.Datastore
	}
	type args struct {
		ctx  context.Context
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "cannot delete global pack",
			fields: fields{
				ds: ds,
			},
			args: args{
				ctx:  test.UserContext(&user),
				name: gp.Name,
			},
			wantErr: true,
		},
		{
			name: "cannot delete team pack",
			fields: fields{
				ds: ds,
			},
			args: args{
				ctx:  test.UserContext(&user),
				name: tp.Name,
			},
			wantErr: true,
		},
		{
			name: "delete pack that doesn't exist",
			fields: fields{
				ds: ds,
			},
			args: args{
				ctx:  test.UserContext(&user),
				name: "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.fields.ds, nil, nil)
			if err := svc.DeletePack(tt.args.ctx, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("DeletePack() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
