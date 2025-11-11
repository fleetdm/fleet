package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	owner       = "fleetdm"
	repo        = "fleet"
	project67ID = "PVT_kwDOBWkFJM4ANsqq" // Your Project 67
	project71ID = "PVT_kwDOBWkFJM4ANsq4" // Your Project 71
	statusField = "Status"
	readyLabel  = "READY"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run print_issue.go <issue_number>")
		os.Exit(1)
	}

	issueNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid issue number: %v", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("Please set GITHUB_TOKEN environment variable")
	}

	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)
	// ctx := context.Background()

	var query struct {
		Repository struct {
			Issue struct {
				ID    githubv4.ID
				Title string
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String("fleetdm"),
		"name":   githubv4.String("fleet"),
		"number": githubv4.Int(issueNumber),
	}

	err = client.Query(context.Background(), &query, variables)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Issue #%d: %s\nID: %v\n", issueNumber, query.Repository.Issue.Title, query.Repository.Issue.ID)

	return

	/*


		// Step 1: Get issue metadata
		variables := map[string]interface{}{
			"owner":  githubv4.String(owner),
			"name":   githubv4.String(repo),
			"number": githubv4.Int(issueNumber),
		}
		err := client.Query(ctx, &issueQuery, variables)
		if err != nil {
			log.Fatalf("Fetching issue: %v", err)
		}
		issue := issueQuery.Repository.Issue

		// Prompt before continuing
		fmt.Printf("Move issue \"%s\"?\n%s\nContinue? [y/N]: ", issue.Title, issue.URL)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return
		}

		// Step 2: Remove from project 67
		var itemIDToRemove githubv4.ID
		for _, item := range issue.ProjectItems.Nodes {
			if item.ProjectID == githubv4.ID(project67ID) {
				itemIDToRemove = item.ID
				break
			}
		}
		if itemIDToRemove != "" {
			var deleteMutation struct {
				DeleteProjectV2ItemByID struct {
					DeletedItemID string
				} `graphql:"deleteProjectV2ItemById(input: {itemId: $itemId})"`
			}
			err = client.Mutate(ctx, &deleteMutation, map[string]interface{}{
				"itemId": itemIDToRemove,
			})
			if err != nil {
				log.Fatalf("Deleting from project 67: %v", err)
			}
			fmt.Println("Removed from project 67")
		}

		// Step 3: Add to project 71
		var addMutation struct {
			AddProjectV2ItemByID struct {
				Item struct {
					ID githubv4.ID
				}
			} `graphql:"addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId})"`
		}
		err = client.Mutate(ctx, &addMutation, map[string]interface{}{
			"projectId": githubv4.ID(project71ID),
			"contentId": issue.ID,
		})
		if err != nil {
			log.Fatalf("Adding to project 71: %v", err)
		}
		newItemID := addMutation.AddProjectV2ItemByID.Item.ID
		fmt.Println("Added to project 71")

		// Step 4: Set "Status" to "READY"
		var fieldsQuery struct {
			Node struct {
				ProjectV2 struct {
					Fields struct {
						Nodes []struct {
							ID      githubv4.ID
							Name    string
							Options []struct {
								ID   githubv4.ID
								Name string
							} `graphql:"... on ProjectV2SingleSelectField"`
						}
					} `graphql:"nodes"`
				} `graphql:"... on ProjectV2"`
			} `graphql:"node(id: $projectId)"`
		}
		err = client.Query(ctx, &fieldsQuery, map[string]interface{}{
			"projectId": githubv4.ID(project71ID),
		})
		if err != nil {
			log.Fatalf("Fetching project fields: %v", err)
		}

		var statusFieldID, readyOptionID githubv4.ID
		for _, field := range fieldsQuery.Node.ProjectV2.Fields.Nodes {
			if field.Name == statusField {
				statusFieldID = field.ID
				for _, option := range field.Options {
					if option.Name == readyLabel {
						readyOptionID = option.ID
					}
				}
			}
		}
		if statusFieldID == "" || readyOptionID == "" {
			log.Fatal("Couldn't find 'READY' status option in project 71")
		}

		var updateMutation struct {
			UpdateProjectV2ItemFieldValue struct {
				ProjectV2Item struct {
					ID githubv4.ID
				}
			} `graphql:"updateProjectV2ItemFieldValue(input: {projectId: $projectId, itemId: $itemId, fieldId: $fieldId, value: {singleSelectOptionId: $optionId}})"`
		}
		err = client.Mutate(ctx, &updateMutation, map[string]interface{}{
			"projectId": githubv4.ID(project71ID),
			"itemId":    newItemID,
			"fieldId":   statusFieldID,
			"optionId":  readyOptionID,
		})
		if err != nil {
			log.Fatalf("Setting READY status: %v", err)
		}
		fmt.Println("Moved to READY in project 71")


	*/
}
