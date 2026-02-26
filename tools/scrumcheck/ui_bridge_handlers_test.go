package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
		reason: "bridge closed",
		done:   make(chan struct{}),
		srv:    &http.Server{},
	}
}

func postReq(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Origin", "http://127.0.0.1:9999")
	req.Header.Set("Referer", "http://127.0.0.1:9999/report")
	req.Header.Set("X-Qacheck-Session", "sess")
	return req
}

func TestHandleReportValidation(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/report", nil)
	b.handleReport(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusMethodNotAllowed)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/report", nil)
	b.handleReport(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusServiceUnavailable)
	}
}

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

func TestHandleCloseAndHealth(t *testing.T) {
	t.Parallel()
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleHealth(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}

	rr = httptest.NewRecorder()
	b.handleClose(rr, postReq("/api/close", `{"reason":"done"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
}
