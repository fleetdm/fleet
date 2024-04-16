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
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewLabelFunc = func(ctx context.Context, lbl *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
		return lbl, nil
	}
	ds.SaveLabelFunc = func(ctx context.Context, lbl *fleet.Label) (*fleet.Label, []uint, error) {
		return lbl, nil, nil
	}
	ds.DeleteLabelFunc = func(ctx context.Context, nm string) error {
		return nil
	}
	ds.ApplyLabelSpecsFunc = func(ctx context.Context, specs []*fleet.LabelSpec) error {
		return nil
	}
	ds.LabelFunc = func(ctx context.Context, id uint) (*fleet.Label, []uint, error) {
		return &fleet.Label{}, nil, nil
	}
	ds.ListLabelsFunc = func(ctx context.Context, filter fleet.TeamFilter, opts fleet.ListOptions) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.LabelsSummaryFunc = func(ctx context.Context) ([]*fleet.LabelSummary, error) {
		return nil, nil
	}
	ds.ListHostsInLabelFunc = func(ctx context.Context, filter fleet.TeamFilter, lid uint, opts fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}
	ds.GetLabelSpecsFunc = func(ctx context.Context) ([]*fleet.LabelSpec, error) {
		return nil, nil
	}
	ds.GetLabelSpecFunc = func(ctx context.Context, name string) (*fleet.LabelSpec, error) {
		return &fleet.LabelSpec{}, nil
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
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, _, err := svc.NewLabel(ctx, fleet.LabelPayload{Name: t.Name(), Query: `SELECT 1`})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, _, err = svc.ModifyLabel(ctx, 1, fleet.ModifyLabelPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, _, err = svc.GetLabel(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetLabelSpecs(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetLabelSpec(ctx, "abc")
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ListLabels(ctx, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.LabelsSummary((ctx))
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ListHostsInLabel(ctx, 1, fleet.HostListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			err = svc.DeleteLabel(ctx, "abc")
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteLabelByID(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
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
	svc, ctx := newTestService(t, ds, nil, nil)

	label := &fleet.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err := ds.NewLabel(ctx, label)
	assert.Nil(t, err)
	assert.NotZero(t, label.ID)

	labelVerify, _, err := svc.GetLabel(test.UserContext(ctx, test.UserAdmin), label.ID)
	assert.Nil(t, err)
	assert.Equal(t, label.ID, labelVerify.ID)
}

func testLabelsListLabels(t *testing.T, ds *mysql.Datastore) {
	svc, ctx := newTestService(t, ds, nil, nil)
	require.NoError(t, ds.MigrateData(context.Background()))

	labels, err := svc.ListLabels(test.UserContext(ctx, test.UserAdmin), fleet.ListOptions{Page: 0, PerPage: 1000})
	require.NoError(t, err)
	require.Len(t, labels, 8)

	labelsSummary, err := svc.LabelsSummary(test.UserContext(ctx, test.UserAdmin))
	require.NoError(t, err)
	require.Len(t, labelsSummary, 8)
}
