package packaging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteOsqueryFlagfile_OverwritesExistingFlags reproduces
// https://github.com/fleetdm/fleet/issues/50720
//
// When a fleetd package is built WITHOUT --osquery-flagfile, the function
// writes an empty osquery.flags into the package payload.  Installing that
// package over an existing install silently replaces whatever flags the
// operator had previously configured.
func TestWriteOsqueryFlagfile_OverwritesExistingFlags(t *testing.T) {
	orbitRoot := t.TempDir()
	flagfilePath := filepath.Join(orbitRoot, "osquery.flags")

	// ── Step 1: simulate an initial install that shipped real flags ──
	// (equivalent to: fleetctl package --osquery-flagfile flagfile.txt)
	srcFlagfile := filepath.Join(t.TempDir(), "flagfile.txt")
	originalContent := "--verbose=true\n--extensions_autoload=/opt/ext/autoload.conf\n"
	require.NoError(t, os.WriteFile(srcFlagfile, []byte(originalContent), 0o644))

	err := writeOsqueryFlagfile(Options{OsqueryFlagfile: srcFlagfile}, orbitRoot)
	require.NoError(t, err)

	got, err := os.ReadFile(flagfilePath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(got),
		"after first install the flagfile should contain the operator's flags")

	// ── Step 2: simulate a second install built WITHOUT --osquery-flagfile ──
	// This is the bug: the empty-flagfile branch overwrites the existing file.
	err = writeOsqueryFlagfile(Options{OsqueryFlagfile: ""}, orbitRoot)
	require.NoError(t, err)

	got, err = os.ReadFile(flagfilePath)
	require.NoError(t, err)

	assert.Equal(t, originalContent, string(got),
		"osquery.flags must not be overwritten when no flagfile is supplied")
}
