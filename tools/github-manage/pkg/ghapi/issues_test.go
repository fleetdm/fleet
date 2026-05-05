package ghapi

import (
	"encoding/json"
	"testing"
)

func TestParseJSONtoIssues(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectCount int
	}{
		{
			name: "valid single issue",
			jsonData: `[{
				"number": 123,
				"title": "Test Issue",
				"author": {"login": "testuser", "is_bot": false, "name": "Test User", "id": "1"},
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z",
				"state": "open",
				"labels": [{"id": "1", "name": "bug", "description": "Something isn't working", "color": "ff0000"}]
			}]`,
			expectError: false,
			expectCount: 1,
		},
		{
			name: "multiple issues",
			jsonData: `[
				{
					"number": 123,
					"title": "Test Issue 1",
					"author": {"login": "testuser1", "is_bot": false, "name": "Test User 1", "id": "1"},
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"state": "open",
					"labels": []
				},
				{
					"number": 456,
					"title": "Test Issue 2",
					"author": {"login": "testuser2", "is_bot": true, "name": "Test Bot", "id": "2"},
					"createdAt": "2024-01-02T00:00:00Z",
					"updatedAt": "2024-01-02T00:00:00Z",
					"state": "closed",
					"labels": []
				}
			]`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:        "empty array",
			jsonData:    `[]`,
			expectError: false,
			expectCount: 0,
		},
		{
			name:        "invalid json",
			jsonData:    `{invalid json}`,
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "null json",
			jsonData:    `null`,
			expectError: false, // Go's json.Unmarshal handles null differently - it doesn't error but gives nil slice
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := ParseJSONtoIssues([]byte(tt.jsonData))

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

			if len(issues) != tt.expectCount {
				t.Errorf("Expected %d issues, got %d", tt.expectCount, len(issues))
			}

			// Validate first issue if exists
			if len(issues) > 0 {
				issue := issues[0]
				if issue.Number == 0 {
					t.Error("Issue number should not be zero")
				}
				if issue.Title == "" {
					t.Error("Issue title should not be empty")
				}
			}
		})
	}
}

func TestGetIssues(t *testing.T) {
	// Note: This test would require mocking the GitHub CLI or setting up a test environment
	// For now, we'll test the basic structure but skip actual execution
	t.Skip("GetIssues requires GitHub CLI setup and mocking for proper testing")
}

func TestAddLabelToIssue(t *testing.T) {
	// Note: This test would require mocking the GitHub CLI
	t.Skip("AddLabelToIssue requires GitHub CLI setup and mocking for proper testing")
}

func TestRemoveLabelFromIssue(t *testing.T) {
	// Note: This test would require mocking the GitHub CLI
	t.Skip("RemoveLabelFromIssue requires GitHub CLI setup and mocking for proper testing")
}

func TestSetMilestoneToIssue(t *testing.T) {
	// Note: This test would require mocking the GitHub CLI
	t.Skip("SetMilestoneToIssue requires GitHub CLI setup and mocking for proper testing")
}

// Test helper function to validate Issue struct
func TestIssueStructure(t *testing.T) {
	issue := Issue{
		Number: 123,
		Title:  "Test Issue",
		Author: Author{
			Login: "testuser",
			IsBot: false,
			Name:  "Test User",
			ID:    "1",
		},
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		State:     "open",
		Labels: []Label{
			{
				ID:          "1",
				Name:        "bug",
				Description: "Something isn't working",
				Color:       "ff0000",
			},
		},
	}

	// Test JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(issue)
	if err != nil {
		t.Errorf("Failed to marshal issue: %v", err)
	}

	var unmarshaledIssue Issue
	err = json.Unmarshal(jsonData, &unmarshaledIssue)
	if err != nil {
		t.Errorf("Failed to unmarshal issue: %v", err)
	}

	if unmarshaledIssue.Number != issue.Number {
		t.Errorf("Expected issue number %d, got %d", issue.Number, unmarshaledIssue.Number)
	}

	if unmarshaledIssue.Title != issue.Title {
		t.Errorf("Expected issue title %s, got %s", issue.Title, unmarshaledIssue.Title)
	}
}
