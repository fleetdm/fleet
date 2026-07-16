// Package ide enumerates installed editor/IDE plugins across all major editor
// families by reading their on-disk install locations and manifests. It is
// fully self-contained (no dependency on osquery's built-in vscode_extensions
// table) and adds an AI-classification layer via the classify package.
package ide

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// Plugin is one installed editor extension/plugin.
type Plugin struct {
	UID, Username string
	Editor        string // vscode, cursor, intellij-idea, zed, sublime, neovim, emacs, ...
	EditorFamily  string // vscode | jetbrains | zed | sublime | vim | emacs
	PluginID      string
	Name          string
	Version       string
	Publisher     string
	InstallPath   string
	ManifestPath  string
	AICategory    string
}

// Scan returns every plugin discovered under the given home directory.
func Scan(h homes.Home) []Plugin {
	r := paths.For(h.Dir)
	var out []Plugin
	out = append(out, scanVSCodeFamily(h, r)...)
	out = append(out, scanJetBrains(h, r)...)
	out = append(out, scanZed(h, r)...)
	out = append(out, scanSublime(h, r)...)
	out = append(out, scanVim(h)...)
	out = append(out, scanEmacs(h)...)
	return out
}

// finish stamps ownership and the AI classification onto a plugin row. It is
// only called for AI-classified plugins — non-AI plugins are skipped at the
// scanner so the table surfaces AI tools only.
func (p Plugin) finish(h homes.Home, cat string) Plugin {
	p.UID, p.Username = h.UID, h.Username
	p.AICategory = cat
	return p
}

// classifyByName is the fallback classifier for editors without a curated id
// map (Zed, Sublime, Vim, Emacs).
func classifyByName(s string) (bool, string) { return classify.ByName(s) }
