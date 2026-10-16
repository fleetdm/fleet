package winget

import (
	"encoding/json"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// newWingetPkgsFixture creates a git repository laid out like microsoft/winget-pkgs
// holding the given files (paths relative to the repository root).
func newWingetPkgsFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	dir := t.TempDir()
	for p, contents := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(contents), 0o644))
	}

	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "-A"},
		{"-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "fixture"},
	} {
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		require.NoError(t, err, string(out))
	}
	return dir
}

func marshalFixture(t *testing.T, v any) string {
	t.Helper()
	b, err := yaml.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// fooVersionFiles returns the installer and locale manifests of package Foo at version v.
func fooVersionFiles(t *testing.T, v string) map[string]string {
	return map[string]string{
		"manifests/f/Foo/" + v + "/Foo.installer.yaml": marshalFixture(t, installerManifest{
			ProductCode:    "{ABCDEF}",
			InstallerType:  "msi",
			Scope:          "machine",
			PackageVersion: v,
			Installers: []installer{
				{Architecture: "x64", InstallerType: "msi", ProductCode: "{ABCDEF}", Scope: "machine"},
			},
		}),
		"manifests/f/Foo/" + v + "/Foo.locale.en-US.yaml": marshalFixture(t, localeManifest{PackageName: "foo", Publisher: "Bar, Inc."}),
	}
}

func mergeFiles(ms ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range ms {
		maps.Copy(out, m)
	}
	return out
}

var fooInput = inputApp{
	Name:              "Foo",
	UniqueIdentifier:  "Foo",
	PackageIdentifier: "Foo",
	Slug:              "foo/windows",
	InstallerArch:     "x64",
	InstallerType:     "msi",
	InstallerScope:    "machine",
}

func TestIngestOneFromLocalRepo(t *testing.T) {
	newIngester := func(t *testing.T, files map[string]string) *wingetIngester {
		local, err := newLocalRepo(t.Context(), newWingetPkgsFixture(t, files))
		require.NoError(t, err)
		return &wingetIngester{logger: slog.New(slog.DiscardHandler), local: local}
	}

	t.Run("picks the newest version and ignores folders that are not versions", func(t *testing.T) {
		i := newIngester(t, mergeFiles(
			fooVersionFiles(t, "1.0"),
			fooVersionFiles(t, "2.0"),
			map[string]string{
				"manifests/f/Foo/Portable/Foo.installer.yaml": "not a version folder",
				"manifests/f/Foo/2030/Foo.installer.yaml":     "a bare year folder",
			},
		))

		out, err := i.ingestOne(t.Context(), fooInput)
		require.NoError(t, err)
		require.Equal(t, "2.0", out.Version)
	})

	t.Run("a newest version without an installer manifest falls through to the next", func(t *testing.T) {
		i := newIngester(t, mergeFiles(
			fooVersionFiles(t, "1.0"),
			map[string]string{
				"manifests/f/Foo/2.0/Foo.locale.en-US.yaml": marshalFixture(t, localeManifest{PackageName: "foo", Publisher: "Bar, Inc."}),
			},
		))

		out, err := i.ingestOne(t.Context(), fooInput)
		require.NoError(t, err)
		require.Equal(t, "1.0", out.Version)
	})

	t.Run("a version with an installer manifest but no locale manifest fails the app", func(t *testing.T) {
		files := fooVersionFiles(t, "1.0")
		delete(files, "manifests/f/Foo/1.0/Foo.locale.en-US.yaml")
		files["manifests/f/Foo/1.0/Foo.yaml"] = "version manifest"
		i := newIngester(t, files)

		_, err := i.ingestOne(t.Context(), fooInput)
		require.ErrorContains(t, err, "locale")
	})

	t.Run("an unknown package fails the app", func(t *testing.T) {
		i := newIngester(t, fooVersionFiles(t, "1.0"))

		input := fooInput
		input.PackageIdentifier = "Missing"
		_, err := i.ingestOne(t.Context(), input)
		require.ErrorContains(t, err, "no version manifest directories")
	})
}

func TestIngestAppsFromLocalRepo(t *testing.T) {
	t.Setenv(wingetPkgsRepoEnv, newWingetPkgsFixture(t, fooVersionFiles(t, "3.1")))

	inputs := t.TempDir()
	input, err := json.Marshal(map[string]any{
		"name":               "Foo",
		"slug":               "foo/windows",
		"unique_identifier":  "Foo",
		"package_identifier": "Foo",
		"installer_arch":     "x64",
		"installer_type":     "msi",
		"installer_scope":    "machine",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(inputs, "foo.json"), input, 0o644))

	apps, err := IngestApps(t.Context(), slog.New(slog.DiscardHandler), inputs, "")
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, "3.1", apps[0].Version)
}

func TestNewLocalRepoValidation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	_, err := newLocalRepo(t.Context(), filepath.Join(t.TempDir(), "missing"))
	require.ErrorContains(t, err, wingetPkgsRepoEnv)

	_, err = newLocalRepo(t.Context(), t.TempDir())
	require.ErrorContains(t, err, "not a git repository")
}
