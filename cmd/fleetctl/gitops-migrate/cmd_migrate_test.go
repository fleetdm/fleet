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

// Validates expectations of the GitOps YAML files _before_ this test runs
// the migration.
func gitopsMigratePre(t *testing.T, testDir string) {
	t.Helper()

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
	// Ensure the following keys are _all_ present in the map.
	// - self_service
	// - categories
	// - labels_exclude_any
	// - labels_include_any
	// - setup_experience
	require.Contains(t, swmap, keySelfService)
	require.Contains(t, swmap, keyCategories)
	require.Contains(t, swmap, keyLabelsExclude)
	require.Contains(t, swmap, keyLabelsInclude)
	require.Contains(t, swmap, keySetupExperience)

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
	// Expect a single package, ensure it's a non-nil 'map[string]any'.
	require.Len(t, packages, 1)
	pkg, ok := packages[0].(map[any]any)
	require.True(t, ok)
	require.NotNil(t, pkg)
	// Ensure the following keys are _not_ present in the map.
	// - self_service
	// - categories
	// - labels_exclude_any
	// - labels_include_any
	// - setup_experience
	require.NotContains(t, pkg, keySelfService)
	require.NotContains(t, pkg, keyCategories)
	require.NotContains(t, pkg, keyLabelsExclude)
	require.NotContains(t, pkg, keyLabelsInclude)
	require.NotContains(t, pkg, keySetupExperience)
}

func gitopsMigrate(t *testing.T, testDir string) {
	cmdMigrateExec(t.Context(), Args{
		Commands: []string{testDir},
	})
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
	// Ensure the following keys are _not_ present in the map.
	// - self_service
	// - categories
	// - labels_exclude_any
	// - labels_include_any
	// - setup_experience
	require.NotContains(t, swmap, keySelfService)
	require.NotContains(t, swmap, keyCategories)
	require.NotContains(t, swmap, keyLabelsExclude)
	require.NotContains(t, swmap, keyLabelsInclude)
	require.NotContains(t, swmap, keySetupExperience)

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
	// Expect a single package, ensure it's a non-nil 'map[string]any'.
	require.Len(t, packages, 1)
	pkg, ok := packages[0].(map[any]any)
	require.True(t, ok)
	require.NotNil(t, pkg)
	// Ensure the following keys are _all_ present in the map.
	// - self_service
	// - categories
	// - labels_exclude_any
	// - labels_include_any
	// - setup_experience
	require.Contains(t, pkg, keySelfService)
	require.Contains(t, pkg, keyCategories)
	require.Contains(t, pkg, keyLabelsExclude)
	require.Contains(t, pkg, keyLabelsInclude)
	require.Contains(t, pkg, keySetupExperience)
}
