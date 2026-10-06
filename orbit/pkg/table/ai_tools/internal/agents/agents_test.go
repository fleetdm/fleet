package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/evidence"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
	"github.com/stretchr/testify/require"
)

func TestDetectClaudeCode(t *testing.T) {
	home := t.TempDir()

	// npm-global install with a manifest (version source) + a binary symlink target.
	write(t, filepath.Join(home, ".npm-global", "lib", "node_modules", "@anthropic-ai", "claude-code", "package.json"),
		`{"name":"@anthropic-ai/claude-code","version":"1.2.3"}`)
	writeExec(t, filepath.Join(home, ".local", "bin", "claude"))

	got := Scan(homes.Home{Dir: home, Username: "tester"}, &proc.Snapshot{Procs: map[int]proc.Process{}}, nil)

	var cc *Agent
	for i := range got {
		if got[i].Name == "claude-code" {
			cc = &got[i]
		}
	}
	if cc == nil {
		t.Fatalf("claude-code not detected; got %d agents", len(got))
	}
	if cc.Version != "1.2.3" {
		t.Errorf("version=%q want 1.2.3 (must come from manifest, not exec)", cc.Version)
	}
	if cc.InstallMethod != "npm-global" {
		t.Errorf("install_method=%q want npm-global", cc.InstallMethod)
	}
	if cc.Binary != "claude" {
		t.Errorf("binary=%q want claude", cc.Binary)
	}
}

func TestMarkRunning(t *testing.T) {
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "aider"))
	write(t, filepath.Join(home, ".local", "pipx", "venvs", "aider-chat", "pyvenv.cfg"), "home = /usr\n")

	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		55: {PID: 55, Name: "aider", Cmdline: "/home/u/.local/bin/aider --model gpt-4"},
	}}
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap, nil)
	for _, a := range got {
		if a.Name == "aider" {
			if a.Running != 1 || a.PID != 55 {
				t.Errorf("aider running not detected: %+v", a)
			}
			return
		}
	}
	t.Fatal("aider not detected")
}

func TestShortBinaryNoFalseRunning(t *testing.T) {
	// Catalog agent amazon-q uses binary "q". A process named "icq" must not
	// mark amazon-q as running.
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "q"))

	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		7: {PID: 7, Name: "icq", Exe: "/usr/bin/icq", Cmdline: "/usr/bin/icq"},
	}}
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap, nil)
	for _, a := range got {
		if a.Name == "amazon-q" && a.Running == 1 {
			t.Fatalf("amazon-q falsely marked running from process %q pid=%d", "icq", a.PID)
		}
	}
}

func TestShortBinaryExactRunning(t *testing.T) {
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "q"))

	snap := &proc.Snapshot{Procs: map[int]proc.Process{
		9: {PID: 9, Name: "q", Exe: filepath.Join(home, ".local", "bin", "q"), Cmdline: filepath.Join(home, ".local", "bin", "q") + " chat"},
	}}
	got := Scan(homes.Home{Dir: home, Username: "tester"}, snap, nil)
	for _, a := range got {
		if a.Name == "amazon-q" {
			if a.Running != 1 || a.PID != 9 {
				t.Fatalf("amazon-q should be running: %+v", a)
			}
			return
		}
	}
	t.Fatal("amazon-q not detected")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // test fixture: simulates an executable agent binary
		t.Fatal(err)
	}
}

func scanWithEvidence(t *testing.T, h homes.Home) []Agent {
	t.Helper()
	b := evidence.Gather(t.Context(), []homes.Home{h}, nil, map[string]struct{}{"agents": {}},
		map[string][]fsutil.WalkedDir{h.Dir: fsutil.WalkHome(h.Dir, evidence.WalkProbes())})
	return Scan(h, &proc.Snapshot{Procs: map[int]proc.Process{}}, b)
}

func byName(agents []Agent, name string) []Agent {
	var out []Agent
	for _, a := range agents {
		if a.Name == name {
			out = append(out, a)
		}
	}
	return out
}

func TestClaudeHomeWithSkillsIsOneRow(t *testing.T) {
	home := t.TempDir()
	writeExec(t, filepath.Join(home, ".local", "bin", "claude"))
	write(t, filepath.Join(home, ".claude", "CLAUDE.md"), "# rules")
	write(t, filepath.Join(home, ".claude", "skills", "qa", "SKILL.md"), "# qa")

	got := byName(scanWithEvidence(t, homes.Home{Dir: home, Username: "u"}), "claude-code")
	require.Len(t, got, 1, "%+v", got)
	require.Equal(t, 100, got[0].Confidence)
	require.Contains(t, got[0].Evidence, "catalog")
	require.Contains(t, got[0].Evidence, "workspace_shape")
	require.Equal(t, "agent-runtime", got[0].Category)
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(target, link))
}

func TestNativeInstallVersionAndResolvedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native installers link only on macOS and Linux")
	}
	for _, tc := range []struct {
		name, target, version string // target relative to home
	}{
		{"binary named by version", ".local/share/claude/versions/2.0.14", "2.0.14"},
		{"binary in version dir", ".local/share/claude/versions/2.0.14/claude", "2.0.14"},
		{"outside the installer's versions", "opt/claude", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			target := filepath.Join(home, filepath.FromSlash(tc.target))
			writeExec(t, target)
			link := filepath.Join(home, ".local", "bin", "claude")
			symlink(t, target, link)

			got := byName(Scan(homes.Home{Dir: home, Username: "u"}, nil, nil), "claude-code")
			require.Len(t, got, 1)
			require.Equal(t, tc.version, got[0].Version)
			require.Equal(t, target, got[0].Path)
			require.Equal(t, link, got[0].BinaryPath)
		})
	}
}

// A link the scanner won't resolve (fsutil.LinkTargetWithin covers which) stays
// the reported path, and no version is read from where it points.
func TestNativeInstallUnresolvedLinkKeepsLinkPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native installers link only on macOS and Linux")
	}
	home := t.TempDir()
	link := filepath.Join(home, ".local", "bin", "claude")
	symlink(t, filepath.Join(home, ".local", "share", "claude", "versions", "2.0.14"), link)

	got := byName(Scan(homes.Home{Dir: home, Username: "u"}, nil, nil), "claude-code")
	require.Len(t, got, 1)
	require.Empty(t, got[0].Version)
	require.Equal(t, link, got[0].Path)
}
