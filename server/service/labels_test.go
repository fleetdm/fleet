package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
		lbl.ID = 1
		if lbl.Name == "Other label" {
			lbl.ID = 2
		}
		return lbl, nil
	}
	ds.SaveLabelFunc = func(ctx context.Context, lbl *fleet.Label, filter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
		return &fleet.LabelWithTeamName{Label: *lbl}, nil, nil
	}
	ds.DeleteLabelFunc = func(ctx context.Context, nm string, filter fleet.TeamFilter) error {
		return nil
	}
	ds.ApplyLabelSpecsFunc = func(ctx context.Context, specs []*fleet.LabelSpec) error {
		return nil
	}
	ds.LabelFunc = func(ctx context.Context, id uint, filter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
		switch id {
		case uint(1):
			return &fleet.LabelWithTeamName{Label: fleet.Label{ID: id, AuthorID: &filter.User.ID}}, nil, nil
		case uint(2):
			return &fleet.LabelWithTeamName{Label: fleet.Label{ID: id}}, nil, nil
		}

		return nil, nil, ctxerr.Wrap(ctx, notFoundErr{"label", fleet.ErrorWithUUID{}})
	}
	ds.LabelByNameFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) (*fleet.Label, error) {
		return &fleet.Label{ID: 2, Name: name}, nil // for deletes, TODO add cases for authorship/team differences
	}
	ds.ListLabelsFunc = func(ctx context.Context, filter fleet.TeamFilter, opts fleet.ListOptions, includeHostCounts bool) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.LabelsSummaryFunc = func(ctx context.Context, filter fleet.TeamFilter) ([]*fleet.LabelSummary, error) {
		return nil, nil
	}
	ds.ListHostsInLabelFunc = func(ctx context.Context, filter fleet.TeamFilter, lid uint, opts fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}
	ds.GetLabelSpecsFunc = func(ctx context.Context, filter fleet.TeamFilter) ([]*fleet.LabelSpec, error) {
		return nil, nil
	}
	ds.GetLabelSpecFunc = func(ctx context.Context, filter fleet.TeamFilter, name string) (*fleet.LabelSpec, error) {
		return &fleet.LabelSpec{}, nil
	}

	testCases := []struct {
		name                          string
		user                          *fleet.User
		shouldFailGlobalWrite         bool
		shouldFailGlobalRead          bool
		shouldFailGlobalWriteIfAuthor bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			false,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			false,
			true,
		},
	}

	// add a new label authored by no one so we can check writes for labels that aren't authored by the user
	otherLabel, _, err := svc.NewLabel(viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleMaintainer)}}), fleet.LabelPayload{Name: "Other label", Query: "SELECT 0"})
	require.NoError(t, err)

	// Create a team and team label for testing team permissions
	team1 := &fleet.Team{ID: 1, Name: "team1"}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == team1.ID {
			return team1, nil
		}
		return nil, ctxerr.Wrap(ctx, notFoundErr{"team", fleet.ErrorWithUUID{}})
	}

	// Create team label
	teamLabel := &fleet.Label{
		ID:     3,
		Name:   "Team label",
		Query:  "SELECT 1",
		TeamID: &team1.ID,
	}
	ds.NewLabelFunc = func(ctx context.Context, lbl *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
		lbl.ID = 1
		if lbl.Name == "Other label" {
			lbl.ID = 2
		}
		if lbl.Name == "Team label" {
			lbl.ID = 3
			lbl.TeamID = &team1.ID
		}
		return lbl, nil
	}
	ds.LabelFunc = func(ctx context.Context, id uint, filter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
		switch id {
		case uint(1):
			return &fleet.LabelWithTeamName{Label: fleet.Label{ID: id, AuthorID: &filter.User.ID}}, nil, nil
		case uint(2):
			return &fleet.LabelWithTeamName{Label: fleet.Label{ID: id}}, nil, nil
		case uint(3):
			return &fleet.LabelWithTeamName{Label: fleet.Label{ID: id, TeamID: &team1.ID}}, nil, nil
		}
		return nil, nil, ctxerr.Wrap(ctx, notFoundErr{"label", fleet.ErrorWithUUID{}})
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			myLabel, _, err := svc.NewLabel(ctx, fleet.LabelPayload{Name: t.Name(), Query: `SELECT 1`})
			checkAuthErr(t, tt.shouldFailGlobalWriteIfAuthor, err) // team write users can still create global labels

			_, _, err = svc.ModifyLabel(ctx, otherLabel.ID, fleet.ModifyLabelPayload{})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			if myLabel != nil {
				_, _, err = svc.ModifyLabel(ctx, myLabel.ID, fleet.ModifyLabelPayload{})
				checkAuthErr(t, tt.shouldFailGlobalWriteIfAuthor, err)
			}

			err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{}, nil, nil)
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, _, err = svc.GetLabel(ctx, otherLabel.ID)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.GetLabelSpecs(ctx, nil)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.GetLabelSpec(ctx, "abc")
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.ListLabels(ctx, fleet.ListOptions{}, nil, true)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.LabelsSummary(ctx, nil)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.ListHostsInLabel(ctx, 1, fleet.HostListOptions{})
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			err = svc.DeleteLabel(ctx, "abc")
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			err = svc.DeleteLabelByID(ctx, otherLabel.ID)
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			if myLabel != nil {
				err = svc.DeleteLabelByID(ctx, myLabel.ID)
				checkAuthErr(t, tt.shouldFailGlobalWriteIfAuthor, err)
			}

			// Test team label permissions
			// Team maintainers can create team labels
			isTeamMaintainer := len(tt.user.Teams) > 0 && tt.user.Teams[0].Role == fleet.RoleMaintainer
			isGlobalWrite := tt.user.GlobalRole != nil && (*tt.user.GlobalRole == fleet.RoleAdmin || *tt.user.GlobalRole == fleet.RoleMaintainer)

			// Try to get team label
			_, _, err = svc.GetLabel(ctx, teamLabel.ID)
			if !isGlobalWrite && !isTeamMaintainer {
				checkAuthErr(t, true, err) // Should fail for observers and team members without access
			} else {
				// Global admins/maintainers and team maintainers should succeed
				// But observers should still be able to read
				checkAuthErr(t, false, err)
			}

			// Try to modify team label
			_, _, err = svc.ModifyLabel(ctx, teamLabel.ID, fleet.ModifyLabelPayload{})
			if isGlobalWrite || isTeamMaintainer {
				checkAuthErr(t, false, err) // Should succeed for global admins/maintainers and team maintainers
			} else {
				checkAuthErr(t, true, err) // Should fail for others
			}

			// Try to delete team label
			err = svc.DeleteLabelByID(ctx, teamLabel.ID)
			if isGlobalWrite || isTeamMaintainer {
				checkAuthErr(t, false, err) // Should succeed for global admins/maintainers and team maintainers
			} else {
				checkAuthErr(t, true, err) // Should fail for others
			}
		})
	}
}

func TestListLabelsHostCountOptions(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	ds.ListLabelsFunc = func(ctx context.Context, filter fleet.TeamFilter, opts fleet.ListOptions, includeHostCounts bool) ([]*fleet.Label, error) {
		// Expect host counts not to be requested
		require.False(t, includeHostCounts)
		return nil, nil
	}

	// Test explicitly setting include_host_counts to false
	_, err := svc.ListLabels(ctx, fleet.ListOptions{}, nil, false)
	require.NoError(t, err)

	ds.ListLabelsFunc = func(ctx context.Context, filter fleet.TeamFilter, opts fleet.ListOptions, includeHostCounts bool) ([]*fleet.Label, error) {
		// Expect host counts to be requested
		require.True(t, includeHostCounts)
		// Expect the team filter to be set
		require.Equal(t, filter.User, user)
		return nil, nil
	}

	// Test explicitly setting include_host_counts to true
	_, err = svc.ListLabels(ctx, fleet.ListOptions{}, nil, true)
	require.NoError(t, err)
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

	labels, err := svc.ListLabels(test.UserContext(ctx, test.UserAdmin), fleet.ListOptions{Page: 0, PerPage: 1000}, nil, true)
	require.NoError(t, err)
	require.Len(t, labels, 8)

	labelsSummary, err := svc.LabelsSummary(test.UserContext(ctx, test.UserAdmin), nil)
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

	ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
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
	err := svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
	require.NoError(t, err)

	// trying to add a regular label with the same name as a built-in label should fail
	for name := range fleet.ReservedLabelNames() {
		err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{
			{
				Name:        name,
				Description: description,
				Query:       query,
				LabelType:   fleet.LabelTypeRegular,
			},
		}, nil, nil)
		assert.ErrorContains(t, err,
			fmt.Sprintf("cannot add label '%s' because it conflicts with the name of a built-in label", name))
	}

	const errorMessage = "cannot modify or add built-in label"
	// not ok -- built-in label name doesn't exist
	name = "not-foo"
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
	assert.ErrorContains(t, err, errorMessage)
	name = "foo"

	// not ok -- description does not match
	description = "not-bar"
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
	assert.ErrorContains(t, err, errorMessage)
	description = "bar"

	// not ok -- query does not match
	query = "select * from not-foo;"
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
	assert.ErrorContains(t, err, errorMessage)
	query = "select * from foo;"

	// not ok -- label type does not match
	labelType = fleet.LabelTypeRegular
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
	assert.ErrorContains(t, err, errorMessage)
	labelType = fleet.LabelTypeBuiltIn

	// not ok -- label membership type does not match
	labelMembershipType = fleet.LabelMembershipTypeManual
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
	assert.ErrorContains(t, err, errorMessage)
	labelMembershipType = fleet.LabelMembershipTypeDynamic

	// not ok -- DB error
	ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
		return nil, assert.AnError
	}
	err = svc.ApplyLabelSpecs(ctx, []*fleet.LabelSpec{spec}, nil, nil)
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

	lblWithName, hostIDs, err := svc.ModifyLabel(ctx, lbl.ID, fleet.ModifyLabelPayload{Hosts: []string{"host1"}})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID}, hostIDs)
	require.Equal(t, 1, lblWithName.HostCount)
	require.Equal(t, user.ID, *lblWithName.AuthorID)

	// reading this label without replication returns the old data as it only uses the reader
	lblWithName, hostIDs, err = svc.GetLabel(ctx, lblWithName.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID, h2.ID}, hostIDs)
	require.Equal(t, 2, lblWithName.HostCount)
	require.Equal(t, user.ID, *lblWithName.AuthorID)

	// running the replication makes the updated data available
	opts.RunReplication("labels", "label_membership")

	lblWithName, hostIDs, err = svc.GetLabel(ctx, lblWithName.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{h1.ID}, hostIDs)
	require.Equal(t, 1, lblWithName.HostCount)
	require.Equal(t, user.ID, *lblWithName.AuthorID)
}

func TestBatchValidateLabels(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	t.Run("no auth context", func(t *testing.T) {
		_, err := svc.BatchValidateLabels(context.Background(), nil, nil)
		require.ErrorContains(t, err, "Authentication required")
	})

	authCtx := authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, &authCtx)

	t.Run("no auth checked", func(t *testing.T) {
		_, err := svc.BatchValidateLabels(ctx, nil, nil)
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

	ds.LabelIDsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]uint, error) {
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
	ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
		res := make(map[string]*fleet.Label)
		if names == nil {
			return res, nil
		}
		for _, name := range names {
			if id, ok := mockLabels[name]; ok {
				res[name] = &fleet.Label{
					ID:   id,
					Name: name,
				}
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
			got, err := svc.BatchValidateLabels(ctx, nil, tt.labelNames)
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
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, label fleet.Label, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), label.ID)
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
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, label fleet.Label, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), label.ID)
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

	ds.LabelFunc = func(ctx context.Context, lid uint, teamFilter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
		return &fleet.LabelWithTeamName{
			Label: fleet.Label{
				ID:                  lid,
				LabelMembershipType: fleet.LabelMembershipTypeManual,
			},
		}, nil, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
		return []uint{99, 100}, nil
	}
	ds.SaveLabelFunc = func(ctx context.Context, lbl *fleet.Label, filter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
		return nil, nil, nil
	}

	t.Run("using hostnames", func(t *testing.T) {
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, label fleet.Label, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), label.ID)
			require.Equal(t, []uint{99, 100}, hostIds)
			return nil, nil, nil
		}
		_, _, err := svc.ModifyLabel(ctx, 1, fleet.ModifyLabelPayload{
			Hosts: []string{"host1", "host2"},
		})
		require.NoError(t, err)
	})

	t.Run("using IDs", func(t *testing.T) {
		ds.UpdateLabelMembershipByHostIDsFunc = func(ctx context.Context, label fleet.Label, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
			require.Equal(t, uint(1), label.ID)
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
		assert.Equal(t, "SELECT %s FROM %s RIGHT JOIN host_scim_user ON (hosts.id = host_scim_user.host_id) JOIN scim_users ON (host_scim_user.scim_user_id = scim_users.id) LEFT JOIN scim_user_group ON (host_scim_user.scim_user_id = scim_user_group.scim_user_id) LEFT JOIN scim_groups ON (scim_user_group.group_id = scim_groups.id) WHERE scim_groups.display_name = ? GROUP BY hosts.id", query)
		assert.Equal(t, `["admin"]`, string(queryValuesJson))
	})
}
