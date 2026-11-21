package fleet_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mock "github.com/fleetdm/fleet/v4/server/mock/software"
	"github.com/stretchr/testify/require"
)

func TestRemoteInstallerReader(t *testing.T) {
	// Spans several 1 MiB blocks and ends mid-block.
	content := make([]byte, 3<<20+12345)
	for i := range content {
		content[i] = byte(i * 31)
	}
	store := mock.NewMemStreamingInstallerStore()
	require.NoError(t, store.PutStaging(t.Context(), "id", bytes.NewReader(content)))

	r, err := fleet.NewRemoteInstallerReader(t.Context(), store, "id", int64(len(content)), "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })

	// Generic io.Reader/io.ReaderAt/io.Seeker conformance.
	require.NoError(t, iotest.TestReader(r, content))
	require.NoError(t, r.Rewind())

	t.Run("ReadAt across blocks and past the end", func(t *testing.T) {
		buf := make([]byte, 100)
		n, err := r.ReadAt(buf, 1<<20-50)
		require.NoError(t, err)
		require.Equal(t, 100, n)
		require.Equal(t, content[1<<20-50:1<<20+50], buf)

		n, err = r.ReadAt(buf, int64(len(content))-10)
		require.ErrorIs(t, err, io.EOF)
		require.Equal(t, 10, n)
		require.Equal(t, content[len(content)-10:], buf[:n])

		n, err = r.ReadAt(buf, int64(len(content)))
		require.ErrorIs(t, err, io.EOF)
		require.Zero(t, n)
	})

	t.Run("ReadAt caches blocks", func(t *testing.T) {
		before := store.RangeRequests
		buf := make([]byte, 10)
		for i := range 100 {
			_, err := r.ReadAt(buf, int64(2<<20+i*100))
			require.NoError(t, err)
		}
		require.LessOrEqual(t, store.RangeRequests-before, 1)
	})

	t.Run("sequential Read uses a single request", func(t *testing.T) {
		require.NoError(t, r.Rewind())
		before := store.RangeRequests
		got, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, content, got)
		require.Equal(t, 1, store.RangeRequests-before)
	})

	t.Run("truncated object", func(t *testing.T) {
		short, err := fleet.NewRemoteInstallerReader(t.Context(), store, "id", int64(len(content))+10, "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = short.Close() })
		_, err = io.ReadAll(short)
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}

func TestStageInstaller(t *testing.T) {
	content := []byte("installer bytes")
	wantSum := sha256.Sum256(content)

	t.Run("success", func(t *testing.T) {
		store := mock.NewMemStreamingInstallerStore()
		r, err := fleet.StageInstaller(t.Context(), t.Context(), store, bytes.NewReader(content))
		require.NoError(t, err)
		require.Equal(t, wantSum[:], r.KnownSHA256())
		size, err := r.Size()
		require.NoError(t, err)
		require.EqualValues(t, len(content), size)
		require.Equal(t, 1, store.StagingCount())

		require.NoError(t, store.Promote(t.Context(), r.StagingID(), "sha"))
		require.Equal(t, content, store.Installers["sha"])
		require.Zero(t, store.StagingCount())
		require.NoError(t, r.Close())
	})

	t.Run("close deletes the staging object", func(t *testing.T) {
		store := mock.NewMemStreamingInstallerStore()
		r, err := fleet.StageInstaller(t.Context(), t.Context(), store, bytes.NewReader(content))
		require.NoError(t, err)
		require.Equal(t, 1, store.StagingCount())
		require.NoError(t, r.Close())
		require.Zero(t, store.StagingCount())
	})

	t.Run("validators", func(t *testing.T) {
		store := mock.NewMemStreamingInstallerStore()
		first, second := &recordingValidator{}, &recordingValidator{format: "fmt"}
		r, err := fleet.StageInstaller(t.Context(), t.Context(), store, bytes.NewReader(content), first, second)
		require.NoError(t, err)
		require.Equal(t, content, first.buf.Bytes())
		require.Equal(t, content, second.buf.Bytes())
		require.True(t, first.finished)
		require.Equal(t, "fmt", r.PrevalidatedFormat())
		require.NoError(t, r.Close())

		// Finished on failure too, without reporting a format.
		v := &recordingValidator{format: "fmt"}
		_, err = fleet.StageInstaller(t.Context(), t.Context(), store, iotest.ErrReader(errors.New("boom")), v)
		require.Error(t, err)
		require.True(t, v.finished)
	})

	t.Run("body read error is returned as is", func(t *testing.T) {
		store := mock.NewMemStreamingInstallerStore()
		readErr := errors.New("too large")
		_, err := fleet.StageInstaller(t.Context(), t.Context(), store, iotest.ErrReader(readErr))
		require.ErrorIs(t, err, readErr)
		require.Zero(t, store.StagingCount())
	})

	t.Run("store error", func(t *testing.T) {
		store := mock.NewMemStreamingInstallerStore()
		store.PutStagingErr = errors.New("store down")
		_, err := fleet.StageInstaller(t.Context(), t.Context(), store, bytes.NewReader(content))
		require.ErrorIs(t, err, store.PutStagingErr)
		require.Zero(t, store.StagingCount())
	})
}

type recordingValidator struct {
	buf      bytes.Buffer
	format   string
	finished bool
}

func (v *recordingValidator) Write(p []byte) (int, error) { return v.buf.Write(p) }

func (v *recordingValidator) Finish() string {
	v.finished = true
	return v.format
}

func TestPinnedRemoteInstallerReader(t *testing.T) {
	content := []byte("pinned installer bytes")
	store := mock.NewMemStreamingInstallerStore()
	require.NoError(t, store.PutStaging(t.Context(), "id", bytes.NewReader(content)))
	size, etag, err := store.StagingETag(t.Context(), "id")
	require.NoError(t, err)

	_, err = fleet.NewPinnedRemoteInstallerReader(t.Context(), store, "id", size, "", "")
	require.Error(t, err)

	r, err := fleet.NewPinnedRemoteInstallerReader(t.Context(), store, "id", size, etag, "")
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, content, got)
	require.NoError(t, r.Promote(t.Context(), "sha"))
	require.Equal(t, content, store.Installers["sha"])
	require.Equal(t, 1, store.StagingCount(), "a pinned object isn't Fleet's to delete")

	require.NoError(t, store.PutStaging(t.Context(), "id", bytes.NewReader([]byte("other"))))
	_, err = r.ReadAt(make([]byte, 4), 0)
	require.ErrorIs(t, err, fleet.ErrStagingChanged)
	require.ErrorIs(t, r.Promote(t.Context(), "sha2"), fleet.ErrStagingChanged)
	require.NoError(t, r.Close())
	require.Equal(t, 1, store.StagingCount())
}
