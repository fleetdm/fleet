// Package ghapi provides views for managed searches and display templates for GitHub issues.

package ghapi

import (
	"fmt"
)

type ViewType string

const (
	ISSUE_LIST     ViewType = "issue_list"
	ISSUE_DETAIL   ViewType = "issue_detail"
	PROJECT_DETAIL ViewType = "project_detail"
	MDM_LABEL               = "#g-mdm"
)

type View struct {
	Type    ViewType `json:"type"`
	Title   string   `json:"title"`
	Filters []string `json:"filters,omitempty"` // Search filters for issues
}

// NewView creates a new view with the specified type, title, and optional filters.
func NewView(viewType ViewType, title string, filters ...string) *View {
	return &View{
		Type:    viewType,
		Title:   title,
		Filters: filters,
	}
}

// GetMDMTicketsEstimated returns estimated tickets from the MDM project.
func GetMDMTicketsEstimated() ([]ProjectItem, error) {
	return GetEstimatedTicketsForProject(58, 500)
}

// GetEstimatedTicketsForProject gets estimated tickets from the drafting project filtered by the project's label.
func GetEstimatedTicketsForProject(projectID, limit int) ([]ProjectItem, error) {
	// Get the label for this project
	label, exists := ProjectLabels[projectID]
	if !exists {
		return nil, fmt.Errorf("no label mapping found for project ID %d. Available projects: %v", projectID, getProjectIDsWithLabels())
	}

	// Grab issues from Drafting project
	draftingProjectID := Aliases["draft"]
	estimatedName, err := FindFieldValueByName(draftingProjectID, "Status", "estimated")
	if err != nil {
		return nil, err
	}

	issues, err := GetProjectItems(draftingProjectID, limit)
	if err != nil {
		return nil, err
	}

	// filter down to issues that are estimated with the specified label
	var estimatedIssues []ProjectItem
	for _, issue := range issues {
		if issue.Labels != nil {
			for _, issueLabel := range issue.Labels {
				if issueLabel == label {
					if issue.Status == estimatedName {
						estimatedIssues = append(estimatedIssues, issue)
					}
					break
				}
			}
		}
	}
	return estimatedIssues, nil
}

// getProjectIDsWithLabels returns a slice of project IDs that have label mappings.
func getProjectIDsWithLabels() []int {
	ids := make([]int, 0, len(ProjectLabels))
	for id := range ProjectLabels {
		ids = append(ids, id)
	}
	return ids
}
