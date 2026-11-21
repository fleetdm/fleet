package azure

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/sync/errgroup"
)

var _ fleet.StreamingInstallerStore = (*SoftwareInstallerStore)(nil)

const (
	// streamUploadBlockSize and streamUploadConcurrency bound the memory used
	// to upload a stream, which the SDK buffers block by block.
	streamUploadBlockSize   = 8 << 20 // 8 MiB, i.e. up to ~390 GiB with Azure's 50,000 blocks
	streamUploadConcurrency = 4

	// copySourceSASExpiry is how long the copy source URL stays valid.
	copySourceSASExpiry = 1 * time.Hour

	copyConcurrency = 4
)

// copyBlockSize is the size of the blocks staged from the source when copying
// a blob (Azure allows up to 4000 MiB per block and 50,000 blocks per blob).
// It is a variable so tests can lower it.
var copyBlockSize int64 = 100 << 20

// keyForStaging builds the blob name of a staging object. Staging objects use
// the staged uploads prefix, so the staged uploads cleanup removes abandoned
// ones.
func (s *SoftwareInstallerStore) keyForStaging(stagingID string) string {
	return path.Join(stagedUploadPrefix, stagingID)
}

func (s *SoftwareInstallerStore) blobClient(key string) *blob.Client {
	return s.client.ServiceClient().NewContainerClient(s.container).NewBlobClient(key)
}

// PutStaging streams r to the staging object stagingID.
func (s *SoftwareInstallerStore) PutStaging(ctx context.Context, stagingID string, r io.Reader) error {
	if stagingID == "" {
		return errors.New("azure blob staging identifier is empty")
	}
	_, err := s.client.UploadStream(ctx, s.container, s.keyForStaging(stagingID), r, &azblob.UploadStreamOptions{
		BlockSize:   streamUploadBlockSize,
		Concurrency: streamUploadConcurrency,
	})
	if err != nil {
		return ctxerr.Wrapf(ctx, newAPIError(err, s.container), "uploading staged %s to Azure Blob store", s.fileLabel)
	}
	return nil
}

// GetStagingRange returns the n bytes of the staging object starting at off.
func (s *SoftwareInstallerStore) GetStagingRange(ctx context.Context, stagingID string, off, n int64) (io.ReadCloser, error) {
	if off < 0 || n < 0 {
		return nil, ctxerr.Errorf(ctx, "invalid range %d+%d for staged %s", off, n, s.fileLabel)
	}
	if n == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	resp, err := s.client.DownloadStream(ctx, s.container, s.keyForStaging(stagingID), &azblob.DownloadStreamOptions{
		Range: azblob.HTTPRange{Offset: off, Count: n},
	})
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, fileNotFoundError{}
		}
		return nil, ctxerr.Wrapf(ctx, newAPIError(err, s.container), "retrieving staged %s from Azure Blob store", s.fileLabel)
	}
	return resp.Body, nil
}

// StagingSize returns the size of the staging object.
func (s *SoftwareInstallerStore) StagingSize(ctx context.Context, stagingID string) (int64, error) {
	return s.blobSize(ctx, s.keyForStaging(stagingID))
}

// Promote copies the staging object to the installer key installerID, unless
// that installer already exists, and then deletes the staging object. The
// installer is stored once the copy succeeds, so failing to delete the staging
// object isn't an error: the staged uploads cleanup removes it.
func (s *SoftwareInstallerStore) Promote(ctx context.Context, stagingID, installerID string) error {
	if installerID == "" {
		return errors.New("azure blob file identifier is empty")
	}
	exists, err := s.Exists(ctx, installerID)
	if err != nil {
		return err
	}
	if !exists {
		if err := s.copyBlob(ctx, s.keyForStaging(stagingID), s.keyForFile(installerID)); err != nil {
			return err
		}
	}
	_ = s.DeleteStaging(ctx, stagingID)
	return nil
}

// CopyStaging copies the staging object srcStagingID to dstStagingID.
func (s *SoftwareInstallerStore) CopyStaging(ctx context.Context, srcStagingID, dstStagingID string) error {
	return s.copyBlob(ctx, s.keyForStaging(srcStagingID), s.keyForStaging(dstStagingID))
}

// DeleteStaging deletes the staging object. Deleting a missing object is not
// an error.
func (s *SoftwareInstallerStore) DeleteStaging(ctx context.Context, stagingID string) error {
	return s.deleteKey(ctx, s.keyForStaging(stagingID))
}

// copyBlob copies src to dst server-side, by staging the destination's blocks
// from ranges of the source and committing them. Unlike Copy Blob (which is
// asynchronous and makes the destination visible while the copy is pending),
// the destination only appears once all blocks are committed, so a failure
// never leaves a partial installer under its final key.
//
// Emulators that don't implement Put Block From URL (e.g. Azurite) fall back
// to streaming the bytes through Fleet, which is also committed atomically.
func (s *SoftwareInstallerStore) copyBlob(ctx context.Context, src, dst string) error {
	size, err := s.blobSize(ctx, src)
	if err != nil {
		return err
	}

	// A SAS URL authorizes the read of the source independently of how the
	// requests on the destination are authorized.
	srcURL, err := s.blobClient(src).GetSASURL(sas.BlobPermissions{Read: true}, time.Now().Add(copySourceSASExpiry), nil)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "signing staged %s copy source in Azure Blob store", s.fileLabel)
	}

	err = s.copyBlobFromURL(ctx, srcURL, dst, size)
	if isNotImplemented(err) {
		err = s.copyBlobThroughFleet(ctx, src, dst)
	}
	if err != nil {
		return ctxerr.Wrapf(ctx, newAPIError(err, s.container), "copying staged %s in Azure Blob store", s.fileLabel)
	}
	return nil
}

func (s *SoftwareInstallerStore) copyBlobFromURL(ctx context.Context, srcURL, dst string, size int64) error {
	numBlocks := (size + copyBlockSize - 1) / copyBlockSize
	blockIDs := make([]string, numBlocks)
	dstClient := s.client.ServiceClient().NewContainerClient(s.container).NewBlockBlobClient(dst)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(copyConcurrency)
	for i := range numBlocks {
		// Block IDs of a blob must all have the same length.
		blockIDs[i] = base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%016d", i))
		off := i * copyBlockSize
		g.Go(func() error {
			_, err := dstClient.StageBlockFromURL(gctx, blockIDs[i], srcURL, &blockblob.StageBlockFromURLOptions{
				Range: blob.HTTPRange{Offset: off, Count: min(copyBlockSize, size-off)},
			})
			return err
		})
	}
	if err := g.Wait(); err != nil {
		// The staged blocks are uncommitted, Azure garbage collects them.
		return err
	}
	_, err := dstClient.CommitBlockList(ctx, blockIDs, nil)
	return err
}

func (s *SoftwareInstallerStore) copyBlobThroughFleet(ctx context.Context, src, dst string) error {
	resp, err := s.client.DownloadStream(ctx, s.container, src, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.client.UploadStream(ctx, s.container, dst, resp.Body, &azblob.UploadStreamOptions{
		BlockSize:   streamUploadBlockSize,
		Concurrency: streamUploadConcurrency,
	})
	return err
}

func (s *SoftwareInstallerStore) blobSize(ctx context.Context, key string) (int64, error) {
	props, err := s.blobClient(key).GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return 0, fileNotFoundError{}
		}
		return 0, ctxerr.Wrapf(ctx, newAPIError(err, s.container), "checking size of staged %s in Azure Blob store", s.fileLabel)
	}
	if props.ContentLength == nil {
		return 0, ctxerr.Errorf(ctx, "missing size of staged %s in Azure Blob store", s.fileLabel)
	}
	return *props.ContentLength, nil
}

func isNotImplemented(err error) bool {
	var respErr *azcore.ResponseError
	return errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotImplemented
}
