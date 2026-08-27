package evidence

import (
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// knownToolHomes maps directory name under $HOME to agent label.
// Catalog sugar + Tier B anchors for tools with private install roots.
var knownToolHomes = []struct {
	dir      string // relative to home
	name     string
	binaries []string
}{
	{".grok", "grok", []string{"grok"}},
	{".claude", "claude-code", []string{"claude"}},
	{".codex", "codex", []string{"codex"}},
	{".gemini", "gemini-cli", []string{"gemini"}},
	{".cursor", "cursor-agent", []string{"cursor-agent", "cursor"}},
	{".continue", "continue-cli", []string{"cn", "continue"}},
	{".openclaw", "openclaw", []string{"openclaw"}},
	{".hermes", "hermes", []string{"hermes"}},
	{".opencode", "opencode", []string{"opencode"}},
	{".aider", "aider", []string{"aider"}},
	{".config/opencode", "opencode", []string{"opencode"}},
	{".config/hermes", "hermes", []string{"hermes"}},
}

func scanToolHomes(h homes.Home) []ToolHome {
	var out []ToolHome
	for _, k := range knownToolHomes {
		p := filepath.Join(h.Dir, filepath.FromSlash(k.dir))
		fi, err := os.Stat(p)
		if err != nil || !fi.IsDir() {
			continue
		}
		th := ToolHome{
			UID:       h.UID,
			Username:  h.Username,
			Name:      k.name,
			Path:      p,
			HasConfig: dirHasContent(p),
		}
		// Prefer binary under tool home, then ~/.local/bin, then home/bin.
		for _, b := range k.binaries {
			if bp := findBinNear(h.Dir, p, b); bp != "" {
				th.BinaryPath = bp
				th.BinaryName = b
				break
			}
		}
		// Empty dirs with no config and no binary are noise.
		if !th.HasConfig && th.BinaryPath == "" {
			continue
		}
		out = append(out, th)
	}
	return out
}

func dirHasContent(dir string) bool {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	// Hidden entries count as content, so any entry at all is enough.
	return len(ents) > 0
}

func findBinNear(home, toolHome, name string) string {
	cands := []string{
		filepath.Join(toolHome, "bin", name),
		filepath.Join(toolHome, name),
		filepath.Join(home, ".local", "bin", name),
		filepath.Join(home, "bin", name),
		filepath.Join(home, ".grok", "bin", name),
	}
	for _, c := range cands {
		if fi, err := os.Lstat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}
