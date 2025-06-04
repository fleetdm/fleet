package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
)

func TestCreateSecretVariables(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) error {
		return nil
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

				err := svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "foo", Value: "bar"}}, false)
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
		assert.ErrorContains(t, err, "Couldn't save secret variables. Missing required private key")
		testSetEmptyPrivateKey = false

		ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []fleet.SecretVariable) error {
			return errors.New("test error")
		}
		err = svc.CreateSecretVariables(ctx, []fleet.SecretVariable{{Name: "foo", Value: "bar"}}, false)
		assert.ErrorContains(t, err, "test error")
	})

}
