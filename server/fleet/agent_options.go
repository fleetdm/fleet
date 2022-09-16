package fleet

import (
	"encoding/json"
)

type AgentOptions struct {
	// Config is the base config options.
	Config json.RawMessage `json:"config"`
	// Overrides includes any platform-based overrides.
	Overrides AgentOptionsOverrides `json:"overrides,omitempty"`
	// CommandLineStartUpFlags are the osquery CLI_FLAGS
	CommandLineStartUpFlags json.RawMessage `json:"command_line_flags,omitempty"`
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
