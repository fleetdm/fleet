package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelsAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewLabelFunc = func(ctx context.Context, lbl *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
		return lbl, nil
	}
	ds.SaveLabelFunc = func(ctx context.Context, lbl *fleet.Label, filter fleet.TeamFilter) (*fleet.Label, []uint, error) {
		return lbl, nil, nil
	}
	ds.DeleteLabelFunc = func(ctx context.Context, nm string) error {
		return nil
	}
	ds.ApplyLabelSpecsFunc = func(ctx context.Context, specs []*fleet.LabelSpec) error {
		return nil
	}
	ds.LabelFunc = func(ctx context.Context, id uint, filter fleet.TeamFilter) (*fleet.Label, []uint, error) {
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
			false,
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
	assert.Nil(t, label.AuthorID)
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

func TestApplyLabelSpecsWithBuiltInLabels(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	user := &fleet.User{
		ID:         3,
		Email:      "foo@bar.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})
	name := "foo"
	description := "bar"
	query := "select * from foo;"
	platform := ""
	labelType := fleet.LabelTypeBuiltIn
	labelMembershipType := fleet.LabelMembershipTypeDynamic
	spec := &fleet.LabelSpec{
		Name:                name,
		Description:         description,
		Query:               query,
		LabelType:           labelType,
		LabelMembershipType: labelMembershipType,
	}

	ds.LabelsByNameFunc = func(ctx context.Context, names []string) (map[string]*fleet.Label, error) {
		return map[string]*fleet.Label{
			name: {
				Name:                name,
				Description:         description,
				Query:               query,
				Platform:            platform,
				LabelType:           labelType,
				LabelMembershipType: labelMembershipType,
			},
		}, nil
	}

	// all good
	err := svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	require.NoError(t, err)

	const errorMessage = "cannot modify or add built-in label"
	// not ok -- built-in label name doesn't exist
	name = "not-foo"
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	assert.ErrorContains(t, err, errorMessage)
	name = "foo"

	// not ok -- description does not match
	description = "not-bar"
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	assert.ErrorContains(t, err, errorMessage)
	description = "bar"

	// not ok -- query does not match
	query = "select * from not-foo;"
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	assert.ErrorContains(t, err, errorMessage)
	query = "select * from foo;"

	// not ok -- label type does not match
	labelType = fleet.LabelTypeRegular
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	assert.ErrorContains(t, err, errorMessage)
	labelType = fleet.LabelTypeBuiltIn

	// not ok -- label membership type does not match
	labelMembershipType = fleet.LabelMembershipTypeManual
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	assert.ErrorContains(t, err, errorMessage)
	labelMembershipType = fleet.LabelMembershipTypeDynamic

	// not ok -- DB error
	ds.LabelsByNameFunc = func(ctx context.Context, names []string) (map[string]*fleet.Label, error) {
		return nil, assert.AnError
	}
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec})
	assert.ErrorIs(t, err, assert.AnError)
}

func TestLabelsWithReplica(t *testing.T) {
	opts := &testing_utils.DatastoreTestOptions{DummyReplica: true}
	ds := mysql.CreateMySQLDSWithOptions(t, opts)
	defer ds.Close()

	svc, ctx := newTestService(t, ds, nil, nil)
	user, err := ds.NewUser(ctx, &fleet.User{
		Name:       "Adminboi",
		Password:   []byte("p4ssw0rd.123"),
		Email:      "admin@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	// create a couple hosts
	h1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "host1",
		HardwareSerial:  uuid.NewString(),
		UUID:            uuid.NewString(),
		Platform:        "darwin",
		LastEnrolledAt:  time.Now(),
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)
	h2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "host2",
		HardwareSerial:  uuid.NewString(),
		UUID:            uuid.NewString(),
		Platform:        "darwin",
		LastEnrolledAt:  time.Now(),
		DetailUpdatedAt: time.Now(),
	})
	require.NoError(t, err)
	// make the newly-created hosts available to the reader
	opts.RunReplication()

	lbl, hostIDs, err := svc.NewLabel(ctx, fleet.LabelPayload{Name: "label1", Hosts: []string{"host1", "host2"}})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID, h2.ID}, hostIDs)
	require.Equal(t, 2, lbl.HostCount)
	require.Equal(t, user.ID, *lbl.AuthorID)

	// make the newly-created label available to the reader
	opts.RunReplication("labels", "label_membership")

	lbl, hostIDs, err = svc.ModifyLabel(ctx, lbl.ID, fleet.ModifyLabelPayload{Hosts: []string{"host1"}})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID}, hostIDs)
	require.Equal(t, 1, lbl.HostCount)
	require.Equal(t, user.ID, *lbl.AuthorID)

	// reading this label without replication returns the old data as it only uses the reader
	lbl, hostIDs, err = svc.GetLabel(ctx, lbl.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID, h2.ID}, hostIDs)
	require.Equal(t, 2, lbl.HostCount)
	require.Equal(t, user.ID, *lbl.AuthorID)

	// running the replication makes the updated data available
	opts.RunReplication("labels", "label_membership")

	lbl, hostIDs, err = svc.GetLabel(ctx, lbl.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID}, hostIDs)
	require.Equal(t, 1, lbl.HostCount)
	require.Equal(t, user.ID, *lbl.AuthorID)
}

func TestBatchValidateLabels(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	t.Run("no auth context", func(t *testing.T) {
		_, err := svc.BatchValidateLabels(context.Background(), nil)
		require.ErrorContains(t, err, "Authentication required")
	})

	authCtx := authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, &authCtx)

	t.Run("no auth checked", func(t *testing.T) {
		_, err := svc.BatchValidateLabels(ctx, nil)
		require.ErrorContains(t, err, "Authentication required")
	})

	// validator requires that an authz check has been performed upstream so we'll set it now for
	// the rest of the tests
	authCtx.SetChecked()

	mockLabels := map[string]uint{
		"foo": 1,
		"bar": 2,
		"baz": 3,
	}

	mockLabelIdent := func(name string, id uint) fleet.LabelIdent {
		return fleet.LabelIdent{LabelID: id, LabelName: name}
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) (map[string]uint, error) {
		res := make(map[string]uint)
		if names == nil {
			return res, nil
		}
		for _, name := range names {
			if id, ok := mockLabels[name]; ok {
				res[name] = id
			}
		}
		return res, nil
	}

	testCases := []struct {
		name         string
		labelNames   []string
		expectLabels map[string]fleet.LabelIdent
		expectError  string
	}{
		{
			"no labels",
			nil,
			nil,
			"",
		},
		{
			"include labels",
			[]string{"foo", "bar"},
			map[string]fleet.LabelIdent{
				"foo": mockLabelIdent("foo", 1),
				"bar": mockLabelIdent("bar", 2),
			},
			"",
		},
		{
			"non-existent label",
			[]string{"foo", "qux"},
			nil,
			"some or all the labels provided don't exist",
		},
		{
			"duplicate label",
			[]string{"foo", "foo"},
			map[string]fleet.LabelIdent{
				"foo": mockLabelIdent("foo", 1),
			},
			"",
		},
		{
			"empty slice",
			[]string{},
			nil,
			"",
		},
		{
			"empty string",
			[]string{""},
			nil,
			"some or all the labels provided don't exist",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.BatchValidateLabels(ctx, tt.labelNames)
			if tt.expectError != "" {
				require.Contains(t, err.Error(), tt.expectError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectLabels, got)
			}
		})
	}
}

func TestNewManualLabel(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.NewLabelFunc = func(ctx context.Context, lbl *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
		lbl.ID = 1
		lbl.LabelMembershipType = fleet.LabelMembershipTypeManual
		return lbl, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
		return []uint{99, 100}, nil
	}

	t.Run("using hostnames", func(t *testing.T) {
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, labelID uint, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), labelID)
			require.Equal(t, []uint{99, 100}, hostIds)
			return nil, nil, nil
		}
		_, _, err := svc.NewLabel(ctx, fleet.LabelPayload{
			Name:  "foo",
			Hosts: []string{"host1", "host2"},
		})
		require.NoError(t, err)
	})

	t.Run("using IDs", func(t *testing.T) {
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, labelID uint, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), labelID)
			require.Equal(t, []uint{1, 2}, hostIds)
			return nil, nil, nil
		}
		_, _, err := svc.NewLabel(ctx, fleet.LabelPayload{
			Name:    "foo",
			HostIDs: []uint{1, 2},
		})
		require.NoError(t, err)
	})
}

func TestModifyManualLabel(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.LabelFunc = func(ctx context.Context, lid uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
		return &fleet.Label{
			ID:                  lid,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		}, nil, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
		return []uint{99, 100}, nil
	}
	ds.SaveLabelFunc = func(ctx context.Context, lbl *fleet.Label, filter fleet.TeamFilter) (*fleet.Label, []uint, error) {
		return nil, nil, nil
	}

	t.Run("using hostnames", func(t *testing.T) {
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, labelID uint, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), labelID)
			require.Equal(t, []uint{99, 100}, hostIds)
			return nil, nil, nil
		}
		_, _, err := svc.ModifyLabel(ctx, 1, fleet.ModifyLabelPayload{
			Hosts: []string{"host1", "host2"},
		})
		require.NoError(t, err)
	})

	t.Run("using IDs", func(t *testing.T) {
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, labelID uint, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), labelID)
			require.Equal(t, []uint{1, 2}, hostIds)
			return nil, nil, nil
		}
		_, _, err := svc.ModifyLabel(ctx, 1, fleet.ModifyLabelPayload{
			HostIDs: []uint{1, 2},
		})
		require.NoError(t, err)
	})
}

func TestNewHostVitalsLabel(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	ds.NewLabelFunc = func(ctx context.Context, lbl *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
		return lbl, nil
	}

	t.Run("create host vitals label", func(t *testing.T) {
		lbl, _, err := svc.NewLabel(ctx, fleet.LabelPayload{
			Name: "foo",
			Criteria: &fleet.HostVitalCriteria{
				Vital: ptr.String("end_user_idp_group"),
				Value: ptr.String("admin"),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, fleet.LabelTypeRegular, lbl.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeHostVitals, lbl.LabelMembershipType)

		// Test parsing the criteria
		query, queryValues, err := lbl.CalculateHostVitalsQuery()
		require.NoError(t, err)
		queryValuesJson, err := json.Marshal(queryValues)
		require.NoError(t, err)
		assert.Equal(t, "SELECT %s FROM %s RIGHT JOIN host_scim_user ON (hosts.id = host_scim_user.host_id) JOIN scim_users ON (host_scim_user.scim_user_id = scim_users.id) JOIN scim_user_group ON (host_scim_user.scim_user_id = scim_user_group.scim_user_id) JOIN scim_groups ON (scim_user_group.group_id = scim_groups.id) WHERE scim_groups.display_name = ? GROUP BY hosts.id", query)
		assert.Equal(t, `["admin"]`, string(queryValuesJson))
	})
}
