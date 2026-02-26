package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestStartUIBridgeBindsToLoopback provides scrumcheck behavior for this unit.
func TestStartUIBridgeBindsToLoopback(t *testing.T) {
	t.Parallel()

	b, err := startUIBridge("dummy-token", time.Minute, nil, bridgePolicy{})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "permission denied") {
			t.Skipf("listener creation blocked in sandbox: %v", err)
		}
		t.Fatalf("startUIBridge() error: %v", err)
	}
	defer func() { _ = b.stop("test done") }()

	if !strings.Contains(b.baseURL, "127.0.0.1:") {
		t.Fatalf("expected loopback base URL, got %q", b.baseURL)
	}
}

// TestPrepareRequestRejectsForeignOrigin provides scrumcheck behavior for this unit.
func TestPrepareRequestRejectsForeignOrigin(t *testing.T) {
	t.Parallel()

	b := &uiBridge{
		baseURL: "http://127.0.0.1:9999",
		origin:  "http://127.0.0.1:9999",
		session: "abc123",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/apply-milestone", strings.NewReader(`{}`))
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("X-Qacheck-Session", "abc123")
	rr := httptest.NewRecorder()

	ok := b.prepareRequest(rr, req)
	if ok {
		t.Fatal("expected request to be rejected")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

// TestPrepareRequestRejectsForeignReferer provides scrumcheck behavior for this unit.
func TestPrepareRequestRejectsForeignReferer(t *testing.T) {
	t.Parallel()

	b := &uiBridge{
		baseURL: "http://127.0.0.1:9999",
		origin:  "http://127.0.0.1:9999",
		session: "abc123",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/apply-checklist", strings.NewReader(`{}`))
	req.Header.Set("Origin", "http://127.0.0.1:9999")
	req.Header.Set("Referer", "http://evil.example/report")
	req.Header.Set("X-Qacheck-Session", "abc123")
	rr := httptest.NewRecorder()

	ok := b.prepareRequest(rr, req)
	if ok {
		t.Fatal("expected request to be rejected")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

// TestPrepareRequestRejectsMissingSessionHeader provides scrumcheck behavior for this unit.
func TestPrepareRequestRejectsMissingSessionHeader(t *testing.T) {
	t.Parallel()

	b := &uiBridge{
		baseURL: "http://127.0.0.1:9999",
		origin:  "http://127.0.0.1:9999",
		session: "abc123",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/apply-checklist", strings.NewReader(`{}`))
	req.Header.Set("Origin", "http://127.0.0.1:9999")
	req.Header.Set("Referer", "http://127.0.0.1:9999/report")
	rr := httptest.NewRecorder()

	ok := b.prepareRequest(rr, req)
	if ok {
		t.Fatal("expected request to be rejected")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

// TestPrepareRequestAcceptsValidLocalRequest provides scrumcheck behavior for this unit.
func TestPrepareRequestAcceptsValidLocalRequest(t *testing.T) {
	t.Parallel()

	b := &uiBridge{
		baseURL: "http://127.0.0.1:9999",
		origin:  "http://127.0.0.1:9999",
		session: "abc123",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/apply-checklist", strings.NewReader(`{}`))
	req.Header.Set("Origin", "http://127.0.0.1:9999")
	req.Header.Set("Referer", "http://127.0.0.1:9999/report")
	req.Header.Set("X-Qacheck-Session", "abc123")
	rr := httptest.NewRecorder()

	ok := b.prepareRequest(rr, req)
	if !ok {
		t.Fatalf("expected request to pass, got status %d", rr.Code)
	}
}
