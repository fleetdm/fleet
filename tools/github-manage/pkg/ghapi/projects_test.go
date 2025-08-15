package ghapi

import (
	"reflect"
	"testing"
)

func TestParseJSONtoProjectItems(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		limit         int
		expectError   bool
		expectedCount int
	}{
		{
			name: "valid project items response",
			jsonData: `{
				"items": [
					{
						"id": "item1",
						"title": "Test Item 1",
						"content": {
							"body": "Test body 1",
							"number": 123,
							"title": "Test Title 1",
							"type": "Issue",
							"url": "https://github.com/org/repo/issues/123"
						},
						"estimate": 5,
						"repository": "org/repo",
						"labels": ["bug", "priority-high"],
						"assignees": ["user1"],
						"status": "In Progress"
					},
					{
						"id": "item2",
						"title": "Test Item 2",
						"content": {
							"body": "Test body 2",
							"number": 456,
							"title": "Test Title 2",
							"type": "Issue",
							"url": "https://github.com/org/repo/issues/456"
						},
						"estimate": 3,
						"repository": "org/repo",
						"labels": ["feature"],
						"assignees": ["user2"],
						"status": "Done"
					}
				],
				"totalCount": 2
			}`,
			limit:         0,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "empty items response",
			jsonData: `{
				"items": [],
				"totalCount": 0
			}`,
			limit:         0,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:        "invalid json",
			jsonData:    `{invalid json}`,
			limit:       0,
			expectError: true,
		},
		{
			name: "limit warning test",
			jsonData: `{
				"items": [
					{"id": "item1", "title": "Item 1", "content": {"number": 101, "title": "Title 1", "type": "Issue"}},
					{"id": "item2", "title": "Item 2", "content": {"number": 102, "title": "Title 2", "type": "Issue"}},
					{"id": "item3", "title": "Item 3", "content": {"number": 103, "title": "Title 3", "type": "Issue"}}
				],
				"totalCount": 10
			}`,
			limit:         5,
			expectError:   false,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := ParseJSONtoProjectItems([]byte(tt.jsonData), tt.limit)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(items) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(items))
			}

			// Validate first item if exists
			if len(items) > 0 {
				item := items[0]
				if item.ID == "" {
					t.Error("Item ID should not be empty")
				}
				if item.Content.Number == 0 {
					t.Error("Item content number should not be zero")
				}
			}
		})
	}
}

func TestAliases(t *testing.T) {
	expectedAliases := map[string]int{
		"mdm":             58,
		"g-mdm":           58,
		"draft":           67,
		"drafting":        67,
		"g-software":      70,
		"soft":            70,
		"g-orchestration": 71,
		"orch":            71,
	}

	if !reflect.DeepEqual(Aliases, expectedAliases) {
		t.Errorf("Aliases map does not match expected values. Got: %+v, Expected: %+v", Aliases, expectedAliases)
	}
}

func TestGetProjectItems(t *testing.T) {
	// Note: This test would require mocking the GitHub CLI
	t.Skip("GetProjectItems requires GitHub CLI setup and mocking for proper testing")
}

func TestGetProjectFields(t *testing.T) {
	// Note: This test would require mocking the GitHub CLI
	t.Skip("GetProjectFields requires GitHub CLI setup and mocking for proper testing")
}

func TestLoadProjectFields(t *testing.T) {
	// Test caching mechanism
	projectID := 123

	// Clear any existing cache for this test
	if _, exists := MapProjectFieldNameToField[projectID]; exists {
		delete(MapProjectFieldNameToField, projectID)
	}

	// Since we can't mock GitHub CLI easily, we'll test the caching logic by
	// pre-populating the cache and verifying it returns cached data
	testFields := map[string]ProjectField{
		"Status": {
			ID:   "field1",
			Name: "Status",
			Type: "select",
			Options: []ProjectFieldOption{
				{ID: "opt1", Name: "In Progress"},
				{ID: "opt2", Name: "Done"},
			},
		},
	}

	// Pre-populate cache
	MapProjectFieldNameToField[projectID] = testFields

	// Test that cached data is returned
	fields, err := LoadProjectFields(projectID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(fields, testFields) {
		t.Error("LoadProjectFields should return cached fields")
	}

	// Clean up
	delete(MapProjectFieldNameToField, projectID)
}

func TestLookupProjectFieldName(t *testing.T) {
	projectID := 456
	testFields := map[string]ProjectField{
		"Status": {
			ID:   "field1",
			Name: "Status",
			Type: "select",
		},
		"Assignee": {
			ID:   "field2",
			Name: "Assignee",
			Type: "user",
		},
	}

	// Pre-populate cache to avoid GitHub CLI calls
	MapProjectFieldNameToField[projectID] = testFields

	tests := []struct {
		name        string
		fieldName   string
		expectError bool
	}{
		{
			name:        "existing field",
			fieldName:   "Status",
			expectError: false,
		},
		{
			name:        "another existing field",
			fieldName:   "Assignee",
			expectError: false,
		},
		{
			name:        "non-existing field",
			fieldName:   "NonExistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, err := LookupProjectFieldName(projectID, tt.fieldName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for non-existing field")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if field.Name != tt.fieldName {
				t.Errorf("Expected field name %s, got %s", tt.fieldName, field.Name)
			}
		})
	}

	// Clean up
	delete(MapProjectFieldNameToField, projectID)
}

func TestFindFieldValueByName(t *testing.T) {
	projectID := 789
	testFields := map[string]ProjectField{
		"Status": {
			ID:   "field1",
			Name: "Status",
			Type: "select",
			Options: []ProjectFieldOption{
				{ID: "opt1", Name: "In Progress"},
				{ID: "opt2", Name: "Done"},
				{ID: "opt3", Name: "To Do"},
			},
		},
	}

	// Pre-populate cache
	MapProjectFieldNameToField[projectID] = testFields

	tests := []struct {
		name        string
		fieldName   string
		search      string
		expectError bool
		expectedVal string
	}{
		{
			name:        "exact match",
			fieldName:   "Status",
			search:      "Done",
			expectError: false,
			expectedVal: "Done",
		},
		{
			name:        "case insensitive partial match",
			fieldName:   "Status",
			search:      "progress",
			expectError: false,
			expectedVal: "In Progress",
		},
		{
			name:        "no match",
			fieldName:   "Status",
			search:      "NonExistent",
			expectError: true,
		},
		{
			name:        "non-existing field",
			fieldName:   "NonExistentField",
			search:      "anything",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("FindFieldValueByName requires GitHub CLI setup and mocking for proper testing")
			value, err := FindFieldValueByName(projectID, tt.fieldName, tt.search)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if value != tt.expectedVal {
				t.Errorf("Expected value %s, got %s", tt.expectedVal, value)
			}
		})
	}

	// Clean up
	delete(MapProjectFieldNameToField, projectID)
}

func TestSetProjectItemFieldValue(t *testing.T) {
	// Test the stub function
	t.Skip("SetProjectItemFieldValue is a stub and requires GitHub CLI setup for proper testing")
	err := SetProjectItemFieldValue("item123", 123, "Status", "Done")
	if err != nil {
		t.Errorf("SetProjectItemFieldValue stub should not return error, got: %v", err)
	}
}
