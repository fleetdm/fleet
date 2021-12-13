package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Create User
////////////////////////////////////////////////////////////////////////////////

type createUserRequest struct {
	fleet.UserPayload
}

type createUserResponse struct {
	User *fleet.User `json:"user,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r createUserResponse) error() error { return r.Err }

func createUserEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createUserRequest)
	user, err := svc.CreateUser(ctx, req.UserPayload)
	if err != nil {
		return createUserResponse{Err: err}, nil
	}
	return createUserResponse{User: user}, nil
}

func (svc *Service) CreateUser(ctx context.Context, p fleet.UserPayload) (*fleet.User, error) {
	var teams []fleet.UserTeam
	if p.Teams != nil {
		teams = *p.Teams
	}
	if err := svc.authz.Authorize(ctx, &fleet.User{Teams: teams}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if invite, err := svc.ds.InviteByEmail(ctx, *p.Email); err == nil && invite != nil {
		return nil, ctxerr.Errorf(ctx, "%s already invited", *p.Email)
	}

	if p.AdminForcedPasswordReset == nil {
		// By default, force password reset for users created this way.
		p.AdminForcedPasswordReset = ptr.Bool(true)
	}

	return svc.newUser(ctx, p)
}

////////////////////////////////////////////////////////////////////////////////
// List Users
////////////////////////////////////////////////////////////////////////////////

type listUsersRequest struct {
	ListOptions fleet.UserListOptions `url:"user_options"`
}

type listUsersResponse struct {
	Users []fleet.User `json:"users"`
	Err   error        `json:"error,omitempty"`
}

func (r listUsersResponse) error() error { return r.Err }

func listUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listUsersRequest)
	users, err := svc.ListUsers(ctx, req.ListOptions)
	if err != nil {
		return listUsersResponse{Err: err}, nil
	}

	resp := listUsersResponse{Users: []fleet.User{}}
	for _, user := range users {
		resp.Users = append(resp.Users, *user)
	}
	return resp, nil
}

func (svc *Service) ListUsers(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(ctx, opt)
}
