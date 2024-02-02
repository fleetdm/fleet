package spec

import (
	"encoding/json"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"os"
	"path/filepath"
	"unicode"
)

type BaseItem struct {
	Path *string `json:"path"`
}

type Controls struct {
	BaseItem
	MacOSUpdates   interface{} `json:"macos_updates"`
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
	TeamName     *string
	AgentOptions *json.RawMessage
	Controls     Controls
	OrgSettings  map[string]interface{}
	Policies     []*fleet.PolicySpec
	Queries      []*fleet.QuerySpec
}

// GitOpsFromBytes parses a GitOps yaml file.
func GitOpsFromBytes(b []byte, baseDir string) (*GitOps, error) {
	var top map[string]json.RawMessage
	b = []byte(os.ExpandEnv(string(b))) // replace $var and ${var} with env values
	if err := yaml.Unmarshal(b, &top); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %w: \n", err)
	}

	var errors []string
	result := &GitOps{}

	// TODO: Check if any additional unknown top-level fields are present. If so, return an error.

	// Figure out if this is an org or team settings file
	team, teamOk := top["name"]
	_, teamSettingsOk := top["team_settings"]
	orgSettingsRaw, orgOk := top["org_settings"]
	if orgOk {
		if teamOk || teamSettingsOk {
			errors = append(errors, "'org_settings' cannot be used with 'name' or 'team_settings'")
		} else {
			errors = parseOrgSettings(orgSettingsRaw, result, baseDir, errors)
		}
	} else if teamOk && teamSettingsOk {
		teamName := string(team)
		if !isASCII(teamName) {
			errors = append(errors, fmt.Sprintf("team name must be in ASCII: %s", teamName))
		} else {
			result.TeamName = &teamName
		}
	} else {
		errors = append(errors, "either 'org_settings' or 'name' and 'team_settings' must be present")
	}

	// Validate the required top level options
	errors = parseControls(top, result, baseDir, errors)
	errors = parseAgentOptions(top, result, baseDir, errors)
	errors = parsePolicies(top, result, baseDir, errors)
	errors = parseQueries(top, result, baseDir, errors)
	if len(errors) > 0 {
		err := "\n"
		for _, e := range errors {
			err += e + "\n"
		}
		return nil, fmt.Errorf("YAML processing errors: %s", err)
	}

	return result, nil
}

func parseOrgSettings(raw json.RawMessage, result *GitOps, baseDir string, errors []string) []string {
	var orgSettingsTop BaseItem
	if err := yaml.Unmarshal(raw, &orgSettingsTop); err != nil {
		errors = append(errors, fmt.Sprintf("failed to unmarshal org_settings: %v", err))
	} else {
		noError := true
		if orgSettingsTop.Path == nil {
			result.AgentOptions = &raw
		} else {
			fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *orgSettingsTop.Path))
			if err != nil {
				noError = false
				errors = append(errors, fmt.Sprintf("failed to read org settings file %s: %v", *orgSettingsTop.Path, err))
			} else {
				fileBytes = []byte(os.ExpandEnv(string(fileBytes)))
				var pathOrgSettings BaseItem
				if err := yaml.Unmarshal(fileBytes, &pathOrgSettings); err != nil {
					noError = false
					errors = append(errors, fmt.Sprintf("failed to unmarshal org settings file %s: %v", *orgSettingsTop.Path, err))
				} else {
					if pathOrgSettings.Path != nil {
						noError = false
						errors = append(
							errors,
							fmt.Sprintf("nested paths are not supported: %s in %s", *pathOrgSettings.Path, *orgSettingsTop.Path),
						)
					} else {
						raw = fileBytes
					}
				}
			}
		}
		if noError {
			if err = yaml.Unmarshal(raw, &result.OrgSettings); err != nil {
				errors = append(errors, fmt.Sprintf("failed to unmarshal org settings: %v", err))
			} else {
				errors = parseSecrets(result, errors)
			}
			// TODO: Validate that integrations.(jira|zendesk)[].api_token is not empty or fleet.MaskedPassword
		}
	}
	return errors
}

// TODO: Support this method for teams.
func parseSecrets(result *GitOps, errors []string) []string {
	rawSecrets, ok := result.OrgSettings["secrets"]
	if !ok {
		errors = append(errors, "'org_settings.secrets' is required")
	} else {
		result.OrgSettings["secrets"] = []*fleet.EnrollSecret{}
		if rawSecrets != nil {
			secrets, ok := rawSecrets.([]interface{})
			if !ok {
				errors = append(errors, "'org_settings.secrets' must be a list of secret items")
			} else {
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
						errors = append(
							errors, "each item in 'org_settings.secrets' must have a 'secret' key containing an ASCII string value",
						)
					} else {
						result.OrgSettings["secrets"] = append(
							result.OrgSettings["secrets"].([]*fleet.EnrollSecret), &fleet.EnrollSecret{Secret: secret},
						)
					}
				}
			}
		}
	}
	return errors
}

func parseAgentOptions(top map[string]json.RawMessage, result *GitOps, baseDir string, errors []string) []string {
	agentOptionsRaw, ok := top["agent_options"]
	if !ok {
		errors = append(errors, "'agent_options' is required")
	} else {
		var agentOptionsTop BaseItem
		if err := yaml.Unmarshal(agentOptionsRaw, &agentOptionsTop); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unmarshal agent_options: %v", err))
		} else {
			if agentOptionsTop.Path == nil {
				result.AgentOptions = &agentOptionsRaw
			} else {
				fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *agentOptionsTop.Path))
				if err != nil {
					errors = append(errors, fmt.Sprintf("failed to read agent options file %s: %v", *agentOptionsTop.Path, err))
				} else {
					fileBytes = []byte(os.ExpandEnv(string(fileBytes)))
					var pathAgentOptions BaseItem
					if err := yaml.Unmarshal(fileBytes, &pathAgentOptions); err != nil {
						errors = append(errors, fmt.Sprintf("failed to unmarshal agent options file %s: %v", *agentOptionsTop.Path, err))
					} else {
						if pathAgentOptions.Path != nil {
							errors = append(
								errors,
								fmt.Sprintf("nested paths are not supported: %s in %s", *pathAgentOptions.Path, *agentOptionsTop.Path),
							)
						} else {
							var raw json.RawMessage
							if err := yaml.Unmarshal(fileBytes, &raw); err != nil {
								errors = append(
									errors, fmt.Sprintf("failed to unmarshal agent options file %s: %v", *agentOptionsTop.Path, err),
								)
							} else {
								result.AgentOptions = &raw
							}
						}
					}
				}
			}
		}
	}
	return errors
}

func parseControls(top map[string]json.RawMessage, result *GitOps, baseDir string, errors []string) []string {
	controlsRaw, ok := top["controls"]
	if !ok {
		errors = append(errors, "'controls' is required")
	} else {
		var controlsTop Controls
		if err := yaml.Unmarshal(controlsRaw, &controlsTop); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unmarshal controls: %v", err))
		} else {
			if controlsTop.Path == nil {
				result.Controls = controlsTop
			} else {
				fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *controlsTop.Path))
				if err != nil {
					errors = append(errors, fmt.Sprintf("failed to read controls file %s: %v", *controlsTop.Path, err))
				} else {
					fileBytes = []byte(os.ExpandEnv(string(fileBytes)))
					var pathControls Controls
					if err := yaml.Unmarshal(fileBytes, &pathControls); err != nil {
						errors = append(errors, fmt.Sprintf("failed to unmarshal agent options file %s: %v", *controlsTop.Path, err))
					} else {
						if pathControls.Path != nil {
							errors = append(
								errors,
								fmt.Sprintf("nested paths are not supported: %s in %s", *pathControls.Path, *controlsTop.Path),
							)
						} else {
							result.Controls = pathControls
						}
					}
				}
			}
		}
	}
	return errors
}

func parsePolicies(top map[string]json.RawMessage, result *GitOps, baseDir string, errors []string) []string {
	policiesRaw, ok := top["policies"]
	if !ok {
		errors = append(errors, "'policies' key is required")
	} else {
		var policies []Policy
		if err := yaml.Unmarshal(policiesRaw, &policies); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unmarshal policies: %v", err))
		} else {
			for _, item := range policies {
				item := item
				if item.Path == nil {
					result.Policies = append(result.Policies, &item.PolicySpec)
				} else {
					fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *item.Path))
					if err != nil {
						errors = append(errors, fmt.Sprintf("failed to read policies file %s: %v", *item.Path, err))
					} else {
						fileBytes = []byte(os.ExpandEnv(string(fileBytes)))
						var pathPolicies []*Policy
						if err := yaml.Unmarshal(fileBytes, &pathPolicies); err != nil {
							errors = append(errors, fmt.Sprintf("failed to unmarshal policies file %s: %v", *item.Path, err))
						} else {
							for _, pp := range pathPolicies {
								pp := pp
								if pp != nil {
									if pp.Path != nil {
										errors = append(
											errors, fmt.Sprintf("nested paths are not supported: %s in %s", *pp.Path, *item.Path),
										)
									} else {
										result.Policies = append(result.Policies, &pp.PolicySpec)
									}
								}
							}
						}
					}
				}
			}
			// Make sure team name is correct
			for _, item := range result.Policies {
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
				errors = append(errors, fmt.Sprintf("duplicate policy names: %v", duplicates))
			}
		}
	}
	return errors
}

func parseQueries(top map[string]json.RawMessage, result *GitOps, baseDir string, errors []string) []string {
	queriesRaw, ok := top["queries"]
	if !ok {
		errors = append(errors, "'queries' key is required")
	} else {
		var queries []Query
		if err := yaml.Unmarshal(queriesRaw, &queries); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unmarshal queries: %v", err))
		} else {
			for _, item := range queries {
				item := item
				if item.Path == nil {
					result.Queries = append(result.Queries, &item.QuerySpec)
				} else {
					fileBytes, err := os.ReadFile(resolveApplyRelativePath(baseDir, *item.Path))
					if err != nil {
						errors = append(errors, fmt.Sprintf("failed to read queries file %s: %v", *item.Path, err))
					} else {
						fileBytes = []byte(os.ExpandEnv(string(fileBytes)))
						var pathQueries []*Query
						if err := yaml.Unmarshal(fileBytes, &pathQueries); err != nil {
							errors = append(errors, fmt.Sprintf("failed to unmarshal queries file %s: %v", *item.Path, err))
						} else {
							for _, pq := range pathQueries {
								pq := pq
								if pq != nil {
									if pq.Path != nil {
										errors = append(
											errors, fmt.Sprintf("nested paths are not supported: %s in %s", *pq.Path, *item.Path),
										)
									} else {
										result.Queries = append(result.Queries, &pq.QuerySpec)
									}
								}
							}
						}
					}
				}
			}
			for _, q := range result.Queries {
				// Make sure team name is correct
				if result.TeamName != nil {
					q.TeamName = *result.TeamName
				} else {
					q.TeamName = ""
				}
				// Don't use non-ASCII
				if !isASCII(q.Name) {
					errors = append(errors, fmt.Sprintf("query name must be in ASCII: %s", q.Name))
				}
			}
			duplicates := getDuplicateNames(
				result.Queries, func(q *fleet.QuerySpec) string {
					return q.Name
				},
			)
			if len(duplicates) > 0 {
				errors = append(errors, fmt.Sprintf("duplicate query names: %v", duplicates))
			}
		}
	}
	return errors
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
