package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type genericQueryDefinition struct {
	Title string
	Query string
}

// genericQueryDefinitions is the single source of truth for custom query checks.
//
// Template placeholders:
//   - <<group>>: expanded once per normalized -l label (as a GitHub label term, e.g. "#g-security-compliance")
//   - <<project>>: expanded once per selected project number from -p
//
// If a query contains a placeholder but the corresponding runtime values are missing,
// that query is skipped entirely.
var genericQueryDefinitions = []genericQueryDefinition{
	{
		Title: "No on any project",
		Query: `is:issue is:open label:bug label:<<group>> -project:fleetdm/97 -project:fleetdm/71 -project:fleetdm/67 -project:fleetdm/79 -label::product -label:~engineering-initiated -label::reproduce`,
	},
}

// runGenericQueryChecks executes each configured generic query template,
// expands placeholders, runs the resulting searches, and returns results in
// config order for deterministic UI rendering.
func runGenericQueryChecks(
	ctx context.Context,
	token string,
	projectNums []int,
	groupLabels []string,
) []GenericQueryResult {
	// Preserve declaration order so UI review order matches this config order.
	results := make([]GenericQueryResult, 0)
	for _, def := range genericQueryDefinitions {
		queries := expandGenericQueryTemplate(def.Query, projectNums, groupLabels)
		for _, expanded := range queries {
			items := fetchGenericQueryIssues(ctx, token, expanded)
			results = append(results, GenericQueryResult{
				Title: def.Title,
				Query: expanded,
				Items: items,
			})
		}
	}
	return results
}

// expandGenericQueryTemplate turns one template into concrete query strings by
// expanding <<group>> and <<project>> placeholders from runtime inputs.
// If a required placeholder has no input values, the query is skipped.
func expandGenericQueryTemplate(template string, projectNums []int, groupLabels []string) []string {
	template = strings.TrimSpace(template)
	if template == "" {
		return nil
	}
	queries := []string{template}

	if strings.Contains(template, "<<group>>") {
		if len(groupLabels) == 0 {
			return nil
		}
		next := make([]string, 0, len(queries)*len(groupLabels))
		for _, q := range queries {
			for _, group := range groupLabels {
				next = append(next, strings.ReplaceAll(q, "<<group>>", formatGroupLabelForQuery(group)))
			}
		}
		queries = next
	}

	if strings.Contains(template, "<<project>>") {
		if len(projectNums) == 0 {
			return nil
		}
		next := make([]string, 0, len(queries)*len(projectNums))
		for _, q := range queries {
			for _, project := range projectNums {
				next = append(next, strings.ReplaceAll(q, "<<project>>", strconv.Itoa(project)))
			}
		}
		queries = next
	}
	return queries
}

// formatGroupLabelForQuery normalizes a group label and emits a quoted token
// suitable for GitHub label search terms (for example "#g-orchestration").
func formatGroupLabelForQuery(group string) string {
	norm := normalizeLabelName(group)
	if norm == "" {
		return group
	}
	return fmt.Sprintf(`"%s"`, "#"+norm)
}

// fetchGenericQueryIssues runs paged GitHub issue search for one concrete query
// and maps/deduplicates results into GenericQueryIssue records.
func fetchGenericQueryIssues(ctx context.Context, token, query string) []GenericQueryIssue {
	seen := make(map[string]struct{})
	out := make([]GenericQueryIssue, 0, 32)
	for page := 1; page <= 10; page++ {
		endpoint := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100&page=%d", urlQueryEscape(query), page)
		body, ok := executeIssueSearchRequest(ctx, endpoint, token)
		if !ok {
			break
		}
		if len(body.Items) == 0 {
			break
		}
		for _, it := range body.Items {
			key := fmt.Sprintf("%s#%d", it.RepositoryURL, it.Number)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			owner, repo := parseRepoFromRepositoryAPIURL(it.RepositoryURL)
			if owner == "" || repo == "" {
				continue
			}
			labels := make([]string, 0, len(it.Labels))
			for _, label := range it.Labels {
				name := strings.TrimSpace(label.Name)
				if name == "" {
					continue
				}
				labels = append(labels, name)
			}
			assignees := make([]string, 0, len(it.Assignees))
			for _, assignee := range it.Assignees {
				login := strings.TrimSpace(assignee.Login)
				if login == "" {
					continue
				}
				assignees = append(assignees, login)
			}
			status := strings.Title(strings.ToLower(strings.TrimSpace(it.State)))
			out = append(out, GenericQueryIssue{
				Number:           it.Number,
				Title:            it.Title,
				URL:              it.HTMLURL,
				RepoOwner:        owner,
				RepoName:         repo,
				Status:           status,
				CurrentLabels:    labels,
				CurrentAssignees: assignees,
			})
		}
		if len(body.Items) < 100 {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RepoOwner != out[j].RepoOwner {
			return out[i].RepoOwner < out[j].RepoOwner
		}
		if out[i].RepoName != out[j].RepoName {
			return out[i].RepoName < out[j].RepoName
		}
		return out[i].Number < out[j].Number
	})
	return out
}

// countGenericQueryIssues returns the total issue count across all expanded
// generic queries for progress output and summary pills.
func countGenericQueryIssues(results []GenericQueryResult) int {
	total := 0
	for _, result := range results {
		total += len(result.Items)
	}
	return total
}
