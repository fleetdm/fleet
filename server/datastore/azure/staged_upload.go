package azure

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// stagedUploadPrefix must not string-prefix match the other stores' path
// prefixes (or vice versa): commonFileStore.Cleanup lists by prefix, so an
// overlap would let one store's sweep delete the other's objects.
const stagedUploadPrefix = "uploads"

// StagedUploadStore holds staging objects awaiting promotion to a software
// installer. Azure doesn't support client uploads straight to the store
// (PresignPut), it is used for the staging objects of streamed installer
// downloads and for the cleanup of abandoned ones.
type StagedUploadStore struct {
	*commonFileStore
}

var _ fleet.StagedUploadStore = (*StagedUploadStore)(nil)

// NewStagedUploadStore creates a staged uploads store using the software
// installers Azure config.
func NewStagedUploadStore(cfg config.AzureConfig) (*StagedUploadStore, error) {
	store, err := newAzureBlobStore(cfg.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &StagedUploadStore{
		&commonFileStore{
			azureBlobStore: store,
			pathPrefix:     stagedUploadPrefix,
			fileLabel:      "staged upload",
		},
	}, nil
}

// PresignPut is not supported with Azure Blob Storage.
func (s *StagedUploadStore) PresignPut(ctx context.Context, _ string, _ int64, _ time.Duration) (string, error) {
	return "", ctxerr.Wrap(ctx, fleet.ErrNotConfigured, "presigning staged upload URL in Azure Blob store")
}
