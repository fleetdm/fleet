// Package spec contains functionality to parse "Fleet specs" yaml files
// (which are concatenated yaml files) that can be applied to a Fleet server.
package spec

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
)

var yamlSeparator = regexp.MustCompile(`(?m:^---[\t ]*)`)

var (
	envOverridesMu sync.RWMutex
	envOverrides   map[string]string
)

// SetEnvOverrides sets environment variable overrides that take precedence over
// os.LookupEnv during env expansion in GitOps file parsing. Pass nil to clear.
func SetEnvOverrides(overrides map[string]string) {
	envOverridesMu.Lock()
	defer envOverridesMu.Unlock()
	if overrides == nil {
		envOverrides = nil
		return
	}
	envOverrides = make(map[string]string, len(overrides))
	maps.Copy(envOverrides, overrides)
}

// lookupEnv checks env overrides first, then falls back to os.LookupEnv.
func lookupEnv(key string) (string, bool) {
	envOverridesMu.RLock()
	if envOverrides != nil {
		if v, ok := envOverrides[key]; ok {
			envOverridesMu.RUnlock()
			return v, true
		}
	}
	envOverridesMu.RUnlock()
	return os.LookupEnv(key)
}

// Group holds a set of "specs" that can be applied to a Fleet server.
type Group struct {
	Queries  []*fleet.QuerySpec
	Teams    []json.RawMessage
	Packs    []*fleet.PackSpec
	Labels   []*fleet.LabelSpec
	Policies []*fleet.PolicySpec
	Software []*fleet.SoftwarePackageSpec
	// This needs to be interface{} to allow for the patch logic. Otherwise we send a request that looks to the
	// server like the user explicitly set the zero values.
	AppConfig              interface{}
	EnrollSecret           *fleet.EnrollSecretSpec
	UsersRoles             *fleet.UsersRoleSpec
	TeamsDryRunAssumptions *fleet.TeamSpecsDryRunAssumptions
	CertificateAuthorities *fleet.GroupedCertificateAuthorities
}

// Metadata holds the metadata for a single YAML section/item.
type Metadata struct {
	Kind    string          `json:"kind"`
	Version string          `json:"apiVersion"`
	Spec    json.RawMessage `json:"spec"`
}

// rewriteNewToOldKeys uses RewriteDeprecatedKeys to rewrite new (renameto)
// key names back to old (json tag) names so that structs can be unmarshaled
// correctly when input uses the new key names.
func rewriteNewToOldKeys(raw json.RawMessage, target any) (json.RawMessage, map[string]string, error) {
	rules := endpointer.ExtractAliasRules(target)
	if len(rules) == 0 {
		return raw, nil, nil
	}
	result, deprecatedKeysMap, err := endpointer.RewriteDeprecatedKeys(raw, rules)
	if err != nil {
		return nil, nil, err // fall back to original on error
	}
	return result, deprecatedKeysMap, nil
}

type GroupFromBytesOpts struct {
	LogFn func(format string, args ...any)
}

// GroupFromBytes parses a Group from concatenated YAML specs.
func GroupFromBytes(b []byte, options ...GroupFromBytesOpts) (*Group, error) {
	// Get optional logger.
	var logFn func(format string, args ...any)
	if len(options) > 0 {
		logFn = options[0].LogFn
	}

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

		var deprecatedKeysMap map[string]string
		switch kind {
		case fleet.QueryKind, fleet.ReportKind:
			if logFn != nil && kind == fleet.QueryKind {
				logFn("[!] `kind: query` is deprecated, please use `kind: report` instead.\n")
			}
			var err error
			s.Spec, deprecatedKeysMap, err = rewriteNewToOldKeys(s.Spec, fleet.QuerySpec{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
			var querySpec *fleet.QuerySpec
			if err := yaml.Unmarshal(s.Spec, &querySpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Queries = append(specs.Queries, querySpec)

		case fleet.PackKind:
			var err error
			s.Spec, deprecatedKeysMap, err = rewriteNewToOldKeys(s.Spec, fleet.PackSpec{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
			var packSpec *fleet.PackSpec
			if err := yaml.Unmarshal(s.Spec, &packSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Packs = append(specs.Packs, packSpec)

		case fleet.LabelKind:
			var err error
			s.Spec, deprecatedKeysMap, err = rewriteNewToOldKeys(s.Spec, fleet.LabelSpec{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
			var labelSpec *fleet.LabelSpec
			if err := yaml.Unmarshal(s.Spec, &labelSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			// Distinguish between hosts key omitted (nil, preserve membership)
			// and hosts key present with null value (clear all hosts). Both
			// unmarshal to nil, so check the raw YAML for key presence.
			if labelSpec.Hosts == nil {
				if hostsKeyPresent(s.Spec) {
					labelSpec.Hosts = []string{}
				}
			}
			specs.Labels = append(specs.Labels, labelSpec)

		case fleet.PolicyKind:
			var err error
			s.Spec, deprecatedKeysMap, err = rewriteNewToOldKeys(s.Spec, fleet.PolicySpec{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
			var policySpec *fleet.PolicySpec
			if err := yaml.Unmarshal(s.Spec, &policySpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.Policies = append(specs.Policies, policySpec)

		case fleet.AppConfigKind:
			var err error
			s.Spec, deprecatedKeysMap, err = rewriteNewToOldKeys(s.Spec, fleet.AppConfig{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
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
			var err error
			s.Spec, deprecatedKeysMap, err = rewriteNewToOldKeys(s.Spec, fleet.UsersRoleSpec{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
			var userRoleSpec *fleet.UsersRoleSpec
			if err := yaml.Unmarshal(s.Spec, &userRoleSpec); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			specs.UsersRoles = userRoleSpec

		case fleet.TeamKind, fleet.FleetKind:
			if logFn != nil && kind == fleet.TeamKind {
				logFn("[!] `kind: team` is deprecated, please use `kind: fleet` instead.\n")
			}
			// unmarshal to a raw map as we don't want to strip away unknown/invalid
			// fields at this point - that validation is done in the apply spec/teams
			// endpoint so that it is enforced for both the API and the CLI.
			rawTeam := make(map[string]json.RawMessage)
			if err := yaml.Unmarshal(s.Spec, &rawTeam); err != nil {
				return nil, fmt.Errorf("unmarshaling %s spec: %w", kind, err)
			}
			teamRaw := rawTeam[kind]
			var err error
			teamRaw, deprecatedKeysMap, err = rewriteNewToOldKeys(teamRaw, fleet.TeamSpec{})
			if err != nil {
				return nil, fmt.Errorf("in %s spec: %w", kind, err)
			}
			specs.Teams = append(specs.Teams, teamRaw)

		default:
			return nil, fmt.Errorf("unknown kind %q", s.Kind)
		}

		if logFn != nil && len(deprecatedKeysMap) > 0 && logging.TopicEnabled(logging.DeprecatedFieldTopic) {
			oldKeys := make([]string, 0, len(deprecatedKeysMap))
			for oldKey := range deprecatedKeysMap {
				oldKeys = append(oldKeys, oldKey)
			}
			sort.Strings(oldKeys)
			for _, oldKey := range oldKeys {
				logFn(fmt.Sprintf("[!] In %s: `%s` is deprecated, please use `%s` instead.\n", kind, oldKey, deprecatedKeysMap[oldKey]))
			}
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

// secretHandling defines how to handle FLEET_SECRET_ variables
type secretHandling int

const (
	// secretsReject returns an error if FLEET_SECRET_ variables are found
	secretsReject secretHandling = iota
	// secretsIgnore leaves FLEET_SECRET_ variables as-is (for server to handle)
	secretsIgnore
	// secretsExpand expands FLEET_SECRET_ variables (for client-side validation only)
	secretsExpand
)

func ExpandEnv(s string) (string, error) {
	out, err := expandEnv(s, secretsReject)
	return out, err
}

// expandEnv expands environment variables for a gitops file.
// $ can be escaped with a backslash, e.g. \$VAR
// \$ can be escaped with another backslash, etc., e.g. \\\$VAR
// $FLEET_VAR_XXX will not be expanded. These variables are expanded on the server.
// The secretMode parameter controls how $FLEET_SECRET_XXX variables are handled.
func expandEnv(s string, secretMode secretHandling) (string, error) {
	// Generate a random escaping prefix that doesn't exist in s.
	var preventEscapingPrefix string
	for {
		preventEscapingPrefix = "PREVENT_ESCAPING_" + generateRandomString(8)
		if !strings.Contains(s, preventEscapingPrefix) {
			break
		}
	}

	s = escapeString(s, preventEscapingPrefix)
	exclusionZones := getExclusionZones(s)
	trimmed := strings.TrimSpace(s)
	documentIsXML := strings.HasPrefix(trimmed, "<") // We need to be more aggressive here, to also escape XML in Windows profiles which does not begin with <?xml
	documentIsJSON := strings.HasPrefix(trimmed, "{")

	escapeValue := func(value string, env string) (string, error) {
		switch {
		case documentIsJSON:
			// Escape JSON special characters so the value is safe to embed inside
			// a JSON string literal (Apple DDM declarations, Android profiles).
			return jsonEscapeString(value), nil
		case documentIsXML:
			var b strings.Builder
			if xmlErr := xml.EscapeText(&b, []byte(value)); xmlErr != nil {
				return "", fmt.Errorf("failed to XML escape fleet secret %s", env)
			}
			return b.String(), nil
		default:
			return value, nil
		}
	}

	var err *multierror.Error
	s = fleet.MaybeExpand(s, func(env string, startPos, endPos int) (string, bool) {
		switch {
		case strings.HasPrefix(env, preventEscapingPrefix):
			return "$" + strings.TrimPrefix(env, preventEscapingPrefix), true
		case strings.HasPrefix(strings.ToUpper(env), fleet.ServerVarPrefix):
			// Don't expand fleet vars -- they will be expanded on the server
			return "", false
		case strings.HasPrefix(env, fleet.ServerSecretPrefix):
			switch secretMode {
			case secretsExpand:
				// Expand secrets for client-side validation
				v, ok := lookupEnv(env)
				if ok {
					escaped, escErr := escapeValue(v, env)
					if escErr != nil {
						err = multierror.Append(err, escErr)
						return "", false
					}
					return escaped, true
				}
				// If secret not found, leave as-is for server to handle
				return "", false
			case secretsReject:
				err = multierror.Append(err, fmt.Errorf("environment variables with %q prefix are only allowed in profiles and scripts: %q",
					fleet.ServerSecretPrefix, env))
				return "", false
			default:
				// Leave as-is for server to handle
				return "", false
			}
		}

		// Don't expand fleet vars if they are inside an 'exclusion' zone,
		// i.e. 'description' or 'resolution'....
		for _, z := range exclusionZones {
			if startPos >= z[0] && endPos <= z[1] {
				return "", false
			}
		}

		v, ok := lookupEnv(env)
		if !ok {
			err = multierror.Append(err, fmt.Errorf("environment variable %q not set", env))
			return "", false
		}
		escaped, escErr := escapeValue(v, env)
		if escErr != nil {
			err = multierror.Append(err, escErr)
			return "", false
		}
		return escaped, true
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

func ExpandEnvBytesIgnoreSecrets(b []byte) ([]byte, error) {
	s, err := expandEnv(string(b), secretsIgnore)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// ExpandEnvBytesIncludingSecrets expands environment variables including FLEET_SECRET_ variables.
// This should only be used for client-side validation where the actual secrets are needed temporarily.
// The expanded secrets are never sent to the server.
// Missing FLEET_SECRET_ variables do not fail the method; they are just not expanded.
func ExpandEnvBytesIncludingSecrets(b []byte) ([]byte, error) {
	s, err := expandEnv(string(b), secretsExpand)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// LookupEnvSecrets only looks up FLEET_SECRET_XXX environment variables.
// This is used for finding secrets in profiles and scripts. The original string is not modified.
// A map of secret names to raw (unescaped) values is updated.
// XML escaping is intentionally NOT done here — it is handled server-side during
// secret expansion (see expandEmbeddedSecrets in secret_variables.go).
func LookupEnvSecrets(s string, secretsMap map[string]string) error {
	if secretsMap == nil {
		return errors.New("secretsMap cannot be nil")
	}

	var err *multierror.Error
	_ = fleet.MaybeExpand(s, func(env string, startPos, endPos int) (string, bool) {
		if strings.HasPrefix(env, fleet.ServerSecretPrefix) {
			// lookup the secret and save it, but don't replace
			v, ok := lookupEnv(env)
			if !ok {
				err = multierror.Append(err, fmt.Errorf("environment variable %q not set", env))
				return "", false
			}

			secretsMap[env] = v
		}
		return "", false
	})
	if err != nil {
		return err
	}
	return nil
}

// jsonEscapeString returns the JSON-escaped interior of a string value
// (without surrounding quotes), suitable for embedding inside a JSON string.
func jsonEscapeString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// json.Marshal on a string should never fail, but return the
		// original string as a fallback.
		return s
	}
	return string(b[1 : len(b)-1])
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

// getExclusionZones returns which positions inside 's' should be
// excluded from variable interpolation.
func getExclusionZones(s string) [][2]int {
	// We need a different pattern per section because
	// the delimiting end pattern ((?:^\s+\w+:|\z)) includes the next
	// section token, meaning the matching logic won't work in case
	// we have a 'resolution:' followed by a 'description:' or
	// vice versa, and we try using something like (?:resolution:|description:)
	toExclude := []string{
		"resolution",
		"description",
	}
	patterns := make([]*regexp.Regexp, 0, len(toExclude))
	for _, e := range toExclude {
		pattern := fmt.Sprintf(`(?m)^\s*(?:%s:)(.|[\r\n])*?(?:^\s+\w+:|\z)`, e)
		patterns = append(patterns, regexp.MustCompile(pattern))
	}

	var zones [][2]int
	for _, pattern := range patterns {
		result := pattern.FindAllStringIndex(s, -1)
		for _, r := range result {
			zones = append(zones, [2]int{r[0], r[1]})
		}
	}
	return zones
}

// hostsKeyPresent checks if the "hosts" key is present in raw spec bytes.
// The input may be YAML or JSON; YAML is converted to JSON before inspection.
// Used to distinguish between an omitted hosts key (nil, no-op) and an
// explicit hosts key with null value (should clear hosts).
func hostsKeyPresent(rawBytes []byte) bool {
	jsonBytes, err := yaml.YAMLToJSON(rawBytes)
	if err != nil {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return false
	}
	_, ok := raw["hosts"]
	return ok
}
