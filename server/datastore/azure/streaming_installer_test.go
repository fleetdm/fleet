package azure

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallerStreaming(t *testing.T) {
	ctx := t.Context()
	store := setupTestSoftwareInstallerStore(t, "software-installers-streaming-test")

	newContent := func(size int) []byte {
		b := make([]byte, size)
		_, err := rand.Read(b)
		require.NoError(t, err)
		return b
	}
	readInstaller := func(id string) []byte {
		rc, _, err := store.Get(ctx, id)
		require.NoError(t, err)
		defer rc.Close()
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		return b
	}

	// missing staging object
	_, err := store.StagingSize(ctx, "missing")
	require.True(t, fleet.IsNotFound(err))
	_, err = store.GetStagingRange(ctx, "missing", 0, 10)
	require.True(t, fleet.IsNotFound(err))
	require.NoError(t, store.DeleteStaging(ctx, "missing"))

	t.Run("stage, read ranges and promote", func(t *testing.T) {
		// Several copy blocks, the last one partial.
		oldBlockSize := copyBlockSize
		copyBlockSize = 3 << 20
		t.Cleanup(func() { copyBlockSize = oldBlockSize })

		// More than one upload block.
		content := newContent(streamUploadBlockSize + 1234)
		require.NoError(t, store.PutStaging(ctx, "staged-1", bytes.NewReader(content)))

		size, err := store.StagingSize(ctx, "staged-1")
		require.NoError(t, err)
		require.EqualValues(t, len(content), size)

		rc, err := store.GetStagingRange(ctx, "staged-1", 100, 50)
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)
		require.Equal(t, content[100:150], got)

		require.NoError(t, store.Promote(ctx, "staged-1", "installer-1"))
		require.Equal(t, content, readInstaller("installer-1"))
		_, err = store.StagingSize(ctx, "staged-1")
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("copy staging", func(t *testing.T) {
		content := newContent(1000)
		require.NoError(t, store.PutStaging(ctx, "staged-src", bytes.NewReader(content)))
		require.NoError(t, store.CopyStaging(ctx, "staged-src", "staged-dst"))
		// Overwriting the source doesn't change the copy.
		require.NoError(t, store.PutStaging(ctx, "staged-src", bytes.NewReader(newContent(1000))))
		rc, err := store.GetStagingRange(ctx, "staged-dst", 0, int64(len(content)))
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)
		require.Equal(t, content, got)
		require.NoError(t, store.DeleteStaging(ctx, "staged-src"))
		require.NoError(t, store.DeleteStaging(ctx, "staged-dst"))

		require.Error(t, store.CopyStaging(ctx, "missing", "staged-dst"))
		require.Error(t, store.Promote(ctx, "missing", "installer-missing"))
		exists, err := store.Exists(ctx, "installer-missing")
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("promote to an existing installer keeps it", func(t *testing.T) {
		existing := newContent(100)
		require.NoError(t, store.Put(ctx, "installer-2", bytes.NewReader(existing)))
		require.NoError(t, store.PutStaging(ctx, "staged-2", bytes.NewReader(newContent(100))))

		require.NoError(t, store.Promote(ctx, "staged-2", "installer-2"))
		require.Equal(t, existing, readInstaller("installer-2"))
		_, err := store.StagingSize(ctx, "staged-2")
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("staging objects are cleaned up with staged uploads", func(t *testing.T) {
		require.NoError(t, store.PutStaging(ctx, "staged-3", bytes.NewReader(newContent(10))))
		uploads, err := NewStagedUploadStore(testAzureConfig(store.container))
		require.NoError(t, err)
		n, err := uploads.Cleanup(ctx, nil, time.Now().Add(time.Hour))
		require.NoError(t, err)
		require.Equal(t, 1, n)
		_, err = store.StagingSize(ctx, "staged-3")
		require.True(t, fleet.IsNotFound(err))
		// installers are untouched
		require.NotEmpty(t, readInstaller("installer-1"))

		require.NoError(t, uploads.Delete(ctx, "missing"))
		_, err = uploads.PresignPut(ctx, "id", 10, time.Minute)
		require.ErrorIs(t, err, fleet.ErrNotConfigured)
	})
}
