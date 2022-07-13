package s3

import (
	"context"
	"fmt"
	"path"
	"time"

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
	secret  string
	ext     string
	desktop bool
}

// key builds an S3 key to search for the installer
func (p Installer) key() string {
	file := fmt.Sprintf("%s.%s", executable, p.ext)
	dir := ""
	if p.desktop {
		dir = desktopPath
	}
	return path.Join(p.secret, dir, file)
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

// CanAccess checks if an installer exists in the S3 bucket
func (i *InstallerStore) CanAccess(ctx context.Context, installer Installer) bool {
	key := installer.key()
	_, err := i.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &i.bucket, Key: &key})
	return err != nil
}

// GetLink returns a pre-signed S3 link that can be used to download the
// installer
func (i *InstallerStore) GetLink(ctx context.Context, installer Installer) (string, error) {
	key := installer.key()
	req, _ := i.s3client.GetObjectRequest(&s3.GetObjectInput{Bucket: &i.bucket, Key: &key})

	url, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "presigned link for installer")
	}
	return url, nil
}
