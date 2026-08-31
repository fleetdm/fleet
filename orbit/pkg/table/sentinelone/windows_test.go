package sentinelone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyWindowsCanonicalPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    map[string]string
		wants map[string]string
	}{
		{
			name: "healthy host",
			in: map[string]string{
				"monitor_build_id":                 "25.1.4.434+8d4abf01154f6752-Release.x64",
				"disable_state":                    "Not disabled by the user",
				"self_protection_status":           "On",
				"sentinelnetworkmonitor_is_loaded": "loaded",
				"sentinelmonitor_is_loaded":        "loaded",
				"sentinelagent_is_running":         "running as PPL",
			},
			wants: map[string]string{
				"agent_version":                  "25.1.4.434",
				"agent_agent_operational_state":  "enabled",
				"tamper_protection":              "enabled",
				"agent_agent_network_monitoring": "started",
				"agent_es_framework":             "started",
				"agent_ready":                    "yes",
				// Windows sentinelctl reports no threat protection state.
				"agent_protection": "",
			},
		},
		{
			name: "disabled and unprotected host",
			in: map[string]string{
				"disable_state":                    "Disabled by the user",
				"self_protection_status":           "Off",
				"sentinelnetworkmonitor_is_loaded": "not loaded",
				"sentinelmonitor_is_loaded":        "not loaded",
				"sentinelagent_is_running":         "not running",
			},
			wants: map[string]string{
				"agent_agent_operational_state":  "disabled",
				"tamper_protection":              "disabled",
				"agent_agent_network_monitoring": "stopped",
				"agent_es_framework":             "stopped",
				"agent_ready":                    "no",
			},
		},
		{
			name: "unrecognized states",
			in: map[string]string{
				"disable_state":             "Something new",
				"self_protection_status":    "Partial",
				"sentinelmonitor_is_loaded": "confused",
			},
			wants: map[string]string{
				"agent_agent_operational_state": "unknown",
				"tamper_protection":             "unknown",
				"agent_es_framework":            "unknown",
			},
		},
		{
			name: "build id without a version",
			in: map[string]string{
				"monitor_build_id": "Release.x64",
			},
			wants: map[string]string{
				"agent_version": "",
			},
		},
		{
			// macOS output has none of these keys, so nothing is projected.
			name:  "no windows fields",
			in:    map[string]string{"agent_version": "25.3.4.8365"},
			wants: map[string]string{"agent_version": "25.3.4.8365", "agent_ready": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			applyWindowsCanonicalPaths(tt.in)
			for path, want := range tt.wants {
				assert.Equal(t, want, tt.in[path], "path %q", path)
			}
		})
	}
}

func TestNormalizeLoadedState(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"loaded":      "started",
		"Loaded":      "started",
		"running":     "started",
		"started":     "started",
		"not loaded":  "stopped",
		"not running": "stopped",
		"stopped":     "stopped",
		"":            "unknown",
		"whatever":    "unknown",
	}
	for in, want := range tests {
		assert.Equal(t, want, normalizeLoadedState(in), "normalizeLoadedState(%q)", in)
	}
}

func TestExtractVersionPrefix(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"25.1.4.434+8d4abf01154f6752-Release.x64": "25.1.4.434",
		"25.1.4.434": "25.1.4.434",
		"25.1":       "25.1",
		"x25.1.4":    "",
		"25":         "",
		"":           "",
	}
	for in, want := range tests {
		assert.Equal(t, want, extractVersionPrefix(in), "extractVersionPrefix(%q)", in)
	}
}
