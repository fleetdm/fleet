package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// ---- Zed (extension.toml) ----

func zedExtensionsDirs(r paths.Roots) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(r.MacAppSupport, "Zed", "extensions", "installed")}
	case "windows":
		return []string{filepath.Join(r.LocalAppData, "Zed", "extensions", "installed")}
	default:
		return []string{filepath.Join(r.XDGData, "zed", "extensions", "installed")}
	}
}

func scanZed(h homes.Home, r paths.Roots) []Plugin {
	var out []Plugin
	for _, dir := range zedExtensionsDirs(r) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			manifest := filepath.Join(dir, e.Name(), "extension.toml")
			fields := readSimpleTOML(manifest)
			if len(fields) == 0 {
				continue
			}
			id := firstNonEmptyStr(fields["id"], e.Name())
			name := firstNonEmptyStr(fields["name"], id)
			isAI, cat := classifyByName(id + " " + name)
			if !isAI {
				continue // AI tools only — skip non-AI extensions
			}
			p := Plugin{
				Editor:       "zed",
				EditorFamily: "zed",
				PluginID:     id,
				Name:         name,
				Version:      fields["version"],
				InstallPath:  filepath.Join(dir, e.Name()),
				ManifestPath: manifest,
			}
			out = append(out, p.finish(h, cat))
		}
	}
	return out
}

// readSimpleTOML extracts top-level `key = "value"` pairs. extension.toml uses a
// flat header, so a full TOML parser (and its dependency) is unnecessary.
func readSimpleTOML(path string) map[string]string {
	b, err := os.ReadFile(path) // #nosec G304 -- fixed manifest name under enumerated dir
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := strings.Trim(strings.TrimSpace(v), `"'`)
		if _, exists := out[key]; !exists {
			out[key] = val
		}
	}
	return out
}

// ---- Sublime Text (Package Control package-metadata.json) ----

func sublimePackagesDirs(r paths.Roots) []string {
	switch runtime.GOOS {
	case "darwin":
		base := r.MacAppSupport
		return []string{
			filepath.Join(base, "Sublime Text", "Packages"),
			filepath.Join(base, "Sublime Text 3", "Packages"),
		}
	case "windows":
		return []string{
			filepath.Join(r.AppData, "Sublime Text", "Packages"),
			filepath.Join(r.AppData, "Sublime Text 3", "Packages"),
		}
	default:
		return []string{
			filepath.Join(r.XDGConfig, "sublime-text", "Packages"),
			filepath.Join(r.XDGConfig, "sublime-text-3", "Packages"),
		}
	}
}

type sublimeMeta struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func scanSublime(h homes.Home, r paths.Roots) []Plugin {
	var out []Plugin
	for _, dir := range sublimePackagesDirs(r) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			manifest := filepath.Join(dir, e.Name(), "package-metadata.json")
			version := ""
			if b, err := os.ReadFile(manifest); err == nil { // #nosec G304 -- fixed name under enumerated dir
				var m sublimeMeta
				if json.Unmarshal(b, &m) == nil {
					version = m.Version
				}
			} else {
				manifest = ""
			}
			isAI, cat := classifyByName(e.Name())
			if !isAI {
				continue // AI tools only — skip non-AI packages
			}
			p := Plugin{
				Editor:       "sublime",
				EditorFamily: "sublime",
				PluginID:     e.Name(),
				Name:         e.Name(),
				Version:      version,
				InstallPath:  filepath.Join(dir, e.Name()),
				ManifestPath: manifest,
			}
			out = append(out, p.finish(h, cat))
		}
	}
	return out
}

// ---- Neovim / Vim (plugin-manager directories; plugins are git repos) ----

func scanVim(h homes.Home) []Plugin {
	dirs := []struct{ editor, path string }{
		{"neovim", filepath.Join(h.Dir, ".local", "share", "nvim", "lazy")},
		{"neovim", filepath.Join(h.Dir, ".local", "share", "nvim", "site", "pack", "packer", "start")},
		{"neovim", filepath.Join(h.Dir, ".local", "share", "nvim", "site", "pack", "packer", "opt")},
		{"vim", filepath.Join(h.Dir, ".vim", "plugged")},
		{"vim", filepath.Join(h.Dir, ".vim", "pack")},
	}
	var out []Plugin
	for _, d := range dirs {
		entries, err := os.ReadDir(d.path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			isAI, cat := classifyByName(e.Name())
			if !isAI {
				continue // AI tools only — skip non-AI plugins
			}
			p := Plugin{
				Editor:       d.editor,
				EditorFamily: "vim",
				PluginID:     e.Name(),
				Name:         e.Name(),
				InstallPath:  filepath.Join(d.path, e.Name()),
			}
			out = append(out, p.finish(h, cat))
		}
	}
	return out
}

// ---- Emacs (ELPA) ----

func scanEmacs(h homes.Home) []Plugin {
	dirs := []string{
		filepath.Join(h.Dir, ".emacs.d", "elpa"),
		filepath.Join(h.Dir, ".config", "emacs", "elpa"),
	}
	var out []Plugin
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name, version := splitELPAName(e.Name())
			if name == "" {
				continue
			}
			isAI, cat := classifyByName(name)
			if !isAI {
				continue // AI tools only — skip non-AI packages
			}
			p := Plugin{
				Editor:       "emacs",
				EditorFamily: "emacs",
				PluginID:     name,
				Name:         name,
				Version:      version,
				InstallPath:  filepath.Join(dir, e.Name()),
			}
			out = append(out, p.finish(h, cat))
		}
	}
	return out
}

// splitELPAName splits "magit-20240101.1234" into ("magit", "20240101.1234").
func splitELPAName(dir string) (string, string) {
	i := strings.LastIndex(dir, "-")
	if i <= 0 || i == len(dir)-1 {
		return dir, ""
	}
	suffix := dir[i+1:]
	if suffix == "" || !(suffix[0] >= '0' && suffix[0] <= '9') {
		return dir, ""
	}
	return dir[:i], suffix
}
