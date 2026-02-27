package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestBridge creates a bridge instance configured for handler testing.
func newTestBridge() *uiBridge {
	return &uiBridge{
		baseURL: "http://127.0.0.1:9999",
		origin:  "http://127.0.0.1:9999",
		session: "sess",
		allowMilestones: map[string]map[int]bool{
			issueKey("fleetdm/fleet", 1): {10: true},
		},
		allowChecklist: map[string]map[string]bool{
			issueKey("fleetdm/fleet", 1): {"check one": true},
		},
		allowAssignees: map[string]map[string]bool{
			issueKey("fleetdm/fleet", 1): {"alice": true},
		},
		allowSprints: map[string]sprintApplyTarget{
			"ITEM_1": {ProjectID: "PROJ", FieldID: "FIELD", IterationID: "ITER"},
		},
		allowRelease: map[string]releaseLabelTarget{
			issueKey("fleetdm/fleet", 1): {NeedsProductRemoval: true, NeedsReleaseAdd: true},
		},
		sseSubscribers: make(map[chan string]struct{}),
		reason:         "bridge closed",
		done:           make(chan struct{}),
		srv:            &http.Server{},
	}
}

// postReq builds a JSON POST request with local/session headers.
func postReq(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Host = "127.0.0.1:9999"
	req.Header.Set("Origin", "http://127.0.0.1:9999")
	req.Header.Set("Referer", "http://127.0.0.1:9999/report")
	req.Header.Set("X-Qacheck-Session", "sess")
	return req
}

// getReq builds a local GET request that passes host validation.
func getReq(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = "127.0.0.1:9999"
	return req
}

// TestHandleReportValidation verifies /report request validation rules.
func TestHandleReportValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/report", nil)
	req.Host = "127.0.0.1:9999"
	b.handleReport(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = getReq("/report")
	b.handleReport(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
}

// TestHandleAssets verifies static frontend assets are served from /assets/*.
func TestHandleAssets(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	req := getReq("/assets/app.js")
	b.handleAsset(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "renderDraftingFromState") {
		t.Fatalf("expected app.js payload, got %q", rr.Body.String())
	}
}

// TestHandleEvents verifies SSE endpoint authorization and initial frame.
func TestHandleEvents(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	req := getReq("/api/events")
	b.handleEvents(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rr = httptest.NewRecorder()
	req = getReq("/api/events").WithContext(ctx)
	req.Header.Set("X-Qacheck-Session", "sess")
	done := make(chan struct{})
	go func() {
		b.handleEvents(rr, req)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "event: ready") {
		t.Fatalf("expected initial SSE frame, got %q", rr.Body.String())
	}
}

// TestHandleApplyMilestoneValidation verifies milestone mutation guardrails.
func TestHandleApplyMilestoneValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleApplyMilestone(rr, postReq("/api/apply-milestone", `{"repo":"bad/slug/extra","issue":"1","milestone_number":10}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}

	rr = httptest.NewRecorder()
	b.handleApplyMilestone(rr, postReq("/api/apply-milestone", `{"repo":"fleetdm/fleet","issue":"abc","milestone_number":10}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}

	rr = httptest.NewRecorder()
	b.handleApplyMilestone(rr, postReq("/api/apply-milestone", `{"repo":"fleetdm/fleet","issue":"2","milestone_number":10}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusForbidden)
	}
}

// TestHandleChecklistValidation verifies checklist mutation guardrails.
func TestHandleChecklistValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleApplyChecklist(rr, postReq("/api/apply-checklist", `{"repo":"fleetdm/fleet","issue":"2","check_text":"check one"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusForbidden)
	}

	rr = httptest.NewRecorder()
	b.handleApplyChecklist(rr, postReq("/api/apply-checklist", `{"repo":"fleetdm/fleet","issue":"x","check_text":"check one"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

// TestHandleSprintValidation verifies sprint mutation guardrails.
func TestHandleSprintValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleApplySprint(rr, postReq("/api/apply-sprint", `{"item_id":""}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}

	rr = httptest.NewRecorder()
	b.handleApplySprint(rr, postReq("/api/apply-sprint", `{"item_id":"ITEM_2"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusForbidden)
	}
}

// TestHandleAssigneeValidation verifies assignee mutation guardrails.
func TestHandleAssigneeValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleAddAssignee(rr, postReq("/api/add-assignee", `{"repo":"fleetdm/fleet","issue":"1","assignee":"bob"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusForbidden)
	}

	rr = httptest.NewRecorder()
	b.handleAddAssignee(rr, postReq("/api/add-assignee", `{"repo":"fleetdm/fleet","issue":"x","assignee":"alice"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

// TestHandleReleaseValidation verifies release-label mutation guardrails.
func TestHandleReleaseValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":"fleetdm/fleet","issue":"2"}`))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusForbidden)
	}

	rr = httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":"fleetdm/fleet","issue":"x"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

// TestHandleCloseAndHealth verifies close and health endpoints.
func TestHandleCloseAndHealth(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleHealth(rr, getReq("/healthz"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}

	rr = httptest.NewRecorder()
	b.handleClose(rr, postReq("/api/close", `{"reason":"done"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
}

// TestHandleTimestampCheck verifies timestamp check endpoint responses.
func TestHandleTimestampCheck(t *testing.T) {
	t.Parallel()
	b := newTestBridge()
	b.setTimestampCheckResult(TimestampCheckResult{
		URL:          updatesTimestampURL,
		ExpiresAt:    time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		DurationLeft: 48 * time.Hour,
		DaysLeft:     2.0,
		MinDays:      5,
		OK:           false,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/check/timestamp", nil)
	req.Host = "127.0.0.1:9999"
	b.handleTimestampCheck(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/timestamp")
	b.handleTimestampCheck(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/timestamp")
	req.Header.Set("X-Qacheck-Session", "sess")
	b.handleTimestampCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), updatesTimestampURL) {
		t.Fatalf("expected response to include timestamp URL, got %s", rr.Body.String())
	}

	b.setRefreshAllState(func(context.Context) (HTMLReportData, bridgePolicy, error) {
		return HTMLReportData{
			TimestampCheck: TimestampCheckResult{
				URL:     "https://updates.fleetdm.com/timestamp.json",
				MinDays: 5,
				OK:      true,
			},
		}, bridgePolicy{}, nil
	})
	rr = httptest.NewRecorder()
	req = getReq("/api/check/timestamp?refresh=1")
	req.Header.Set("X-Qacheck-Session", "sess")
	b.handleTimestampCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Fatalf("expected refreshed payload to set ok=true, got %s", rr.Body.String())
	}
}

// TestHandleUnassignedUnreleasedCheck verifies unreleased-bug refresh endpoint.
func TestHandleUnassignedUnreleasedCheck(t *testing.T) {
	t.Parallel()
	b := newTestBridge()
	b.setUnassignedUnreleasedResults([]UnassignedUnreleasedProjectReport{
		{
			GroupLabel: "g-orchestration",
			Columns: []UnassignedUnreleasedStatusReport{
				{
					Key:   "awaiting-qa",
					Label: "Awaiting QA",
					RedItems: []MissingMilestoneReportItem{
						{
							Number:    40408,
							Title:     "Newly created policy does not appear until refresh",
							URL:       "https://github.com/fleetdm/fleet/issues/40408",
							Repo:      "fleetdm/fleet",
							Status:    "Awaiting QA",
							Assignees: nil,
							Labels:    []string{"g-orchestration", "~unreleased bug"},
						},
					},
				},
			},
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/check/unassigned-unreleased", nil)
	req.Host = "127.0.0.1:9999"
	b.handleUnassignedUnreleasedCheck(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/unassigned-unreleased")
	b.handleUnassignedUnreleasedCheck(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/unassigned-unreleased")
	req.Header.Set("X-Qacheck-Session", "sess")
	b.handleUnassignedUnreleasedCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "g-orchestration") || !strings.Contains(rr.Body.String(), "40408") {
		t.Fatalf("expected response payload with group/issue, got %s", rr.Body.String())
	}
}

// TestHandleReleaseStoryTODOCheck verifies release-story TODO refresh endpoint.
func TestHandleReleaseStoryTODOCheck(t *testing.T) {
	t.Parallel()
	b := newTestBridge()
	b.setReleaseStoryTODOResults([]ReleaseStoryTODOProjectReport{
		{
			ProjectNum: 97,
			Columns: []MissingMilestoneGroupReport{
				{
					Key:   "in_review",
					Label: "In review",
					Items: []MissingMilestoneReportItem{
						{
							Number:      37498,
							Title:       "Team maintainers can not add Certificate templates",
							URL:         "https://github.com/fleetdm/fleet/issues/37498",
							Repo:        "fleetdm/fleet",
							Status:      "In review",
							Assignees:   []string{"sharon-fdm"},
							Labels:      []string{":release", "story"},
							BodyPreview: []string{"- TODO: finalize copy"},
						},
					},
				},
			},
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/check/release-story-todo", nil)
	req.Host = "127.0.0.1:9999"
	b.handleReleaseStoryTODOCheck(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/release-story-todo")
	b.handleReleaseStoryTODOCheck(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/release-story-todo")
	req.Header.Set("X-Qacheck-Session", "sess")
	b.handleReleaseStoryTODOCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "37498") || !strings.Contains(rr.Body.String(), "ProjectNum") {
		t.Fatalf("expected response payload with project/issue, got %s", rr.Body.String())
	}
}

// TestHandleMissingSprintCheck verifies missing-sprint refresh endpoint.
func TestHandleMissingSprintCheck(t *testing.T) {
	t.Parallel()
	b := newTestBridge()
	b.setMissingSprintResults([]MissingSprintProjectReport{
		{
			ProjectNum: 71,
			Columns: []MissingSprintGroupReport{
				{
					Key:   "in_review",
					Label: "In review",
					Items: []MissingSprintReportItem{
						{
							ProjectNum:    71,
							ItemID:        "ITEM_40408",
							Number:        40408,
							Title:         "Newly created policy does not appear until refresh",
							URL:           "https://github.com/fleetdm/fleet/issues/40408",
							Status:        "In review",
							CurrentSprint: "",
							Milestone:     "4.82.0",
						},
					},
				},
			},
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/check/missing-sprint", nil)
	req.Host = "127.0.0.1:9999"
	b.handleMissingSprintCheck(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/missing-sprint")
	b.handleMissingSprintCheck(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/missing-sprint")
	req.Header.Set("X-Qacheck-Session", "sess")
	b.handleMissingSprintCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ITEM_40408") || !strings.Contains(rr.Body.String(), "40408") {
		t.Fatalf("expected response payload with item and issue, got %s", rr.Body.String())
	}
}

// TestHandleStateCheck verifies refresh-all state endpoint reporting.
func TestHandleStateCheck(t *testing.T) {
	t.Parallel()
	b := newTestBridge()
	b.setReportData(HTMLReportData{
		Org:                   "fleetdm",
		TotalNoSprint:         1,
		SprintClean:           false,
		TotalReleaseStoryTODO: 2,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/check/state", nil)
	req.Host = "127.0.0.1:9999"
	b.handleStateCheck(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/state")
	b.handleStateCheck(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}

	rr = httptest.NewRecorder()
	req = getReq("/api/check/state")
	req.Header.Set("X-Qacheck-Session", "sess")
	b.handleStateCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"TotalNoSprint":1`) {
		t.Fatalf("expected state payload with TotalNoSprint=1, got %s", rr.Body.String())
	}
}
