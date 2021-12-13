package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelsAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.NewLabelFunc = func(ctx context.Context, lbl *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
		return lbl, nil
	}
	ds.SaveLabelFunc = func(ctx context.Context, lbl *fleet.Label) (*fleet.Label, error) {
		return lbl, nil
	}
	ds.LabelFunc = func(ctx context.Context, id uint) (*fleet.Label, error) {
		return &fleet.Label{}, nil
	}
	ds.ListLabelsFunc = func(ctx context.Context, filter fleet.TeamFilter, opts fleet.ListOptions) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.ListHostsInLabelFunc = func(ctx context.Context, filter fleet.TeamFilter, lid uint, opts fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.NewLabel(ctx, fleet.LabelPayload{Name: ptr.String(t.Name()), Query: ptr.String(`SELECT 1`)})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ModifyLabel(ctx, 1, fleet.ModifyLabelPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetLabel(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ListLabels(ctx, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ListHostsInLabel(ctx, 1, fleet.HostListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

func TestLabelsWithDS(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *mysql.Datastore)
	}{
		{"GetLabel", testLabelsGetLabel},
		{"ListLabels", testLabelsListLabels},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testLabelsGetLabel(t *testing.T, ds *mysql.Datastore) {
	svc := newTestService(ds, nil, nil)

	label := &fleet.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err := ds.NewLabel(context.Background(), label)
	assert.Nil(t, err)
	assert.NotZero(t, label.ID)

	labelVerify, err := svc.GetLabel(test.UserContext(test.UserAdmin), label.ID)
	assert.Nil(t, err)
	assert.Equal(t, label.ID, labelVerify.ID)
}

func testLabelsListLabels(t *testing.T, ds *mysql.Datastore) {
	svc := newTestService(ds, nil, nil)
	require.NoError(t, ds.MigrateData(context.Background()))

	labels, err := svc.ListLabels(test.UserContext(test.UserAdmin), fleet.ListOptions{Page: 0, PerPage: 1000})
	require.NoError(t, err)
	require.Len(t, labels, 7)
}
