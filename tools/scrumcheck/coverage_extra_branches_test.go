package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckUpdatesTimestampBranches(t *testing.T) {
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusInternalServerError, map[string]any{"error": "x"}), nil
	}))
	if got := checkUpdatesTimestamp(context.Background(), time.Now().UTC()); got.Error == "" {
		t.Fatal("expected status-code error")
	}

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("{bad-json")),
		}, nil
	}))
	if got := checkUpdatesTimestamp(context.Background(), time.Now().UTC()); !strings.Contains(got.Error, "parse timestamp.json") {
		t.Fatalf("expected parse error, got %q", got.Error)
	}

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, map[string]any{
			"signed": map[string]any{"expires": "not-a-time"},
		}), nil
	}))
	if got := checkUpdatesTimestamp(context.Background(), time.Now().UTC()); !strings.Contains(got.Error, "parse expires value") {
		t.Fatalf("expected expires parse error, got %q", got.Error)
	}

	future := time.Now().UTC().Add(40 * 24 * time.Hour).Format(time.RFC3339)
	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, map[string]any{
			"signed": map[string]any{"expires": future},
		}), nil
	}))
	got := checkUpdatesTimestamp(context.Background(), time.Now().UTC())
	if got.Error != "" || !got.OK {
		t.Fatalf("expected successful timestamp check, got %#v", got)
	}
}

func TestPrepareRequestBranches(t *testing.T) {
	b := &uiBridge{
		baseURL: "http://127.0.0.1:12345",
		origin:  "http://127.0.0.1:12345",
		session: "sess",
	}

	makeReq := func(method string) *http.Request {
		r := httptest.NewRequest(method, "/api/x", nil)
		r.Host = "127.0.0.1:12345"
		r.Header.Set("X-Qacheck-Session", "sess")
		return r
	}

	rr := httptest.NewRecorder()
	r := makeReq(http.MethodPost)
	r.Header.Set("Origin", "http://evil.example")
	if b.prepareRequest(rr, r) {
		t.Fatal("expected forbidden origin")
	}

	rr = httptest.NewRecorder()
	r = makeReq(http.MethodPost)
	r.Header.Set("Referer", "http://evil.example/report")
	if b.prepareRequest(rr, r) {
		t.Fatal("expected forbidden referer")
	}

	rr = httptest.NewRecorder()
	r = makeReq(http.MethodOptions)
	if b.prepareRequest(rr, r) {
		t.Fatal("expected options to be rejected")
	}

	rr = httptest.NewRecorder()
	r = makeReq(http.MethodGet)
	if b.prepareRequest(rr, r) {
		t.Fatal("expected method rejection")
	}

	rr = httptest.NewRecorder()
	r = makeReq(http.MethodPost)
	r.Header.Del("X-Qacheck-Session")
	if b.prepareRequest(rr, r) {
		t.Fatal("expected unauthorized without session")
	}

	rr = httptest.NewRecorder()
	r = makeReq(http.MethodPost)
	if !b.prepareRequest(rr, r) {
		t.Fatalf("expected success prepareRequest, status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleApplyReleaseLabelValidationBranches(t *testing.T) {
	b := newTestBridge()

	rr := httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid json bad request, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":"","issue":""}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected required fields bad request, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	b.handleApplyReleaseLabel(rr, postReq("/api/apply-release-label", `{"repo":"bad/repo/slug","issue":"1"}`))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid repo slug bad request, got %d", rr.Code)
	}
}
