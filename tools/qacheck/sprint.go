package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type projectIteration struct {
	ID        string
	Title     string
	StartDate string
	Duration  int
}

type sprintProjectConfig struct {
	ProjectNum      int
	ProjectID       githubv4.ID
	SprintFieldID   githubv4.ID
	SprintFieldName string
	CurrentIterID   string
	CurrentIterName string
	StatusFieldID   githubv4.ID
}

type MissingSprintViolation struct {
	ProjectNum      int
	ProjectID       githubv4.ID
	ItemID          githubv4.ID
	Item            Item
	Status          string
	CurrentSprintID string
	CurrentSprint   string
	SprintFieldID   githubv4.ID
	SprintFieldName string
}

func runMissingSprintChecks(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNums []int,
	limit int,
) []MissingSprintViolation {
	out := make([]MissingSprintViolation, 0)
	now := time.Now().UTC()

	for _, projectNum := range projectNums {
		cfg, ok := fetchSprintProjectConfig(ctx, client, org, projectNum, now)
		if !ok {
			continue
		}
		items := fetchItems(ctx, client, cfg.ProjectID, limit)
		for _, it := range items {
			if getNumber(it) == 0 {
				continue
			}
			if inDoneColumn(it) {
				continue
			}
			hasSprint := false
			status := ""
			for _, fv := range it.FieldValues.Nodes {
				if fv.IterationValue.Field.Common.ID == cfg.SprintFieldID && strings.TrimSpace(string(fv.IterationValue.IterationID)) != "" {
					hasSprint = true
				}
				if fv.SingleSelectValue.Field.Common.ID == cfg.StatusFieldID {
					status = strings.TrimSpace(string(fv.SingleSelectValue.Name))
				}
			}
			if hasSprint {
				continue
			}

			out = append(out, MissingSprintViolation{
				ProjectNum:      projectNum,
				ProjectID:       cfg.ProjectID,
				ItemID:          it.ID,
				Item:            it,
				Status:          status,
				CurrentSprintID: cfg.CurrentIterID,
				CurrentSprint:   cfg.CurrentIterName,
				SprintFieldID:   cfg.SprintFieldID,
				SprintFieldName: cfg.SprintFieldName,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ProjectNum != out[j].ProjectNum {
			return out[i].ProjectNum < out[j].ProjectNum
		}
		return getNumber(out[i].Item) < getNumber(out[j].Item)
	})
	return out
}

func fetchSprintProjectConfig(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	projectNum int,
	now time.Time,
) (sprintProjectConfig, bool) {
	var q struct {
		Organization struct {
			ProjectV2 struct {
				ID     githubv4.ID
				Fields struct {
					Nodes []struct {
						Typename string `graphql:"__typename"`
						Common   struct {
							ID   githubv4.ID
							Name githubv4.String
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
		"number": githubv4.Int(projectNum),
	}
	if err := client.Query(ctx, &q, vars); err != nil {
		return sprintProjectConfig{}, false
	}
	if q.Organization.ProjectV2.ID == nil {
		return sprintProjectConfig{}, false
	}

	var statusFieldID githubv4.ID
	var sprintFieldID githubv4.ID
	sprintFieldName := ""
	iterations := make([]projectIteration, 0)
	for _, n := range q.Organization.ProjectV2.Fields.Nodes {
		name := strings.TrimSpace(strings.ToLower(string(n.Common.Name)))
		switch n.Typename {
		case "ProjectV2SingleSelectField":
			if name == "status" {
				statusFieldID = n.Common.ID
			}
		case "ProjectV2IterationField":
			if name == "sprint" && sprintFieldID == nil {
				sprintFieldID = n.Common.ID
				sprintFieldName = string(n.Common.Name)
				for _, it := range n.IterCfg.Configuration.Iterations {
					iterations = append(iterations, projectIteration{
						ID:        string(it.ID),
						Title:     string(it.Title),
						StartDate: string(it.StartDate),
						Duration:  int(it.Duration),
					})
				}
			}
		}
	}
	if sprintFieldID == nil || statusFieldID == nil {
		return sprintProjectConfig{}, false
	}
	current, ok := pickCurrentIteration(now, iterations)
	if !ok {
		return sprintProjectConfig{}, false
	}

	return sprintProjectConfig{
		ProjectNum:      projectNum,
		ProjectID:       q.Organization.ProjectV2.ID,
		SprintFieldID:   sprintFieldID,
		SprintFieldName: sprintFieldName,
		CurrentIterID:   current.ID,
		CurrentIterName: current.Title,
		StatusFieldID:   statusFieldID,
	}, true
}

func pickCurrentIteration(now time.Time, iters []projectIteration) (projectIteration, bool) {
	type span struct {
		it    projectIteration
		start time.Time
		end   time.Time
	}
	spans := make([]span, 0, len(iters))
	for _, it := range iters {
		if it.StartDate == "" || it.Duration <= 0 {
			continue
		}
		start, err := time.Parse("2006-01-02", it.StartDate)
		if err != nil {
			continue
		}
		spans = append(spans, span{it: it, start: start, end: start.AddDate(0, 0, it.Duration)})
	}
	if len(spans) == 0 {
		return projectIteration{}, false
	}

	for _, s := range spans {
		if !now.Before(s.start) && now.Before(s.end) {
			return s.it, true
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start.After(spans[j].start) })
	for _, s := range spans {
		if !now.Before(s.start) {
			return s.it, true
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start.Before(spans[j].start) })
	return spans[0].it, true
}

func sprintColumnGroup(status string) string {
	n := normalizeStatusName(status)
	switch {
	case strings.Contains(n, "ready for release"):
		return "ready_for_release"
	case strings.Contains(n, "awaiting qa"):
		return "awaiting_qa"
	case strings.Contains(n, "in review"),
		strings.Contains(n, "review"):
		return "in_review"
	case strings.Contains(n, "in progress"):
		return "in_progress"
	case strings.Contains(n, "waiting"):
		return "waiting"
	case n == "ready",
		strings.Contains(n, "ready to estimate"),
		strings.Contains(n, "ready for estimate"):
		return "ready"
	default:
		return "other"
	}
}

func sprintColumnOrder() []string {
	return []string{
		"ready",
		"waiting",
		"in_progress",
		"in_review",
		"awaiting_qa",
		"ready_for_release",
		"other",
	}
}

func sprintColumnLabel(group string) string {
	switch group {
	case "ready":
		return "Ready"
	case "waiting":
		return "Waiting"
	case "in_progress":
		return "In progress"
	case "in_review":
		return "In review"
	case "awaiting_qa":
		return "Awaiting QA"
	case "ready_for_release":
		return "Ready for release"
	case "other":
		return "Other"
	default:
		return "All"
	}
}

func setCurrentSprintForItem(
	token string,
	projectID githubv4.ID,
	itemID githubv4.ID,
	fieldID githubv4.ID,
	iterationID string,
) error {
	query := fmt.Sprintf(`mutation {
  updateProjectV2ItemFieldValue(input:{
    projectId:"%s",
    itemId:"%s",
    fieldId:"%s",
    value:{iterationId:"%s"}
  }) { projectV2Item { id } }
}`, fmt.Sprintf("%v", projectID), fmt.Sprintf("%v", itemID), fmt.Sprintf("%v", fieldID), iterationID)
	return githubGraphQLMutation(token, query)
}

func githubGraphQLMutation(token string, query string) error {
	payload := map[string]any{"query": query}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(body))
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
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if len(out.Errors) > 0 {
		msgs := make([]string, 0, len(out.Errors))
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
