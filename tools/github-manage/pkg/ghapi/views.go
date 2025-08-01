// views are managed searches and display templates for GitHub issues

package ghapi

import (
	"log"
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

func NewView(viewType ViewType, title string, filters ...string) *View {
	return &View{
		Type:    viewType,
		Title:   title,
		Filters: filters,
	}
}

func GetMDMTicketsEstimated() ([]ProjectItem, error) {
	// Grab issues from Drafting project
	draftingProjectID := Aliases["draft"]
	estimatedName, err := FindFieldValueByName(draftingProjectID, "Status", "estimated")
	if err != nil {
		log.Printf("Error looking up Status field: %v", err)
		return nil, err
	}

	issues, err := GetProjectItems(draftingProjectID, 500)
	if err != nil {
		log.Printf("Error fetching issues from Drafting project: %v", err)
		return nil, err
	}

	log.Printf("Fetched %d issues from Drafting project", len(issues))
	// filter down to issues that are estimated with the label "#g-mdm"
	var estimatedIssues []ProjectItem
	for _, issue := range issues {
		if issue.Labels != nil {
			for _, label := range issue.Labels {
				if label == MDM_LABEL {
					if issue.Status == estimatedName {
						estimatedIssues = append(estimatedIssues, issue)
					}
					break
				}
			}
		}
	}
	// log.Printf("Found %d estimated issues with label '#g-mdm'", len(estimatedIssues))
	return estimatedIssues, nil
}
