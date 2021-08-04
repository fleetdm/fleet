package database

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabase(t *testing.T) {
	t.Parallel()

	tmpDir, err := ioutil.TempDir("", "orbit-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Open and write
	db, err := Open(tmpDir)
	require.NoError(t, err)

	err = db.Update(func(tx *badger.Txn) error {
		require.NoError(t, tx.Set([]byte("key"), []byte("value")))
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, db.Close())

	// Reopen and read
	db, err = Open(tmpDir)
	require.NoError(t, err)

	err = db.View(func(tx *badger.Txn) error {
		item, err := tx.Get([]byte("key"))
		require.NoError(t, err)
		err = item.Value(func(val []byte) error {
			assert.Equal(t, []byte("value"), val)
			return nil
		})
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func TestCompactionPanic(t *testing.T) {
	t.Parallel()

	tmpDir, err := ioutil.TempDir("", "orbit-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	db, err := Open(tmpDir)
	require.NoError(t, err)

	// Try to start the compaction routine again
	assert.Panics(t, func() { db.startBackgroundCompaction() })
}

func TestCompactionRestart(t *testing.T) {
	t.Parallel()

	tmpDir, err := ioutil.TempDir("", "orbit-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	db, err := Open(tmpDir)
	require.NoError(t, err)
	go func() {
		require.NoError(t, db.Close())
	}()

	db.stopBackgroundCompaction()

	assert.NotPanics(t, func() { db.startBackgroundCompaction() })
}
