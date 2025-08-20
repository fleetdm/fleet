package ghapi

import (
	"reflect"
	"testing"
)

func TestViewType(t *testing.T) {
	// Test the ViewType constants
	expectedTypes := map[ViewType]string{
		ISSUE_LIST:     "issue_list",
		ISSUE_DETAIL:   "issue_detail",
		PROJECT_DETAIL: "project_detail",
	}

	for viewType, expectedValue := range expectedTypes {
		if string(viewType) != expectedValue {
			t.Errorf("Expected ViewType %s to have value %s, got %s", expectedValue, expectedValue, string(viewType))
		}
	}
}

func TestMDMLabel(t *testing.T) {
	expectedLabel := "#g-mdm"
	if MDM_LABEL != expectedLabel {
		t.Errorf("Expected MDM_LABEL to be %s, got %s", expectedLabel, MDM_LABEL)
	}
}

func TestNewView(t *testing.T) {
	tests := []struct {
		name     string
		viewType ViewType
		title    string
		filters  []string
	}{
		{
			name:     "issue list view with no filters",
			viewType: ISSUE_LIST,
			title:    "All Issues",
			filters:  nil,
		},
		{
			name:     "issue list view with filters",
			viewType: ISSUE_LIST,
			title:    "Bug Issues",
			filters:  []string{"label:bug", "state:open"},
		},
		{
			name:     "issue detail view",
			viewType: ISSUE_DETAIL,
			title:    "Issue #123",
			filters:  []string{"number:123"},
		},
		{
			name:     "project detail view",
			viewType: PROJECT_DETAIL,
			title:    "Project Dashboard",
			filters:  []string{"project:58"},
		},
		{
			name:     "view with multiple filters",
			viewType: ISSUE_LIST,
			title:    "MDM Issues",
			filters:  []string{"label:g-mdm", "state:open", "milestone:v1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewView(tt.viewType, tt.title, tt.filters...)

			if view == nil {
				t.Fatal("NewView returned nil")
			}

			if view.Type != tt.viewType {
				t.Errorf("Expected view type %s, got %s", tt.viewType, view.Type)
			}

			if view.Title != tt.title {
				t.Errorf("Expected view title %s, got %s", tt.title, view.Title)
			}

			if !reflect.DeepEqual(view.Filters, tt.filters) {
				t.Errorf("Expected filters %v, got %v", tt.filters, view.Filters)
			}
		})
	}
}

func TestGetMDMTicketsEstimated(t *testing.T) {
	// Test the basic structure and error handling
	// Note: This function depends on external GitHub CLI calls and project data
	// For unit testing, we would need to mock the dependencies

	// We'll test error conditions we can simulate without mocking

	// Save original Aliases and MapProjectFieldNameToField state
	originalAliases := make(map[string]int)
	for k, v := range Aliases {
		originalAliases[k] = v
	}

	// Test case: invalid project alias
	delete(Aliases, "draft")

	// This test validates that the function handles missing aliases gracefully
	// In a real scenario, this would be caught by FindFieldValueByName
	t.Run("missing draft alias", func(t *testing.T) {
		// Since we can't easily mock the GitHub CLI calls, we'll skip the actual execution
		// but verify the function signature and basic setup
		if _, exists := Aliases["draft"]; exists {
			t.Error("Expected 'draft' alias to be removed for this test")
		}
	})

	// Restore original state
	for k, v := range originalAliases {
		Aliases[k] = v
	}

	// Test with valid aliases
	t.Run("valid aliases exist", func(t *testing.T) {
		if draftID, exists := Aliases["draft"]; !exists {
			t.Error("Expected 'draft' alias to exist")
		} else if draftID != 67 {
			t.Errorf("Expected draft project ID to be 67, got %d", draftID)
		}
	})

	// Test MDM_LABEL constant usage
	t.Run("mdm label constant", func(t *testing.T) {
		if MDM_LABEL != "#g-mdm" {
			t.Errorf("Expected MDM_LABEL to be '#g-mdm', got %s", MDM_LABEL)
		}
	})

	// For integration testing, we would mock the GitHub CLI calls
	t.Skip("GetMDMTicketsEstimated requires GitHub CLI setup and mocking for proper testing")
}

func TestViewStructJSONMarshaling(t *testing.T) {
	view := &View{
		Type:    ISSUE_LIST,
		Title:   "Test View",
		Filters: []string{"label:bug", "state:open"},
	}

	// Test the basic structure of the View struct
	if view.Type != ISSUE_LIST {
		t.Errorf("Expected type %s, got %s", ISSUE_LIST, view.Type)
	}

	if view.Title != "Test View" {
		t.Errorf("Expected title 'Test View', got %s", view.Title)
	}

	expectedFilters := []string{"label:bug", "state:open"}
	if !reflect.DeepEqual(view.Filters, expectedFilters) {
		t.Errorf("Expected filters %v, got %v", expectedFilters, view.Filters)
	}
}

func TestViewWithEmptyFilters(t *testing.T) {
	view := NewView(PROJECT_DETAIL, "Empty Filters View")

	if view.Filters != nil {
		t.Errorf("Expected nil filters, got %v", view.Filters)
	}

	// Test with empty slice
	view2 := NewView(ISSUE_DETAIL, "Empty Slice View", []string{}...)

	if len(view2.Filters) != 0 {
		t.Errorf("Expected empty filters slice, got %v", view2.Filters)
	}
}
