package ghapi

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
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

// ResolveProjectID resolves a project identifier (alias or numeric string) to a project ID
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

// getAliasKeys returns a slice of all available alias keys
func getAliasKeys() []string {
	keys := make([]string, 0, len(Aliases))
	for k := range Aliases {
		keys = append(keys, k)
	}
	return keys
}

var MapProjectFieldNameToField = map[int]map[string]ProjectField{}

func ParseJSONtoProjectItems(jsonData []byte, limit int) ([]ProjectItem, error) {
	var items ProjectItemsResponse
	err := json.Unmarshal(jsonData, &items)
	if err != nil {
		return nil, err
	}
	if limit > 0 && items.TotalCount > limit {
		log.Printf("Warning: The number of items returned exceeds the specified limit. Only the first %d / %d items will be returned.\n", limit, items.TotalCount)
	}
	return items.Items, nil
}

func GetProjectItems(projectID, limit int) ([]ProjectItem, error) {
	// Run the command to get project items
	results, err := RunCommandAndReturnOutput(fmt.Sprintf("gh project item-list --owner fleetdm --format json --limit %d %d", limit, projectID))
	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		return nil, err
	}
	items, err := ParseJSONtoProjectItems(results, limit)
	if err != nil {
		log.Printf("Error parsing issues: %v", err)
		return nil, err
	}
	// log.Printf("Fetched %d items from project %d", len(items), projectID)
	return items, nil
}

func GetProjectFields(projectID int) (map[string]ProjectField, error) {
	// Run the command to get project fields
	results, err := RunCommandAndReturnOutput(fmt.Sprintf("gh project field-list --owner fleetdm --format json %d", projectID))
	if err != nil {
		log.Printf("Error fetching project fields: %v", err)
		return nil, err
	}
	var fields ProjectFieldsResponse
	err = json.Unmarshal(results, &fields)
	if err != nil {
		log.Printf("Error parsing project fields: %v", err)
		return nil, err
	}
	fieldMap := make(map[string]ProjectField)
	for _, field := range fields.Fields {
		fieldMap[field.Name] = field
	}
	return fieldMap, nil
}

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

func LookupProjectFieldName(projectID int, fieldName string) (ProjectField, error) {
	fields, err := LoadProjectFields(projectID)
	if err != nil {
		return ProjectField{}, err
	}
	field, exists := fields[fieldName]
	if !exists {
		return ProjectField{}, fmt.Errorf("field '%s' not found in project %d", fieldName, projectID)
	}
	return field, nil
}

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

// gh project item-edit --id "$ITEM_ID" --project-id "$PROJECT_ID" --field-id "$FIELD_ID" --number "$ESTIMATE"
func SetProjectItemFieldValue(itemID string, fieldName, value string) error {
	return nil
}
