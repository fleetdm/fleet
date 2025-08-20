package ghapi

import (
	"testing"
)

// TestSuite runs all tests in the ghapi package
func TestSuite(t *testing.T) {
	t.Run("CLI Tests", func(t *testing.T) {
		t.Run("RunCommandAndReturnOutput", TestRunCommandAndReturnOutput)
		t.Run("RunCommandAndReturnOutput_ErrorHandling", TestRunCommandAndReturnOutput_ErrorHandling)
		t.Run("RunCommandAndReturnOutput_EmptyCommand", TestRunCommandAndReturnOutput_EmptyCommand)
	})

	t.Run("Issues Tests", func(t *testing.T) {
		t.Run("ParseJSONtoIssues", TestParseJSONtoIssues)
		t.Run("IssueStructure", TestIssueStructure)
	})

	t.Run("Models Tests", func(t *testing.T) {
		t.Run("ConvertItemsToIssues", TestConvertItemsToIssues)
		t.Run("StructMarshaling", TestStructMarshaling)
		t.Run("ProjectItemsResponse", TestProjectItemsResponse)
		t.Run("ProjectFieldsResponse", TestProjectFieldsResponse)
	})

	t.Run("Projects Tests", func(t *testing.T) {
		t.Run("ParseJSONtoProjectItems", TestParseJSONtoProjectItems)
		t.Run("Aliases", TestAliases)
		t.Run("LoadProjectFields", TestLoadProjectFields)
		t.Run("LookupProjectFieldName", TestLookupProjectFieldName)
		t.Run("FindFieldValueByName", TestFindFieldValueByName)
		t.Run("SetProjectItemFieldValue", TestSetProjectItemFieldValue)
	})

	t.Run("Views Tests", func(t *testing.T) {
		t.Run("ViewType", TestViewType)
		t.Run("MDMLabel", TestMDMLabel)
		t.Run("NewView", TestNewView)
		t.Run("ViewStructJSONMarshaling", TestViewStructJSONMarshaling)
		t.Run("ViewWithEmptyFilters", TestViewWithEmptyFilters)
	})

	t.Run("Workflows Tests", func(t *testing.T) {
		t.Run("BulkAddLabel", TestBulkAddLabel)
		t.Run("BulkRemoveLabel", TestBulkRemoveLabel)
		t.Run("BulkSprintKickoff", TestBulkSprintKickoff)
		t.Run("BulkMilestoneClose", TestBulkMilestoneClose)
		t.Run("WorkflowFunctionsSignatures", TestWorkflowFunctionsSignatures)
		t.Run("WorkflowsWithValidIssues", TestWorkflowsWithValidIssues)
	})
}

// BenchmarkSuite runs benchmarks for performance-critical functions
func BenchmarkSuite(b *testing.B) {
	// Benchmark JSON parsing
	jsonData := `[{
		"number": 123,
		"title": "Test Issue",
		"author": {"login": "testuser", "is_bot": false, "name": "Test User", "id": "1"},
		"createdAt": "2024-01-01T00:00:00Z",
		"updatedAt": "2024-01-01T00:00:00Z",
		"state": "open",
		"labels": [{"id": "1", "name": "bug", "description": "Something isn't working", "color": "ff0000"}]
	}]`

	b.Run("ParseJSONtoIssues", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ParseJSONtoIssues([]byte(jsonData))
		}
	})

	// Benchmark project items parsing
	projectItemsJSON := `{
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
			}
		],
		"totalCount": 1
	}`

	b.Run("ParseJSONtoProjectItems", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ParseJSONtoProjectItems([]byte(projectItemsJSON), 0)
		}
	})

	// Benchmark conversion
	projectItems := []ProjectItem{
		{
			ID: "item1",
			Content: ProjectItemContent{
				Number: 123,
				Title:  "Test Issue",
				Body:   "Test description",
				Type:   "Issue",
			},
			Labels:    []string{"bug", "feature"},
			Assignees: []string{"user1"},
		},
	}

	b.Run("ConvertItemsToIssues", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ConvertItemsToIssues(projectItems)
		}
	})
}
