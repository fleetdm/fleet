package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a git repo with a committed FMA outputs tree.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	writeFile := func(rel, content string) {
		t.Helper()
		full := filepath.Join(repoRoot, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}

	runGit("init", "--initial-branch", "main")

	writeFile("ee/maintained-apps/outputs/apps.json", `{
  "version": 1,
  "apps": [
    {"name": "Box Drive", "slug": "box-drive/darwin", "platform": "darwin", "unique_identifier": "com.box.desktop", "description": ""},
    {"name": "PuTTY", "slug": "putty/windows", "platform": "windows", "unique_identifier": "PuTTY release", "description": ""}
  ]
}
`)
	writeFile("ee/maintained-apps/outputs/box-drive/darwin.json", manifestJSON("2.52.0", "https://e3.boxcdn.net/BoxDrive-2.52.0.pkg", "aaaa"))
	writeFile("ee/maintained-apps/outputs/putty/windows.json", manifestJSON("0.83.0.0", "https://the.earth.li/putty-0.83.msi", "bbbb"))

	runGit("add", ".")
	runGit("commit", "-m", "base")

	return repoRoot
}

func manifestJSON(version, url, sha string) string {
	return `{
  "versions": [
    {
      "version": "` + version + `",
      "queries": {"exists": "SELECT 1;"},
      "installer_url": "` + url + `",
      "install_script_ref": "aaaaaaaa",
      "uninstall_script_ref": "bbbbbbbb",
      "sha256": "` + sha + `",
      "default_categories": ["Productivity"]
    }
  ],
  "refs": {"aaaaaaaa": "install", "bbbbbbbb": "uninstall"}
}
`
}

func testConfig(repoRoot string) *config {
	return &config{
		repoRoot:    repoRoot,
		changedFrom: "HEAD",
		logger:      slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}

func TestChangedAppsNoChanges(t *testing.T) {
	repoRoot := setupTestRepo(t)
	targets, err := changedApps(context.Background(), testConfig(repoRoot))
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestChangedAppsVersionBump(t *testing.T) {
	repoRoot := setupTestRepo(t)
	manifestPath := filepath.Join(repoRoot, "ee", "maintained-apps", "outputs", "putty", "windows.json")
	require.NoError(t, os.WriteFile(manifestPath,
		[]byte(manifestJSON("0.84.0.0", "https://the.earth.li/putty-0.84.msi", "cccc")), 0o644))

	targets, err := changedApps(context.Background(), testConfig(repoRoot))
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "putty/windows", targets[0].Slug)
	require.Equal(t, "PuTTY", targets[0].Name)
	require.False(t, targets[0].IsNew)
	require.ElementsMatch(t, []string{"version", "installer_url", "sha256"}, targets[0].ChangedFields)
	require.Equal(t, "0.84.0.0", targets[0].Manifest.Version)
	require.Equal(t, "windows", targets[0].Manifest.Platform())
}

func TestChangedAppsScriptOnlyChangeIgnored(t *testing.T) {
	repoRoot := setupTestRepo(t)
	manifestPath := filepath.Join(repoRoot, "ee", "maintained-apps", "outputs", "putty", "windows.json")
	content, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	// Change only the refs content, keeping version/url/sha identical.
	updated := []byte(string(content[:len(content)-len("\"uninstall\"}\n}\n")]) + "\"new uninstall\"}\n}\n")
	require.NoError(t, os.WriteFile(manifestPath, updated, 0o644))

	targets, err := changedApps(context.Background(), testConfig(repoRoot))
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestChangedAppsNewUntrackedApp(t *testing.T) {
	repoRoot := setupTestRepo(t)
	newManifest := filepath.Join(repoRoot, "ee", "maintained-apps", "outputs", "rectangle", "darwin.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(newManifest), 0o755))
	require.NoError(t, os.WriteFile(newManifest,
		[]byte(manifestJSON("0.98", "https://github.com/rxhanson/Rectangle0.98.dmg", "dddd")), 0o644))

	targets, err := changedApps(context.Background(), testConfig(repoRoot))
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "rectangle/darwin", targets[0].Slug)
	require.True(t, targets[0].IsNew)
	require.Equal(t, []string{"new"}, targets[0].ChangedFields)
	// Not in apps.json, so the name falls back to the slug token.
	require.Equal(t, "rectangle", targets[0].Name)
}

func TestChangedAppsIgnoresAppsJSONAndDeletions(t *testing.T) {
	repoRoot := setupTestRepo(t)
	// Modify apps.json (not an app manifest).
	appsJSON := filepath.Join(repoRoot, "ee", "maintained-apps", "outputs", "apps.json")
	content, err := os.ReadFile(appsJSON)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(appsJSON, append(content, '\n'), 0o644))
	// Delete an app manifest.
	require.NoError(t, os.Remove(filepath.Join(repoRoot, "ee", "maintained-apps", "outputs", "box-drive", "darwin.json")))

	targets, err := changedApps(context.Background(), testConfig(repoRoot))
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestAllApps(t *testing.T) {
	repoRoot := setupTestRepo(t)
	targets, err := allApps(testConfig(repoRoot))
	require.NoError(t, err)
	require.Len(t, targets, 2)
	require.Equal(t, "box-drive/darwin", targets[0].Slug)
	require.Equal(t, "putty/windows", targets[1].Slug)
}

func TestSlugFromManifestPath(t *testing.T) {
	testCases := []struct {
		path string
		slug string
		ok   bool
	}{
		{"ee/maintained-apps/outputs/box-drive/darwin.json", "box-drive/darwin", true},
		{"ee/maintained-apps/outputs/7-zip/windows.json", "7-zip/windows", true},
		{"ee/maintained-apps/outputs/apps.json", "", false},
		{"ee/maintained-apps/outputs/foo/linux.json", "", false},
		{"ee/maintained-apps/inputs/winget/box-drive.json", "", false},
		{"ee/maintained-apps/outputs/foo/bar/darwin.json", "", false},
	}
	for _, tc := range testCases {
		slug, ok := slugFromManifestPath(tc.path)
		require.Equal(t, tc.ok, ok, "path: %s", tc.path)
		require.Equal(t, tc.slug, slug, "path: %s", tc.path)
	}
}
