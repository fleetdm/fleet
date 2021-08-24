package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Create Team
////////////////////////////////////////////////////////////////////////////////

type createTeamRequest struct {
	payload fleet.TeamPayload
}

type teamResponse struct {
	Team *fleet.Team `json:"team,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r teamResponse) error() error { return r.Err }

func makeCreateTeamEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createTeamRequest)

		team, err := svc.NewTeam(ctx, req.payload)
		if err != nil {
			return teamResponse{Err: err}, nil
		}

		return teamResponse{Team: team}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Team
////////////////////////////////////////////////////////////////////////////////

type modifyTeamRequest struct {
	ID      uint
	payload fleet.TeamPayload
}

func makeModifyTeamEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyTeamRequest)
		team, err := svc.ModifyTeam(ctx, req.ID, req.payload)
		if err != nil {
			return teamResponse{Err: err}, nil
		}

		return teamResponse{Team: team}, err
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Team Agent Options
////////////////////////////////////////////////////////////////////////////////

type modifyTeamAgentOptionsRequest struct {
	ID      uint
	options json.RawMessage
}

func makeModifyTeamAgentOptionsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyTeamAgentOptionsRequest)
		team, err := svc.ModifyTeamAgentOptions(ctx, req.ID, req.options)
		if err != nil {
			return teamResponse{Err: err}, nil
		}

		return teamResponse{Team: team}, err
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Teams
////////////////////////////////////////////////////////////////////////////////

type listTeamsRequest struct {
	ListOptions fleet.ListOptions
}

type listTeamsResponse struct {
	Teams []fleet.Team `json:"teams"`
	Err   error        `json:"error,omitempty"`
}

func (r listTeamsResponse) error() error { return r.Err }

func makeListTeamsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listTeamsRequest)
		teams, err := svc.ListTeams(ctx, req.ListOptions)
		if err != nil {
			return listTeamsResponse{Err: err}, nil
		}

		resp := listTeamsResponse{Teams: []fleet.Team{}}
		for _, team := range teams {
			resp.Teams = append(resp.Teams, *team)
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Team
////////////////////////////////////////////////////////////////////////////////

type deleteTeamRequest struct {
	ID uint `json:"id"`
}

type deleteTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteTeamResponse) error() error { return r.Err }

func makeDeleteTeamEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteTeamRequest)
		err := svc.DeleteTeam(ctx, req.ID)
		if err != nil {
			return deleteTeamResponse{Err: err}, nil
		}
		return deleteTeamResponse{}, nil
	}
}

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
