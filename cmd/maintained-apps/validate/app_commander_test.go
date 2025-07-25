//go:build darwin || windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestExpectToChangeFileSystem(t *testing.T) {
	var ac AppCommander

	testCases := []struct {
		name     string
		before   func()
		testFunc func(*testing.T)
	}{
		{
			name:   "no changes",
			before: func() {},
			testFunc: func(t *testing.T) {
				appPath, changerError, listError := ac.expectToChangeFileSystem(
					func() error {
						return nil
					},
				)
				require.NoError(t, changerError)
				require.NoError(t, listError)
				require.Empty(t, appPath, "Expected no")
			},
		},
		{
			name:   "item added",
			before: func() {},
			testFunc: func(t *testing.T) {
				appPath, changerError, listError := ac.expectToChangeFileSystem(
					func() error {
						err := os.Mkdir(filepath.Join(ac.cfg.installationSearchDirectory, "app1"), 0o755)
						if err != nil {
							t.Fatalf("Failed to create directory: %v, test cannot properly run", err)
						}
						return nil
					},
				)
				require.NoError(t, changerError)
				require.NoError(t, listError)
				expectedPath := filepath.Join(ac.cfg.installationSearchDirectory, "app1")
				require.Equal(t, expectedPath, appPath, "Expected appPath to return path to new item")
			},
		},
		{
			name: "item removed",
			before: func() {
				err := os.Mkdir(filepath.Join(ac.cfg.installationSearchDirectory, "app1"), 0o755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v, test cannot properly run", err)
				}
				err = os.Mkdir(filepath.Join(ac.cfg.installationSearchDirectory, "app2"), 0o755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v, test cannot properly run", err)
				}
			},
			testFunc: func(t *testing.T) {
				appPath, changerError, listError := ac.expectToChangeFileSystem(
					func() error {
						err := os.Remove(filepath.Join(ac.cfg.installationSearchDirectory, "app2"))
						if err != nil {
							t.Fatalf("Failed to remove directory: %v, test cannot properly run", err)
						}
						return nil
					},
				)
				require.NoError(t, changerError)
				require.NoError(t, listError)
				expectedPath := filepath.Join(ac.cfg.installationSearchDirectory, "app2")
				require.Equal(t, expectedPath, appPath, "Expected appPath to return path to removed item")
			},
		},
		{
			name:   "error inside change function",
			before: func() {},
			testFunc: func(t *testing.T) {
				appPath, changerError, listError := ac.expectToChangeFileSystem(
					func() error {
						return errors.New("simulated error in change function")
					},
				)
				require.Error(t, changerError, "Expected an error from the change function")
				require.NoError(t, listError)
				require.Empty(t, appPath, "Expected no appPath due to error in change function")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// installationSearchDirectory is the variable that expectToChangeFileSystem uses
			installationSearchDirectory, err := os.MkdirTemp("", "TestExpectToChangeFileSystem-")
			require.NoError(t, err)
			defer os.RemoveAll(installationSearchDirectory)

			cfg := &Config{
				logger:                      kitlog.NewNopLogger(),
				installationSearchDirectory: installationSearchDirectory,
			}

			ac = AppCommander{cfg: cfg}
			tc.before()
			tc.testFunc(t)
		})
	}
}
