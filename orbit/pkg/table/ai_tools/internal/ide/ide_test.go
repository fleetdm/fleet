package ide

import (
	"context"
	"errors"
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

// useAppRoots points the bundled-extension scan at fixture directories for the
// duration of the test, so Scan never reads the applications actually installed
// on the machine running the tests.
func useAppRoots(t *testing.T, roots ...appRoot) {
	t.Helper()
	prev := vscodeAppRoots
	vscodeAppRoots = func([]homes.Home) []appRoot { return roots }
	t.Cleanup(func() { vscodeAppRoots = prev })
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

	useAppRoots(t) // no application installs: this test covers the profile scan only

	got, err := Scan(t.Context(), []homes.Home{{Dir: home, Username: "tester"}})
	if err != nil {
		t.Fatal(err)
	}
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
	// A profile extension belongs to the user whose home it was found in.
	if cop.Scope != scopeUser || cop.Username != "tester" {
		t.Errorf("scope=%q username=%q want user/tester", cop.Scope, cop.Username)
	}
	// The table surfaces AI tools only: a non-AI extension (Prettier) must not appear.
	if _, ok := by["esbenp.prettier-vscode"]; ok {
		t.Error("non-AI prettier should not be surfaced (AI-only table)")
	}
	if _, ok := by["old.ext"]; ok {
		t.Error("obsolete extension old.ext should have been skipped")
	}
}

// cancelAfterCtx reports itself cancelled only from its nth Err call onwards,
// which lands the cancellation inside the bundled pass rather than at the checks
// that precede it.
type cancelAfterCtx struct {
	context.Context
	n     int
	calls int
}

func (c *cancelAfterCtx) Err() error {
	c.calls++
	if c.calls >= c.n {
		return context.Canceled
	}
	return nil
}

// A query cancelled while the bundled pass is walking application directories must
// stop there and be reported as cancelled — never answered with a partial
// inventory that reads like a complete one.
func TestScanStopsOnCancellation(t *testing.T) {
	apps := t.TempDir()
	write(t, filepath.Join(apps, "Visual Studio Code.app", "Contents", "Resources", "app", "extensions", "copilot", "package.json"),
		`{"name":"copilot-chat","publisher":"GitHub","version":"1.132.0","displayName":"GitHub Copilot"}`)
	useAppRoots(t, appRoot{dir: apps, scope: scopeSystem})

	// 3 clears the per-home and pre-bundled checks, so cancellation first bites
	// once the bundled scan is already under way.
	ctx := &cancelAfterCtx{Context: t.Context(), n: 3}

	got, err := Scan(ctx, []homes.Home{{UID: "501", Username: "tester", Dir: t.TempDir()}})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err=%v want context.Canceled", err)
	}
	if got != nil {
		t.Errorf("got %+v, want no rows alongside the error", got)
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
