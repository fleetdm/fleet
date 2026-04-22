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
// being set to an empty JSON document `{}` and Orbit osquery.flags file being
// an empty file. Such scenario should trigger no update of flags.
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
		Flags: json.RawMessage(`{"--verbose": true}`),
	}
	err = fr.Run(testConfig)
	require.NoError(t, err)
	require.True(t, restartQueued)

	// Empty Fleet flags and osquery.flags has non-empty flags.
	restartQueued = false
	testConfig = &fleet.OrbitConfig{
		Flags: json.RawMessage("{}"),
	}
	err = os.WriteFile(osqueryFlagsFile, []byte("--verbose=true\n"), 0o644)
	require.NoError(t, err)
	err = fr.Run(testConfig)
	require.NoError(t, err)
	require.True(t, restartQueued)
}

// TestDoFlagsUpdateWithNilFlags covers the case where the server returns no
// Flags field at all (nil json.RawMessage). This is what the server does when
// the team has no agent-options command_line_flags set — which is the common
// case when orbit debug logging is toggled off: the merged verbose/tls_dump
// flags are gone but the admin never had any other flags, so the server sends
// nil. FlagRunner must still reconcile the on-disk osquery.flags so osqueryd
// drops its verbose behavior on next restart.
func TestDoFlagsUpdateWithNilFlags(t *testing.T) {
	rootDir := t.TempDir()
	osqueryFlagsFile := filepath.Join(rootDir, "osquery.flags")

	var restartQueued bool
	queueOrbitRestart := func(string) { restartQueued = true }
	fr := NewFlagReceiver(queueOrbitRestart, FlagUpdateOptions{RootDir: rootDir})

	// Nil Flags + no file on disk → no-op (preserves behavior for hosts that
	// never had agent-options flags).
	err := fr.Run(&fleet.OrbitConfig{Flags: nil})
	require.NoError(t, err)
	require.False(t, restartQueued)

	// Nil Flags + file on disk with content → reconcile to empty + restart.
	// This is the key case for orbit-debug-logging disable.
	err = os.WriteFile(osqueryFlagsFile, []byte("--verbose=true\n--tls_dump=true\n"), 0o644)
	require.NoError(t, err)
	err = fr.Run(&fleet.OrbitConfig{Flags: nil})
	require.NoError(t, err)
	require.True(t, restartQueued)

	// After the reconciliation, the file should be empty.
	contents, err := os.ReadFile(osqueryFlagsFile)
	require.NoError(t, err)
	require.Empty(t, string(contents))
}

// TestDoFlagsUpdateStartupDebugIsFloor verifies that when FlagRunner is
// constructed with StartedInDebug=true, --verbose and --tls_dump are always
// preserved in the osquery.flags file even when the server doesn't push them.
// This mirrors the DebugLogReceiver's startup-flag-is-floor semantic for
// orbit's zerolog level.
func TestDoFlagsUpdateStartupDebugIsFloor(t *testing.T) {
	rootDir := t.TempDir()
	osqueryFlagsFile := filepath.Join(rootDir, "osquery.flags")

	var restartQueued bool
	queueOrbitRestart := func(string) { restartQueued = true }
	fr := NewFlagReceiver(queueOrbitRestart, FlagUpdateOptions{
		RootDir:        rootDir,
		StartedInDebug: true,
	})

	// Server sends nil Flags + no file on disk. With the floor on, the
	// resulting file must contain verbose/tls_dump.
	err := fr.Run(&fleet.OrbitConfig{Flags: nil})
	require.NoError(t, err)
	require.True(t, restartQueued)

	diskFlags, err := readFlagFile(rootDir)
	require.NoError(t, err)
	require.Equal(t, "true", diskFlags["--verbose"])
	require.Equal(t, "true", diskFlags["--tls_dump"])

	// Server now pushes an unrelated flag — verbose/tls_dump still stay.
	restartQueued = false
	_ = os.Remove(osqueryFlagsFile)
	err = fr.Run(&fleet.OrbitConfig{
		Flags: json.RawMessage(`{"distributed_interval": 30}`),
	})
	require.NoError(t, err)
	require.True(t, restartQueued)

	diskFlags, err = readFlagFile(rootDir)
	require.NoError(t, err)
	require.Equal(t, "true", diskFlags["--verbose"])
	require.Equal(t, "true", diskFlags["--tls_dump"])
	require.Equal(t, "30", diskFlags["--distributed_interval"])

	// Admin explicitly sets verbose:false in command_line_flags. Their
	// choice wins over the startup floor (we only inject when not already
	// specified) — gives admins an escape hatch.
	restartQueued = false
	err = fr.Run(&fleet.OrbitConfig{
		Flags: json.RawMessage(`{"verbose": false}`),
	})
	require.NoError(t, err)
	require.True(t, restartQueued)

	diskFlags, err = readFlagFile(rootDir)
	require.NoError(t, err)
	require.Equal(t, "false", diskFlags["--verbose"])
	require.Equal(t, "true", diskFlags["--tls_dump"])
}
