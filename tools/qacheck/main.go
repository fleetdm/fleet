package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	awaitingQAColumn = "✔️Awaiting QA"
	checkText        = "Engineer: Added comment to user story confirming successful completion of test plan."

	// Drafting board (Project 67) check:
	draftingProjectNum   = 67
	draftingStatusNeedle = "Ready to estimate,Estimated"
	testPlanFinalized    = "Test plan is finalized"
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
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
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

	// Check 1: items in ✔️Awaiting QA with the engineer test-plan confirmation line still unchecked.
	projectID := fetchProjectID(ctx, client, *org, *projectNum)
	items := fetchItems(ctx, client, projectID, *limit)

	var badAwaitingQA []Item
	for _, it := range items {
		if !inAwaitingQA(it) {
			continue
		}
		if hasUncheckedChecklistLine(getBody(it), checkText) {
			badAwaitingQA = append(badAwaitingQA, it)
		}
	}

	fmt.Printf("\nFound %d items in %q with UNCHECKED test-plan confirmation:\n\n", len(badAwaitingQA), awaitingQAColumn)
	for _, it := range badAwaitingQA {
		fmt.Printf("❌ #%d – %s\n   %s\n\n", getNumber(it), getTitle(it), getURL(it))
	}

	// Check 2: drafting board (project 67) items in Ready to estimate / Estimated with "Test plan is finalized" unchecked.
	draftingProjectID := fetchProjectID(ctx, client, *org, draftingProjectNum)
	draftingItems := fetchItems(ctx, client, draftingProjectID, *limit)

	needles := strings.Split(draftingStatusNeedle, ",")
	var badDrafting []Item
	for _, it := range draftingItems {
		if !inAnyStatus(it, needles) {
			continue
		}
		if hasUncheckedChecklistLine(getBody(it), testPlanFinalized) {
			badDrafting = append(badDrafting, it)
		}
	}

	fmt.Printf("\nFound %d items in Drafting (project %d) in Ready to estimate / Estimated with UNCHECKED %q:\n\n", len(badDrafting), draftingProjectNum, testPlanFinalized)
	for _, it := range badDrafting {
		fmt.Printf("❌ #%d – %s\n   %s\n\n", getNumber(it), getTitle(it), getURL(it))
	}
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

func inAwaitingQA(it Item) bool {
	for _, v := range it.FieldValues.Nodes {
		if string(v.SingleSelectValue.Name) == awaitingQAColumn {
			return true
		}
	}
	return false
}

func inAnyStatus(it Item, needles []string) bool {
	for _, v := range it.FieldValues.Nodes {
		name := normalizeStatusName(string(v.SingleSelectValue.Name))
		for _, n := range needles {
			needle := strings.ToLower(strings.TrimSpace(n))
			if needle == "" {
				continue
			}
			if strings.Contains(name, needle) {
				return true
			}
		}
	}
	return false
}

// Remove leading emojis/symbols so we can match status names even if the project uses icons.
func normalizeStatusName(s string) string {
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			break
		}
		s = strings.TrimSpace(s[size:])
	}
	return strings.ToLower(s)
}

// Only flag if the unchecked checklist line exists.
// Ignore if missing or checked.
func hasUncheckedChecklistLine(body string, text string) bool {
	if body == "" || text == "" {
		return false
	}

	unchecked1 := "- [ ] " + text
	unchecked2 := "[ ] " + text

	checked := []string{
		"- [x] " + text,
		"- [X] " + text,
		"[x] " + text,
		"[X] " + text,
	}

	for _, c := range checked {
		if strings.Contains(body, c) {
			return false
		}
	}

	return strings.Contains(body, unchecked1) || strings.Contains(body, unchecked2)
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
