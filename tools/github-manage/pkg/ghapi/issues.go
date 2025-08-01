package ghapi

import (
	"encoding/json"
	"fmt"
	"log"
)

func ParseJSONtoIssues(jsonData []byte) ([]Issue, error) {
	var issues []Issue
	err := json.Unmarshal(jsonData, &issues)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func GetIssues(search string) ([]Issue, error) {
	// This function would typically interact with the GitHub API to fetch issues.
	// For now, we return an empty slice and nil error for demonstration purposes.
	// log.Printf("Fetching issues from GitHub API...")
	var issues []Issue

	command := "gh issue list --json number,title,author,createdAt,updatedAt,state,labels"
	if search != "" {
		command = fmt.Sprintf("%s -S '%s'", command, search)
	}

	results, err := RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		log.Printf("Command: %s", results)
		return nil, err
	}
	issues, err = ParseJSONtoIssues(results)
	if err != nil {
		log.Printf("Error parsing issues: %v", err)
		return nil, err
	}
	// log.Printf("Fetched %d issues from GitHub API", len(issues))
	return issues, nil
}

func AddLabelToIssue(issueNumber int, label string) error {
	command := fmt.Sprintf("gh issue edit %d --add-label %s", issueNumber, label)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error adding label to issue %d: %v", issueNumber, err)
		return err
	}
	// log.Printf("Added label '%s' to issue #%d", label, issueNumber)
	return nil
}

func RemoveLabelFromIssue(issueNumber int, label string) error {
	command := fmt.Sprintf("gh issue edit %d --remove-label %s", issueNumber, label)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error removing label from issue %d: %v", issueNumber, err)
		return err
	}
	// log.Printf("Removed label '%s' from issue #%d", label, issueNumber)
	return nil
}

func SetMilestoneToIssue(issueNumber int, milestone string) error {
	command := fmt.Sprintf("gh issue edit %d --milestone %s", issueNumber, milestone)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error setting milestone for issue %d: %v", issueNumber, err)
		return err
	}
	// log.Printf("Set milestone '%s' for issue #%d", milestone, issueNumber)
	return nil
}

func AddIssueToProject(issueNumber int, projectID int) error {
	command := fmt.Sprintf("gh project item-add %d --owner fleetdm --url https://github.com/fleetdm/fleet/issues/%d", projectID, issueNumber)
	_, err := RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error adding issue %d to project %d: %v", issueNumber, projectID, err)
		return err
	}
	// log.Printf("Added issue #%d to project %d", issueNumber, projectID)
	return nil
}

func RemoveIssueFromProject(issueNumber int, projectID int) error {
	// Get the project item ID for this issue using the same method as other functions
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		// If the issue is not found in the project, that's not an error
		if err.Error() == fmt.Sprintf("issue #%d not found in project %d", issueNumber, projectID) {
			log.Printf("Issue #%d not found in project %d (already removed or never added)", issueNumber, projectID)
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

	log.Printf("Removed issue #%d from project %d", issueNumber, projectID)
	return nil
}

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
		log.Printf("No estimate found for issue #%d in source project %d", issueNumber, sourceProjectID)
		return nil
	}

	// Use the updated SetProjectItemFieldValue function to set the estimate in target project
	err = SetProjectItemFieldValue(targetItemID, targetProjectID, "Estimate", sourceEstimate)
	if err != nil {
		return fmt.Errorf("failed to sync estimate: %v", err)
	}

	log.Printf("Synced estimate %s for issue #%d from project %d to project %d",
		sourceEstimate, issueNumber, sourceProjectID, targetProjectID)
	return nil
}

func SetCurrentSprint(issueNumber int, projectID int) error {
	// For now, this is a placeholder function
	// In a real implementation, this would set the sprint field to the current sprint
	// for the given issue in the specified project
	log.Printf("Setting current sprint for issue #%d in project %d (placeholder)", issueNumber, projectID)
	return nil
}

func SetIssueStatus(issueNumber int, projectID int, status string) error {
	// Get the project item ID for this issue
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project item ID: %v", err)
	}

	// Use the updated SetProjectItemFieldValue function which handles GraphQL properly
	err = SetProjectItemFieldValue(itemID, projectID, "Status", status)
	if err != nil {
		return fmt.Errorf("failed to set status: %v", err)
	}

	log.Printf("Set status '%s' for issue #%d in project %d", status, issueNumber, projectID)
	return nil
}
