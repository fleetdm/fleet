package azure

import (
	"context"
	"errors"
	"io"
	"path"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/sync/errgroup"
)

// commonFileStore implements the common Get, Put, Exists, Sign and Cleanup
// operations typical for storage of files in an Azure Blob Storage container.
// It mirrors the s3 package's commonFileStore. The only variable thing is the
// path prefix inside the configured container, e.g. for software installers
// it is:
//
//	<container>/software-installers/<fileID>
type commonFileStore struct {
	*azureBlobStore
	pathPrefix string
	fileLabel  string // how to call the file in error messages
}

// Get retrieves the requested file from Azure Blob Storage.
// It is important that the caller closes the reader when done.
func (s *commonFileStore) Get(ctx context.Context, fileID string) (io.ReadCloser, int64, error) {
	key := s.keyForFile(fileID)

	resp, err := s.client.DownloadStream(ctx, s.container, key, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
			return nil, int64(0), fileNotFoundError{}
		}
		return nil, int64(0), ctxerr.Wrapf(ctx, err, "retrieving %s from Azure Blob store", s.fileLabel)
	}

	var contentLength int64
	if resp.ContentLength != nil {
		contentLength = *resp.ContentLength
	}
	return resp.Body, contentLength, nil
}

// Put uploads a file to Azure Blob Storage.
func (s *commonFileStore) Put(ctx context.Context, fileID string, content io.ReadSeeker) error {
	if fileID == "" {
		return errors.New("azure blob file identifier is empty")
	}

	key := s.keyForFile(fileID)

	_, err := s.client.UploadStream(ctx, s.container, key, content, nil)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "uploading %s to Azure Blob store", s.fileLabel)
	}
	return nil
}

// Exists checks if a file exists in the Azure Blob Storage container for the ID.
func (s *commonFileStore) Exists(ctx context.Context, fileID string) (bool, error) {
	key := s.keyForFile(fileID)

	blobClient := s.client.ServiceClient().NewContainerClient(s.container).NewBlobClient(key)
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ContainerNotFound) {
			return false, nil
		}
		return false, ctxerr.Wrapf(ctx, err, "checking existence of %s in Azure Blob store", s.fileLabel)
	}
	return true, nil
}

func (s *commonFileStore) Cleanup(ctx context.Context, usedFileIDs []string, removeCreatedBefore time.Time) (int, error) {
	removeCreatedBefore = removeCreatedBefore.UTC()

	usedSet := make(map[string]struct{}, len(usedFileIDs))
	for _, id := range usedFileIDs {
		usedSet[id] = struct{}{}
	}

	// Only list a single page, same tradeoff as the S3 store's Cleanup: doing so
	// bounds the number of API requests and the window in which an unused file
	// could become used again between listing and deleting.
	prefix := s.pathPrefix + "/"
	pager := s.client.NewListBlobsFlatPager(s.container, &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	if !pager.More() {
		return 0, nil
	}
	page, err := pager.NextPage(ctx)
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "listing %s in Azure Blob store", s.fileLabel)
	}

	var toDelete []string
	if page.Segment != nil {
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			if _, ok := usedSet[path.Base(*item.Name)]; ok {
				continue
			}
			// default to doing the cleanup if we don't have the timestamp information
			if item.Properties == nil || item.Properties.LastModified == nil || !item.Properties.LastModified.UTC().After(removeCreatedBefore) {
				toDelete = append(toDelete, *item.Name)
			}
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	var deleted atomic.Int32
	var g errgroup.Group
	g.SetLimit(10)

	for _, key := range toDelete {
		key := key
		g.Go(func() error {
			_, err := s.client.DeleteBlob(ctx, s.container, key, nil)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "deleting %s in Azure Blob store", s.fileLabel)
			}
			deleted.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return int(deleted.Load()), ctxerr.Wrap(ctx, err, "errors occurred during Azure Blob deletion")
	}

	return int(deleted.Load()), nil
}

func (s *commonFileStore) Sign(ctx context.Context, fileID string, expiresIn time.Duration) (string, error) {
	if !s.signedURL {
		return "", ctxerr.Wrapf(ctx, fleet.ErrNotConfigured, "signing %s URL in Azure Blob store", s.fileLabel)
	}

	key := s.keyForFile(fileID)
	blobClient := s.client.ServiceClient().NewContainerClient(s.container).NewBlobClient(key)
	signedURL, err := blobClient.GetSASURL(sas.BlobPermissions{Read: true}, time.Now().Add(expiresIn), nil)
	if err != nil {
		return "", ctxerr.Wrapf(ctx, err, "signing %s URL in Azure Blob store", s.fileLabel)
	}
	return signedURL, nil
}

// keyForFile builds an Azure blob name to identify the file.
func (s *commonFileStore) keyForFile(fileID string) string {
	return path.Join(s.pathPrefix, fileID)
}
