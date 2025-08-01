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
