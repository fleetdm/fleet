package ide

import (
	"path/filepath"
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
		{"Visual Studio Code - Insiders.app", "vscode-insiders"},
		{"code-insiders", "vscode-insiders"},
		{"VSCodium.app", "vscodium"},
		{"codium", "vscodium"},
		{"Cursor.app", "cursor"},
		{"Windsurf", "windsurf"},

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

	h := homes.Home{UID: "501", Username: "tester", Dir: t.TempDir()}
	got := scanVSCodeBuiltinsIn([]string{root}, h, map[string]struct{}{})

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
	if cop.UID != "501" || cop.Username != "tester" {
		t.Errorf("ownership not stamped: %+v", cop)
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

// A user-profile install of the same extension overrides the bundled copy in the
// editor itself, so it must not be reported twice.
func TestScanVSCodeBuiltinsSkipsAlreadySeen(t *testing.T) {
	root := t.TempDir()
	exts := filepath.Join(root, "code", "resources", "app", "extensions")
	write(t, filepath.Join(exts, "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.130.0","displayName":"Copilot Chat"}`)

	h := homes.Home{UID: "501", Username: "tester", Dir: t.TempDir()}
	seen := map[string]struct{}{
		vscodePluginKey("vscode", "github.copilot-chat"): {},
	}
	if got := scanVSCodeBuiltinsIn([]string{root}, h, seen); len(got) != 0 {
		t.Errorf("got %+v, want no rows: the user-profile copy was already reported", got)
	}
}
