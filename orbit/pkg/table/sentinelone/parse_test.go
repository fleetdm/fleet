package sentinelone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStatusDarwin(t *testing.T) {
	t.Parallel()

	parsed := parseStatus(darwinStatusFixture)

	wantPaths := map[string]string{
		"agent_version":                          "25.3.4.8365",
		"agent_id":                               "d7f94b33-7f99-4c43-9c6b-1c6e0d01aabb",
		"agent_agent_operational_state":          "enabled",
		"agent_network_extension_content_filter": "active",
		"command_authentication":                 "enabled",
		"daemons_services_agent_helper":          "ready",
		"daemons_services_lib_hooks_service":     "not ready",
		"daemons_integrity_sentineld_shell":      "not running",
		"launchd_sentineld_guard":                "valid",
		"management_server":                      "https://euce1-109.sentinelone.net",
		"management_connected":                   "yes",
	}
	for path, want := range wantPaths {
		assert.Equal(t, want, parsed[path], "path %q", path)
	}

	// "Missing Authorizations" is a valueless header nested under Agent. It
	// must not become a value, and it must not swallow the fields that follow
	// it at the same indentation.
	assert.NotContains(t, parsed, "agent_missing_authorizations")
	assert.Equal(t, "started", parsed["agent_es_framework"])
}

func TestParseStatusWindows(t *testing.T) {
	t.Parallel()

	parsed := parseStatus(windowsStatusFixture)

	wantPaths := map[string]string{
		"disable_state":                    "Not disabled by the user",
		"sentinelmonitor_is_loaded":        "loaded",
		"self_protection_status":           "On",
		"monitor_build_id":                 "25.1.4.434+8d4abf01154f6752-Release.x64",
		"sentinelnetworkmonitor_is_loaded": "loaded",
		"sentinelagent_is_loaded":          "loaded",
		"sentinelagent_is_running":         "running as PPL",
		"mitigation_policy":                "none",
	}
	for path, want := range wantPaths {
		assert.Equal(t, want, parsed[path], "path %q", path)
	}
}

func TestParseStatusValueWithColon(t *testing.T) {
	t.Parallel()

	parsed := parseStatus("Management\n   Server: https://host:8443/path\n   Last Seen: 6/1/26, 1:14:28 PM\n")

	assert.Equal(t, "https://host:8443/path", parsed["management_server"])
	assert.Equal(t, "6/1/26, 1:14:28 PM", parsed["management_last_seen"])
}

func TestParseStatusFirstValueWins(t *testing.T) {
	t.Parallel()

	parsed := parseStatus("Agent\n   Version: 1.0\n   Version: 2.0\n")

	assert.Equal(t, "1.0", parsed["agent_version"])
}

func TestParsePredicateLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line      string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{line: "SentinelAgent is loaded", wantKey: "sentinelagent_is_loaded", wantValue: "loaded", wantOK: true},
		{line: "SentinelAgent is running as PPL", wantKey: "sentinelagent_is_running", wantValue: "running as PPL", wantOK: true},
		// A negated state keys off the same suffix, so an unhealthy host
		// populates the same column as a healthy one.
		{line: "SentinelAgent is not loaded", wantKey: "sentinelagent_is_loaded", wantValue: "not loaded", wantOK: true},
		{line: "SentinelMonitor is not running", wantKey: "sentinelmonitor_is_running", wantValue: "not running", wantOK: true},
		// Anything else keeps the bare subject as its key.
		{line: "Mitigation policy is unavailable", wantKey: "mitigation_policy", wantValue: "unavailable", wantOK: true},
		{line: "Daemons", wantOK: false},
		{line: "is loaded", wantOK: false},
		{line: "SentinelAgent is ", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()

			key, value, ok := parsePredicateLine(tt.line)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestNormalizeKey(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"Console URL":              "console_url",
		"  Agent Version  ":        "agent_version",
		"Last Successful Connect.": "last_successful_connect",
		"DB-Version":               "db_version",
		"agent-helper":             "agent_helper",
		"sentineld_guard":          "sentineld_guard",
		"Self-Protection status":   "self_protection_status",
		"":                         "",
		"---":                      "",
	}
	for in, want := range tests {
		assert.Equal(t, want, normalizeKey(in), "normalizeKey(%q)", in)
	}
}

func TestLeadingIndent(t *testing.T) {
	t.Parallel()

	tests := map[string]int{
		"":        0,
		"x":       0,
		"  x":     2,
		"      x": 6,
		"\tx":     1,
		"   ":     3,
	}
	for in, want := range tests {
		assert.Equal(t, want, leadingIndent(in), "leadingIndent(%q)", in)
	}
}
