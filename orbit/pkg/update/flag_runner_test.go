package update

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

var rawJSONFlags = json.RawMessage(`{"verbose":true, "num":5, "hello":"world"}`)

func TestGetFlagsFromJson(t *testing.T) {
	flagsJson, err := getFlagsFromJSON(rawJSONFlags)
	require.NoError(t, err)

	require.NotEmpty(t, flagsJson)

	value, ok := flagsJson["--verbose"]
	if !ok {
		t.Errorf("key \"--verbose\" expected but not found")
	}
	if value != "true" {
		t.Errorf("expected \"true\", got %s", value)
	}

	value, ok = flagsJson["--num"]
	if !ok {
		t.Errorf("key \"--num\" expected but not found")
	}
	if value != "5" {
		t.Errorf("expected \"5\", got %s", value)
	}

	value, ok = flagsJson["--hello"]
	if !ok {
		t.Errorf("key \"--hello\" expected but not found")
	}
	if value != "world" {
		t.Errorf("expected \"world\", got %s", value)
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
