package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

var rawJSONFlags = json.RawMessage(`{"verbose":true, "num":5, "hello":"world", "largeNum":1234567890}`)

func TestGetFlagsFromJson(t *testing.T) {
	flagsJson, err := getFlagsFromJSON(rawJSONFlags)
	require.NoError(t, err)

	require.NotEmpty(t, flagsJson)

	value, ok := flagsJson["--verbose"]
	if !ok {
		t.Errorf(`key ""--verbose" expected but not found`)
	}
	if value != "true" {
		t.Errorf(`expected "true", got %s`, value)
	}

	value, ok = flagsJson["--num"]
	if !ok {
		t.Errorf(`key "--num" expected but not found`)
	}
	if value != "5" {
		t.Errorf(`expected "5", got %s`, value)
	}

	value, ok = flagsJson["--hello"]
	if !ok {
		t.Errorf(`key "--hello" expected but not found`)
	}
	if value != "world" {
		t.Errorf(`expected "world", got %s`, value)
	}

	value, ok = flagsJson["--largeNum"]
	if !ok {
		t.Errorf(`key "--largeNum" expected but not found`)
	}
	if value != "1234567890" {
		t.Errorf(`expected "1234567890", got %s`, value)
	}
}

func TestWriteFlagFile(t *testing.T) {
	flags, err := getFlagsFromJSON(rawJSONFlags)
	require.NoError(t, err)

	tempDir := t.TempDir()
	err = writeFlagFile(tempDir, flags)
	require.NoError(t, err)

	diskFlags, err := readFlagFile(tempDir)
	require.NoError(t, err)
	require.NotEmpty(t, diskFlags)

	if !reflect.DeepEqual(flags, diskFlags) {
		t.Errorf("expected flags to be equal: %v, %v", flags, diskFlags)
	}
}

func touchFile(t *testing.T, name string) {
	t.Helper()

	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0o644) // nolint:gosec // G302
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

// TestDoFlagsUpdateWithEmptyFlags tests the scenario of Fleet flag `command_line_flags`
// being set to an empty JSON document `{}`. Setting it to `{}` is an explicit instruction
// to clear osquery flags, so Orbit reconciles the osquery.flags file to empty (distinct
// from command_line_flags being unset, which is covered by TestDoFlagsUpdateWithNilFlags).
func TestDoFlagsUpdateWithEmptyFlags(t *testing.T) {
	rootDir := t.TempDir()
	osqueryFlagsFile := filepath.Join(rootDir, "osquery.flags")
	touchFile(t, osqueryFlagsFile)

	testConfig := &fleet.OrbitConfig{
		Flags: json.RawMessage("{}"),
	}

	var restartQueued bool
	queueOrbitRestart := func(string) { restartQueued = true }

	fr := NewFlagReceiver(queueOrbitRestart, FlagUpdateOptions{
		RootDir: rootDir,
	})

	err := fr.Run(testConfig)
	require.NoError(t, err)
	require.False(t, restartQueued)

	// Non-empty fleet flags and osquery.flags has empty flags.
	testConfig = &fleet.OrbitConfig{
		Flags: json.RawMessage(`{"verbose": true}`),
	}
	err = fr.Run(testConfig)
	require.NoError(t, err)
	require.True(t, restartQueued)

	// Empty Fleet flags ({}) and osquery.flags has non-empty flags: the file is
	// cleared and a restart is triggered.
	restartQueued = false
	testConfig = &fleet.OrbitConfig{
		Flags: json.RawMessage("{}"),
	}
	err = os.WriteFile(osqueryFlagsFile, []byte("--verbose=true\n"), 0o644)
	require.NoError(t, err)
	err = fr.Run(testConfig)
	require.NoError(t, err)
	require.True(t, restartQueued)

	contents, err := os.ReadFile(osqueryFlagsFile)
	require.NoError(t, err)
	require.Empty(t, string(contents))
}

// TestDoFlagsUpdateWithNilFlags verifies that when the server is not managing
// osquery command-line flags (command_line_flags unset, so config.Flags is
// nil/empty), Orbit leaves the osquery.flags file untouched, preserving any
// pre-packaged or user-provided flags.
func TestDoFlagsUpdateWithNilFlags(t *testing.T) {
	rootDir := t.TempDir()
	osqueryFlagsFile := filepath.Join(rootDir, "osquery.flags")

	var restartQueued bool
	queueOrbitRestart := func(string) { restartQueued = true }
	fr := NewFlagReceiver(queueOrbitRestart, FlagUpdateOptions{RootDir: rootDir})

	// Nil Flags + no file: no-op.
	err := fr.Run(&fleet.OrbitConfig{Flags: nil})
	require.NoError(t, err)
	require.False(t, restartQueued)

	// Nil Flags + existing user-provided file: file is preserved, no restart.
	userFlags := "--verbose=true\n--tls_dump=true\n"
	err = os.WriteFile(osqueryFlagsFile, []byte(userFlags), 0o644)
	require.NoError(t, err)
	err = fr.Run(&fleet.OrbitConfig{Flags: nil})
	require.NoError(t, err)
	require.False(t, restartQueued)

	contents, err := os.ReadFile(osqueryFlagsFile)
	require.NoError(t, err)
	require.Equal(t, userFlags, string(contents))

	// Empty (but non-nil) Flags is treated the same as nil: file preserved.
	err = fr.Run(&fleet.OrbitConfig{Flags: json.RawMessage("")})
	require.NoError(t, err)
	require.False(t, restartQueued)

	contents, err = os.ReadFile(osqueryFlagsFile)
	require.NoError(t, err)
	require.Equal(t, userFlags, string(contents))
}

// TestDoFlagsUpdateWithNullFlags verifies that the JSON literal "null" is an
// explicit instruction to clear osquery flags, distinct from command_line_flags
// being unset (which preserves the file).
func TestDoFlagsUpdateWithNullFlags(t *testing.T) {
	rootDir := t.TempDir()
	osqueryFlagsFile := filepath.Join(rootDir, "osquery.flags")

	var restartQueued bool
	queueOrbitRestart := func(string) { restartQueued = true }
	fr := NewFlagReceiver(queueOrbitRestart, FlagUpdateOptions{RootDir: rootDir})

	// "null" + existing flags: the file is cleared and a restart is triggered.
	err := os.WriteFile(osqueryFlagsFile, []byte("--verbose=true\n"), 0o644)
	require.NoError(t, err)
	err = fr.Run(&fleet.OrbitConfig{Flags: json.RawMessage("null")})
	require.NoError(t, err)
	require.True(t, restartQueued)

	contents, err := os.ReadFile(osqueryFlagsFile)
	require.NoError(t, err)
	require.Empty(t, string(contents))
}
