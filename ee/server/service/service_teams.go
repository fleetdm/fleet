package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc *Service) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Copy team options from global options
	globalConfig, err := svc.ds.AppConfig()
	if err != nil {
		return nil, err
	}
	team := &fleet.Team{
		AgentOptions: globalConfig.AgentOptions,
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
			return nil, errors.Wrap(err, "generate enroll secret string")
		}
		team.Secrets = []*fleet.EnrollSecret{{Secret: secret}}
	}

	team, err = svc.ds.NewTeam(team)
	if err != nil {
		return nil, err
	}

	if err := svc.ds.NewActivity(
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

	team, err := svc.ds.Team(teamID)
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
	if payload.Secrets != nil {
		team.Secrets = payload.Secrets
	}

	return svc.ds.SaveTeam(team)
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, teamID uint, options json.RawMessage) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(teamID)
	if err != nil {
		return nil, err
	}

	if options != nil {
		team.AgentOptions = &options
	} else {
		team.AgentOptions = nil
	}

	return svc.ds.SaveTeam(team)
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	idMap := make(map[uint]fleet.TeamUser)
	for _, user := range users {
		if !fleet.ValidTeamRole(user.Role) {
			return nil, fleet.NewInvalidArgumentError("users", fmt.Sprintf("%s is not a valid role for a team user", user.Role))
		}
		idMap[user.ID] = user
	}

	team, err := svc.ds.Team(teamID)
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

	return svc.ds.SaveTeam(team)
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	idMap := make(map[uint]bool)
	for _, user := range users {
		idMap[user.ID] = true
	}

	team, err := svc.ds.Team(teamID)
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

	return svc.ds.SaveTeam(team)
}

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]*fleet.User, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	team, err := svc.ds.Team(teamID)
	if err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(fleet.UserListOptions{ListOptions: opt, TeamID: team.ID})
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

	return svc.ds.ListTeams(filter, opt)
}

func (svc *Service) DeleteTeam(ctx context.Context, teamID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	team, err := svc.ds.Team(teamID)
	if err != nil {
		return err
	}
	name := team.Name

	if err := svc.ds.DeleteTeam(teamID); err != nil {
		return err
	}

	logging.WithExtras(ctx, "id", teamID)

	return svc.ds.NewActivity(
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedTeam,
		&map[string]interface{}{"team_id": teamID, "team_name": name},
	)
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.TeamEnrollSecrets(teamID)
}
