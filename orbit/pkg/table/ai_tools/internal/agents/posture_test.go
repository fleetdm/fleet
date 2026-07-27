package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

func TestClaudePermissionMode(t *testing.T) {
	home := t.TempDir()
	if got := claudePermissionMode(home); got != "" {
		t.Errorf("no settings: got %q want empty", got)
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"),
		[]byte(`{"permissions":{"defaultMode":"bypassPermissions"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := claudePermissionMode(home); got != "bypassPermissions" {
		t.Errorf("got %q want bypassPermissions", got)
	}
}

func TestEnrichPostureRuntimeFlag(t *testing.T) {
	cc := known{name: "claude-code", autoFlags: []string{"--dangerously-skip-permissions", "skip-permissions"}}
	a := &Agent{Name: "claude-code"}
	enrichPosture(a, cc, homes.Home{Dir: t.TempDir()}, "node /x/claude --dangerously-skip-permissions")
	if !strings.Contains(a.RiskFlags, "skip_permissions_runtime") {
		t.Errorf("RiskFlags=%q missing skip_permissions_runtime", a.RiskFlags)
	}

	b := &Agent{Name: "claude-code"}
	enrichPosture(b, cc, homes.Home{Dir: t.TempDir()}, "node /x/claude")
	if b.RiskFlags != "" {
		t.Errorf("RiskFlags=%q want empty for normal launch", b.RiskFlags)
	}
}

func TestEnrichPostureSettingsMode(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"),
		[]byte(`{"permissions":{"defaultMode":"acceptEdits"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Name: "claude-code"}
	enrichPosture(a, known{name: "claude-code"}, homes.Home{Dir: home}, "")
	if a.PermissionMode != "acceptEdits" {
		t.Errorf("PermissionMode=%q want acceptEdits", a.PermissionMode)
	}
	if !strings.Contains(a.RiskFlags, "auto_accept_edits") {
		t.Errorf("RiskFlags=%q missing auto_accept_edits", a.RiskFlags)
	}
}
