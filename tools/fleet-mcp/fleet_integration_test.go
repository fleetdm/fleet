package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestListSoftwareTitles_PaginatesUntilShortPage(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fleet/software/titles" {
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		var titles []SoftwareTitle
		switch page {
		case 0:
			titles = make([]SoftwareTitle, 100)
			for i := range titles {
				titles[i] = SoftwareTitle{ID: uint(i + 1), Name: fmt.Sprintf("pkg%d", i), Source: "apps"}
			}
		case 1:
			titles = make([]SoftwareTitle, 25)
			for i := range titles {
				titles[i] = SoftwareTitle{ID: uint(100 + i + 1), Name: fmt.Sprintf("pkg%d", 100+i), Source: "apps"}
			}
		default:
			t.Errorf("unexpected page %d", page)
			http.Error(w, "unexpected page", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(struct {
			SoftwareTitles []SoftwareTitle `json:"software_titles"`
		}{SoftwareTitles: titles})
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	// perPage 0 means "no client-side cap" — paginate until the short page.
	out, truncated, err := fc.ListSoftwareTitles(context.Background(), "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Errorf("expected truncated=false")
	}
	if got, want := len(out), 125; got != want {
		t.Errorf("len(out) = %d, want %d", got, want)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 page calls, got %d", got)
	}
}

func TestListSoftwareTitles_AppliesSourceFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fleet/software/titles" {
			http.NotFound(w, r)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page > 0 {
			// Short page on page 1 to end pagination.
			_ = json.NewEncoder(w).Encode(struct {
				SoftwareTitles []SoftwareTitle `json:"software_titles"`
			}{})
			return
		}
		// Mixed-source payload: 3 npm, 2 python, 5 apps. Short page (8 < 100)
		// so pagination ends after this response.
		titles := []SoftwareTitle{
			{ID: 1, Name: "left-pad", Source: "npm_packages"},
			{ID: 2, Name: "lodash", Source: "npm_packages"},
			{ID: 3, Name: "axios", Source: "npm_packages"},
			{ID: 4, Name: "requests", Source: "python_packages"},
			{ID: 5, Name: "numpy", Source: "python_packages"},
			{ID: 6, Name: "Slack.app", Source: "apps"},
			{ID: 7, Name: "Chrome.app", Source: "apps"},
			{ID: 8, Name: "Zoom.app", Source: "apps"},
		}
		_ = json.NewEncoder(w).Encode(struct {
			SoftwareTitles []SoftwareTitle `json:"software_titles"`
		}{SoftwareTitles: titles})
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	out, _, err := fc.ListSoftwareTitles(context.Background(), "", "", "", "", "npm_packages", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := len(out), 3; got != want {
		t.Errorf("len(out) = %d, want %d (3 npm)", got, want)
	}
	for _, row := range out {
		if !strings.EqualFold(row.Source, "npm_packages") {
			t.Errorf("unexpected source %q in filtered result", row.Source)
		}
	}

	// Case-insensitive should also work.
	out2, _, err := fc.ListSoftwareTitles(context.Background(), "", "", "", "", "NPM_PACKAGES", 0)
	if err != nil {
		t.Fatalf("unexpected error (case-insensitive): %v", err)
	}
	if len(out2) != 3 {
		t.Errorf("case-insensitive filter returned %d rows, want 3", len(out2))
	}
}

func TestGetHostSoftware_PropagatesTruncated(t *testing.T) {
	// Lower the cap so a small fixture trips truncation deterministically.
	orig := fetchSoftwareHardCap
	fetchSoftwareHardCap = 4
	t.Cleanup(func() { fetchSoftwareHardCap = orig })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/fleet/hosts/") || !strings.HasSuffix(r.URL.Path, "/software") {
			http.NotFound(w, r)
			return
		}
		// Single page with 10 matching rows — hard cap of 4 should fire
		// before the page is fully consumed.
		rows := make([]HostSoftware, 10)
		for i := range rows {
			rows[i] = HostSoftware{ID: uint(i + 1), Name: fmt.Sprintf("pkg%d", i), Source: "apps"}
		}
		_ = json.NewEncoder(w).Encode(struct {
			Software []HostSoftware `json:"software"`
		}{Software: rows})
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	// perPage 0 — don't short-circuit on client-side cap. Force the hard-cap
	// path to fire instead. source="" matches everything.
	out, truncated, err := fc.GetHostSoftware(context.Background(), 42, "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated {
		t.Errorf("expected truncated=true when hard cap fires")
	}
	if got, want := len(out), 4; got != want {
		t.Errorf("len(out) = %d, want %d (hard cap)", got, want)
	}
}

func TestResolveHostWithUsers_AmbiguousCandidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/fleet/hosts":
			// Substring search returns multiple collisions.
			hosts := []Endpoint{
				{ID: 1, Name: "mac-1.local"},
				{ID: 2, Name: "mac-2.local"},
				{ID: 3, Name: "mac-3.local"},
			}
			_ = json.NewEncoder(w).Encode(struct {
				Hosts []Endpoint `json:"hosts"`
			}{Hosts: hosts})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	host, ambiguous, candidates, err := resolveHostWithUsers(context.Background(), fc, 0, "mac")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ambiguous {
		t.Errorf("expected ambiguous=true for multi-match identifier")
	}
	if host != nil {
		t.Errorf("expected host=nil when ambiguous, got %+v", host)
	}
	if got, want := len(candidates), 3; got != want {
		t.Errorf("len(candidates) = %d, want %d", got, want)
	}
}

func TestGetHostByIDWithUsers_DecodesUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fleet/hosts/42" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"host": map[string]any{
				"id":       42,
				"hostname": "test.local",
				"users": []map[string]any{
					{"uid": 501, "username": "alice", "type": "regular", "groupname": "staff", "shell": "/bin/zsh"},
					{"uid": 502, "username": "bob", "type": "regular", "groupname": "staff", "shell": "/bin/bash"},
				},
			},
		})
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	host, err := fc.GetHostByIDWithUsers(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host == nil {
		t.Fatalf("nil host")
	}
	if got, want := host.ID, uint(42); got != want {
		t.Errorf("host.ID = %d, want %d", got, want)
	}
	if got, want := len(host.Users), 2; got != want {
		t.Errorf("len(users) = %d, want %d", got, want)
	}
	if host.Users[0].Username != "alice" || host.Users[1].Shell != "/bin/bash" {
		t.Errorf("user decode mismatch: %+v", host.Users)
	}
}

func TestFilterHostUsers_CaseInsensitiveAcrossFields(t *testing.T) {
	users := []HostUser{
		{UID: 501, Username: "alice", GroupName: "staff", Shell: "/bin/zsh"},
		{UID: 502, Username: "bob", GroupName: "wheel", Shell: "/bin/bash"},
		{UID: 0, Username: "root", GroupName: "wheel", Shell: "/bin/sh"},
	}
	cases := []struct {
		query string
		want  int
	}{
		{"alice", 1}, // username exact
		{"ALICE", 1}, // case-insensitive
		{"wheel", 2}, // groupname
		{"bash", 1},  // shell
		{"50", 2},    // uid prefix (matches 501, 502)
		{"nomatch", 0},
	}
	for _, tc := range cases {
		got := filterHostUsers(users, tc.query)
		if len(got) != tc.want {
			t.Errorf("filterHostUsers(%q) returned %d, want %d", tc.query, len(got), tc.want)
		}
	}
}

func TestValidateGetSoftwareArgs(t *testing.T) {
	cases := []struct {
		name                        string
		perHost                     bool
		fleet, platform, vulnerable string
		wantErr                     bool
	}{
		{"per-host alone ok", true, "", "", "", false},
		{"per-host + fleet rejected", true, "Workstations", "", "", true},
		{"per-host + platform rejected", true, "", "macos", "", true},
		{"cross-host none ok (full inventory)", false, "", "", "", false},
		{"cross-host fleet alone ok", false, "Workstations", "", "", false},
		{"cross-host platform alone rejected", false, "", "macos", "", true},
		{"cross-host platform + fleet ok", false, "Workstations", "macos", "", false},
		{"vulnerable=true ok", false, "", "", "true", false},
		{"vulnerable=false ok", false, "", "", "false", false},
		{"vulnerable bad value rejected", false, "", "", "maybe", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGetSoftwareArgs(tc.perHost, tc.fleet, tc.platform, tc.vulnerable)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMatchesSoftwareSource(t *testing.T) {
	cases := []struct {
		row, want string
		expect    bool
	}{
		{"apps", "", true},                     // empty want matches anything
		{"apps", "apps", true},                 // exact
		{"NPM_Packages", "npm_packages", true}, // case-insensitive
		{"deb_packages", "apps", false},        // mismatch
	}
	for _, tc := range cases {
		if got := matchesSoftwareSource(tc.row, tc.want); got != tc.expect {
			t.Errorf("matchesSoftwareSource(%q,%q) = %v, want %v", tc.row, tc.want, got, tc.expect)
		}
	}
}

func TestResolveHost_NumericFetchesByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/fleet/hosts/42" {
			_ = json.NewEncoder(w).Encode(struct {
				Host Endpoint `json:"host"`
			}{Host: Endpoint{ID: 42, Name: "h42.local"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	fc := newTestClient(srv.URL)

	// numeric host_id is verified via GetHostByID (confirms it exists, gets the name)
	host, _, ambiguous, err := resolveHost(context.Background(), fc, 42, "")
	if err != nil || ambiguous || host == nil || host.ID != 42 || host.Name != "h42.local" {
		t.Fatalf("numeric: host=%+v ambiguous=%v err=%v, want id=42 with name", host, ambiguous, err)
	}
}

func TestParseHostIDArg(t *testing.T) {
	cases := []struct {
		in      string
		want    uint
		wantErr bool
	}{
		{"", 0, false},
		{"42", 42, false},
		{"abc", 0, true},
		{"0", 0, true},
		{"-1", 0, true},
	}
	for _, tc := range cases {
		got, err := parseHostIDArg(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseHostIDArg(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("parseHostIDArg(%q) = (%d, %v), want (%d, nil)", tc.in, got, err, tc.want)
		}
	}
}

func TestResolveHost_IdentifierSingleAndFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/fleet/hosts":
			hosts := []Endpoint{}
			if r.URL.Query().Get("query") == "solo" { // single unambiguous match
				hosts = []Endpoint{{ID: 7, Name: "solo.local"}}
			}
			_ = json.NewEncoder(w).Encode(struct {
				Hosts []Endpoint `json:"hosts"`
			}{Hosts: hosts})
		case r.URL.Path == "/api/v1/fleet/hosts/identifier/ghost":
			_ = json.NewEncoder(w).Encode(struct {
				Host Endpoint `json:"host"`
			}{Host: Endpoint{ID: 9, Name: "ghost.local"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	fc := newTestClient(srv.URL)

	// single substring match -> that host, not ambiguous
	host, _, ambiguous, err := resolveHost(context.Background(), fc, 0, "solo")
	if err != nil || ambiguous || host == nil || host.ID != 7 {
		t.Fatalf("single match: host=%+v ambiguous=%v err=%v, want id=7", host, ambiguous, err)
	}
	// zero substring matches -> identifier-endpoint fallback
	host, _, ambiguous, err = resolveHost(context.Background(), fc, 0, "ghost")
	if err != nil || ambiguous || host == nil || host.ID != 9 {
		t.Fatalf("fallback: host=%+v ambiguous=%v err=%v, want id=9", host, ambiguous, err)
	}
}

func TestResolveHostWithUsers_SingleMatchAndFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/fleet/hosts":
			hosts := []Endpoint{}
			if r.URL.Query().Get("query") == "solo" {
				hosts = []Endpoint{{ID: 5, Name: "solo.local"}}
			}
			_ = json.NewEncoder(w).Encode(struct {
				Hosts []Endpoint `json:"hosts"`
			}{Hosts: hosts})
		case "/api/v1/fleet/hosts/5":
			_ = json.NewEncoder(w).Encode(struct {
				Host HostWithUsers `json:"host"`
			}{Host: HostWithUsers{Endpoint: Endpoint{ID: 5, Name: "solo.local"}, Users: []HostUser{{UID: 501, Username: "alice"}}}})
		case "/api/v1/fleet/hosts/identifier/ghost":
			// identifier endpoint resolves the host but carries NO users
			_ = json.NewEncoder(w).Encode(struct {
				Host Endpoint `json:"host"`
			}{Host: Endpoint{ID: 9, Name: "ghost.local"}})
		case "/api/v1/fleet/hosts/9":
			// users come from the by-id refetch
			_ = json.NewEncoder(w).Encode(struct {
				Host HostWithUsers `json:"host"`
			}{Host: HostWithUsers{Endpoint: Endpoint{ID: 9, Name: "ghost.local"}, Users: []HostUser{{UID: 0, Username: "root"}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	fc := newTestClient(srv.URL)

	// single match -> fetched by id, users populated
	host, ambiguous, _, err := resolveHostWithUsers(context.Background(), fc, 0, "solo")
	if err != nil || ambiguous || host == nil || host.ID != 5 || len(host.Users) != 1 {
		t.Fatalf("single match: host=%+v ambiguous=%v err=%v", host, ambiguous, err)
	}
	// zero matches -> identifier endpoint (no users) then by-id refetch (users)
	host, ambiguous, _, err = resolveHostWithUsers(context.Background(), fc, 0, "ghost")
	if err != nil || ambiguous || host == nil || host.ID != 9 || len(host.Users) != 1 || host.Users[0].Username != "root" {
		t.Fatalf("fallback: host=%+v ambiguous=%v err=%v", host, ambiguous, err)
	}
}

func TestGetHostSoftware_DecodesNestedInstalledVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/software") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"software":[{"id":1,"name":"curl","source":"deb_packages","installed_versions":[{"version":"7.88.1","vulnerabilities":["CVE-2026-1111"],"installed_paths":["/usr/bin/curl"]}]}]}`))
	}))
	defer srv.Close()
	fc := newTestClient(srv.URL)

	out, _, err := fc.GetHostSoftware(context.Background(), 1, "", "", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || len(out[0].InstalledVersions) != 1 {
		t.Fatalf("decoded = %+v, want 1 row with 1 installed version", out)
	}
	v := out[0].InstalledVersions[0]
	if v.Version != "7.88.1" || len(v.Vulnerabilities) != 1 || v.Vulnerabilities[0] != "CVE-2026-1111" || len(v.InstalledPaths) != 1 {
		t.Errorf("nested installed_version not decoded: %+v", v)
	}
}

func TestGetHostSoftware_SourceFilterAndPerPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/software") {
			http.NotFound(w, r)
			return
		}
		rows := []HostSoftware{
			{ID: 1, Name: "a", Source: "apps"},
			{ID: 2, Name: "b", Source: "deb_packages"},
			{ID: 3, Name: "c", Source: "apps"},
			{ID: 4, Name: "d", Source: "npm_packages"},
			{ID: 5, Name: "e", Source: "apps"},
		}
		_ = json.NewEncoder(w).Encode(struct {
			Software []HostSoftware `json:"software"`
		}{Software: rows})
	}))
	defer srv.Close()
	fc := newTestClient(srv.URL)

	// source=apps keeps only apps rows; perPage=2 caps the merged result early
	out, truncated, err := fc.GetHostSoftware(context.Background(), 42, "", "", "apps", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Errorf("expected truncated=false when perPage is reached")
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2 (perPage cap on matching rows)", len(out))
	}
	for _, sw := range out {
		if sw.Source != "apps" {
			t.Errorf("source filter leaked non-apps row: %+v", sw)
		}
	}
}
