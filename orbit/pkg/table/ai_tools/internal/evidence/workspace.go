package evidence

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// toolHomeLabels maps directory basenames under $HOME to agent names.
var toolHomeLabels = map[string]string{
	".grok": "grok", ".claude": "claude-code", ".codex": "codex",
	".gemini": "gemini-cli", ".cursor": "cursor-agent", ".continue": "continue-cli",
	".openclaw": "openclaw", ".hermes": "hermes", ".opencode": "opencode",
	".aider": "aider", "opencode": "opencode", "hermes": "hermes",
}

// scanWorkspaces finds agent-shaped user homes and bounded project roots.
func scanWorkspaces(h homes.Home) []Workspace {
	var out []Workspace
	seen := map[string]struct{}{}

	emit := func(root string, extra []string) {
		root = filepath.Clean(root)
		if _, dup := seen[root]; root == "" || dup {
			return
		}
		markers := detectMarkers(root)
		markers = append(markers, extra...)
		markers = unique(markers)
		if len(markers) == 0 {
			return
		}
		seen[root] = struct{}{}
		strong := isStrong(markers)
		cat := "agent-runtime"
		if hasAny(markers, "skills") || hasAny(markers, "loop_config") || countShape(markers) >= 3 {
			cat = "agent-harness"
		}
		name := filepath.Base(root)
		// Map known tool-home basenames to stable agent labels (avoid ".grok" rows).
		if label, ok := toolHomeLabels[name]; ok {
			name = label
		}
		out = append(out, Workspace{
			UID:      h.UID,
			Username: h.Username,
			Root:     root,
			Name:     name,
			Markers:  markers,
			Strong:   strong,
			Category: cat,
		})
	}

	// User-global tool homes are covered by ToolHomes gatherer for agents
	// inventory; still evaluate shape for harness labeling when strong.
	for _, rel := range []string{
		".claude", ".codex", ".gemini", ".cursor", ".grok",
		".openclaw", ".hermes", ".continue", ".opencode",
		filepath.Join(".config", "opencode"),
		filepath.Join(".config", "hermes"),
	} {
		p := filepath.Join(h.Dir, filepath.FromSlash(rel))
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			emit(p, nil)
		}
	}

	// Bounded project roots (same spirit as instructions collector).
	for _, root := range projectRoots(h.Dir) {
		fsutil.WalkBounded(root, 3, func(dir string) {
			// Only evaluate dirs that look promising to keep cost down.
			if hasQuickHint(dir) {
				emit(dir, nil)
			}
		})
	}
	return out
}

func hasQuickHint(dir string) bool {
	hints := []string{
		"AGENTS.md", "CLAUDE.md", "GEMINI.md", ".cursorrules",
		"mcp.json", ".mcp.json", "SOUL.md",
		filepath.Join(".cursor", "mcp.json"),
		"skills", ".agents",
	}
	for _, h := range hints {
		if fsutil.Exists(filepath.Join(dir, h)) {
			return true
		}
	}
	return false
}

func detectMarkers(root string) []string {
	var m []string
	// Instructions
	for _, f := range []string{
		"AGENTS.md", "CLAUDE.md", "CLAUDE.local.md", "GEMINI.md",
		".cursorrules", ".windsurfrules", ".clinerules", "SOUL.md",
		filepath.Join(".github", "copilot-instructions.md"),
		filepath.Join(".codex", "AGENTS.md"),
		filepath.Join(".claude", "CLAUDE.md"),
	} {
		if fsutil.Exists(filepath.Join(root, f)) {
			m = append(m, "instructions")
			break
		}
	}
	// Cursor rules dir
	if matches, _ := filepath.Glob(filepath.Join(root, ".cursor", "rules", "*")); len(matches) > 0 {
		if !hasAny(m, "instructions") {
			m = append(m, "instructions")
		}
	}
	// Skills / tools tree
	for _, d := range []string{"skills", ".agents", filepath.Join(".agents", "skills"), "tools"} {
		if fi, err := os.Stat(filepath.Join(root, d)); err == nil && fi.IsDir() {
			if ents, err := os.ReadDir(filepath.Join(root, d)); err == nil && len(ents) > 0 {
				m = append(m, "skills")
				break
			}
		}
	}
	// MCP config
	for _, f := range []string{
		"mcp.json", ".mcp.json",
		filepath.Join(".cursor", "mcp.json"),
		filepath.Join(".vscode", "mcp.json"),
		filepath.Join(".claude", "settings.json"),
	} {
		p := filepath.Join(root, f)
		if fsutil.Exists(p) && fileMentionsMCP(p) {
			m = append(m, "mcp_config")
			break
		}
	}
	// Also bare mcpServers in claude json at root. Presence alone is not the
	// signal: an unrelated .claude.json would otherwise lend a project enough
	// workspace shape to emit an agent row on its own.
	if p := filepath.Join(root, ".claude.json"); fsutil.Exists(p) && fileMentionsMCP(p) {
		m = append(m, "mcp_config")
	}
	// Loop / harness config
	for _, f := range []string{
		"agent.yaml", "agent.yml", "agents.yaml", "agents.yml",
		"hermes.yaml", "openclaw.json", "openclaw.yaml",
		filepath.Join(".claude", "settings.json"),
		"permissions.json",
	} {
		if fsutil.Exists(filepath.Join(root, f)) {
			m = append(m, "loop_config")
			break
		}
	}
	// State
	for _, d := range []string{"memory", ".memory", "sessions", ".sessions", "state", ".state"} {
		if fi, err := os.Stat(filepath.Join(root, d)); err == nil && fi.IsDir() {
			m = append(m, "state")
			break
		}
	}
	return unique(m)
}

func fileMentionsMCP(path string) bool {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return false
	}
	// Cap read
	if len(b) > 64<<10 {
		b = b[:64<<10]
	}
	s := string(b)
	return strings.Contains(s, "mcpServers") || strings.Contains(s, `"servers"`) ||
		strings.Contains(s, "mcp") || strings.Contains(s, "MCP")
}

func isStrong(markers []string) bool {
	return countShape(markers) >= 2
}

func countShape(markers []string) int {
	// Distinct shape families
	n := 0
	for _, fam := range []string{"instructions", "skills", "mcp_config", "loop_config", "state"} {
		if hasAny(markers, fam) {
			n++
		}
	}
	return n
}

func hasAny(ss []string, v string) bool {
	return slices.Contains(ss, v)
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, dup := seen[s]; s == "" || dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// projectRoots mirrors the instruction collector's cheap project discovery.
func projectRoots(home string) []string {
	cands := []string{
		filepath.Join(home, "Projects"),
		filepath.Join(home, "projects"),
		filepath.Join(home, "Developer"),
		filepath.Join(home, "dev"),
		filepath.Join(home, "src"),
		filepath.Join(home, "code"),
		filepath.Join(home, "repos"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "workspace"),
	}
	var out []string
	for _, c := range cands {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			out = append(out, c)
		}
	}
	return out
}
