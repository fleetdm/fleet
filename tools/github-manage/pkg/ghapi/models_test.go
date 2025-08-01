package ghapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertItemsToIssues(t *testing.T) {
	tests := []struct {
		name          string
		items         []ProjectItem
		expectedCount int
	}{
		{
			name:          "empty items",
			items:         []ProjectItem{},
			expectedCount: 0,
		},
		{
			name: "single item with basic fields",
			items: []ProjectItem{
				{
					ID: "item1",
					Content: ProjectItemContent{
						Number: 123,
						Title:  "Test Issue",
						Body:   "Test description",
						Type:   "Issue",
						URL:    "https://github.com/org/repo/issues/123",
					},
					Estimate:   5,
					Repository: "org/repo",
					Labels:     []string{"bug", "priority-high"},
					Assignees:  []string{"user1", "user2"},
				},
			},
			expectedCount: 1,
		},
		{
			name: "item with milestone",
			items: []ProjectItem{
				{
					ID: "item2",
					Content: ProjectItemContent{
						Number: 456,
						Title:  "Feature Request",
						Body:   "Add new feature",
						Type:   "Issue",
					},
					Milestone: &Milestone{
						Number:      1,
						Title:       "v1.0",
						Description: "First release",
						DueOn:       "2024-12-31T23:59:59Z",
					},
				},
			},
			expectedCount: 1,
		},
		{
			name: "items with different label types",
			items: []ProjectItem{
				{
					ID: "story",
					Content: ProjectItemContent{
						Number: 101,
						Title:  "Story Item",
					},
					Labels: []string{"story"},
				},
				{
					ID: "bug",
					Content: ProjectItemContent{
						Number: 102,
						Title:  "Bug Item",
					},
					Labels: []string{"bug"},
				},
				{
					ID: "task",
					Content: ProjectItemContent{
						Number: 103,
						Title:  "Task Item",
					},
					Labels: []string{"~sub-task"},
				},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ConvertItemsToIssues(tt.items)

			if len(issues) != tt.expectedCount {
				t.Errorf("Expected %d issues, got %d", tt.expectedCount, len(issues))
			}

			// Validate specific conversions
			for i, item := range tt.items {
				if i >= len(issues) {
					break
				}
				issue := issues[i]

				if issue.ID != item.ID {
					t.Errorf("Expected issue ID %s, got %s", item.ID, issue.ID)
				}

				if issue.Number != item.Content.Number {
					t.Errorf("Expected issue number %d, got %d", item.Content.Number, issue.Number)
				}

				if issue.Title != item.Content.Title {
					t.Errorf("Expected issue title %s, got %s", item.Content.Title, issue.Title)
				}

				if issue.Body != item.Content.Body {
					t.Errorf("Expected issue body %s, got %s", item.Content.Body, issue.Body)
				}

				if issue.Estimate != item.Estimate {
					t.Errorf("Expected issue estimate %d, got %d", item.Estimate, issue.Estimate)
				}

				// Check milestone conversion
				if item.Milestone != nil {
					if issue.Milestone == nil {
						t.Error("Expected milestone to be converted, got nil")
					} else {
						if issue.Milestone.Number != item.Milestone.Number {
							t.Errorf("Expected milestone number %d, got %d", item.Milestone.Number, issue.Milestone.Number)
						}
						if issue.Milestone.Title != item.Milestone.Title {
							t.Errorf("Expected milestone title %s, got %s", item.Milestone.Title, issue.Milestone.Title)
						}
					}
				}

				// Check assignees conversion
				if len(item.Assignees) != len(issue.Assignees) {
					t.Errorf("Expected %d assignees, got %d", len(item.Assignees), len(issue.Assignees))
				}

				// Check label conversion and typename assignment
				for _, label := range item.Labels {
					found := false
					for _, issueLabel := range issue.Labels {
						if issueLabel.Name == label {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected label %s to be converted", label)
					}

					// Check typename assignment
					switch label {
					case "story":
						if issue.Typename != "Feature" {
							t.Errorf("Expected typename 'Feature' for story label, got %s", issue.Typename)
						}
					case "bug":
						if issue.Typename != "Bug" {
							t.Errorf("Expected typename 'Bug' for bug label, got %s", issue.Typename)
						}
					case "~sub-task":
						if issue.Typename != "Task" {
							t.Errorf("Expected typename 'Task' for ~sub-task label, got %s", issue.Typename)
						}
					}
				}
			}
		})
	}
}

func TestStructMarshaling(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "Author struct",
			data: Author{
				Login: "testuser",
				IsBot: false,
				Name:  "Test User",
				ID:    "123",
			},
		},
		{
			name: "Label struct",
			data: Label{
				ID:          "label1",
				Name:        "bug",
				Description: "Something isn't working",
				Color:       "ff0000",
			},
		},
		{
			name: "Milestone struct",
			data: Milestone{
				Number:      1,
				Title:       "v1.0",
				Description: "First release",
				DueOn:       "2024-12-31T23:59:59Z",
			},
		},
		{
			name: "Issue struct",
			data: Issue{
				Typename:  "Bug",
				ID:        "issue1",
				Number:    123,
				Title:     "Test Issue",
				Body:      "Test description",
				Author:    Author{Login: "testuser"},
				Assignees: []Author{{Login: "assignee1"}},
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-01T00:00:00Z",
				State:     "open",
				Labels:    []Label{{Name: "bug"}},
				Milestone: &Milestone{Title: "v1.0"},
				Estimate:  5,
			},
		},
		{
			name: "ProjectItem struct",
			data: ProjectItem{
				ID:    "item1",
				Title: "Test Item",
				Content: ProjectItemContent{
					Body:   "Test body",
					Number: 123,
					Title:  "Test Title",
					Type:   "Issue",
					URL:    "https://example.com",
				},
				Estimate:   3,
				Repository: "org/repo",
				Labels:     []string{"bug", "feature"},
				Assignees:  []string{"user1"},
				Status:     "In Progress",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonData, err := json.Marshal(tt.data)
			if err != nil {
				t.Errorf("Failed to marshal %s: %v", tt.name, err)
				return
			}

			// Test unmarshaling
			dataType := reflect.TypeOf(tt.data)
			newValue := reflect.New(dataType).Interface()

			err = json.Unmarshal(jsonData, newValue)
			if err != nil {
				t.Errorf("Failed to unmarshal %s: %v", tt.name, err)
				return
			}

			// Compare original and unmarshaled data
			originalValue := reflect.ValueOf(tt.data)
			unmarshaledValue := reflect.ValueOf(newValue).Elem()

			if !reflect.DeepEqual(originalValue.Interface(), unmarshaledValue.Interface()) {
				t.Errorf("Original and unmarshaled %s data do not match", tt.name)
			}
		})
	}
}

func TestProjectItemsResponse(t *testing.T) {
	response := ProjectItemsResponse{
		Items: []ProjectItem{
			{ID: "item1", Title: "Item 1"},
			{ID: "item2", Title: "Item 2"},
		},
		TotalCount: 2,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal ProjectItemsResponse: %v", err)
	}

	var unmarshaled ProjectItemsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProjectItemsResponse: %v", err)
	}

	if unmarshaled.TotalCount != response.TotalCount {
		t.Errorf("Expected TotalCount %d, got %d", response.TotalCount, unmarshaled.TotalCount)
	}

	if len(unmarshaled.Items) != len(response.Items) {
		t.Errorf("Expected %d items, got %d", len(response.Items), len(unmarshaled.Items))
	}
}

func TestProjectFieldsResponse(t *testing.T) {
	response := ProjectFieldsResponse{
		Fields: []ProjectField{
			{
				ID:   "field1",
				Name: "Status",
				Type: "select",
				Options: []ProjectFieldOption{
					{ID: "opt1", Name: "In Progress"},
					{ID: "opt2", Name: "Done"},
				},
			},
		},
		TotalCount: 1,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal ProjectFieldsResponse: %v", err)
	}

	var unmarshaled ProjectFieldsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal ProjectFieldsResponse: %v", err)
	}

	if unmarshaled.TotalCount != response.TotalCount {
		t.Errorf("Expected TotalCount %d, got %d", response.TotalCount, unmarshaled.TotalCount)
	}

	if len(unmarshaled.Fields) != len(response.Fields) {
		t.Errorf("Expected %d fields, got %d", len(response.Fields), len(unmarshaled.Fields))
	}
}
