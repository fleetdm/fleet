package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// enrichPosture fills the agent's autonomy posture: the declared permission
// mode (from on-disk settings) and any runtime auto-approve flags observed on
// the live process command line. Both feed RiskFlags.
func enrichPosture(a *Agent, k known, h homes.Home, cmdline string) {
	var flags []string

	// 1. Runtime: agent process launched in an unattended / sandbox-disabled mode.
	if cmdline != "" && len(k.autoFlags) > 0 {
		low := strings.ToLower(cmdline)
		for _, f := range k.autoFlags {
			if strings.Contains(low, f) {
				flags = append(flags, "skip_permissions_runtime")
				break
			}
		}
	}

	// 2. Declared: Claude Code persists its autonomy posture in settings.json.
	if k.name == "claude-code" {
		if mode := claudePermissionMode(h.Dir); mode != "" {
			a.PermissionMode = mode
			switch mode {
			case "bypassPermissions":
				flags = append(flags, "bypass_permissions")
			case "acceptEdits":
				flags = append(flags, "auto_accept_edits")
			}
		}
	}

	a.RiskFlags = strings.Join(dedupe(flags), ",")
}

// claudePermissionMode reads permissions.defaultMode from the user-level Claude
// Code settings (settings.json wins over settings.local.json). Returns "" when
// unset or unreadable.
func claudePermissionMode(home string) string {
	for _, p := range []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.local.json"),
	} {
		b, err := os.ReadFile(p) // #nosec G304 -- fixed path under the user's ~/.claude
		if err != nil {
			continue
		}
		var s struct {
			Permissions struct {
				DefaultMode string `json:"defaultMode"`
			} `json:"permissions"`
		}
		if err := json.Unmarshal(b, &s); err != nil {
			continue
		}
		if s.Permissions.DefaultMode != "" {
			return s.Permissions.DefaultMode
		}
	}
	return ""
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := in[:0]
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
