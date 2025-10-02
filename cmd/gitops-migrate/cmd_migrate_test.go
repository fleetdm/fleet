package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var (
	// The software+team YAML files.
	//
	//go:embed testdata
	testdata embed.FS
	// These only exist here for the purposes of compile-time guarantees that
	// these files exist.
	//
	//go:embed testdata/mozilla-firefox.yml
	_ []byte
	//go:embed testdata/team.yml
	_ []byte
	//go:embed testdata/safari.yml
	_ []byte
)

const (
	dirNameTestdata = "testdata"
	fileNameFirefox = "mozilla-firefox.yml"
	fileNameSafari  = "safari.yml"
	fileNameTeam    = "team.yml"
)

func TestGitopsMigrate(t *testing.T) {
	// Create a test temp directory.
	testDir := t.TempDir()

	// Write the test files to the test directory.
	testdataSub, err := fs.Sub(testdata, dirNameTestdata)
	require.NoError(t, err)
	require.NoError(t, os.CopyFS(testDir, testdataSub))

	// Validate the expected state of the test files before we begin.
	//
	// This expects one of _each_ of the things we're migrating to be present.
	gitopsMigratePre(t, testDir)

	t.Logf("using temp directory: %s", testDir)

	// Perform the migration.
	gitopsMigrate(t, testDir)

	// Validate the expected state of the test files when we're finished.
	gitopsMigratePost(t, testDir)
}

const (
	sha256HashEmpty = "01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b"
	keyHashSHA256   = "hash_sha256"
	keyURL          = "url"
	defaultURL      = "https://fleetdm.com"
)

// Validates expectations of the GitOps YAML files _before_ this test runs
// the migration.
func gitopsMigratePre(t *testing.T, testDir string) {
	t.Helper()

	t.Run("pre-validate-"+fileNameFirefox, func(t *testing.T) {
		// Read the file.
		content, err := os.ReadFile(filepath.Join(
			testDir, fileNameFirefox,
		))
		require.NoError(t, err)
		require.NotEmpty(t, content)

		// Unmarshal the file.
		firefoxMap := make(map[string]any)
		require.NoError(t, yaml.Unmarshal(content, &firefoxMap))

		// Ensure all of the keys we will migrate are present.
		require.Contains(t, firefoxMap, keySelfService)
		require.Contains(t, firefoxMap, keyCategories)
		require.Contains(t, firefoxMap, keyLabelsExclude)
		require.Contains(t, firefoxMap, keyLabelsInclude)
	})

	t.Run("pre-validate-"+fileNameSafari, func(t *testing.T) {
		// Read the file.
		content, err := os.ReadFile(filepath.Join(
			testDir, fileNameSafari,
		))
		require.NoError(t, err)
		require.NotEmpty(t, content)

		// Unmarshal the file.
		safariMap := make(map[string]any)
		require.NoError(t, yaml.Unmarshal(content, &safariMap))

		// Ensure all of the keys we will migrate are present.
		require.NotContains(t, safariMap, keySelfService)
		require.NotContains(t, safariMap, keyCategories)
		require.NotContains(t, safariMap, keyLabelsExclude)
		require.NotContains(t, safariMap, keyLabelsInclude)
	})

	t.Run("pre-validate-"+fileNameTeam, func(t *testing.T) {
		// Read the file.
		content, err := os.ReadFile(filepath.Join(
			testDir, fileNameTeam,
		))
		require.NoError(t, err)
		require.NotEmpty(t, content)

		// Unmarshal it.
		team := make(map[string]any)
		require.NoError(t, yaml.Unmarshal(content, &team))

		t.Run("controls", func(t *testing.T) {
			t.SkipNow() // NOTE(max): The below works but was de-scoped.

			// Grab the 'controls' key in the map, ensure it's a non-nil
			// 'map[string]any'.
			require.Contains(t, team, keyControls)
			controls, ok := team[keyControls].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, controls)

			// Retrieve the 'macos_setup' key, assert as 'map[string]any'.
			require.Contains(t, controls, keyMacosSetup)
			macosSetup, ok := controls[keyMacosSetup].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, macosSetup)

			// Retrieve the 'software' key, assert as 'map[string]any'.
			require.Contains(t, macosSetup, keySoftware)
			software, ok := macosSetup[keySoftware].([]any)
			require.True(t, ok)
			require.NotNil(t, software)

			// Expect a single array item.
			require.Len(t, software, 1)
		})

		t.Run("software", func(t *testing.T) {
			// Grab the 'software' key in the map, ensure it's a non-nil
			// 'map[string]any'.
			require.Contains(t, team, keySoftware)
			software, ok := team[keySoftware].(map[any]any)
			require.True(t, ok, "%#v", team)
			require.NotNil(t, software)

			// Grab the 'packages' key in the software map, ensure it's a non-nil
			// '[]any'.
			require.Contains(t, software, keyPackages)
			packages, ok := software[keyPackages].([]any)
			require.True(t, ok)
			require.NotNil(t, packages)

			// Expect three packages.
			require.Len(t, packages, 3)

			// For the first package: expect only a 'path' key pointing to the firefox
			// software package file.
			pkg, ok := packages[0].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, pkg)
			require.Len(t, pkg, 1)
			require.Contains(t, pkg, keyPath)
			require.Equal(t, pkg[keyPath], fileNameFirefox)

			// For the second package: expect a 'hash_sha256' key with the empty
			// SHA256 hash value.
			pkg2, ok := packages[1].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, pkg2)
			require.Len(t, pkg2, 1)
			require.Contains(t, pkg2, keyHashSHA256)
			require.Equal(t, sha256HashEmpty, pkg2[keyHashSHA256])

			// For the third package: expect a only a 'path' key with the Safari
			// software file path.
			pkg3, ok := packages[2].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, pkg3)
			require.Len(t, pkg3, 1)
			require.Contains(t, pkg3, keyPath)
			require.Equal(t, fileNameSafari, pkg3[keyPath])
		})
	})
}

func gitopsMigrate(t *testing.T, testDir string) {
	require.NoError(t, cmdMigrateExec(t.Context(), Args{
		Commands: []string{testDir},
	}))
}

func gitopsMigratePost(t *testing.T, testDir string) {
	t.Helper()

	t.Run("post-validate-"+fileNameFirefox, func(t *testing.T) {
		// Read the file.
		content, err := os.ReadFile(filepath.Join(
			testDir, fileNameFirefox,
		))
		require.NoError(t, err)
		require.NotEmpty(t, content)

		// Unmarshal the file.
		swmap := make(map[string]any)
		require.NoError(t, yaml.Unmarshal(content, &swmap))

		// Expect only a single remaining key ('url').
		require.Len(t, swmap, 1)
		require.Contains(t, swmap, keyURL)
		require.Equal(t, swmap[keyURL], defaultURL)
	})

	t.Run("post-validate-"+fileNameSafari, func(t *testing.T) {
		// Read the file.
		content, err := os.ReadFile(filepath.Join(
			testDir, fileNameSafari,
		))
		require.NoError(t, err)
		require.NotEmpty(t, content)

		// Unmarshal the file.
		swmap := make(map[string]any)
		require.NoError(t, yaml.Unmarshal(content, &swmap))

		// This file should still only contain the single line it had before.
		require.Len(t, swmap, 1)
		require.Contains(t, swmap, keyURL)
		require.Equal(t, swmap[keyURL], defaultURL)
	})

	t.Run("post-validate-"+fileNameTeam, func(t *testing.T) {
		// Read the file.
		content, err := os.ReadFile(filepath.Join(
			testDir, fileNameTeam,
		))
		require.NoError(t, err)

		// Unmarshal it.
		team := make(map[string]any)
		require.NoError(t, yaml.Unmarshal(content, &team))

		t.Run("controls", func(t *testing.T) {
			t.SkipNow() // NOTE(max): The below works but was de-scoped.

			// Retrieve the 'controls' key, assert as 'map[any]any'.
			controls, ok := team[keyControls].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, controls)

			// Retrieve the 'macos_setup' key, assert as 'map[any]any'.
			require.Contains(t, controls, keyMacosSetup)
			macosSetup, ok := controls[keyMacosSetup].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, macosSetup)

			// Retrieve the 'software' key, assert as '[]any'.
			require.Contains(t, macosSetup, keySoftware)
			software, ok := macosSetup[keySoftware].([]any)
			require.True(t, ok)
			require.Empty(t, software)
		})

		t.Run("software", func(t *testing.T) {
			// Grab the 'software' key in the map, ensure it's a non-nil 'map[string]any'.
			require.Contains(t, team, keySoftware)
			software, ok := team[keySoftware].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, software)

			// Grab the 'packages' key in the software map, ensure it's a non-nil '[]any'.
			require.Contains(t, software, keyPackages)
			packages, ok := software[keyPackages].([]any)
			require.True(t, ok)
			require.NotNil(t, packages)

			// Expect three packages.
			require.Len(t, packages, 3)

			// For the first package: expect _all_ keys that can possibly be moved in this
			// migration.
			pkg, ok := packages[0].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, pkg)
			require.Contains(t, pkg, keySelfService)
			require.Contains(t, pkg, keyCategories)
			require.Contains(t, pkg, keyLabelsExclude)
			require.Contains(t, pkg, keyLabelsInclude)

			// For the second package: ensure it still holds only the single 'hash_sha256'
			// key with the same value as before the migration.
			pkg2, ok := packages[1].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, pkg2)
			require.Len(t, pkg2, 1)
			require.Contains(t, pkg2, keyHashSHA256)
			require.Equal(t, sha256HashEmpty, pkg2[keyHashSHA256])

			// For the third package: expect no changes (only a 'path' key present).
			pkg3, ok := packages[2].(map[any]any)
			require.True(t, ok)
			require.NotNil(t, pkg3)
			require.Len(t, pkg3, 1)
			require.Contains(t, pkg3, keyPath)
			require.Equal(t, pkg3[keyPath], fileNameSafari)
		})
	})

	t.Run("post-validate-no-mutation-"+fileNameSafari, func(t *testing.T) {
		// Since no relevent fields for migration exist in the safari software
		// package file, this file should be unchanged therefore the comments should
		// still be there.

		// Read the copy we wrote to disk (and would have transformed).
		testFilePath := filepath.Join(testDir, fileNameSafari)
		testContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)

		// Read the original copy from the embedded FS.
		originalFilePath := filepath.Join(dirNameTestdata, fileNameSafari)
		originalContent, err := testdata.ReadFile(originalFilePath)
		require.NoError(t, err)

		// These should still be the same!
		require.Equal(
			t, originalContent, testContent,
			"%s\n---\n%s\n",
			originalContent, testContent,
		)
	})
}

func TestResolvePackagePath(t *testing.T) {
	root, err := os.Getwd()
	require.NoError(t, err)

	// Standard case.
	teamPath := "gitops/teams/team.yml"
	pkgPath := "../software/firefox.yml"
	path, err := resolvePackagePath(teamPath, pkgPath)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "gitops", "software", "firefox.yml"), path)

	// 'teamPath' is not a file path.
	teamPath = "gitops/teams"
	pkgPath = "../software/firefox.yml"
	path, err = resolvePackagePath(teamPath, pkgPath)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "gitops", "software", "firefox.yml"), path)
}
