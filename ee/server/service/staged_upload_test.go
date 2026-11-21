package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mocksvc "github.com/fleetdm/fleet/v4/server/mock/service"
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

// PresignPut records whether a URL was requested, without contacting a bucket.
type presignStagedUploadStore struct {
	fleet.StagedUploadStore
	invoked bool
}

func (s *presignStagedUploadStore) PresignPut(context.Context, string, int64, time.Duration) (string, error) {
	s.invoked = true
	return "https://example.com/upload", nil
}

func TestCreateStagedUploadAppleMDM(t *testing.T) {
	configErr := errors.New("load app config")
	for _, tt := range []struct {
		name      string
		target    fleet.StagedUploadTarget
		mdmErr    error
		wantErr   error
		wantCheck bool
	}{
		{"bootstrap MDM disabled", fleet.StagedUploadTargetBootstrapPackage, fleet.ErrMDMNotConfigured, fleet.ErrMDMNotConfigured, true},
		{"bootstrap MDM enabled", fleet.StagedUploadTargetBootstrapPackage, nil, nil, true},
		{"bootstrap config error", fleet.StagedUploadTargetBootstrapPackage, configErr, configErr, true},
		{"software MDM disabled", fleet.StagedUploadTargetSoftwarePackage, fleet.ErrMDMNotConfigured, nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authorizer, err := authz.NewAuthorizer()
			require.NoError(t, err)
			checked := false
			store := &presignStagedUploadStore{}
			svc := &Service{
				authz:             authorizer,
				config:            config.TestConfig(),
				stagedUploadStore: store,
				Service: &mocksvc.Service{VerifyMDMAppleConfiguredFunc: func(context.Context) error {
					checked = true
					return tt.mdmErr
				}},
			}
			ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
			upload, err := svc.CreateStagedUpload(ctx, tt.target, 0, 1)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantCheck, checked)
			require.Equal(t, tt.wantErr == nil, store.invoked)
			if tt.wantErr != nil {
				require.Nil(t, upload)
			} else {
				require.NotEmpty(t, upload.UploadID)
			}
		})
	}
}
