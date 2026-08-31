// Package ide enumerates installed editor/IDE plugins across all major editor
// families by reading their on-disk install locations and manifests. It is
// fully self-contained (no dependency on osquery's built-in vscode_extensions
// table) and adds an AI-classification layer via the classify package.
package ide

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// Scope values for Plugin.Scope.
const (
	scopeUser   = "user"   // installed under one user's home; attributed to that user
	scopeSystem = "system" // installed machine-wide; available to every user
)

// Plugin is one installed editor extension/plugin.
type Plugin struct {
	UID, Username string
	Scope         string // user | system
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

// Scan returns every plugin discovered across the given homes.
//
// It takes all homes at once rather than one per call because not every plugin
// belongs to a user: extensions bundled inside a machine-wide editor install are
// a property of the host, so they are collected once, after the per-home pass,
// and reported with system scope and no owner.
func Scan(ctx context.Context, hs []homes.Home) ([]Plugin, error) {
	var out []Plugin
	seen := map[string]struct{}{}
	for _, h := range hs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		r := paths.For(h.Dir)
		out = append(out, scanVSCodeProfiles(h, seen)...)
		out = append(out, scanJetBrains(h, r)...)
		out = append(out, scanZed(h, r)...)
		out = append(out, scanSublime(h, r)...)
		out = append(out, scanVim(h)...)
		out = append(out, scanEmacs(h)...)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out = append(out, scanVSCodeBuiltins(ctx, hs, seen)...)
	// The bundled pass stops early on cancellation and hands back what it had, so
	// re-check here: a partial inventory must be reported as the error it is, never
	// as a complete result.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// finish stamps ownership and the AI classification onto a plugin row. It is
// only called for AI-classified plugins — non-AI plugins are skipped at the
// scanner so the table surfaces AI tools only.
//
// A zero homes.Home leaves the owner columns empty, which is how a system-scoped
// row says "installed for the whole host, not for one user".
func (p Plugin) finish(h homes.Home, cat string) Plugin {
	p.UID, p.Username = h.UID, h.Username
	p.AICategory = cat
	if p.Scope == "" {
		p.Scope = scopeUser
	}
	return p
}

// classifyByName is the fallback classifier for editors without a curated id
// map (Zed, Sublime, Vim, Emacs).
func classifyByName(s string) (bool, string) { return classify.ByName(s) }

// firstNonEmptyStr returns a unless it is empty or all whitespace, otherwise b.
// Every family's scanner uses it to fall back from a manifest's display name to
// its internal name.
func firstNonEmptyStr(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
