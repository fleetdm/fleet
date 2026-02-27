package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type searchIssueResponse struct {
	Items []searchIssueItem `json:"items"`
}

type searchIssueItem struct {
	Number        int    `json:"number"`
	Title         string `json:"title"`
	HTMLURL       string `json:"html_url"`
	State         string `json:"state"`
	RepositoryURL string `json:"repository_url"`
	Body          string `json:"body"`
	Assignees     []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

var searchUnreleasedIssuesByGroup = fetchUnreleasedIssuesByGroup

// runUnassignedUnreleasedBugChecks finds unreleased bugs by group label and
// marks whether each item is assigned or unassigned.
func runUnassignedUnreleasedBugChecks(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNums []int,
	limit int,
	token string,
	labelFilter map[string]struct{},
	groupLabels []string,
) []UnassignedUnreleasedBugIssue {
	// This check is scoped to provided group labels.
	if len(labelFilter) == 0 || len(groupLabels) == 0 {
		return nil
	}

	_ = client
	_ = projectNums
	_ = limit

	keyed := make(map[string]UnassignedUnreleasedBugIssue)
	for _, group := range groupLabels {
		// Query once per configured group so the report can preserve group-based
		// categorization while still deduplicating shared issues.
		issues := searchUnreleasedIssuesByGroup(ctx, token, org, group)
		for _, issue := range issues {
			owner, repo := parseRepoFromRepositoryAPIURL(issue.RepositoryURL)
			if owner == "" || repo == "" {
				continue
			}
			labels := make([]string, 0, len(issue.Labels))
			for _, label := range issue.Labels {
				name := strings.TrimSpace(label.Name)
				if name == "" {
					continue
				}
				labels = append(labels, name)
			}
			currentAssignees := make([]string, 0, len(issue.Assignees))
			for _, a := range issue.Assignees {
				login := strings.TrimSpace(a.Login)
				if login == "" {
					continue
				}
				currentAssignees = append(currentAssignees, login)
			}

			status := strings.Title(strings.ToLower(strings.TrimSpace(issue.State)))
			k := fmt.Sprintf("%s/%s#%d", owner, repo, issue.Number)
			if existing, ok := keyed[k]; ok {
				// Same issue can match multiple groups; merge group membership.
				if !containsNormalized(existing.MatchingGroups, group) {
					existing.MatchingGroups = append(existing.MatchingGroups, normalizeLabelName(group))
					sort.Strings(existing.MatchingGroups)
				}
				keyed[k] = existing
				continue
			}

			item := Item{}
			num, err := toGithubInt(issue.Number)
			if err != nil {
				continue
			}
			// Build a compact Item envelope for compatibility with shared helpers
			// (title/url/number extraction in report rendering).
			item.Content.Issue.Number = num
			item.Content.Issue.Title = githubv4.String(issue.Title)
			if parsed, err := parseIssueURL(issue.HTMLURL); err == nil {
				item.Content.Issue.URL = githubv4.URI{URL: parsed}
			}
			keyed[k] = UnassignedUnreleasedBugIssue{
				Item:             item,
				ProjectNum:       0,
				RepoOwner:        owner,
				RepoName:         repo,
				Status:           status,
				CurrentLabels:    labels,
				CurrentAssignees: currentAssignees,
				Unassigned:       len(currentAssignees) == 0,
				MatchingGroups:   []string{normalizeLabelName(group)},
			}
		}
	}

	out := make([]UnassignedUnreleasedBugIssue, 0, len(keyed))
	for _, issue := range keyed {
		out = append(out, issue)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RepoOwner != out[j].RepoOwner {
			return out[i].RepoOwner < out[j].RepoOwner
		}
		if out[i].RepoName != out[j].RepoName {
			return out[i].RepoName < out[j].RepoName
		}
		return getNumber(out[i].Item) < getNumber(out[j].Item)
	})
	return out
}

// issueMatchingGroups returns which configured group labels exist on an issue.
func issueMatchingGroups(labels []string, groupLabels []string) []string {
	if len(labels) == 0 || len(groupLabels) == 0 {
		return nil
	}
	labelSet := make(map[string]struct{}, len(labels))
	for _, l := range labels {
		norm := normalizeLabelName(l)
		if norm == "" {
			continue
		}
		labelSet[norm] = struct{}{}
	}
	out := make([]string, 0, len(groupLabels))
	for _, g := range groupLabels {
		if _, ok := labelSet[normalizeLabelName(g)]; ok {
			out = append(out, normalizeLabelName(g))
		}
	}
	return out
}

// fetchUnreleasedIssuesByGroup runs the unreleased-bug search for one group.
func fetchUnreleasedIssuesByGroup(ctx context.Context, token, org, groupLabel string) []struct {
	Number        int    `json:"number"`
	Title         string `json:"title"`
	HTMLURL       string `json:"html_url"`
	State         string `json:"state"`
	RepositoryURL string `json:"repository_url"`
	Body          string `json:"body"`
	Assignees     []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
} {
	groupNorm := normalizeLabelName(groupLabel)
	if groupNorm == "" {
		return nil
	}

	queryLabel := "#" + groupNorm
	seen := map[string]struct{}{}
	out := make([]struct {
		Number        int    `json:"number"`
		Title         string `json:"title"`
		HTMLURL       string `json:"html_url"`
		State         string `json:"state"`
		RepositoryURL string `json:"repository_url"`
		Body          string `json:"body"`
		Assignees     []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}, 0, 32)

	for page := 1; page <= 10; page++ {
		// GitHub label name includes the leading "~" for this label.
		query := fmt.Sprintf(`org:%s is:issue is:open label:bug label:"~%s" label:"%s"`, org, unreleasedBugLabel, queryLabel)
		endpoint := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100&page=%d", urlQueryEscape(query), page)
		body, ok := executeIssueSearchRequest(ctx, endpoint, token)
		if !ok {
			break
		}

		if len(body.Items) == 0 {
			break
		}
		for _, it := range body.Items {
			// Defensive dedupe across pages.
			key := fmt.Sprintf("%s#%d", it.RepositoryURL, it.Number)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, it)
		}
		if len(body.Items) < 100 {
			break
		}
	}
	return out
}

// executeIssueSearchRequest executes GitHub issue search with retry on 403.
func executeIssueSearchRequest(ctx context.Context, endpoint, token string) (searchIssueResponse, bool) {
	respBody, ok := executeIssueSearchRequestOnce(ctx, endpoint, token)
	if ok {
		return respBody, true
	}
	if token == "" {
		return searchIssueResponse{}, false
	}
	// Some fine-grained tokens can fail for search while public unauthenticated search still works.
	// Retry once without Authorization so public repo searches can still succeed.
	return executeIssueSearchRequestOnce(ctx, endpoint, "")
}

// executeIssueSearchRequestOnce executes a single GitHub issue search request.
func executeIssueSearchRequestOnce(ctx context.Context, endpoint, token string) (searchIssueResponse, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return searchIssueResponse{}, false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		log.Printf("unreleased search request failed: %v", err)
		return searchIssueResponse{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		log.Printf("unreleased search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		return searchIssueResponse{}, false
	}
	var decoded searchIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		log.Printf("unreleased search decode failed: %v", err)
		return searchIssueResponse{}, false
	}
	return decoded, true
}

// parseRepoFromRepositoryAPIURL parses owner/repo from repository API URLs.
func parseRepoFromRepositoryAPIURL(repositoryURL string) (string, string) {
	parts := strings.Split(strings.TrimSpace(repositoryURL), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[len(parts)-2], parts[len(parts)-1]
}

// parseIssueURL parses and validates a GitHub issue URL.
func parseIssueURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

// urlQueryEscape URL-escapes the search query string.
func urlQueryEscape(s string) string { return url.QueryEscape(s) }

// hasLabel reports whether the label list contains the target label.
func hasLabel(labels []string, wanted string) bool {
	for _, label := range labels {
		if normalizeLabelName(label) == normalizeLabelName(wanted) {
			return true
		}
	}
	return false
}

// containsNormalized compares labels using normalized name matching.
func containsNormalized(values []string, wanted string) bool {
	w := normalizeLabelName(wanted)
	for _, v := range values {
		if normalizeLabelName(v) == w {
			return true
		}
	}
	return false
}
