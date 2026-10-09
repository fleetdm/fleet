package evidence

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// scanWorkspaces finds agent-shaped tool homes and project directories.
func scanWorkspaces(h homes.Home, dirs []fsutil.WalkedDir) []Workspace {
	var out []Workspace
	seen := map[string]struct{}{}

	emit := func(d fsutil.WalkedDir) {
		root := filepath.Clean(d.Path)
		if _, dup := seen[root]; root == "" || dup {
			return
		}
		markers := detectMarkers(d)
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
		if label, ok := toolHomeLabel(name); ok {
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
	for _, k := range knownToolHomes {
		p := filepath.Join(h.Dir, filepath.FromSlash(k.dir))
		if d, ok := fsutil.ListDir(p); ok {
			emit(d)
		}
	}

	for _, d := range dirs {
		// Only evaluate dirs that look promising to keep cost down.
		if d.Project && hasQuickHint(d) {
			emit(d)
		}
	}
	return out
}

// The names a walked directory is probed for: quickHints decide whether it's
// evaluated at all, and the marker lists classify it.
var (
	quickHints = []string{
		"AGENTS.md", "CLAUDE.md", "GEMINI.md", ".cursorrules",
		"mcp.json", ".mcp.json", "SOUL.md",
		filepath.Join(".cursor", "mcp.json"),
		"skills", ".agents",
	}
	instructionMarkers = []string{
		"AGENTS.md", "CLAUDE.md", "CLAUDE.local.md", "GEMINI.md",
		".cursorrules", ".windsurfrules", ".clinerules", "SOUL.md",
		filepath.Join(".github", "copilot-instructions.md"),
		filepath.Join(".codex", "AGENTS.md"),
		filepath.Join(".claude", "CLAUDE.md"),
	}
	skillDirMarkers  = []string{"skills", ".agents", filepath.Join(".agents", "skills"), "tools"}
	mcpConfigMarkers = []string{
		"mcp.json", ".mcp.json",
		filepath.Join(".cursor", "mcp.json"),
		filepath.Join(".vscode", "mcp.json"),
		filepath.Join(".claude", "settings.json"),
	}
	loopConfigMarkers = []string{
		"agent.yaml", "agent.yml", "agents.yaml", "agents.yml",
		"hermes.yaml", "openclaw.json", "openclaw.yaml",
		filepath.Join(".claude", "settings.json"),
		"permissions.json",
	}
	stateDirMarkers    = []string{"memory", ".memory", "sessions", ".sessions", "state", ".state"}
	frameworkManifests = []string{"package.json", "pyproject.toml", "requirements.txt"}
)

// WalkProbes are the paths the workspace and framework scans check in each
// walked directory, for fsutil.WalkHome.
func WalkProbes() []string {
	return slices.Concat(quickHints, instructionMarkers, skillDirMarkers, mcpConfigMarkers,
		loopConfigMarkers, stateDirMarkers, frameworkManifests,
		[]string{filepath.Join(".cursor", "rules"), ".claude.json"})
}

func hasQuickHint(d fsutil.WalkedDir) bool {
	return slices.ContainsFunc(quickHints, d.Exists)
}

// detectMarkers reads a directory's agent-shaped markers from its listing;
// only nested paths and marker contents touch the disk.
func detectMarkers(d fsutil.WalkedDir) []string {
	root := d.Path
	var m []string
	// Instructions
	if slices.ContainsFunc(instructionMarkers, d.Exists) {
		m = append(m, "instructions")
	}
	// Cursor rules dir
	if d.IsDir(".cursor") {
		if matches, _ := filepath.Glob(filepath.Join(root, ".cursor", "rules", "*")); len(matches) > 0 && !hasAny(m, "instructions") {
			m = append(m, "instructions")
		}
	}
	// Skills / tools tree
	for _, sd := range skillDirMarkers {
		if !d.IsDir(sd) {
			continue
		}
		if ents, err := os.ReadDir(filepath.Join(root, sd)); err == nil && len(ents) > 0 {
			m = append(m, "skills")
			break
		}
	}
	// MCP config
	for _, f := range mcpConfigMarkers {
		if d.Exists(f) && fileMentionsMCP(filepath.Join(root, f)) {
			m = append(m, "mcp_config")
			break
		}
	}
	// Also bare mcpServers in claude json at root. Presence alone is not the
	// signal: an unrelated .claude.json would otherwise lend a project enough
	// workspace shape to emit an agent row on its own.
	if d.Exists(".claude.json") && fileMentionsMCP(filepath.Join(root, ".claude.json")) {
		m = append(m, "mcp_config")
	}
	// Loop / harness config
	if slices.ContainsFunc(loopConfigMarkers, d.Exists) {
		m = append(m, "loop_config")
	}
	// State
	if slices.ContainsFunc(stateDirMarkers, d.IsDir) {
		m = append(m, "state")
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
