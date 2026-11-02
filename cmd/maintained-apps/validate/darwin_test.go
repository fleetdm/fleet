//go:build darwin

package main

import (
	"context"
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
