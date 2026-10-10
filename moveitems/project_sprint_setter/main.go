package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type Iteration struct {
	ID        string
	Title     string
	StartDate string
	Duration  int
}

type SprintField struct {
	FieldID     githubv4.ID
	FieldName   string
	Iterations  []Iteration
	CurrentIter Iteration
}

func main() {
	var (
		org          = flag.String("org", "fleetdm", "GitHub org login")
		projectNum   = flag.Int("project", 71, "GitHub Project (v2) number")
		fieldNameArg = flag.String("field", "Sprint", "Iteration field name to use (case-insensitive match, exact match preferred)")
		statusField  = flag.String("status-field", "Status", "Status/column field name (single select) to use for excluding columns")
		excludeCols  = flag.String("exclude", "✅ Ready for release,Done", "Comma-separated exact Status values to exclude")
		limit        = flag.Int("limit", 0, "Optional max number of items to update (0 = no limit)")
		yes          = flag.Bool("yes", false, "Skip confirmation prompt (dangerous)")
		dryRun       = flag.Bool("dry-run", false, "Don't update anything; only print what would change")
	)
	flag.Parse()

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		log.Fatal("GITHUB_TOKEN is required")
	}

	ctx := context.Background()
	client := ghv4Client(token)

	projectID, projectTitle, err := fetchProjectIDAndTitle(ctx, client, *org, *projectNum)
	if err != nil {
		log.Fatalf("Failed to fetch project %d: %v", *projectNum, err)
	}

	sprintField, err := fetchIterationField(ctx, client, *org, *projectNum, *fieldNameArg)
	if err != nil {
		log.Fatalf("Failed to find iteration field: %v", err)
	}

	statusFieldID, statusFieldName, err := fetchSingleSelectField(ctx, client, *org, *projectNum, *statusField)
	if err != nil {
		log.Fatalf("Failed to find status field: %v", err)
	}

	excluded := parseExactSet(*excludeCols)

	if len(sprintField.Iterations) == 0 {
		log.Fatalf("Iteration field %q has no iterations configured", sprintField.FieldName)
	}

	cur, ok := pickCurrentIteration(time.Now(), sprintField.Iterations)
	if !ok {
		log.Fatalf("Couldn't determine current iteration for field %q (no iteration spans today)", sprintField.FieldName)
	}
	sprintField.CurrentIter = cur

	fmt.Printf("Project: %s (#%d)\n", projectTitle, *projectNum)
	fmt.Printf("Sprint field: %s (%s)\n", sprintField.FieldName, sprintField.FieldID)
	fmt.Printf("Status field: %s (%s)\n", statusFieldName, statusFieldID)
	if len(excluded) > 0 {
		fmt.Printf("Excluding columns (exact Status match): %s\n", strings.Join(sortedKeys(excluded), ", "))
	}
	fmt.Printf("Current iteration: %s (start %s, %d days)\n\n", cur.Title, cur.StartDate, cur.Duration)

	items, err := fetchItemsMissingIteration(ctx, client, projectID, sprintField.FieldID, statusFieldID, excluded)
	if err != nil {
		log.Fatalf("Failed to list items: %v", err)
	}

	if len(items) == 0 {
		fmt.Println("✅ No project items missing a sprint.")
		return
	}

	// Optional cap
	if *limit > 0 && len(items) > *limit {
		items = items[:*limit]
	}

	printItems(items, sprintField.FieldName)

	if *dryRun {
		fmt.Printf("\n(dry-run) Would set %s=%q for %d item(s).\n", sprintField.FieldName, cur.Title, len(items))
		return
	}

	if !*yes {
		ok, err := confirm(fmt.Sprintf("\nSet %s=%q for these %d item(s)? (y/N): ", sprintField.FieldName, cur.Title, len(items)))
		if err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}
		if !ok {
			fmt.Println("Aborted.")
			return
		}
	}

	updated := 0
	for _, it := range items {
		if err := setIteration(token, projectID, it.ItemID, sprintField.FieldID, cur.ID); err != nil {
			fmt.Printf("❌ %s: %v\n", it.URL, err)
			continue
		}
		updated++
		fmt.Printf("✅ Set sprint for %s\n", it.URL)
	}

	fmt.Printf("\nDone. Updated %d/%d items.\n", updated, len(items))
}

func ghv4Client(token string) *githubv4.Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	return githubv4.NewClient(httpClient)
}

func fetchProjectIDAndTitle(ctx context.Context, client *githubv4.Client, org string, number int) (githubv4.ID, string, error) {
	var q struct {
		Organization struct {
			ProjectV2 struct {
				ID    githubv4.ID
				Title string
			} `graphql:"projectV2(number:$number)"`
		} `graphql:"organization(login:$login)"`
	}
	vars := map[string]any{
		"login":  githubv4.String(org),
		"number": githubv4.Int(number),
	}
	if err := client.Query(ctx, &q, vars); err != nil {
		return "", "", err
	}
	if q.Organization.ProjectV2.ID == "" {
		return "", "", errors.New("project not found or no access")
	}
	return q.Organization.ProjectV2.ID, q.Organization.ProjectV2.Title, nil
}

func fetchIterationField(ctx context.Context, client *githubv4.Client, org string, projectNumber int, wantedName string) (*SprintField, error) {
	var q struct {
		Organization struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						Typename string        `graphql:"__typename"`
						Common struct {
							ID   githubv4.ID   `graphql:"id"`
							Name githubv4.String `graphql:"name"`
						} `graphql:"... on ProjectV2FieldCommon"`
						IterCfg struct {
							Configuration struct {
								Iterations []struct {
									ID        githubv4.String `graphql:"id"`
									Title     githubv4.String `graphql:"title"`
									StartDate githubv4.String `graphql:"startDate"`
									Duration  githubv4.Int    `graphql:"duration"`
								} `graphql:"iterations"`
							} `graphql:"configuration"`
						} `graphql:"... on ProjectV2IterationField"`
					}
				} `graphql:"fields(first:50)"`
			} `graphql:"projectV2(number:$number)"`
		} `graphql:"organization(login:$login)"`
	}
	vars := map[string]any{
		"login":  githubv4.String(org),
		"number": githubv4.Int(projectNumber),
	}
	if err := client.Query(ctx, &q, vars); err != nil {
		return nil, err
	}

	wantedLower := strings.ToLower(strings.TrimSpace(wantedName))

	type candidate struct {
		field SprintField
		score int
	}

	var cands []candidate

	for _, n := range q.Organization.ProjectV2.Fields.Nodes {
		if n.Typename != "ProjectV2IterationField" {
			continue
		}
		name := string(n.Common.Name)
		nameLower := strings.ToLower(name)

		score := 0
		switch {
		case nameLower == wantedLower:
			score = 100
		case strings.Contains(nameLower, wantedLower):
			score = 80
		case wantedLower == "sprint" && strings.Contains(nameLower, "sprint"):
			score = 70
		default:
			continue
		}

		sf := SprintField{
			FieldID:   n.Common.ID,
			FieldName: name,
		}
		for _, it := range n.IterCfg.Configuration.Iterations {
			sf.Iterations = append(sf.Iterations, Iteration{
				ID:        string(it.ID),
				Title:     string(it.Title),
				StartDate: string(it.StartDate),
				Duration:  int(it.Duration),
			})
		}
		cands = append(cands, candidate{field: sf, score: score})
	}

	if len(cands) == 0 {
		return nil, fmt.Errorf("no iteration field matched %q (tried exact/contains, case-insensitive)", wantedName)
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	best := cands[0].field
	return &best, nil
}

func fetchSingleSelectField(ctx context.Context, client *githubv4.Client, org string, projectNumber int, wantedName string) (githubv4.ID, string, error) {
	var q struct {
		Organization struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						Typename string `graphql:"__typename"`
						Common   struct {
							ID   githubv4.ID     `graphql:"id"`
							Name githubv4.String `graphql:"name"`
						} `graphql:"... on ProjectV2FieldCommon"`
					} `graphql:"nodes"`
				} `graphql:"fields(first:50)"`
			} `graphql:"projectV2(number:$number)"`
		} `graphql:"organization(login:$login)"`
	}
	vars := map[string]any{
		"login":  githubv4.String(org),
		"number": githubv4.Int(projectNumber),
	}
	if err := client.Query(ctx, &q, vars); err != nil {
		return "", "", err
	}

	wantedLower := strings.ToLower(strings.TrimSpace(wantedName))
	bestScore := -1
	var bestID githubv4.ID
	var bestName string
	for _, n := range q.Organization.ProjectV2.Fields.Nodes {
		if n.Typename != "ProjectV2SingleSelectField" {
			continue
		}
		name := string(n.Common.Name)
		nameLower := strings.ToLower(name)
		score := -1
		switch {
		case nameLower == wantedLower:
			score = 100
		case strings.Contains(nameLower, wantedLower) && wantedLower != "":
			score = 80
		default:
			continue
		}
		if score > bestScore {
			bestScore = score
			bestID = n.Common.ID
			bestName = name
		}
	}
	if bestScore < 0 {
		return "", "", fmt.Errorf("no single select field matched %q", wantedName)
	}
	return bestID, bestName, nil
}

type ProjectItem struct {
	ItemID githubv4.ID
	Number int
	Title  string
	URL    string
	Type   string // Issue/PR/Draft
}

func fetchItemsMissingIteration(ctx context.Context, client *githubv4.Client, projectID githubv4.ID, sprintFieldID githubv4.ID, statusFieldID githubv4.ID, excludedStatus map[string]struct{}) ([]ProjectItem, error) {
	var all []ProjectItem
	var after *githubv4.String

	for {
		var q struct {
			Node struct {
				ProjectV2 struct {
					Items struct {
						PageInfo struct {
							HasNextPage githubv4.Boolean
							EndCursor   githubv4.String
						}
						Nodes []struct {
							ID githubv4.ID

							// content (issue/pr/draft)
							Content struct {
								Typename string `graphql:"__typename"`
								Issue    struct {
									Number githubv4.Int
									Title  githubv4.String
									URL    githubv4.URI
								} `graphql:"... on Issue"`
								PR struct {
									Number githubv4.Int
									Title  githubv4.String
									URL    githubv4.URI
								} `graphql:"... on PullRequest"`
								Draft struct {
									Title githubv4.String
								} `graphql:"... on DraftIssue"`
							} `graphql:"content"`

							FieldValues struct {
								Nodes []struct {
									// NOTE: fieldValues.nodes is a UNION (ProjectV2ItemFieldValue),
									// so *all* selections (except __typename) must live inside inline fragments.
									Typename string `graphql:"__typename"`

									Iter struct {
										Field struct {
											Common struct {
												ID   githubv4.ID     `graphql:"id"`
												Name githubv4.String `graphql:"name"`
											} `graphql:"... on ProjectV2FieldCommon"`
										} `graphql:"field"`
										IterationID githubv4.String `graphql:"iterationId"`
										Title       githubv4.String `graphql:"title"`
									} `graphql:"... on ProjectV2ItemFieldIterationValue"`

									Single struct {
										Field struct {
											Common struct {
												ID   githubv4.ID     `graphql:"id"`
												Name githubv4.String `graphql:"name"`
											} `graphql:"... on ProjectV2FieldCommon"`
										} `graphql:"field"`
										Name githubv4.String `graphql:"name"`
									} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
								}
							} `graphql:"fieldValues(first:20)"`
						}
					} `graphql:"items(first:50, after:$after)"`
				} `graphql:"... on ProjectV2"`
			} `graphql:"node(id:$id)"`
		}

		vars := map[string]any{"id": projectID, "after": after}
		if err := client.Query(ctx, &q, vars); err != nil {
			return nil, err
		}

		for _, n := range q.Node.ProjectV2.Items.Nodes {
			var (
				hasSprint   bool
				statusValue string
			)
			for _, fv := range n.FieldValues.Nodes {
				// Sprint (iteration)
				if fv.Typename == "ProjectV2ItemFieldIterationValue" && fv.Iter.Field.Common.ID == sprintFieldID {
					if string(fv.Iter.IterationID) != "" {
						hasSprint = true
					}
				}
				// Status (single select)
				if fv.Typename == "ProjectV2ItemFieldSingleSelectValue" && fv.Single.Field.Common.ID == statusFieldID {
					statusValue = string(fv.Single.Name)
				}
			}
			if hasSprint {
				continue
			}
			if statusValue != "" {
				if _, ok := excludedStatus[statusValue]; ok {
					continue
				}
			}

			item := ProjectItem{ItemID: n.ID}

			switch n.Content.Typename {
			case "Issue":
				item.Type = "Issue"
				item.Number = int(n.Content.Issue.Number)
				item.Title = string(n.Content.Issue.Title)
				item.URL = n.Content.Issue.URL.String()
			case "PullRequest":
				item.Type = "PR"
				item.Number = int(n.Content.PR.Number)
				item.Title = string(n.Content.PR.Title)
				item.URL = n.Content.PR.URL.String()
			case "DraftIssue":
				item.Type = "Draft"
				item.Number = 0
				item.Title = string(n.Content.Draft.Title)
				// Draft issues don't have a URL
				item.URL = "(draft)"
			default:
				item.Type = n.Content.Typename
				item.Title = "(unknown content type)"
				item.URL = ""
			}

			all = append(all, item)
		}

		if !bool(q.Node.ProjectV2.Items.PageInfo.HasNextPage) {
			break
		}
		cur := q.Node.ProjectV2.Items.PageInfo.EndCursor
		after = &cur
	}

	// Stable ordering for printing
	sort.Slice(all, func(i, j int) bool {
		if all[i].Type != all[j].Type {
			return all[i].Type < all[j].Type
		}
		if all[i].Number != all[j].Number {
			return all[i].Number < all[j].Number
		}
		return all[i].Title < all[j].Title
	})

	return all, nil
}

func pickCurrentIteration(now time.Time, iters []Iteration) (Iteration, bool) {
	type span struct {
		it    Iteration
		start time.Time
		end   time.Time
	}
	var spans []span
	for _, it := range iters {
		if it.StartDate == "" || it.Duration <= 0 {
			continue
		}
		start, err := time.Parse("2006-01-02", it.StartDate)
		if err != nil {
			continue
		}
		end := start.AddDate(0, 0, it.Duration)
		spans = append(spans, span{it: it, start: start, end: end})
	}
	if len(spans) == 0 {
		return Iteration{}, false
	}

	// Prefer iteration that spans today.
	for _, s := range spans {
		if !now.Before(s.start) && now.Before(s.end) {
			return s.it, true
		}
	}

	// Otherwise, pick the latest iteration that started in the past.
	sort.Slice(spans, func(i, j int) bool { return spans[i].start.After(spans[j].start) })
	for _, s := range spans {
		if !now.Before(s.start) {
			return s.it, true
		}
	}

	// Otherwise, pick the earliest (all in the future).
	sort.Slice(spans, func(i, j int) bool { return spans[i].start.Before(spans[j].start) })
	return spans[0].it, true
}

func printItems(items []ProjectItem, sprintFieldName string) {
	fmt.Printf("Found %d item(s) missing %s:\n\n", len(items), sprintFieldName)
	for i, it := range items {
		switch it.Type {
		case "Issue":
			fmt.Printf("%3d) %-5s #%d  %s\n     %s\n", i+1, it.Type, it.Number, it.Title, it.URL)
		case "PR":
			fmt.Printf("%3d) %-5s #%d  %s\n     %s\n", i+1, it.Type, it.Number, it.Title, it.URL)
		default:
			fmt.Printf("%3d) %-5s %s\n", i+1, it.Type, it.Title)
		}
	}
}

func confirm(prompt string) (bool, error) {
	fmt.Print(prompt)
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func setIteration(token string, projectID githubv4.ID, itemID githubv4.ID, fieldID githubv4.ID, iterationID string) error {
	// We use raw HTTP here (same spirit as your existing tools) to avoid any schema mismatches
	// in githubv4 structs for UpdateProjectV2ItemFieldValueInput across versions.
	q := fmt.Sprintf(`mutation {
  updateProjectV2ItemFieldValue(input:{
    projectId:"%s",
    itemId:"%s",
    fieldId:"%s",
    value:{iterationId:"%s"}
  }) { projectV2Item { id } }
}`, projectID, itemID, fieldID, iterationID)

	payload := map[string]any{"query": q}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		Data   any `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if len(out.Errors) > 0 {
		var msgs []string
		for _, e := range out.Errors {
			msgs = append(msgs, e.Message)
		}
		return errors.New(strings.Join(msgs, "; "))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub GraphQL HTTP %s", resp.Status)
	}
	return nil
}

func parseExactSet(csv string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(csv, ",") {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
