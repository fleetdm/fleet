package s3

import (
	"context"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const softwareInstallersPrefix = "software-installers"

// SoftwareInstallerStore implements the fleet.SoftwareInstallerStore to store
// and retrieve software installers from S3.
type SoftwareInstallerStore struct {
	*s3store
}

// NewSoftwareInstallerStore creates a new instance with the given S3 config.
func NewSoftwareInstallerStore(config config.S3Config) (*SoftwareInstallerStore, error) {
	s3store, err := newS3store(config.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &SoftwareInstallerStore{s3store}, nil
}

// Get retrieves the requested software installer from S3.
// It is important that the caller closes the reader when done.
func (i *SoftwareInstallerStore) Get(ctx context.Context, installerID string) (io.ReadCloser, int64, error) {
	key := i.keyForInstaller(installerID)

	req, err := i.s3client.GetObject(&s3.GetObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket, "NotFound":
				return nil, int64(0), installerNotFoundError{}
			}
		}
		return nil, int64(0), ctxerr.Wrap(ctx, err, "retrieving software installer from S3 store")
	}
	return req.Body, *req.ContentLength, nil
}

// Put uploads a software installer to S3.
func (i *SoftwareInstallerStore) Put(ctx context.Context, installerID string, content io.ReadSeeker) error {
	key := i.keyForInstaller(installerID)
	_, err := i.s3client.PutObject(&s3.PutObjectInput{
		Bucket: &i.bucket,
		Body:   content,
		Key:    &key,
	})
	return err
}

// Exists checks if a software installer exists in the S3 bucket for the ID.
func (i *SoftwareInstallerStore) Exists(ctx context.Context, installerID string) (bool, error) {
	key := i.keyForInstaller(installerID)

	_, err := i.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket, "NotFound":
				return false, nil
			}
		}
		return false, ctxerr.Wrap(ctx, err, "checking existence of software installer in S3 store")
	}
	return true, nil
}

func (i *SoftwareInstallerStore) Cleanup(ctx context.Context, usedInstallerIDs []string) (int, error) {
	usedSet := make(map[string]struct{}, len(usedInstallerIDs))
	for _, id := range usedInstallerIDs {
		usedSet[id] = struct{}{}
	}

	// ListObjectsV2 defaults to a max of 1000 keys, which is sufficient for the
	// cleanup task - if more software installers are present, the next run will
	// get another 1000 and will periodically complete the cleanups.
	//
	// Iterating over all pages would potentially take a long time and would make
	// it more likely that a conflict arises, where an unused software installer
	// becomes used again. This approach makes it only two API requests between
	// the read of used installers and the deletions.
	prefix := path.Join(i.prefix, softwareInstallersPrefix)
	page, err := i.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &i.bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "listing software installers in S3 store")
	}

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

	res, err := i.s3client.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: &i.bucket,
		Delete: &s3.Delete{
			Objects: toDeleteKeys,
		},
	})
	return len(res.Deleted), ctxerr.Wrap(ctx, err, "deleting software installers in S3 store")
}

// keyForInstaller builds an S3 key to identify the software installer.
func (i *SoftwareInstallerStore) keyForInstaller(installerID string) string {
	return path.Join(i.prefix, softwareInstallersPrefix, installerID)
}
