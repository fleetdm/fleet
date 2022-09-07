package fleet

import (
	"encoding/json"
	"fmt"
)

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

// ValidateJSONAgentOptions validates the given raw JSON bytes as an Agent
// Options payload. It ensures that all fields are known and have valid values.
// The validation always uses the most recent Osquery version that is available
// at the time of the Fleet release.
func ValidateJSONAgentOptions(rawJSON json.RawMessage) error {
	var opts AgentOptions
	if err := json.Unmarshal(rawJSON, &opts); err != nil {
		return fmt.Errorf("failed to unmarshal raw agent options: %w", err)
	}

	if err := validateJSONAgentOptionsSet(opts.Config); err != nil {
		return fmt.Errorf("common config: %w", err)
	}
	for platform, platformOpts := range opts.Overrides.Platforms {
		if err := validateJSONAgentOptionsSet(platformOpts); err != nil {
			return fmt.Errorf("%s platform config: %w", platform, err)
		}
	}
	return nil
}

// JSON definition of the available configuration options in osquery.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/#configuration-specification
// and `osqueryd --help` to see which CLI flags are valid as configuration options.
type osqueryAgentOptions struct {
	Options struct {
	} `json:"options"`

	Schedule struct {
	} `json:"schedule"`

	Packs struct {
	} `json:"packs"`

	FilePaths struct {
	} `json:"file_paths"`

	FileAccesses struct {
	} `json:"file_accesses"`

	YARA struct {
	} `json:"yara"`

	PrometheusTargets struct {
	} `json:"prometheus_targets"`

	Views struct {
	} `json:"views"`

	Decorators struct {
	} `json:"decorators"`

	AutoTableConstruction struct {
	} `json:"auto_table_construction"`

	Events struct {
	} `json:"events"`
}

func validateJSONAgentOptionsSet(rawJSON json.RawMessage) error {
	// while ValidateJSONAgentOptions validates an entire Agent Options payload,
	// this unexported function validates a single set of options. That is, in an
	// Agent Options payload, the top-level "config" key defines a set, and each
	// of the platform overrides defines other sets. They all have the same
	// validation rules.
	panic("unimplemented")
}

// ValidateYAMLAgentOptions validates the given raw YAML bytes as an Agent
// Options payload. It ensures that all fields are known and have valid values.
// The validation always uses the most recent Osquery version that is available
// at the time of the Fleet release.
func ValidateYAMLAgentOptions(rawYAML []byte) error {
	// TODO(mna): determine if this is necessary - `fleetctl apply` will marshal
	// the YAML to JSON so any invalid field will still be sent and validated in
	// the JSON validation.
	panic("unimplemented")
}
