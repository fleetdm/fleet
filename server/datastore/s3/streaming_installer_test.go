package s3

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallerStreaming(t *testing.T) {
	ctx := t.Context()
	store := setupTestSoftwareInstallerStore(t, "software-installers-streaming-test", "prefix")

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
		content := newContent(1 << 20)
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

	t.Run("multipart copy", func(t *testing.T) {
		oldMax, oldPart := maxSingleCopySize, multipartCopyPartSize
		maxSingleCopySize, multipartCopyPartSize = 6<<20, 5<<20 // 5 MiB is S3's minimum part size
		t.Cleanup(func() { maxSingleCopySize, multipartCopyPartSize = oldMax, oldPart })

		content := newContent(11<<20 + 7)
		require.NoError(t, store.PutStaging(ctx, "staged-3", bytes.NewReader(content)))
		require.NoError(t, store.Promote(ctx, "staged-3", "installer-3"))
		require.Equal(t, content, readInstaller("installer-3"))
	})

	t.Run("ETag pinning", func(t *testing.T) {
		content := newContent(1000)
		require.NoError(t, store.PutStaging(ctx, "staged-pin", bytes.NewReader(content)))
		size, etag, err := store.StagingETag(ctx, "staged-pin")
		require.NoError(t, err)
		require.EqualValues(t, len(content), size)
		require.NotEmpty(t, etag)

		rc, err := store.GetStagingRangeIfMatch(ctx, "staged-pin", etag, 10, 20)
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)
		require.Equal(t, content[10:30], got)

		// Promoting a pinned object keeps it: it isn't Fleet's own.
		require.NoError(t, store.PromoteIfMatch(ctx, "staged-pin", etag, "installer-pin"))
		require.Equal(t, content, readInstaller("installer-pin"))
		_, err = store.StagingSize(ctx, "staged-pin")
		require.NoError(t, err)

		// Overwriting the object fails reads and promotion.
		require.NoError(t, store.PutStaging(ctx, "staged-pin", bytes.NewReader(newContent(1000))))
		_, err = store.GetStagingRangeIfMatch(ctx, "staged-pin", etag, 0, 10)
		require.ErrorIs(t, err, fleet.ErrStagingChanged)
		require.ErrorIs(t, store.PromoteIfMatch(ctx, "staged-pin", etag, "installer-pin-2"), fleet.ErrStagingChanged)
		exists, err := store.Exists(ctx, "installer-pin-2")
		require.NoError(t, err)
		require.False(t, exists)

		// The copy itself is conditional too, in case the object changes after
		// the ETag check.
		src := store.keyForStaging("staged-pin")
		err = store.copyObject(ctx, src, etag, store.keyForFile("installer-pin-3"))
		require.ErrorIs(t, err, fleet.ErrStagingChanged)

		oldMax, oldPart := maxSingleCopySize, multipartCopyPartSize
		maxSingleCopySize, multipartCopyPartSize = 6<<20, 5<<20
		t.Cleanup(func() { maxSingleCopySize, multipartCopyPartSize = oldMax, oldPart })
		big := newContent(11 << 20)
		require.NoError(t, store.PutStaging(ctx, "staged-pin-big", bytes.NewReader(big)))
		_, bigETag, err := store.StagingETag(ctx, "staged-pin-big")
		require.NoError(t, err)
		err = store.multipartCopy(ctx, store.keyForStaging("staged-pin-big"), `"stale"`, store.keyForFile("installer-pin-4"), int64(len(big)))
		require.ErrorIs(t, err, fleet.ErrStagingChanged)
		require.NoError(t, store.PromoteIfMatch(ctx, "staged-pin-big", bigETag, "installer-pin-4"))
		require.Equal(t, big, readInstaller("installer-pin-4"))

		require.NoError(t, store.DeleteStaging(ctx, "staged-pin"))
		require.NoError(t, store.DeleteStaging(ctx, "staged-pin-big"))
	})

	t.Run("staging objects are cleaned up with staged uploads", func(t *testing.T) {
		require.NoError(t, store.PutStaging(ctx, "staged-4", bytes.NewReader(newContent(10))))
		uploads := &StagedUploadStore{&commonFileStore{s3store: store.s3store, pathPrefix: stagedUploadPrefix, fileLabel: "staged upload"}}
		rc, _, err := uploads.Get(ctx, "staged-4")
		require.NoError(t, err)
		rc.Close()
	})
}
