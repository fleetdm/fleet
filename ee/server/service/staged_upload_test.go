package service

import (
	"context"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
