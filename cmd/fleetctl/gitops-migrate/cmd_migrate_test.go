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
)

const (
	swFileName   = "mozilla-firefox.yml"
	teamFileName = "team.yml"
)

func TestGitopsMigrate(t *testing.T) {
	// Create a test temp directory.
	testDir := t.TempDir()

	// Write the test files to the test directory.
	testdataSub, err := fs.Sub(testdata, "testdata")
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

const sha256HashEmpty = "01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b"

// Validates expectations of the GitOps YAML files _before_ this test runs
// the migration.
func gitopsMigratePre(t *testing.T, testDir string) {
	t.Helper()

	// Validate expectations for the software file.
	//
	// Read the file.
	content, err := os.ReadFile(filepath.Join(
		testDir, swFileName,
	))
	require.NoError(t, err)
	require.NotEmpty(t, content)
	// Unmarshal the file.
	swmap := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(content, &swmap))
	// Ensure all of the keys we will migrate are present.
	require.Contains(t, swmap, keySelfService)
	require.Contains(t, swmap, keyCategories)
	require.Contains(t, swmap, keyLabelsExclude)
	require.Contains(t, swmap, keyLabelsInclude)

	// Validate expectations for the team file.
	//
	// Read the file.
	content, err = os.ReadFile(filepath.Join(
		testDir, teamFileName,
	))
	require.NoError(t, err)
	// Unmarshal it.
	team := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(content, &team))
	// Grab the 'software' key in the map, ensure it's a non-nil 'map[string]any'.
	require.Contains(t, team, keySoftware)
	software, ok := team[keySoftware].(map[any]any)
	require.True(t, ok, "%#v", team)
	require.NotNil(t, software)
	// Grab the 'packages' key in the software map, ensure it's a non-nil '[]any'.
	require.Contains(t, software, keyPackages)
	packages, ok := software[keyPackages].([]any)
	require.True(t, ok)
	require.NotNil(t, packages)

	// Expect two packages.
	//
	// For the first: ensure it's a non-nil 'map[string]any'.
	require.Len(t, packages, 2)
	pkg, ok := packages[0].(map[any]any)
	require.True(t, ok)
	require.NotNil(t, pkg)
	// Ensure none of the keys we will migrate are present.
	require.NotContains(t, pkg, keySelfService)
	require.NotContains(t, pkg, keyCategories)
	require.NotContains(t, pkg, keyLabelsExclude)
	require.NotContains(t, pkg, keyLabelsInclude)
	// For the second: expect a 'hash_sha256' key with the empty SHA256 hash
	// value.
	pkg2, ok := packages[1].(map[any]any)
	require.True(t, ok)
	require.NotNil(t, pkg2)
	require.Len(t, pkg2, 1)
	require.Contains(t, pkg2, "hash_sha256")
	require.Equal(t, sha256HashEmpty, pkg2["hash_sha256"])
}

func gitopsMigrate(t *testing.T, testDir string) {
	require.NoError(t, cmdMigrateExec(t.Context(), Args{
		Commands: []string{testDir},
	}))
}

func gitopsMigratePost(t *testing.T, testDir string) {
	t.Helper()

	// Validate expectations for the software file.
	//
	// Read the file.
	content, err := os.ReadFile(filepath.Join(
		testDir, swFileName,
	))
	require.NoError(t, err)
	require.NotEmpty(t, content)
	// Unmarshal the file.
	swmap := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(content, &swmap))
	// Ensure none of the keys we migrated are present.
	require.NotContains(t, swmap, keySelfService)
	require.NotContains(t, swmap, keyCategories)
	require.NotContains(t, swmap, keyLabelsExclude)
	require.NotContains(t, swmap, keyLabelsInclude)

	// Validate expectations for the team file.
	//
	// Read the file.
	content, err = os.ReadFile(filepath.Join(
		testDir, teamFileName,
	))
	require.NoError(t, err)
	// Unmarshal it.
	team := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(content, &team))
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

	// Expect two packages.
	//
	// For the first: ensure it's a non-nil 'map[any]any'.
	require.Len(t, packages, 2)
	pkg, ok := packages[0].(map[any]any)
	require.True(t, ok)
	require.NotNil(t, pkg)
	// Ensure all appropriate keys are now present.
	require.Contains(t, pkg, keySelfService)
	require.Contains(t, pkg, keyCategories)
	require.Contains(t, pkg, keyLabelsExclude)
	require.Contains(t, pkg, keyLabelsInclude)
	// For the second: ensure it's a non-nil 'map[string]any'.
	pkg2, ok := packages[1].(map[any]any)
	require.True(t, ok)
	require.NotNil(t, pkg2)
	// Ensure this package is unchanged (just a 'hash_sha256' key with the empty
	// sha256 hash value).
	require.Len(t, pkg2, 1)
	require.Contains(t, pkg2, "hash_sha256")
	require.Equal(t, sha256HashEmpty, pkg2["hash_sha256"])
}
