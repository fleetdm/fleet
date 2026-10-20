package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOutputVersion(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}

	got, err := readOutputVersion(filepath.Join(dir, "missing.json"))
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = readOutputVersion(write("empty.json", ""))
	require.NoError(t, err)
	assert.Nil(t, got)

	// Only the header matters; the rest of the document isn't decoded.
	got, err = readOutputVersion(write("valid.json", `{"format_version": 1, "version": {"apple": {"release": "Release-v27.0", "commit": "`+testSHA+`"},
		"generated_at": "2026-10-15T00:00:00Z"}, "apple_profiles": "not checked here"}`))
	require.NoError(t, err)
	assert.Equal(t, &profilevalidation.Version{
		Apple:       profilevalidation.AppleVersion{Release: "Release-v27.0", Commit: testSHA},
		GeneratedAt: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
	}, got)

	_, err = readOutputVersion(write("bad.json", "not json"))
	require.ErrorContains(t, err, "read version from")
}
