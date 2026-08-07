package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// vscodeEditor maps an editor label to the home-relative extensions directory
// shared by the whole VS Code family. All variants use the same package.json
// manifest layout (folder named publisher.name-version).
type vscodeEditor struct {
	editor  string
	relPath string // relative to home
}

func vscodeEditors() []vscodeEditor {
	return []vscodeEditor{
		{"vscode", filepath.Join(".vscode", "extensions")},
		{"vscode-insiders", filepath.Join(".vscode-insiders", "extensions")},
		{"vscodium", filepath.Join(".vscode-oss", "extensions")},
		{"cursor", filepath.Join(".cursor", "extensions")},
		{"windsurf", filepath.Join(".windsurf", "extensions")},
		{"vscode-server", filepath.Join(".vscode-server", "extensions")},
		{"code-server", filepath.Join(".local", "share", "code-server", "extensions")},
		{"trae", filepath.Join(".trae", "extensions")},
		{"antigravity", filepath.Join(".antigravity", "extensions")},
		{"antigravity-ide", filepath.Join(".antigravity-ide", "extensions")},
	}
}

type vscodeManifest struct {
	Name        string `json:"name"`
	Publisher   string `json:"publisher"`
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
}

func scanVSCodeFamily(h homes.Home, r paths.Roots) []Plugin {
	var out []Plugin
	// Tracks what the user profile already reported, so the bundled pass below
	// does not report a second row for an extension installed both ways.
	seen := map[string]struct{}{}
	for _, ed := range vscodeEditors() {
		dir := filepath.Join(h.Dir, ed.relPath)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		obsolete := readObsolete(dir)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			folder := e.Name()
			if _, ok := obsolete[folder]; ok {
				continue
			}
			manifestPath := filepath.Join(dir, folder, "package.json")
			m, ok := readVSCodeManifest(manifestPath)
			if !ok {
				continue
			}
			p, cat, ok := vscodePluginFromManifest(m, ed.editor, manifestPath, filepath.Join(dir, folder))
			if !ok {
				continue // AI tools only — skip non-AI extensions
			}
			seen[vscodePluginKey(ed.editor, p.PluginID)] = struct{}{}
			out = append(out, p.finish(h, cat))
		}
	}
	return append(out, scanVSCodeBuiltins(h, r, seen)...)
}

// vscodePluginFromManifest derives a plugin row from a package.json, shared by
// the user-profile and bundled-extension scanners. It reports false for anything
// that is not an AI extension.
func vscodePluginFromManifest(m vscodeManifest, editor, manifestPath, installPath string) (Plugin, string, bool) {
	id := strings.ToLower(m.Publisher + "." + m.Name)
	if m.Publisher == "" {
		id = strings.ToLower(m.Name)
	}
	isAI, cat := classify.VSCodePlugin(id, m.DisplayName)
	if !isAI {
		return Plugin{}, "", false
	}
	return Plugin{
		Editor:       editor,
		EditorFamily: "vscode",
		PluginID:     id,
		Name:         firstNonEmptyStr(m.DisplayName, m.Name),
		Version:      m.Version,
		Publisher:    m.Publisher,
		InstallPath:  installPath,
		ManifestPath: manifestPath,
	}, cat, true
}

func readVSCodeManifest(path string) (vscodeManifest, bool) {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return vscodeManifest{}, false
	}
	var m vscodeManifest
	if err := json.Unmarshal(b, &m); err != nil || m.Name == "" {
		return vscodeManifest{}, false
	}
	return m, true
}

// readObsolete returns the set of extension folder names marked uninstalled in
// the extensions dir's .obsolete file ({"publisher.name-version": true}).
func readObsolete(dir string) map[string]struct{} {
	out := map[string]struct{}{}
	b, err := fsutil.ReadFileBounded(filepath.Join(dir, ".obsolete"))
	if err != nil {
		return out
	}
	var m map[string]bool
	if err := json.Unmarshal(b, &m); err != nil {
		return out
	}
	for k, v := range m {
		if v {
			out[k] = struct{}{}
		}
	}
	return out
}

func firstNonEmptyStr(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
