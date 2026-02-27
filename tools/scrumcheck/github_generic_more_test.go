package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

func TestFetchProjectIDAndFetchItems(t *testing.T) {
	client := githubv4.NewClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			var req struct {
				Query string `json:"query"`
			}
			_ = json.Unmarshal(raw, &req)

			if strings.Contains(req.Query, "projectV2(number: $num)") {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"data": map[string]any{
						"organization": map[string]any{
							"projectV2": map[string]any{"id": "PROJ_71"},
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
									issueNode(1234, "Issue", "https://github.com/fleetdm/fleet/issues/1234", "Waiting", "body", nil, time.Now().UTC(), nil),
								},
							},
						},
					},
				}), nil
			}
			return jsonResponse(t, http.StatusOK, map[string]any{"data": map[string]any{}}), nil
		}),
	})

	projectID := fetchProjectID(context.Background(), client, "fleetdm", 71)
	if fmt.Sprintf("%v", projectID) != "PROJ_71" {
		t.Fatalf("unexpected project id: %q", projectID)
	}
	items := fetchItems(context.Background(), client, projectID, 10)
	if len(items) != 1 || getNumber(items[0]) != 1234 {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestFetchProjectIDAndFetchItemsFailurePaths(t *testing.T) {
	client := githubv4.NewClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(`{"errors":[{"message":"boom"}]}`)),
			}, nil
		}),
	})

	projectID := fetchProjectID(context.Background(), client, "fleetdm", 71)
	if projectID != "" {
		t.Fatalf("expected empty project id on error, got %q", projectID)
	}
	if items := fetchItems(context.Background(), client, "P", 10); len(items) != 0 {
		t.Fatalf("expected no items on error, got %d", len(items))
	}
}

func TestMustGithubIntOutOfRange(t *testing.T) {
	if got := mustGithubInt(1 << 40); got != 0 {
		t.Fatalf("expected zero for out-of-range conversion, got %d", got)
	}
	if got := mustGithubInt(42); got != 42 {
		t.Fatalf("expected valid conversion, got %d", got)
	}
}

func TestExpandGenericQueryTemplateBranches(t *testing.T) {
	if out := expandGenericQueryTemplate("", []int{97}, []string{"g-orchestration"}); len(out) != 0 {
		t.Fatalf("expected empty for blank template, got %#v", out)
	}
	if out := expandGenericQueryTemplate("label:<<group>>", []int{97}, nil); len(out) != 0 {
		t.Fatalf("expected skip when group placeholder has no groups, got %#v", out)
	}
	if out := expandGenericQueryTemplate("project:<<project>>", nil, []string{"g-orchestration"}); len(out) != 0 {
		t.Fatalf("expected skip when project placeholder has no projects, got %#v", out)
	}

	out := expandGenericQueryTemplate("label:<<group>> project:<<project>>", []int{71, 97}, []string{"g-orchestration"})
	if len(out) != 2 {
		t.Fatalf("expected 2 expanded queries, got %d", len(out))
	}
	if !strings.Contains(out[0], "\"#g-orchestration\"") {
		t.Fatalf("expected normalized quoted group label, got %q", out[0])
	}
}

func TestFetchGenericQueryIssuesMappingAndDedup(t *testing.T) {
	calls := 0
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.Path, "/search/issues") {
			return jsonResponse(t, http.StatusNotFound, map[string]any{}), nil
		}
		calls++
		return jsonResponse(t, http.StatusOK, searchIssueResponse{
			Items: []searchIssueItem{
				{
					Number:        10,
					Title:         "Valid",
					HTMLURL:       "https://github.com/fleetdm/fleet/issues/10",
					State:         "OPEN",
					RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
					Labels: []struct {
						Name string `json:"name"`
					}{
						{Name: "bug"},
						{Name: ""},
					},
					Assignees: []struct {
						Login string `json:"login"`
					}{
						{Login: "alice"},
						{Login: ""},
					},
				},
				{
					Number:        10, // duplicate key; should be dropped
					Title:         "Duplicate",
					HTMLURL:       "https://github.com/fleetdm/fleet/issues/10",
					State:         "open",
					RepositoryURL: "https://api.github.com/repos/fleetdm/fleet",
				},
				{
					Number:        11, // invalid repo URL; should be skipped
					Title:         "Bad repo url",
					HTMLURL:       "https://github.com/fleetdm/fleet/issues/11",
					State:         "open",
					RepositoryURL: "bad",
				},
			},
		}), nil
	}))

	items := fetchGenericQueryIssues(context.Background(), "tok", `is:open is:issue`)
	if calls == 0 {
		t.Fatal("expected at least one search call")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 deduped valid item, got %d (%#v)", len(items), items)
	}
	if items[0].Status != "Open" {
		t.Fatalf("expected title-cased status, got %q", items[0].Status)
	}
	if len(items[0].CurrentLabels) != 1 || items[0].CurrentLabels[0] != "bug" {
		t.Fatalf("unexpected labels mapping: %#v", items[0].CurrentLabels)
	}
	if len(items[0].CurrentAssignees) != 1 || items[0].CurrentAssignees[0] != "alice" {
		t.Fatalf("unexpected assignees mapping: %#v", items[0].CurrentAssignees)
	}
}

func TestExecuteIssueSearchRequestFallbackAndNoToken(t *testing.T) {
	callCount := 0
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		callCount++
		if r.Header.Get("Authorization") != "" {
			return jsonResponse(t, http.StatusForbidden, map[string]any{"message": "forbidden"}), nil
		}
		return jsonResponse(t, http.StatusOK, searchIssueResponse{
			Items: []searchIssueItem{{Number: 1, RepositoryURL: "https://api.github.com/repos/fleetdm/fleet"}},
		}), nil
	}))

	body, ok := executeIssueSearchRequest(context.Background(), "https://api.github.com/search/issues?q=x", "token")
	if !ok || len(body.Items) != 1 {
		t.Fatalf("expected fallback success, ok=%v body=%#v", ok, body)
	}
	if callCount < 2 {
		t.Fatalf("expected retried request, calls=%d", callCount)
	}

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusForbidden, map[string]any{"message": "forbidden"}), nil
	}))
	if _, ok := executeIssueSearchRequest(context.Background(), "https://api.github.com/search/issues?q=x", ""); ok {
		t.Fatal("expected failure without token fallback")
	}
}

func TestHandleApplyReleaseLabelDelete404Allowed(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(r.URL.Path, "/issues/1/labels") && r.Method == http.MethodPost:
			return jsonResponse(t, http.StatusCreated, map[string]any{}), nil
		case strings.Contains(r.URL.Path, "/issues/1/labels/") && r.Method == http.MethodDelete:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("missing")),
			}, nil
		default:
			return jsonResponse(t, http.StatusOK, map[string]any{}), nil
		}
	}))

	b := newTestBridge()
	b.allowRelease = map[string]releaseLabelTarget{
		issueKey("fleetdm/fleet", 1): {
			NeedsReleaseAdd:     true,
			NeedsProductRemoval: true,
		},
	}
	rr := httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":"fleetdm/fleet","issue":"1"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 when delete returns 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOpenInBrowserErrorWhenLauncherMissing(t *testing.T) {
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	_ = os.Setenv("PATH", "/path/that/does/not/exist")

	if err := openInBrowser("about:blank"); err == nil {
		t.Fatal("expected openInBrowser to fail when launcher is unavailable")
	}
}
