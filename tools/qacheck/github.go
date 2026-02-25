package main

import (
	"context"
	"log"

	"github.com/shurcooL/githubv4"
)

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
		"num": githubv4.Int(num),
	})
	if err != nil {
		log.Fatalf("project query failed: %v", err)
	}

	return q.Organization.ProjectV2.ID
}

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
		"first": githubv4.Int(limit),
	})
	if err != nil {
		log.Fatalf("items query failed: %v", err)
	}

	if len(q.Node.ProjectV2.Items.Nodes) == limit {
		log.Printf(
			"NOTE: scanned %d items (limit reached, no pagination by design). Increase -limit if needed.",
			limit,
		)
	}

	return q.Node.ProjectV2.Items.Nodes
}

func getBody(it Item) string {
	if it.Content.Issue.Number != 0 {
		return string(it.Content.Issue.Body)
	}
	return string(it.Content.PullRequest.Body)
}

func getTitle(it Item) string {
	if it.Content.Issue.Number != 0 {
		return string(it.Content.Issue.Title)
	}
	return string(it.Content.PullRequest.Title)
}

func getNumber(it Item) int {
	if it.Content.Issue.Number != 0 {
		return int(it.Content.Issue.Number)
	}
	return int(it.Content.PullRequest.Number)
}

func getURL(it Item) string {
	if it.Content.Issue.Number != 0 {
		return it.Content.Issue.URL.String()
	}
	return it.Content.PullRequest.URL.String()
}
