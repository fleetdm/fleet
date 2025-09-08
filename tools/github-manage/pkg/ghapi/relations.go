package ghapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
)

// relatedIssuesCache stores related issue lookups to avoid repeated API calls.
var relatedIssuesCache = struct {
	sync.RWMutex
	data map[int][]int
}{data: make(map[int][]int)}

// GetRelatedIssueNumbers attempts to discover "related" issues (sub-tasks) for the given
// issue by parsing its body for GitHub task list references such as:
//   - [ ] #1234 Some task
//   - [x] #5678 Done task
//
// The previous attempt used unsupported fields (trackingIssues / trackedInIssues) which
// are not exposed via `gh issue view --json` in the current GitHub CLI. This fallback
// is lightweight, fast, and avoids extra GraphQL complexity while still capturing the
// common parent -> child task relationships.
// NOTE: We only discover downward (children) links from the issue's own task list. We do
// not currently scan other issues' bodies to infer parents (that would require a repo-wide
// search and could be expensive). For the current selection toggle UX this is sufficient.
func GetRelatedIssueNumbers(issueNumber int) ([]int, error) {
	// Cache check
	relatedIssuesCache.RLock()
	if cached, ok := relatedIssuesCache.data[issueNumber]; ok {
		relatedIssuesCache.RUnlock()
		return cached, nil
	}
	relatedIssuesCache.RUnlock()

	// First attempt: GraphQL subIssues (official hierarchical issues feature)
	related, err := getSubIssuesViaGraphQL(issueNumber)
	if err == nil && len(related) > 0 {
		relatedIssuesCache.Lock()
		relatedIssuesCache.data[issueNumber] = related
		relatedIssuesCache.Unlock()
		return related, nil
	}

	// Fallback: parse task list references in the issue body.
	bodyRelated := getTaskListIssueRefs(issueNumber)
	relatedIssuesCache.Lock()
	relatedIssuesCache.data[issueNumber] = bodyRelated
	relatedIssuesCache.Unlock()
	return bodyRelated, nil
}

// getSubIssuesViaGraphQL fetches sub-issue numbers using the GitHub GraphQL API.
// Returns (nil, error) on hard failures so caller can fallback.
func getSubIssuesViaGraphQL(issueNumber int) ([]int, error) {
	owner, repo, err := getRepoOwnerAndName()
	if err != nil {
		return nil, err
	}

	// Single query: fetch issue by number and include subIssues.
	// We request up to 100 which is well above typical usage; pagination can be added if needed.
	query := `query($owner:String!,$repo:String!,$number:Int!){repository(owner:$owner,name:$repo){issue(number:$number){subIssues(first:100){nodes{number}}}}}`
	cmd := fmt.Sprintf("gh api graphql -f query='%s' -f owner='%s' -f repo='%s' -F number=%d", query, owner, repo, issueNumber)
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			Repository struct {
				Issue struct {
					SubIssues struct {
						Nodes []struct {
							Number int `json:"number"`
						} `json:"nodes"`
					} `json:"subIssues"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if jerr := json.Unmarshal(out, &resp); jerr != nil {
		return nil, jerr
	}
	nodes := resp.Data.Repository.Issue.SubIssues.Nodes
	if len(nodes) == 0 {
		return []int{}, nil
	}
	related := make([]int, 0, len(nodes))
	seen := make(map[int]struct{})
	for _, n := range nodes {
		if n.Number == 0 || n.Number == issueNumber {
			continue
		}
		if _, ok := seen[n.Number]; ok {
			continue
		}
		seen[n.Number] = struct{}{}
		related = append(related, n.Number)
	}
	return related, nil
}

// getTaskListIssueRefs parses the issue body for task list style references as a fallback.
func getTaskListIssueRefs(issueNumber int) []int {
	cmd := fmt.Sprintf("gh issue view %d --json number,body", issueNumber)
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return []int{}
	}
	var parsed struct {
		Number int    `json:"number"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return []int{}
	}
	re := regexp.MustCompile(`(?m)^\s*- \[[ xX]\] #([0-9]+)\b`)
	matches := re.FindAllStringSubmatch(parsed.Body, -1)
	if matches == nil {
		return []int{}
	}
	seen := make(map[int]struct{})
	related := make([]int, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		var num int
		fmt.Sscanf(m[1], "%d", &num)
		if num == 0 || num == issueNumber {
			continue
		}
		if _, exists := seen[num]; !exists {
			seen[num] = struct{}{}
			related = append(related, num)
		}
	}
	return related
}

// repoOwnerNameCache caches owner/name lookups.
var repoOwnerNameCache struct {
	sync.Mutex
	owner string
	name  string
	ok    bool
}

// getRepoOwnerAndName uses `gh repo view` to discover the current repository owner/name once.
func getRepoOwnerAndName() (string, string, error) {
	repoOwnerNameCache.Lock()
	if repoOwnerNameCache.ok {
		owner := repoOwnerNameCache.owner
		name := repoOwnerNameCache.name
		repoOwnerNameCache.Unlock()
		return owner, name, nil
	}
	repoOwnerNameCache.Unlock()

	cmd := "gh repo view --json owner,name"
	out, err := RunCommandAndReturnOutput(cmd)
	if err != nil {
		return "", "", err
	}
	var parsed struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return "", "", err
	}
	repoOwnerNameCache.Lock()
	repoOwnerNameCache.owner = parsed.Owner.Login
	repoOwnerNameCache.name = parsed.Name
	repoOwnerNameCache.ok = true
	repoOwnerNameCache.Unlock()
	return parsed.Owner.Login, parsed.Name, nil
}
