//go:build darwin

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssessGatekeeperBundleLookup(t *testing.T) {
	originalRunOsqueryi := runOsqueryi
	t.Cleanup(func() { runOsqueryi = originalRunOsqueryi })

	testCases := []struct {
		name             string
		bundleIdentifier string
		existsQuery      string
		wantQuery        string
		wantErr          string
	}{
		{
			name:             "exists query matches another bundle identifier",
			bundleIdentifier: "com.openai.chat",
			existsQuery:      "SELECT 1 FROM apps WHERE bundle_identifier = 'com.openai.chat' OR (bundle_identifier = 'com.openai.codex' AND bundle_name = 'ChatGPT');",
			wantQuery:        "SELECT path FROM apps WHERE bundle_identifier = 'com.openai.chat' OR (bundle_identifier = 'com.openai.codex' AND bundle_name = 'ChatGPT');",
		},
		{
			name:             "stale unique identifier",
			bundleIdentifier: "org.m0k.transmission.LSSharedFileList",
			existsQuery:      "SELECT 1 FROM apps WHERE bundle_identifier = 'org.m0k.transmission';",
			wantQuery:        "SELECT path FROM apps WHERE bundle_identifier = 'org.m0k.transmission';",
		},
		{
			name:             "other exists query shape falls back to the bundle identifier",
			bundleIdentifier: "com.example.foo",
			existsQuery:      "SELECT 1 WHERE EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.example.foo');",
			wantQuery:        "SELECT path FROM apps WHERE bundle_identifier = 'com.example.foo';",
		},
		{
			name:             "invalid bundle identifier in the fallback",
			bundleIdentifier: "com.example.foo'; DROP TABLE apps; --",
			existsQuery:      "",
			wantErr:          "Invalid character found in bundle identifier",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery string
			runOsqueryi = func(_ context.Context, query string) ([]byte, error) {
				gotQuery = query
				// No rows, so spctl isn't run.
				return []byte("[\n\n]\n"), nil
			}

			err := assessGatekeeper(t.Context(), slog.New(slog.DiscardHandler), tc.bundleIdentifier, tc.existsQuery)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Empty(t, gotQuery, "osqueryi shouldn't run")
				return
			}
			require.Equal(t, tc.wantQuery, gotQuery)
			require.ErrorContains(t, err, "Gatekeeper assessment found no app: "+tc.wantQuery)
		})
	}
}

func TestAppExistsLogsEveryRow(t *testing.T) {
	originalRunOsqueryi := runOsqueryi
	t.Cleanup(func() { runOsqueryi = originalRunOsqueryi })

	row := func(path, short, bundle string) string {
		return fmt.Sprintf(`{"name":"Example","path":%q,"bundle_short_version":%q,"bundle_version":%q}`, path, short, bundle)
	}
	const app, stale = "/Applications/Example.app", "/Library/Application Support/Example/Backups/Example.app"

	testCases := []struct {
		name       string
		bundleID   string
		version    string
		rows       []string
		wantExists bool
		wantLog    string
	}{
		{name: "short version matches", rows: []string{row(app, "6.7.0.3", "")}, wantExists: true},
		{name: "shorter short version", rows: []string{row(app, "6.7.0", "6.7.0")}, wantExists: true},
		{name: "only bundle version matches", rows: []string{row(app, "", "6.7.0.3")}, wantExists: true},
		{name: "older copy listed before the match", rows: []string{row(stale, "6.6.2", "6.6.2"), row(app, "6.7.0.3", "")}, wantExists: true},
		{name: "older copy listed after the match", rows: []string{row(app, "6.7.0.3", ""), row(stale, "6.6.2", "6.6.2")}, wantExists: true},
		{
			name:       "per-app exception doesn't stop the scan",
			bundleID:   "com.microsoft.OneDrive",
			version:    "26.198.1011",
			rows:       []string{row(app, "26.200.1", "26.200.1"), row(stale, "26.100.1", "26.100.1")},
			wantExists: true,
		},
		{
			name:       "version matches after sanitizing",
			bundleID:   "com.raspberrypi.rpi-imager",
			version:    "6.7.0.3",
			rows:       []string{row(app, "v6.7.0.3", "v6.7.0.3")},
			wantExists: true,
			// The patched query compares the raw value, so the log shows it.
			wantLog: "Version: 6.7.0.3 (osquery: v6.7.0.3)",
		},
		{name: "no match", rows: []string{row(app, "6.6.2", "6.6.2")}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var queries int
			runOsqueryi = func(context.Context, string) ([]byte, error) {
				queries++
				return []byte("[" + strings.Join(tc.rows, ",") + "]"), nil
			}

			bundleID, version := "com.example.app", "6.7.0.3"
			if tc.bundleID != "" {
				bundleID, version = tc.bundleID, tc.version
			}
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			exists, _, err := appExists(t.Context(), logger, "Example", bundleID, version, "SELECT 1;", "", "")
			require.NoError(t, err)
			require.Equal(t, tc.wantExists, exists)
			// The patched query is checked separately, after the exists query.
			require.Equal(t, 1, queries)
			require.Equal(t, len(tc.rows), strings.Count(logs.String(), "Found app"))
			if tc.wantLog != "" {
				require.Contains(t, logs.String(), tc.wantLog)
			} else {
				require.NotContains(t, logs.String(), "(osquery:")
			}
		})
	}
}

func TestAssessGatekeeperRetriesReplacedBundle(t *testing.T) {
	originalRunOsqueryi, originalRunSpctl, originalDelay := runOsqueryi, runSpctl, gatekeeperRetryDelay
	t.Cleanup(func() {
		runOsqueryi, runSpctl, gatekeeperRetryDelay = originalRunOsqueryi, originalRunSpctl, originalDelay
	})
	gatekeeperRetryDelay = 0
	const existsQuery = "SELECT 1 FROM apps WHERE bundle_identifier = 'com.microsoft.OneDrive';"
	const oneDrive = `[{"path":"/Applications/OneDrive.app"}]`
	assess := func(t *testing.T) error {
		return assessGatekeeper(t.Context(), slog.New(slog.DiscardHandler), "com.microsoft.OneDrive", existsQuery)
	}

	for _, replaced := range []string{
		"/Applications/OneDrive.app: No such file or directory",
		"/Applications/OneDrive.app: cannot find code object on disk",
	} {
		t.Run("passes once the replaced bundle settles: "+replaced, func(t *testing.T) {
			runOsqueryi = func(context.Context, string) ([]byte, error) { return []byte(oneDrive), nil }
			calls := 0
			runSpctl = func(context.Context, string) ([]byte, error) {
				calls++
				if calls == 1 {
					return []byte(replaced), errors.New("exit status 1")
				}
				return []byte("/Applications/OneDrive.app: accepted"), nil
			}
			require.NoError(t, assess(t))
			require.Equal(t, 2, calls)
		})
	}

	t.Run("keeps retrying while the new bundle isn't registered yet", func(t *testing.T) {
		lookups := 0
		runOsqueryi = func(context.Context, string) ([]byte, error) {
			lookups++
			if lookups == 2 {
				return []byte("[\n\n]\n"), nil
			}
			return []byte(oneDrive), nil
		}
		calls := 0
		runSpctl = func(context.Context, string) ([]byte, error) {
			calls++
			if calls == 1 {
				return []byte("/Applications/OneDrive.app: cannot find code object on disk"), errors.New("exit status 1")
			}
			return []byte("/Applications/OneDrive.app: accepted"), nil
		}
		require.NoError(t, assess(t))
		require.Equal(t, 3, lookups)
		require.Equal(t, 2, calls)
	})

	t.Run("gives up after the last attempt", func(t *testing.T) {
		runOsqueryi = func(context.Context, string) ([]byte, error) { return []byte(oneDrive), nil }
		calls := 0
		runSpctl = func(context.Context, string) ([]byte, error) {
			calls++
			return []byte("/Applications/OneDrive.app: No such file or directory"), errors.New("exit status 1")
		}
		require.ErrorContains(t, assess(t), "No such file or directory")
		require.Equal(t, gatekeeperAttempts, calls)
	})

	t.Run("doesn't retry a rejection", func(t *testing.T) {
		runOsqueryi = func(context.Context, string) ([]byte, error) { return []byte(oneDrive), nil }
		calls := 0
		runSpctl = func(context.Context, string) ([]byte, error) {
			calls++
			return []byte("/Applications/OneDrive.app: rejected"), errors.New("exit status 3")
		}
		require.ErrorContains(t, assess(t), "Gatekeeper rejected: /Applications/OneDrive.app: rejected")
		require.Equal(t, 1, calls)
	})

	t.Run("doesn't retry an app that was never found", func(t *testing.T) {
		lookups := 0
		runOsqueryi = func(context.Context, string) ([]byte, error) {
			lookups++
			return []byte("[\n\n]\n"), nil
		}
		require.ErrorContains(t, assess(t), "Gatekeeper assessment found no app")
		require.Equal(t, 1, lookups)
	})
}

func TestAssessGatekeeperSkipsNestedBundles(t *testing.T) {
	originalRunOsqueryi, originalRunSpctl := runOsqueryi, runSpctl
	t.Cleanup(func() { runOsqueryi, runSpctl = originalRunOsqueryi, originalRunSpctl })
	runOsqueryi = func(context.Context, string) ([]byte, error) {
		return []byte(`[{"path":"/Applications/Syncovery.app"},{"path":"/Applications/Syncovery.app/Contents/MacOS/BackgroundScheduler.app"},{"path":"/Applications/Syncovery Helper.app"}]`), nil
	}
	var assessed []string
	runSpctl = func(_ context.Context, path string) ([]byte, error) {
		assessed = append(assessed, path)
		return []byte(path + ": accepted"), nil
	}

	err := assessGatekeeper(t.Context(), slog.New(slog.DiscardHandler), "com.company.Syncovery", "SELECT 1 FROM apps WHERE bundle_identifier = 'com.company.Syncovery';")
	require.NoError(t, err)
	require.Equal(t, []string{"/Applications/Syncovery.app", "/Applications/Syncovery Helper.app"}, assessed)
}
