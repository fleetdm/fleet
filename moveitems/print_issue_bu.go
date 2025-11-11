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
	owner           = "fleetdm"
	repo            = "fleet"
	project67Number = 67
	project71Number = 71
)

type AddProjectV2ItemByIdInput struct {
	ProjectID githubv4.ID `json:"projectId"`
	ContentID githubv4.ID `json:"contentId"`
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: go run main.go <issue_number>")
	}
	issueNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid issue number: %v", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("Please set the GITHUB_TOKEN environment variable")
	}

	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	// Fetch project IDs for 67 and 71 dynamically
	project67ID, err := fetchProjectID(client, owner, project67Number)
	if err != nil {
		log.Fatalf("Failed to fetch project 67 ID: %v", err)
	}
	project71ID, err := fetchProjectID(client, owner, project71Number)
	if err != nil {
		log.Fatalf("Failed to fetch project 71 ID: %v", err)
	}

	// Fetch issue info
	issueID, issueTitle, issueURL, err := fetchIssue(client, owner, repo, issueNumber)
	if err != nil {
		log.Fatalf("Failed to fetch issue: %v", err)
	}

	fmt.Printf("Issue Title: %s\nIssue URL: %s\n", issueTitle, issueURL)

	// Confirm before moving
	fmt.Print("Move this issue from Project 67 to Project 71? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if strings.ToLower(input) != "y" {
		fmt.Println("Aborted.")
		return
	}

	// Add issue to project 71
	err = addIssueToProject(client, project71ID, issueID)
	if err != nil {
		log.Fatalf("Failed to add issue to Project 71: %v", err)
	}
	fmt.Println("✅ Issue added to Project 71")

	// Remove issue from project 67
	err = removeIssueFromProject(client, project67ID, issueID)
	if err != nil {
		log.Fatalf("Failed to remove issue from Project 67: %v", err)
	}
	fmt.Println("✅ Issue removed from Project 67")
}

func fetchProjectID(client *githubv4.Client, orgLogin string, projectNumber int) (githubv4.ID, error) {
	var query struct {
		Organization struct {
			ProjectV2 struct {
				ID    githubv4.ID
				Title string
			} `graphql:"projectV2(number: $number)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":  githubv4.String(orgLogin),
		"number": githubv4.Int(projectNumber),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}
	return query.Organization.ProjectV2.ID, nil
}

func fetchIssue(client *githubv4.Client, owner, repo string, issueNumber int) (githubv4.ID, string, string, error) {
	var query struct {
		Repository struct {
			Issue struct {
				ID    githubv4.ID
				Title string
				URL   string
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(repo),
		"number": githubv4.Int(issueNumber),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return "", "", "", err
	}
	return query.Repository.Issue.ID, query.Repository.Issue.Title, query.Repository.Issue.URL, nil
}

func addIssueToProject(client *githubv4.Client, projectID, contentID githubv4.ID) error {
	var mutation struct {
		AddProjectV2ItemById struct {
			Item struct {
				ID githubv4.ID
			}
		} `graphql:"addProjectV2ItemById(input: $input)"`
	}

	input := AddProjectV2ItemByIdInput{
		ProjectID: projectID,
		ContentID: contentID,
	}

	return client.Mutate(context.Background(), &mutation, input, nil)
}

func removeIssueFromProject(client *githubv4.Client, projectID, contentID githubv4.ID) error {
	// First fetch the project item ID for this issue in the given project
	var query struct {
		ProjectV2 struct {
			Items struct {
				Nodes []struct {
					ID      githubv4.ID
					Content struct {
						ID githubv4.ID
					}
				}
			} `graphql:"items(first: 100)"`
		} `graphql:"node(id: $projectID)"`
	}

	variables := map[string]interface{}{
		"projectID": githubv4.ID(projectID),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return err
	}

	// Collect all matching items
	var matchedItems []githubv4.ID
	for _, item := range query.ProjectV2.Items.Nodes {
		if item.Content.ID == contentID {
			matchedItems = append(matchedItems, item.ID)
		}
	}

	// Check that there is exactly one matching item
	if len(matchedItems) == 0 {
		return fmt.Errorf("issue not found in project %s", projectID)
	}
	if len(matchedItems) > 1 {
		return fmt.Errorf("multiple project items (%d) found for the issue in project %s, aborting deletion", len(matchedItems), projectID)
	}

	projectItemID := matchedItems[0]

	// Delete the project item
	var mutation struct {
		DeleteProjectV2Item struct {
			DeletedProjectItemID githubv4.ID
		} `graphql:"deleteProjectV2Item(input: $input)"`
	}

	input := struct {
		ItemID githubv4.ID `json:"itemId"`
	}{
		ItemID: projectItemID,
	}

	return client.Mutate(context.Background(), &mutation, input, nil)
}
