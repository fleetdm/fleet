package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	owner           = "fleetdm"
	repo            = "fleet"
	project67Number = 67
)

type AddProjectV2ItemByIdInput struct {
	ProjectID githubv4.ID `json:"projectId"`
	ContentID githubv4.ID `json:"contentId"`
}

func main() {
	if len(os.Args) < 5 {
		log.Fatalf("Usage: go run main.go <project_number> <milestone> <issue_number> <username 1 or more>")
	}
	ctx := context.Background()

	// NEW: target project number as CLI arg (replaces hardcoded 71)
	targetProjectNumber, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid project number: %v", err)
	}

	milestone := os.Args[2]
	issueNumber, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatalf("Invalid issue number: %v", err)
	}
	assignees := os.Args[4:]

	// Map real names to GitHub logins
	realNameToLogin := map[string]string{
		"da":      "dantecatalfamo",
		"dante":   "dantecatalfamo",
		"er":      "eashaw",
		"eric":    "eashaw",
		"vi":      "getvictor",
		"victor":  "getvictor",
		"ja":      "jacobshandling",
		"jacob":   "jacobshandling",
		"ju":      "juan-fdz-hawa",
		"juan":    "juan-fdz-hawa",
		"lu":      "lucasmrod",
		"lucas":   "lucasmrod",
		"ra":      "rachaelshaw",
		"rachael": "rachaelshaw",
		"re":      "xpkoala",
		"reed":    "xpkoala",
		"sh":      "sharon-fdm",
		"sharon":  "sharon-fdm",
		"sc":      "sgress454",
		"ia":      "iansltx",
		"ni":      "nulmete",
		"ti":      "mostlikelee",
		"ko":      "ksykulev",
		"scott":   "sgress454",
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("Please set the GITHUB_TOKEN environment variable")
	}

	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	// Fetch target project ID dynamically (was: project71Number)
	project71ID, err := fetchProjectID(client, owner, targetProjectNumber)
	if err != nil {
		log.Fatalf("Failed to fetch project %d ID: %v", targetProjectNumber, err)
	}

	// project67ID, err := fetchProjectID(client, owner, project67Number)
	// if err != nil {
	// 	log.Fatalf("Failed to fetch project 71 ID: %v", err)
	// }

	// Fetch issue info
	issueID, issueTitle, issueURL, err := fetchIssue(client, owner, repo, issueNumber)
	if err != nil {
		log.Fatalf("Failed to fetch issue: %v", err)
	}

	fmt.Printf("🔧 Issue Title: %s\n🔧 Issue URL: %s\n", issueTitle, issueURL)
	// fmt.Print("🔧 Adding this issue from Project 67 to Project 71 ... \n")
	// fmt.Printf("🔧 Debug Info:  project71ID: %s, project67ID: %s, issueID: %s\n", project71ID, project67ID, issueID)

	// Add issue to target project (was: Project 71)
	err = addIssueToProject(client, project71ID, issueID)
	if err != nil {
		log.Fatalf("Failed to add issue to Project %d: %v", targetProjectNumber, err)
	}
	fmt.Printf("✅ Issue added to Project %d \n", targetProjectNumber)

	//_ = printIssueDetails(client, issueID)

	// Remove all current assignees before assigning new user
	err = removeAllAssignees(token, issueNumber)
	if err != nil {
		fmt.Println("❌ Failed to remove current assignees: " + err.Error())
	} else {
		fmt.Println("✅ Removed all current assignees")
	}

	for _, name := range assignees {
		login, ok := realNameToLogin[strings.ToLower(name)]
		if !ok {
			login = name
		}

		fmt.Printf("👤 Assigning GitHub user %s to issue #%d...\n", login, issueNumber)
		err := assignUserToIssue(token, client, issueID, login)
		if err != nil {
			fmt.Printf("❌ Failed to assign %s: %v\n", login, err)
		} else {
			fmt.Printf("✅ Assigned %s successfully.\n", login)
		}
	}

	estimate, err := getEstimateFromProject67(token, issueNumber)
	if err != nil {
		fmt.Println("❌ Error getting Estimate from Draft project : " + err.Error())
	} else {
		// UPDATED: pass target project number instead of hardcoded 71
		err = setEstimateInProject71(issueNumber, targetProjectNumber, estimate)
		if err != nil {
			fmt.Println("❌ Error Setting Estimate from Draft project : " + err.Error())
		}
		fmt.Printf("✅ Estimation in Project %d set to: %.1f\n", targetProjectNumber, estimate)
	}

	err = setIssueMilestone(token, client, issueID, milestone)
	if err != nil {
		log.Fatalf("Error setting milestone: %v", err)
	}
	fmt.Printf("✅ Milestone set to %s\n", milestone)

	if err := removeIssueFromProjectWithCurl(ctx, issueNumber, project67Number); err != nil {
		fmt.Println("Error:", err)
	}

	// Update labels: remove ":product" and add ":release"
	err = updateIssueLabels(token, issueNumber)
	if err != nil {
		fmt.Println("❌ Error updating labels: " + err.Error())
	} else {
		fmt.Println("✅ Labels updated: removed ':product', added ':release'")
	}

	return
}

func fetchProjectID(client *githubv4.Client, orgLogin string, projectNumber int) (githubv4.ID, error) {
	var query struct {
		Organization struct {
			ProjectV2 struct {
				ID githubv4.ID
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

/*func printIssueDetails(client *githubv4.Client, issueID githubv4.ID) error {
	var query struct {
		Node struct {
			Issue struct {
				Number int
				Title  string
				URL    string
			} `graphql:"... on Issue"`
		} `graphql:"node(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": issueID,
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return fmt.Errorf("failed to fetch issue details: %v", err)
	}

	fmt.Printf("🧾 Issue #%d\n🔗 %s\n📝 %s\n", query.Node.Issue.Number, query.Node.Issue.URL, query.Node.Issue.Title)
	return nil
}*/

func confirmAssignee(client *githubv4.Client, issueID, expectedUserID githubv4.ID) error {
	var query struct {
		Node struct {
			Issue struct {
				Assignees struct {
					Nodes []struct {
						ID    githubv4.ID
						Login string
					}
				} `graphql:"assignees(first: 10)"`
			} `graphql:"... on Issue"`
		} `graphql:"node(id: $id)"`
	}

	err := client.Query(context.Background(), &query, map[string]interface{}{
		"id": issueID,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch assignees: %v", err)
	}

	fmt.Println("🔍 Assignees after mutation:")
	for _, a := range query.Node.Issue.Assignees.Nodes {
		//		fmt.Printf(" - %s (%s)\n", a.Login, a.ID)
		if a.ID == expectedUserID {
			//			fmt.Println("✅ Assignment confirmed.")
			return nil
		}
	}

	return fmt.Errorf("❌ Assignment not confirmed — user is not listed as assignee")
}

func fetchUserID(client *githubv4.Client, login string) (githubv4.ID, error) {
	var query struct {
		User struct {
			ID githubv4.ID // This is the new global Relay ID (starts with "U_")
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(login),
	}

	err := client.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}

	// Print the user ID for debugging:
	// fmt.Printf("Fetched user ID for %s: %s\n", login, query.User.ID)

	return query.User.ID, nil
}

func assignUserToIssue(token string, client *githubv4.Client, issueID githubv4.ID, assigneeLogin string) error {
	// token := os.Getenv("GITHUB_TOKEN")
	// if token == "" {
	// 	return fmt.Errorf("GITHUB_TOKEN is not set")
	// }

	//	fmt.Println("🔍 Fetching user global ID via GraphQL REST API...")

	// Build GraphQL query to get next_global_id
	queryPayload := fmt.Sprintf(`{
      "query": "query($login: String!) { user(login: $login) { id login databaseId } }",
      "variables": {
        "login": "%s"
      }
    }`, assigneeLogin)

	// Run curl to get the user ID
	cmd := exec.Command("curl",
		"-s",
		"-X", "POST",
		"-H", "Authorization: bearer "+token,
		"-H", "Content-Type: application/json",
		"https://api.github.com/graphql",
		"-d", queryPayload)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run curl for user ID: %v", err)
	}

	// Parse the output JSON to get the user ID
	var result struct {
		Data struct {
			User struct {
				ID         string `json:"id"`
				Login      string `json:"login"`
				DatabaseID int    `json:"databaseId"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse user ID JSON: %v", err)
	}

	userID := result.Data.User.ID
	if userID == "" {
		return fmt.Errorf("user ID not found for login %s", assigneeLogin)
	}

	//	fmt.Printf("✅ Fetched user global ID: %s\n", userID)

	// Build GraphQL mutation payload to assign
	mutationPayload := fmt.Sprintf(`{
      "query": "mutation($issueId: ID!, $assigneeIds: [ID!]!) { addAssigneesToAssignable(input: {assignableId: $issueId, assigneeIds: $assigneeIds}) { assignable { assignees(first: 10) { nodes { login id } } } } }",
      "variables": {
        "issueId": "%s",
        "assigneeIds": ["%s"]
      }
    }`, issueID, userID)

	//	fmt.Println("🔧 Assigning user via GraphQL REST API...")

	// Run curl to perform assignment
	assignCmd := exec.Command("curl",
		"-s",
		"-X", "POST",
		"-H", "Authorization: bearer "+token,
		"-H", "Content-Type: application/json",
		"https://api.github.com/graphql",
		"-d", mutationPayload)

	_, err = assignCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run curl for assignment: %v", err)
	}

	//	fmt.Println("✅ Assignment response:")
	//	fmt.Println(string(assignOutput))

	//	fmt.Println("✅ User assigned successfully via REST GraphQL API.")
	return nil
}

func setIssueMilestone(token string, client *githubv4.Client, issueID githubv4.ID, milestoneTitle string) error {
	// token := os.Getenv("GITHUB_TOKEN")
	// if token == "" {
	// 	return fmt.Errorf("GITHUB_TOKEN is not set")
	// }

	owner := "fleetdm"
	repo := "fleet"

	// Step 1: Fetch milestones via REST API to get milestone number
	getMilestonesCmd := exec.Command("curl",
		"-s",
		"-H", "Authorization: Bearer "+token,
		"-H", "Accept: application/vnd.github+json",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/milestones?state=open", owner, repo),
	)

	milestonesJSON, err := getMilestonesCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to fetch milestones: %w", err)
	}

	var milestones []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	err = json.Unmarshal(milestonesJSON, &milestones)
	if err != nil {
		return fmt.Errorf("failed to parse milestones JSON: %w", err)
	}

	milestoneNumber := 0
	for _, m := range milestones {
		if m.Title == milestoneTitle {
			milestoneNumber = m.Number
			break
		}
	}
	if milestoneNumber == 0 {
		return fmt.Errorf("milestone %q not found", milestoneTitle)
	}

	// Step 2: Get issue number from issueID via GraphQL
	var query struct {
		Node struct {
			Issue struct {
				Number int
			} `graphql:"... on Issue"`
		} `graphql:"node(id: $id)"`
	}
	variables := map[string]interface{}{
		"id": issueID,
	}
	err = client.Query(context.Background(), &query, variables)
	if err != nil {
		return fmt.Errorf("failed to get issue number: %w", err)
	}
	issueNumber := query.Node.Issue.Number

	// Step 3: Set milestone using REST API PATCH
	patchData := fmt.Sprintf(`{"milestone": %d}`, milestoneNumber)
	setMilestoneCmd := exec.Command("curl",
		"-s",
		"-X", "PATCH",
		"-H", "Authorization: Bearer "+token,
		"-H", "Accept: application/vnd.github+json",
		"-H", "Content-Type: application/json",
		"-d", patchData,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, issueNumber),
	)

	patchOutput, err := setMilestoneCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set milestone: %w, output: %s", err, patchOutput)
	}

	//	fmt.Println("✅ Milestone set mutation completed.")

	// Step 4: Verify milestone via GraphQL
	var verifyQuery struct {
		Node struct {
			Issue struct {
				Milestone struct {
					Title string
				}
			} `graphql:"... on Issue"`
		} `graphql:"node(id: $id)"`
	}
	err = client.Query(context.Background(), &verifyQuery, variables)
	if err != nil {
		return fmt.Errorf("failed to verify milestone: %w", err)
	}

	if verifyQuery.Node.Issue.Milestone.Title == milestoneTitle {
		//		fmt.Printf("✅ Verified milestone: \"%s\"\n", milestoneTitle)
		return nil
	}

	return fmt.Errorf("❌ Milestone verification failed: expected %q but found %q",
		milestoneTitle, verifyQuery.Node.Issue.Milestone.Title)
}

func getEstimateFromProject67(token string, issueNumber int) (float64, error) {
	// token := os.Getenv("GITHUB_TOKEN")
	// if token == "" {
	// 	return 0, fmt.Errorf("GITHUB_TOKEN is not set")
	// }

	//	fmt.Println("🔍 Fetching Estimate from Project 67...")

	queryPayload := fmt.Sprintf(`{
		"query": "query($owner:String!, $repo:String!, $number:Int!) { repository(owner:$owner, name:$repo) { issue(number:$number) { projectItems(first:20) { nodes { project { number } fieldValues(first:20) { nodes { ... on ProjectV2ItemFieldNumberValue { number field { ... on ProjectV2FieldCommon { name } } } } } } } } } }",
		"variables": {
			"owner": "%s",
			"repo": "%s",
			"number": %d
		}
	}`, owner, repo, issueNumber)

	cmd := exec.Command("curl",
		"-s",
		"-X", "POST",
		"-H", "Authorization: bearer "+token,
		"-H", "Content-Type: application/json",
		"https://api.github.com/graphql",
		"-d", queryPayload)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run curl: %v", err)
	}

	//	fmt.Println("🔍 Raw JSON response:")
	//	fmt.Println(string(output))

	var resp struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							Project struct {
								Number int `json:"number"`
							} `json:"project"`
							FieldValues struct {
								Nodes []struct {
									Number float64 `json:"number"`
									Field  struct {
										Name string `json:"name"`
									} `json:"field"`
								} `json:"nodes"`
							} `json:"fieldValues"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &resp); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %v", err)
	}

	for _, item := range resp.Data.Repository.Issue.ProjectItems.Nodes {
		if item.Project.Number == 67 {
			for _, fv := range item.FieldValues.Nodes {
				if fv.Field.Name == "Estimate" {
					//					fmt.Printf("✅ Found Estimate: %.1f\n", fv.Number)
					return fv.Number, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("Estimate not found in Project 67")
}

// ORIGINAL NAME KEPT; now takes targetProjectNumber and uses it internally instead of hardcoded 71
func setEstimateInProject71(issueNumber int, targetProjectNumber int, estimate float64) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN is not set")
	}

	// 1. Fetch the target project item ID (was: Project 71 item ID)
	//	fmt.Println("🔍 Fetching Project item ID...")

	query := fmt.Sprintf(`{
      "query": "query($owner:String!,$repo:String!,$number:Int!) { repository(owner:$owner, name:$repo) { issue(number:$number) { projectItems(first:20) { nodes { id project { number id } } } } } }",
      "variables": {
        "owner": "fleetdm",
        "repo": "fleet",
        "number": %d
      }
    }`, issueNumber)

	cmd := exec.Command("curl",
		"-s",
		"-X", "POST",
		"-H", "Authorization: bearer "+token,
		"-H", "Content-Type: application/json",
		"https://api.github.com/graphql",
		"-d", query)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run curl to get Project item ID: %v", err)
	}

	type response struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							ID      string
							Project struct {
								Number int
								ID     string
							}
						}
					}
				}
			}
		}
	}

	var res response
	if err := json.Unmarshal(output, &res); err != nil {
		return fmt.Errorf("failed to parse JSON for Project item ID: %v", err)
	}

	var project71ItemID, project71ID string
	for _, node := range res.Data.Repository.Issue.ProjectItems.Nodes {
		if node.Project.Number == targetProjectNumber { // was: == 71
			project71ItemID = node.ID
			project71ID = node.Project.ID
			break
		}
	}
	if project71ItemID == "" || project71ID == "" {
		return fmt.Errorf("Project %d item or ID not found", targetProjectNumber)
	}
	//	fmt.Printf("✅ Project item ID: %s\n", project71ItemID)
	//	fmt.Printf("✅ Project ID: %s\n", project71ID)

	// 2. Fetch the Estimate field ID in the target project
	//	fmt.Println("🔍 Fetching Estimate field ID in target project...")
	fieldsQuery := fmt.Sprintf(`{
      "query": "query { node(id:\"%s\") { ... on ProjectV2 { fields(first:50) { nodes { ... on ProjectV2FieldCommon { id name dataType } } } } } }"
    }`, project71ID)

	fieldCmd := exec.Command("curl",
		"-s",
		"-X", "POST",
		"-H", "Authorization: bearer "+token,
		"-H", "Content-Type: application/json",
		"https://api.github.com/graphql",
		"-d", fieldsQuery)

	fieldOutput, err := fieldCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run curl to get fields: %v", err)
	}

	var fieldRes struct {
		Data struct {
			Node struct {
				Fields struct {
					Nodes []struct {
						ID       string
						Name     string
						DataType string
					}
				}
			}
		}
	}
	if err := json.Unmarshal(fieldOutput, &fieldRes); err != nil {
		return fmt.Errorf("failed to parse fields JSON: %v", err)
	}

	var estimateFieldID string
	for _, f := range fieldRes.Data.Node.Fields.Nodes {
		if f.Name == "Estimate" && f.DataType == "NUMBER" {
			estimateFieldID = f.ID
			break
		}
	}
	if estimateFieldID == "" {
		return fmt.Errorf("Estimate field not found in Project %d", targetProjectNumber)
	}
	//	fmt.Printf("✅ Estimate field ID: %s\n", estimateFieldID)

	// 3. Set the Estimate value
	//	fmt.Printf("🔧 Setting Estimate %.1f on Project target item...\n", estimate)

	mutation := fmt.Sprintf(`{
      "query": "mutation { updateProjectV2ItemFieldValue(input:{projectId:\"%s\", itemId:\"%s\", fieldId:\"%s\", value:{number:%f}}) { projectV2Item { id } } }"
    }`, project71ID, project71ItemID, estimateFieldID, estimate)

	setCmd := exec.Command("curl",
		"-s",
		"-X", "POST",
		"-H", "Authorization: bearer "+token,
		"-H", "Content-Type: application/json",
		"https://api.github.com/graphql",
		"-d", mutation)

	_, err = setCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to set estimate: %v", err)
	}

	//	fmt.Println("✅ Update response:")
	//	fmt.Println(string(setOutput))

	return nil
}

func removeIssueFromProject(client *githubv4.Client, owner, repo string, issueNumber, projectNumber int) error {
	ctx := context.Background()

	// Helper function to fetch Project Item IDs for the issue in the project
	fetchProjectItem := func() (githubv4.ID, githubv4.ID, bool, error) {
		var query struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							ID      githubv4.ID
							Project struct {
								Number int
								ID     githubv4.ID
							}
						}
					} `graphql:"projectItems(first: 20)"`
				} `graphql:"issue(number: $number)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}
		vars := map[string]interface{}{
			"owner":  githubv4.String(owner),
			"repo":   githubv4.String(repo),
			"number": githubv4.Int(issueNumber),
		}
		if err := client.Query(ctx, &query, vars); err != nil {
			return "", "", false, fmt.Errorf("query project items: %w", err)
		}
		for _, node := range query.Repository.Issue.ProjectItems.Nodes {
			if node.Project.Number == projectNumber {
				return node.ID, node.Project.ID, true, nil
			}
		}
		return "", "", false, nil
	}

	// Check if the item exists before deletion
	itemID, projectID, found, err := fetchProjectItem()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no project item found for issue #%d in project %d; nothing to remove", issueNumber, projectNumber)
	}

	log.Printf("Found project item ID %s in project ID %s for issue #%d; attempting removal...\n", itemID, projectID, issueNumber)

	// Perform deletion mutation
	var mutation struct {
		DeleteProjectV2Item struct {
			DeletedItemID githubv4.ID
		} `graphql:"deleteProjectV2Item(input: $input)"`
	}
	input := map[string]interface{}{
		"itemId":    itemID,
		"projectId": projectID,
	}
	if err := client.Mutate(ctx, &mutation, input, nil); err != nil {
		return fmt.Errorf("deleteProjectV2Item mutation failed: %w", err)
	}

	log.Printf("Mutation response: deleted item ID %s\n", mutation.DeleteProjectV2Item.DeletedItemID)

	// Verify deletion by querying again
	_, _, foundAfterDeletion, err := fetchProjectItem()
	if err != nil {
		return fmt.Errorf("verification query failed: %w", err)
	}
	if foundAfterDeletion {
		return fmt.Errorf("verification failed: project item still exists after deletion")
	}

	log.Println("Verification successful: issue removed from project")
	return nil
}

func removeIssueFromProjectWithCurl(ctx context.Context, issueNumber int, projectNumber int) error {
	// fmt.Printf("🔍 Looking up project item for issue #%d...\n", issueNumber)

	// First: query the project items to get itemId and projectId
	queryTemplate := `{
  "query": "query($owner: String!, $repo: String!, $number: Int!) { repository(owner: $owner, name: $repo) { issue(number: $number) { projectItems(first: 20) { nodes { id project { id number title } } } } } }",
  "variables": {
    "owner": "fleetdm",
    "repo": "fleet",
    "number": %d
  }
}`
	queryJSON := fmt.Sprintf(queryTemplate, issueNumber)

	cmdQuery := exec.CommandContext(ctx, "curl",
		"-s",
		"-H", "Authorization: bearer "+os.Getenv("GITHUB_TOKEN"),
		"-H", "Content-Type: application/json",
		"-X", "POST",
		"https://api.github.com/graphql",
		"-d", queryJSON,
	)
	queryOut, err := cmdQuery.Output()
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	var queryResp struct {
		Data struct {
			Repository struct {
				Issue struct {
					ProjectItems struct {
						Nodes []struct {
							ID      string `json:"id"`
							Project struct {
								ID     string `json:"id"`
								Number int    `json:"number"`
								Title  string `json:"title"`
							} `json:"project"`
						} `json:"nodes"`
					} `json:"projectItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(queryOut, &queryResp); err != nil {
		fmt.Println("Raw query response:", string(queryOut))
		return fmt.Errorf("decoding query response: %w", err)
	}

	var itemID, projectID string
	for _, node := range queryResp.Data.Repository.Issue.ProjectItems.Nodes {
		if node.Project.Number == projectNumber {
			itemID = node.ID
			projectID = node.Project.ID
			break
		}
	}
	if itemID == "" || projectID == "" {
		return fmt.Errorf("project #%d item not found on issue #%d", projectNumber, issueNumber)
	}

	// fmt.Printf("✅ Found item ID %s in project ID %s (%d)\n", itemID, projectID, projectNumber)

	// Second: perform the deletion
	mutationTemplate := `{
  "query": "mutation($itemId: ID!, $projectId: ID!) { deleteProjectV2Item(input: { itemId: $itemId, projectId: $projectId }) { deletedItemId } }",
  "variables": {
    "itemId": "%s",
    "projectId": "%s"
  }
}`
	mutationJSON := fmt.Sprintf(mutationTemplate, itemID, projectID)

	cmdDelete := exec.CommandContext(ctx, "curl",
		"-s",
		"-H", "Authorization: bearer "+os.Getenv("GITHUB_TOKEN"),
		"-H", "Content-Type: application/json",
		"-X", "POST",
		"https://api.github.com/graphql",
		"-d", mutationJSON,
	)
	deleteOut, err := cmdDelete.Output()
	if err != nil {
		return fmt.Errorf("delete mutation failed: %w", err)
	}

	var deleteResp struct {
		Data struct {
			DeleteProjectV2Item struct {
				DeletedItemID string `json:"deletedItemId"`
			} `json:"deleteProjectV2Item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(deleteOut, &deleteResp); err != nil {
		fmt.Println("Raw delete response:", string(deleteOut))
		return fmt.Errorf("decoding delete response: %w", err)
	}
	if deleteResp.Data.DeleteProjectV2Item.DeletedItemID == "" {
		fmt.Println("Raw delete response:", string(deleteOut))
		return fmt.Errorf("delete failed: no deletedItemId returned")
	}

	fmt.Printf("✅ Successfully removed item from project #%d.\n", projectNumber)
	return nil
}

func updateIssueLabels(token string, issueNumber int) error {
	// Step 1: Get current labels
	getCurrentLabelsCmd := exec.Command("curl",
		"-s",
		"-H", "Authorization: Bearer "+token,
		"-H", "Accept: application/vnd.github+json",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, issueNumber),
	)

	issueJSON, err := getCurrentLabelsCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to fetch issue labels: %w", err)
	}

	var issue struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	err = json.Unmarshal(issueJSON, &issue)
	if err != nil {
		return fmt.Errorf("failed to parse issue JSON: %w", err)
	}

	// Step 2: Create new labels list
	var newLabels []string
	hasRelease := false

	for _, label := range issue.Labels {
		if label.Name != ":product" {
			newLabels = append(newLabels, label.Name)
		}
		if label.Name == ":release" {
			hasRelease = true
		}
	}

	// Add ":release" if not already present
	if !hasRelease {
		newLabels = append(newLabels, ":release")
	}

	// Step 3: Update labels using REST API
	labelsJSON, err := json.Marshal(map[string][]string{"labels": newLabels})
	if err != nil {
		return fmt.Errorf("failed to marshal labels JSON: %w", err)
	}

	updateLabelsCmd := exec.Command("curl",
		"-s",
		"-X", "PUT",
		"-H", "Authorization: Bearer "+token,
		"-H", "Accept: application/vnd.github+json",
		"-H", "Content-Type: application/json",
		"-d", string(labelsJSON),
		fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/labels", owner, repo, issueNumber),
	)

	updateOutput, err := updateLabelsCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update labels: %w, output: %s", err, updateOutput)
	}

	return nil
}

func removeAllAssignees(token string, issueNumber int) error {
	// Step 1: Get current assignees
	getCurrentAssigneesCmd := exec.Command("curl",
		"-s",
		"-H", "Authorization: Bearer "+token,
		"-H", "Accept: application/vnd.github+json",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, issueNumber),
	)

	issueJSON, err := getCurrentAssigneesCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to fetch issue assignees: %w", err)
	}

	var issue struct {
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
	}
	err = json.Unmarshal(issueJSON, &issue)
	if err != nil {
		return fmt.Errorf("failed to parse issue JSON: %w", err)
	}

	// If no assignees, nothing to remove
	if len(issue.Assignees) == 0 {
		return nil
	}

	// Step 2: Create list of assignee logins to remove
	var assigneeLogins []string
	for _, assignee := range issue.Assignees {
		assigneeLogins = append(assigneeLogins, assignee.Login)
	}

	// Step 3: Remove all assignees using REST API
	assigneesJSON, err := json.Marshal(map[string][]string{"assignees": assigneeLogins})
	if err != nil {
		return fmt.Errorf("failed to marshal assignees JSON: %w", err)
	}

	removeAssigneesCmd := exec.Command("curl",
		"-s",
		"-X", "DELETE",
		"-H", "Authorization: Bearer "+token,
		"-H", "Accept: application/vnd.github+json",
		"-H", "Content-Type: application/json",
		"-d", string(assigneesJSON),
		fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/assignees", owner, repo, issueNumber),
	)

	removeOutput, err := removeAssigneesCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove assignees: %w, output: %s", err, removeOutput)
	}

	return nil
}
