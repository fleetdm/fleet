package main

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/shurcooL/githubv4"
)

// fetchProjectID looks up a GitHub ProjectV2 node ID from org + project number.
// On query failure it logs and returns an empty ID instead of terminating.
func fetchProjectID(ctx context.Context, client *githubv4.Client, org string, num int) githubv4.ID {
	var q struct {
		Organization struct {
			ProjectV2 struct {
				ID githubv4.ID
			} `graphql:"projectV2(number: $num)"`
		} `graphql:"organization(login: $org)"`
	}

	err := client.Query(ctx, &q, map[string]interface{}{
		"org": githubv4.String(org),
		"num": mustGithubInt(num),
	})
	if err != nil {
		log.Printf("project query failed: %v", err)
		return ""
	}

	return q.Organization.ProjectV2.ID
}

// fetchItems loads up to limit items from one ProjectV2 by ID.
// On query failure it logs and returns no items instead of terminating.
func fetchItems(
	ctx context.Context,
	client *githubv4.Client,
	projectID githubv4.ID,
	limit int,
) []Item {
	var q struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []Item
				} `graphql:"items(first: $first)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	err := client.Query(ctx, &q, map[string]interface{}{
		"id":    projectID,
		"first": mustGithubInt(limit),
	})
	if err != nil {
		log.Printf("items query failed: %v", err)
		return nil
	}

	return q.Node.ProjectV2.Items.Nodes
}

// mustGithubInt converts int to githubv4.Int with explicit int32 bounds
// validation and falls back to zero when out of range.
func mustGithubInt(v int) githubv4.Int {
	converted, err := toGithubInt(v)
	if err != nil {
		log.Printf("integer conversion failed: %v", err)
		return 0
	}
	return converted
}

// toGithubInt converts int to githubv4.Int with bounds checks and returns an
// error instead of terminating on invalid values.
func toGithubInt(v int) (githubv4.Int, error) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, fmt.Errorf("integer %d out of range for githubv4.Int", v)
	}
	return githubv4.Int(v), nil
}

// fetchAllItems paginates through all items in a project and returns the full
// list. This is used where partial "first N" fetch is not enough.
// On a page query error it logs and returns what was gathered so far.
func fetchAllItems(
	ctx context.Context,
	client *githubv4.Client,
	projectID githubv4.ID,
) []Item {
	type fieldNode struct {
		ProjectV2 struct {
			Items struct {
				Nodes    []Item
				PageInfo struct {
					HasNextPage githubv4.Boolean `graphql:"hasNextPage"`
					EndCursor   githubv4.String  `graphql:"endCursor"`
				} `graphql:"pageInfo"`
			} `graphql:"items(first: 100, after: $after)"`
		} `graphql:"... on ProjectV2"`
	}
	var q struct {
		Node fieldNode `graphql:"node(id: $id)"`
	}

	out := make([]Item, 0, 256)
	var after *githubv4.String
	for {
		vars := map[string]interface{}{
			"id":    projectID,
			"after": after,
		}
		if err := client.Query(ctx, &q, vars); err != nil {
			log.Printf("items paged query failed: %v", err)
			break
		}
		nodes := q.Node.ProjectV2.Items.Nodes
		out = append(out, nodes...)
		if !bool(q.Node.ProjectV2.Items.PageInfo.HasNextPage) {
			break
		}
		cursor := q.Node.ProjectV2.Items.PageInfo.EndCursor
		after = &cursor
	}
	return out
}

// getBody returns the issue/pr body string from a project item.
func getBody(it Item) string {
	if it.Content.Issue.Number != 0 {
		return string(it.Content.Issue.Body)
	}
	return string(it.Content.PullRequest.Body)
}

// getTitle returns the issue/pr title from a project item.
func getTitle(it Item) string {
	if it.Content.Issue.Number != 0 {
		return string(it.Content.Issue.Title)
	}
	return string(it.Content.PullRequest.Title)
}

// getNumber returns the issue/pr number from a project item.
func getNumber(it Item) int {
	if it.Content.Issue.Number != 0 {
		return int(it.Content.Issue.Number)
	}
	return int(it.Content.PullRequest.Number)
}

// getURL returns the issue/pr URL string from a project item.
func getURL(it Item) string {
	if it.Content.Issue.Number != 0 {
		return it.Content.Issue.URL.String()
	}
	return it.Content.PullRequest.URL.String()
}
