package fleet

import (
	"context"
	"encoding/json"
)

type AgentOptionsService interface {
	// AgentOptionsForHost gets the agent options for the provided host.
	//
	// The host information should be used for filtering based on team,
	// platform, etc.
	AgentOptionsForHost(ctx context.Context, host *Host) (json.RawMessage, error)
}

type AgentOptions struct {
	// Config is the base config options.
	Config json.RawMessage `json:"config"`
	// Overrides includes any platform-based overrides.
	Overrides AgentOptionsOverrides `json:"overrides,omitempty"`
}

type AgentOptionsOverrides struct {
	// Platforms is a map from platform name to the config override.
	Platforms map[string]json.RawMessage `json:"platforms,omitempty"`
}

func (o *AgentOptions) ForPlatform(platform string) json.RawMessage {
	// Return matching platform override if available.
	if opt, ok := o.Overrides.Platforms[platform]; ok {
		return opt
	}

	// Otherwise return base config for team.
	return o.Config
}
