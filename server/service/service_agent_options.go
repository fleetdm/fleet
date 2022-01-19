package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// AgentOptionsForHost gets the agent options for the provided host.
// The host information should be used for filtering based on team, platform, etc.
func (svc *Service) AgentOptionsForHost(ctx context.Context, hostTeamID *uint, hostPlatform string) (json.RawMessage, error) {
	// Team agent options have priority over global options.
	if hostTeamID != nil {
		teamAgentOptions, err := svc.ds.TeamAgentOptions(ctx, *hostTeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load team agent options for host")
		}

		if teamAgentOptions != nil && len(*teamAgentOptions) > 0 {
			var options fleet.AgentOptions
			if err := json.Unmarshal(*teamAgentOptions, &options); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "unmarshal team agent options")
			}
			return options.ForPlatform(hostPlatform), nil
		}
	}
	// Otherwise return the appropriate override for global options.
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load global agent options")
	}
	var options fleet.AgentOptions
	if appConfig.AgentOptions != nil {
		if err := json.Unmarshal(*appConfig.AgentOptions, &options); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshal global agent options")
		}
	}
	return options.ForPlatform(hostPlatform), nil
}
