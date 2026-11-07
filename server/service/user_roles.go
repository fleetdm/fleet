package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"gopkg.in/guregu/null.v3"
)

type applyUserRoleSpecsRequest struct {
	Spec *fleet.UsersRoleSpec `json:"spec"`
}

type applyUserRoleSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyUserRoleSpecsResponse) Error() error { return r.Err }

func applyUserRoleSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyUserRoleSpecsRequest)
	err := svc.ApplyUserRolesSpecs(ctx, *req.Spec)
	if err != nil {
		return applyUserRoleSpecsResponse{Err: err}, nil
	}
	return applyUserRoleSpecsResponse{}, nil
}

func (svc *Service) ApplyUserRolesSpecs(ctx context.Context, specs fleet.UsersRoleSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionWrite); err != nil {
		return err
	}

	var users []*fleet.User
	for email, spec := range specs.Roles {
		user, err := svc.ds.UserByEmail(ctx, email)
		if err != nil {
			return err
		}
		// If an admin is downgraded, make sure there is at least one other admin
		err = svc.checkAtLeastOneAdmin(ctx, user, spec, email)
		if err != nil {
			return err
		}
		user.GlobalRole = spec.GlobalRole
		var teams []fleet.UserTeam
		for _, team := range spec.Teams {
			t, err := svc.ds.TeamByName(ctx, team.Name)
			if err != nil {
				if fleet.IsNotFound(err) {
					return &fleet.BadRequestError{
						Message:     err.Error(),
						InternalErr: err,
					}
				}
				return err
			}
			teams = append(teams, fleet.UserTeam{
				Team: *t,
				Role: team.Role,
			})
		}
		user.Teams = teams
		users = append(users, user)
	}

	previous, err := svc.ds.SaveUsers(ctx, users)
	if err != nil {
		return err
	}

	// Spec keys that differ only in case resolve to one user, and the last entry saved wins.
	saved := make(map[uint]*fleet.User, len(users))
	for _, user := range users {
		saved[user.ID] = user
	}

	adminUser := authz.UserFromContext(ctx)
	for id, user := range saved {
		old := previous[id]
		if err := fleet.LogRoleChangeActivities(ctx, svc, adminUser, old.GlobalRole, old.Teams, user, false); err != nil {
			return ctxerr.Wrap(ctx, err, "log activities for role change")
		}
	}
	return nil
}

func (svc *Service) checkAtLeastOneAdmin(ctx context.Context, user *fleet.User, spec *fleet.UserRoleSpec, email string) error {
	if null.StringFromPtr(user.GlobalRole).ValueOrZero() == fleet.RoleAdmin &&
		null.StringFromPtr(spec.GlobalRole).ValueOrZero() != fleet.RoleAdmin {
		users, err := svc.ds.ListUsers(ctx, fleet.UserListOptions{})
		if err != nil {
			return err
		}
		adminsExceptCurrent := 0
		for _, u := range users {
			if u.Email == email {
				continue
			}
			if null.StringFromPtr(u.GlobalRole).ValueOrZero() == fleet.RoleAdmin {
				adminsExceptCurrent++
			}
		}
		if adminsExceptCurrent == 0 {
			return fleet.NewError(fleet.ErrNoOneAdminNeeded, "You need at least one admin")
		}
	}
	return nil
}
