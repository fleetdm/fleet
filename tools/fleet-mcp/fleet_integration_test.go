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
	"time"

	"github.com/gorilla/websocket"
)

func newTestClient(serverURL string) *FleetClient {
	return &FleetClient{
		baseURL:    serverURL,
		apiKey:     "test",
		httpClient: http.DefaultClient,
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
		{"CVE-1999-0001", false},      // 4-digit minimum
		{"  CVE-2026-12345  ", false}, // trims
		{"", true},
		{"   ", true},
		{"cve-2026-12345", true},  // case-sensitive
		{"CVE-26-12345", true},    // year too short
		{"CVE-2026-123", true},    // suffix too short
		{"CVE-2026-12345x", true}, // trailing junk
		{"CVE-2026", true},        // missing suffix
		{"<script>", true},        // injection-shaped junk
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

// campaignTestServer stands up an httptest server that answers the campaign
// create POST and upgrades the results websocket, then hands the connection to
// drive() (after consuming the auth + select_campaign handshake) so each test
// can script the frames the server sends back.
func campaignTestServer(t *testing.T, campaignID uint, drive func(conn *websocket.Conn)) *httptest.Server {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/fleet/reports/run":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"campaign": map[string]interface{}{"id": campaignID},
			})
		case r.URL.Path == "/api/v1/fleet/results/websocket":
			conn, err := up.Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("websocket upgrade: %v", err)
				return
			}
			defer conn.Close()
			// Validate the client speaks the handshake protocol the real server
			// enforces: an auth frame carrying a token, then a select_campaign
			// frame naming this campaign.
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				t.Errorf("read auth frame: %v", err)
				return
			}
			if msg["type"] != "auth" {
				t.Errorf("first frame type = %v, want auth", msg["type"])
			}
			if data, _ := msg["data"].(map[string]interface{}); data["token"] == "" || data["token"] == nil {
				t.Errorf("auth frame missing token, got %v", msg["data"])
			}
			if err := conn.ReadJSON(&msg); err != nil {
				t.Errorf("read select_campaign frame: %v", err)
				return
			}
			if msg["type"] != "select_campaign" {
				t.Errorf("second frame type = %v, want select_campaign", msg["type"])
			}
			// JSON numbers decode to float64 in an interface{} map.
			if data, _ := msg["data"].(map[string]interface{}); data["campaign_id"] != float64(campaignID) {
				t.Errorf("select_campaign campaign_id = %v, want %d", data["campaign_id"], campaignID)
			}
			drive(conn)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func writeWSFrame(t *testing.T, conn *websocket.Conn, typ string, data interface{}) {
	if err := conn.WriteJSON(map[string]interface{}{"type": typ, "data": data}); err != nil {
		t.Errorf("server write %s frame: %v", typ, err)
	}
}

// runMultiHostCampaign creates an ad-hoc campaign and streams results over the
// websocket, aggregating each host's rows into one result.
func TestRunMultiHostCampaign_AggregatesResults(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s")
	srv := campaignTestServer(t, 42, func(conn *websocket.Conn) {
		writeWSFrame(t, conn, "totals", map[string]interface{}{"count": 2, "online": 2})
		writeWSFrame(t, conn, "result", map[string]interface{}{
			"host": map[string]interface{}{"id": 10, "hostname": "h10", "display_name": "Host 10"},
			"rows": []map[string]string{{"answer": "42"}},
		})
		writeWSFrame(t, conn, "result", map[string]interface{}{
			"host": map[string]interface{}{"id": 20, "hostname": "h20", "display_name": "Host 20"},
			"rows": []map[string]string{{"answer": "43"}},
		})
		writeWSFrame(t, conn, "status", map[string]interface{}{"expected_results": 2, "actual_results": 2, "status": "finished"})
	})
	defer srv.Close()

	fc := newTestClient(srv.URL)
	nameByID := map[uint]Endpoint{10: {ID: 10, Name: "host-10"}, 20: {ID: 20, Name: "host-20"}}
	res, err := fc.runMultiHostCampaign(t.Context(), []uint{10, 20}, "SELECT 1;", nameByID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TargetedHostCount != 2 {
		t.Errorf("TargetedHostCount = %d, want 2 (from totals)", res.TargetedHostCount)
	}
	if res.RespondedHostCount != 2 {
		t.Errorf("RespondedHostCount = %d, want 2 (from status.actual_results)", res.RespondedHostCount)
	}
	if len(res.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(res.Results))
	}
	// The locally-resolved host name wins over the server-reported one.
	for _, row := range res.Results {
		if row["host_id"] == uint(10) && row["host_name"] != "host-10" {
			t.Errorf("host 10 name = %v, want host-10", row["host_name"])
		}
	}
}

// Offline hosts never report, so the stream stops once every online host has
// responded rather than waiting out the deadline.
func TestRunMultiHostCampaign_StopsWhenOnlineHostsRespond(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s")
	srv := campaignTestServer(t, 7, func(conn *websocket.Conn) {
		// 3 targeted, only 2 online; host 30 is offline and silent.
		writeWSFrame(t, conn, "totals", map[string]interface{}{"count": 3, "online": 2})
		writeWSFrame(t, conn, "result", map[string]interface{}{
			"host": map[string]interface{}{"id": 10}, "rows": []map[string]string{{"k": "v"}},
		})
		writeWSFrame(t, conn, "result", map[string]interface{}{
			"host": map[string]interface{}{"id": 20}, "rows": []map[string]string{{"k": "v"}},
		})
		writeWSFrame(t, conn, "status", map[string]interface{}{"expected_results": 2, "actual_results": 2, "status": "finished"})
	})
	defer srv.Close()

	fc := newTestClient(srv.URL)
	nameByID := map[uint]Endpoint{10: {ID: 10}, 20: {ID: 20}, 30: {ID: 30}}
	start := time.Now()
	res, err := fc.runMultiHostCampaign(t.Context(), []uint{10, 20, 30}, "SELECT 1;", nameByID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Errorf("expected prompt return once online hosts responded, took %s", elapsed)
	}
	if res.TargetedHostCount != 3 {
		t.Errorf("TargetedHostCount = %d, want 3", res.TargetedHostCount)
	}
	if res.RespondedHostCount != 2 {
		t.Errorf("RespondedHostCount = %d, want 2", res.RespondedHostCount)
	}
	if len(res.Results) != 2 {
		t.Errorf("len(Results) = %d, want 2 (offline host produced no row)", len(res.Results))
	}
}

// A per-host osquery error in a result frame surfaces as an error on that host's
// row without failing the whole query.
func TestRunMultiHostCampaign_HostErrorRow(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s")
	srv := campaignTestServer(t, 1, func(conn *websocket.Conn) {
		writeWSFrame(t, conn, "totals", map[string]interface{}{"count": 1, "online": 1})
		writeWSFrame(t, conn, "result", map[string]interface{}{
			"host":  map[string]interface{}{"id": 10},
			"rows":  []map[string]string{},
			"error": "no such table: bogus",
		})
		writeWSFrame(t, conn, "status", map[string]interface{}{"expected_results": 1, "actual_results": 1, "status": "finished"})
	})
	defer srv.Close()

	fc := newTestClient(srv.URL)
	res, err := fc.runMultiHostCampaign(t.Context(), []uint{10}, "SELECT * FROM bogus;", map[uint]Endpoint{10: {ID: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(res.Results))
	}
	if res.Results[0]["error"] != "no such table: bogus" {
		t.Errorf("expected host error row, got %+v", res.Results[0])
	}
}

// A failed campaign creation surfaces as an error (no websocket is opened).
func TestRunMultiHostCampaign_CreateFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/fleet/results/websocket" {
			t.Errorf("websocket must not be dialed when campaign creation fails")
		}
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	_, err := fc.runMultiHostCampaign(t.Context(), []uint{10, 20}, "SELECT 1;", map[uint]Endpoint{})
	if err == nil {
		t.Fatal("expected error on failed campaign creation, got nil")
	}
}

// An "error" frame from the server (campaign not found, unauthorized, pubsub
// failure) must surface as an error, not a silent empty result.
func TestRunMultiHostCampaign_ServerErrorFrame(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s")
	srv := campaignTestServer(t, 99, func(conn *websocket.Conn) {
		writeWSFrame(t, conn, "error", "cannot find campaign for ID 99")
	})
	defer srv.Close()

	fc := newTestClient(srv.URL)
	_, err := fc.runMultiHostCampaign(t.Context(), []uint{10, 20}, "SELECT 1;", map[uint]Endpoint{})
	if err == nil {
		t.Fatal("expected error from server error frame, got nil")
	}
	if !strings.Contains(err.Error(), "cannot find campaign") {
		t.Errorf("error %q should include the server's message", err)
	}
}

// A mid-stream read failure (connection dropped before a terminal status) must
// surface as an error rather than masquerading as an empty successful result.
func TestRunMultiHostCampaign_StreamReadError(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s")
	srv := campaignTestServer(t, 5, func(conn *websocket.Conn) {
		// Two hosts online but only one responds, then the connection drops
		// abruptly (no "finished" status, no clean close handshake).
		writeWSFrame(t, conn, "totals", map[string]interface{}{"count": 2, "online": 2})
		writeWSFrame(t, conn, "result", map[string]interface{}{
			"host": map[string]interface{}{"id": 10}, "rows": []map[string]string{{"k": "v"}},
		})
		conn.Close()
	})
	defer srv.Close()

	fc := newTestClient(srv.URL)
	_, err := fc.runMultiHostCampaign(t.Context(), []uint{10, 20}, "SELECT 1;", map[uint]Endpoint{10: {ID: 10}, 20: {ID: 20}})
	if err == nil {
		t.Fatal("expected error on abrupt stream drop, got nil")
	}
}

func TestCampaignWebsocketURL(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{"http://localhost:8080", "ws://localhost:8080/api/v1/fleet/results/websocket"},
		{"https://fleet.example.com", "wss://fleet.example.com/api/v1/fleet/results/websocket"},
		{"https://fleet.example.com/", "wss://fleet.example.com/api/v1/fleet/results/websocket"},
	}
	for _, tc := range cases {
		fc := newTestClient(tc.base)
		got, err := fc.campaignWebsocketURL()
		if err != nil {
			t.Errorf("campaignWebsocketURL(%q) error: %v", tc.base, err)
			continue
		}
		if got != tc.want {
			t.Errorf("campaignWebsocketURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}
