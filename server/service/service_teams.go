package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/server/kolide"
)

func (svc service) NewTeam(ctx context.Context, p kolide.TeamPayload) (*kolide.Team, error) {
	team := &kolide.Team{}

	if p.Name == nil {
		return nil, newInvalidArgumentError("name", "missing required argument")
	}
	if *p.Name == "" {
		return nil, newInvalidArgumentError("name", "may not be empty")
	}
	team.Name = *p.Name

	if p.Description != nil {
		team.Description = *p.Description
	}

	team, err := svc.ds.NewTeam(team)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (svc service) ModifyTeam(ctx context.Context, id uint, payload kolide.TeamPayload) (*kolide.Team, error) {
	team, err := svc.ds.Team(id)
	if err != nil {
		return nil, err
	}
	if payload.Name != nil {
		if *payload.Name == "" {
			return nil, newInvalidArgumentError("name", "may not be empty")
		}
		team.Name = *payload.Name
	}
	if payload.Description != nil {
		team.Description = *payload.Description
	}

	return svc.ds.SaveTeam(team)
}

func (svc service) AddTeamUsers(ctx context.Context, teamID uint, users []kolide.TeamUser) (*kolide.Team, error) {
	idMap := make(map[uint]kolide.TeamUser)
	for _, user := range users {
		if !kolide.ValidTeamRole(user.Role) {
			return nil, newInvalidArgumentError("users", fmt.Sprintf("%s is not a valid role for a team user", user.Role))
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

	return svc.ds.SaveTeam(team)
}

func (svc service) DeleteTeamUsers(ctx context.Context, teamID uint, users []kolide.TeamUser) (*kolide.Team, error) {
	idMap := make(map[uint]bool)
	for _, user := range users {
		idMap[user.ID] = true
	}

	team, err := svc.ds.Team(teamID)
	if err != nil {
		return nil, err
	}

	newUsers := []kolide.TeamUser{}
	// Delete existing
	for _, existingUser := range team.Users {
		if _, ok := idMap[existingUser.ID]; !ok {
			// Only add non-deleted users
			newUsers = append(newUsers, existingUser)
		}
	}
	team.Users = newUsers

	return svc.ds.SaveTeam(team)
}

func (svc service) ListTeamUsers(ctx context.Context, teamID uint, opt kolide.ListOptions) ([]*kolide.User, error) {
	team, err := svc.ds.Team(teamID)
	if err != nil {
		return nil, err
	}

	return svc.ds.ListUsers(kolide.UserListOptions{ListOptions: opt, TeamID: team.ID})
}

func (svc service) ListTeams(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Team, error) {
	return svc.ds.ListTeams(opt)
}

func (svc service) DeleteTeam(ctx context.Context, tid uint) error {
	return svc.ds.DeleteTeam(tid)
}
