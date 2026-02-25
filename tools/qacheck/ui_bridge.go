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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type uiBridge struct {
	baseURL     string
	session     string
	token       string
	idleTimeout time.Duration
	onEvent     func(string)

	srv      *http.Server
	listener net.Listener

	mu     sync.Mutex
	timer  *time.Timer
	done   chan struct{}
	reason string
}

func startUIBridge(token string, idleTimeout time.Duration, onEvent func(string)) (*uiBridge, error) {
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
		baseURL:     "http://" + ln.Addr().String(),
		session:     session,
		token:       token,
		idleTimeout: idleTimeout,
		onEvent:     onEvent,
		listener:    ln,
		done:        make(chan struct{}),
		reason:      "bridge closed",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/apply-milestone", b.handleApplyMilestone)
	mux.HandleFunc("/api/apply-checklist", b.handleApplyChecklist)
	mux.HandleFunc("/healthz", b.handleHealth)
	b.srv = &http.Server{Handler: mux}
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

func (b *uiBridge) waitUntilDone(ctx context.Context) string {
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

func (b *uiBridge) closeDone() {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
}

func (b *uiBridge) touch() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.timer != nil {
		b.timer.Reset(b.idleTimeout)
	}
}

func (b *uiBridge) signal(msg string) {
	if b.onEvent != nil {
		b.onEvent(msg)
	}
}

func (b *uiBridge) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (b *uiBridge) handleApplyMilestone(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
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

	issueNum, err := strconv.Atoi(req.Issue)
	if err != nil || issueNum <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}

	b.signal(fmt.Sprintf("📡 UI cmd: apply milestone #%d to %s#%d", req.MilestoneNumber, req.Repo, issueNum))
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", req.Repo, issueNum)
	payload := map[string]any{"milestone": req.MilestoneNumber}
	if err := b.githubJSON(r.Context(), http.MethodPatch, endpoint, payload, nil); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (b *uiBridge) handleApplyChecklist(w http.ResponseWriter, r *http.Request) {
	if !b.prepareRequest(w, r) {
		return
	}
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
	issueNum, err := strconv.Atoi(req.Issue)
	if err != nil || issueNum <= 0 {
		http.Error(w, "invalid issue number", http.StatusBadRequest)
		return
	}

	b.signal(fmt.Sprintf("🛰️ UI cmd: apply checklist check on %s#%d", req.Repo, issueNum))
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", req.Repo, issueNum)
	var issueResp struct {
		Body string `json:"body"`
	}
	if err := b.githubJSON(r.Context(), http.MethodGet, endpoint, nil, &issueResp); err != nil {
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
		return
	}

	if err := b.githubJSON(r.Context(), http.MethodPatch, endpoint, map[string]any{"body": updatedBody}, nil); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"updated":         true,
		"already_checked": false,
	})
}

func (b *uiBridge) prepareRequest(w http.ResponseWriter, r *http.Request) bool {
	writeCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	if r.Header.Get("X-QACheck-Session") != b.session {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	b.touch()
	return true
}

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

func writeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-QACheck-Session")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	writeCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func randomHex(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func replaceUncheckedChecklistLine(body string, checkText string) (string, bool, bool) {
	text := strings.TrimSpace(checkText)
	if text == "" {
		return body, false, false
	}

	escaped := regexp.QuoteMeta(text)
	checkedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?im)(^|\n)\s*[-*]\s*\[x\]\s*` + escaped + `(?=\n|$)`),
		regexp.MustCompile(`(?im)(^|\n)\s*\[x\]\s*` + escaped + `(?=\n|$)`),
	}
	for _, p := range checkedPatterns {
		if p.MatchString(body) {
			return body, false, true
		}
	}

	unchecked := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{
			pattern:     regexp.MustCompile(`(?im)(^|\n)\s*-\s*\[ \]\s*` + escaped + `(?=\n|$)`),
			replacement: `$1- [x] ` + text,
		},
		{
			pattern:     regexp.MustCompile(`(?im)(^|\n)\s*\*\s*\[ \]\s*` + escaped + `(?=\n|$)`),
			replacement: `$1* [x] ` + text,
		},
		{
			pattern:     regexp.MustCompile(`(?im)(^|\n)\s*\[ \]\s*` + escaped + `(?=\n|$)`),
			replacement: `$1[x] ` + text,
		},
	}
	for _, p := range unchecked {
		if p.pattern.MatchString(body) {
			return p.pattern.ReplaceAllString(body, p.replacement), true, false
		}
	}
	return body, false, false
}
