package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mocksoftware "github.com/fleetdm/fleet/v4/server/mock/software"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Only Get is reachable from openStagedUpload; the embedded interface covers the rest.
type fakeStagedUploadStore struct {
	fleet.StagedUploadStore
	getErr     error
	getInvoked bool
}

func (f *fakeStagedUploadStore) Get(ctx context.Context, uploadID string) (io.ReadCloser, int64, error) {
	f.getInvoked = true
	return nil, 0, f.getErr
}

func TestOpenStagedUploadRequiresDirectUpload(t *testing.T) {
	t.Run("option off", func(t *testing.T) {
		store := &fakeStagedUploadStore{}
		svc := &Service{stagedUploadStore: store}

		_, err := svc.openStagedUpload(t.Context(), uuid.NewString())
		require.ErrorContains(t, err, "Direct upload isn't available")
		require.False(t, store.getInvoked)
	})

	t.Run("option on", func(t *testing.T) {
		store := &fakeStagedUploadStore{getErr: &notFoundError{}}
		cfg := config.FleetConfig{}
		cfg.S3.SoftwareInstallersSignedURL = true
		cfg.S3.SoftwareInstallersBucket = "test-bucket"
		svc := &Service{stagedUploadStore: store, config: cfg}

		_, err := svc.openStagedUpload(t.Context(), uuid.NewString())
		require.ErrorContains(t, err, "Upload not found or expired")
		require.True(t, store.getInvoked)
	})
}

func TestOpenStagedSoftwareUploadStreaming(t *testing.T) {
	store := mocksoftware.NewMemStreamingInstallerStore()
	store.NoETagPinning = true
	cfg := config.FleetConfig{}
	cfg.S3.SoftwareInstallersSignedURL = true
	cfg.S3.SoftwareInstallersBucket = "test-bucket"
	cfg.CloudStorage.StreamInstallerDownloads = true
	svc := &Service{stagedUploadStore: &fakeStagedUploadStore{}, softwareInstallStore: store, config: cfg, logger: discardLogger()}

	_, err := svc.openStagedSoftwareUpload(t.Context(), uuid.NewString())
	require.ErrorContains(t, err, "Upload not found or expired")

	uploadID := uuid.NewString()
	original := []byte("original bytes")
	require.NoError(t, store.PutStaging(t.Context(), uploadID, bytes.NewReader(original)))

	r, err := svc.openStagedSoftwareUpload(t.Context(), uploadID)
	require.NoError(t, err)
	rir, ok := r.(*fleet.RemoteInstallerReader)
	require.True(t, ok)
	require.NotEqual(t, uploadID, rir.StagingID(), "reads a copy Fleet owns")
	wantSum := sha256.Sum256(original)
	require.Equal(t, wantSum[:], rir.KnownSHA256())

	// The client overwriting its upload doesn't change what Fleet reads and stores.
	require.NoError(t, store.PutStaging(t.Context(), uploadID, bytes.NewReader([]byte("swapped bytes!"))))
	got, err := io.ReadAll(rir)
	require.NoError(t, err)
	require.Equal(t, original, got)
	require.NoError(t, putInstallerToStore(t.Context(), store, hex.EncodeToString(wantSum[:]), rir))
	require.Equal(t, original, store.Installers[hex.EncodeToString(wantSum[:])])

	// Closing removes Fleet's copy but keeps the client's upload, which is only
	// deleted once the request succeeds.
	require.NoError(t, rir.Close())
	require.Equal(t, 1, store.StagingCount())
	_, ok = store.Staging[uploadID]
	require.True(t, ok)
}

func TestOpenStagedSoftwareUploadPinned(t *testing.T) {
	store := mocksoftware.NewMemStreamingInstallerStore()
	cfg := config.FleetConfig{}
	cfg.S3.SoftwareInstallersSignedURL = true
	cfg.S3.SoftwareInstallersBucket = "test-bucket"
	cfg.CloudStorage.StreamInstallerDownloads = true
	svc := &Service{stagedUploadStore: &fakeStagedUploadStore{}, softwareInstallStore: store, config: cfg, logger: discardLogger()}

	_, err := svc.openStagedSoftwareUpload(t.Context(), uuid.NewString())
	require.ErrorContains(t, err, "Upload not found or expired")

	open := func(t *testing.T, content []byte) (string, *fleet.RemoteInstallerReader) {
		uploadID := uuid.NewString()
		require.NoError(t, store.PutStaging(t.Context(), uploadID, bytes.NewReader(content)))
		r, err := svc.openStagedSoftwareUpload(t.Context(), uploadID)
		require.NoError(t, err)
		rir, ok := r.(*fleet.RemoteInstallerReader)
		require.True(t, ok)
		require.Equal(t, uploadID, rir.StagingID(), "reads the upload in place")
		require.Equal(t, 1, store.StagingCount(), "no copy")
		return uploadID, rir
	}

	t.Run("promote", func(t *testing.T) {
		original := []byte("original bytes")
		uploadID, rir := open(t, original)
		wantSum := sha256.Sum256(original)
		require.Equal(t, wantSum[:], rir.KnownSHA256())
		require.Empty(t, rir.PrevalidatedFormat())

		require.NoError(t, putInstallerToStore(t.Context(), store, hex.EncodeToString(wantSum[:]), rir))
		require.Equal(t, original, store.Installers[hex.EncodeToString(wantSum[:])])

		// The client's upload is kept until the request succeeds.
		require.NoError(t, rir.Close())
		_, ok := store.Staging[uploadID]
		require.True(t, ok)
		delete(store.Staging, uploadID)
	})

	t.Run("upload swapped after hashing", func(t *testing.T) {
		original := []byte("other original bytes")
		uploadID, rir := open(t, original)

		require.NoError(t, store.PutStaging(t.Context(), uploadID, bytes.NewReader([]byte("swapped bytes!"))))
		_, err := io.ReadAll(rir)
		require.ErrorIs(t, err, fleet.ErrStagingChanged)
		wantSum := sha256.Sum256(original)
		err = putInstallerToStore(t.Context(), store, hex.EncodeToString(wantSum[:]), rir)
		require.ErrorIs(t, err, fleet.ErrStagingChanged)
		require.NotContains(t, store.Installers, hex.EncodeToString(wantSum[:]))

		// The request fails with a client error and keeps the upload.
		svc.finishStagedUpload(t.Context(), uploadID, rir, &err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		require.Contains(t, badReq.Message, "upload changed")
		_, ok := store.Staging[uploadID]
		require.True(t, ok)
		delete(store.Staging, uploadID)
	})

	t.Run("tar.gz is validated while hashing", func(t *testing.T) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: "a.txt", Mode: 0o644, Size: 3}))
		_, err := tw.Write([]byte("abc"))
		require.NoError(t, err)
		require.NoError(t, tw.Close())
		require.NoError(t, gz.Close())

		uploadID, rir := open(t, buf.Bytes())
		require.Equal(t, "tar.gz", rir.PrevalidatedFormat())
		before := store.RangeRequests
		meta, err := file.ExtractInstallerMetadata(rir)
		require.NoError(t, err)
		require.Equal(t, "tar.gz", meta.Extension)
		wantSum := sha256.Sum256(buf.Bytes())
		require.Equal(t, wantSum[:], meta.SHASum)
		// Only the type sniffing reads the content again.
		require.Equal(t, 1, store.RangeRequests-before)
		require.NoError(t, rir.Close())
		delete(store.Staging, uploadID)
	})
}
