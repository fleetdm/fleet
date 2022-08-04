// Package spec contains functionality to parse "Fleet specs" yaml files
// (which are concatenated yaml files) that can be applied to a Fleet server.
package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
)

var yamlSeparator = regexp.MustCompile(`(?m:^---[\t ]*)`)

// Group holds a set of "specs" that can be applied to a Fleet server.
type Group struct {
	Queries  []*fleet.QuerySpec
	Teams    []*fleet.TeamSpec
	Packs    []*fleet.PackSpec
	Labels   []*fleet.LabelSpec
	Policies []*fleet.PolicySpec
	// This needs to be interface{} to allow for the patch logic. Otherwise we send a request that looks to the
	// server like the user explicitly set the zero values.
	AppConfig    interface{}
	EnrollSecret *fleet.EnrollSecretSpec
	UsersRoles   *fleet.UsersRoleSpec
}

// Metadata holds the metadata for a single YAML section/item.
type Metadata struct {
	Kind    string          `json:"kind"`
	Version string          `json:"apiVersion"`
	Spec    json.RawMessage `json:"spec"`
}

// TeamSpec holds a spec to be applied to a team.
type TeamSpec struct {
	Team *fleet.TeamSpec `json:"team"`
}

// GroupFromBytes parses a Group from concatenated YAML specs.
func GroupFromBytes(b []byte) (*Group, error) {
	specs := &Group{}
	for _, specItem := range SplitYaml(string(b)) {
		var s Metadata
		if err := yaml.Unmarshal([]byte(specItem), &s); err != nil {
			return nil, err
		}

		if s.Spec == nil {
			return nil, fmt.Errorf("no spec field on %q document", s.Kind)
		}

		kind := strings.ToLower(s.Kind)

		switch kind {
		case fleet.QueryKind:
			var querySpec *fleet.QuerySpec
			if err := yaml.Unmarshal(s.Spec, &querySpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Queries = append(specs.Queries, querySpec)

		case fleet.PackKind:
			var packSpec *fleet.PackSpec
			if err := yaml.Unmarshal(s.Spec, &packSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Packs = append(specs.Packs, packSpec)

		case fleet.LabelKind:
			var labelSpec *fleet.LabelSpec
			if err := yaml.Unmarshal(s.Spec, &labelSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Labels = append(specs.Labels, labelSpec)

		case fleet.PolicyKind:
			var policySpec *fleet.PolicySpec
			if err := yaml.Unmarshal(s.Spec, &policySpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Policies = append(specs.Policies, policySpec)

		case fleet.AppConfigKind:
			if specs.AppConfig != nil {
				return nil, errors.New("config defined twice in the same file")
			}

			var appConfigSpec interface{}
			if err := yaml.Unmarshal(s.Spec, &appConfigSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.AppConfig = appConfigSpec

		case fleet.EnrollSecretKind:
			if specs.AppConfig != nil {
				return nil, errors.New("enroll_secret defined twice in the same file")
			}

			var enrollSecretSpec *fleet.EnrollSecretSpec
			if err := yaml.Unmarshal(s.Spec, &enrollSecretSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.EnrollSecret = enrollSecretSpec

		case fleet.UserRolesKind:
			var userRoleSpec *fleet.UsersRoleSpec
			if err := yaml.Unmarshal(s.Spec, &userRoleSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.UsersRoles = userRoleSpec

		case fleet.TeamKind:
			var teamSpec TeamSpec
			if err := yaml.Unmarshal(s.Spec, &teamSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Teams = append(specs.Teams, teamSpec.Team)

		default:
			return nil, fmt.Errorf("unknown kind %q", s.Kind)
		}
	}
	return specs, nil
}

// SplitYaml splits a text file into separate yaml documents divided by ---
func SplitYaml(in string) []string {
	var out []string
	for _, chunk := range yamlSeparator.Split(in, -1) {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		out = append(out, chunk)
	}
	return out
}
