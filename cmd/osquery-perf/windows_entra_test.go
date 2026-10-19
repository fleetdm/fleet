package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const windowsEntraJWKSFile = "windows_entra_loadtest_jwks.json"

// The committed JWK set is what load-test Fleet servers fetch, so it must stay the public half of the embedded key.
// Regenerate it with UPDATE_WINDOWS_ENTRA_JWKS=1 go test -run TestWindowsEntraJWKSMatchesKey ./cmd/osquery-perf/
func TestWindowsEntraJWKSMatchesKey(t *testing.T) {
	ctx := context.Background()
	key, err := windowsEntraSigningKey()
	require.NoError(t, err)
	want, err := windowsEntraJWKS(ctx, key)
	require.NoError(t, err)

	if os.Getenv("UPDATE_WINDOWS_ENTRA_JWKS") == "1" {
		require.NoError(t, os.WriteFile(windowsEntraJWKSFile, append(want, '\n'), 0o644))
	}

	got, err := os.ReadFile(windowsEntraJWKSFile)
	require.NoError(t, err)
	var wantSet, gotSet any
	require.NoError(t, json.Unmarshal(want, &wantSet))
	require.NoError(t, json.Unmarshal(got, &gotSet))
	require.Equal(t, wantSet, gotSet)
}
