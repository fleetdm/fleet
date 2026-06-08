//go:build darwin || windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListDirectoryContents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestDirectoryContents-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// directories
	err = os.Mkdir(filepath.Join(tmpDir, "app1"), 0o755)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmpDir, "app2"), 0o755)
	require.NoError(t, err)

	// file
	err = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0o644)
	require.NoError(t, err)

	// symlinks
	err = os.Symlink(filepath.Join(tmpDir, "app1"), filepath.Join(tmpDir, "symlinkdir"))
	require.NoError(t, err)
	err = os.Symlink(filepath.Join(tmpDir, "file1.txt"), filepath.Join(tmpDir, "symlinkfile"))
	require.NoError(t, err)

	directoryList, err := listDirectoryContents(tmpDir)
	require.NoError(t, err)
	expectedList := map[string]struct{}{
		"app1": {},
		"app2": {},
	}
	require.Equal(t, expectedList, directoryList)
}

func TestDetectApplicationChange(t *testing.T) {
	// mock filepath.Join as it is operating system specific
	originalJoin := pathJoin
	defer func() { pathJoin = originalJoin }()
	pathJoin = func(parts ...string) string {
		return strings.Join(parts, "__")
	}

	directory := "__MadeupOS"

	testCases := []struct {
		name              string
		preList           map[string]struct{}
		postList          map[string]struct{}
		expectedPath      string
		expectedDirection bool
	}{
		{
			name:              "Item added, returns new item",
			preList:           map[string]struct{}{"App1": {}, "App2": {}},
			postList:          map[string]struct{}{"App1": {}, "App2": {}, "App3": {}},
			expectedPath:      "__MadeupOS__App3",
			expectedDirection: true,
		},
		{
			name:              "Item removed, returns removed item",
			preList:           map[string]struct{}{"App1": {}, "App2": {}},
			postList:          map[string]struct{}{"App1": {}},
			expectedPath:      "__MadeupOS__App2",
			expectedDirection: false,
		},
		{
			name:              "No changes, returns empty",
			preList:           map[string]struct{}{"App1": {}, "App2": {}},
			postList:          map[string]struct{}{"App1": {}, "App2": {}},
			expectedPath:      "",
			expectedDirection: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, direction := detectApplicationChange(
				directory,
				tc.preList,
				tc.postList,
			)
			if path != tc.expectedPath {
				t.Errorf("expected path `%s`, got `%s`", tc.expectedPath, path)
			}
			if direction != tc.expectedDirection {
				t.Errorf("expected direction `%t`, got `%t`", tc.expectedDirection, direction)
			}
		})
	}
}

func TestValidateSqlInput(t *testing.T) {
	testCases := []struct {
		input   string
		isValid bool
	}{
		{"", true},
		{"Amazon DCV", true},
		{"/Applications/DCV Viewer.app", true},
		{"C:\\Program Files\\TeamViewer", true},
		{"com.google.Chrome", true},
		{"' OR '1'='1", false},
		{"' OR 1=1--", false},
		{"'; DROP TABLE users; --", false},
		{"'; WAITFOR DELAY '00:00:05';--", false},
		{"1; DELETE FROM table;--", false},
		{"'; INSERT INTO users(username, password) VALUES('hacker','pass');--", false},
	}
	for _, tc := range testCases {
		t.Run("Validating input `"+tc.input+"`", func(t *testing.T) {
			got := validateSqlInput(tc.input)
			if got != nil && tc.isValid {
				t.Errorf("expected `%s` to be valid input, but got error", tc.input)
			} else if got == nil && !tc.isValid {
				t.Errorf("expected validation error for `%s`, but got no error", tc.input)
			}
		})
	}
}

func TestNormalizeArch(t *testing.T) {
	cases := map[string]string{
		"x64":     "amd64",
		"X64":     "amd64",
		"amd64":   "amd64",
		"x86_64":  "amd64",
		"x86":     "386",
		"386":     "386",
		"i386":    "386",
		"arm64":   "arm64",
		"ARM64":   "arm64",
		"aarch64": "arm64",
		" arm64 ": "arm64",
	}
	for in, want := range cases {
		require.Equal(t, want, normalizeArch(in), "normalizeArch(%q)", in)
	}
}

func TestInstallerArchMatchesHost(t *testing.T) {
	// An empty arch always matches (backward compatibility with manifests
	// generated before the installer_arch field existed).
	require.True(t, installerArchMatchesHost(""))
	require.True(t, installerArchMatchesHost("   "))

	// The host's own architecture matches regardless of spelling.
	host := normalizeArch(runtime.GOARCH)
	switch host {
	case "amd64":
		require.True(t, installerArchMatchesHost("x64"))
		require.True(t, installerArchMatchesHost("amd64"))
		require.False(t, installerArchMatchesHost("arm64"))
		require.False(t, installerArchMatchesHost("x86"))
	case "arm64":
		require.True(t, installerArchMatchesHost("arm64"))
		require.False(t, installerArchMatchesHost("x64"))
		require.False(t, installerArchMatchesHost("x86"))
	case "386":
		require.True(t, installerArchMatchesHost("x86"))
		require.False(t, installerArchMatchesHost("x64"))
		require.False(t, installerArchMatchesHost("arm64"))
	default:
		// Unknown host arch: only an exactly-matching token should pass.
		require.True(t, installerArchMatchesHost(runtime.GOARCH))
		require.False(t, installerArchMatchesHost("definitely-not-an-arch"))
	}
}
