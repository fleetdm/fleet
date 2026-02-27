package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	neturl "net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shurcooL/githubv4"
)

const maxBridgeBodyBytes = 16 * 1024

var repoSlugPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type uiBridge struct {
	baseURL     string
	session     string
	token       string
	idleTimeout time.Duration
	onEvent     func(string)

	srv        *http.Server
	listener   net.Listener
	reportPath string
	origin     string

	mu     sync.Mutex
	timer  *time.Timer
	done   chan struct{}
	reason string

	allowChecklist       map[string]map[string]bool
	allowMilestones      map[string]map[int]bool
	allowAssignees       map[string]map[string]bool
	allowSprints         map[string]sprintApplyTarget
	allowRelease         map[string]releaseLabelTarget
	timestampCheck       TimestampCheckResult
	unreleasedBugs       []UnassignedUnreleasedProjectReport
	releaseStoryTODO     []ReleaseStoryTODOProjectReport
	missingSprint        []MissingSprintProjectReport
	reportData           HTMLReportData
	refreshTimestamp     func(context.Context) (TimestampCheckResult, error)
	refreshUnreleased    func(context.Context) ([]UnassignedUnreleasedProjectReport, error)
	refreshReleaseTODO   func(context.Context) ([]ReleaseStoryTODOProjectReport, error)
	refreshMissingSprint func(context.Context) ([]MissingSprintProjectReport, map[string]sprintApplyTarget, error)
	refreshAllState      func(context.Context) (HTMLReportData, bridgePolicy, error)
}

// startUIBridge starts the local bridge server and wires all HTTP handlers.
func startUIBridge(token string, idleTimeout time.Duration, onEvent func(string), policy bridgePolicy) (*uiBridge, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("missing token")
	}
	if idleTimeout < time.Minute {
		idleTimeout = 15 * time.Minute
	}

	session, err := randomHex(18)
	if err != nil {
		return nil, fmt.Errorf("generate bridge session: %w", err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen bridge: %w", err)
	}

	b := &uiBridge{
		baseURL:         "http://" + ln.Addr().String(),
		origin:          "http://" + ln.Addr().String(),
		session:         session,
		token:           token,
		idleTimeout:     idleTimeout,
		onEvent:         onEvent,
		listener:        ln,
		done:            make(chan struct{}),
		reason:          "bridge closed",
		allowChecklist:  policy.ChecklistByIssue,
		allowMilestones: policy.MilestonesByIssue,
		allowAssignees:  policy.AssigneesByIssue,
		allowSprints:    policy.SprintsByItemID,
		allowRelease:    policy.ReleaseByIssue,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/report", b.handleReport)
	mux.HandleFunc("/api/check/timestamp", b.handleTimestampCheck)
	mux.HandleFunc("/api/check/unassigned-unreleased", b.handleUnassignedUnreleasedCheck)
	mux.HandleFunc("/api/check/release-story-todo", b.handleReleaseStoryTODOCheck)
	mux.HandleFunc("/api/check/missing-sprint", b.handleMissingSprintCheck)
	mux.HandleFunc("/api/check/state", b.handleStateCheck)
	mux.HandleFunc("/api/apply-milestone", b.handleApplyMilestone)
	mux.HandleFunc("/api/apply-checklist", b.handleApplyChecklist)
	mux.HandleFunc("/api/apply-sprint", b.handleApplySprint)
	mux.HandleFunc("/api/add-assignee", b.handleAddAssignee)
	mux.HandleFunc("/api/apply-release-label", b.handleApplyReleaseLabel)
	mux.HandleFunc("/api/close", b.handleClose)
	mux.HandleFunc("/healthz", b.handleHealth)
	b.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	b.timer = time.AfterFunc(idleTimeout, func() {
		b.signal("⌛ UI uplink idle timeout reached")
		_ = b.stop("idle timeout")
	})

	go func() {
		err := b.srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			b.signal("🔴 UI uplink server error: " + err.Error())
			_ = b.stop("server error")
			return
		}
		b.closeDone()
	}()

	return b, nil
}

// setReportPath stores the current report path and precomputes its URL form.
func (b *uiBridge) setReportPath(path string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reportPath = path
}

// reportURL returns the bridge-served report URL for browser navigation.
func (b *uiBridge) reportURL() string {
	return b.baseURL + "/report"
}

// sessionToken returns the per-process session token for bridge API calls.
func (b *uiBridge) sessionToken() string {
	return b.session
}

// setTimestampCheckResult updates cached timestamp-check output for refresh APIs.
func (b *uiBridge) setTimestampCheckResult(result TimestampCheckResult) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timestampCheck = result
}

// setUnassignedUnreleasedResults updates cached unreleased-bug check results.
func (b *uiBridge) setUnassignedUnreleasedResults(results []UnassignedUnreleasedProjectReport) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.unreleasedBugs = results
}

// setReleaseStoryTODOResults updates cached release-story TODO check results.
func (b *uiBridge) setReleaseStoryTODOResults(results []ReleaseStoryTODOProjectReport) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.releaseStoryTODO = results
}

// setMissingSprintResults updates cached missing-sprint check results.
func (b *uiBridge) setMissingSprintResults(results []MissingSprintProjectReport) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.missingSprint = results
}

// setReportData stores the latest rendered report model for state inspection.
func (b *uiBridge) setReportData(data HTMLReportData) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reportData = data
}

// setTimestampRefresher registers the callback used by /api/check/timestamp.
func (b *uiBridge) setTimestampRefresher(fn func(context.Context) (TimestampCheckResult, error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshTimestamp = fn
}

// setUnreleasedRefresher registers the callback used by unreleased-bug refresh.
func (b *uiBridge) setUnreleasedRefresher(fn func(context.Context) ([]UnassignedUnreleasedProjectReport, error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshUnreleased = fn
}

// setReleaseStoryTODORefresher registers the callback used by release-story refresh.
func (b *uiBridge) setReleaseStoryTODORefresher(fn func(context.Context) ([]ReleaseStoryTODOProjectReport, error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshReleaseTODO = fn
}

// setMissingSprintRefresher registers the callback used by missing-sprint refresh.
func (b *uiBridge) setMissingSprintRefresher(fn func(context.Context) ([]MissingSprintProjectReport, map[string]sprintApplyTarget, error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshMissingSprint = fn
}

// setRefreshAllState updates state for the global refresh-all operation.
func (b *uiBridge) setRefreshAllState(fn func(context.Context) (HTMLReportData, bridgePolicy, error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refreshAllState = fn
}

// refreshAllIfRequested triggers a full recompute when explicitly requested.
func (b *uiBridge) refreshAllIfRequested(ctx context.Context, refresh bool) error {
	if !refresh {
		return nil
	}
	b.mu.Lock()
	refreshFn := b.refreshAllState
	b.mu.Unlock()
	if refreshFn == nil {
		return nil
	}
	data, policy, err := refreshFn(ctx)
	if err != nil {
		return err
	}

	b.mu.Lock()
	b.reportData = data
	b.timestampCheck = data.TimestampCheck
	b.unreleasedBugs = data.UnassignedUnreleased
	b.releaseStoryTODO = data.ReleaseStoryTODO
	b.missingSprint = data.MissingSprint
	b.allowChecklist = policy.ChecklistByIssue
	b.allowMilestones = policy.MilestonesByIssue
	b.allowAssignees = policy.AssigneesByIssue
	b.allowSprints = policy.SprintsByItemID
	b.allowRelease = policy.ReleaseByIssue
	b.mu.Unlock()

	if path, err := writeHTMLReport(data); err == nil {
		b.mu.Lock()
		b.reportPath = path
		b.mu.Unlock()
	}
	return nil
}

// stop gracefully shuts down the bridge server and marks it closed.
func (b *uiBridge) stop(reason string) error {
	b.mu.Lock()
	if b.reason != "bridge closed" {
		b.mu.Unlock()
		return nil
	}
	b.reason = reason
	if b.timer != nil {
		b.timer.Stop()
	}
	b.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := b.srv.Shutdown(ctx)
	b.closeDone()
	return err
}

// waitUntilDone blocks until the bridge done channel is closed.
func (b *uiBridge) waitUntilDone(ctx context.Context) string {
	// Select waits for whichever happens first:
	// - ctx.Done(): caller requested shutdown (for example Ctrl+C); we emit a
	//   signal and trigger bridge stop.
	// - b.done: bridge already finished (server exited or stop was called).
	// After this first wait, we read b.done once more to guarantee the channel is
	// closed before returning the final reason. That second receive is safe even
	// when already closed (it returns immediately), and avoids a race where stop
	// was initiated but the done-close hasn't been observed yet.
	select {
	case <-ctx.Done():
		b.signal("🧯 Shutdown signal received (Ctrl+C)")
		_ = b.stop("interrupted (Ctrl+C)")
	case <-b.done:
	}
	<-b.done
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.reason
}

// closeDone idempotently closes the done channel.
func (b *uiBridge) closeDone() {
	// Select implements idempotent close:
	// - if b.done is already closed, receiving from it succeeds immediately and
	//   we do nothing.
	// - otherwise default branch runs and closes b.done exactly once.
	select {
	case <-b.done:
	default:
		close(b.done)
	}
}

// touch records client activity to support idle shutdown behavior.
func (b *uiBridge) touch() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.timer != nil {
		b.timer.Reset(b.idleTimeout)
	}
}

// signal forwards bridge activity text to progress/log consumers.
func (b *uiBridge) signal(msg string) {
	if b.onEvent != nil {
		b.onEvent(msg)
	}
}

// handleHealth serves a basic liveness endpoint.
func (b *uiBridge) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleReport serves the generated report file through the bridge.
func (b *uiBridge) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b.touch()
	b.mu.Lock()
	reportPath := b.reportPath
	b.mu.Unlock()
	if strings.TrimSpace(reportPath) == "" {
		http.Error(w, "report not ready", http.StatusServiceUnavailable)
		return
	}

	http.ServeFile(w, r, reportPath)
}

// handleTimestampCheck runs timestamp refresh and returns JSON results.
func (b *uiBridge) handleTimestampCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionHeader := strings.TrimSpace(r.Header.Get("X-Qacheck-Session"))
	if sessionHeader == "" || sessionHeader != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.touch()
	if err := b.refreshAllIfRequested(r.Context(), r.URL.Query().Get("refresh") == "1"); err != nil {
		http.Error(w, "refresh failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	b.mu.Lock()
	result := b.timestampCheck
	b.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"url":            result.URL,
		"expires_at":     result.ExpiresAt.Format(time.RFC3339),
		"duration_hours": result.DurationLeft.Hours(),
		"days_left":      result.DaysLeft,
		"min_days":       result.MinDays,
		"ok":             result.OK,
		"error":          result.Error,
	})
}

// handleUnassignedUnreleasedCheck refreshes unreleased-bug findings via API.
func (b *uiBridge) handleUnassignedUnreleasedCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionHeader := strings.TrimSpace(r.Header.Get("X-Qacheck-Session"))
	if sessionHeader == "" || sessionHeader != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.touch()
	if err := b.refreshAllIfRequested(r.Context(), r.URL.Query().Get("refresh") == "1"); err != nil {
		http.Error(w, "refresh failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	b.mu.Lock()
	payload := make([]UnassignedUnreleasedProjectReport, len(b.unreleasedBugs))
	copy(payload, b.unreleasedBugs)
	b.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"groups": payload,
	})
}

// handleReleaseStoryTODOCheck refreshes release-story TODO findings via API.
func (b *uiBridge) handleReleaseStoryTODOCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionHeader := strings.TrimSpace(r.Header.Get("X-Qacheck-Session"))
	if sessionHeader == "" || sessionHeader != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.touch()
	if err := b.refreshAllIfRequested(r.Context(), r.URL.Query().Get("refresh") == "1"); err != nil {
		http.Error(w, "refresh failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	b.mu.Lock()
	payload := make([]ReleaseStoryTODOProjectReport, len(b.releaseStoryTODO))
	copy(payload, b.releaseStoryTODO)
	b.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": payload,
	})
}

// handleMissingSprintCheck refreshes missing-sprint findings via API.
func (b *uiBridge) handleMissingSprintCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionHeader := strings.TrimSpace(r.Header.Get("X-Qacheck-Session"))
	if sessionHeader == "" || sessionHeader != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.touch()
	if err := b.refreshAllIfRequested(r.Context(), r.URL.Query().Get("refresh") == "1"); err != nil {
		http.Error(w, "refresh failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	b.mu.Lock()
	payload := make([]MissingSprintProjectReport, len(b.missingSprint))
	copy(payload, b.missingSprint)
	b.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": payload,
	})
}

// handleStateCheck returns refresh-all status and current check counters.
func (b *uiBridge) handleStateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionHeader := strings.TrimSpace(r.Header.Get("X-Qacheck-Session"))
	if sessionHeader == "" || sessionHeader != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b.touch()

	if err := b.refreshAllIfRequested(r.Context(), r.URL.Query().Get("refresh") == "1"); err != nil {
		http.Error(w, "refresh failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	b.mu.Lock()
	data := b.reportData
	b.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"state": data,
	})
}

// handleApplyMilestone validates and applies milestone mutations.
func (b *uiBridge) handleApplyMilestone(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBridgeBodyBytes)
	var req struct {
		Repo            string `json:"repo"`
		Issue           string `json:"issue"`
		MilestoneNumber int    `json:"milestone_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Repo == "" || req.Issue == "" || req.MilestoneNumber <= 0 {
		http.Error(w, "repo, issue, and milestone_number are required", http.StatusBadRequest)
		return
	}
	if !isValidRepoSlug(req.Repo) {
		http.Error(w, "invalid repo slug", http.StatusBadRequest)
		return
	}

	issueNum, err := strconv.Atoi(req.Issue)
	if err != nil || issueNum <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}
	if !b.isAllowedMilestone(req.Repo, issueNum, req.MilestoneNumber) {
		http.Error(w, "operation not allowed for this issue/milestone", http.StatusForbidden)
		return
	}

	start := time.Now()
	caller := callerAddr(r)
	b.signalBridgeOp(caller, "apply-milestone", "start", "working", req.Repo, issueNum, "")
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", req.Repo, issueNum)
	payload := map[string]any{"milestone": req.MilestoneNumber}
	if err := b.githubJSON(r.Context(), http.MethodPatch, endpoint, payload, nil); err != nil {
		b.signalBridgeOp(caller, "apply-milestone", "done", "error", req.Repo, issueNum, shortDuration(time.Since(start)))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	b.signalBridgeOp(caller, "apply-milestone", "done", "ok", req.Repo, issueNum, shortDuration(time.Since(start)))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleApplyChecklist validates and applies checklist text mutations.
func (b *uiBridge) handleApplyChecklist(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBridgeBodyBytes)
	var req struct {
		Repo      string `json:"repo"`
		Issue     string `json:"issue"`
		CheckText string `json:"check_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Repo == "" || req.Issue == "" || strings.TrimSpace(req.CheckText) == "" {
		http.Error(w, "repo, issue, and check_text are required", http.StatusBadRequest)
		return
	}
	if !isValidRepoSlug(req.Repo) {
		http.Error(w, "invalid repo slug", http.StatusBadRequest)
		return
	}
	issueNum, err := strconv.Atoi(req.Issue)
	if err != nil || issueNum <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}
	if !b.isAllowedChecklist(req.Repo, issueNum, req.CheckText) {
		http.Error(w, "operation not allowed for this issue/checklist item", http.StatusForbidden)
		return
	}

	start := time.Now()
	caller := callerAddr(r)
	b.signalBridgeOp(caller, "apply-checklist", "start", "working", req.Repo, issueNum, "")
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", req.Repo, issueNum)
	var issueResp struct {
		Body string `json:"body"`
	}
	if err := b.githubJSON(r.Context(), http.MethodGet, endpoint, nil, &issueResp); err != nil {
		b.signalBridgeOp(caller, "apply-checklist", "done", "error", req.Repo, issueNum, shortDuration(time.Since(start)))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	updatedBody, updated, alreadyChecked := replaceUncheckedChecklistLine(issueResp.Body, req.CheckText)
	if !updated {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":              true,
			"updated":         false,
			"already_checked": alreadyChecked,
		})
		b.signalBridgeOp(caller, "apply-checklist", "done", "ok", req.Repo, issueNum, shortDuration(time.Since(start)))
		return
	}

	if err := b.githubJSON(r.Context(), http.MethodPatch, endpoint, map[string]any{"body": updatedBody}, nil); err != nil {
		b.signalBridgeOp(caller, "apply-checklist", "done", "error", req.Repo, issueNum, shortDuration(time.Since(start)))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	b.signalBridgeOp(caller, "apply-checklist", "done", "ok", req.Repo, issueNum, shortDuration(time.Since(start)))
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"updated":         true,
		"already_checked": false,
	})
}

// handleClose closes the bridge session and responds with shutdown status.
func (b *uiBridge) handleClose(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBridgeBodyBytes)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "closed from UI"
	}
	b.signal(fmt.Sprintf("🧯 caller=%s requested bridge shutdown", callerAddr(r)))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	go func() {
		_ = b.stop(reason)
	}()
}

// handleApplySprint validates and applies sprint mutations.
func (b *uiBridge) handleApplySprint(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBridgeBodyBytes)
	var req struct {
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		http.Error(w, "item_id is required", http.StatusBadRequest)
		return
	}

	target, ok := b.allowedSprintForItem(itemID)
	if !ok {
		http.Error(w, "operation not allowed for this item", http.StatusForbidden)
		return
	}

	start := time.Now()
	caller := callerAddr(r)
	b.signal(fmt.Sprintf("BRIDGE_OP caller=%s op=set-sprint stage=start status=working repo=project item=%s elapsed=-", strings.ReplaceAll(caller, " ", "_"), itemID))
	if err := setCurrentSprintForItem(
		b.token,
		githubv4.ID(target.ProjectID),
		githubv4.ID(itemID),
		githubv4.ID(target.FieldID),
		target.IterationID,
	); err != nil {
		b.signal(fmt.Sprintf("BRIDGE_OP caller=%s op=set-sprint stage=done status=error repo=project item=%s elapsed=%s", strings.ReplaceAll(caller, " ", "_"), itemID, shortDuration(time.Since(start))))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	b.signal(fmt.Sprintf("BRIDGE_OP caller=%s op=set-sprint stage=done status=ok repo=project item=%s elapsed=%s", strings.ReplaceAll(caller, " ", "_"), itemID, shortDuration(time.Since(start))))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleAddAssignee validates and applies assignee mutations.
func (b *uiBridge) handleAddAssignee(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBridgeBodyBytes)
	var req struct {
		Repo     string `json:"repo"`
		Issue    string `json:"issue"`
		Assignee string `json:"assignee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Repo == "" || req.Issue == "" || strings.TrimSpace(req.Assignee) == "" {
		http.Error(w, "repo, issue, and assignee are required", http.StatusBadRequest)
		return
	}
	if !isValidRepoSlug(req.Repo) {
		http.Error(w, "invalid repo slug", http.StatusBadRequest)
		return
	}
	issueNum, err := strconv.Atoi(req.Issue)
	if err != nil || issueNum <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}
	assignee := strings.TrimSpace(req.Assignee)
	if !b.isAllowedAssignee(req.Repo, issueNum, assignee) {
		http.Error(w, "operation not allowed for this issue/assignee", http.StatusForbidden)
		return
	}

	start := time.Now()
	caller := callerAddr(r)
	b.signalBridgeOp(caller, "add-assignee", "start", "working", req.Repo, issueNum, "")
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/assignees", req.Repo, issueNum)
	payload := map[string]any{
		"assignees": []string{assignee},
	}
	if err := b.githubJSON(r.Context(), http.MethodPost, endpoint, payload, nil); err != nil {
		b.signalBridgeOp(caller, "add-assignee", "done", "error", req.Repo, issueNum, shortDuration(time.Since(start)))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	b.signalBridgeOp(caller, "add-assignee", "done", "ok", req.Repo, issueNum, shortDuration(time.Since(start)))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleApplyReleaseLabel validates and applies release label mutations.
func (b *uiBridge) handleApplyReleaseLabel(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBridgeBodyBytes)
	var req struct {
		Repo  string `json:"repo"`
		Issue string `json:"issue"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Repo == "" || req.Issue == "" {
		http.Error(w, "repo and issue are required", http.StatusBadRequest)
		return
	}
	if !isValidRepoSlug(req.Repo) {
		http.Error(w, "invalid repo slug", http.StatusBadRequest)
		return
	}
	issueNum, err := strconv.Atoi(req.Issue)
	if err != nil || issueNum <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}
	target, ok := b.allowedReleaseForIssue(req.Repo, issueNum)
	if !ok {
		http.Error(w, "operation not allowed for this issue", http.StatusForbidden)
		return
	}

	start := time.Now()
	caller := callerAddr(r)
	b.signalBridgeOp(caller, "apply-release-label", "start", "working", req.Repo, issueNum, "")
	if target.NeedsReleaseAdd {
		addEndpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/labels", req.Repo, issueNum)
		if err := b.githubJSON(r.Context(), http.MethodPost, addEndpoint, map[string]any{"labels": []string{releaseLabel}}, nil); err != nil {
			b.signalBridgeOp(caller, "apply-release-label", "done", "error", req.Repo, issueNum, shortDuration(time.Since(start)))
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
	if target.NeedsProductRemoval {
		delEndpoint := fmt.Sprintf(
			"https://api.github.com/repos/%s/issues/%d/labels/%s",
			req.Repo,
			issueNum,
			neturl.PathEscape(productLabel),
		)
		err := b.githubJSON(r.Context(), http.MethodDelete, delEndpoint, nil, nil)
		if err != nil && !strings.Contains(err.Error(), "GitHub API error 404") {
			b.signalBridgeOp(caller, "apply-release-label", "done", "error", req.Repo, issueNum, shortDuration(time.Since(start)))
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
	b.signalBridgeOp(caller, "apply-release-label", "done", "ok", req.Repo, issueNum, shortDuration(time.Since(start)))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// callerAddr extracts remote client address information for audit logs.
func callerAddr(r *http.Request) string {
	hostPort := strings.TrimSpace(r.RemoteAddr)
	if hostPort == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return hostPort
	}
	if addr.IsLoopback() {
		return hostPort + " (loopback)"
	}
	return hostPort + " (non-loopback)"
}

// prepareRequest enforces local-origin/session validation for API calls.
func (b *uiBridge) prepareRequest(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && origin != b.origin {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return false
	}
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer != "" && !strings.HasPrefix(referer, b.baseURL+"/report") {
		http.Error(w, "forbidden referer", http.StatusForbidden)
		return false
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	sessionHeader := strings.TrimSpace(r.Header.Get("X-Qacheck-Session"))
	if sessionHeader == "" || sessionHeader != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	b.touch()
	return true
}

// githubJSON sends a GitHub API request and decodes JSON response bodies.
func (b *uiBridge) githubJSON(ctx context.Context, method, endpoint string, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil {
		raw, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+b.token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("GitHub request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1200))
		return fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// writeJSON writes JSON responses with status and content-type headers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// randomHex returns a cryptographically random lowercase hex string.
func randomHex(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// isValidRepoSlug validates owner/repo slug format for safety checks.
func isValidRepoSlug(value string) bool {
	return repoSlugPattern.MatchString(strings.TrimSpace(value))
}

// issueKey builds a stable key for repo+issue identity checks.
func issueKey(repo string, issue int) string {
	return strings.ToLower(strings.TrimSpace(repo)) + "#" + strconv.Itoa(issue)
}

// isAllowedMilestone verifies milestone writes are allowed for the issue.
func (b *uiBridge) isAllowedMilestone(repo string, issue int, milestone int) bool {
	key := issueKey(repo, issue)
	choices, ok := b.allowMilestones[key]
	if !ok {
		return false
	}
	return choices[milestone]
}

// isAllowedChecklist verifies checklist writes are allowed for the issue.
func (b *uiBridge) isAllowedChecklist(repo string, issue int, checklistText string) bool {
	key := issueKey(repo, issue)
	choices, ok := b.allowChecklist[key]
	if !ok {
		return false
	}
	return choices[strings.TrimSpace(checklistText)]
}

// isAllowedAssignee verifies assignee writes are allowed for the issue.
func (b *uiBridge) isAllowedAssignee(repo string, issue int, assignee string) bool {
	key := issueKey(repo, issue)
	choices, ok := b.allowAssignees[key]
	if !ok {
		return false
	}
	return choices[strings.ToLower(strings.TrimSpace(assignee))]
}

// allowedReleaseForIssue verifies release-label writes are allowed for the issue.
func (b *uiBridge) allowedReleaseForIssue(repo string, issue int) (releaseLabelTarget, bool) {
	target, ok := b.allowRelease[issueKey(repo, issue)]
	return target, ok
}

// signalBridgeOp emits structured operation telemetry for progress UI parsing.
func (b *uiBridge) signalBridgeOp(caller, op, stage, status, repo string, issue int, elapsed string) {
	if elapsed == "" {
		elapsed = "-"
	}
	caller = strings.ReplaceAll(caller, " ", "_")
	b.signal(fmt.Sprintf(
		"BRIDGE_OP caller=%s op=%s stage=%s status=%s repo=%s issue=%d elapsed=%s",
		caller,
		op,
		stage,
		status,
		repo,
		issue,
		elapsed,
	))
}

// allowedSprintForItem verifies sprint writes are allowed for the project item.
func (b *uiBridge) allowedSprintForItem(itemID string) (sprintApplyTarget, bool) {
	target, ok := b.allowSprints[strings.TrimSpace(itemID)]
	return target, ok
}

// replaceUncheckedChecklistLine marks one matching checklist line as checked.
func replaceUncheckedChecklistLine(body string, checkText string) (string, bool, bool) {
	text := strings.TrimSpace(checkText)
	if text == "" {
		return body, false, false
	}
	lines := strings.Split(body, "\n")

	uncheckedPrefixToChecked := map[string]string{
		"- [ ] ": "- [x] ",
		"* [ ] ": "* [x] ",
		"[ ] ":   "[x] ",
	}
	checkedPrefixes := []string{
		"- [x] ", "- [X] ",
		"* [x] ", "* [X] ",
		"[x] ", "[X] ",
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range checkedPrefixes {
			if strings.HasPrefix(trimmed, prefix) && strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)) == text {
				return body, false, true
			}
		}
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for unchecked, checked := range uncheckedPrefixToChecked {
			if strings.HasPrefix(trimmed, unchecked) && strings.TrimSpace(strings.TrimPrefix(trimmed, unchecked)) == text {
				leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = leading + checked + text
				return strings.Join(lines, "\n"), true, false
			}
		}
	}

	return body, false, false
}
