package connectivity

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCatalogue(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ListCatalogue(&buf, Catalogue()))
	out := buf.String()

	// Representative endpoints from multiple features must appear.
	for _, want := range []string{
		"/api/osquery/enroll",
		"/api/fleet/device/ping",
		"/api/fleet/orbit/config",
		"/mdm/apple/scep",
		"/api/mdm/microsoft/discovery",
		"/enroll",
		"/mdm/scep/proxy/probe",
	} {
		assert.Contains(t, out, want, "ListCatalogue missing %s", want)
	}

	// Features must be grouped in AllFeatures() order (osquery before
	// fleet-desktop, etc.).
	osqIdx := strings.Index(out, "osquery")
	dtIdx := strings.Index(out, "fleet-desktop")
	require.NotEqual(t, -1, osqIdx)
	require.NotEqual(t, -1, dtIdx)
	assert.Less(t, osqIdx, dtIdx)
}

func TestTruncateLatency(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"sub-second rounds to ms", 12345 * time.Microsecond, "12ms"},
		{"exactly one second", time.Second, "1s"},
		{"above one second rounds to 10ms", 1234 * time.Millisecond, "1.23s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, truncateLatency(tc.in))
		})
	}
}

func TestDetailCoversEveryStatus(t *testing.T) {
	cases := []struct {
		name   string
		result Result
		want   string
	}{
		{"reachable", Result{Status: StatusReachable, HTTPStatus: 200, Latency: 10 * time.Millisecond}, "HTTP 200"},
		{"blocked with error", Result{Status: StatusBlocked, Error: "timeout"}, "blocked: timeout"},
		{"blocked without error", Result{Status: StatusBlocked}, "blocked"},
		{"forbidden", Result{Status: StatusForbidden, HTTPStatus: 403}, "reverse proxy or WAF"},
		{"not-fleet with error", Result{Status: StatusNotFleet, HTTPStatus: 401, Error: "authenticated probe rejected with HTTP 401"}, "authenticated probe rejected"},
		{"not-fleet without error", Result{Status: StatusNotFleet, HTTPStatus: 200}, "does not look like Fleet"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, detail(tc.result), tc.want)
		})
	}
}

func TestStatusMarker(t *testing.T) {
	cases := []struct {
		name string
		in   Result
		want string
	}{
		{"reachable", Result{Status: StatusReachable}, "✅"},
		{"not-fleet", Result{Status: StatusNotFleet}, "⚠️"},
		{"blocked", Result{Status: StatusBlocked}, "❌"},
		{"forbidden", Result{Status: StatusForbidden}, "❌"},
		{"not-found", Result{Status: StatusNotFound}, "❌"},
		{"unknown falls back", Result{Status: "something-else"}, "?"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, statusMarker(tc.in), tc.want)
		})
	}
}

func TestFeatureNamesListIncludesAllFeatures(t *testing.T) {
	got := FeatureNamesList()
	for _, f := range AllFeatures() {
		assert.Contains(t, got, string(f))
	}
}

func TestRenderHumanIncludesAuthenticatedProbeDetail(t *testing.T) {
	results := []Result{
		{
			Check:      Check{Feature: FeatureDesktop, Method: "POST", Path: "/api/fleet/orbit/config", Fingerprint: FingerprintCapabilitiesHeader, Auth: AuthOrbitNodeKey},
			Status:     StatusNotFleet,
			HTTPStatus: 401,
			Error:      "authenticated probe rejected with HTTP 401",
		},
	}
	var buf bytes.Buffer
	require.NoError(t, RenderHuman(&buf, "https://fleet.example", results))
	out := buf.String()
	assert.Contains(t, out, "authenticated probe rejected")
	assert.Contains(t, out, "Legend:")
}
