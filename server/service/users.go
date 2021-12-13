package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

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
