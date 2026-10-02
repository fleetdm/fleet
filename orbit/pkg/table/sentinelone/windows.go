package sentinelone

import "strings"

// applyWindowsCanonicalPaths projects the flatter fields that Windows
// sentinelctl reports onto the section paths macOS and Linux produce, so one
// query works across platforms. Windows sentinelctl reports a subset of the
// macOS fields, so columns with no Windows source stay empty.
//
// Sample Windows output:
//
//	Disable State: Not disabled by the user
//	SentinelMonitor is loaded
//	Self-Protection status: On
//	Monitor Build id: 25.1.4.434+8d4abf01154f6752-Release.x64
//	SentinelNetworkMonitor is loaded
//	SentinelAgent is loaded
//	SentinelAgent is running as PPL
//	Mitigation policy: none
func applyWindowsCanonicalPaths(parsed map[string]string) {
	if v := extractVersionPrefix(parsed["monitor_build_id"]); v != "" {
		parsed["agent_version"] = v
	}

	if v := strings.ToLower(strings.TrimSpace(parsed["disable_state"])); v != "" {
		switch {
		case strings.Contains(v, "not disabled"):
			parsed["agent_agent_operational_state"] = "enabled"
		case strings.Contains(v, "disabled"):
			parsed["agent_agent_operational_state"] = "disabled"
		default:
			parsed["agent_agent_operational_state"] = "unknown"
		}
	}

	// Self-protection is anti-tamper, not the threat protection `protection`
	// reports on macOS, so it gets its own column rather than filling that one.
	if v := strings.ToLower(strings.TrimSpace(parsed["self_protection_status"])); v != "" {
		switch v {
		case "on", "enabled":
			parsed["tamper_protection"] = "enabled"
		case "off", "disabled":
			parsed["tamper_protection"] = "disabled"
		default:
			parsed["tamper_protection"] = "unknown"
		}
	}

	if v := parsed["sentinelnetworkmonitor_is_loaded"]; v != "" {
		parsed["agent_agent_network_monitoring"] = normalizeLoadedState(v)
	}

	// SentinelMonitor is the Windows counterpart of the macOS EndpointSecurity
	// framework: the kernel-level component that feeds the agent.
	if v := parsed["sentinelmonitor_is_loaded"]; v != "" {
		parsed["agent_es_framework"] = normalizeLoadedState(v)
	}

	if v := parsed["sentinelagent_is_running"]; v != "" {
		if strings.HasPrefix(strings.ToLower(v), "running") {
			parsed["agent_ready"] = "yes"
		} else {
			parsed["agent_ready"] = "no"
		}
	}
}

// normalizeLoadedState maps a Windows load/run state onto the vocabulary macOS
// sentinelctl uses for the same columns.
func normalizeLoadedState(v string) string {
	switch state := strings.ToLower(strings.TrimSpace(v)); {
	case strings.HasPrefix(state, "not "), strings.HasPrefix(state, "stopped"):
		return "stopped"
	case strings.HasPrefix(state, "loaded"), strings.HasPrefix(state, "running"), strings.HasPrefix(state, "started"):
		return "started"
	default:
		return "unknown"
	}
}

// extractVersionPrefix returns the leading dotted-numeric version in s, e.g.
// "25.1.4.434+8d4abf01154f6752-Release.x64" yields "25.1.4.434".
func extractVersionPrefix(s string) string {
	s = strings.TrimSpace(s)
	end := 0
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			end++
			continue
		}
		break
	}
	v := strings.Trim(s[:end], ".")
	if !strings.Contains(v, ".") {
		return ""
	}
	return v
}
