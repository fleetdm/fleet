package connectivity

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeClassifiesStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reach":
			w.WriteHeader(http.StatusUnauthorized)
		case "/bad":
			w.WriteHeader(http.StatusBadRequest)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/waf":
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	t.Cleanup(srv.Close)

	checks := []Check{
		{Feature: FeatureOsquery, Method: "GET", Path: "/reach"},
		{Feature: FeatureOsquery, Method: "GET", Path: "/bad"},
		{Feature: FeatureDesktop, Method: "GET", Path: "/missing"},
		{Feature: FeatureDesktop, Method: "GET", Path: "/waf"},
	}

	results, err := Probe(t.Context(), Options{BaseURL: srv.URL}, checks)
	require.NoError(t, err)
	require.Len(t, results, 4)

	assert.Equal(t, StatusReachable, results[0].Status)
	assert.Equal(t, http.StatusUnauthorized, results[0].HTTPStatus)
	assert.Equal(t, StatusReachable, results[1].Status)
	assert.Equal(t, StatusNotFound, results[2].Status)
	assert.Equal(t, StatusForbidden, results[3].Status)
	assert.Equal(t, http.StatusForbidden, results[3].HTTPStatus)
}

func TestProbeBlockedOnNetworkError(t *testing.T) {
	// Port 1 is privileged; an unprivileged test process can't bind it, so
	// the connect reliably fails with ECONNREFUSED. Avoids the ephemeral-port
	// reuse race of closing an httptest.Server and reusing its URL.
	results, err := Probe(t.Context(), Options{BaseURL: "http://127.0.0.1:1"}, []Check{
		{Feature: FeatureOsquery, Method: "GET", Path: "/anywhere"},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, StatusBlocked, results[0].Status)
	assert.NotEmpty(t, results[0].Error)
	assert.Zero(t, results[0].HTTPStatus)
}

func TestProbeRejectsBadBaseURL(t *testing.T) {
	_, err := Probe(t.Context(), Options{BaseURL: ""}, nil)
	require.Error(t, err)

	_, err = Probe(t.Context(), Options{BaseURL: "not-a-url"}, nil)
	require.Error(t, err)
}

func TestProbeOrdersResultsWithInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	checks := make([]Check, 0, 20)
	for range 20 {
		checks = append(checks, Check{Feature: FeatureOsquery, Method: "GET", Path: "/p"})
	}
	// Replace one path so ordering would be observable if broken.
	checks[7].Path = "/seven"

	results, err := Probe(t.Context(), Options{BaseURL: srv.URL}, checks)
	require.NoError(t, err)
	require.Len(t, results, len(checks))
	assert.Equal(t, "/seven", results[7].Check.Path)
}

func TestCataloguePathDedupe(t *testing.T) {
	// /enroll appears under both iOS and Android in the raw catalogue; the
	// public Catalogue() must emit it exactly once.
	all := Catalogue()
	var enrollCount int
	for _, c := range all {
		if c.Path == "/enroll" {
			enrollCount++
		}
	}
	assert.Equal(t, 1, enrollCount, "Catalogue() must dedupe /enroll across features")
}

func TestCatalogueFilter(t *testing.T) {
	only := Catalogue(FeatureOsquery)
	require.NotEmpty(t, only)
	for _, c := range only {
		assert.Equal(t, FeatureOsquery, c.Feature)
	}
}

func TestParseFeatures(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		got, err := ParseFeatures("")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("valid list", func(t *testing.T) {
		got, err := ParseFeatures("osquery, mdm-macos")
		require.NoError(t, err)
		assert.Equal(t, []Feature{FeatureOsquery, FeatureMDMMacOS}, got)
	})
	t.Run("unknown errors", func(t *testing.T) {
		_, err := ParseFeatures("osquery,bogus")
		require.Error(t, err)
	})
	t.Run("dedupes", func(t *testing.T) {
		got, err := ParseFeatures("osquery,osquery")
		require.NoError(t, err)
		assert.Equal(t, []Feature{FeatureOsquery}, got)
	})
}

func TestProbeFingerprintCapabilitiesHeader(t *testing.T) {
	// Server sets the capability header — counts as Fleet. Second server
	// omits it — the probe should mark StatusNotFleet.
	fleetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(fleet.CapabilitiesHeader, "orbit")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(fleetSrv.Close)
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(proxySrv.Close)

	check := []Check{{Feature: FeatureDesktop, Method: "GET", Path: "/p", Fingerprint: FingerprintCapabilitiesHeader}}

	got, err := Probe(t.Context(), Options{BaseURL: fleetSrv.URL}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusReachable, got[0].Status)

	got, err = Probe(t.Context(), Options{BaseURL: proxySrv.URL}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusNotFleet, got[0].Status)
}

func TestProbeFingerprintJSONError(t *testing.T) {
	fleetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Authentication required","errors":[{"name":"base","reason":"no token"}]}`))
	}))
	t.Cleanup(fleetSrv.Close)
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<html><body>proxy login required</body></html>`))
	}))
	t.Cleanup(proxySrv.Close)

	check := []Check{{Feature: FeatureFleetctl, Method: "GET", Path: "/api/latest/fleet/version", Fingerprint: FingerprintFleetJSONError}}

	got, err := Probe(t.Context(), Options{BaseURL: fleetSrv.URL}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusReachable, got[0].Status)

	got, err = Probe(t.Context(), Options{BaseURL: proxySrv.URL}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusNotFleet, got[0].Status)
}

func TestProbeAuthOrbitNodeKey(t *testing.T) {
	const expectedKey = "test-orbit-node-key"
	var bodyReceived string
	var contentTypeReceived string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodyReceived = string(b)
		contentTypeReceived = r.Header.Get("Content-Type")
		var parsed map[string]string
		if len(b) > 0 {
			_ = json.Unmarshal(b, &parsed)
		}
		if parsed["orbit_node_key"] != expectedKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set(fleet.CapabilitiesHeader, "orbit")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	check := []Check{{
		Feature: FeatureDesktop, Method: "POST", Path: "/api/fleet/orbit/config",
		Fingerprint: FingerprintCapabilitiesHeader, Auth: AuthOrbitNodeKey,
	}}

	// With correct key: reachable.
	got, err := Probe(t.Context(), Options{BaseURL: srv.URL, OrbitNodeKey: expectedKey}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusReachable, got[0].Status)
	assert.Equal(t, "application/json", contentTypeReceived)
	assert.Contains(t, bodyReceived, expectedKey)

	// With wrong key: not-fleet (auth rejected, flagged as suspicious).
	got, err = Probe(t.Context(), Options{BaseURL: srv.URL, OrbitNodeKey: "wrong"}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusNotFleet, got[0].Status)
	assert.Contains(t, got[0].Error, "authenticated probe rejected")

	// With no key: downgrades to unauthenticated. Server returns 401 (no
	// match), but since auth wasn't attempted this is a fingerprint-mismatch
	// path — missing capabilities header means StatusNotFleet.
	got, err = Probe(t.Context(), Options{BaseURL: srv.URL}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusNotFleet, got[0].Status)
	assert.Empty(t, bodyReceived, "probe must not send body when OrbitNodeKey is unset")
}

func TestProbeAuthRejectedButFleetish(t *testing.T) {
	// A real Fleet server rejecting a revoked/rotated orbit node key
	// returns 401 with the standard {message,errors} envelope. The probe
	// should flag this as reachable-with-auth-problem, not not-fleet, so
	// users debug enrollment instead of intermediaries.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Authentication required","errors":[{"name":"base","reason":"invalid orbit node key"}]}`))
	}))
	t.Cleanup(srv.Close)

	check := []Check{{
		Feature: FeatureDesktop, Method: "POST", Path: "/api/fleet/orbit/config",
		Fingerprint: FingerprintCapabilitiesHeader | FingerprintFleetJSONError,
		Auth:        AuthOrbitNodeKey,
	}}
	got, err := Probe(t.Context(), Options{BaseURL: srv.URL, OrbitNodeKey: "stale-key"}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusReachable, got[0].Status)
	assert.Contains(t, got[0].Error, "stale orbit node key")
	assert.Equal(t, http.StatusUnauthorized, got[0].HTTPStatus)
}

func TestLooksLikeFleetHTMLTitle(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"exact title", `<html><title>Fleet</title></html>`, true},
		{"uppercase tag", `<HTML><TITLE>FLEET</TITLE>`, true},
		{"fleet with suffix", `<title>Fleet | device</title>`, true},
		{"other brand", `<title>Okta</title>`, false},
		{"no title", `<html></html>`, false},
		{"empty", ``, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, looksLikeFleetHTMLTitle([]byte(tc.body)))
		})
	}
}

func TestProbePrepareRequestFallsBackWhenKeyMissing(t *testing.T) {
	// When a check asks for orbit-key auth but no key is available, the
	// probe must drop to unauthenticated: no body, no Content-Type.
	var got struct {
		contentType string
		body        []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.contentType = r.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(r.Body)
		w.Header().Set("X-Fleet-Capabilities", "orbit_endpoints")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	check := []Check{{
		Feature: FeatureDesktop, Method: "POST", Path: "/x",
		Fingerprint: FingerprintCapabilitiesHeader, Auth: AuthOrbitNodeKey,
	}}
	// No OrbitNodeKey in Options.
	_, err := Probe(t.Context(), Options{BaseURL: srv.URL}, check)
	require.NoError(t, err)
	assert.Empty(t, got.contentType)
	assert.Empty(t, got.body)
}

func TestProbeFingerprintHTMLTitleMatches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Fleet</title></head></html>`))
	}))
	t.Cleanup(srv.Close)

	check := []Check{{Feature: FeatureMDMIOS, Method: "GET", Path: "/enroll", Fingerprint: FingerprintFleetHTMLTitle}}
	got, err := Probe(t.Context(), Options{BaseURL: srv.URL}, check)
	require.NoError(t, err)
	assert.Equal(t, StatusReachable, got[0].Status)
}

func TestLooksLikeFleetJSONError(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"fleet error", `{"message":"hi","errors":[]}`, true},
		{"missing errors", `{"message":"hi"}`, false},
		{"missing message", `{"errors":[]}`, false},
		{"html", `<html></html>`, false},
		{"empty", ``, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, looksLikeFleetJSONError([]byte(tc.body)))
		})
	}
}

func TestRenderJSONShape(t *testing.T) {
	results := []Result{
		{Check: Check{Feature: FeatureOsquery, Method: "GET", Path: "/x"}, Status: StatusReachable, HTTPStatus: 200},
		{Check: Check{Feature: FeatureDesktop, Method: "GET", Path: "/y"}, Status: StatusBlocked, Error: "timeout"},
	}
	var buf bytes.Buffer
	require.NoError(t, RenderJSON(&buf, "https://fleet.example", results))

	var parsed struct {
		FleetURL string `json:"fleet_url"`
		Summary  struct {
			Total     int `json:"total"`
			Reachable int `json:"reachable"`
			Blocked   int `json:"blocked"`
		} `json:"summary"`
		Results []json.RawMessage `json:"results"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "https://fleet.example", parsed.FleetURL)
	assert.Equal(t, 2, parsed.Summary.Total)
	assert.Equal(t, 1, parsed.Summary.Reachable)
	assert.Equal(t, 1, parsed.Summary.Blocked)
	assert.Len(t, parsed.Results, 2)
}

func TestRenderHumanIncludesStatusAndSummary(t *testing.T) {
	results := []Result{
		{Check: Check{Feature: FeatureOsquery, Method: "GET", Path: "/a"}, Status: StatusReachable, HTTPStatus: 200},
		{Check: Check{Feature: FeatureOsquery, Method: "GET", Path: "/b"}, Status: StatusNotFound, HTTPStatus: 404},
		{Check: Check{Feature: FeatureOsquery, Method: "GET", Path: "/d"}, Status: StatusForbidden, HTTPStatus: 403},
		{Check: Check{Feature: FeatureDesktop, Method: "GET", Path: "/c"}, Status: StatusBlocked, Error: "timeout"},
	}
	var buf bytes.Buffer
	require.NoError(t, RenderHuman(&buf, "https://fleet.example", results))

	out := buf.String()
	assert.Contains(t, out, "https://fleet.example")
	assert.Contains(t, out, "osquery")
	assert.Contains(t, out, "Fleet Desktop")
	assert.Contains(t, out, "/a")
	assert.Contains(t, out, "/b")
	assert.Contains(t, out, "/c")
	assert.Contains(t, out, "/d")
	assert.Contains(t, out, "blocked: timeout")
	assert.Contains(t, out, "likely blocked by reverse proxy or WAF")
	assert.Contains(t, out, "1 reachable, 0 not-fleet, 1 forbidden, 1 route-not-found, 1 blocked")

	// Groups must appear in AllFeatures order.
	osIdx := strings.Index(out, "osquery")
	dtIdx := strings.Index(out, "Fleet Desktop")
	require.Positive(t, osIdx)
	require.Positive(t, dtIdx)
	assert.Less(t, osIdx, dtIdx)
}
