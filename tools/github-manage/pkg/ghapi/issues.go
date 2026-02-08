// Package ghapi provides GitHub issue management functionality including
// label operations, milestone management, and project interactions.
package ghapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"fleetdm/gm/pkg/logger"
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

	command := "gh issue list --json number,title,author,createdAt,updatedAt,state,labels,body"
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

// GetIssuesByMilestoneLimited returns up to 'limit' issues for the given milestone title.
// It fetches limit+1 to detect whether there are more results than the limit; the returned
// slice is trimmed to 'limit', and the boolean indicates if more were available.
func GetIssuesByMilestoneLimited(title string, limit int) ([]Issue, bool, error) {
	if limit <= 0 {
		limit = 300
	}
	// Request one extra to detect overflow
	reqLimit := limit + 1
	cmd := fmt.Sprintf("gh issue list --milestone %q --json number,title,author,createdAt,updatedAt,state,labels,body --limit %d", title, reqLimit)
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return nil, false, err
	}
	issues, err := ParseJSONtoIssues(out)
	if err != nil {
		return nil, false, err
	}
	exceeded := false
	if len(issues) > limit {
		exceeded = true
		issues = issues[:limit]
	}
	return issues, exceeded, nil
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

// CloseIssue closes a GitHub issue.
func CloseIssue(issueNumber int) error {
	command := fmt.Sprintf("gh issue close %d", issueNumber)
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
	sourceEstimate, err := GetProjectItemFieldValue(sourceItemID, sourceProjectID, "Estimate")
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

// IssueEvent represents a single event in an issue's timeline.
type IssueEvent struct {
	Event     string `json:"event"`
	Label     Label  `json:"label,omitempty"`
	CreatedAt string `json:"created_at"`
}

// GetIssueTimeline fetches the timeline/events for a specific issue.
func GetIssueTimeline(repo string, issueNumber int, verbose bool) ([]IssueEvent, error) {
	command := fmt.Sprintf("gh api repos/%s/issues/%d/timeline --paginate", repo, issueNumber)
	if verbose {
		fmt.Fprintf(os.Stderr, "  [Timeline #%d] %s\n", issueNumber, command)
	}
	results, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return nil, err
	}

	var events []IssueEvent
	err = json.Unmarshal(results, &events)
	if err != nil {
		return nil, err
	}
	return events, nil
}

// IssueHadLabel checks if an issue ever had a specific label (even if removed).
func IssueHadLabel(repo string, issueNumber int, labelName string, verbose bool) (bool, error) {
	events, err := GetIssueTimeline(repo, issueNumber, verbose)
	if err != nil {
		return false, err
	}

	for _, event := range events {
		if event.Event == "labeled" && strings.EqualFold(event.Label.Name, labelName) {
			return true, nil
		}
	}
	return false, nil
}

// APIIssue represents the JSON structure returned by the GitHub REST API.
type APIIssue struct {
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	State       string           `json:"state"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
	Body        string           `json:"body"`
	User        APIUser          `json:"user"`
	Labels      []APILabel       `json:"labels"`
	PullRequest *PullRequestInfo `json:"pull_request,omitempty"` // If present, this is a PR not an issue
}

// PullRequestInfo is used to detect if an item is a PR (issues don't have this field).
type PullRequestInfo struct {
	URL string `json:"url"`
}

// APIUser represents the author in the GitHub REST API response.
type APIUser struct {
	Login string `json:"login"`
}

// APILabel represents a label in the GitHub REST API response.
type APILabel struct {
	Name string `json:"name"`
}

// GetIssuesCreatedSinceWithLabel finds issues created since a given date that had a specific label at any point.
func GetIssuesCreatedSinceWithLabel(repo string, sinceDate string, labelName string, verbose bool, concurrency int, olderThan int) ([]Issue, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "Fetching issues created since %s...\n", sinceDate)
	}

	// Fetch issues with manual pagination, stopping when we hit issues created before the target date
	// Don't use 'since' param - it filters by updated_at, not created_at
	var allIssues []Issue
	page := 1
	perPage := 100
	sinceTimestamp := sinceDate + "T00:00:00Z"

listLoop:
	for {
		command := fmt.Sprintf("gh api repos/%s/issues -X GET -f state=all -f per_page=%d -f page=%d -f sort=created -f direction=desc", repo, perPage, page)
		results, err := RunCommandAndReturnOutput(command)
		if err != nil {
			return nil, err
		}

		var apiIssues []APIIssue
		err = json.Unmarshal(results, &apiIssues)
		if err != nil {
			return nil, err
		}

		// Print progress for this page
		fmt.Fprint(os.Stderr, ".")

		// Process this batch
		for _, apiIssue := range apiIssues {
			// Skip pull requests
			if apiIssue.PullRequest != nil {
				continue
			}

			// Since we're sorted by created desc, remaining issues in this page and all later pages will also be too old
			if apiIssue.CreatedAt < sinceTimestamp {
				break listLoop
			}

			// Don't add issues outside olderThan to the queue to pull timelines
			if olderThan > 0 && apiIssue.Number >= olderThan {
				continue
			}

			issue := Issue{
				Number:    apiIssue.Number,
				Title:     apiIssue.Title,
				State:     apiIssue.State,
				CreatedAt: apiIssue.CreatedAt,
				UpdatedAt: apiIssue.UpdatedAt,
				Body:      apiIssue.Body,
				Author:    Author{Login: apiIssue.User.Login},
			}
			for _, label := range apiIssue.Labels {
				issue.Labels = append(issue.Labels, Label{Name: label.Name})
			}
			allIssues = append(allIssues, issue)
		}

		if len(apiIssues) < perPage { // no more pages
			break
		}

		page++
	}

	// Newline after page dots
	fmt.Fprintln(os.Stderr, "")

	issues := allIssues

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d issues created since %s. Evaluating each for label '%s'...\n\n", len(issues), sinceDate, labelName)
	} else {
		fmt.Fprintf(os.Stderr, "Fetched %d issues. Evaluating for label '%s'...", len(issues), labelName)
	}

	// Use a semaphore to limit concurrent goroutines
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var filteredIssues []Issue

	for i, issue := range issues {
		wg.Add(1)
		go func(idx int, iss Issue) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if verbose {
				mu.Lock()
				fmt.Fprintf(os.Stderr, "[%d/%d] Checking issue #%d: %s...", idx+1, len(issues), iss.Number, iss.Title)
				mu.Unlock()
			}

			hadLabel := iss.HasLabel(labelName) // short circuit if it currently has the label

			var err error
			if !hadLabel {
				hadLabel, err = IssueHadLabel(repo, iss.Number, labelName, verbose)
				if err != nil {
					if verbose {
						mu.Lock()
						fmt.Fprintf(os.Stderr, " ERROR\n")
						mu.Unlock()
					}
					logger.Errorf("Error checking timeline for issue #%d: %v", iss.Number, err)
					return
				}
			}

			mu.Lock()
			if hadLabel {
				if verbose {
					fmt.Fprintf(os.Stderr, " MATCH\n")
				}
				filteredIssues = append(filteredIssues, iss)
			} else if verbose {
				fmt.Fprintf(os.Stderr, " no label\n")
			}
			// Print progress dot in non-verbose mode
			if !verbose {
				fmt.Fprint(os.Stderr, ".")
			}
			mu.Unlock()
		}(i, issue)
	}

	wg.Wait()

	if verbose {
		fmt.Fprintf(os.Stderr, "\nCompleted evaluation. %d of %d issues had label '%s' at some point.\n\n", len(filteredIssues), len(issues), labelName)
	} else {
		fmt.Fprintf(os.Stderr, " done\n")
	}

	return filteredIssues, nil
}
