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
	log.Printf("Fetching issues from GitHub API...")
	var issues []Issue

	command := "gh issue list --json number,title,author,createdAt,updatedAt,state,labels"
	if search != "" {
		command = fmt.Sprintf("%s -S %s", command, search)
	}

	results, err := RunCommandAndParseJSON(command)
	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		return nil, err
	}
	issues, err = ParseJSONtoIssues(results)
	if err != nil {
		log.Printf("Error parsing issues: %v", err)
		return nil, err
	}
	log.Printf("Fetched %d issues from GitHub API", len(issues))
	// log.Printf("Issues: %+v", issues)
	return issues, nil
}
