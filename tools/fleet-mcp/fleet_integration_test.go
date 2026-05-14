package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func newTestClient(serverURL string) *FleetClient {
	return &FleetClient{
		baseURL:    serverURL,
		apiKey:     "test",
		httpClient: http.DefaultClient,
	}
}

func TestIsTempQueryName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"global temp", tempQueryNamePrefix + "1234-abc", true},
		{"team-scoped temp", "[Workstations] " + tempQueryNamePrefix + "1234-abc", true},
		{"team-scoped temp with emoji", "[💻 Workstations] " + tempQueryNamePrefix + "1234-abc", true},
		{"unrelated global query", "Top-level CPU usage", false},
		{"unrelated team query", "[Servers] Disk space check", false},
		{"prefix substring not at start", "prefixed-" + tempQueryNamePrefix + "abc", false},
		{"empty", "", false},
		{"just brackets", "[abc]", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTempQueryName(tc.in); got != tc.want {
				t.Errorf("isTempQueryName(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestEndpointMatchesHostname(t *testing.T) {
	cases := []struct {
		name string
		ep   Endpoint
		in   string
		want bool
	}{
		{
			name: "matches Name exactly",
			ep:   Endpoint{Name: "alpha.local"},
			in:   "alpha.local",
			want: true,
		},
		{
			name: "matches ComputerName case-insensitively",
			ep:   Endpoint{ComputerName: "MyMac"},
			in:   "mymac",
			want: true,
		},
		{
			name: "matches DisplayName",
			ep:   Endpoint{DisplayName: "USS Protostar"},
			in:   "USS Protostar",
			want: true,
		},
		{
			name: "no match — substring on serial only",
			ep:   Endpoint{Name: "host123.local", HardwareSerial: "trex-serial"},
			in:   "trex",
			want: false,
		},
		{
			name: "no match — substring on IP only",
			ep:   Endpoint{Name: "host.local", PrimaryIP: "192.168.1.42"},
			in:   "192.168",
			want: false,
		},
		{
			name: "different hostname does not match",
			ep:   Endpoint{Name: "alpha.local", ComputerName: "alpha", DisplayName: "Alpha"},
			in:   "beta.local",
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := endpointMatchesHostname(tc.ep, tc.in); got != tc.want {
				t.Errorf("endpointMatchesHostname(%+v, %q) = %v, want %v", tc.ep, tc.in, got, tc.want)
			}
		})
	}
}

func TestFetchHostsFromPathBounded_PaginatesUntilShortPage(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		page := r.URL.Query().Get("page")
		var n int
		switch page {
		case "0":
			n = 500
		case "1":
			n = 200
		default:
			t.Errorf("unexpected page %q", page)
			http.Error(w, "unexpected page", http.StatusBadRequest)
			return
		}
		hosts := make([]Endpoint, n)
		for i := range hosts {
			hosts[i] = Endpoint{ID: uint(i + 1)}
		}
		_ = json.NewEncoder(w).Encode(struct {
			Hosts []Endpoint `json:"hosts"`
		}{Hosts: hosts})
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	out, truncated, err := fc.fetchHostsFromPathBounded(context.Background(), "/api/v1/fleet/hosts", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Errorf("expected truncated=false")
	}
	if got, want := len(out), 700; got != want {
		t.Errorf("len(out) = %d, want %d", got, want)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 page calls, got %d", got)
	}
}

func TestFetchHostsFromPathBounded_HardCapTruncates(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		hosts := make([]Endpoint, 500)
		for i := range hosts {
			hosts[i] = Endpoint{ID: uint(n)*1000 + uint(i+1)}
		}
		_ = json.NewEncoder(w).Encode(struct {
			Hosts []Endpoint `json:"hosts"`
		}{Hosts: hosts})
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	out, truncated, err := fc.fetchHostsFromPathBounded(context.Background(), "/api/v1/fleet/hosts", 600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated {
		t.Errorf("expected truncated=true")
	}
	if got, want := len(out), 600; got != want {
		t.Errorf("len(out) = %d, want %d (cap)", got, want)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 page calls before cap kicks in, got %d", got)
	}
}

func TestGetVulnerabilityImpact_PropagatesTruncated(t *testing.T) {
	// Lower the cap so a small mock host set trips truncation.
	orig := fetchHostsHardCap
	fetchHostsHardCap = 5
	t.Cleanup(func() { fetchHostsHardCap = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/fleet/hosts":
			// Step 3: return more hosts than the cap (set to 5 above) so the
			// page-truncate branch fires and sets truncated=true.
			hosts := make([]Endpoint, 10)
			for i := range hosts {
				hosts[i] = Endpoint{ID: uint(i + 1)}
			}
			_ = json.NewEncoder(w).Encode(struct {
				Hosts []Endpoint `json:"hosts"`
			}{Hosts: hosts})
		case strings.HasPrefix(r.URL.Path, "/api/v1/fleet/software/titles/"):
			// Step 2: one version per title.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"software_title": map[string]any{
					"versions": []map[string]any{{"id": 99}},
				},
			})
		case r.URL.Path == "/api/v1/fleet/software/titles":
			// Step 1: one title, short page → stop.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"software_titles": []map[string]any{{"id": 1}},
			})
		case r.URL.Path == "/api/v1/fleet/hosts/count":
			_ = json.NewEncoder(w).Encode(map[string]any{"count": 1000})
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	impact, err := fc.GetVulnerabilityImpact(context.Background(), "CVE-2026-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !impact.Truncated {
		t.Errorf("expected Truncated=true to propagate from per-version-id fetch")
	}
	if impact.ImpactedSystems == 0 {
		t.Errorf("expected ImpactedSystems > 0, got %d", impact.ImpactedSystems)
	}
}

func TestBearerAuthMiddleware(t *testing.T) {
	const token = "secret-token"
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	h := bearerAuthMiddleware(token, next)

	cases := []struct {
		name       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{"missing header", "", http.StatusUnauthorized, false},
		{"wrong scheme", "Basic " + token, http.StatusUnauthorized, false},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized, false},
		{"correct token", "Bearer " + token, http.StatusOK, true},
		{"trailing junk", "Bearer " + token + "x", http.StatusUnauthorized, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest("GET", "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Errorf("next called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestRateLimiterMiddleware_BurstThen429(t *testing.T) {
	rl := newIPRateLimiter(1, 2) // 2-token bucket, 1 rps refill
	allowed := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { allowed++ })
	h := rl.Middleware(next)

	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "203.0.113.7:5000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// First 2 in burst should pass.
	if rec := send(); rec.Code != http.StatusOK {
		t.Errorf("burst req 1: status = %d, want 200", rec.Code)
	}
	if rec := send(); rec.Code != http.StatusOK {
		t.Errorf("burst req 2: status = %d, want 200", rec.Code)
	}
	// 3rd within the same instant should 429 with Retry-After.
	rec := send()
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Errorf("missing Retry-After header on 429")
	}
	if allowed != 2 {
		t.Errorf("next called %d times, want 2", allowed)
	}
}

func TestValidateCVEID(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"CVE-2026-12345", false},
		{"CVE-1999-0001", false},     // 4-digit minimum
		{"  CVE-2026-12345  ", false}, // trims
		{"", true},
		{"   ", true},
		{"cve-2026-12345", true},   // case-sensitive
		{"CVE-26-12345", true},     // year too short
		{"CVE-2026-123", true},     // suffix too short
		{"CVE-2026-12345x", true},  // trailing junk
		{"CVE-2026", true},         // missing suffix
		{"<script>", true},         // injection-shaped junk
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateCVEID(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateCVEID(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}

func TestParsePositiveUintString(t *testing.T) {
	cases := []struct {
		in      string
		wantN   uint64
		wantErr bool
	}{
		{"1", 1, false},
		{"42", 42, false},
		{"  42  ", 42, false},
		{"0", 0, true},
		{"", 0, true},
		{"   ", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"1.5", 0, true},
		{"1e2", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			n, err := parsePositiveUintString("policy_id", tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if n != tc.wantN {
				t.Errorf("n=%d, want %d", n, tc.wantN)
			}
		})
	}
}

func TestGetHostsForCVE_PaginatesTitles(t *testing.T) {
	var titlesCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/fleet/software/titles/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"software_title": map[string]any{"versions": []any{}},
			})
		case r.URL.Path == "/api/v1/fleet/software/titles":
			titlesCalls.Add(1)
			page := r.URL.Query().Get("page")
			n, _ := strconv.Atoi(page)
			var count int
			switch n {
			case 0:
				count = 100
			case 1:
				count = 30
			default:
				t.Errorf("unexpected titles page %d", n)
				http.Error(w, "unexpected page", http.StatusBadRequest)
				return
			}
			type title struct {
				ID uint `json:"id"`
			}
			titles := make([]title, count)
			for i := range titles {
				titles[i].ID = uint(n*1000 + i + 1)
			}
			_ = json.NewEncoder(w).Encode(struct {
				SoftwareTitles []title `json:"software_titles"`
			}{SoftwareTitles: titles})
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	hosts, truncated, err := fc.GetHostsForCVE(context.Background(), "CVE-2026-12345", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Errorf("expected truncated=false (no per-version-id fan-out hit cap)")
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts (titles had no versions), got %d", len(hosts))
	}
	if got := titlesCalls.Load(); got != 2 {
		t.Errorf("expected 2 titles pages (100 + 30 short page), got %d", got)
	}
}

func TestGetPolicies_IncludesQueryField(t *testing.T) {
	const wantSQL = "SELECT 1;"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/fleet/global/policies":
			_, _ = w.Write([]byte(`{"policies":[
				{"id":1,"name":"with sql","query":"` + wantSQL + `"},
				{"id":2,"name":"empty sql","query":""}
			]}`))
		case "/api/v1/fleet/teams":
			_, _ = w.Write([]byte(`{"teams":[]}`))
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	policies, err := fc.GetPolicies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if policies[0].Query != wantSQL {
		t.Errorf("policy 1 Query = %q, want %q", policies[0].Query, wantSQL)
	}
	if policies[1].Query != "" {
		t.Errorf("policy 2 Query = %q, want empty string", policies[1].Query)
	}
}
