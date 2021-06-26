package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc *Service) AgentOptionsForHost(ctx context.Context, host *fleet.Host) (json.RawMessage, error) {
	// If host has a team and team has non-empty options, prioritize that.
	if host.TeamID != nil {
		team, err := svc.ds.Team(*host.TeamID)
		if err != nil {
			return nil, errors.Wrap(err, "load team for host")
		}

		if team.AgentOptions != nil && len(*team.AgentOptions) > 0 {
			var options fleet.AgentOptions
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

	var options fleet.AgentOptions
	if appConfig.AgentOptions != nil {
		if err := json.Unmarshal(*appConfig.AgentOptions, &options); err != nil {
			return nil, errors.Wrap(err, "unmarshal global agent options")
		}
	}

	return options.ForPlatform(host.Platform), nil
}
