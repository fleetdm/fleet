package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"golang.org/x/text/unicode/norm"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// List Teams
func listTeamsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListTeamsRequest)
	teams, err := svc.ListTeams(ctx, req.ListOptions)
	if err != nil {
		return fleet.ListTeamsResponse{Err: err}, nil
	}

	resp := fleet.ListTeamsResponse{Teams: []fleet.Team{}}
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

// Get Team
func getTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetTeamRequest)

	team, err := svc.GetTeam(ctx, req.ID)
	if err != nil {
		return fleet.GetTeamResponse{Err: err}, nil
	}

	// Special handling for team ID 0 - return DefaultTeam structure
	if team.ID == 0 {
		defaultTeam := &fleet.DefaultTeam{
			ID:   team.ID,
			Name: team.Name,
			DefaultTeamConfig: fleet.DefaultTeamConfig{
				WebhookSettings: fleet.DefaultTeamWebhookSettings{
					FailingPoliciesWebhook: team.Config.WebhookSettings.FailingPoliciesWebhook,
				},
				Integrations: fleet.DefaultTeamIntegrations{
					Jira:    team.Config.Integrations.Jira,
					Zendesk: team.Config.Integrations.Zendesk,
				},
			},
		}
		return fleet.DefaultTeamResponse{Team: defaultTeam}, nil
	}

	return fleet.GetTeamResponse{Team: team}, nil
}

func (svc *Service) GetTeam(ctx context.Context, tid uint) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Create Team
func createTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CreateTeamRequest)

	team, err := svc.NewTeam(ctx, req.TeamPayload)
	if err != nil {
		return fleet.TeamResponse{Err: err}, nil
	}
	return fleet.TeamResponse{Team: team}, nil
}

func (svc *Service) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Modify Team
func modifyTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyTeamRequest)

	// AppleOSUpdateSettings.UpdateNewHosts is only used in macOS ... so ignore any values sent for iOS/iPadOS
	if req.TeamPayload.MDM != nil {
		if req.TeamPayload.MDM.IOSUpdates != nil {
			req.TeamPayload.MDM.IOSUpdates.UpdateNewHosts = optjson.Bool{}
		}
		if req.TeamPayload.MDM.IPadOSUpdates != nil {
			req.TeamPayload.MDM.IPadOSUpdates.UpdateNewHosts = optjson.Bool{}
		}
	}

	team, err := svc.ModifyTeam(ctx, req.ID, req.TeamPayload)
	if err != nil {
		return fleet.TeamResponse{Err: err}, nil
	}

	// Special handling for team ID 0 - return limited fields
	if req.ID == 0 {
		// Convert to DefaultTeam with limited fields
		defaultTeam := &fleet.DefaultTeam{
			ID:   team.ID,
			Name: team.Name,
			DefaultTeamConfig: fleet.DefaultTeamConfig{
				WebhookSettings: fleet.DefaultTeamWebhookSettings{
					FailingPoliciesWebhook: team.Config.WebhookSettings.FailingPoliciesWebhook,
				},
				Integrations: fleet.DefaultTeamIntegrations{
					Jira:    team.Config.Integrations.Jira,
					Zendesk: team.Config.Integrations.Zendesk,
				},
			},
		}
		return fleet.DefaultTeamResponse{Team: defaultTeam}, nil
	}

	return fleet.TeamResponse{Team: team}, err
}

func (svc *Service) ModifyTeam(ctx context.Context, id uint, payload fleet.TeamPayload) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Delete Team
func deleteTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteTeamRequest)
	err := svc.DeleteTeam(ctx, req.ID)
	if err != nil {
		return fleet.DeleteTeamResponse{Err: err}, nil
	}
	return fleet.DeleteTeamResponse{}, nil
}

func (svc *Service) DeleteTeam(ctx context.Context, tid uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

// Apply Team Specs
func applyTeamSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ApplyTeamSpecsRequest)
	if !req.DryRun {
		req.DryRunAssumptions = nil
	}

	// remove any nil spec (may happen in conversion from YAML to JSON with fleetctl, but also
	// with the API should someone send such JSON)
	actualSpecs := make([]*fleet.TeamSpec, 0, len(req.Specs))
	for _, spec := range req.Specs {
		if spec != nil {
			// Normalize the team name for full Unicode support to prevent potential issue further in the spec flow
			spec.Name = norm.NFC.String(spec.Name)
			actualSpecs = append(actualSpecs, spec)
		}
	}

	idsByName, err := svc.ApplyTeamSpecs(
		ctx, actualSpecs, fleet.ApplyTeamSpecOptions{
			ApplySpecOptions: fleet.ApplySpecOptions{
				Force:  req.Force,
				DryRun: req.DryRun,
			},
			DryRunAssumptions: req.DryRunAssumptions,
		})
	if err != nil {
		return fleet.ApplyTeamSpecsResponse{Err: err}, nil
	}
	return fleet.ApplyTeamSpecsResponse{TeamIDsByName: idsByName}, nil
}

func (svc Service) ApplyTeamSpecs(ctx context.Context, _ []*fleet.TeamSpec, _ fleet.ApplyTeamSpecOptions) (map[string]uint, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Modify Team Agent Options
func modifyTeamAgentOptionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyTeamAgentOptionsRequest)
	team, err := svc.ModifyTeamAgentOptions(ctx, req.ID, req.RawMessage, fleet.ApplySpecOptions{
		Force:  req.Force,
		DryRun: req.DryRun,
	})
	if err != nil {
		return fleet.TeamResponse{Err: err}, nil
	}
	return fleet.TeamResponse{Team: team}, err
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, id uint, teamOptions json.RawMessage, applyOptions fleet.ApplySpecOptions) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// List Team Users
func listTeamUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListTeamUsersRequest)
	users, err := svc.ListTeamUsers(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return fleet.ListUsersResponse{Err: err}, nil
	}

	resp := fleet.ListUsersResponse{Users: []fleet.User{}}
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

// Add / Delete Team Users
// same request struct for add and delete

func addTeamUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyTeamUsersRequest)
	team, err := svc.AddTeamUsers(ctx, req.TeamID, req.Users)
	if err != nil {
		return fleet.TeamResponse{Err: err}, nil
	}
	return fleet.TeamResponse{Team: team}, err
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

func deleteTeamUsersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyTeamUsersRequest)
	team, err := svc.DeleteTeamUsers(ctx, req.TeamID, req.Users)
	if err != nil {
		return fleet.TeamResponse{Err: err}, nil
	}
	return fleet.TeamResponse{Team: team}, err
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Get enroll secrets for team
func teamEnrollSecretsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.TeamEnrollSecretsRequest)
	secrets, err := svc.TeamEnrollSecrets(ctx, req.TeamID)
	if err != nil {
		return fleet.TeamEnrollSecretsResponse{Err: err}, nil
	}

	return fleet.TeamEnrollSecretsResponse{Secrets: secrets}, err
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Modify enroll secrets for team
func modifyTeamEnrollSecretsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyTeamEnrollSecretsRequest)
	secrets, err := svc.ModifyTeamEnrollSecrets(ctx, req.TeamID, req.Secrets)
	if err != nil {
		return fleet.TeamEnrollSecretsResponse{Err: err}, nil
	}

	return fleet.TeamEnrollSecretsResponse{Secrets: secrets}, err
}

func (svc *Service) ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []fleet.EnrollSecret) ([]*fleet.EnrollSecret, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
