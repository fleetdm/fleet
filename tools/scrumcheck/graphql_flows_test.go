package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newGraphQLStubClient(t *testing.T) *githubv4.Client {
	t.Helper()
	hc := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			var req struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}
			_ = json.Unmarshal(raw, &req)

			resp := map[string]any{"data": map[string]any{}}
			switch {
			case strings.Contains(req.Query, "projectV2(number: $num)"):
				num := int(req.Variables["num"].(float64))
				resp["data"] = map[string]any{
					"organization": map[string]any{
						"projectV2": map[string]any{
							"id": fmt.Sprintf("P%d", num),
						},
					},
				}
			case strings.Contains(req.Query, "node(id: $id)"):
				id := req.Variables["id"].(string)
				resp["data"] = map[string]any{
					"node": map[string]any{
						"items": map[string]any{
							"nodes": graphNodesForProjectID(id),
						},
					},
				}
			default:
				resp["data"] = map[string]any{}
			}

			b, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(b)),
			}, nil
		}),
	}
	return githubv4.NewClient(hc)
}

func graphNodesForProjectID(id string) []map[string]any {
	switch id {
	case "P67":
		return []map[string]any{
			issueNode(7001, "Drafting issue", "https://github.com/fleetdm/fleet/issues/7001", "Ready to estimate", "- [ ] check one", []string{"bug"}, time.Now().UTC(), nil),
		}
	case "P71":
		return []map[string]any{
			issueNode(7101, "Awaiting stale", "https://github.com/fleetdm/fleet/issues/7101", "✔️Awaiting QA", "- [ ] "+checkText, []string{":product"}, time.Now().UTC().Add(-72*time.Hour), nil),
			issueNode(7102, "Unreleased unassigned", "https://github.com/fleetdm/fleet/issues/7102", "🦃 In review", "bug body", []string{"g-orchestration", "~unreleased bug", ":release"}, time.Now().UTC(), nil),
		}
	case "P97":
		return []map[string]any{
			issueNode(9701, "Done item", "https://github.com/fleetdm/fleet/issues/9701", "Done", "- [ ] x", []string{"x"}, time.Now().UTC(), nil),
		}
	default:
		return nil
	}
}

func issueNode(number int, title, u, status, body string, labels []string, updated time.Time, assignees []string) map[string]any {
	labelNodes := make([]map[string]any, 0, len(labels))
	for _, l := range labels {
		labelNodes = append(labelNodes, map[string]any{"name": l})
	}
	assigneeNodes := make([]map[string]any, 0, len(assignees))
	for _, a := range assignees {
		assigneeNodes = append(assigneeNodes, map[string]any{"login": a})
	}
	return map[string]any{
		"id":        fmt.Sprintf("ITEM_%d", number),
		"updatedAt": updated.Format(time.RFC3339),
		"content": map[string]any{
			"number":    number,
			"title":     title,
			"body":      body,
			"url":       u,
			"milestone": map[string]any{"title": ""},
			"labels":    map[string]any{"nodes": labelNodes},
			"assignees": map[string]any{"nodes": assigneeNodes},
		},
		"fieldValues": map[string]any{
			"nodes": []map[string]any{
				{
					"name": status,
					"field": map[string]any{
						"id":   "FIELD_STATUS",
						"name": "Status",
					},
				},
			},
		},
	}
}

func TestGraphQLFlowHelpersAndChecks(t *testing.T) {
	t.Parallel()

	client := newGraphQLStubClient(t)
	ctx := context.Background()

	pid := fetchProjectID(ctx, client, "fleetdm", 71)
	if fmt.Sprintf("%v", pid) != "P71" {
		t.Fatalf("project id=%v want P71", pid)
	}

	items := fetchItems(ctx, client, githubv4.ID("P71"), 10)
	if len(items) != 2 {
		t.Fatalf("items len=%d want=2", len(items))
	}

	awaiting, stale := runAwaitingQACheck(ctx, client, "fleetdm", 20, []int{71, 97}, 24*time.Hour, nil)
	if len(awaiting[71]) != 1 || len(stale[71]) != 1 {
		t.Fatalf("unexpected awaiting/stale: awaiting=%d stale=%d", len(awaiting[71]), len(stale[71]))
	}
	if len(awaiting[97]) != 0 {
		t.Fatalf("expected done item ignored for awaiting violation")
	}

	drafting := runDraftingCheck(ctx, client, "fleetdm", 20, nil)
	if len(drafting) != 1 {
		t.Fatalf("drafting len=%d want=1", len(drafting))
	}

	release := runReleaseLabelChecks(ctx, client, "fleetdm", []int{67, 71, 97}, 20, nil)
	if len(release) != 1 || release[0].ProjectNum != 71 {
		t.Fatalf("unexpected release results: %#v", release)
	}

	origSearch := searchUnreleasedIssuesByGroup
	searchUnreleasedIssuesByGroup = func(ctx context.Context, token, org, groupLabel string) []struct {
		Number        int    `json:"number"`
		Title         string `json:"title"`
		HTMLURL       string `json:"html_url"`
		State         string `json:"state"`
		RepositoryURL string `json:"repository_url"`
		Body          string `json:"body"`
		Assignees     []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	} {
		if groupLabel != "g-orchestration" {
			return nil
		}
		return []struct {
			Number        int    `json:"number"`
			Title         string `json:"title"`
			HTMLURL       string `json:"html_url"`
			State         string `json:"state"`
			RepositoryURL string `json:"repository_url"`
			Body          string `json:"body"`
			Assignees     []struct {
				Login string `json:"login"`
			} `json:"assignees"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		}{
			{
				Number:        7102,
				Title:         "Unreleased unassigned",
				HTMLURL:       "https://github.com/fleetdm/fleet/issues/7102",
				State:         "open",
				RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
				Assignees:     nil,
				Labels: []struct {
					Name string `json:"name"`
				}{
					{Name: "g-orchestration"},
					{Name: "~unreleased bug"},
				},
			},
		}
	}
	defer func() { searchUnreleasedIssuesByGroup = origSearch }()

	unassignedUnreleased := runUnassignedUnreleasedBugChecks(
		ctx,
		client,
		"fleetdm",
		[]int{71, 97},
		20,
		"",
		compileLabelFilter([]string{"g-orchestration"}),
		orderedGroupLabels([]string{"g-orchestration"}),
	)
	if len(unassignedUnreleased) != 1 {
		t.Fatalf("unexpected unassigned unreleased results: %#v", unassignedUnreleased)
	}
}
