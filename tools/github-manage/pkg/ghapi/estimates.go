package ghapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// DefaultEstimateSourceProjects returns the default set of projects to scan for estimates
// when syncing to Roadmap: drafting and product group projects.
func DefaultEstimateSourceProjects() []int {
	// unique list; ignore roadmap itself (87) as a source
	return []int{Aliases["draft"], Aliases["mdm"], Aliases["g-software"], Aliases["g-orchestration"], Aliases["g-security-compliance"]}
}

// GetEstimateFromProject returns the numeric estimate for an issue from a specific project.
// Second return indicates whether a non-zero estimate was found.
func GetEstimateFromProject(issueNumber int, projectID int) (int, bool, error) {
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		return 0, false, err
	}
	val, err := getProjectItemFieldValue(itemID, projectID, "Estimate")
	if err != nil {
		return 0, false, err
	}
	if val == "" || val == "0" {
		return 0, false, nil
	}
	i, convErr := strconv.Atoi(val)
	if convErr != nil {
		return 0, false, fmt.Errorf("invalid estimate value '%s' for issue #%d in project %d", val, issueNumber, projectID)
	}
	return i, true, nil
}

// GetEstimateForIssueAcrossProjects scans the provided projects in order and returns
// the first non-zero estimate found for the issue, along with the project ID that provided it.
func GetEstimateForIssueAcrossProjects(issueNumber int, projects []int) (int, int, error) {
	// Efficient path: single per-issue GraphQL to fetch all project item estimates.
	m, err := getIssueEstimatesAcrossProjects(issueNumber)
	if err == nil {
		for _, pid := range projects {
			if v, ok := m[pid]; ok && v > 0 {
				return v, pid, nil
			}
		}
		for pid, v := range m {
			if v > 0 {
				return v, pid, nil
			}
		}
		return 0, 0, nil
	}
	// Fallback path (older approach) if GraphQL fails: check projects one by one
	for _, pid := range projects {
		itemID, e := GetProjectItemID(issueNumber, pid)
		if e != nil {
			continue
		}
		val, e := getProjectItemFieldValue(itemID, pid, "Estimate")
		if e != nil || val == "" || val == "0" {
			continue
		}
		if i, convErr := strconv.Atoi(val); convErr == nil && i > 0 {
			return i, pid, nil
		}
	}
	return 0, 0, nil
}

// SumEstimatesFromSubIssues sums the estimates of direct sub-issues (one level)
// using the first-found estimate across the provided projects for each child.
func SumEstimatesFromSubIssues(issueNumber int, projects []int) (int, error) {
	children, err := GetRelatedIssueNumbers(issueNumber)
	if err != nil {
		return 0, err
	}
	if len(children) == 0 {
		return 0, nil
	}
	sum := 0
	for _, child := range children {
		if est, _, _ := GetEstimateFromAnyProject(child, projects); est > 0 {
			sum += est
		}
	}
	return sum, nil
}

// IsIssueInProject checks whether an issue is a member of the given project.
func IsIssueInProject(issueNumber int, projectID int) bool {
	_, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		return false
	}
	return true
}

// SetEstimateInProject sets the Estimate field for the given issue in the specified project.
func SetEstimateInProject(issueNumber int, projectID int, estimate int) error {
	// Defensive: never set zero/negative estimates; treat as "leave blank"
	if estimate <= 0 {
		return nil
	}
	itemID, err := GetProjectItemID(issueNumber, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project item for issue #%d in project %d: %v", issueNumber, projectID, err)
	}
	return SetProjectItemFieldValue(itemID, projectID, "Estimate", strconv.Itoa(estimate))
}

// IssueInSprint checks whether the given issue has the specified sprint title in any of the provided projects.
// Returns true along with the project ID where the match was found.
func IssueInSprint(issueNumber int, sprintTitle string, projects []int) (bool, int) {
	st := strings.TrimSpace(sprintTitle)
	if st == "" {
		return false, 0
	}
	for _, pid := range projects {
		itemID, err := GetProjectItemID(issueNumber, pid)
		if err != nil {
			continue
		}
		title, err := getProjectItemFieldValue(itemID, pid, "Sprint")
		if err != nil || strings.TrimSpace(title) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(title), st) {
			return true, pid
		}
	}
	return false, 0
}

// GetIssueNumbersForSprint returns all issue numbers in the given project whose Sprint title matches.
func GetIssueNumbersForSprint(projectID int, sprintTitle string, limit int) ([]int, error) {
	st := strings.TrimSpace(sprintTitle)
	if st == "" {
		return []int{}, nil
	}
	items, _, err := GetProjectItemsWithTotal(projectID, limit)
	if err != nil {
		return nil, err
	}
	var nums []int
	for _, it := range items {
		if it.Content.Number == 0 || it.Sprint == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(it.Sprint.Title), st) {
			nums = append(nums, it.Content.Number)
		}
	}
	return nums, nil
}

// GetIssueNumbersForSprintAcrossProjects builds a union set of issue numbers
// that are in the specified sprint across multiple projects.
func GetIssueNumbersForSprintAcrossProjects(projectIDs []int, sprintTitle string, limit int) (map[int]struct{}, error) {
	out := make(map[int]struct{})
	for _, pid := range projectIDs {
		nums, err := GetIssueNumbersForSprint(pid, sprintTitle, limit)
		if err != nil {
			// Skip problematic project but continue others
			continue
		}
		for _, n := range nums {
			out[n] = struct{}{}
		}
	}
	return out, nil
}

// --- Efficient per-issue estimate discovery via GraphQL ---

var (
	issueEstimateCache   = make(map[int]map[int]int) // issue -> (projectNumber -> estimate)
	issueEstimateCacheMu sync.RWMutex
)

// getIssueEstimatesAcrossProjects fetches all project items for the issue and returns
// a map of project number -> Estimate value (if present and >0).
func getIssueEstimatesAcrossProjects(issueNumber int) (map[int]int, error) {
	// Cache lookup
	issueEstimateCacheMu.RLock()
	if m, ok := issueEstimateCache[issueNumber]; ok {
		issueEstimateCacheMu.RUnlock()
		return m, nil
	}
	issueEstimateCacheMu.RUnlock()

	owner, repo, err := getRepoOwnerAndName()
	if err != nil {
		return nil, err
	}
	query := `query($owner:String!,$repo:String!,$number:Int!){
		repository(owner:$owner,name:$repo){
			issue(number:$number){
				projectItems(first:100){
					nodes{
						project{ number }
						fieldValues(first:20){
							nodes{
								__typename
								... on ProjectV2ItemFieldNumberValue{
									number
									field{ ... on ProjectV2FieldCommon{ name } }
								}
							}
						}
					}
				}
			}
		}
	}`
	cmd := fmt.Sprintf("gh api graphql -f query='%s' -f owner='%s' -f repo='%s' -F number=%d", query, owner, repo, issueNumber)
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							Project struct {
								Number int `json:"number"`
							} `json:"project"`
							FieldValues struct {
								Nodes []struct {
									Typename string   `json:"__typename"`
									Number   *float64 `json:"number,omitempty"`
									Field    struct {
										Name string `json:"name"`
									} `json:"field"`
								} `json:"nodes"`
							} `json:"fieldValues"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	result := make(map[int]int)
	for _, it := range resp.Data.Repository.Issue.ProjectItems.Nodes {
		pid := it.Project.Number
		for _, fv := range it.FieldValues.Nodes {
			if strings.EqualFold(fv.Field.Name, "Estimate") && fv.Number != nil {
				v := int(*fv.Number)
				if v > 0 {
					result[pid] = v
				}
			}
		}
	}
	issueEstimateCacheMu.Lock()
	issueEstimateCache[issueNumber] = result
	issueEstimateCacheMu.Unlock()
	return result, nil
}

// GetEstimateFromAnyProject returns the first positive estimate found for the issue,
// preferring the provided project order when specified.
func GetEstimateFromAnyProject(issueNumber int, preferredProjects []int) (int, int, error) {
	m, err := getIssueEstimatesAcrossProjects(issueNumber)
	if err != nil {
		return 0, 0, err
	}
	for _, pid := range preferredProjects {
		if v, ok := m[pid]; ok && v > 0 {
			return v, pid, nil
		}
	}
	for pid, v := range m {
		if v > 0 {
			return v, pid, nil
		}
	}
	return 0, 0, nil
}
