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

	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0o644)
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
