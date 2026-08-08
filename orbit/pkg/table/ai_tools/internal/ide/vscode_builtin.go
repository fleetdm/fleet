package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
)

// Bundled ("built-in") extensions live inside the application install, not the
// user profile's extensions directory. GitHub Copilot Chat is the reason this
// matters: from VS Code 1.130 it ships bundled and the standalone marketplace
// extension is deprecated, so a profile-only scan reports nothing for a user who
// actively has Copilot enabled.

// vscodeAppNames maps an application directory name to the editor label already
// used by the user-profile scanner, so both sources report the same `source`
// value. Names cover the macOS bundle, the Windows install folder, and the Linux
// package directory for each product.
//
// Unrecognized directories are skipped rather than reported under a guessed
// label: plenty of unrelated Electron apps also have a resources/app tree.
var vscodeAppNames = map[string]string{
	"visual studio code":            "vscode",
	"microsoft vs code":             "vscode",
	"code":                          "vscode",
	"visual studio code - insiders": "vscode-insiders",
	"microsoft vs code insiders":    "vscode-insiders",
	"code-insiders":                 "vscode-insiders",
	"vscodium":                      "vscodium",
	"codium":                        "vscodium",
	"cursor":                        "cursor",
	"windsurf":                      "windsurf",
	"trae":                          "trae",
	"antigravity":                   "antigravity",
}

// vscodeEditorForApp resolves an application directory name to its editor label.
func vscodeEditorForApp(dirName string) (string, bool) {
	n := strings.ToLower(strings.TrimSuffix(dirName, ".app"))
	ed, ok := vscodeAppNames[n]
	return ed, ok
}

// vscodeBundledDirs returns the candidate bundled-extensions directories inside
// an application directory: the macOS .app bundle layout first, then the plain
// layout the Windows and Linux builds use. Both are tried on every platform —
// the wrong one simply does not exist, and not branching on runtime.GOOS keeps
// this testable anywhere.
func vscodeBundledDirs(appDir string) []string {
	return []string{
		filepath.Join(appDir, "Contents", "Resources", "app", "extensions"),
		filepath.Join(appDir, "resources", "app", "extensions"),
	}
}

// vscodeAppSearchRoots returns the directories whose immediate children may be
// VS Code-family application installs. Per-user locations come from the home
// being scanned rather than the environment, matching the paths package: when
// running as root over another user's home we cannot read their environment.
func vscodeAppSearchRoots(r paths.Roots) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Applications", filepath.Join(r.Home, "Applications")}
	case "windows":
		// The VS Code "user installer" — the download the site offers by default —
		// installs under LocalAppData\Programs, not Program Files.
		roots := []string{filepath.Join(r.LocalAppData, "Programs")}
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
			if v := os.Getenv(env); v != "" {
				roots = append(roots, v)
			}
		}
		return roots
	default: // linux and other unix
		return []string{"/usr/share", "/usr/local/share", "/opt", filepath.Join(r.XDGData, "applications")}
	}
}

// vscodePluginKey identifies a plugin row within one editor, so a bundled
// extension is not reported alongside a user-profile copy of the same id. VS Code
// itself gives the profile copy precedence when both are present.
func vscodePluginKey(editor, id string) string { return editor + "\x00" + id }

// scanVSCodeBuiltins reports the AI extensions bundled inside every VS Code-family
// application installed on the host. seen holds the keys already reported from
// the user profile and is added to as rows are produced.
func scanVSCodeBuiltins(h homes.Home, r paths.Roots, seen map[string]struct{}) []Plugin {
	return scanVSCodeBuiltinsIn(vscodeAppSearchRoots(r), h, seen)
}

func scanVSCodeBuiltinsIn(searchRoots []string, h homes.Home, seen map[string]struct{}) []Plugin {
	var out []Plugin
	for _, root := range searchRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			editor, ok := vscodeEditorForApp(e.Name())
			if !ok {
				continue
			}
			for _, dir := range vscodeBundledDirs(filepath.Join(root, e.Name())) {
				out = append(out, scanVSCodeBundledDir(dir, editor, h, seen)...)
			}
		}
	}
	return out
}

func scanVSCodeBundledDir(dir, editor string, h homes.Home, seen map[string]struct{}) []Plugin {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Plugin
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(dir, e.Name(), "package.json")
		m, ok := readVSCodeManifest(manifestPath)
		if !ok {
			continue
		}
		// Bundled manifests localize displayName as an "%nlsKey%" placeholder,
		// resolved from package.nls.json beside them.
		m.DisplayName = resolveVSCodeNLS(filepath.Join(dir, e.Name()), m.DisplayName)
		p, cat, ok := vscodePluginFromManifest(m, editor, manifestPath, filepath.Join(dir, e.Name()))
		if !ok {
			continue
		}
		key := vscodePluginKey(editor, p.PluginID)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p.finish(h, cat))
	}
	return out
}

// resolveVSCodeNLS resolves a "%key%" display name against package.nls.json in
// extDir. A value that is not a placeholder, or a key that isn't there, is
// returned unchanged.
func resolveVSCodeNLS(extDir, displayName string) string {
	if len(displayName) < 3 || !strings.HasPrefix(displayName, "%") || !strings.HasSuffix(displayName, "%") {
		return displayName
	}
	b, err := fsutil.ReadFileBounded(filepath.Join(extDir, "package.nls.json"))
	if err != nil {
		return displayName
	}
	// Values are usually strings but may be {"message": ..., "comment": ...}
	// objects, so decode loosely and only accept a plain string.
	var nls map[string]json.RawMessage
	if err := json.Unmarshal(b, &nls); err != nil {
		return displayName
	}
	var s string
	if err := json.Unmarshal(nls[strings.Trim(displayName, "%")], &s); err != nil || s == "" {
		return displayName
	}
	return s
}
