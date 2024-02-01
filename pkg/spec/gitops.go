package spec

import (
	"encoding/json"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"os"
	"path"
)

type Policy struct {
	Path *string `json:"path"`
	fleet.PolicySpec
}

type Query struct {
	Path *string `json:"path"`
	fleet.QuerySpec
}

type GitOps struct {
	IsTeam   bool
	TeamName string
	Policies []*fleet.PolicySpec
	Queries  []*fleet.QuerySpec
}

// GitOpsFromBytes parses a GitOps yaml file.
func GitOpsFromBytes(b []byte, baseDir string) (*GitOps, error) {
	// var top GitOpsTop
	var top map[string]json.RawMessage
	if err := yaml.Unmarshal(b, &top); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %w: \n", err)
	}

	var errors []string
	result := &GitOps{}

	// Figure out if this is an org or team settings file
	_, teamOk := top["name"]
	_, teamSettingsOk := top["team_settings"]
	_, orgOk := top["org_settings"]
	if orgOk {
		if teamOk || teamSettingsOk {
			errors = append(errors, "'org_settings' cannot be used with 'name' or 'team_settings'")
		}
		// } else if teamOk && teamSettingsOk {
	} else {
		errors = append(errors, "either 'org_settings' or 'name' and 'team_settings' must be present")
	}

	// Validate the required top level options
	_, ok := top["agent_options"]
	if !ok {
		errors = append(errors, "'agent_options' is required")
	}
	_, ok = top["controls"]
	if !ok {
		errors = append(errors, "'controls' is required")
	}
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
					fileBytes, err := os.ReadFile(path.Join(baseDir, *item.Path))
					if err != nil {
						errors = append(errors, fmt.Sprintf("failed to read policies file %s: %v", *item.Path, err))
					} else {
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
				if result.IsTeam {
					item.Team = result.TeamName
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
					fileBytes, err := os.ReadFile(path.Join(baseDir, *item.Path))
					if err != nil {
						errors = append(errors, fmt.Sprintf("failed to read queries file %s: %v", *item.Path, err))
					} else {
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
			// Make sure team name is correct
			for _, q := range result.Queries {
				if result.IsTeam {
					q.TeamName = result.TeamName
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
