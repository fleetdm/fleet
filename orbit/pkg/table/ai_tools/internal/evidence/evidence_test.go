package evidence

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
	"github.com/stretchr/testify/require"
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
	b := Gather(t.Context(), []homes.Home{h}, snap, map[string]struct{}{"agents": {}}, walks(h))
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
	b := Gather(t.Context(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}}, walks(h))
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
	b := Gather(t.Context(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}}, walks(h))
	cands := AgentCandidates(h, nil, b)
	for _, c := range cands {
		if filepath.Base(c.Path) == "notes" {
			t.Fatalf("weak AGENTS-only project must not emit agents candidate: %+v", c)
		}
	}
}

func TestFrameworkCrewAI(t *testing.T) {
	for _, tc := range []struct{ file, content string }{
		{"package.json", `{"name":"bot","dependencies":{"crewai":"^1.0.0"}}`},
		{"requirements.txt", "crewai>=0.80\n"},
		{"pyproject.toml", "[project]\ndependencies = [\"crewai>=0.1\"]\n"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			home := t.TempDir()
			write(t, filepath.Join(home, "src", "bot", tc.file), tc.content)

			h := homes.Home{Dir: home, Username: "u"}
			b := Gather(t.Context(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}}, walks(h))
			cands := AgentCandidates(h, nil, b)
			require.Len(t, cands, 1, "frameworks=%+v cands=%+v", b.Frameworks, cands)
			require.Equal(t, "bot", cands[0].Name)
			require.True(t, cands[0].Signals.Has("framework:crewai"))
			require.Equal(t, "agent-harness", cands[0].Category)
		})
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
	b := Gather(t.Context(), homesList, nil, map[string]struct{}{"agents": {}}, walks(homesList...))

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
	for _, m := range detectMarkers(listDir(t, root)) {
		if m == "mcp_config" {
			t.Fatal("unrelated .claude.json produced an mcp_config marker")
		}
	}

	if err := os.WriteFile(filepath.Join(root, ".claude.json"), []byte(`{"mcpServers":{"fs":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, m := range detectMarkers(listDir(t, root)) {
		if m == "mcp_config" {
			found = true
		}
	}
	if !found {
		t.Error("a .claude.json declaring mcpServers should produce an mcp_config marker")
	}
}

func listDir(t *testing.T, dir string) fsutil.WalkedDir {
	t.Helper()
	d, ok := fsutil.ListDir(dir)
	if !ok {
		t.Fatalf("list %s", dir)
	}
	return d
}

func walks(hs ...homes.Home) map[string][]fsutil.WalkedDir {
	m := map[string][]fsutil.WalkedDir{}
	for _, h := range hs {
		m[h.Dir] = fsutil.WalkHome(h.Dir, WalkProbes())
	}
	return m
}

func TestPythonDependencyNamesIgnoreSpecifiers(t *testing.T) {
	for _, tc := range []struct {
		name, file, content string
		want                []string
	}{
		{"bare", "pyproject.toml", `dependencies = ["crewai", "requests"]`, []string{"crewai"}},
		{"lower bound", "pyproject.toml", `dependencies=["crewai>=0.1"]`, []string{"crewai"}},
		{"pinned", "pyproject.toml", `dependencies = ["crewai==0.80.0"]`, []string{"crewai"}},
		{"compatible", "pyproject.toml", "dependencies = [\n  'crewai~=0.80',\n]", []string{"crewai"}},
		{"extras", "pyproject.toml", `dependencies = ["crewai[tools]>=0.80"]`, []string{"crewai"}},
		{"marker", "pyproject.toml", `dependencies = ["crewai; python_version>='3.10'"]`, []string{"crewai"}},
		{"normalized name", "pyproject.toml", `dependencies = ["Semantic_Kernel>=1"]`, []string{"semantic-kernel"}},
		{"separator run", "requirements.txt", "Semantic__Kernel\nlangchain._core\n", []string{"langchain", "semantic-kernel"}},
		{"poetry table", "pyproject.toml", "[tool.poetry.dependencies]\npython = \"^3.11\"\ncrewai = \"^0.80\"", []string{"crewai"}},
		{"longer name", "pyproject.toml", `dependencies = ["crewai-tools>=0.1", "notcrewai"]`, nil},
		{"prose", "pyproject.toml", `description = "A crewai bot"`, nil},
		{"requirements", "requirements.txt", "# agents\nrequests\nCrewAI>=0.80  # pinned\n-r base.txt\nlangchain-core==0.3; python_version>'3.9'\n", []string{"crewai", "langchain"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), tc.file)
			write(t, p, tc.content)
			got := parsePythonDependencyNames(p)
			slices.Sort(got)
			require.Equal(t, tc.want, got)
		})
	}
}

// A tool's own home that looks like an agent workspace (Claude Code creates
// ~/.claude/skills) is that tool, not a second agent.
func TestToolHomeWorkspaceShapeJoinsToolHomeCandidate(t *testing.T) {
	for _, tc := range []struct{ dir, name string }{
		{".claude", "claude-code"},
		{".hermes", "hermes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			write(t, filepath.Join(home, tc.dir, "AGENTS.md"), "# rules")
			write(t, filepath.Join(home, tc.dir, "skills", "qa", "SKILL.md"), "# qa")

			h := homes.Home{Dir: home, Username: "u"}
			b := Gather(t.Context(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}}, walks(h))
			cands := AgentCandidates(h, nil, b)
			require.Len(t, cands, 1, "%+v", cands)
			c := cands[0]
			require.Equal(t, tc.name, c.Name)
			require.True(t, c.Signals.Has("tool_home"))
			require.True(t, c.Signals.Has("workspace_shape"))
			// A catalog tool's row keeps its own category when this merges into
			// it (see agents); a tool known only by its home is a harness.
			require.Equal(t, "agent-harness", c.Category)
		})
	}
}

// A global npm or pipx install of a catalog agent is that agent, which the
// catalog already reports with its version; only a project depending on one,
// or a globally installed framework, is framework evidence.
func TestGlobalInstallOfCatalogAgentIsNotAFramework(t *testing.T) {
	home := t.TempDir()
	nm := filepath.Join(home, ".npm-global", "lib", "node_modules")
	write(t, filepath.Join(nm, "@anthropic-ai", "claude-code", "package.json"), `{"name":"@anthropic-ai/claude-code","version":"2.1.0"}`)
	write(t, filepath.Join(nm, "crewai", "package.json"), `{"name":"crewai"}`)
	write(t, filepath.Join(home, ".local", "pipx", "venvs", "aider-chat", "pyvenv.cfg"), "home = /usr")
	write(t, filepath.Join(home, "src", "sdkbot", "package.json"), `{"dependencies":{"@anthropic-ai/claude-code":"^2"}}`)

	h := homes.Home{Dir: home, Username: "u"}
	b := Gather(t.Context(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}}, walks(h))
	got := map[string]string{}
	for _, fw := range b.Frameworks {
		got[fw.Name+"@"+fw.Source] = fw.Path
	}
	require.Contains(t, got, "crewai@global-node")
	require.Contains(t, got, "claude-code@package.json", "a project depending on a catalog agent is still evidence")
	require.NotContains(t, got, "claude-code@global-node")
	require.NotContains(t, got, "aider@pipx")
}
