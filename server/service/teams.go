package service

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// List Teams
////////////////////////////////////////////////////////////////////////////////

type listTeamsRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listTeamsResponse struct {
	Teams []fleet.Team `json:"teams"`
	Err   error        `json:"error,omitempty"`
}

func (r listTeamsResponse) error() error { return r.Err }

func listTeamsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listTeamsRequest)
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

func (svc *Service) ListTeams(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Create Team
////////////////////////////////////////////////////////////////////////////////

type createTeamRequest struct {
	fleet.TeamPayload
}

type teamResponse struct {
	Team *fleet.Team `json:"team,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r teamResponse) error() error { return r.Err }

func createTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createTeamRequest)

	team, err := svc.NewTeam(ctx, req.TeamPayload)
	if err != nil {
		return teamResponse{Err: err}, nil
	}

	return teamResponse{Team: team}, nil
}

func (svc *Service) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Modify Team
////////////////////////////////////////////////////////////////////////////////

type modifyTeamRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.TeamPayload
}

func modifyTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyTeamRequest)
	team, err := svc.ModifyTeam(ctx, req.ID, req.TeamPayload)
	if err != nil {
		return teamResponse{Err: err}, nil
	}

	return teamResponse{Team: team}, err
}

func (svc *Service) ModifyTeam(ctx context.Context, id uint, payload fleet.TeamPayload) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Delete Team
////////////////////////////////////////////////////////////////////////////////

type deleteTeamRequest struct {
	ID uint `url:"id"`
}

type deleteTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteTeamResponse) error() error { return r.Err }

func deleteTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteTeamRequest)
	err := svc.DeleteTeam(ctx, req.ID)
	if err != nil {
		return deleteTeamResponse{Err: err}, nil
	}
	return deleteTeamResponse{}, nil
}

func (svc *Service) DeleteTeam(ctx context.Context, tid uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Apply Team Specs
////////////////////////////////////////////////////////////////////////////////

type applyTeamSpecsRequest struct {
	Specs []*fleet.TeamSpec `json:"specs"`
}

type applyTeamSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyTeamSpecsResponse) error() error { return r.Err }

func applyTeamSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*applyTeamSpecsRequest)
	err := svc.ApplyTeamSpecs(ctx, req.Specs)
	if err != nil {
		return applyTeamSpecsResponse{Err: err}, nil
	}
	return applyTeamSpecsResponse{}, nil
}

func (svc Service) ApplyTeamSpecs(ctx context.Context, specs []*fleet.TeamSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return err
	}

	config, err := svc.AppConfig(ctx)
	if err != nil {
		return err
	}

	for _, spec := range specs {
		var secrets []*fleet.EnrollSecret
		for _, secret := range spec.Secrets {
			secrets = append(secrets, &fleet.EnrollSecret{
				Secret: secret.Secret,
			})
		}

		team, err := svc.ds.TeamByName(ctx, spec.Name)
		if err != nil {
			if err := ctxerr.Cause(err); err == sql.ErrNoRows {
				agentOptions := spec.AgentOptions
				if agentOptions == nil {
					agentOptions = config.AgentOptions
				}
				_, err = svc.ds.NewTeam(ctx, &fleet.Team{
					Name:         spec.Name,
					AgentOptions: agentOptions,
					Secrets:      secrets,
				})
				if err != nil {
					return err
				}
				continue
			}

			return err
		}
		team.Name = spec.Name
		team.AgentOptions = spec.AgentOptions
		team.Secrets = secrets

		_, err = svc.ds.SaveTeam(ctx, team)
		if err != nil {
			return err
		}

		err = svc.ds.ApplyEnrollSecrets(ctx, ptr.Uint(team.ID), secrets)
		if err != nil {
			return err
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Modify Team Agent Options
////////////////////////////////////////////////////////////////////////////////

type modifyTeamAgentOptionsRequest struct {
	ID uint `json:"-" url:"id"`
	json.RawMessage
}

func modifyTeamAgentOptionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyTeamAgentOptionsRequest)
	team, err := svc.ModifyTeamAgentOptions(ctx, req.ID, req.RawMessage)
	if err != nil {
		return teamResponse{Err: err}, nil
	}

	return teamResponse{Team: team}, err
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, id uint, options json.RawMessage) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// List Team Users
////////////////////////////////////////////////////////////////////////////////

type listTeamUsersRequest struct {
	TeamID      uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

func listTeamUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listTeamUsersRequest)
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

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]*fleet.User, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Add / Delete Team Users
////////////////////////////////////////////////////////////////////////////////

// same request struct for add and delete
type modifyTeamUsersRequest struct {
	TeamID uint `json:"-" url:"id"`
	// User ID and role must be specified for add users, user ID must be
	// specified for delete users.
	Users []fleet.TeamUser `json:"users"`
}

func addTeamUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyTeamUsersRequest)
	team, err := svc.AddTeamUsers(ctx, req.TeamID, req.Users)
	if err != nil {
		return teamResponse{Err: err}, nil
	}

	return teamResponse{Team: team}, err
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

func deleteTeamUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyTeamUsersRequest)
	team, err := svc.DeleteTeamUsers(ctx, req.TeamID, req.Users)
	if err != nil {
		return teamResponse{Err: err}, nil
	}

	return teamResponse{Team: team}, err
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get enroll secrets for team
////////////////////////////////////////////////////////////////////////////////

type teamEnrollSecretsRequest struct {
	TeamID uint `url:"id"`
}

type teamEnrollSecretsResponse struct {
	Secrets []*fleet.EnrollSecret `json:"secrets"`
	Err     error                 `json:"error,omitempty"`
}

func (r teamEnrollSecretsResponse) error() error { return r.Err }

func teamEnrollSecretsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*teamEnrollSecretsRequest)
	secrets, err := svc.TeamEnrollSecrets(ctx, req.TeamID)
	if err != nil {
		return teamEnrollSecretsResponse{Err: err}, nil
	}

	return teamEnrollSecretsResponse{Secrets: secrets}, err
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Modify enroll secrets for team
////////////////////////////////////////////////////////////////////////////////

type modifyTeamEnrollSecretsRequest struct {
	TeamID  uint                 `url:"team_id"`
	Secrets []fleet.EnrollSecret `json:"secrets"`
}

func modifyTeamEnrollSecretsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyTeamEnrollSecretsRequest)
	secrets, err := svc.ModifyTeamEnrollSecrets(ctx, req.TeamID, req.Secrets)
	if err != nil {
		return teamEnrollSecretsResponse{Err: err}, nil
	}

	return teamEnrollSecretsResponse{Secrets: secrets}, err
}

func (svc *Service) ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []fleet.EnrollSecret) ([]*fleet.EnrollSecret, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
