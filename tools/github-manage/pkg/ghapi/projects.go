// Package ghapi provides GitHub project management functionality including
// project item management, field operations, and GraphQL API interactions.
package ghapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"fleetdm/gm/pkg/logger"
)

var Aliases = map[string]int{
	"mdm":             58,
	"g-mdm":           58,
	"draft":           67,
	"drafting":        67,
	"g-software":      70,
	"soft":            70,
	"g-orchestration": 71,
	"orch":            71,
}

// ProjectLabels maps project IDs to their corresponding label filters for the drafting project
var ProjectLabels = map[int]string{
	58: "#g-mdm",           // mdm project
	70: "#g-software",      // g-software project
	71: "#g-orchestration", // g-orchestration project
}

// ResolveProjectID resolves a project identifier (alias or numeric string) to a project ID.
func ResolveProjectID(identifier string) (int, error) {
	// First check if it's an alias
	if projectID, exists := Aliases[identifier]; exists {
		return projectID, nil
	}

	// Try to parse as a number
	if projectID, err := strconv.Atoi(identifier); err == nil {
		return projectID, nil
	}

	return 0, fmt.Errorf("invalid project identifier '%s'. Must be a numeric ID or one of these aliases: %v", identifier, getAliasKeys())
}

// getAliasKeys returns a slice of all available alias keys.
func getAliasKeys() []string {
	keys := make([]string, 0, len(Aliases))
	for k := range Aliases {
		keys = append(keys, k)
	}
	return keys
}

// ParseJSONtoProjectItems converts JSON data to a slice of ProjectItem structs.
func ParseJSONtoProjectItems(jsonData []byte, limit int) ([]ProjectItem, error) {
	var items ProjectItemsResponse
	err := json.Unmarshal(jsonData, &items)
	if err != nil {
		return nil, err
	}
	if items.TotalCount > limit {
		logger.Infof("Project item limit less than total, could be missing items - l: %d, t: %d", limit, items.TotalCount)
	}
	return items.Items, nil
}

// GetProjectItems retrieves project items for a specific project with a limit.
func GetProjectItems(projectID, limit int) ([]ProjectItem, error) {
	// Run the command to get project items
	results, err := RunCommandAndReturnOutput(fmt.Sprintf("gh project item-list --owner fleetdm --format json --limit %d %d", limit, projectID))
	if err != nil {
		return nil, err
	}
	items, err := ParseJSONtoProjectItems(results, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetProjectFields retrieves all fields for a specific project.
func GetProjectFields(projectID int) (map[string]ProjectField, error) {
	// Run the command to get project fields
	results, err := RunCommandAndReturnOutput(fmt.Sprintf("gh project field-list --owner fleetdm --format json %d", projectID))
	if err != nil {
		return nil, err
	}
	var fields ProjectFieldsResponse
	err = json.Unmarshal(results, &fields)
	if err != nil {
		return nil, err
	}
	fieldMap := make(map[string]ProjectField)
	for _, field := range fields.Fields {
		fieldMap[field.Name] = field
	}
	return fieldMap, nil
}

// LoadProjectFields loads project fields from cache or API.
func LoadProjectFields(projectID int) (map[string]ProjectField, error) {
	if fields, exists := MapProjectFieldNameToField[projectID]; exists {
		return fields, nil
	}
	fields, err := GetProjectFields(projectID)
	if err != nil {
		return nil, err
	}
	MapProjectFieldNameToField[projectID] = fields
	return fields, nil
}

// LookupProjectFieldName looks up a project field by name (case-insensitive).
func LookupProjectFieldName(projectID int, fieldName string) (ProjectField, error) {
	fields, err := LoadProjectFields(projectID)
	if err != nil {
		return ProjectField{}, err
	}

	// First try exact match
	if field, exists := fields[fieldName]; exists {
		return field, nil
	}

	// Then try case-insensitive match
	lowercaseSearch := strings.ToLower(fieldName)
	for name, field := range fields {
		if strings.ToLower(name) == lowercaseSearch {
			return field, nil
		}
	}

	return ProjectField{}, fmt.Errorf("field '%s' not found in project %d", fieldName, projectID)
}

// FindFieldValueByName finds a field option by partial name match (case-insensitive).
func FindFieldValueByName(projectID int, fieldName, search string) (string, error) {
	field, err := LookupProjectFieldName(projectID, fieldName)
	if err != nil {
		return "", err
	}
	for _, option := range field.Options {
		if strings.Contains(strings.ToLower(option.Name), strings.ToLower(search)) {
			return option.Name, nil
		}
	}
	return "", fmt.Errorf("field '%s' not found in project %d", fieldName, projectID)
}

// SetProjectItemFieldValue sets a field value for a project item.
// Uses GraphQL node IDs for proper API compatibility.
func SetProjectItemFieldValue(itemID string, projectID int, fieldName, value string) error {
	// Get the field information
	field, err := LookupProjectFieldName(projectID, fieldName)
	if err != nil {
		return fmt.Errorf("failed to lookup field '%s': %v", fieldName, err)
	}

	// Get the project's GraphQL node ID
	projectNodeID, err := getProjectNodeID(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project node ID: %v", err)
	}

	// If itemID is provided, use it directly
	if itemID != "" {
		// For number fields (like Estimate) - try different possible type names
		if field.Type == "NUMBER" || field.Type == "ProjectV2Field" || strings.Contains(strings.ToLower(field.Type), "number") {
			command := fmt.Sprintf(`gh api graphql -f query='mutation { updateProjectV2ItemFieldValue(input: { projectId: "%s", itemId: "%s", fieldId: "%s", value: { number: %s } }) { projectV2Item { id } } }'`,
				projectNodeID, itemID, field.ID, value)

			_, err := RunCommandAndReturnOutput(command)
			if err != nil {
				return fmt.Errorf("failed to set number field value: %v", err)
			}
			return nil
		}

		// For single select fields (like Status)
		if field.Type == "SINGLE_SELECT" || field.Type == "ProjectV2SingleSelectField" || strings.Contains(strings.ToLower(field.Type), "select") {
			// Use FindFieldValueByName to get the actual option name (which may include emojis)
			actualOptionName, err := FindFieldValueByName(projectID, fieldName, value)
			if err != nil {
				return fmt.Errorf("failed to find option '%s' for field '%s': %v", value, fieldName, err)
			}

			// Find the option ID for the actual option name
			var optionID string
			for _, option := range field.Options {
				if option.Name == actualOptionName {
					optionID = option.ID
					break
				}
			}

			if optionID == "" {
				return fmt.Errorf("option ID not found for '%s' in field '%s'", actualOptionName, fieldName)
			}

			command := fmt.Sprintf(`gh api graphql -f query='mutation { updateProjectV2ItemFieldValue(input: { projectId: "%s", itemId: "%s", fieldId: "%s", value: { singleSelectOptionId: "%s" } }) { projectV2Item { id } } }'`,
				projectNodeID, itemID, field.ID, optionID)

			_, err = RunCommandAndReturnOutput(command)
			if err != nil {
				return fmt.Errorf("failed to set single select field value: %v", err)
			}
			return nil
		}

		// For iteration fields (Sprint)
		if field.Type == "ITERATION" || field.Type == "ProjectV2IterationField" || strings.Contains(strings.ToLower(field.Type), "iteration") {
			// For iteration fields, if value is "@current", we need to find the current iteration ID
			if value == "@current" {
				// First, get the current iteration ID for this field
				currentIterationID, err := getCurrentIterationID(projectNodeID, field.ID)
				if err != nil {
					return fmt.Errorf("failed to get current iteration ID: %v", err)
				}

				if currentIterationID == "" {
					return fmt.Errorf("no current iteration found for field '%s'", fieldName)
				}

				// Use GraphQL mutation with the actual current iteration ID
				command := fmt.Sprintf(`gh api graphql -f query='mutation { updateProjectV2ItemFieldValue(input: { projectId: "%s", itemId: "%s", fieldId: "%s", value: { iterationId: "%s" } }) { projectV2Item { id } } }'`,
					projectNodeID, itemID, field.ID, currentIterationID)

				out, err := RunCommandAndReturnOutput(command)
				if err != nil {
					return fmt.Errorf("failed to set current iteration: %v", err)
				}
				logger.Debugf("GraphQL command input: %s", command)
				logger.Debugf("GraphQL command output: %s", string(out))
				return nil
			}

			// For other iteration values, we would need to look up the iteration ID
			// This is more complex and would require additional API calls
			return fmt.Errorf("setting specific iteration values (other than @current) is not yet implemented")
		}

		// For text fields
		if field.Type == "TEXT" || strings.Contains(strings.ToLower(field.Type), "text") {
			command := fmt.Sprintf(`gh api graphql -f query='mutation { updateProjectV2ItemFieldValue(input: { projectId: "%s", itemId: "%s", fieldId: "%s", value: { text: "%s" } }) { projectV2Item { id } } }'`,
				projectNodeID, itemID, field.ID, value)

			_, err := RunCommandAndReturnOutput(command)
			if err != nil {
				return fmt.Errorf("failed to set text field value: %v", err)
			}
			return nil
		}

		// If we can't determine the type, try to infer from field name or context
		if strings.EqualFold(fieldName, "Estimate") || strings.Contains(strings.ToLower(fieldName), "estimate") {
			command := fmt.Sprintf(`gh api graphql -f query='mutation { updateProjectV2ItemFieldValue(input: { projectId: "%s", itemId: "%s", fieldId: "%s", value: { number: %s } }) { projectV2Item { id } } }'`,
				projectNodeID, itemID, field.ID, value)

			_, err := RunCommandAndReturnOutput(command)
			if err != nil {
				return fmt.Errorf("failed to set number field value (inferred): %v", err)
			}
			return nil
		}

		return fmt.Errorf("unsupported field type: %s for field: %s", field.Type, fieldName)
	}

	return fmt.Errorf("itemID is required for SetProjectItemFieldValue")
}

// GetProjectItemID finds the project item ID for a given issue number in a project with caching.
// Uses GitHub API directly for better performance and reliability.
func GetProjectItemID(issueNumber int, projectID int) (string, error) {
	// Check cache first
	cacheKey := generateProjectItemCacheKey(issueNumber, projectID)
	projectItemIDMutex.RLock()
	if itemID, exists := projectItemIDCache[cacheKey]; exists {
		projectItemIDMutex.RUnlock()
		return itemID, nil
	}
	projectItemIDMutex.RUnlock()

	// Not in cache, fetch from API

	// First, we need to get the project's node ID
	projectNodeID, err := getProjectNodeID(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get project node ID: %v", err)
	}

	// GraphQL query to find the project item for the specific issue
	query := fmt.Sprintf(`{
		node(id: "%s") {
			... on ProjectV2 {
				items(first: 100) {
					nodes {
						id
						content {
							... on Issue {
								number
							}
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}
	}`, projectNodeID)

	// Use gh api to execute the GraphQL query
	command := fmt.Sprintf(`gh api graphql -f query='%s'`, query)
	output, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to query project items via API: %v", err)
	}

	// Parse the GraphQL response
	var response struct {
		Data struct {
			Node struct {
				Items struct {
					Nodes []struct {
						ID      string `json:"id"`
						Content struct {
							Number int `json:"number"`
						} `json:"content"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"items"`
			} `json:"node"`
		} `json:"data"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse GraphQL response: %v", err)
	}

	// Search through the items to find the matching issue number
	for _, item := range response.Data.Node.Items.Nodes {
		if item.Content.Number == issueNumber {
			// Cache the result
			projectItemIDMutex.Lock()
			projectItemIDCache[cacheKey] = item.ID
			projectItemIDMutex.Unlock()

			return item.ID, nil
		}
	}

	// If we have more pages, we should search them too
	if response.Data.Node.Items.PageInfo.HasNextPage {
		return getProjectItemIDWithPagination(issueNumber, projectNodeID, response.Data.Node.Items.PageInfo.EndCursor, cacheKey)
	}

	return "", fmt.Errorf("issue #%d not found in project %d", issueNumber, projectID)
}

// getProjectNodeID gets the GraphQL node ID for a project with caching.
func getProjectNodeID(projectID int) (string, error) {
	// Check cache first
	projectNodeIDMutex.RLock()
	if nodeID, exists := projectNodeIDCache[projectID]; exists {
		projectNodeIDMutex.RUnlock()
		return nodeID, nil
	}
	projectNodeIDMutex.RUnlock()

	// Not in cache, fetch from API
	command := fmt.Sprintf("gh project view --owner fleetdm --format json %d", projectID)
	output, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to get project details: %v", err)
	}

	var response struct {
		ID string `json:"id"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse project details response: %v", err)
	}

	if response.ID == "" {
		return "", fmt.Errorf("project ID not found in response for project %d", projectID)
	}

	// Cache the result
	projectNodeIDMutex.Lock()
	projectNodeIDCache[projectID] = response.ID
	projectNodeIDMutex.Unlock()

	return response.ID, nil
}

// getProjectItemIDWithPagination handles pagination when searching for project items with caching.
func getProjectItemIDWithPagination(issueNumber int, projectNodeID, cursor, cacheKey string) (string, error) {
	query := fmt.Sprintf(`{
		node(id: "%s") {
			... on ProjectV2 {
				items(first: 100, after: "%s") {
					nodes {
						id
						content {
							... on Issue {
								number
							}
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}
	}`, projectNodeID, cursor)

	command := fmt.Sprintf(`gh api graphql -f query='%s'`, query)
	output, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to query project items via API (pagination): %v", err)
	}

	var response struct {
		Data struct {
			Node struct {
				Items struct {
					Nodes []struct {
						ID      string `json:"id"`
						Content struct {
							Number int `json:"number"`
						} `json:"content"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"items"`
			} `json:"node"`
		} `json:"data"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse GraphQL response (pagination): %v", err)
	}

	// Search through this page of items
	for _, item := range response.Data.Node.Items.Nodes {
		if item.Content.Number == issueNumber {
			// Cache the result
			projectItemIDMutex.Lock()
			projectItemIDCache[cacheKey] = item.ID
			projectItemIDMutex.Unlock()

			return item.ID, nil
		}
	}

	// If there are more pages, continue searching
	if response.Data.Node.Items.PageInfo.HasNextPage {
		return getProjectItemIDWithPagination(issueNumber, projectNodeID, response.Data.Node.Items.PageInfo.EndCursor, cacheKey)
	}

	return "", fmt.Errorf("issue #%d not found in project after searching all pages", issueNumber)
}

// getProjectItemFieldValue retrieves the value of a specific field for a project item.
func getProjectItemFieldValue(itemID string, projectID int, fieldName string) (string, error) {
	// Get the field information
	field, err := LookupProjectFieldName(projectID, fieldName)
	if err != nil {
		return "", fmt.Errorf("failed to lookup field '%s': %v", fieldName, err)
	}

	// GraphQL query to get the specific project item with field values
	query := fmt.Sprintf(`{
		node(id: "%s") {
			... on ProjectV2Item {
				fieldValues(first: 10) {
					nodes {
						... on ProjectV2ItemFieldNumberValue {
							field {
								... on ProjectV2FieldCommon {
									id
								}
							}
							number
						}
						... on ProjectV2ItemFieldTextValue {
							field {
								... on ProjectV2FieldCommon {
									id
								}
							}
							text
						}
						... on ProjectV2ItemFieldSingleSelectValue {
							field {
								... on ProjectV2FieldCommon {
									id
								}
							}
							name
						}
					}
				}
			}
		}
	}`, itemID)

	command := fmt.Sprintf(`gh api graphql -f query='%s'`, query)
	output, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to query project item field values: %v", err)
	}

	// Parse the GraphQL response
	var response struct {
		Data struct {
			Node struct {
				FieldValues struct {
					Nodes []struct {
						Field struct {
							ID string `json:"id"`
						} `json:"field"`
						Number *float64 `json:"number,omitempty"`
						Text   *string  `json:"text,omitempty"`
						Name   *string  `json:"name,omitempty"`
					} `json:"nodes"`
				} `json:"fieldValues"`
			} `json:"node"`
		} `json:"data"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse field values response: %v", err)
	}

	// Find the field value that matches our field ID
	for _, fieldValue := range response.Data.Node.FieldValues.Nodes {
		if fieldValue.Field.ID == field.ID {
			if fieldValue.Number != nil {
				return fmt.Sprintf("%.0f", *fieldValue.Number), nil
			}
			if fieldValue.Text != nil {
				return *fieldValue.Text, nil
			}
			if fieldValue.Name != nil {
				return *fieldValue.Name, nil
			}
		}
	}

	return "", nil // Field not set or empty value
}

// getCurrentIterationID finds the current iteration ID for a project's iteration field.
func getCurrentIterationID(projectNodeID, fieldID string) (string, error) {
	// GraphQL query to get the iteration field configuration and find current iteration
	query := fmt.Sprintf(`{
		node(id: "%s") {
			... on ProjectV2 {
				fields(first: 20) {
					nodes {
						... on ProjectV2IterationField {
							id
							name
							configuration {
								iterations {
									id
									title
									startDate
									duration
								}
							}
						}
					}
				}
			}
		}
	}`, projectNodeID)

	command := fmt.Sprintf(`gh api graphql -f query='%s'`, query)
	output, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to query iterations: %v", err)
	}

	// Debug: Let's see what the actual response looks like
	logger.Debugf("Iterations query response: %s", string(output))

	// Parse the GraphQL response
	var response struct {
		Data struct {
			Node struct {
				Fields struct {
					Nodes []struct {
						ID            string `json:"id"`
						Name          string `json:"name"`
						Configuration struct {
							Iterations []struct {
								ID        string `json:"id"`
								Title     string `json:"title"`
								StartDate string `json:"startDate"`
								Duration  int    `json:"duration"`
							} `json:"iterations"`
						} `json:"configuration"`
					} `json:"nodes"`
				} `json:"fields"`
			} `json:"node"`
		} `json:"data"`
	}

	err = json.Unmarshal(output, &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse iterations response: %v", err)
	}

	// Find the field that matches our fieldID
	var targetField *struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Configuration struct {
			Iterations []struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				StartDate string `json:"startDate"`
				Duration  int    `json:"duration"`
			} `json:"iterations"`
		} `json:"configuration"`
	}

	for _, field := range response.Data.Node.Fields.Nodes {
		if field.ID == fieldID {
			targetField = &field
			break
		}
	}

	if targetField == nil {
		return "", fmt.Errorf("iteration field with ID %s not found", fieldID)
	}

	// Find the current iteration (the first one in the list is the current active one)
	// GitHub appears to return iterations in chronological order with past iterations removed
	iterations := targetField.Configuration.Iterations
	if len(iterations) > 0 {
		// Return the first iteration (current active one)
		logger.Infof("Selected current iteration: %s (ID: %s)", iterations[0].Title, iterations[0].ID)
		return iterations[0].ID, nil
	}

	return "", fmt.Errorf("no iterations found for field")
}
