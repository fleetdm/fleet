package ide

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanVSCodeFamily(t *testing.T) {
	home := t.TempDir()
	extDir := filepath.Join(home, ".vscode", "extensions")

	write(t, filepath.Join(extDir, "github.copilot-1.250.0", "package.json"),
		`{"name":"copilot","publisher":"github","version":"1.250.0","displayName":"GitHub Copilot"}`)
	write(t, filepath.Join(extDir, "esbenp.prettier-vscode-10.4.0", "package.json"),
		`{"name":"prettier-vscode","publisher":"esbenp","version":"10.4.0","displayName":"Prettier"}`)
	// An uninstalled extension still on disk, marked obsolete — must be skipped.
	write(t, filepath.Join(extDir, "old.ext-0.0.1", "package.json"),
		`{"name":"ext","publisher":"old","version":"0.0.1","displayName":"Old"}`)
	write(t, filepath.Join(extDir, ".obsolete"), `{"old.ext-0.0.1": true}`)

	got := Scan(homes.Home{Dir: home, Username: "tester"})
	by := map[string]Plugin{}
	for _, p := range got {
		by[p.PluginID] = p
	}

	cop, ok := by["github.copilot"]
	if !ok {
		t.Fatalf("github.copilot not found; got %d plugins", len(got))
	}
	if cop.AICategory == "" {
		t.Errorf("copilot should be classified AI, got cat=%q", cop.AICategory)
	}
	if cop.Version != "1.250.0" || cop.Publisher != "github" || cop.EditorFamily != "vscode" {
		t.Errorf("copilot metadata wrong: %+v", cop)
	}
	// The table surfaces AI tools only: a non-AI extension (Prettier) must not appear.
	if _, ok := by["esbenp.prettier-vscode"]; ok {
		t.Error("non-AI prettier should not be surfaced (AI-only table)")
	}
	if _, ok := by["old.ext"]; ok {
		t.Error("obsolete extension old.ext should have been skipped")
	}
}

func TestSplitELPAName(t *testing.T) {
	cases := []struct{ in, name, ver string }{
		{"magit-20240101.1234", "magit", "20240101.1234"},
		{"company-mode-0.9.13", "company-mode", "0.9.13"},
		{"no-version-dir", "no-version-dir", ""},
	}
	for _, c := range cases {
		n, v := splitELPAName(c.in)
		if n != c.name || v != c.ver {
			t.Errorf("splitELPAName(%q) = (%q,%q) want (%q,%q)", c.in, n, v, c.name, c.ver)
		}
	}
}

func TestProductEditorName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"IntelliJIdea2026.1", "intellijidea"},
		{"PyCharm2025.3", "pycharm"},
		{"GoLand2026.1", "goland"},
	}
	for _, c := range cases {
		if got := productEditorName(c.in); got != c.want {
			t.Errorf("productEditorName(%q) = %q want %q", c.in, got, c.want)
		}
	}
}
