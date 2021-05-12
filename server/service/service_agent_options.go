package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) AgentOptionsForHost(ctx context.Context, host *kolide.Host) (json.RawMessage, error) {
	// If host has a team and team has non-empty options, prioritize that.
	if host.TeamID.Valid {
		team, err := svc.ds.Team(uint(host.TeamID.Int64))
		if err != nil {
			return nil, errors.Wrap(err, "load team for host")
		}

		if team.AgentOptions != nil && len(*team.AgentOptions) > 0 {
			var options kolide.AgentOptions
			if err := json.Unmarshal(*team.AgentOptions, &options); err != nil {
				return nil, errors.Wrap(err, "unmarshal team agent options")
			}

			return options.ForPlatform(host.Platform), nil
		}
	}

	// Otherwise return the appropriate override for global options.
	appConfig, err := svc.ds.AppConfig()
	if err != nil {
		return nil, errors.Wrap(err, "load global agent options")
	}

	var options kolide.AgentOptions
	if err := json.Unmarshal(appConfig.AgentOptions, &options); err != nil {
		return nil, errors.Wrap(err, "unmarshal global agent options")
	}

	return options.ForPlatform(host.Platform), nil
}
