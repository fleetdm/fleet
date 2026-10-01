package evidence

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	if !found.Signals.Has("tool_home") || !found.Signals.Has("binary") {
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
	if !found.Signals.Has("workspace_shape") {
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

func TestUnderHomeBobBobby(t *testing.T) {
	// /Users/bob must not claim /Users/bobby artifacts.
	if underHome("/Users/bobby/.grok", "/Users/bob") {
		t.Fatal("bobby path must not be under bob home")
	}
	if underHome("/Users/bobby", "/Users/bob") {
		t.Fatal("bobby home must not be under bob home")
	}
	if !underHome("/Users/bob/.grok", "/Users/bob") {
		t.Fatal("bob/.grok should be under bob")
	}
	if !underHome("/Users/bob", "/Users/bob") {
		t.Fatal("home itself should match")
	}
}

func TestHomeIsolationBobBobbyCandidates(t *testing.T) {
	root := t.TempDir()
	bob := filepath.Join(root, "bob")
	bobby := filepath.Join(root, "bobby")
	for _, h := range []string{bob, bobby} {
		if err := os.MkdirAll(filepath.Join(h, ".grok", "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(h, ".grok", "config.json"), `{"model":"grok"}`)
		writeExec(t, filepath.Join(h, ".grok", "bin", "grok"))
	}

	// Bundle gathered across both homes (as tables.generate does).
	homesList := []homes.Home{
		{Dir: bob, Username: "bob", UID: "501"},
		{Dir: bobby, Username: "bobby", UID: "502"},
	}
	b := Gather(context.Background(), homesList, nil, map[string]struct{}{"agents": {}})

	bobCands := AgentCandidates(homes.Home{Dir: bob, Username: "bob", UID: "501"}, nil, b)
	for _, c := range bobCands {
		if strings.Contains(c.Path, "bobby") || c.Username == "bobby" {
			t.Fatalf("bob candidates leaked bobby artifact: %+v", c)
		}
		if c.Name == "grok" && !strings.HasPrefix(c.Path, bob) {
			t.Fatalf("bob grok path outside bob home: %s", c.Path)
		}
	}
	// bob should still see his own grok
	found := false
	for _, c := range bobCands {
		if c.Name == "grok" {
			found = true
		}
	}
	if !found {
		t.Fatalf("bob should detect own grok; cands=%+v homes=%+v", bobCands, b.ToolHomes)
	}

	bobbyCands := AgentCandidates(homes.Home{Dir: bobby, Username: "bobby", UID: "502"}, nil, b)
	for _, c := range bobbyCands {
		if strings.Contains(c.Path, string(filepath.Separator)+"bob"+string(filepath.Separator)) ||
			(c.Username == "bob" && c.UID == "501") {
			t.Fatalf("bobby candidates leaked bob artifact: %+v", c)
		}
	}
}

func TestProcMatchesBinShortName(t *testing.T) {
	// amazon-q binary "q" must not match processes whose name merely ends in q.
	if procMatchesBin("icq", "/usr/bin/icq", "/usr/bin/icq", "q") {
		t.Fatal("suffix process name must not match short bin q")
	}
	if procMatchesBin("sq", "/bin/sq", "sq", "q") {
		t.Fatal("sq must not match q")
	}
	if !procMatchesBin("q", "/usr/local/bin/q", "/usr/local/bin/q chat", "q") {
		t.Fatal("exact q process should match")
	}
	if !procMatchesBin("Amazon Q", "/opt/homebrew/bin/q", "/opt/homebrew/bin/q", "q") {
		// name is not exact, but exe base is q
		t.Fatal("exe base q should match")
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
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A symlinked tool home whose target sits outside the home directory must not
// pass the boundary check: the producers reach underHome through os.Stat, which
// follows symlinks, so a purely lexical prefix test would accept it.
func TestUnderHomeRejectsSymlinkEscape(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()

	link := filepath.Join(home, ".claude")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if underHome(link, home) {
		t.Errorf("underHome(%q, %q) = true, want false: the link resolves to %q, outside the home", link, home, outside)
	}

	// A genuine directory inside the home is still accepted.
	realDir := filepath.Join(home, ".grok")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !underHome(realDir, home) {
		t.Errorf("underHome(%q, %q) = false, want true", realDir, home)
	}
}

// A .claude.json that carries no MCP configuration must not contribute the
// mcp_config marker, which would otherwise lend an ordinary project enough
// workspace shape to emit an agent row on its own.
func TestClaudeJSONWithoutMCPIsNotAnMCPConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".claude.json"), []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, m := range detectMarkers(root) {
		if m == "mcp_config" {
			t.Fatal("unrelated .claude.json produced an mcp_config marker")
		}
	}

	if err := os.WriteFile(filepath.Join(root, ".claude.json"), []byte(`{"mcpServers":{"fs":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, m := range detectMarkers(root) {
		if m == "mcp_config" {
			found = true
		}
	}
	if !found {
		t.Error("a .claude.json declaring mcpServers should produce an mcp_config marker")
	}
}
