package azure

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type OrgLogoStore struct {
	*commonFileStore
}

// NewOrgLogoStore reuses the software-installers Azure config (same
// container, distinct prefix) — see azure.NewSoftwareTitleIconStore for the
// precedent.
func NewOrgLogoStore(cfg config.AzureConfig) (*OrgLogoStore, error) {
	store, err := newAzureBlobStore(cfg.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &OrgLogoStore{
		&commonFileStore{
			azureBlobStore: store,
			pathPrefix:     "org-logos",
			fileLabel:      "org logo",
		},
	}, nil
}

// Put stores the logo bytes for mode under <container>/org-logos/<mode>.
func (s *OrgLogoStore) Put(ctx context.Context, mode fleet.OrgLogoMode, content io.ReadSeeker) error {
	return s.commonFileStore.Put(ctx, string(mode), content)
}

func (s *OrgLogoStore) Get(ctx context.Context, mode fleet.OrgLogoMode) (io.ReadCloser, int64, error) {
	return s.commonFileStore.Get(ctx, string(mode))
}

func (s *OrgLogoStore) Exists(ctx context.Context, mode fleet.OrgLogoMode) (bool, error) {
	return s.commonFileStore.Exists(ctx, string(mode))
}

func (s *OrgLogoStore) Delete(ctx context.Context, mode fleet.OrgLogoMode) error {
	key := s.keyForFile(string(mode))
	_, err := s.client.DeleteBlob(ctx, s.container, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "deleting org logo from Azure Blob store")
	}
	return nil
}
