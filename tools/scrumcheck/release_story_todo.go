package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/shurcooL/githubv4"
)

const storyLabel = "story"

// runReleaseStoryTODOChecks fetches story/:release issues containing TODO in the
// body for each selected project and normalizes them into report items.
func runReleaseStoryTODOChecks(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNums []int,
	limit int,
	token string,
	labelFilter map[string]struct{},
) []ReleaseStoryTODOIssue {
	_ = client
	_ = limit
	_ = labelFilter

	keyed := make(map[string]ReleaseStoryTODOIssue)
	out := make([]ReleaseStoryTODOIssue, 0)
	for _, projectNum := range projectNums {
		// Query each selected project independently so results can stay grouped
		// by project in report output.
		issues := fetchReleaseStoryTODOByProject(ctx, token, org, projectNum)
		for _, issue := range issues {
			owner, repo := parseRepoFromRepositoryAPIURL(issue.RepositoryURL)
			if owner == "" || repo == "" {
				continue
			}
			// Deduplicate by project+repo+issue because search results can overlap
			// across repeated refreshes.
			k := fmt.Sprintf("%d:%s/%s#%d", projectNum, owner, repo, issue.Number)
			if _, ok := keyed[k]; ok {
				continue
			}

			labels := make([]string, 0, len(issue.Labels))
			for _, l := range issue.Labels {
				if l.Name != "" {
					labels = append(labels, l.Name)
				}
			}

			item := Item{}
			num, err := toGithubInt(issue.Number)
			if err != nil {
				continue
			}
			// Build a minimal Item wrapper so existing report rendering can reuse
			// the same helper functions used by project-sourced items.
			item.Content.Issue.Number = num
			item.Content.Issue.Title = githubv4.String(issue.Title)
			item.Content.Issue.Body = githubv4.String(issue.Body)
			if parsed, err := parseIssueURL(issue.HTMLURL); err == nil {
				item.Content.Issue.URL = githubv4.URI{URL: parsed}
			}

			keyed[k] = ReleaseStoryTODOIssue{
				Item:          item,
				ProjectNum:    projectNum,
				RepoOwner:     owner,
				RepoName:      repo,
				Status:        "Open",
				CurrentLabels: labels,
				BodyPreview:   previewBodyLines(issue.Body, 4),
			}
		}
	}
	for _, v := range keyed {
		out = append(out, v)
	}

	// Keep deterministic ordering: project first, then issue number.
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectNum != out[j].ProjectNum {
			return out[i].ProjectNum < out[j].ProjectNum
		}
		return getNumber(out[i].Item) < getNumber(out[j].Item)
	})
	return out
}

// fetchReleaseStoryTODOByProject runs the GitHub search query for one project.
func fetchReleaseStoryTODOByProject(ctx context.Context, token, org string, projectNum int) []searchIssueItem {
	query := fmt.Sprintf(`is:open is:issue label:%s todo in:body label:%s project:%s/%d repo:%s/fleet`, storyLabel, releaseLabel, org, projectNum, org)
	endpoint := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100", urlQueryEscape(query))
	body, ok := executeIssueSearchRequest(ctx, endpoint, token)
	if !ok {
		return nil
	}
	return body.Items
}
