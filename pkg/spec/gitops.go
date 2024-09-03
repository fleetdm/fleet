package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"unicode"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/text/unicode/norm"
)

type BaseItem struct {
	Path *string `json:"path"`
}

type Controls struct {
	BaseItem
	MacOSUpdates   interface{} `json:"macos_updates"`
	IOSUpdates     interface{} `json:"ios_updates"`
	IPadOSUpdates  interface{} `json:"ipados_updates"`
	MacOSSettings  interface{} `json:"macos_settings"`
	MacOSSetup     interface{} `json:"macos_setup"`
	MacOSMigration interface{} `json:"macos_migration"`

	WindowsUpdates              interface{} `json:"windows_updates"`
	WindowsSettings             interface{} `json:"windows_settings"`
	WindowsEnabledAndConfigured interface{} `json:"windows_enabled_and_configured"`

	EnableDiskEncryption interface{} `json:"enable_disk_encryption"`

	Scripts []BaseItem `json:"scripts"`
}

type Policy struct {
	BaseItem
	fleet.PolicySpec
}

type Query struct {
	BaseItem
	fleet.QuerySpec
}

type GitOps struct {
	TeamID       *uint
	TeamName     *string
	TeamSettings map[string]interface{}
	OrgSettings  map[string]interface{}
	AgentOptions *json.RawMessage
	Controls     Controls
	Policies     []*fleet.PolicySpec
	Queries      []*fleet.QuerySpec
	// Software is only allowed on teams, not on global config.
	Software GitOpsSoftware
}

type GitOpsSoftware struct {
	Packages     []*fleet.SoftwarePackageSpec
	AppStoreApps []*fleet.TeamSpecAppStoreApp
}

// GitOpsFromFile parses a GitOps yaml file.
func GitOpsFromFile(filePath, baseDir string, appConfig *fleet.EnrichedAppConfig) (*GitOps, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %w", filePath, err)
	}

	// Replace $var and ${var} with env values.
	b, err = ExpandEnvBytes(b)
	if err != nil {
		return nil, fmt.Errorf("failed to expand environment in file %s: %w", filePath, err)
	}

	var top map[string]json.RawMessage
	if err := yaml.Unmarshal(b, &top); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %w: \n", err)
	}

	var multiError *multierror.Error
	result := &GitOps{}

	topKeys := []string{"name", "team_settings", "org_settings", "agent_options", "controls", "policies", "queries", "software"}
	for k := range top {
		if !slices.Contains(topKeys, k) {
			multiError = multierror.Append(multiError, fmt.Errorf("unknown top-level field: %s", k))
		}
	}

	// Figure out if this is an org or team settings file
	teamRaw, teamOk := top["name"]
	teamSettingsRaw, teamSettingsOk := top["team_settings"]
	orgSettingsRaw, orgOk := top["org_settings"]
	if orgOk {
		if teamOk || teamSettingsOk {
			multiError = multierror.Append(multiError, errors.New("'org_settings' cannot be used with 'name', 'team_settings'"))
		} else {
			multiError = parseOrgSettings(orgSettingsRaw, result, baseDir, multiError)
		}
	} else if teamOk && teamSettingsOk {
		multiError = parseName(teamRaw, result, multiError)
		multiError = parseTeamSettings(teamSettingsRaw, result, baseDir, multiError)
	} else {
		multiError = multierror.Append(multiError, errors.New("either 'org_settings' or 'name' and 'team_settings' is required"))
	}

	// Validate the required top level options
	multiError = parseControls(top, result, baseDir, multiError)
	multiError = parseAgentOptions(top, result, baseDir, multiError)
	multiError = parsePolicies(top, result, baseDir, multiError)
	multiError = parseQueries(top, result, baseDir, multiError)

	if appConfig != nil && appConfig.License.IsPremium() {
		multiError = parseSoftware(top, result, baseDir, multiError)
	}

	return result, multiError.ErrorOrNil()
}

func parseName(raw json.RawMessage, result *GitOps, multiError *multierror.Error) *multierror.Error {
	if err := json.Unmarshal(raw, &result.TeamName); err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to unmarshal name: %v", err))
	}
	if result.TeamName == nil || *result.TeamName == "" {
		return multierror.Append(multiError, errors.New("team 'name' is required"))
	}
	// Normalize team name for full Unicode support, so that we can assume team names are unique going forward
	normalized := norm.NFC.String(*result.TeamName)
	result.TeamName = &normalized
	return multiError
}

func parseOrgSettings(raw json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	var orgSettingsTop BaseItem
	if err := json.Unmarshal(raw, &orgSettingsTop); err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to unmarshal org_settings: %v", err))
	}
	noError := true
	if orgSettingsTop.Path != nil {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *orgSettingsTop.Path))
		if err != nil {
			noError = false
			multiError = multierror.Append(multiError, fmt.Errorf("failed to read org settings file %s: %v", *orgSettingsTop.Path, err))
		} else {
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				noError = false
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *orgSettingsTop.Path, err),
				)
			} else {
				var pathOrgSettings BaseItem
				if err := yaml.Unmarshal(fileBytes, &pathOrgSettings); err != nil {
					noError = false
					multiError = multierror.Append(
						multiError, fmt.Errorf("failed to unmarshal org settings file %s: %v", *orgSettingsTop.Path, err),
					)
				} else {
					if pathOrgSettings.Path != nil {
						noError = false
						multiError = multierror.Append(
							multiError,
							fmt.Errorf("nested paths are not supported: %s in %s", *pathOrgSettings.Path, *orgSettingsTop.Path),
						)
					} else {
						raw = fileBytes
					}
				}
			}
		}
	}
	if noError {
		if err := yaml.Unmarshal(raw, &result.OrgSettings); err != nil {
			// This error is currently unreachable because we know the file is valid YAML when we checked for nested path
			multiError = multierror.Append(multiError, fmt.Errorf("failed to unmarshal org settings: %v", err))
		} else {
			multiError = parseSecrets(result, multiError)
		}
		// TODO: Validate that integrations.(jira|zendesk)[].api_token is not empty or fleet.MaskedPassword
	}
	return multiError
}

func parseTeamSettings(raw json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	var teamSettingsTop BaseItem
	if err := json.Unmarshal(raw, &teamSettingsTop); err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to unmarshal team_settings: %v", err))
	}
	noError := true
	if teamSettingsTop.Path != nil {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *teamSettingsTop.Path))
		if err != nil {
			noError = false
			multiError = multierror.Append(multiError, fmt.Errorf("failed to read team settings file %s: %v", *teamSettingsTop.Path, err))
		} else {
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				noError = false
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *teamSettingsTop.Path, err),
				)
			} else {
				var pathTeamSettings BaseItem
				if err := yaml.Unmarshal(fileBytes, &pathTeamSettings); err != nil {
					noError = false
					multiError = multierror.Append(
						multiError, fmt.Errorf("failed to unmarshal team settings file %s: %v", *teamSettingsTop.Path, err),
					)
				} else {
					if pathTeamSettings.Path != nil {
						noError = false
						multiError = multierror.Append(
							multiError,
							fmt.Errorf("nested paths are not supported: %s in %s", *pathTeamSettings.Path, *teamSettingsTop.Path),
						)
					} else {
						raw = fileBytes
					}
				}
			}
		}
	}
	if noError {
		if err := yaml.Unmarshal(raw, &result.TeamSettings); err != nil {
			// This error is currently unreachable because we know the file is valid YAML when we checked for nested path
			multiError = multierror.Append(multiError, fmt.Errorf("failed to unmarshal team settings: %v", err))
		} else {
			multiError = parseSecrets(result, multiError)
		}
	}
	return multiError
}

func parseSecrets(result *GitOps, multiError *multierror.Error) *multierror.Error {
	var rawSecrets interface{}
	var ok bool
	if result.TeamName == nil {
		rawSecrets, ok = result.OrgSettings["secrets"]
		if !ok {
			return multierror.Append(multiError, errors.New("'org_settings.secrets' is required"))
		}
	} else {
		rawSecrets, ok = result.TeamSettings["secrets"]
		if !ok {
			return multierror.Append(multiError, errors.New("'team_settings.secrets' is required"))
		}
	}
	// When secrets slice is empty, all secrets are removed.
	enrollSecrets := make([]*fleet.EnrollSecret, 0)
	if rawSecrets != nil {
		secrets, ok := rawSecrets.([]interface{})
		if !ok {
			return multierror.Append(multiError, errors.New("'secrets' must be a list of secret items"))
		}
		for _, enrollSecret := range secrets {
			var secret string
			var secretInterface interface{}
			secretMap, ok := enrollSecret.(map[string]interface{})
			if ok {
				secretInterface, ok = secretMap["secret"]
			}
			if ok {
				secret, ok = secretInterface.(string)
			}
			if !ok || secret == "" {
				multiError = multierror.Append(
					multiError, errors.New("each item in 'secrets' must have a 'secret' key containing an ASCII string value"),
				)
				break
			}
			enrollSecrets = append(
				enrollSecrets, &fleet.EnrollSecret{Secret: secret},
			)
		}
	}
	if result.TeamName == nil {
		result.OrgSettings["secrets"] = enrollSecrets
	} else {
		result.TeamSettings["secrets"] = enrollSecrets
	}
	return multiError
}

func parseAgentOptions(top map[string]json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	agentOptionsRaw, ok := top["agent_options"]
	if !ok {
		return multierror.Append(multiError, errors.New("'agent_options' is required"))
	}
	var agentOptionsTop BaseItem
	if err := json.Unmarshal(agentOptionsRaw, &agentOptionsTop); err != nil {
		multiError = multierror.Append(multiError, fmt.Errorf("failed to unmarshal agent_options: %v", err))
	} else {
		if agentOptionsTop.Path == nil {
			result.AgentOptions = &agentOptionsRaw
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *agentOptionsTop.Path))
			if err != nil {
				return multierror.Append(multiError, fmt.Errorf("failed to read agent options file %s: %v", *agentOptionsTop.Path, err))
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *agentOptionsTop.Path, err),
				)
			} else {
				var pathAgentOptions BaseItem
				if err := yaml.Unmarshal(fileBytes, &pathAgentOptions); err != nil {
					return multierror.Append(
						multiError, fmt.Errorf("failed to unmarshal agent options file %s: %v", *agentOptionsTop.Path, err),
					)
				}
				if pathAgentOptions.Path != nil {
					return multierror.Append(
						multiError,
						fmt.Errorf("nested paths are not supported: %s in %s", *pathAgentOptions.Path, *agentOptionsTop.Path),
					)
				}
				var raw json.RawMessage
				if err := yaml.Unmarshal(fileBytes, &raw); err != nil {
					// This error is currently unreachable because we know the file is valid YAML when we checked for nested path
					return multierror.Append(
						multiError, fmt.Errorf("failed to unmarshal agent options file %s: %v", *agentOptionsTop.Path, err),
					)
				}
				result.AgentOptions = &raw
			}
		}
	}
	return multiError
}

func parseControls(top map[string]json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	controlsRaw, ok := top["controls"]
	if !ok {
		return multierror.Append(multiError, errors.New("'controls' is required"))
	}
	var controlsTop Controls
	if err := json.Unmarshal(controlsRaw, &controlsTop); err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to unmarshal controls: %v", err))
	}
	if controlsTop.Path == nil {
		result.Controls = controlsTop
	} else {
		fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *controlsTop.Path))
		if err != nil {
			return multierror.Append(multiError, fmt.Errorf("failed to read controls file %s: %v", *controlsTop.Path, err))
		}
		// Replace $var and ${var} with env values.
		fileBytes, err = ExpandEnvBytes(fileBytes)
		if err != nil {
			multiError = multierror.Append(
				multiError, fmt.Errorf("failed to expand environment in file %s: %v", *controlsTop.Path, err),
			)
		} else {
			var pathControls Controls
			if err := yaml.Unmarshal(fileBytes, &pathControls); err != nil {
				return multierror.Append(multiError, fmt.Errorf("failed to unmarshal controls file %s: %v", *controlsTop.Path, err))
			}
			if pathControls.Path != nil {
				return multierror.Append(
					multiError,
					fmt.Errorf("nested paths are not supported: %s in %s", *pathControls.Path, *controlsTop.Path),
				)
			}
			result.Controls = pathControls
		}
	}
	return multiError
}

func parsePolicies(top map[string]json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	policiesRaw, ok := top["policies"]
	if !ok {
		return multierror.Append(multiError, errors.New("'policies' key is required"))
	}
	var policies []Policy
	if err := json.Unmarshal(policiesRaw, &policies); err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to unmarshal policies: %v", err))
	}
	for _, item := range policies {
		item := item
		if item.Path == nil {
			result.Policies = append(result.Policies, &item.PolicySpec)
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *item.Path))
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read policies file %s: %v", *item.Path, err))
				continue
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *item.Path, err),
				)
			} else {
				var pathPolicies []*Policy
				if err := yaml.Unmarshal(fileBytes, &pathPolicies); err != nil {
					multiError = multierror.Append(multiError, fmt.Errorf("failed to unmarshal policies file %s: %v", *item.Path, err))
					continue
				}
				for _, pp := range pathPolicies {
					pp := pp
					if pp != nil {
						if pp.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pp.Path, *item.Path),
							)
						} else {
							result.Policies = append(result.Policies, &pp.PolicySpec)
						}
					}
				}
			}
		}
	}
	// Make sure team name is correct, and do additional validation
	for _, item := range result.Policies {
		if item.Name == "" {
			multiError = multierror.Append(multiError, errors.New("policy name is required for each policy"))
		} else {
			item.Name = norm.NFC.String(item.Name)
		}
		if item.Query == "" {
			multiError = multierror.Append(multiError, errors.New("policy query is required for each policy"))
		}
		if result.TeamName != nil {
			item.Team = *result.TeamName
		} else {
			item.Team = ""
		}
	}
	duplicates := getDuplicateNames(
		result.Policies, func(p *fleet.PolicySpec) string {
			return p.Name
		},
	)
	if len(duplicates) > 0 {
		multiError = multierror.Append(multiError, fmt.Errorf("duplicate policy names: %v", duplicates))
	}
	return multiError
}

func parseQueries(top map[string]json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	queriesRaw, ok := top["queries"]
	if !ok {
		return multierror.Append(multiError, errors.New("'queries' key is required"))
	}
	var queries []Query
	if err := json.Unmarshal(queriesRaw, &queries); err != nil {
		return multierror.Append(multiError, fmt.Errorf("failed to unmarshal queries: %v", err))
	}
	for _, item := range queries {
		item := item
		if item.Path == nil {
			result.Queries = append(result.Queries, &item.QuerySpec)
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *item.Path))
			if err != nil {
				multiError = multierror.Append(multiError, fmt.Errorf("failed to read queries file %s: %v", *item.Path, err))
				continue
			}
			// Replace $var and ${var} with env values.
			fileBytes, err = ExpandEnvBytes(fileBytes)
			if err != nil {
				multiError = multierror.Append(
					multiError, fmt.Errorf("failed to expand environment in file %s: %v", *item.Path, err),
				)
			} else {
				var pathQueries []*Query
				if err := yaml.Unmarshal(fileBytes, &pathQueries); err != nil {
					multiError = multierror.Append(multiError, fmt.Errorf("failed to unmarshal queries file %s: %v", *item.Path, err))
					continue
				}
				for _, pq := range pathQueries {
					pq := pq
					if pq != nil {
						if pq.Path != nil {
							multiError = multierror.Append(
								multiError, fmt.Errorf("nested paths are not supported: %s in %s", *pq.Path, *item.Path),
							)
						} else {
							result.Queries = append(result.Queries, &pq.QuerySpec)
						}
					}
				}
			}
		}
	}
	// Make sure team name is correct and do additional validation
	for _, q := range result.Queries {
		if q.Name == "" {
			multiError = multierror.Append(multiError, errors.New("query name is required for each query"))
		}
		if q.Query == "" {
			multiError = multierror.Append(multiError, errors.New("query SQL query is required for each query"))
		}
		// Don't use non-ASCII
		if !isASCII(q.Name) {
			multiError = multierror.Append(multiError, fmt.Errorf("query name must be in ASCII: %s", q.Name))
		}
		if result.TeamName != nil {
			q.TeamName = *result.TeamName
		} else {
			q.TeamName = ""
		}
	}
	duplicates := getDuplicateNames(
		result.Queries, func(q *fleet.QuerySpec) string {
			return q.Name
		},
	)
	if len(duplicates) > 0 {
		multiError = multierror.Append(multiError, fmt.Errorf("duplicate query names: %v", duplicates))
	}
	return multiError
}

func parseSoftware(top map[string]json.RawMessage, result *GitOps, baseDir string, multiError *multierror.Error) *multierror.Error {
	softwareRaw, ok := top["software"]
	if !ok {
		return multierror.Append(multiError, errors.New("'software' is required"))
	}
	var software fleet.SoftwareSpec
	if len(softwareRaw) > 0 {
		if err := json.Unmarshal(softwareRaw, &software); err != nil {
			var typeErr *json.UnmarshalTypeError
			if errors.As(err, &typeErr) {
				return multierror.Append(multiError, fmt.Errorf("Couldn't edit software. \"%s\" must be a %s", typeErr.Field, typeErr.Type.String()))
			}
			return multierror.Append(multiError, fmt.Errorf("failed to unmarshall softwarespec: %v", err))
		}
	}
	if software.AppStoreApps.Set {
		for _, item := range software.AppStoreApps.Value {
			item := item
			if item.AppStoreID == "" {
				multiError = multierror.Append(multiError, errors.New("software app store id required"))
				continue
			}
			result.Software.AppStoreApps = append(result.Software.AppStoreApps, &item)
		}
	}
	if software.Packages.Set {
		for _, item := range software.Packages.Value {
			item := item
			if item.URL == "" {
				multiError = multierror.Append(multiError, errors.New("software URL is required"))
				continue
			}
			result.Software.Packages = append(result.Software.Packages, &item)
		}
	}

	return multiError
}

func getDuplicateNames[T any](slice []T, getComparableString func(T) string) []string {
	// We are using the allKeys map as a set here. True means the item is a duplicate.
	allKeys := make(map[string]bool)
	var duplicates []string
	for _, item := range slice {
		name := getComparableString(item)
		if isDuplicate, exists := allKeys[name]; exists {
			// If this name hasn't already been marked as a duplicate.
			if !isDuplicate {
				duplicates = append(duplicates, name)
			}
			allKeys[name] = true
		} else {
			allKeys[name] = false
		}
	}
	return duplicates
}

func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// resolves the paths to an absolute path relative to the baseDir, which should
// be the path of the YAML file where the relative paths were specified. If the
// path is already absolute, it is left untouched.
func resolveApplyRelativePath(baseDir string, path string) string {
	if baseDir == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
