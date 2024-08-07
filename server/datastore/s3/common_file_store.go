package s3

import (
	"context"
	"errors"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// commonFileStore implements the common Get, Put, Exists and Cleanup
// operations typical for storage of files in the SoftwareInstallers S3 bucket
// configuration. It is used by the SoftwareInstallerStore and the
// BootstrapPackageStore. The only variable thing is the path prefix inside
// the configured bucket, e.g. for software installers it is:
//
//	<bucket>/<prefix>/software-installers/<fileID>
//
// and for the bootstrap packages it is:
//
//	<bucket>/<prefix>/bootstrap-packages/<fileID>
type commonFileStore struct {
	*s3store
	pathPrefix string
	fileLabel  string // how to call the file in error messages
}

// Get retrieves the requested file from S3.
// It is important that the caller closes the reader when done.
func (s *commonFileStore) Get(ctx context.Context, fileID string) (io.ReadCloser, int64, error) {
	key := s.keyForFile(fileID)

	req, err := s.s3client.GetObject(&s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket, "NotFound":
				return nil, int64(0), installerNotFoundError{}
			}
		}
		return nil, int64(0), ctxerr.Wrapf(ctx, err, "retrieving %s from S3 store", s.fileLabel)
	}
	return req.Body, *req.ContentLength, nil
}

// Put uploads a file to S3.
func (s *commonFileStore) Put(ctx context.Context, fileID string, content io.ReadSeeker) error {
	if fileID == "" {
		return errors.New("S3 file identifier is empty")
	}

	key := s.keyForFile(fileID)
	_, err := s.s3client.PutObject(&s3.PutObjectInput{
		Bucket: &s.bucket,
		Body:   content,
		Key:    &key,
	})
	return err
}

// Exists checks if a file exists in the S3 bucket for the ID.
func (s *commonFileStore) Exists(ctx context.Context, fileID string) (bool, error) {
	key := s.keyForFile(fileID)

	_, err := s.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket, "NotFound":
				return false, nil
			}
		}
		return false, ctxerr.Wrapf(ctx, err, "checking existence of %s in S3 store", s.fileLabel)
	}
	return true, nil
}

func (s *commonFileStore) Cleanup(ctx context.Context, usedFileIDs []string) (int, error) {
	usedSet := make(map[string]struct{}, len(usedFileIDs))
	for _, id := range usedFileIDs {
		usedSet[id] = struct{}{}
	}

	// ListObjectsV2 defaults to a max of 1000 keys, which is sufficient for the
	// cleanup task - if more files are present, the next run will get another
	// 1000 and will periodically complete the cleanups.
	//
	// Iterating over all pages would potentially take a long time and would make
	// it more likely that a conflict arises, where an unused file becomes used
	// again. This approach makes it only two API requests between the read of
	// used files and the deletions.
	prefix := path.Join(s.prefix, s.pathPrefix)
	page, err := s.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &s.bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "listing %s in S3 store", s.fileLabel)
	}

	// TODO(mna): there is an inherent risk that we could delete files that were
	// added between the query to list used IDs and now. We could minimize that
	// risk by checking that the S3 file is older than a few seconds.
	var toDeleteKeys []*s3.ObjectIdentifier
	for _, item := range page.Contents {
		if item.Key == nil {
			continue
		}
		if _, ok := usedSet[path.Base(*item.Key)]; ok {
			continue
		}
		toDeleteKeys = append(toDeleteKeys, &s3.ObjectIdentifier{Key: item.Key})
	}

	if len(toDeleteKeys) == 0 {
		return 0, nil
	}

	res, err := s.s3client.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: &s.bucket,
		Delete: &s3.Delete{
			Objects: toDeleteKeys,
		},
	})
	return len(res.Deleted), ctxerr.Wrapf(ctx, err, "deleting %s in S3 store", s.fileLabel)
}

// keyForFile builds an S3 key to identify the file.
func (s *commonFileStore) keyForFile(fileID string) string {
	return path.Join(s.prefix, s.pathPrefix, fileID)
}
