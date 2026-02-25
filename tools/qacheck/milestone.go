package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type milestoneResponse struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

func runMissingMilestoneChecks(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNums []int,
	limit int,
	token string,
) []MissingMilestoneIssue {
	cache := make(map[string][]MilestoneOption)
	out := make([]MissingMilestoneIssue, 0)
	for _, projectNum := range projectNums {
		projectID := fetchProjectID(ctx, client, org, projectNum)
		items := fetchItems(ctx, client, projectID, limit)

		for _, it := range items {
			if it.Content.Issue.Number == 0 {
				continue
			}
			if inDoneColumn(it) {
				continue
			}
			if strings.TrimSpace(string(it.Content.Issue.Milestone.Title)) != "" {
				continue
			}

			owner, repo := parseRepoFromIssueURL(getURL(it))
			if owner == "" || repo == "" {
				continue
			}
			cacheKey := owner + "/" + repo
			if _, ok := cache[cacheKey]; !ok {
				cache[cacheKey] = fetchAllMilestones(ctx, token, owner, repo)
			}
			suggestions := append([]MilestoneOption(nil), cache[cacheKey]...)

			out = append(out, MissingMilestoneIssue{
				Item:                it,
				ProjectNum:          projectNum,
				RepoOwner:           owner,
				RepoName:            repo,
				SuggestedMilestones: suggestions,
			})
		}
	}
	return out
}

func parseRepoFromIssueURL(issueURL string) (string, string) {
	u, err := url.Parse(issueURL)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(strings.Trim(path.Clean(u.Path), "/"), "/")
	if len(parts) < 4 {
		return "", ""
	}
	if parts[2] != "issues" && parts[2] != "pull" {
		return "", ""
	}
	return parts[0], parts[1]
}

func fetchAllMilestones(ctx context.Context, token, owner, repo string) []MilestoneOption {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/milestones?state=all&sort=due_on&direction=desc&per_page=100",
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

	var milestones []milestoneResponse
	if err := json.NewDecoder(resp.Body).Decode(&milestones); err != nil {
		return nil
	}

	out := make([]MilestoneOption, 0, len(milestones))
	seen := make(map[string]bool)
	for _, m := range milestones {
		title := strings.TrimSpace(m.Title)
		if title == "" || seen[title] {
			continue
		}
		seen[title] = true
		out = append(out, MilestoneOption{Number: m.Number, Title: title})
	}
	return out
}
