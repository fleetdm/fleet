package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type userLoginResponse struct {
	Login string `json:"login"`
}

var searchAssignedIssuesByProject = fetchAssignedIssuesByProject

// runMissingAssigneeChecks builds the combined assignee findings used by both
// "Missing assignee" and "Assigned to me" checks.
// It scans selected projects, skips Done-column items, and augments results
// with direct GitHub search hits for assignee:@me in each selected project.
func runMissingAssigneeChecks(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNums []int,
	limit int,
	token string,
) []MissingAssigneeIssue {
	viewer := strings.ToLower(strings.TrimSpace(fetchViewerLogin(ctx, token)))
	issueKey := func(owner, repo string, number int) string {
		return strings.ToLower(strings.TrimSpace(owner)) + "/" +
			strings.ToLower(strings.TrimSpace(repo)) + "#" + fmt.Sprintf("%d", number)
	}

	cache := make(map[string][]AssigneeOption)
	assignedSearchByProject := make(map[int]map[string]searchIssueItem, len(projectNums))
	for _, projectNum := range projectNums {
		// This side query intentionally mirrors `assignee:@me` behavior so items
		// assigned to the viewer are still surfaced even if local project-item
		// fetch windows miss them.
		found := []searchIssueItem(nil)
		if viewer != "" {
			found = searchAssignedIssuesByProject(ctx, token, org, projectNum)
		}
		byNumber := make(map[string]searchIssueItem, len(found))
		for _, it := range found {
			owner, repo := parseRepoFromRepositoryAPIURL(it.RepositoryURL)
			if owner == "" || repo == "" {
				continue
			}
			byNumber[issueKey(owner, repo, it.Number)] = it
		}
		assignedSearchByProject[projectNum] = byNumber
	}

	out := make([]MissingAssigneeIssue, 0)
	for _, projectNum := range projectNums {
		projectID := fetchProjectID(ctx, client, org, projectNum)
		items := fetchItems(ctx, client, projectID, limit)
		searchAssigned := assignedSearchByProject[projectNum]
		seenAssigned := make(map[string]struct{})

		for _, it := range items {
			// Assign-to-me classification is derived from either project card
			// assignees or direct search results to make the check resilient to
			// field projection differences.
			if it.Content.Issue.Number == 0 {
				continue
			}
			currentAssignees := issueAssignees(it)
			number := getNumber(it)
			owner, repo := parseRepoFromIssueURL(getURL(it))
			if owner == "" || repo == "" {
				continue
			}
			key := issueKey(owner, repo, number)
			_, searchHit := searchAssigned[key]
			assignedToMe := containsLogin(currentAssignees, viewer) || searchHit
			if assignedToMe {
				seenAssigned[key] = struct{}{}
			}
			if inDoneColumn(it) {
				continue
			}
			if len(currentAssignees) > 0 && !assignedToMe {
				continue
			}

			cacheKey := owner + "/" + repo
			if _, ok := cache[cacheKey]; !ok {
				// Repo assignee suggestions are reused across multiple items.
				cache[cacheKey] = fetchRepoAssignees(ctx, token, owner, repo)
			}
			suggestions := append([]AssigneeOption(nil), cache[cacheKey]...)

			out = append(out, MissingAssigneeIssue{
				Item:               it,
				ProjectNum:         projectNum,
				RepoOwner:          owner,
				RepoName:           repo,
				CurrentAssignees:   currentAssignees,
				AssignedToMe:       assignedToMe,
				SuggestedAssignees: suggestions,
			})
		}

		// Include assigned-to-me items found by project search even if outside fetched project item window.
		for key, issue := range searchAssigned {
			if _, ok := seenAssigned[key]; ok {
				continue
			}
			owner, repo := parseRepoFromRepositoryAPIURL(issue.RepositoryURL)
			if owner == "" || repo == "" {
				continue
			}
			item := Item{}
			num, err := toGithubInt(issue.Number)
			if err != nil {
				continue
			}
			// Build a minimal synthetic Item so the downstream report/render path
			// can treat API-search hits and project-item hits uniformly.
			item.Content.Issue.Number = num
			item.Content.Issue.Title = githubv4.String(issue.Title)
			if parsed, err := parseIssueURL(issue.HTMLURL); err == nil {
				item.Content.Issue.URL = githubv4.URI{URL: parsed}
			}
			currentAssignees := make([]string, 0, len(issue.Assignees))
			for _, a := range issue.Assignees {
				login := strings.TrimSpace(a.Login)
				if login == "" {
					continue
				}
				currentAssignees = append(currentAssignees, login)
			}
			out = append(out, MissingAssigneeIssue{
				Item:             item,
				ProjectNum:       projectNum,
				RepoOwner:        owner,
				RepoName:         repo,
				CurrentAssignees: currentAssignees,
				AssignedToMe:     true,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectNum != out[j].ProjectNum {
			return out[i].ProjectNum < out[j].ProjectNum
		}
		return getNumber(out[i].Item) < getNumber(out[j].Item)
	})
	return out
}

// fetchAssignedIssuesByProject runs a GitHub issue search for open tickets
// assigned to the current user in one project, excluding Done status.
func fetchAssignedIssuesByProject(ctx context.Context, token, org string, projectNum int) []searchIssueItem {
	query := fmt.Sprintf(`is:issue is:open project:%s/%d assignee:@me -status:"Done" repo:%s/fleet`, org, projectNum, org)
	endpoint := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100", urlQueryEscape(query))
	body, ok := executeIssueSearchRequest(ctx, endpoint, token)
	if !ok {
		return nil
	}
	return body.Items
}

// fetchViewerLogin resolves the login of the current GitHub token owner.
func fetchViewerLogin(ctx context.Context, token string) string {
	endpoint := "https://api.github.com/user"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	// Only login is needed from this endpoint for assignee:@me comparisons.
	var body userLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}
	return strings.TrimSpace(body.Login)
}

// fetchRepoAssignees returns assignable users for a repo, deduplicated and
// sorted alphabetically for stable UI suggestion lists.
func fetchRepoAssignees(ctx context.Context, token, owner, repo string) []AssigneeOption {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/assignees?per_page=100",
		owner,
		repo,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var users []userLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil
	}

	out := make([]AssigneeOption, 0, len(users))
	seen := make(map[string]bool)
	for _, u := range users {
		// Normalize by lowercase key while preserving original casing for display.
		login := strings.TrimSpace(u.Login)
		if login == "" {
			continue
		}
		key := strings.ToLower(login)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, AssigneeOption{Login: login})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Login) < strings.ToLower(out[j].Login)
	})
	return out
}

// issueAssignees extracts issue assignees from a project item, normalizes and
// de-duplicates them, then returns a stable alphabetical list.
func issueAssignees(it Item) []string {
	out := make([]string, 0, len(it.Content.Issue.Assignees.Nodes))
	seen := make(map[string]bool)
	for _, n := range it.Content.Issue.Assignees.Nodes {
		login := strings.TrimSpace(string(n.Login))
		if login == "" {
			continue
		}
		key := strings.ToLower(login)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, login)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

// containsLogin reports whether the requested login is present in the list,
// using case-insensitive comparison.
func containsLogin(logins []string, wantedLower string) bool {
	if wantedLower == "" {
		return false
	}
	for _, l := range logins {
		if strings.EqualFold(strings.TrimSpace(l), wantedLower) {
			return true
		}
	}
	return false
}
