package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realAppleDownload returns a copy of the real Apple download, skipping when there isn't one: it
// exists only after running the tool with -update (or -fetch-pinned), not on a fresh checkout.
func realAppleDownload(t *testing.T) string {
	t.Helper()
	src := filepath.Join("downloads", "apple")
	if _, err := os.Stat(filepath.Join(src, commitMarker)); err != nil {
		t.Skip("no Apple download; run go run ./tools/profile-schema-gen -update from the repo root first")
	}
	dst := t.TempDir()
	require.NoError(t, os.CopyFS(dst, os.DirFS(src)))
	return dst
}

func TestRealDataHalfRemovedFailsAndWritesNothing(t *testing.T) {
	dir := realAppleDownload(t)
	logger := slog.New(slog.DiscardHandler)

	// Remove every other profile YAML, keeping the envelope files so loading itself succeeds.
	profiles, err := filepath.Glob(filepath.Join(dir, appleProfilesDir, "*.yaml"))
	require.NoError(t, err)
	require.Greater(t, len(profiles), 100)
	for i, p := range profiles {
		base := filepath.Base(p)
		if i%2 == 0 && base != appleTopLevel+".yaml" && base != appleCommonPayloadKeys+".yaml" {
			require.NoError(t, os.Remove(p))
		}
	}

	src, err := loadApple(t.Context(), logger, dir)
	require.NoError(t, err)
	var schema profilevalidation.ProfileValidationSchema
	require.NoError(t, convertApple(t.Context(), logger, src, &schema))

	out := filepath.Join(t.TempDir(), "profile_schema.json")
	require.NoError(t, os.WriteFile(out, []byte("previous output"), 0o600))

	err = writeOutput(t.Context(), logger, out, &schema, nil, time.Now())
	require.ErrorContains(t, err, "generated schema fails its own checks")
	require.ErrorContains(t, err, "payload types, want at least 100")
	after, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "previous output", string(after), "existing output left untouched")
}
