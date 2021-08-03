package service

import (
	"context"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
	"gopkg.in/guregu/null.v3"
)

type applyUserRoleSpecsRequest struct {
	Spec *fleet.UsersRoleSpec `json:"spec"`
}

type applyUserRoleSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyUserRoleSpecsResponse) error() error { return r.Err }

func makeApplyUserRoleSpecsEndpoint(svc fleet.Service, opts []kithttp.ServerOption) http.Handler {
	return newServer(
		makeAuthenticatedServiceEndpoint(svc, applyUserRoleSpecsEndpoint),
		makeDecoder(applyUserRoleSpecsRequest{}),
		opts,
	)
}

func applyUserRoleSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*applyUserRoleSpecsRequest)
	err := svc.ApplyUserRolesSpecs(ctx, *req.Spec)
	if err != nil {
		return applyUserRoleSpecsResponse{Err: err}, nil
	}
	return applyUserRoleSpecsResponse{}, nil
}

func (svc Service) ApplyUserRolesSpecs(ctx context.Context, specs fleet.UsersRoleSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionWrite); err != nil {
		return err
	}

	var users []*fleet.User
	for email, spec := range specs.Roles {
		user, err := svc.ds.UserByEmail(email)
		if err != nil {
			return err
		}
		// If an admin is downgraded, make sure there is at least one other admin
		err = svc.checkAtLeastOneAdmin(user, spec, email)
		if err != nil {
			return err
		}
		user.GlobalRole = spec.GlobalRole
		var teams []fleet.UserTeam
		for _, team := range spec.Teams {
			t, err := svc.ds.TeamByName(team.Name)
			if err != nil {
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

	return svc.ds.SaveUsers(users)
}

func (svc Service) checkAtLeastOneAdmin(user *fleet.User, spec *fleet.UserRoleSpec, email string) error {
	if null.StringFromPtr(user.GlobalRole).ValueOrZero() == fleet.RoleAdmin &&
		null.StringFromPtr(spec.GlobalRole).ValueOrZero() != fleet.RoleAdmin {
		users, err := svc.ds.ListUsers(fleet.UserListOptions{})
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
