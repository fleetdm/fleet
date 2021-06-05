package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/server/fleet"
)

func (svc *Service) NewTeam(ctx context.Context, p fleet.TeamPayload) (*fleet.Team, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) ModifyTeam(ctx context.Context, id uint, payload fleet.TeamPayload) (*fleet.Team, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, id uint, options json.RawMessage) (*fleet.Team, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []fleet.TeamUser) (*fleet.Team, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]*fleet.User, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) ListTeams(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Team, error) {
	return nil, fleet.ErrMissingLicense
}

func (svc *Service) DeleteTeam(ctx context.Context, tid uint) error {
	return fleet.ErrMissingLicense
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	return nil, fleet.ErrMissingLicense
}
