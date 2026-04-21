package connectivity

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
		default:
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	t.Cleanup(srv.Close)

	checks := []Check{
		{Feature: FeatureOsquery, Method: "GET", Path: "/reach"},
		{Feature: FeatureOsquery, Method: "GET", Path: "/bad"},
		{Feature: FeatureDesktop, Method: "GET", Path: "/missing"},
	}

	results, err := Probe(t.Context(), Options{BaseURL: srv.URL}, checks)
	require.NoError(t, err)
	require.Len(t, results, 3)

	assert.Equal(t, StatusReachable, results[0].Status)
	assert.Equal(t, http.StatusUnauthorized, results[0].HTTPStatus)
	assert.Equal(t, StatusReachable, results[1].Status)
	assert.Equal(t, StatusNotFound, results[2].Status)
}

func TestProbeBlockedOnNetworkError(t *testing.T) {
	// Bind to an unused port by starting and immediately stopping a server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()

	results, err := Probe(t.Context(), Options{BaseURL: addr}, []Check{
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
	assert.Contains(t, out, "blocked: timeout")
	assert.Contains(t, out, "1 reachable, 1 route-not-found, 1 blocked")

	// Groups must appear in AllFeatures order.
	osIdx := strings.Index(out, "osquery")
	dtIdx := strings.Index(out, "Fleet Desktop")
	require.Positive(t, osIdx)
	require.Positive(t, dtIdx)
	assert.Less(t, osIdx, dtIdx)
}
