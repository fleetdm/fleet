package filesystem

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func createIconAndHash(t *testing.T) ([]byte, string) {
	b := make([]byte, 1024)
	_, err := rand.Read(b)
	require.NoError(t, err)

	h := sha256.New()
	_, err = h.Write(b)
	require.NoError(t, err)
	installerID := hex.EncodeToString(h.Sum(nil))

	return b, installerID
}

func assertIconsOnDisk(t *testing.T, dir string, want []string) {
	dirEnts, err := os.ReadDir(filepath.Join(dir, softwareTitleIconsPrefix))
	require.NoError(t, err)
	got := make([]string, 0, len(dirEnts))
	for _, de := range dirEnts {
		if de.Type().IsRegular() {
			got = append(got, de.Name())
		}
	}
	require.ElementsMatch(t, want, got)
}

func TestSoftwareTitleIconStore(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	err := os.MkdirAll(filepath.Join(dir, softwareTitleIconsPrefix), 0o755)
	require.NoError(t, err)
	store, err := NewSoftwareTitleIconStore(dir)
	require.NoError(t, err)

	blob, length, err := store.Get(ctx, "non-existant-icon")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, blob)
	require.Zero(t, length)

	exists, err := store.Exists(ctx, "non-existant-icon")
	require.NoError(t, err)
	require.False(t, exists)

	i0, id0 := createIconAndHash(t)
	err = store.Put(ctx, id0, bytes.NewReader(i0))
	require.NoError(t, err)

	rc, sz, err := store.Get(ctx, id0)
	require.NoError(t, err)
	require.EqualValues(t, len(i0), sz)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, i0, got)

	exists, err = store.Exists(ctx, id0)
	require.NoError(t, err)
	require.True(t, exists)

	i1, id1 := createIconAndHash(t)
	err = store.Put(ctx, id1, bytes.NewReader(i1))
	require.NoError(t, err)

	n, err := store.Cleanup(ctx, []string{}, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.Equal(t, 0, n)
	assertIconsOnDisk(t, dir, []string{id0, id1})

	n, err = store.Cleanup(ctx, []string{id0}, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	assertIconsOnDisk(t, dir, []string{id0})

	_, err = store.Sign(ctx, id0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signing not supported for software title icons in filesystem store")
}
