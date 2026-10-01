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

	got := Scan(homes.Home{Dir: home, Username: "tester"}, &proc.Snapshot{Procs: map[int]proc.Process{}}, nil)

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
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap, nil)
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

func TestShortBinaryNoFalseRunning(t *testing.T) {
	// Catalog agent amazon-q uses binary "q". A process named "icq" must not
	// mark amazon-q as running.
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "q"))

	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		7: {PID: 7, Name: "icq", Exe: "/usr/bin/icq", Cmdline: "/usr/bin/icq"},
	}}
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap, nil)
	for _, a := range got {
		if a.Name == "amazon-q" && a.Running == 1 {
			t.Fatalf("amazon-q falsely marked running from process %q pid=%d", "icq", a.PID)
		}
	}
}

func TestShortBinaryExactRunning(t *testing.T) {
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "q"))

	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		9: {PID: 9, Name: "q", Exe: filepath.Join(home, ".local", "bin", "q"), Cmdline: filepath.Join(home, ".local", "bin", "q") + " chat"},
	}}
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap, nil)
	for _, a := range got {
		if a.Name == "amazon-q" {
			if a.Running != 1 || a.PID != 9 {
				t.Fatalf("amazon-q should be running: %+v", a)
			}
			return
		}
	}
	t.Fatal("amazon-q not detected")
}

func TestProcMatchesBinRejectsSuffix(t *testing.T) {
	if procMatchesBin("myclaude", "/usr/bin/myclaude", "myclaude", "claude") {
		t.Fatal("name suffix must not match")
	}
	if !procMatchesBin("claude", "/usr/local/bin/claude", "/usr/local/bin/claude --help", "claude") {
		t.Fatal("exact match should work")
	}
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
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // test fixture: simulates an executable agent binary
		t.Fatal(err)
	}
}
