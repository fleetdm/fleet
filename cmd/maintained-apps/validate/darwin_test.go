//go:build darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

func TestAppExistsPatchedQueryGate(t *testing.T) {
	originalRunOsqueryi := runOsqueryi
	t.Cleanup(func() { runOsqueryi = originalRunOsqueryi })

	const patchedQuery = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.example.app' AND version_compare(bundle_short_version, '6.7.0.3') < 0);"
	row := func(short, bundle string) string {
		return fmt.Sprintf(`{"name":"Example","path":"/Applications/Example.app","bundle_short_version":%q,"bundle_version":%q}`, short, bundle)
	}
	const noRows = "[\n\n]\n"

	testCases := []struct {
		name        string
		bundleID    string
		version     string
		rows        string
		patchedRows string
		patchedErr  error
		wantExists  bool
		wantPatched bool
		wantErr     string
	}{
		{
			name:       "short version matches",
			rows:       row("6.7.0.3", ""),
			wantExists: true,
		},
		{
			name:        "shorter short version, patched query fails",
			rows:        row("6.7.0", "6.7.0"),
			patchedRows: noRows,
			wantPatched: true,
			wantErr:     "bundle_short_version '6.7.0', a shorter form of '6.7.0.3'",
		},
		{
			name:        "shorter short version, patched query compares bundle_version",
			rows:        row("6.7.0", "270352"),
			patchedRows: `[{"1":"1"}]`,
			wantExists:  true,
			wantPatched: true,
		},
		{
			name:        "concatenated version, patched query fails",
			rows:        row("6.7", "0.3"),
			patchedRows: noRows,
			wantPatched: true,
			wantErr:     "a shorter form of '6.7.0.3'",
		},
		{
			name:        "bundle version matches, short version empty",
			rows:        row("", "6.7.0.3"),
			patchedRows: noRows,
			wantPatched: true,
			wantErr:     "bundle_short_version '', bundle_version '6.7.0.3'",
		},
		{
			name:       "stale shorter copy listed before the exact one",
			rows:       row("6.7.0", "6.7.0") + "," + row("6.7.0.3", ""),
			wantExists: true,
		},
		{
			name:        "patched query fails to run",
			rows:        row("6.7.0", "6.7.0"),
			patchedErr:  errors.New("exit status 1: syntax error"),
			wantPatched: true,
			wantErr:     "executing patched query",
		},
		{
			// MutateSoftwareOnIngestion strips the "v", but the patched query
			// compares the raw column.
			name:       "short version matches before sanitizing",
			bundleID:   "com.raspberrypi.rpi-imager",
			version:    "v6.7.0.3",
			rows:       row("v6.7.0.3", "v6.7.0.3"),
			wantExists: true,
		},
		{
			name: "no match",
			rows: row("6.6.2", "6.6.2"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ranPatched bool
			runOsqueryi = func(_ context.Context, query string) ([]byte, error) {
				if query == patchedQuery {
					ranPatched = true
					return []byte(tc.patchedRows), tc.patchedErr
				}
				return []byte("[" + tc.rows + "]"), nil
			}

			bundleID, version := "com.example.app", "6.7.0.3"
			if tc.bundleID != "" {
				bundleID, version = tc.bundleID, tc.version
			}
			exists, _, err := appExists(t.Context(), slog.New(slog.DiscardHandler), "Example", bundleID, version, patchedQuery, "", "")
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantExists, exists)
			require.Equal(t, tc.wantPatched, ranPatched)
		})
	}
}
