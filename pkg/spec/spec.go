// Package spec contains functionality to parse "Fleet specs" yaml files
// (which are concatenated yaml files) that can be applied to a Fleet server.
package spec

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
)

var yamlSeparator = regexp.MustCompile(`(?m:^---[\t ]*)`)

// Group holds a set of "specs" that can be applied to a Fleet server.
type Group struct {
	Queries  []*fleet.QuerySpec
	Teams    []json.RawMessage
	Packs    []*fleet.PackSpec
	Labels   []*fleet.LabelSpec
	Policies []*fleet.PolicySpec
	// This needs to be interface{} to allow for the patch logic. Otherwise we send a request that looks to the
	// server like the user explicitly set the zero values.
	AppConfig              interface{}
	EnrollSecret           *fleet.EnrollSecretSpec
	UsersRoles             *fleet.UsersRoleSpec
	TeamsDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions
}

// Metadata holds the metadata for a single YAML section/item.
type Metadata struct {
	Kind    string          `json:"kind"`
	Version string          `json:"apiVersion"`
	Spec    json.RawMessage `json:"spec"`
}

// GroupFromBytes parses a Group from concatenated YAML specs.
func GroupFromBytes(b []byte) (*Group, error) {
	specs := &Group{}
	for _, specItem := range SplitYaml(string(b)) {
		var s Metadata
		if err := yaml.Unmarshal([]byte(specItem), &s); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec item %w: \n%s", err, specItem)
		}

		kind := strings.ToLower(s.Kind)

		if s.Spec == nil {
			if kind == "" {
				return nil, errors.New(`Missing required fields ("spec", "kind") on provided configuration.`)
			}
			return nil, fmt.Errorf(`Missing required fields ("spec") on provided %q configuration.`, s.Kind)
		}

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
			// unmarshal to a raw map as we don't want to strip away unknown/invalid
			// fields at this point - that validation is done in the apply spec/teams
			// endpoint so that it is enforced for both the API and the CLI.
			rawTeam := make(map[string]json.RawMessage)
			if err := yaml.Unmarshal(s.Spec, &rawTeam); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Teams = append(specs.Teams, rawTeam["team"])

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

func generateRandomString(sizeBytes int) string {
	b := make([]byte, sizeBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func ExpandEnv(s string) (string, error) {
	// Generate a random escaping prefix that doesn't exist in s.
	var preventEscapingPrefix string
	for {
		preventEscapingPrefix = "PREVENT_ESCAPING_" + generateRandomString(8)
		if !strings.Contains(s, preventEscapingPrefix) {
			break
		}
	}

	s = escapeString(s, preventEscapingPrefix)
	var err *multierror.Error
	s = os.Expand(s, func(env string) string {
		if strings.HasPrefix(env, preventEscapingPrefix) {
			return "$" + strings.TrimPrefix(env, preventEscapingPrefix)
		}
		v, ok := os.LookupEnv(env)
		if !ok {
			err = multierror.Append(err, fmt.Errorf("environment variable %q not set", env))
			return ""
		}
		return v
	})
	if err != nil {
		return "", err
	}
	return s, nil
}

func ExpandEnvBytes(b []byte) ([]byte, error) {
	s, err := ExpandEnv(string(b))
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

var escapePattern = regexp.MustCompile(`(\\+\$)`)

func escapeString(s string, preventEscapingPrefix string) string {
	return escapePattern.ReplaceAllStringFunc(s, func(match string) string {
		if len(match)%2 != 0 {
			return match
		}
		return strings.Repeat("\\", (len(match)/2)-1) + "$" + preventEscapingPrefix
	})
}
