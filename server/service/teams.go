package service

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/pkg/errors"
)

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
			if err := errors.Cause(err); err == sql.ErrNoRows {
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
