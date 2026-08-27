package evidence

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

func TestGrokToolHomeCandidate(t *testing.T) {
	home := t.TempDir()
	// ~/.grok layout + binary
	if err := os.MkdirAll(filepath.Join(home, ".grok", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".grok", "config.json"), `{"model":"grok"}`)
	writeExec(t, filepath.Join(home, ".grok", "bin", "grok"))
	// also ~/.local/bin/grok
	writeExec(t, filepath.Join(home, ".local", "bin", "grok"))

	h := homes.Home{Dir: home, Username: "u", UID: "501"}
	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		99: {PID: 99, Name: "grok", Cmdline: filepath.Join(home, ".grok", "bin", "grok") + " chat"},
	}}
	b := Gather(context.Background(), []homes.Home{h}, snap, map[string]struct{}{"agents": {}})
	cands := AgentCandidates(h, snap, b)
	var found *AgentCandidate
	for i := range cands {
		if cands[i].Name == "grok" {
			found = &cands[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("grok candidate missing; got %+v homes=%+v", cands, b.ToolHomes)
	}
	if found.Confidence < 40 {
		t.Errorf("confidence=%d want >=40 evidence=%s", found.Confidence, found.Signals.CSV())
	}
	if found.Running != 1 {
		t.Errorf("running not set: %+v", found)
	}
	if !found.Signals["tool_home"] || !found.Signals["binary"] {
		t.Errorf("signals=%v", found.Signals.List())
	}
}

func TestJarvisStrongShapeOffline(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "projects", "jarvis")
	if err := os.MkdirAll(filepath.Join(proj, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(proj, "AGENTS.md"), "you are jarvis")
	write(t, filepath.Join(proj, "mcp.json"), `{"mcpServers":{"fs":{"command":"npx"}}}`)
	write(t, filepath.Join(proj, "skills", "weather.md"), "tool")

	h := homes.Home{Dir: home, Username: "u"}
	b := Gather(context.Background(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}})
	cands := AgentCandidates(h, nil, b)
	var found *AgentCandidate
	for i := range cands {
		if cands[i].Name == "jarvis" || filepath.Base(cands[i].Path) == "jarvis" {
			found = &cands[i]
			break
		}
	}
	if found == nil {
		// dump workspaces for debug
		t.Fatalf("jarvis workspace candidate missing; workspaces=%+v cands=%+v", b.Workspaces, cands)
	}
	if !found.Signals["workspace_shape"] {
		t.Errorf("expected strong workspace_shape, got %v", found.Signals.List())
	}
	if found.Confidence < 40 {
		t.Errorf("confidence=%d", found.Confidence)
	}
}

func TestAGENTSOnlyNoCandidate(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "projects", "notes")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(proj, "AGENTS.md"), "just docs")

	h := homes.Home{Dir: home, Username: "u"}
	b := Gather(context.Background(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}})
	cands := AgentCandidates(h, nil, b)
	for _, c := range cands {
		if filepath.Base(c.Path) == "notes" {
			t.Fatalf("weak AGENTS-only project must not emit agents candidate: %+v", c)
		}
	}
}

func TestFrameworkCrewAI(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "src", "bot")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(proj, "package.json"), `{"name":"bot","dependencies":{"crewai":"^1.0.0"}}`)

	h := homes.Home{Dir: home, Username: "u"}
	b := Gather(context.Background(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}})
	cands := AgentCandidates(h, nil, b)
	found := false
	for _, c := range cands {
		for _, tok := range c.Signals.List() {
			if tok == "framework:crewai" {
				found = true
				if c.Category != "agent-harness" {
					t.Errorf("category=%s want agent-harness", c.Category)
				}
			}
		}
	}
	if !found {
		t.Fatalf("framework:crewai not found; frameworks=%+v cands=%+v", b.Frameworks, cands)
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
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
