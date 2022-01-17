package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// List Team Users
////////////////////////////////////////////////////////////////////////////////

type listTeamUsersRequest struct {
	TeamID      uint
	ListOptions fleet.ListOptions
}

func makeListTeamUsersEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listTeamUsersRequest)
		users, err := svc.ListTeamUsers(ctx, req.TeamID, req.ListOptions)
		if err != nil {
			return listUsersResponse{Err: err}, nil
		}

		resp := listUsersResponse{Users: []fleet.User{}}
		for _, user := range users {
			resp.Users = append(resp.Users, *user)
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Add / Delete Team Users
////////////////////////////////////////////////////////////////////////////////

type modifyTeamUsersRequest struct {
	TeamID uint // From request path
	// User ID and role must be specified for add users, user ID must be
	// specified for delete users.
	Users []fleet.TeamUser `json:"users"`
}

func makeAddTeamUsersEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyTeamUsersRequest)
		team, err := svc.AddTeamUsers(ctx, req.TeamID, req.Users)
		if err != nil {
			return teamResponse{Err: err}, nil
		}

		return teamResponse{Team: team}, err
	}
}

func makeDeleteTeamUsersEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyTeamUsersRequest)
		team, err := svc.DeleteTeamUsers(ctx, req.TeamID, req.Users)
		if err != nil {
			return teamResponse{Err: err}, nil
		}

		return teamResponse{Team: team}, err
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get enroll secrets for team
////////////////////////////////////////////////////////////////////////////////

type teamEnrollSecretsRequest struct {
	TeamID uint
}

type teamEnrollSecretsResponse struct {
	Secrets []*fleet.EnrollSecret `json:"secrets"`
	Err     error                 `json:"error,omitempty"`
}

func (r teamEnrollSecretsResponse) error() error { return r.Err }

func makeTeamEnrollSecretsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(teamEnrollSecretsRequest)
		secrets, err := svc.TeamEnrollSecrets(ctx, req.TeamID)
		if err != nil {
			return teamEnrollSecretsResponse{Err: err}, nil
		}

		return teamEnrollSecretsResponse{Secrets: secrets}, err
	}
}
