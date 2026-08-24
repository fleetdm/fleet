package ide

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

func TestVSCodeEditorForApp(t *testing.T) {
	cases := []struct {
		dirName string
		want    string
	}{
		{"Visual Studio Code.app", "vscode"}, // macOS bundle
		{"Microsoft VS Code", "vscode"},      // Windows install dir
		{"code", "vscode"},                   // Linux /usr/share/code
		{"visual-studio-code", "vscode"},     // AUR /opt/visual-studio-code
		{"com.visualstudio.code", "vscode"},  // flatpak application id
		{"Visual Studio Code - Insiders.app", "vscode-insiders"},
		{"code-insiders", "vscode-insiders"},
		{"visual_studio_code_insiders", "vscode-insiders"},
		// The published archives unpack under a build-tagged name, which is where a
		// hand-installed copy usually stays.
		{"VSCode-win32-x64", "vscode"},
		{"VSCode-linux-arm64", "vscode"},
		{"vscode", "vscode"},
		{"VSCode-win32-x64-insider", "vscode-insiders"},
		{"VSCodium.app", "vscodium"},
		{"codium", "vscodium"},
		{"code-oss", "vscodium"},
		{"Cursor.app", "cursor"},
		{"Windsurf", "windsurf"},
		{"code-server", "code-server"},

		{"Safari.app", ""}, // not a VS Code family app
		{"Slack", ""},      // Electron, but not an editor
		{"", ""},
	}
	for _, c := range cases {
		got, ok := vscodeEditorForApp(c.dirName)
		if c.want == "" {
			if ok {
				t.Errorf("vscodeEditorForApp(%q)=%q,true want no match", c.dirName, got)
			}
			continue
		}
		if !ok || got != c.want {
			t.Errorf("vscodeEditorForApp(%q)=%q,%v want %q,true", c.dirName, got, ok, c.want)
		}
	}
}

// TestScanVSCodeBuiltins covers the reported gap: Copilot Chat ships bundled
// inside the application itself on VS Code 1.130+, so it never appears in the
// user profile's extensions directory.
func TestScanVSCodeBuiltins(t *testing.T) {
	root := t.TempDir()

	// macOS bundle layout.
	codeExts := filepath.Join(root, "Visual Studio Code.app", "Contents", "Resources", "app", "extensions")
	write(t, filepath.Join(codeExts, "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.130.0","displayName":"%displayName%"}`)
	write(t, filepath.Join(codeExts, "copilot", "package.nls.json"),
		`{"displayName":"GitHub Copilot Chat","description":"AI chat"}`)
	// A non-AI built-in must be dropped, like any other non-AI extension.
	write(t, filepath.Join(codeExts, "git", "package.json"),
		`{"name":"git","publisher":"vscode","version":"1.0.0","displayName":"Git"}`)

	// Windows/Linux layout, same product family.
	codiumExts := filepath.Join(root, "VSCodium", "resources", "app", "extensions")
	write(t, filepath.Join(codiumExts, "continue", "package.json"),
		`{"name":"continue","publisher":"Continue","version":"0.9.0","displayName":"Continue"}`)

	// An unrelated application that happens to sit next to them.
	write(t, filepath.Join(root, "Slack.app", "Contents", "Resources", "app", "extensions", "x", "package.json"),
		`{"name":"copilot","publisher":"acme","version":"1.0.0"}`)

	got := scanVSCodeBuiltinsIn(t.Context(), []appRoot{{dir: root, scope: scopeSystem}}, map[string]struct{}{})

	by := map[string]Plugin{}
	for _, p := range got {
		by[p.PluginID] = p
	}
	if len(got) != 2 {
		t.Fatalf("got %d plugins, want 2 (copilot chat + continue): %+v", len(got), got)
	}

	cop, ok := by["github.copilot-chat"]
	if !ok {
		t.Fatalf("bundled Copilot Chat not detected; got %+v", got)
	}
	if cop.Editor != "vscode" || cop.EditorFamily != "vscode" {
		t.Errorf("editor=%q family=%q want vscode/vscode", cop.Editor, cop.EditorFamily)
	}
	// displayName is an nls placeholder in bundled manifests; it must be resolved.
	if cop.Name != "GitHub Copilot Chat" {
		t.Errorf("name=%q want %q resolved from package.nls.json", cop.Name, "GitHub Copilot Chat")
	}
	if cop.Version != "1.130.0" || cop.Publisher != "GitHub" {
		t.Errorf("version=%q publisher=%q want 1.130.0/GitHub", cop.Version, cop.Publisher)
	}
	// A machine-wide install belongs to the host, not to any one user.
	if cop.Scope != scopeSystem || cop.UID != "" || cop.Username != "" {
		t.Errorf("scope=%q uid=%q username=%q want system with no owner", cop.Scope, cop.UID, cop.Username)
	}
	if cop.ManifestPath != filepath.Join(codeExts, "copilot", "package.json") {
		t.Errorf("manifest_path=%q want the bundled manifest", cop.ManifestPath)
	}
	if cop.AICategory == "" {
		t.Error("AICategory empty")
	}

	if _, ok := by["continue.continue"]; !ok {
		t.Errorf("bundled extension under the plain layout not detected; got %+v", got)
	}
	if _, ok := by["vscode.git"]; ok {
		t.Error("non-AI built-in should be dropped")
	}
	if _, ok := by["acme.copilot"]; ok {
		t.Error("extensions under a non-editor application should be ignored")
	}
}

// VS Code 1.132 on Windows keeps a launcher at the top of the install directory
// and the application itself under a build-id directory, so the bundled tree sits
// one level deeper than on earlier builds.
func TestScanVSCodeBuiltinsBuildIDLayout(t *testing.T) {
	root := t.TempDir()
	install := filepath.Join(root, "Microsoft VS Code")
	write(t, filepath.Join(install, "df53daabb1", "resources", "app", "extensions", "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.132.0","displayName":"GitHub Copilot"}`)
	// Siblings of the build-id directory must not confuse the probe.
	write(t, filepath.Join(install, "bin", "code.cmd"), "@echo off")

	h := homes.Home{UID: "S-1-5-21-1001", Username: "juan", Dir: t.TempDir()}
	got := scanVSCodeBuiltinsIn(t.Context(), []appRoot{{dir: root, scope: scopeUser, home: h}}, map[string]struct{}{})
	if len(got) != 1 {
		t.Fatalf("got %d plugins, want 1 under the build-id layout: %+v", len(got), got)
	}
	if got[0].PluginID != "github.copilot-chat" || got[0].Editor != "vscode" {
		t.Errorf("id=%q editor=%q want github.copilot-chat/vscode", got[0].PluginID, got[0].Editor)
	}
	if got[0].Username != "juan" || got[0].Scope != scopeUser {
		t.Errorf("username=%q scope=%q want juan/user (a per-user installer)", got[0].Username, got[0].Scope)
	}
}

// An editor installed into a user's own home is that user's, so its bundled
// extensions are attributed to them rather than to the host.
func TestScanVSCodeBuiltinsUserScope(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "Cursor.app", "Contents", "Resources", "app", "extensions", "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.0.0","displayName":"GitHub Copilot Chat"}`)

	h := homes.Home{UID: "501", Username: "tester", Dir: t.TempDir()}
	got := scanVSCodeBuiltinsIn(t.Context(), []appRoot{{dir: root, scope: scopeUser, home: h}}, map[string]struct{}{})
	if len(got) != 1 {
		t.Fatalf("got %d plugins, want 1: %+v", len(got), got)
	}
	if got[0].Scope != scopeUser || got[0].UID != "501" || got[0].Username != "tester" {
		t.Errorf("scope=%q uid=%q username=%q want user/501/tester", got[0].Scope, got[0].UID, got[0].Username)
	}
	if got[0].Editor != "cursor" {
		t.Errorf("editor=%q want cursor", got[0].Editor)
	}
}

// A user-profile install of the same extension overrides the bundled copy in the
// editor itself, so it must not be reported twice.
func TestScanVSCodeBuiltinsSkipsAlreadySeen(t *testing.T) {
	root := t.TempDir()
	exts := filepath.Join(root, "code", "resources", "app", "extensions")
	write(t, filepath.Join(exts, "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.130.0","displayName":"Copilot Chat"}`)

	seen := map[string]struct{}{
		vscodePluginKey(scopeSystem, "", "vscode", "github.copilot-chat"): {},
	}
	if got := scanVSCodeBuiltinsIn(t.Context(), []appRoot{{dir: root, scope: scopeSystem}}, seen); len(got) != 0 {
		t.Errorf("got %+v, want no rows: the user-profile copy was already reported", got)
	}
}

// The bundled scan runs once for the host, not once per user: an extension inside
// a machine-wide install must not be reported one time per account on the box,
// and must not be attributed to accounts that never opened the editor.
func TestScanReportsMachineWideBuiltinOnce(t *testing.T) {
	apps := t.TempDir()
	write(t, filepath.Join(apps, "Visual Studio Code.app", "Contents", "Resources", "app", "extensions", "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.130.0","displayName":"GitHub Copilot Chat"}`)
	useAppRoots(t, appRoot{dir: apps, scope: scopeSystem})

	hs := []homes.Home{
		{UID: "501", Username: "alice", Dir: t.TempDir()},
		{UID: "502", Username: "bob", Dir: t.TempDir()},
		{UID: "503", Username: "carol", Dir: t.TempDir()},
	}
	got, err := Scan(t.Context(), hs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows for %d homes, want 1 machine-wide row: %+v", len(got), len(hs), got)
	}
	if got[0].Scope != scopeSystem || got[0].UID != "" {
		t.Errorf("scope=%q uid=%q want a system row with no owner", got[0].Scope, got[0].UID)
	}
}

// Two users who each have their own copy of an editor each have the extension it
// bundles, so both are reported. De-duplication is per home for user-scoped rows —
// one account's copy must never hide another's.
func TestScanVSCodeBuiltinsPerHomeDedup(t *testing.T) {
	alice := homes.Home{UID: "501", Username: "alice", Dir: t.TempDir()}
	bob := homes.Home{UID: "502", Username: "bob", Dir: t.TempDir()}

	roots := make([]appRoot, 0, 2)
	for _, h := range []homes.Home{alice, bob} {
		appsDir := filepath.Join(h.Dir, "Applications")
		write(t, filepath.Join(appsDir, "Visual Studio Code.app", "Contents", "Resources", "app", "extensions", "copilot", "package.json"),
			`{"name":"copilot-chat","publisher":"GitHub","version":"1.132.0","displayName":"GitHub Copilot"}`)
		roots = append(roots, appRoot{dir: appsDir, scope: scopeUser, home: h})
	}

	got := scanVSCodeBuiltinsIn(t.Context(), roots, map[string]struct{}{})
	if len(got) != 2 {
		t.Fatalf("got %d rows, want one per home: %+v", len(got), got)
	}
	byUser := map[string]Plugin{}
	for _, p := range got {
		byUser[p.Username] = p
	}
	for _, want := range []string{"alice", "bob"} {
		if _, ok := byUser[want]; !ok {
			t.Errorf("no row for %s; got %+v", want, got)
		}
	}
}

// The same user's profile copy still takes precedence over that user's bundled
// copy — the per-home key is shared by both passes.
func TestScanPerHomeProfileWinsOverUserBundled(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".vscode", "extensions", "github.copilot-chat-1.131.0", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.131.0","displayName":"GitHub Copilot"}`)
	appsDir := filepath.Join(home, "Applications")
	write(t, filepath.Join(appsDir, "Visual Studio Code.app", "Contents", "Resources", "app", "extensions", "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.132.0","displayName":"GitHub Copilot"}`)

	h := homes.Home{UID: "501", Username: "alice", Dir: home}
	useAppRoots(t, appRoot{dir: appsDir, scope: scopeUser, home: h})

	got, err := Scan(t.Context(), []homes.Home{h})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(got), got)
	}
	if got[0].Version != "1.131.0" {
		t.Errorf("version=%q want the profile copy 1.131.0", got[0].Version)
	}
}

// One user having their own copy says nothing about anyone else, so it must not
// suppress the machine-wide row: the editor is installed for the whole host and
// every other account still has the extension it bundles.
func TestScanUserCopyDoesNotShadowSystemCopy(t *testing.T) {
	apps := t.TempDir()
	write(t, filepath.Join(apps, "Visual Studio Code.app", "Contents", "Resources", "app", "extensions", "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.130.0","displayName":"GitHub Copilot Chat"}`)
	useAppRoots(t, appRoot{dir: apps, scope: scopeSystem})

	alice := t.TempDir()
	write(t, filepath.Join(alice, ".vscode", "extensions", "github.copilot-chat-1.131.0", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.131.0","displayName":"GitHub Copilot Chat"}`)

	got, err := Scan(t.Context(), []homes.Home{
		{UID: "501", Username: "alice", Dir: alice},
		{UID: "502", Username: "bob", Dir: t.TempDir()}, // no copy of his own
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want alice's profile copy and the machine-wide one: %+v", len(got), got)
	}
	byScope := map[string]Plugin{}
	for _, p := range got {
		byScope[p.Scope] = p
	}
	user, ok := byScope[scopeUser]
	if !ok {
		t.Fatalf("alice's profile copy missing: %+v", got)
	}
	if user.Version != "1.131.0" || user.Username != "alice" {
		t.Errorf("version=%q username=%q want alice's 1.131.0", user.Version, user.Username)
	}
	// Bob is covered by this row: it is what makes the extension visible for every
	// account that has no copy of its own.
	sys, ok := byScope[scopeSystem]
	if !ok {
		t.Fatalf("machine-wide copy suppressed by alice's; bob is now invisible: %+v", got)
	}
	if sys.Version != "1.130.0" || sys.UID != "" {
		t.Errorf("version=%q uid=%q want the bundled 1.130.0 with no owner", sys.Version, sys.UID)
	}
}

func TestVSCodeDisplayName(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "package.nls.json"), `{
		"plain": "Plain String",
		"obj": {"message": "Object Form", "comment": ["translators: ..."]},
		"empty": ""
	}`)

	cases := []struct {
		name        string
		displayName string
		manifest    string
		want        string
	}{
		{"literal is kept as-is", "GitHub Copilot", "copilot-chat", "GitHub Copilot"},
		{"string value is resolved", "%plain%", "copilot-chat", "Plain String"},
		// vscode-nls also allows {"message": ...}; the bundled Copilot Chat manifest
		// ships entries in that form.
		{"object value is resolved", "%obj%", "copilot-chat", "Object Form"},
		{"missing key falls back to name", "%nope%", "copilot-chat", "copilot-chat"},
		{"empty value falls back to name", "%empty%", "copilot-chat", "copilot-chat"},
		{"unreadable nls falls back to name", "%plain%", "other", "other"},
		{"bare percent is not a placeholder", "%", "copilot-chat", "%"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			extDir := dir
			if c.name == "unreadable nls falls back to name" {
				extDir = filepath.Join(dir, "missing")
			}
			got := vscodeDisplayName(extDir, vscodeManifest{Name: c.manifest, DisplayName: c.displayName})
			if got != c.want {
				t.Errorf("vscodeDisplayName(%q) = %q want %q", c.displayName, got, c.want)
			}
		})
	}
}

// An unresolvable placeholder must never reach the row, because classify uses the
// display name as its fallback signal for ids that are not in the KB.
func TestScanVSCodeBuiltinsUnresolvedPlaceholderUsesName(t *testing.T) {
	root := t.TempDir()
	// No package.nls.json beside it, and an id classify does not know.
	write(t, filepath.Join(root, "Cursor.app", "Contents", "Resources", "app", "extensions", "x", "package.json"),
		`{"name":"tabnine-chat","publisher":"Acme","version":"1.0.0","displayName":"%displayName%"}`)

	got := scanVSCodeBuiltinsIn(t.Context(), []appRoot{{dir: root, scope: scopeSystem}}, map[string]struct{}{})
	if len(got) != 1 {
		t.Fatalf("got %d plugins, want 1 (classified via the manifest name): %+v", len(got), got)
	}
	if got[0].Name != "tabnine-chat" {
		t.Errorf("name=%q want the manifest name, never a raw %%placeholder%%", got[0].Name)
	}
}

func TestVSCodeBundledDirs(t *testing.T) {
	got := vscodeBundledDirs(filepath.Join("/snap", "code"))
	want := filepath.Join("/snap", "code", "current", "usr", "share", "code", "resources", "app", "extensions")
	if !slices.Contains(got, want) {
		t.Errorf("snap layout %q not among candidates %q", want, got)
	}
}
