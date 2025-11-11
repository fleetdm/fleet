package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	owner          = "fleetdm"
	repo           = "fleet"
	projectFrom    = "PVT_kwDOBWkFJM4ANsqq" // Project 67
	projectTo      = "PVT_kwDOBWkFJM4ANsq4" // Project 71
	readyFieldName = "Status"
	readyOption    = "Ready"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run print_issue.go <issue_number>")
		os.Exit(1)
	}
	num, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid issue number: %v", err)
	}
	issueNumber := githubv4.Int(num)

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	ctx := context.Background()

	// Step 1: Fetch issue info
	var q struct {
		Repository struct {
			Issue struct {
				ID     githubv4.ID
				Title  string
				URL    githubv4.URI
				Author struct {
					Login string
				}
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(repo),
		"number": issueNumber,
	}
	err = client.Query(ctx, &q, variables)
	if err != nil {
		log.Fatalf("Failed to get issue: %v", err)
	}

	issue := q.Repository.Issue
	fmt.Printf("Issue #%d: %s\nURL: %s\nAuthor: %s\n", issueNumber, issue.Title, issue.URL, issue.Author.Login)

	// Step 2: Confirm
	fmt.Print("Move this issue to Project 71 (READY)? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(answer)) != "y" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	// Step 3: Add to Project 71
	var addItem struct {
		AddProjectV2ItemByID struct {
			Item struct {
				ID githubv4.ID
			}
		} `graphql:"addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId})"`
	}
	err = client.Mutate(ctx, &addItem, nil, map[string]interface{}{
		"projectId": githubv4.ID(projectTo),
		"contentId": issue.ID,
	})
	if err != nil {
		log.Fatalf("Failed to add issue to Project 71: %v", err)
	}
	itemID := addItem.AddProjectV2ItemByID.Item.ID

	// Step 4: Get fields for Project 71
	var fieldQuery struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						ID         githubv4.ID
						Name       string
						__Typename githubv4.String `graphql:"__typename"`
						Options    []struct {
							ID   githubv4.ID
							Name string
						} `graphql:"... on ProjectV2SingleSelectFieldConfiguration"`
					}
				} `graphql:"fields(first: 20)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}
	err = client.Query(ctx, &fieldQuery, map[string]interface{}{
		"id": githubv4.ID(projectTo),
	})
	if err != nil {
		log.Fatalf("Failed to get fields for Project 71: %v", err)
	}

	var readyFieldID githubv4.ID
	var readyOptionID githubv4.ID
	for _, field := range fieldQuery.Node.ProjectV2.Fields.Nodes {
		if field.Name == readyFieldName {
			readyFieldID = field.ID
			for _, option := range field.Options {
				if option.Name == readyOption {
					readyOptionID = option.ID
					break
				}
			}
			break
		}
	}
	if readyFieldID == "" || readyOptionID == "" {
		log.Fatalf("Failed to find READY field/option")
	}

	// Step 5: Set field to READY
	var setField struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID githubv4.ID
			}
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}
	err = client.Mutate(ctx, &setField, nil, map[string]interface{}{
		"input": githubv4.UpdateProjectV2ItemFieldValueInput{
			ProjectID: githubv4.ID(projectTo),
			ItemID:    itemID,
			FieldID:   readyFieldID,
			Value: githubv4.ProjectV2FieldValue{
				SingleSelectOptionID: &githubv4.String(readyOptionID),
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to set READY field: %v", err)
	}

	fmt.Println("✅ Issue moved to Project 71 in READY column.")
}
