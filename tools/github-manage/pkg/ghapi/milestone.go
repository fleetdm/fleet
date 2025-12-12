package ghapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"fleetdm/gm/pkg/logger"
)

// GetIssuesByMilestone returns issue numbers for a given milestone name. Limit controls max items.
func GetIssuesByMilestone(name string, limit int) ([]int, error) {
	if limit <= 0 {
		limit = 1000
	}
	// Use gh to list issues for the current repo by milestone
	// Include closed/open (state all) to reflect full milestone scope
	cmd := fmt.Sprintf("gh issue list --state all --milestone %q --limit %d --json number", name, limit)
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return nil, err
	}
	var arr []struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(out, &arr); err != nil {
		return nil, err
	}
	nums := make([]int, 0, len(arr))
	for _, it := range arr {
		nums = append(nums, it.Number)
	}
	return nums, nil
}

// GetIssuesByMilestoneWithTitles returns issue numbers and titles for a milestone.
func GetIssuesByMilestoneWithTitles(name string, limit int) ([]MilestoneIssue, error) {
	if limit <= 0 {
		limit = 1000
	}
	cmd := fmt.Sprintf("gh issue list --state all --milestone %q --limit %d --json number,title,labels", name, limit)
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return nil, err
	}
	var arr []MilestoneIssue
	if err := json.Unmarshal(out, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

// ProjectInfo represents a GitHub Project (v2) basic descriptor used in reports.
type ProjectInfo struct {
	ID    int    // project number (e.g., 58)
	Title string // project title (e.g., g-mdm)
}

// MilestoneIssue represents a simple issue record for a milestone.
type MilestoneIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// GetIssueProjects returns projects (id->title) that a specific issue belongs to.
func GetIssueProjects(issueNumber int) (map[int]string, error) {
	owner, repo, err := getRepoOwnerAndName()
	if err != nil {
		return nil, err
	}
	query := `query($owner:String!,$repo:String!,$number:Int!){
        repository(owner:$owner,name:$repo){
            issue(number:$number){
                projectItems(first:50){
                    nodes{
                        project{ number title }
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
								Number int    `json:"number"`
								Title  string `json:"title"`
							} `json:"project"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	m := make(map[int]string)
	for _, n := range resp.Data.Repository.Issue.ProjectItems.Nodes {
		if n.Project.Number != 0 {
			m[n.Project.Number] = n.Project.Title
		}
	}
	return m, nil
}

// GetProjectsForIssues gathers the union of projects across the provided issues.
func GetProjectsForIssues(issueNumbers []int) ([]ProjectInfo, error) {
	seen := make(map[int]string)
	for _, num := range issueNumbers {
		prjs, err := GetIssueProjects(num)
		if err != nil {
			// tolerate errors per-issue, continue accumulating from others
			continue
		}
		for id, title := range prjs {
			if _, ok := seen[id]; !ok {
				seen[id] = title
			}
		}
	}
	list := make([]ProjectInfo, 0, len(seen))
	for id, title := range seen {
		list = append(list, ProjectInfo{ID: id, Title: title})
	}
	// stable order: by title then id
	sort.Slice(list, func(i, j int) bool {
		if list[i].Title == list[j].Title {
			return list[i].ID < list[j].ID
		}
		return list[i].Title < list[j].Title
	})
	return list, nil
}

// GetIssueProjectStatuses returns a map of projectID -> Status value for an issue across given projects.
// If the issue is not present in a project or the Status is unset, the value will be an empty string.
type ProjectStatus struct {
	Present bool   // true if the issue is in this project
	Status  string // "" when unset
}

func GetIssueProjectStatuses(issueNumber int, projects []int) (map[int]ProjectStatus, error) {
	// Single GraphQL query to fetch all project items and their Status field for this issue
	owner, repo, err := getRepoOwnerAndName()
	if err != nil {
		// If repo cannot be determined, return absent for all
		res := make(map[int]ProjectStatus, len(projects))
		for _, pid := range projects {
			res[pid] = ProjectStatus{Present: false, Status: ""}
		}
		return res, nil
	}

	query := `query($owner:String!,$repo:String!,$number:Int!){
		repository(owner:$owner,name:$repo){
			issue(number:$number){
				projectItems(first:100){
					nodes{
						project{ number title }
						fieldValues(first:50){
							nodes{
								__typename
								... on ProjectV2ItemFieldSingleSelectValue{
									field{ ... on ProjectV2FieldCommon{ name } }
									name
								}
							}
						}
					}
				}
			}
		}
	}`
	cmd := fmt.Sprintf("gh api graphql -f query='%s' -f owner='%s' -f repo='%s' -F number=%d", query, owner, repo, issueNumber)
	out, err := runCommandWithRetry(cmd, 5, 2*time.Second)
	if err != nil {
		// On error, default to absent for all requested projects
		res := make(map[int]ProjectStatus, len(projects))
		for _, pid := range projects {
			res[pid] = ProjectStatus{Present: false, Status: ""}
		}
		return res, nil
	}
	var resp struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							Project struct {
								Number int    `json:"number"`
								Title  string `json:"title"`
							} `json:"project"`
							FieldValues struct {
								Nodes []struct {
									Typename string `json:"__typename"`
									Field    struct {
										Name string `json:"name"`
									} `json:"field"`
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"fieldValues"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		res := make(map[int]ProjectStatus, len(projects))
		for _, pid := range projects {
			res[pid] = ProjectStatus{Present: false, Status: ""}
		}
		return res, nil
	}

	// Build a map of project number -> status value
	found := make(map[int]ProjectStatus)
	for _, node := range resp.Data.Repository.Issue.ProjectItems.Nodes {
		pid := node.Project.Number
		statusVal := ""
		for _, fv := range node.FieldValues.Nodes {
			if strings.EqualFold(fv.Field.Name, "Status") && fv.Typename == "ProjectV2ItemFieldSingleSelectValue" {
				statusVal = fv.Name
				break
			}
		}
		found[pid] = ProjectStatus{Present: true, Status: statusVal}
	}

	// Compose response for requested projects, marking absences with Present=false
	res := make(map[int]ProjectStatus, len(projects))
	for _, pid := range projects {
		if ps, ok := found[pid]; ok {
			res[pid] = ps
		} else {
			res[pid] = ProjectStatus{Present: false, Status: ""}
		}
	}
	return res, nil
}

// runCommandWithRetry executes a shell command capturing combined output and retries with
// exponential backoff when a rate limit is detected in the output.
func runCommandWithRetry(command string, attempts int, baseDelay time.Duration) ([]byte, error) {
	if attempts < 1 {
		attempts = 1
	}
	delay := baseDelay
	for i := 1; i <= attempts; i++ {
		logger.Debugf("Running COMMAND (attempt %d/%d): %s", i, attempts, command)
		cmd := exec.Command("bash", "-c", command)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		if err == nil {
			return out.Bytes(), nil
		}
		outStr := out.String()
		logger.Errorf("Error running command (attempt %d): %s", i, outStr)
		if i == attempts || !looksLikeRateLimit(outStr) {
			return nil, err
		}
		logger.Infof("Rate limit detected; backing off for %s before retry", delay)
		time.Sleep(delay)
		// Exponential backoff with cap
		if delay < 30*time.Second {
			delay = delay * 2
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
		}
	}
	return nil, fmt.Errorf("command failed after %d attempts", attempts)
}

func looksLikeRateLimit(s string) bool {
	ls := strings.ToLower(s)
	if strings.Contains(ls, "rate limit") || strings.Contains(ls, "graphql_rate_limit") || strings.Contains(ls, "429") {
		return true
	}
	return false
}

// RepoMilestone represents a repository milestone (from REST API).
type RepoMilestone struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// ListRepoMilestones returns milestones for the current repository.
// When includeClosed is false, only open milestones are returned; otherwise all states are included.
func ListRepoMilestones(includeClosed bool) ([]RepoMilestone, error) {
	owner, repo, err := getRepoOwnerAndName()
	if err != nil {
		return nil, err
	}
	state := "open"
	if includeClosed {
		state = "all"
	}
	// Paginate to ensure we gather more than one page if present
	all := make([]RepoMilestone, 0)
	for page := 1; page <= 10; page++ { // hard cap to avoid accidental infinite loops
		cmd := fmt.Sprintf("gh api repos/%s/%s/milestones?state=%s&per_page=100&page=%d", owner, repo, state, page)
		out, err := RunCommandAndReturnOutput(cmd)
		if err != nil {
			return nil, err
		}
		var arr []RepoMilestone
		if err := json.Unmarshal(out, &arr); err != nil {
			return nil, err
		}
		if len(arr) == 0 {
			break
		}
		all = append(all, arr...)
		if len(arr) < 100 {
			break
		}
	}
	// Sort open first by title, then closed by title
	sort.Slice(all, func(i, j int) bool {
		si := strings.ToLower(all[i].State)
		sj := strings.ToLower(all[j].State)
		if si != sj {
			if si == "open" {
				return true
			}
			if sj == "open" {
				return false
			}
		}
		return strings.ToLower(all[i].Title) < strings.ToLower(all[j].Title)
	})
	return all, nil
}
