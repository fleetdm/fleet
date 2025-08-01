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
		command = fmt.Sprintf("%s -S %s", command, search)
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
	// First get the project item ID for this issue
	command := fmt.Sprintf("gh project item-list %d --owner fleetdm --format json", projectID)
	output, err := RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error listing project items for project %d: %v", projectID, err)
		return err
	}

	// Parse the output to find the item ID for this issue
	var items []map[string]interface{}
	err = json.Unmarshal(output, &items)
	if err != nil {
		return err
	}

	var itemID string
	for _, item := range items {
		if content, ok := item["content"].(map[string]interface{}); ok {
			if number, ok := content["number"].(float64); ok && int(number) == issueNumber {
				if id, ok := item["id"].(string); ok {
					itemID = id
					break
				}
			}
		}
	}

	if itemID == "" {
		log.Printf("Issue #%d not found in project %d", issueNumber, projectID)
		return nil // Not an error if item is not in project
	}

	// Remove the item
	command = fmt.Sprintf("gh project item-delete %d --owner fleetdm --id %s", projectID, itemID)
	_, err = RunCommandAndReturnOutput(command)
	if err != nil {
		log.Printf("Error removing issue %d from project %d: %v", issueNumber, projectID, err)
		return err
	}
	// log.Printf("Removed issue #%d from project %d", issueNumber, projectID)
	return nil
}

func SyncEstimateField(issueNumber int, targetProjectID int) error {
	// For now, this is a placeholder function
	// In a real implementation, this would sync the estimate field from the drafting project
	// to the target project for the given issue
	log.Printf("Syncing estimate field for issue #%d to project %d (placeholder)", issueNumber, targetProjectID)
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
	// For now, this is a placeholder function
	// In a real implementation, this would set the status field for the given issue
	// in the specified project using GitHub CLI or API
	log.Printf("Setting status '%s' for issue #%d in project %d (placeholder)", status, issueNumber, projectID)
	return nil
}
