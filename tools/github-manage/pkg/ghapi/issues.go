// Package ghapi provides GitHub issue management functionality including
// label operations, milestone management, and project interactions.
package ghapi

import (
	"encoding/json"
	"fmt"
)

// ParseJSONtoIssues converts JSON data to a slice of Issue structs.
func ParseJSONtoIssues(jsonData []byte) ([]Issue, error) {
	var issues []Issue
	err := json.Unmarshal(jsonData, &issues)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// GetIssues fetches issues from GitHub using optional search criteria.
func GetIssues(search string) ([]Issue, error) {
	var issues []Issue

	command := "gh issue list --json number,title,author,createdAt,updatedAt,state,labels"
	if search != "" {
		command = fmt.Sprintf("%s -S '%s'", command, search)
	}

	results, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return nil, err
	}
	issues, err = ParseJSONtoIssues(results)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// AddLabelToIssue adds a label to an issue.
func AddLabelToIssue(issueNumber int, label string) error {
	command := fmt.Sprintf("gh issue edit %d --add-label %s", issueNumber, label)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return err
	}
	return nil
}

// RemoveLabelFromIssue removes a label from an issue.
func RemoveLabelFromIssue(issueNumber int, label string) error {
	command := fmt.Sprintf("gh issue edit %d --remove-label %s", issueNumber, label)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return err
	}
	return nil
}

// SetMilestoneToIssue sets a milestone for an issue.
func SetMilestoneToIssue(issueNumber int, milestone string) error {
	command := fmt.Sprintf("gh issue edit %d --milestone %s", issueNumber, milestone)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return err
	}
	return nil
}

// AddIssueToProject adds an issue to a project.
func AddIssueToProject(issueNumber int, projectID int) error {
	command := fmt.Sprintf("gh project item-add %d --owner fleetdm --url https://github.com/fleetdm/fleet/issues/%d", projectID, issueNumber)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return err
	}
	return nil
}

// RemoveIssueFromProject removes an issue from a project.
func RemoveIssueFromProject(issueNumber int, projectID int) error {
	// Get the project item ID for this issue using the same method as other functions
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		// If the issue is not found in the project, that's not an error
		if err.Error() == fmt.Sprintf("issue #%d not found in project %d", issueNumber, projectID) {
			return nil
		}
		return fmt.Errorf("failed to get project item ID: %v", err)
	}

	// Remove the item using the project ID and item ID
	command := fmt.Sprintf("gh project item-delete %d --owner fleetdm --id %s", projectID, itemID)
	_, err = RunCommandAndReturnOutput(command)
	if err != nil {
		return fmt.Errorf("failed to remove issue %d from project %d: %v", issueNumber, projectID, err)
	}

	return nil
}

// SyncEstimateField synchronizes the estimate field value from one project to another.
func SyncEstimateField(issueNumber int, sourceProjectID, targetProjectID int) error {
	// Get the source project item to find the current estimate
	sourceItemID, err := GetProjectItemID(issueNumber, sourceProjectID)
	if err != nil {
		return fmt.Errorf("failed to get source project item ID: %v", err)
	}

	// Get the target project item
	targetItemID, err := GetProjectItemID(issueNumber, targetProjectID)
	if err != nil {
		return fmt.Errorf("failed to get target project item ID: %v", err)
	}

	// Get the estimate value from the source project using GraphQL
	sourceEstimate, err := getProjectItemFieldValue(sourceItemID, sourceProjectID, "Estimate")
	if err != nil {
		return fmt.Errorf("failed to get source estimate: %v", err)
	}

	if sourceEstimate == "" || sourceEstimate == "0" {
		return nil // No estimate to sync
	}

	// Set the estimate in the target project
	err = SetProjectItemFieldValue(targetItemID, targetProjectID, "Estimate", sourceEstimate)
	if err != nil {
		return fmt.Errorf("failed to set target estimate: %v", err)
	}

	return nil
}

// SetCurrentSprint sets the current sprint for an issue in a project.
func SetCurrentSprint(issueNumber int, projectID int) error {
	// Get the project item ID
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project item ID: %v", err)
	}

	// Look up the sprint field ID
	sprintField, err := LookupProjectFieldName(projectID, "sprint")
	if err != nil {
		return fmt.Errorf("failed to lookup sprint field: %v", err)
	}

	// Use the general field setting function to set the sprint field to @current
	err = SetProjectItemFieldValue(itemID, projectID, sprintField.Name, "@current")
	if err != nil {
		return fmt.Errorf("failed to set current sprint: %v", err)
	}

	return nil
}

// SetIssueStatus sets the status of an issue in a project using the Status field.
func SetIssueStatus(issueNumber int, projectID int, status string) error {
	// Get the project item ID
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project item ID: %v", err)
	}

	// Use the general field setting function to set the Status field
	err = SetProjectItemFieldValue(itemID, projectID, "Status", status)
	if err != nil {
		return fmt.Errorf("failed to set status: %v", err)
	}

	return nil
}
