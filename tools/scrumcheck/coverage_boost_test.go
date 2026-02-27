package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

func withMockDefaultTransport(t *testing.T, rt roundTripFunc) {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() { http.DefaultTransport = old })
}

func jsonResponse(t *testing.T, status int, v any) *http.Response {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

func TestStringListFlagAndMainHelpers(t *testing.T) {
	var labels stringListFlag
	if err := labels.Set(" #g-orchestration, ,#g-security-compliance "); err != nil {
		t.Fatalf("set labels: %v", err)
	}
	if got, want := labels.String(), "#g-orchestration,#g-security-compliance"; got != want {
		t.Fatalf("labels.String()=%q want %q", got, want)
	}

	missing, assigned := splitAssigneeCounts([]MissingAssigneeIssue{
		{AssignedToMe: false},
		{AssignedToMe: true},
		{AssignedToMe: false},
	})
	if missing != 2 || assigned != 1 {
		t.Fatalf("splitAssigneeCounts=(%d,%d) want (2,1)", missing, assigned)
	}

	out := captureStdout(t, func() {
		printDraftingSummary(map[string][]DraftingCheckViolation{
			"ready to estimate": {
				{
					Item:      testIssueWithStatus(101, "Draft item", "https://github.com/fleetdm/fleet/issues/101", "Ready to estimate"),
					Unchecked: []string{"one"},
					Status:    "Ready to estimate",
				},
			},
		}, 1)
	})
	if !strings.Contains(out, "Drafting checklist audit") {
		t.Fatalf("expected drafting summary output, got: %s", out)
	}
}

func TestGenericQueryExpansionAndExecution(t *testing.T) {
	origDefs := genericQueryDefinitions
	genericQueryDefinitions = []genericQueryDefinition{
		{
			Title: "X",
			Query: `is:issue label:<<group>> project:fleetdm/<<project>>`,
		},
	}
	t.Cleanup(func() { genericQueryDefinitions = origDefs })

	seenQueries := make([]string, 0, 2)
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" || r.URL.Path != "/search/issues" {
			return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
		}
		q := r.URL.Query().Get("q")
		decoded, _ := url.QueryUnescape(q)
		seenQueries = append(seenQueries, decoded)
		page := r.URL.Query().Get("page")
		if page == "2" {
			return jsonResponse(t, http.StatusOK, searchIssueResponse{Items: []searchIssueItem{}}), nil
		}
		return jsonResponse(t, http.StatusOK, searchIssueResponse{
			Items: []searchIssueItem{
				{
					Number:        4444,
					Title:         "Generic hit",
					HTMLURL:       "https://github.com/fleetdm/fleet/issues/4444",
					State:         "open",
					RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
					Assignees:     []struct{ Login string `json:"login"` }{{Login: "alice"}},
					Labels:        []struct{ Name string `json:"name"` }{{Name: "bug"}},
				},
			},
		}), nil
	}))

	got := runGenericQueryChecks(context.Background(), "tok", []int{97}, []string{"#g-orchestration"})
	if len(got) != 1 {
		t.Fatalf("results len=%d want 1", len(got))
	}
	if len(got[0].Items) != 1 {
		t.Fatalf("items len=%d want 1", len(got[0].Items))
	}
	if countGenericQueryIssues(got) != 1 {
		t.Fatalf("countGenericQueryIssues=%d want 1", countGenericQueryIssues(got))
	}
	if len(seenQueries) == 0 || !strings.Contains(seenQueries[0], `label:"#g-orchestration"`) || !strings.Contains(seenQueries[0], "project:fleetdm/97") {
		t.Fatalf("unexpected expanded query: %#v", seenQueries)
	}

	if out := expandGenericQueryTemplate("label:<<group>>", nil, nil); out != nil {
		t.Fatalf("expected nil when group placeholder has no values")
	}
	if out := expandGenericQueryTemplate("project:<<project>>", nil, []string{"x"}); out != nil {
		t.Fatalf("expected nil when project placeholder has no values")
	}
}

func TestExecuteIssueSearchRequestFallbackAndHelpers(t *testing.T) {
	calls := 0
	authHeaders := []string{}
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		if calls == 1 {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`forbidden`)),
			}, nil
		}
		return jsonResponse(t, http.StatusOK, searchIssueResponse{
			Items: []searchIssueItem{
				{
					Number:        1,
					Title:         "ok",
					HTMLURL:       "https://github.com/fleetdm/fleet/issues/1",
					State:         "open",
					RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
					Labels:        []struct{ Name string `json:"name"` }{{Name: "BUG"}},
				},
			},
		}), nil
	}))

	decoded, ok := executeIssueSearchRequest(context.Background(), "https://api.github.com/search/issues?q=x", "tok")
	if !ok || len(decoded.Items) != 1 {
		t.Fatalf("fallback search failed: ok=%v items=%d", ok, len(decoded.Items))
	}
	if calls != 2 {
		t.Fatalf("expected two calls (token then anonymous), got %d", calls)
	}
	if authHeaders[0] == "" || authHeaders[1] != "" {
		t.Fatalf("unexpected auth header sequence: %#v", authHeaders)
	}

	if owner, repo := parseRepoFromRepositoryAPIURL("https://api.github.com/repos/fleetdm/fleet"); owner != "fleetdm" || repo != "fleet" {
		t.Fatalf("parseRepoFromRepositoryAPIURL=(%q,%q)", owner, repo)
	}
	if !hasLabel([]string{"Bug", ":product"}, "bug") {
		t.Fatalf("hasLabel should match normalized labels")
	}
	if !containsNormalized([]string{"g-orchestration"}, "#g-orchestration") {
		t.Fatalf("containsNormalized should match normalized labels")
	}
	if esc := urlQueryEscape("a b"); esc != "a+b" {
		t.Fatalf("urlQueryEscape=%q", esc)
	}
}

func TestUpdatesTimestampCheck(t *testing.T) {
	now := time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC)
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != updatesTimestampURL {
			return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
		}
		return jsonResponse(t, http.StatusOK, map[string]any{
			"signed": map[string]any{
				"expires": now.Add(7 * 24 * time.Hour).Format(time.RFC3339),
			},
		}), nil
	}))
	okResult := checkUpdatesTimestamp(context.Background(), now)
	if !okResult.OK || okResult.Error != "" {
		t.Fatalf("expected OK timestamp check, got %+v", okResult)
	}

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, map[string]any{
			"signed": map[string]any{"expires": "not-a-time"},
		}), nil
	}))
	badResult := checkUpdatesTimestamp(context.Background(), now)
	if badResult.Error == "" {
		t.Fatalf("expected parse error for invalid expires")
	}
}

func TestSprintMutationHelpers(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.github.com/graphql" {
			return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
		}
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), "updateProjectV2ItemFieldValue") {
			t.Fatalf("expected updateProjectV2ItemFieldValue in mutation body")
		}
		return jsonResponse(t, http.StatusOK, map[string]any{"data": map[string]any{}}), nil
	}))

	if err := setCurrentSprintForItem("tok", githubv4.ID("P1"), githubv4.ID("I1"), githubv4.ID("F1"), "ITER"); err != nil {
		t.Fatalf("setCurrentSprintForItem err=%v", err)
	}

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, map[string]any{
			"errors": []map[string]any{{"message": "boom"}},
		}), nil
	}))
	if err := githubGraphQLMutation("tok", "mutation { x }"); err == nil {
		t.Fatalf("expected graphql error")
	}
}

func TestAssigneeMilestoneAndReleaseStoryFlows(t *testing.T) {
	client := newGraphQLStubClient(t)
	ctx := context.Background()

	origSearchAssigned := searchAssignedIssuesByProject
	searchAssignedIssuesByProject = func(ctx context.Context, token, org string, projectNum int) []searchIssueItem {
		return []searchIssueItem{
			{
				Number:        9999,
				Title:         "Assigned via search",
				HTMLURL:       "https://github.com/fleetdm/fleet/issues/9999",
				RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
				Assignees:     []struct{ Login string `json:"login"` }{{Login: "sharon-fdm"}},
			},
		}
	}
	t.Cleanup(func() { searchAssignedIssuesByProject = origSearchAssigned })

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// Switch dispatches mocked REST endpoints used by this integration-style test:
		// - /user: viewer login
		// - /assignees: assignable users
		// - /milestones: milestone suggestions
		// - /search/issues: release-story TODO search results
		// - default: 404 to surface unexpected endpoint usage.
		switch {
		case r.URL.String() == "https://api.github.com/user":
			return jsonResponse(t, http.StatusOK, map[string]any{"login": "sharon-fdm"}), nil
		case strings.Contains(r.URL.Path, "/assignees"):
			return jsonResponse(t, http.StatusOK, []map[string]any{
				{"login": "zoe"},
				{"login": "alice"},
				{"login": "Alice"},
			}), nil
		case strings.Contains(r.URL.Path, "/milestones"):
			return jsonResponse(t, http.StatusOK, []map[string]any{
				{"number": 1, "title": "4.80.0"},
				{"number": 2, "title": "4.80.0"},
				{"number": 3, "title": "4.81.0"},
			}), nil
		case strings.Contains(r.URL.Path, "/search/issues"):
			return jsonResponse(t, http.StatusOK, searchIssueResponse{
				Items: []searchIssueItem{
					{
						Number:        33512,
						Title:         "Story TODO",
						HTMLURL:       "https://github.com/fleetdm/fleet/issues/33512",
						Body:          "TODO: fill details",
						State:         "open",
						RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
						Labels: []struct{ Name string `json:"name"` }{
							{Name: "story"},
							{Name: ":release"},
						},
					},
				},
			}), nil
		default:
			return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
		}
	}))

	if login := fetchViewerLogin(ctx, "tok"); login != "sharon-fdm" {
		t.Fatalf("fetchViewerLogin=%q", login)
	}
	assignees := fetchRepoAssignees(ctx, "tok", "fleetdm", "fleet")
	if len(assignees) != 2 || assignees[0].Login != "alice" {
		t.Fatalf("unexpected assignees: %#v", assignees)
	}
	ms := fetchAllMilestones(ctx, "tok", "fleetdm", "fleet")
	if len(ms) != 2 {
		t.Fatalf("unexpected milestones: %#v", ms)
	}

	assignedSearch := fetchAssignedIssuesByProject(ctx, "tok", "fleetdm", 97)
	if len(assignedSearch) != 1 {
		t.Fatalf("fetchAssignedIssuesByProject len=%d want 1", len(assignedSearch))
	}

	missingAssignee := runMissingAssigneeChecks(ctx, client, "fleetdm", []int{71}, 20, "tok")
	if len(missingAssignee) == 0 {
		t.Fatalf("expected missing/assigned findings")
	}
	foundAssigned := false
	for _, it := range missingAssignee {
		if it.AssignedToMe {
			foundAssigned = true
		}
	}
	if !foundAssigned {
		t.Fatalf("expected at least one AssignedToMe finding")
	}

	missingMilestones := runMissingMilestoneChecks(ctx, client, "fleetdm", []int{71}, 20, "tok", nil)
	if len(missingMilestones) == 0 || len(missingMilestones[0].SuggestedMilestones) == 0 {
		t.Fatalf("expected missing milestone findings with suggestions")
	}

	releaseTODO := runReleaseStoryTODOChecks(ctx, client, "fleetdm", []int{97}, 20, "tok", nil)
	if len(releaseTODO) != 1 {
		t.Fatalf("expected one release story TODO finding, got %d", len(releaseTODO))
	}

	matches := issueMatchingGroups([]string{"#g-orchestration", "bug"}, []string{"g-orchestration", "g-security-compliance"})
	if len(matches) != 1 || matches[0] != "g-orchestration" {
		t.Fatalf("unexpected issueMatchingGroups: %#v", matches)
	}
}

func newSprintStubClient(t *testing.T, now time.Time) *githubv4.Client {
	t.Helper()
	hc := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			var req struct {
				Query     string                 `json:"query"`
				Variables map[string]interface{} `json:"variables"`
			}
			_ = json.Unmarshal(raw, &req)

			if strings.Contains(req.Query, "projectV2(number:$number)") || strings.Contains(req.Query, "projectV2(number: $number)") {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"data": map[string]any{
						"organization": map[string]any{
							"projectV2": map[string]any{
								"id": "P71",
								"fields": map[string]any{
									"nodes": []map[string]any{
										{
											"__typename": "ProjectV2SingleSelectField",
											"id":         "STATUS_FIELD",
											"name":       "Status",
										},
										{
											"__typename": "ProjectV2IterationField",
											"id":         "SPRINT_FIELD",
											"name":       "Sprint",
											"configuration": map[string]any{
												"iterations": []map[string]any{
													{
														"id":        "ITER_1",
														"title":     "Sprint 1",
														"startDate": now.AddDate(0, 0, -1).Format("2006-01-02"),
														"duration":  14,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}), nil
			}

			if strings.Contains(req.Query, "node(id: $id)") {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"data": map[string]any{
						"node": map[string]any{
							"items": map[string]any{
								"nodes": []map[string]any{
									issueNode(4242, "Needs sprint", "https://github.com/fleetdm/fleet/issues/4242", "Waiting", "body", nil, now, nil),
									{
										"id":        "ITEM_DONE",
										"updatedAt": now.Format(time.RFC3339),
										"content": map[string]any{
											"number":    4243,
											"title":     "Done item",
											"body":      "body",
											"url":       "https://github.com/fleetdm/fleet/issues/4243",
											"milestone": map[string]any{"title": ""},
											"labels":    map[string]any{"nodes": []map[string]any{}},
											"assignees": map[string]any{"nodes": []map[string]any{}},
										},
										"fieldValues": map[string]any{
											"nodes": []map[string]any{
												{
													"name": "Done",
													"field": map[string]any{
														"id":   "STATUS_FIELD",
														"name": "Status",
													},
												},
											},
										},
									},
									{
										"id":        "ITEM_WITH_SPRINT",
										"updatedAt": now.Format(time.RFC3339),
										"content": map[string]any{
											"number":    4244,
											"title":     "Has sprint",
											"body":      "body",
											"url":       "https://github.com/fleetdm/fleet/issues/4244",
											"milestone": map[string]any{"title": ""},
											"labels":    map[string]any{"nodes": []map[string]any{}},
											"assignees": map[string]any{"nodes": []map[string]any{}},
										},
										"fieldValues": map[string]any{
											"nodes": []map[string]any{
												{
													"name": "Waiting",
													"field": map[string]any{
														"id":   "STATUS_FIELD",
														"name": "Status",
													},
												},
												{
													"iterationId": "ITER_1",
													"title":       "Sprint 1",
													"field": map[string]any{
														"id":   "SPRINT_FIELD",
														"name": "Sprint",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}), nil
			}
			return jsonResponse(t, http.StatusOK, map[string]any{"data": map[string]any{}}), nil
		}),
	}
	return githubv4.NewClient(hc)
}

func TestRunMissingSprintChecksAndRefreshHelpers(t *testing.T) {
	now := time.Now().UTC()
	client := newSprintStubClient(t, now)
	if cfg, ok := fetchSprintProjectConfig(context.Background(), client, "fleetdm", 71, now); !ok || cfg.CurrentIterID == "" {
		t.Fatalf("expected sprint config, got ok=%v cfg=%+v", ok, cfg)
	}
	out := runMissingSprintChecks(context.Background(), client, "fleetdm", []int{71}, 20, nil)
	if len(out) != 1 {
		t.Fatalf("expected one missing sprint violation, got %d", len(out))
	}
	if out[0].CurrentSprintID == "" || out[0].CurrentSprint == "" {
		t.Fatalf("expected current sprint to be populated: %+v", out[0])
	}

	b := newTestBridge()
	if b.sessionToken() != "sess" {
		t.Fatalf("session token mismatch")
	}
	b.setTimestampRefresher(func(context.Context) (TimestampCheckResult, error) {
		return TimestampCheckResult{OK: true}, nil
	})
	b.setUnreleasedRefresher(func(context.Context) ([]UnassignedUnreleasedProjectReport, error) {
		return nil, nil
	})
	b.setReleaseStoryTODORefresher(func(context.Context) ([]ReleaseStoryTODOProjectReport, error) {
		return nil, nil
	})
	b.setMissingSprintRefresher(func(context.Context) ([]MissingSprintProjectReport, map[string]sprintApplyTarget, error) {
		return nil, nil, nil
	})
	if err := b.refreshAllIfRequested(context.Background(), true); err != nil {
		t.Fatalf("refreshAllIfRequested err=%v", err)
	}
	done := make(chan struct{})
	go func() {
		b.closeDone()
		close(done)
	}()
	_ = b.waitUntilDone(context.Background())
	<-done
}

func TestFetchUnreleasedIssuesByGroupAndFetchAllItemsPaging(t *testing.T) {
	pageCalls := 0
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/search/issues") {
			pageCalls++
			page := r.URL.Query().Get("page")
			if page == "2" {
				return jsonResponse(t, http.StatusOK, searchIssueResponse{Items: []searchIssueItem{}}), nil
			}
			return jsonResponse(t, http.StatusOK, searchIssueResponse{
				Items: []searchIssueItem{
					{
						Number:        1,
						Title:         "Bug A",
						HTMLURL:       "https://github.com/fleetdm/fleet/issues/1",
						State:         "open",
						RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
					},
					{
						Number:        1,
						Title:         "Bug A duplicate",
						HTMLURL:       "https://github.com/fleetdm/fleet/issues/1",
						State:         "open",
						RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
					},
				},
			}), nil
		}
		return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
	}))

	items := fetchUnreleasedIssuesByGroup(context.Background(), "tok", "fleetdm", "#g-orchestration")
	if len(items) != 1 {
		t.Fatalf("expected deduped single unreleased issue, got %d", len(items))
	}
	if pageCalls == 0 {
		t.Fatalf("expected search API calls")
	}

	hc := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			var req struct {
				Variables map[string]any `json:"variables"`
			}
			_ = json.Unmarshal(raw, &req)
			after := ""
			if v, ok := req.Variables["after"]; ok && v != nil {
				after, _ = v.(string)
			}
			if after == "" {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"data": map[string]any{
						"node": map[string]any{
							"items": map[string]any{
								"nodes": []map[string]any{
									issueNode(5001, "Page1", "https://github.com/fleetdm/fleet/issues/5001", "Waiting", "b", nil, time.Now().UTC(), nil),
								},
								"pageInfo": map[string]any{"hasNextPage": true, "endCursor": "CUR_1"},
							},
						},
					},
				}), nil
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"data": map[string]any{
					"node": map[string]any{
						"items": map[string]any{
							"nodes": []map[string]any{
								issueNode(5002, "Page2", "https://github.com/fleetdm/fleet/issues/5002", "Waiting", "b", nil, time.Now().UTC(), nil),
							},
							"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
						},
					},
				},
			}), nil
		}),
	}
	all := fetchAllItems(context.Background(), githubv4.NewClient(hc), githubv4.ID("PX"))
	if len(all) != 2 {
		t.Fatalf("fetchAllItems expected 2, got %d", len(all))
	}
}

func TestBridgeMutationSuccessPaths(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// Switch provides happy-path mocked API responses for each bridge mutation:
		// issue read/update, assignee add, release-label add/remove, and sprint GraphQL.
		// Default branch returns OK so non-critical extra requests do not fail the test.
		switch {
		case strings.Contains(r.URL.Path, "/issues/1") && r.Method == http.MethodGet:
			return jsonResponse(t, http.StatusOK, map[string]any{
				"body": "- [ ] check one\nrest",
			}), nil
		case strings.Contains(r.URL.Path, "/issues/1") && (r.Method == http.MethodPatch || r.Method == http.MethodDelete):
			return jsonResponse(t, http.StatusOK, map[string]any{}), nil
		case strings.Contains(r.URL.Path, "/issues/1/assignees") && r.Method == http.MethodPost:
			return jsonResponse(t, http.StatusCreated, map[string]any{}), nil
		case strings.Contains(r.URL.Path, "/issues/1/labels") && r.Method == http.MethodPost:
			return jsonResponse(t, http.StatusCreated, map[string]any{}), nil
		case r.URL.String() == "https://api.github.com/graphql":
			return jsonResponse(t, http.StatusOK, map[string]any{"data": map[string]any{}}), nil
		default:
			return jsonResponse(t, http.StatusOK, map[string]any{}), nil
		}
	}))

	b := newTestBridge()
	b.token = "tok"

	cases := []struct {
		path string
		body string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{
			path: "/api/apply-milestone",
			body: `{"repo":"fleetdm/fleet","issue":"1","milestone_number":10}`,
			fn:   b.handleApplyMilestone,
		},
		{
			path: "/api/apply-checklist",
			body: `{"repo":"fleetdm/fleet","issue":"1","check_text":"check one"}`,
			fn:   b.handleApplyChecklist,
		},
		{
			path: "/api/apply-sprint",
			body: `{"item_id":"ITEM_1"}`,
			fn:   b.handleApplySprint,
		},
		{
			path: "/api/add-assignee",
			body: `{"repo":"fleetdm/fleet","issue":"1","assignee":"alice"}`,
			fn:   b.handleAddAssignee,
		},
		{
			path: "/api/apply-release-label",
			body: `{"repo":"fleetdm/fleet","issue":"1"}`,
			fn:   b.handleApplyReleaseLabel,
		},
	}
	for i, tc := range cases {
		rr := httptest.NewRecorder()
		tc.fn(rr, postReq(tc.path, tc.body))
		if rr.Code != http.StatusOK {
			t.Fatalf("case %d %s status=%d body=%s", i, tc.path, rr.Code, rr.Body.String())
		}
	}

	if got := callerAddr(&http.Request{RemoteAddr: "127.0.0.1:8080"}); !strings.Contains(got, "loopback") {
		t.Fatalf("callerAddr(loopback)=%q", got)
	}
	if got := callerAddr(&http.Request{RemoteAddr: "example"}); got != "example" {
		t.Fatalf("callerAddr(raw)=%q", got)
	}
}

func TestBridgeMutationErrorBranches(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// Switch forces specific API failures per endpoint so each handler's
		// bad-gateway error path is exercised.
		switch {
		case strings.Contains(r.URL.Path, "/issues/1") && r.Method == http.MethodPatch:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		case strings.Contains(r.URL.Path, "/issues/1") && r.Method == http.MethodGet:
			return jsonResponse(t, http.StatusOK, map[string]any{"body": "- [ ] check one"}), nil
		case strings.Contains(r.URL.Path, "/issues/1/assignees"):
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		case strings.Contains(r.URL.Path, "/issues/1/labels") && r.Method == http.MethodPost:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("boom")),
			}, nil
		case r.URL.String() == "https://api.github.com/graphql":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"errors":[{"message":"bad"}]}`)),
			}, nil
		default:
			return jsonResponse(t, http.StatusOK, map[string]any{}), nil
		}
	}))

	b := newTestBridge()
	b.token = "tok"
	tests := []struct {
		path string
		body string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{"/api/apply-milestone", `{"repo":"fleetdm/fleet","issue":"1","milestone_number":10}`, b.handleApplyMilestone},
		{"/api/apply-checklist", `{"repo":"fleetdm/fleet","issue":"1","check_text":"check one"}`, b.handleApplyChecklist},
		{"/api/apply-sprint", `{"item_id":"ITEM_1"}`, b.handleApplySprint},
		{"/api/add-assignee", `{"repo":"fleetdm/fleet","issue":"1","assignee":"alice"}`, b.handleAddAssignee},
		{"/api/apply-release-label", `{"repo":"fleetdm/fleet","issue":"1"}`, b.handleApplyReleaseLabel},
	}
	for _, tc := range tests {
		rr := httptest.NewRecorder()
		tc.fn(rr, postReq(tc.path, tc.body))
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("%s expected 502, got %d", tc.path, rr.Code)
		}
	}
}

func TestChecklistHandlerNoChangeBranches(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/issues/1") && r.Method == http.MethodGet {
			return jsonResponse(t, http.StatusOK, map[string]any{
				"body": "- [x] check one\n",
			}), nil
		}
		return jsonResponse(t, http.StatusOK, map[string]any{}), nil
	}))
	b := newTestBridge()
	b.token = "tok"

	rr := httptest.NewRecorder()
	b.handleApplyChecklist(rr, postReq("/api/apply-checklist", `{"repo":"fleetdm/fleet","issue":"1","check_text":"check one"}`))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"already_checked":true`) {
		t.Fatalf("expected already_checked branch, got code=%d body=%s", rr.Code, rr.Body.String())
	}

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/issues/1") && r.Method == http.MethodGet {
			return jsonResponse(t, http.StatusOK, map[string]any{
				"body": "- [ ] different item\n",
			}), nil
		}
		return jsonResponse(t, http.StatusOK, map[string]any{}), nil
	}))
	rr = httptest.NewRecorder()
	b.handleApplyChecklist(rr, postReq("/api/apply-checklist", `{"repo":"fleetdm/fleet","issue":"1","check_text":"check one"}`))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"updated":false`) {
		t.Fatalf("expected no-update branch, got code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRunUnassignedUnreleasedMergesMatchingGroups(t *testing.T) {
	orig := searchUnreleasedIssuesByGroup
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
		base := []struct {
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
				Number:        1234,
				Title:         "same issue",
				HTMLURL:       "https://github.com/fleetdm/fleet/issues/1234",
				State:         "open",
				RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
				Assignees:     []struct{ Login string `json:"login"` }{{Login: "alice"}},
				Labels:        []struct{ Name string `json:"name"` }{{Name: "bug"}},
			},
		}
		return base
	}
	t.Cleanup(func() { searchUnreleasedIssuesByGroup = orig })

	out := runUnassignedUnreleasedBugChecks(
		context.Background(),
		nil,
		"fleetdm",
		[]int{71},
		20,
		"tok",
		compileLabelFilter([]string{"g-orchestration", "g-security-compliance"}),
		[]string{"g-orchestration", "g-security-compliance"},
	)
	if len(out) != 1 {
		t.Fatalf("expected one deduped issue, got %d", len(out))
	}
	if len(out[0].MatchingGroups) != 2 {
		t.Fatalf("expected merged matching groups, got %#v", out[0].MatchingGroups)
	}
	if out[0].Unassigned {
		t.Fatalf("expected assigned issue to be marked as non-unassigned")
	}
}

func TestStartUIBridgeSmoke(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "index.html")
	if err := os.WriteFile(reportPath, []byte("<html>ok</html>"), 0o600); err != nil {
		t.Fatalf("write report: %v", err)
	}

	bridge, err := startUIBridge("tok", time.Minute, nil, bridgePolicy{})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("bridge socket not permitted in this environment: %v", err)
		}
		t.Fatalf("startUIBridge: %v", err)
	}
	bridge.setReportPath(reportPath)
	defer func() { _ = bridge.stop("test done") }()

	resp, err := http.Get(bridge.baseURL + "/healthz")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status=%d", resp.StatusCode)
	}

	resp, err = http.Get(bridge.baseURL + "/report")
	if err != nil {
		t.Fatalf("report request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("report status=%d", resp.StatusCode)
	}
}

func TestMiscHelperCoverageBoost(t *testing.T) {
	if mustGithubInt(5) != githubv4.Int(5) {
		t.Fatalf("mustGithubInt mismatch")
	}
	if _, err := toGithubInt(int(math.MaxInt32) + 1); err == nil {
		t.Fatalf("expected out-of-range error")
	}

	if got := orderedGroupLabels([]string{"#g-orchestration", " g-orchestration ", "#g-security-compliance"}); len(got) != 2 {
		t.Fatalf("orderedGroupLabels dedupe failed: %#v", got)
	}
	var nilLabels *stringListFlag
	if got := nilLabels.String(); got != "" {
		t.Fatalf("nil stringListFlag String should be empty, got %q", got)
	}
	if got := formatGroupLabelForQuery(""); got != "" {
		t.Fatalf("formatGroupLabelForQuery empty mismatch: %q", got)
	}

	if got := coloredBar(10, 0, 0, clrGreen); !strings.Contains(got, "░") {
		t.Fatalf("coloredBar pending expected: %q", got)
	}
	if got := coloredBar(10, 20, 10, clrGreen); !strings.Contains(got, "█") {
		t.Fatalf("coloredBar fill expected: %q", got)
	}

	it := testIssueWithStatus(1, "x", "https://github.com/fleetdm/fleet/issues/1", "✔️Awaiting QA")
	if isStaleAwaitingQA(it, time.Now().UTC(), 24*time.Hour) {
		t.Fatalf("zero UpdatedAt should not be stale")
	}
	it.UpdatedAt.Time = time.Now().UTC().Add(-48 * time.Hour)
	if !isStaleAwaitingQA(it, time.Now().UTC(), 24*time.Hour) {
		t.Fatalf("expected stale awaiting item")
	}

	if got := previewBodyLines("\n a \n\n b \n", 0); got != nil {
		t.Fatalf("expected nil for maxLines<=0")
	}
	var blank Item
	if st := itemStatus(blank); st != "" {
		t.Fatalf("empty item status should be blank: %q", st)
	}
}

func TestTimestampNon200Branch(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("nope")),
		}, nil
	}))
	result := checkUpdatesTimestamp(context.Background(), time.Now().UTC())
	if result.Error == "" {
		t.Fatalf("expected non-200 timestamp error")
	}
}

func TestReleaseLabelNoopAndWaitUntilDoneContext(t *testing.T) {
	b := newTestBridge()
	b.allowRelease = map[string]releaseLabelTarget{
		issueKey("fleetdm/fleet", 1): {NeedsProductRemoval: false, NeedsReleaseAdd: false},
	}
	rr := httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":"fleetdm/fleet","issue":"1"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected noop release label apply to succeed, got %d", rr.Code)
	}

	b2 := newTestBridge()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reason := b2.waitUntilDone(ctx)
	if !strings.Contains(reason, "interrupted") {
		t.Fatalf("expected interrupted reason, got %q", reason)
	}
}
