package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSoftwareInstaller(t *testing.T) {
	ctx := context.Background()
	store := SetupTestSoftwareInstallerStore(t, "software-installers-unit-test", "prefix")

	// get a non-existing installer
	blob, length, err := store.Get(ctx, "no-such-installer")
	require.Error(t, err)
	require.Nil(t, blob)
	require.Zero(t, length)

	exists, err := store.Exists(ctx, "no-such-installer")
	require.NoError(t, err)
	require.False(t, exists)

	// store an installer
	b := make([]byte, 1024)
	_, err = rand.Read(b)
	require.NoError(t, err)

	h := sha256.New()
	_, err = h.Write(b)
	require.NoError(t, err)
	installerID := hex.EncodeToString(h.Sum(nil))

	err = store.Put(ctx, installerID, bytes.NewReader(b))
	require.NoError(t, err)

	// read it back, it should match
	rc, sz, err := store.Get(ctx, installerID)
	require.NoError(t, err)
	require.EqualValues(t, len(b), sz)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, b, got)
}
