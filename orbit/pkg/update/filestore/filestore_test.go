package filestore

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorePathError(t *testing.T) {
	t.Parallel()

	tmpDir, err := ioutil.TempDir("", "filestore-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "metadata.json"), constant.DefaultDirMode))

	store, err := New(filepath.Join(tmpDir, "metadata.json"))
	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestFileStore(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	require.NoError(t, os.Chmod(tmpDir, 0700))

	store, err := New(filepath.Join(tmpDir, "metadata.json"))
	require.NoError(t, err)

	expected := map[string]json.RawMessage{
		"test":      json.RawMessage("{}"),
		"test2":     json.RawMessage("{}"),
		"root.json": json.RawMessage(`[{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"0994148e5242118d1d6a9a397a3646e0423545a37794a791c28aa39de3b0c523"}}]`),
	}

	for k, v := range expected {
		require.NoError(t, store.SetMeta(k, v))
	}

	res, err := store.GetMeta()
	require.NoError(t, err)
	assert.Equal(t, expected, res)

	// Reopen and check
	store, err = New(filepath.Join(tmpDir, "metadata.json"))
	require.NoError(t, err)
	res, err = store.GetMeta()
	require.NoError(t, err)
	assert.Equal(t, expected, res)

	// Update and check
	expected["test"] = json.RawMessage("[]")
	require.NoError(t, store.SetMeta("test", expected["test"]))
	res, err = store.GetMeta()
	require.NoError(t, err)
	assert.Equal(t, expected, res)
}
