package service

import (
	"context"
	"errors"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCreateSecretVariables(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) (created []string, updated []string, err error) {
		return nil, nil, nil
	}

	t.Run("authorization checks", func(t *testing.T) {
		testCases := []struct {
			name       string
			user       *fleet.User
			shouldFail bool
		}{
			{
				name:       "global admin",
				user:       &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				shouldFail: false,
			},
			{
				name:       "global maintainer",
				user:       &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
				shouldFail: false,
			},
			{
				name:       "global gitops",
				user:       &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
				shouldFail: false,
			},
			{
				name:       "global observer",
				user:       &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				shouldFail: true,
			},
			{
				name:       "global observer+",
				user:       &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
				shouldFail: true,
			},
			{
				name:       "team admin",
				user:       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
				shouldFail: true,
			},
			{
				name:       "team maintainer",
				user:       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
				shouldFail: true,
			},
			{
				name:       "team observer",
				user:       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
				shouldFail: true,
			},
			{
				name:       "team observer+",
				user:       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
				shouldFail: true,
			},
			{
				name:       "team gitops",
				user:       &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
				shouldFail: true,
			},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

				err := svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "FOO", Value: "bar"}}, false)
				checkAuthErr(t, tt.shouldFail, err)
			})
		}
	})

	t.Run("failure test", func(t *testing.T) {
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)}})
		testSetEmptyPrivateKey = true
		t.Cleanup(func() {
			testSetEmptyPrivateKey = false
		})
		err := svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "foo", Value: "bar"}}, true)
		require.ErrorContains(t, err, "Couldn't save secret variables. Missing required private key")
		testSetEmptyPrivateKey = false

		ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) (created []string, updated []string, err error) {
			return nil, nil, errors.New("test error")
		}
		err = svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "FOO", Value: "bar"}}, false)
		require.ErrorContains(t, err, "test error")
	})
}

func TestCreateSecretVariablesEmitsActivities(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	opts := &TestServerOpts{}
	svc, ctx := newTestService(t, ds, nil, nil, opts)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}})

	t.Run("emits a created activity per created variable and an updated activity per updated variable", func(t *testing.T) {
		ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) (created []string, updated []string, err error) {
			return []string{"CREATED"}, []string{"UPDATED"}, nil
		}
		var activities []activity_api.ActivityDetails
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}
		err := svc.CreateSecretVariables(ctx, []fleet.SecretVariable{
			{Name: "FLEET_SECRET_CREATED", Value: "a"},
			{Name: "FLEET_SECRET_UPDATED", Value: "b"},
		}, false)
		require.NoError(t, err)
		require.Len(t, activities, 2)

		createdActivity, ok := activities[0].(fleet.ActivityCreatedCustomVariable)
		require.True(t, ok)
		require.Equal(t, "CREATED", createdActivity.CustomVariableName)

		updatedActivity, ok := activities[1].(fleet.ActivityUpdatedCustomVariable)
		require.True(t, ok)
		require.Equal(t, "UPDATED", updatedActivity.CustomVariableName)
	})

	t.Run("emits no activity when nothing changed", func(t *testing.T) {
		ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) (created []string, updated []string, err error) {
			return nil, nil, nil
		}
		activityCalled := false
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			activityCalled = true
			return nil
		}
		err := svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "FLEET_SECRET_UNCHANGED", Value: "a"}}, false)
		require.NoError(t, err)
		require.False(t, activityCalled)
	})

	t.Run("emits no activity on a dry run", func(t *testing.T) {
		ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) (created []string, updated []string, err error) {
			t.Fatal("UpsertSecretVariables should not be called on a dry run")
			return nil, nil, nil
		}
		activityCalled := false
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			activityCalled = true
			return nil
		}
		err := svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "FLEET_SECRET_DRY", Value: "a"}}, true)
		require.NoError(t, err)
		require.False(t, activityCalled)
	})
}
