package ide

import (
	"context"
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

// vscodeAppNames maps a normalized application directory name (see
// vscodeEditorForApp) to the editor label the user-profile scanner already uses,
// so both sources report the same `source` value. Entries cover the macOS bundle,
// the Windows install folder, the Linux package directory, and the flatpak
// application id for each product.
//
// Unrecognized directories are skipped rather than reported under a guessed
// label: plenty of unrelated Electron apps also have a resources/app tree.
var vscodeAppNames = map[string]string{
	"visual studio code":             "vscode",
	"microsoft vs code":              "vscode",
	"code":                           "vscode",
	"vscode":                         "vscode",
	"com.visualstudio.code":          "vscode",
	"visual studio code insiders":    "vscode-insiders",
	"microsoft vs code insiders":     "vscode-insiders",
	"code insiders":                  "vscode-insiders",
	"vscode insiders":                "vscode-insiders",
	"vscode insider":                 "vscode-insiders",
	"com.visualstudio.code.insiders": "vscode-insiders",
	"vscodium":                       "vscodium",
	"codium":                         "vscodium",
	"code oss":                       "vscodium", // distro build of the OSS source, shares ~/.vscode-oss
	"com.vscodium.codium":            "vscodium",
	"cursor":                         "cursor",
	"windsurf":                       "windsurf",
	"trae":                           "trae",
	"antigravity":                    "antigravity",
	"code server":                    "code-server",
}

// platformTokens name the build rather than the product. The archives published
// on the download site unpack to a directory carrying them — "VSCode-win32-x64",
// "VSCode-linux-arm64" — which is what a hand-installed copy is usually left in,
// so they are dropped before the lookup.
var platformTokens = map[string]struct{}{
	"win32": {}, "win": {}, "windows": {}, "linux": {}, "darwin": {}, "mac": {}, "macos": {}, "osx": {},
	"x64": {}, "x86": {}, "x86_64": {}, "amd64": {}, "ia32": {}, "arm": {}, "arm64": {}, "armhf": {}, "aarch64": {},
}

// vscodeEditorForApp resolves an application directory name to its editor label.
//
// Packagers spell the same product many ways — "Visual Studio Code.app",
// "Visual Studio Code - Insiders", "visual-studio-code" (AUR), "code-insiders",
// "VSCode-win32-x64" (the published zip) — so the name is folded to lowercase
// words, stripped of build markers, and rejoined with single spaces before the
// lookup, letting one table entry cover every spelling.
func vscodeEditorForApp(dirName string) (string, bool) {
	n := strings.TrimSuffix(strings.ToLower(dirName), ".app")
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(n))
	kept := words[:0]
	for _, w := range words {
		if _, drop := platformTokens[w]; !drop {
			kept = append(kept, w)
		}
	}
	ed, ok := vscodeAppNames[strings.Join(kept, " ")]
	return ed, ok
}

// vscodeBundledDirs returns the candidate bundled-extensions directories inside
// an application directory. Every layout is tried on every platform — the ones
// that don't apply simply do not exist, and not branching on runtime.GOOS keeps
// this testable anywhere.
func vscodeBundledDirs(appDir string) []string {
	// Snap keeps the distro layout one level down, under a directory named after
	// the snap itself.
	snapName := strings.ToLower(filepath.Base(appDir))
	return []string{
		// macOS .app bundle.
		filepath.Join(appDir, "Contents", "Resources", "app", "extensions"),
		// Windows installs and Linux deb/rpm/tarball installs.
		filepath.Join(appDir, "resources", "app", "extensions"),
		// code-server embeds a whole VS Code under lib/vscode.
		filepath.Join(appDir, "lib", "vscode", "extensions"),
		// Snap and flatpak layouts are best-effort: a path that doesn't exist on a
		// given host costs one failed ReadDir.
		filepath.Join(appDir, "current", "usr", "share", snapName, "resources", "app", "extensions"),
		filepath.Join(appDir, "current", "active", "files", "extra", "vscode", "resources", "app", "extensions"),
	}
}

// appRoot is a directory whose immediate children may be VS Code-family
// application installs, together with how rows found beneath it are attributed.
// A system-scoped root has no owner: the install belongs to the host, not to a
// user, and home is left zero so the uid/username columns come out empty.
type appRoot struct {
	dir   string
	scope string
	home  homes.Home
}

// vscodeAppRoots is a variable so tests can point the bundled scan at a fixture
// instead of the host's real application directories.
var vscodeAppRoots = defaultVSCodeAppRoots

// defaultVSCodeAppRoots returns the directories to search for VS Code-family
// installs. Per-user locations are derived from each home rather than from the
// environment, matching the paths package: when running as root over another
// user's home we cannot read their environment.
func defaultVSCodeAppRoots(hs []homes.Home) []appRoot {
	var out []appRoot
	system := func(dirs ...string) {
		for _, d := range dirs {
			if d != "" {
				out = append(out, appRoot{dir: d, scope: scopeSystem})
			}
		}
	}
	perUser := func(dir func(paths.Roots) string) {
		for _, h := range hs {
			if d := dir(paths.For(h.Dir)); d != "" {
				out = append(out, appRoot{dir: d, scope: scopeUser, home: h})
			}
		}
	}

	switch runtime.GOOS {
	case "darwin":
		system("/Applications", "/Applications/Utilities")
		perUser(func(r paths.Roots) string { return filepath.Join(r.Home, "Applications") })
	case "windows":
		system(os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"))
		// The VS Code "user installer" — the download the site offers by default —
		// installs under LocalAppData\Programs, not Program Files.
		perUser(func(r paths.Roots) string { return filepath.Join(r.LocalAppData, "Programs") })
	default: // linux and other unix
		// deb/rpm packages land in /usr/share, tarball and AUR builds in /opt,
		// distro builds of the OSS source in /usr/lib; snap and flatpak keep their
		// own trees (the extra path layers are handled in vscodeBundledDirs).
		system("/usr/share", "/usr/local/share", "/opt", "/usr/lib", "/snap", "/var/lib/flatpak/app")
		perUser(func(r paths.Roots) string { return filepath.Join(r.XDGData, "flatpak", "app") })
	}
	return out
}

// vscodePluginKey identifies a plugin row for de-duplication within one editor,
// so a bundled extension is not reported alongside a copy of the same id that was
// already reported. VS Code itself gives the profile copy precedence when both are
// present for the same user.
//
// Scope decides how wide the key reaches, and the two scopes never share a key
// space. A system-scoped install belongs to the host, so one key covers the whole
// machine. A user-scoped one belongs to a single account, so its home is part of
// the key. Keeping them separate is what stops one account's copy from standing in
// for everyone: two users who each have the extension really do each have it, and
// a machine-wide install remains available to every account no matter who else
// happens to have installed their own copy.
func vscodePluginKey(scope, home, editor, id string) string {
	if scope == scopeSystem {
		return editor + "\x00" + id
	}
	return home + "\x00" + editor + "\x00" + id
}

// scanVSCodeBuiltins reports the AI extensions bundled inside every VS Code-family
// application installed on the host. seen holds the keys already reported from
// user profiles and is added to as rows are produced, so a given extension is
// reported once per editor.
//
// Traversal stops as soon as ctx is cancelled: reading every bundled manifest of
// every editor installed on the host is the most I/O this collector does, so it
// must not outlive the query that asked for it. The caller reports the
// cancellation.
func scanVSCodeBuiltins(ctx context.Context, hs []homes.Home, seen map[string]struct{}) []Plugin {
	return scanVSCodeBuiltinsIn(ctx, vscodeAppRoots(hs), seen)
}

func scanVSCodeBuiltinsIn(ctx context.Context, roots []appRoot, seen map[string]struct{}) []Plugin {
	var out []Plugin
	for _, root := range roots {
		if ctx.Err() != nil {
			return out
		}
		entries, err := os.ReadDir(root.dir)
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
			out = append(out, scanVSCodeAppDir(ctx, filepath.Join(root.dir, e.Name()), editor, root, seen)...)
		}
	}
	return out
}

// scanVSCodeAppDir reports the bundled extensions of one application install.
//
// The layouts in vscodeBundledDirs are tried against the application directory
// first. Only when none of them exists are the directory's immediate children
// tried as well, which is what VS Code 1.132 on Windows needs: it keeps just a
// launcher and a build-id directory at the top level
// (…\Microsoft VS Code\df53daabb1\resources\app\…), putting the application tree
// one level below where earlier builds put it. Probing children finds it whatever
// that directory is named, and the fallback stays free on the flat layouts —
// which matters on macOS, where a case-insensitive volume would otherwise let
// <app>/Contents/resources/... resolve onto the real Contents/Resources tree and
// scan every manifest a second time.
func scanVSCodeAppDir(ctx context.Context, appDir, editor string, root appRoot, seen map[string]struct{}) []Plugin {
	var out []Plugin
	var found bool
	for _, dir := range vscodeBundledDirs(appDir) {
		ps, ok := scanVSCodeBundledDir(ctx, dir, editor, root, seen)
		out, found = append(out, ps...), found || ok
	}
	if found {
		return out
	}
	children, err := os.ReadDir(appDir)
	if err != nil {
		return out
	}
	for _, c := range children {
		if !c.IsDir() {
			continue
		}
		for _, dir := range vscodeBundledDirs(filepath.Join(appDir, c.Name())) {
			ps, _ := scanVSCodeBundledDir(ctx, dir, editor, root, seen)
			out = append(out, ps...)
		}
	}
	return out
}

// scanVSCodeBundledDir reports the AI extensions in one bundled-extensions
// directory. The second return reports whether the directory existed at all,
// which is how the caller tells "this layout is not the one" apart from "this
// layout is right and holds no AI extensions".
func scanVSCodeBundledDir(ctx context.Context, dir, editor string, root appRoot, seen map[string]struct{}) ([]Plugin, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	var out []Plugin
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		extDir := filepath.Join(dir, e.Name())
		manifestPath := filepath.Join(extDir, "package.json")
		m, ok := readVSCodeManifest(manifestPath)
		if !ok {
			continue
		}
		m.DisplayName = vscodeDisplayName(extDir, m)
		p, cat, ok := vscodePluginFromManifest(m, editor, manifestPath, extDir)
		if !ok {
			continue
		}
		key := vscodePluginKey(root.scope, root.home.Dir, editor, p.PluginID)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		p.Scope = root.scope
		out = append(out, p.finish(root.home, cat))
	}
	return out, true
}

// vscodeDisplayName resolves a bundled manifest's localized display name.
// Built-in manifests set displayName to an "%nlsKey%" placeholder resolved from
// package.nls.json beside them (88 of the ~100 extensions VS Code 1.x bundles do
// this). Surfacing the raw placeholder would both show up in the row's name and
// rob classify of its only signal for an extension whose id is not in the KB, so
// an unresolvable placeholder falls back to the manifest name.
func vscodeDisplayName(extDir string, m vscodeManifest) string {
	key, ok := nlsPlaceholder(m.DisplayName)
	if !ok {
		return m.DisplayName
	}
	if s := lookupVSCodeNLS(extDir, key); s != "" {
		return s
	}
	return m.Name
}

// nlsPlaceholder reports whether s is an "%key%" localization placeholder, and
// returns the key it references.
func nlsPlaceholder(s string) (string, bool) {
	if len(s) < 3 || !strings.HasPrefix(s, "%") || !strings.HasSuffix(s, "%") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(s, "%"), "%"), true
}

// lookupVSCodeNLS returns the string bound to key in extDir's package.nls.json,
// or "" when the file is unreadable or the key is absent. A value is either a
// plain string or a {"message": ..., "comment": [...]} object — vscode-nls
// accepts both, and the bundled Copilot Chat manifest ships both forms.
func lookupVSCodeNLS(extDir, key string) string {
	b, err := fsutil.ReadFileBounded(filepath.Join(extDir, "package.nls.json"))
	if err != nil {
		return ""
	}
	var nls map[string]json.RawMessage
	if err := json.Unmarshal(b, &nls); err != nil {
		return ""
	}
	raw, ok := nls[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Message
	}
	return ""
}
