package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	awaitingQAColumn = "✔️Awaiting QA"
	checkText        = "Engineer: Added comment to user story confirming successful completion of test plan."
)

type Item struct {
	ID githubv4.ID

	Content struct {
		Issue struct {
			Number githubv4.Int
			Title  githubv4.String
			Body   githubv4.String
			URL    githubv4.URI
		} `graphql:"... on Issue"`

		PullRequest struct {
			Number githubv4.Int
			Title  githubv4.String
			Body   githubv4.String
			URL    githubv4.URI
		} `graphql:"... on PullRequest"`
	} `graphql:"content"`

	FieldValues struct {
		Nodes []struct {
			SingleSelectValue struct {
				Name githubv4.String
			} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
		}
	} `graphql:"fieldValues(first: 20)"`
}

func main() {
	org := flag.String("org", "", "GitHub org")
	projectNum := flag.Int("project", 0, "Project number")
	flag.Parse()

	if *org == "" || *projectNum == 0 {
		log.Fatal("org and project are required")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN env var is required")
	}

	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := githubv4.NewClient(oauth2.NewClient(ctx, src))

	projectID := fetchProjectID(ctx, client, *org, *projectNum)
	items := fetchItems(ctx, client, projectID)

	var bad []Item

	for _, it := range items {
		if !inAwaitingQA(it) {
			continue
		}

		body := getBody(it)
		if hasUncheckedTestPlanLine(body) {
			bad = append(bad, it)
		}
	}

	fmt.Printf("\nFound %d items in %q with UNCHECKED test-plan confirmation:\n\n", len(bad), awaitingQAColumn)
	for _, it := range bad {
		fmt.Printf("❌ #%d – %s\n   %s\n\n",
			getNumber(it),
			getTitle(it),
			getURL(it),
		)
	}
}

// Only match if the unchecked line is present.
// Ignore if missing or checked.
func hasUncheckedTestPlanLine(body string) bool {
	if body == "" {
		return false
	}

	// Accept both styles:
	// - [ ] Engineer: ...
	// [ ] Engineer: ...
	u1 := "- [ ] " + checkText
	u2 := "[ ] " + checkText

	// Checked variants to ignore (even if someone has both, checked wins for safety)
	c1 := "- [x] " + checkText
	c2 := "- [X] " + checkText
	c3 := "[x] " + checkText
	c4 := "[X] " + checkText

	// If it's checked anywhere, ignore.
	if strings.Contains(body, c1) || strings.Contains(body, c2) || strings.Contains(body, c3) || strings.Contains(body, c4) {
		return false
	}

	// Only flag if the unchecked line exists.
	return strings.Contains(body, u1) || strings.Contains(body, u2)
}

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

func fetchItems(ctx context.Context, client *githubv4.Client, projectID githubv4.ID) []Item {
	var q struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []Item
				} `graphql:"items(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	err := client.Query(ctx, &q, map[string]interface{}{
		"id": projectID,
	})
	if err != nil {
		log.Fatalf("items query failed: %v", err)
	}

	return q.Node.ProjectV2.Items.Nodes
}

func inAwaitingQA(it Item) bool {
	for _, v := range it.FieldValues.Nodes {
		if v.SingleSelectValue.Name == awaitingQAColumn {
			return true
		}
	}
	return false
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
