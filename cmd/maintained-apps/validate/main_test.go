//go:build darwin || windows

package main

import (
	"os"
	"path/filepath"
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
