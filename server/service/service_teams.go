package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/server/kolide"
)

func (svc *Service) NewTeam(ctx context.Context, p kolide.TeamPayload) (*kolide.Team, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) ModifyTeam(ctx context.Context, id uint, payload kolide.TeamPayload) (*kolide.Team, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, id uint, options json.RawMessage) (*kolide.Team, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []kolide.TeamUser) (*kolide.Team, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []kolide.TeamUser) (*kolide.Team, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt kolide.ListOptions) ([]*kolide.User, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) ListTeams(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Team, error) {
	return nil, kolide.ErrMissingLicense
}

func (svc *Service) DeleteTeam(ctx context.Context, tid uint) error {
	return kolide.ErrMissingLicense
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*kolide.EnrollSecret, error) {
	return nil, kolide.ErrMissingLicense
}
