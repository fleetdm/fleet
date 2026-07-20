package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

func TestDetectClaudeCode(t *testing.T) {
	home := t.TempDir()

	// npm-global install with a manifest (version source) + a binary symlink target.
	write(t, filepath.Join(home, ".npm-global", "lib", "node_modules", "@anthropic-ai", "claude-code", "package.json"),
		`{"name":"@anthropic-ai/claude-code","version":"1.2.3"}`)
	writeExec(t, filepath.Join(home, ".local", "bin", "claude"))

	got := Scan(homes.Home{Dir: home, Username: "tester"}, &proc.Snapshot{Procs: map[int]proc.Process{}})

	var cc *Agent
	for i := range got {
		if got[i].Name == "claude-code" {
			cc = &got[i]
		}
	}
	if cc == nil {
		t.Fatalf("claude-code not detected; got %d agents", len(got))
	}
	if cc.Version != "1.2.3" {
		t.Errorf("version=%q want 1.2.3 (must come from manifest, not exec)", cc.Version)
	}
	if cc.InstallMethod != "npm-global" {
		t.Errorf("install_method=%q want npm-global", cc.InstallMethod)
	}
	if cc.Binary != "claude" {
		t.Errorf("binary=%q want claude", cc.Binary)
	}
}

func TestMarkRunning(t *testing.T) {
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "aider"))
	write(t, filepath.Join(home, ".local", "pipx", "venvs", "aider-chat", "pyvenv.cfg"), "home = /usr\n")

	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		55: {PID: 55, Name: "aider", Cmdline: "/home/u/.local/bin/aider --model gpt-4"},
	}}
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap)
	for _, a := range got {
		if a.Name == "aider" {
			if a.Running != 1 || a.PID != 55 {
				t.Errorf("aider running not detected: %+v", a)
			}
			return
		}
	}
	t.Fatal("aider not detected")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil { // #nosec G306 -- test fixture: simulates an executable agent binary
		t.Fatal(err)
	}
}
