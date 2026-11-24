package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/sync/errgroup"
)

var (
	_ fleet.StreamingInstallerStore = (*SoftwareInstallerStore)(nil)
	_ fleet.ETagStagingStore        = (*SoftwareInstallerStore)(nil)
)

const (
	// streamUploadPartSize and streamUploadConcurrency bound the memory used to
	// upload a non-seekable stream, which the uploader buffers part by part.
	streamUploadPartSize    = 8 << 20 // 8 MiB, i.e. up to ~78 GiB with S3's 10,000 parts
	streamUploadConcurrency = 4

	multipartCopyConcurrency = 4
)

// maxSingleCopySize is the largest object CopyObject accepts; larger objects
// are copied part by part with UploadPartCopy. multipartCopyPartSize is the
// size of those parts. Both are variables so tests can lower them.
var (
	maxSingleCopySize     int64 = 5 << 30   // 5 GiB
	multipartCopyPartSize int64 = 512 << 20 // 512 MiB
)

// keyForStaging builds the S3 key of a staging object. Staging objects share
// the staged uploads prefix, so the staged uploads cleanup removes abandoned
// ones and a staged upload can be promoted directly.
func (s *SoftwareInstallerStore) keyForStaging(stagingID string) string {
	return path.Join(s.prefix, stagedUploadPrefix, stagingID)
}

// PutStaging streams r to the staging object stagingID.
func (s *SoftwareInstallerStore) PutStaging(ctx context.Context, stagingID string, r io.Reader) error {
	if stagingID == "" {
		return errors.New("S3 staging identifier is empty")
	}
	key := s.keyForStaging(stagingID)

	uploader := manager.NewUploader(s.s3Client, func(u *manager.Uploader) {
		u.PartSize = streamUploadPartSize
		u.Concurrency = streamUploadConcurrency
	})

	var checksumAlgorithm types.ChecksumAlgorithm
	if s.gcs {
		checksumAlgorithm = types.ChecksumAlgorithmCrc32c // Required for GCS
	}

	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:            &s.bucket,
		Key:               &key,
		Body:              r,
		ChecksumAlgorithm: checksumAlgorithm,
	}); err != nil {
		return ctxerr.Wrapf(ctx, s.wrapAPIError(err), "uploading staged %s to S3 store", s.fileLabel)
	}
	return nil
}

// GetStagingRange returns the n bytes of the staging object starting at off.
func (s *SoftwareInstallerStore) GetStagingRange(ctx context.Context, stagingID string, off, n int64) (io.ReadCloser, error) {
	return s.getStagingRange(ctx, stagingID, "", off, n)
}

func (s *SoftwareInstallerStore) getStagingRange(ctx context.Context, stagingID, ifMatch string, off, n int64) (io.ReadCloser, error) {
	if off < 0 || n < 0 {
		return nil, ctxerr.Errorf(ctx, "invalid range %d+%d for staged %s", off, n, s.fileLabel)
	}
	if n == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	key := s.keyForStaging(stagingID)
	rng := fmt.Sprintf("bytes=%d-%d", off, off+n-1)
	input := &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
		Range:  &rng,
	}
	if ifMatch != "" {
		input.IfMatch = &ifMatch
	}
	res, err := s.s3Client.GetObject(ctx, input)
	if err != nil {
		if isS3NotFound(err) {
			return nil, installerNotFoundError{}
		}
		if isS3PreconditionFailed(err) {
			return nil, ctxerr.Wrapf(ctx, fleet.ErrStagingChanged, "retrieving staged %s from S3 store", s.fileLabel)
		}
		return nil, ctxerr.Wrapf(ctx, s.wrapAPIError(err), "retrieving staged %s from S3 store", s.fileLabel)
	}
	return res.Body, nil
}

// StagingSize returns the size of the staging object.
func (s *SoftwareInstallerStore) StagingSize(ctx context.Context, stagingID string) (int64, error) {
	size, _, err := s.headStaging(ctx, stagingID)
	return size, err
}

func (s *SoftwareInstallerStore) headStaging(ctx context.Context, stagingID string) (size int64, etag string, err error) {
	key := s.keyForStaging(stagingID)
	res, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if isS3NotFound(err) {
			return 0, "", installerNotFoundError{}
		}
		return 0, "", ctxerr.Wrapf(ctx, s.wrapAPIError(err), "checking size of staged %s in S3 store", s.fileLabel)
	}
	if res.ContentLength == nil {
		return 0, "", ctxerr.Errorf(ctx, "missing size of staged %s in S3 store", s.fileLabel)
	}
	if res.ETag != nil {
		etag = *res.ETag
	}
	return *res.ContentLength, etag, nil
}

// Promote copies the staging object to the installer key installerID, unless
// that installer already exists, and then deletes the staging object. The
// installer is stored once the copy succeeds, so failing to delete the staging
// object isn't an error: the staged uploads cleanup removes it.
func (s *SoftwareInstallerStore) Promote(ctx context.Context, stagingID, installerID string) error {
	if err := s.promote(ctx, stagingID, "", installerID); err != nil {
		return err
	}
	_ = s.DeleteStaging(ctx, stagingID)
	return nil
}

// StagingETag returns the size and ETag of the staging object. On GCS, whose
// support for the conditional S3 copy headers isn't verified, the ETag is
// empty so the object is copied instead of pinned.
func (s *SoftwareInstallerStore) StagingETag(ctx context.Context, stagingID string) (int64, string, error) {
	size, etag, err := s.headStaging(ctx, stagingID)
	if err != nil {
		return 0, "", err
	}
	if s.gcs {
		etag = ""
	}
	return size, etag, nil
}

// GetStagingRangeIfMatch is GetStagingRange, failing with
// fleet.ErrStagingChanged if the staging object's ETag isn't etag.
func (s *SoftwareInstallerStore) GetStagingRangeIfMatch(ctx context.Context, stagingID, etag string, off, n int64) (io.ReadCloser, error) {
	if etag == "" {
		return nil, ctxerr.Errorf(ctx, "missing ETag for staged %s", s.fileLabel)
	}
	return s.getStagingRange(ctx, stagingID, etag, off, n)
}

// PromoteIfMatch copies the staging object to the installer key installerID,
// unless that installer already exists, failing with fleet.ErrStagingChanged
// if the staging object's ETag isn't etag. It doesn't delete the staging
// object.
func (s *SoftwareInstallerStore) PromoteIfMatch(ctx context.Context, stagingID, etag, installerID string) error {
	if etag == "" {
		return ctxerr.Errorf(ctx, "missing ETag for staged %s", s.fileLabel)
	}
	return s.promote(ctx, stagingID, etag, installerID)
}

func (s *SoftwareInstallerStore) promote(ctx context.Context, stagingID, ifMatch, installerID string) error {
	if installerID == "" {
		return errors.New("S3 file identifier is empty")
	}
	exists, err := s.Exists(ctx, installerID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.copyFromStaging(ctx, stagingID, ifMatch, s.keyForFile(installerID))
}

// CopyStaging copies the staging object srcStagingID to dstStagingID.
func (s *SoftwareInstallerStore) CopyStaging(ctx context.Context, srcStagingID, dstStagingID string) error {
	return s.copyFromStaging(ctx, srcStagingID, "", s.keyForStaging(dstStagingID))
}

// copyFromStaging copies the staging object to dst. A non-empty ifMatch makes
// the copy conditional on the staging object's ETag.
func (s *SoftwareInstallerStore) copyFromStaging(ctx context.Context, stagingID, ifMatch, dst string) error {
	size, etag, err := s.headStaging(ctx, stagingID)
	if err != nil {
		return err
	}
	if ifMatch != "" && etag != ifMatch {
		return ctxerr.Wrapf(ctx, fleet.ErrStagingChanged, "copying staged %s in S3 store", s.fileLabel)
	}
	src := s.keyForStaging(stagingID)
	// GCS's XML API has no 5 GiB copy limit and doesn't support UploadPartCopy.
	if size <= maxSingleCopySize || s.gcs {
		return s.copyObject(ctx, src, ifMatch, dst)
	}
	return s.multipartCopy(ctx, src, ifMatch, dst, size)
}

// DeleteStaging deletes the staging object. Deleting a missing object is not
// an error.
func (s *SoftwareInstallerStore) DeleteStaging(ctx context.Context, stagingID string) error {
	key := s.keyForStaging(stagingID)
	if _, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}); err != nil {
		if isS3NotFound(err) {
			return nil
		}
		return ctxerr.Wrapf(ctx, s.wrapAPIError(err), "deleting staged %s in S3 store", s.fileLabel)
	}
	return nil
}

func (s *SoftwareInstallerStore) copySource(key string) string {
	segments := strings.Split(key, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return url.PathEscape(s.bucket) + "/" + strings.Join(segments, "/")
}

func (s *SoftwareInstallerStore) copyObject(ctx context.Context, src, ifMatch, dst string) error {
	input := &s3.CopyObjectInput{
		Bucket:     &s.bucket,
		Key:        &dst,
		CopySource: new(s.copySource(src)),
	}
	if ifMatch != "" {
		input.CopySourceIfMatch = &ifMatch
	}
	if _, err := s.s3Client.CopyObject(ctx, input); err != nil {
		return s.copyError(ctx, err, "copying staged %s in S3 store", s.fileLabel)
	}
	return nil
}

// copyError wraps an error of a copy, turning a failed CopySourceIfMatch into
// fleet.ErrStagingChanged.
func (s *SoftwareInstallerStore) copyError(ctx context.Context, err error, format string, args ...any) error {
	if isS3PreconditionFailed(err) {
		err = fleet.ErrStagingChanged
	} else {
		err = s.wrapAPIError(err)
	}
	return ctxerr.Wrapf(ctx, err, format, args...)
}

func (s *SoftwareInstallerStore) multipartCopy(ctx context.Context, src, ifMatch, dst string, size int64) error {
	created, err := s.s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &s.bucket,
		Key:    &dst,
	})
	if err != nil {
		return ctxerr.Wrapf(ctx, s.wrapAPIError(err), "starting multipart copy of staged %s in S3 store", s.fileLabel)
	}
	uploadID := created.UploadId

	numParts := (size + multipartCopyPartSize - 1) / multipartCopyPartSize
	parts := make([]types.CompletedPart, numParts)
	copySource := s.copySource(src)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(multipartCopyConcurrency)
	for i := range numParts {
		start := i * multipartCopyPartSize
		end := min(start+multipartCopyPartSize, size) - 1
		partNumber := int32(i + 1) //nolint:gosec // bounded by S3's 10,000 parts
		g.Go(func() error {
			input := &s3.UploadPartCopyInput{
				Bucket:          &s.bucket,
				Key:             &dst,
				UploadId:        uploadID,
				PartNumber:      &partNumber,
				CopySource:      &copySource,
				CopySourceRange: new(fmt.Sprintf("bytes=%d-%d", start, end)),
			}
			if ifMatch != "" {
				input.CopySourceIfMatch = &ifMatch
			}
			res, err := s.s3Client.UploadPartCopy(gctx, input)
			if err != nil {
				return s.copyError(gctx, err, "copying part %d of staged %s in S3 store", partNumber, s.fileLabel)
			}
			if res.CopyPartResult == nil {
				return ctxerr.Errorf(gctx, "missing result for part %d of staged %s copy", partNumber, s.fileLabel)
			}
			parts[i] = types.CompletedPart{ETag: res.CopyPartResult.ETag, PartNumber: &partNumber}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		s.abortMultipartUpload(ctx, dst, uploadID)
		return err
	}

	if _, err := s.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          &s.bucket,
		Key:             &dst,
		UploadId:        uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: parts},
	}); err != nil {
		s.abortMultipartUpload(ctx, dst, uploadID)
		return ctxerr.Wrapf(ctx, s.wrapAPIError(err), "completing multipart copy of staged %s in S3 store", s.fileLabel)
	}
	return nil
}

// abortMultipartUpload is best effort; a bucket lifecycle rule for incomplete
// multipart uploads is the backstop.
func (s *SoftwareInstallerStore) abortMultipartUpload(ctx context.Context, key string, uploadID *string) {
	_, _ = s.s3Client.AbortMultipartUpload(context.WithoutCancel(ctx), &s3.AbortMultipartUploadInput{
		Bucket:   &s.bucket,
		Key:      &key,
		UploadId: uploadID,
	})
}

// isS3PreconditionFailed reports whether err is the response to a failed
// If-Match or CopySourceIfMatch condition.
func isS3PreconditionFailed(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "PreconditionFailed" {
		return true
	}
	var statusErr interface{ HTTPStatusCode() int }
	return errors.As(err, &statusErr) && statusErr.HTTPStatusCode() == http.StatusPreconditionFailed
}

func isS3NotFound(err error) bool {
	var (
		noSuchKey *types.NoSuchKey
		notFound  *types.NotFound
	)
	return errors.As(err, &noSuchKey) || errors.As(err, &notFound)
}
