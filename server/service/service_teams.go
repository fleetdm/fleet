package service

import (
	"context"

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
