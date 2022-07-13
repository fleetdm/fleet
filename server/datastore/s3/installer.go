package s3

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const (
	desktopPath = "desktop"
	executable  = "fleet-osquery"
)

// Installer describes an installer in an S3 bucket
type Installer struct {
	enrollSecret string
	ext          string
	desktop      bool
}

// key builds an S3 key to search for the installer
func (p Installer) key() string {
	file := fmt.Sprintf("%s.%s", executable, p.ext)
	dir := ""
	if p.desktop {
		dir = desktopPath
	}
	return path.Join(p.enrollSecret, dir, file)
}

// InstallerStore contains methods to retrieve installers from S3
type InstallerStore struct {
	*s3Store
}

// NewInstallerStore creates a new instance with the given S3 config
func NewInstallerStore(config config.S3Config) (*InstallerStore, error) {
	s3Store, err := newS3Store(config)
	if err != nil {
		return nil, err
	}
	return &InstallerStore{s3Store}, nil
}

// Exists checks if an installer exists in the S3 bucket
func (i *InstallerStore) Exists(ctx context.Context, installer Installer) (bool, error) {
	key := installer.key()
	_, err := i.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get retrieves the requested installer from S3
func (i *InstallerStore) Get(ctx context.Context, installer Installer) (io.ReadCloser, error) {
	key := installer.key()
	req, err := i.s3client.GetObject(&s3.GetObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get installer from storage")
	}
	return req.Body, nil
}
