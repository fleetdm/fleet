package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Copy team options from global options
	globalConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	team := &fleet.Team{
		Config: fleet.TeamConfig{
			AgentOptions: globalConfig.AgentOptions,
		},
	}

	if p.Name == nil {
		return nil, fleet.NewInvalidArgumentError("name", "missing required argument")
	}
	if *p.Name == "" {
		return nil, fleet.NewInvalidArgumentError("name", "may not be empty")
	}
	team.Name = *p.Name

	if p.Description != nil {
		team.Description = *p.Description
	}

	if p.Secrets != nil {
		team.Secrets = p.Secrets
	} else {
		// Set up a default enroll secret
		secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
		}
		team.Secrets = []*fleet.EnrollSecret{{Secret: secret}}
	}

	team, err = svc.ds.NewTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedTeam,
		&map[string]interface{}{"team_id": team.ID, "team_name": team.Name},
	); err != nil {
		return nil, err
	}

	return team, nil
}

func (svc *Service) ModifyTeam(ctx context.Context, teamID uint, payload fleet.TeamPayload) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		if *payload.Name == "" {
			return nil, fleet.NewInvalidArgumentError("name", "may not be empty")
		}
		team.Name = *payload.Name
	}
	if payload.Description != nil {
		team.Description = *payload.Description
	}
	if payload.WebhookSettings != nil {
		team.Config.WebhookSettings = *payload.WebhookSettings
	}

	return svc.ds.SaveTeam(ctx, team)
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, teamID uint, options json.RawMessage) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if options != nil {
		team.Config.AgentOptions = &options
	} else {
		team.Config.AgentOptions = nil
	}

	return svc.ds.SaveTeam(ctx, team)
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	currentUser := authz.UserFromContext(ctx)

	idMap := make(map[uint]fleet.TeamUser)
	for _, user := range users {
		if !fleet.ValidTeamRole(user.Role) {
			return nil, fleet.NewInvalidArgumentError("users", fmt.Sprintf("%s is not a valid role for a team user", user.Role))
		}
		idMap[user.ID] = user
		fullUser, err := svc.ds.UserByID(ctx, user.ID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "getting full user with id %d", user.ID)
		}
		if fullUser.GlobalRole != nil && currentUser.GlobalRole == nil {
			return nil, ctxerr.New(ctx, "A user with a global role cannot be added to a team by a non global user.")
		}
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Replace existing
	for i, existingUser := range team.Users {
		if user, ok := idMap[existingUser.ID]; ok {
			team.Users[i] = user
			delete(idMap, user.ID)
		}
	}

	// Add new (that have not already been replaced)
	for _, user := range idMap {
		team.Users = append(team.Users, user)
	}

	logging.WithExtras(ctx, "users", team.Users)

	return svc.ds.SaveTeam(ctx, team)
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	idMap := make(map[uint]bool)
	for _, user := range users {
		idMap[user.ID] = true
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	newUsers := []fleet.TeamUser{}
	// Delete existing
	for _, existingUser := range team.Users {
		if _, ok := idMap[existingUser.ID]; !ok {
			// Only add non-deleted users
			newUsers = append(newUsers, existingUser)
		}
	}
	team.Users = newUsers

	logging.WithExtras(ctx, "users", team.Users)

	return svc.ds.SaveTeam(ctx, team)
}

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(ctx, fleet.UserListOptions{ListOptions: opt, TeamID: team.ID})
}

func (svc *Service) ListTeams(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	return svc.ds.ListTeams(ctx, filter, opt)
}

func (svc *Service) ListAvailableTeamsForUser(ctx context.Context, user *fleet.User) ([]*fleet.TeamSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	availableTeams := []*fleet.TeamSummary{}
	if user.GlobalRole != nil {
		ts, err := svc.ds.TeamsSummary(ctx)
		if err != nil {
			return nil, err
		}
		availableTeams = append(availableTeams, ts...)
	} else {
		for _, t := range user.Teams {
			// Convert from UserTeam to TeamSummary (i.e. omit the role, counts, agent options)
			availableTeams = append(availableTeams, &fleet.TeamSummary{ID: t.ID, Name: t.Name, Description: t.Description})
		}
	}

	return availableTeams, nil
}

func (svc *Service) DeleteTeam(ctx context.Context, teamID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	team, err := svc.ds.Team(ctx, teamID)
	if err != nil {
		return err
	}
	name := team.Name

	if err := svc.ds.DeleteTeam(ctx, teamID); err != nil {
		return err
	}

	logging.WithExtras(ctx, "id", teamID)

	return svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedTeam,
		&map[string]interface{}{"team_id": teamID, "team_name": name},
	)
}

func (svc *Service) GetTeam(ctx context.Context, teamID uint) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	logging.WithExtras(ctx, "id", teamID)

	return svc.ds.Team(ctx, teamID)
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.TeamEnrollSecrets(ctx, teamID)
}

func (svc *Service) ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []fleet.EnrollSecret) ([]*fleet.EnrollSecret, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{TeamID: ptr.Uint(teamID)}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	if secrets == nil {
		return nil, fleet.NewInvalidArgumentError("secrets", "missing required argument")
	}

	var newSecrets []*fleet.EnrollSecret
	for _, secret := range secrets {
		newSecrets = append(newSecrets, &fleet.EnrollSecret{
			Secret: secret.Secret,
		})
	}
	if err := svc.ds.ApplyEnrollSecrets(ctx, ptr.Uint(teamID), newSecrets); err != nil {
		return nil, err
	}

	return newSecrets, nil
}

func (svc Service) ApplyTeamSpecs(ctx context.Context, specs []*fleet.TeamSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return err
	}

	// check auth for all teams specified first
	for _, spec := range specs {
		team, err := svc.ds.TeamByName(ctx, spec.Name)
		if err != nil {
			if err := ctxerr.Cause(err); err == sql.ErrNoRows {
				// can the user create a new team?
				if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
					return err
				}
				continue
			}

			return err
		}

		// can the user modify each team it's trying to modify
		if err := svc.authz.Authorize(ctx, team, fleet.ActionWrite); err != nil {
			return err
		}
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
					Name: spec.Name,
					Config: fleet.TeamConfig{
						AgentOptions: agentOptions,
					},
					Secrets: secrets,
				})
				if err != nil {
					return err
				}
				continue
			}

			return err
		}

		team.Name = spec.Name
		team.Config.AgentOptions = spec.AgentOptions
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
