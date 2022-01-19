package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// TODO(mna): when migrating Session-related endpoints, add auth tests for those
// endpoints (the auth is session-based, not user-based).
//_, err = svc.GetInfoAboutSessionsForUser(ctx, 999)
//checkAuthErr(t, tt.shouldFailTeamWrite, err)
//_, err = svc.GetInfoAboutSessionsForUser(ctx, 888)
//checkAuthErr(t, tt.shouldFailGlobalWrite, err)

func TestSessionAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.InviteByTokenFunc = func(ctx context.Context, token string) (*fleet.Invite, error) {
		return &fleet.Invite{
			Email: "some@email.com",
			Token: "ABCD",
			UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
				CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
				UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
			},
		}, nil
	}
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		return &fleet.User{}, nil
	}
	ds.DeleteInviteFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, errors.New("AA")
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		if id == 999 {
			return &fleet.User{
				ID:    999,
				Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}},
			}, nil
		}
		return &fleet.User{
			ID:         888,
			GlobalRole: ptr.String(fleet.RoleMaintainer),
		}, nil
	}
	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		return nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opts fleet.UserListOptions) ([]*fleet.User, error) {
		return nil, nil
	}
	ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.DestroyAllSessionsForUserFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.ListSessionsForUserFunc = func(ctx context.Context, id uint) ([]*fleet.Session, error) {
		return nil, nil
	}

	testCases := []struct {
		name                  string
		user                  *fleet.User
		shouldFailGlobalWrite bool
		shouldFailTeamWrite   bool
		shouldFailRead        bool
		shouldFailDeleteReset bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			true,
			false,
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
			false,
			true,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
			false,
			true,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			false,
			true,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			false,
			true,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			false,
			true,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
			false,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
			false,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			teams := []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}
			_, err := svc.CreateUser(ctx, fleet.UserPayload{
				Name:     ptr.String("Some Name"),
				Email:    ptr.String("some@email.com"),
				Password: ptr.String("passw0rd."),
				Teams:    &teams,
			})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.CreateUser(ctx, fleet.UserPayload{
				Name:       ptr.String("Some Name"),
				Email:      ptr.String("some@email.com"),
				Password:   ptr.String("passw0rd."),
				GlobalRole: ptr.String(fleet.RoleAdmin),
			})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyUser(ctx, 999, fleet.UserPayload{Teams: &teams})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			_, err = svc.ModifyUser(ctx, 888, fleet.UserPayload{Teams: &teams})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			_, err = svc.ModifyUser(ctx, 888, fleet.UserPayload{GlobalRole: ptr.String(fleet.RoleMaintainer)})
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			err = svc.DeleteUser(ctx, 999)
			checkAuthErr(t, tt.shouldFailDeleteReset, err)

			_, err = svc.RequirePasswordReset(ctx, 999, false)
			checkAuthErr(t, tt.shouldFailDeleteReset, err)

			_, err = svc.ListUsers(ctx, fleet.UserListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.User(ctx, 999)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.User(ctx, 888)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}
