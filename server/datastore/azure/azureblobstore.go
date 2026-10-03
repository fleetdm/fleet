// Package azure implements fleet.SoftwareInstallerStore and friends backed by
// Azure Blob Storage. Azure Blob Storage has no S3-compatible API, so this is
// a separate implementation rather than another branch of the s3 package (see
// the s3 package's isGCS for how S3-compatible backends are handled instead).
package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type azureBlobStore struct {
	client    *azblob.Client
	container string
	// signedURL, when true, makes Sign() return a SAS-signed GET URL instead of
	// erroring out with fleet.ErrNotConfigured.
	signedURL bool
}

type fileNotFoundError struct{}

var _ fleet.NotFoundError = (*fileNotFoundError)(nil)

func (fileNotFoundError) Error() string {
	return "file not found"
}

func (fileNotFoundError) IsNotFound() bool {
	return true
}

// newAzureBlobStore initializes an Azure Blob Storage-backed store using
// shared-key authentication.
func newAzureBlobStore(cfg config.AzureConfigInternal) (*azureBlobStore, error) {
	if cfg.AccountName == "" || cfg.AccountKey == "" {
		return nil, fmt.Errorf("azure storage account name and account key are required")
	}
	if cfg.Container == "" {
		return nil, fmt.Errorf("azure storage container is required")
	}

	cred, err := azblob.NewSharedKeyCredential(cfg.AccountName, cfg.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("creating azure shared key credential: %w", err)
	}

	// ContainerURL overrides the default endpoint; used for Azurite/testing.
	serviceURL := cfg.ContainerURL
	if serviceURL == "" {
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net/", cfg.AccountName)
	}

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating azure blob client: %w", err)
	}

	return &azureBlobStore{
		client:    client,
		container: cfg.Container,
		signedURL: cfg.SignedURL,
	}, nil
}

// CreateTestContainer creates the container associated with this store.
// Only recommended for local testing.
func (s *azureBlobStore) CreateTestContainer(ctx context.Context, name string) error {
	_, err := s.client.CreateContainer(ctx, name, nil)
	if bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return nil
	}
	return err
}

// CleanupTestContainer empties and deletes the container associated with
// this store. Only recommended for local testing. If the container no
// longer exists, it returns nil.
func (s *azureBlobStore) CleanupTestContainer(ctx context.Context) error {
	pager := s.client.NewListBlobsFlatPager(s.container, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		if page.Segment == nil {
			continue
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			if _, err := s.client.DeleteBlob(ctx, s.container, *item.Name, nil); err != nil {
				return err
			}
		}
	}

	_, err := s.client.DeleteContainer(ctx, s.container, nil)
	if bloberror.HasCode(err, bloberror.ContainerNotFound) {
		return nil
	}
	return err
}
