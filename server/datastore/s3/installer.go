package s3

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	desktopPath = "desktop"
	executable  = "fleet-osquery"
)

// InstallerStore contains methods to retrieve installers from S3
type InstallerStore struct {
	*s3store
}

// NewInstallerStore creates a new instance with the given S3 config
func NewInstallerStore(config config.S3Config) (*InstallerStore, error) {
	s3Store, err := newS3store(config)
	if err != nil {
		return nil, err
	}
	return &InstallerStore{s3Store}, nil
}

// keyForInstaller builds an S3 key to search for the installer
func keyForInstaller(i fleet.Installer) string {
	file := fmt.Sprintf("%s.%s", executable, i.Kind)
	dir := ""
	if i.Desktop {
		dir = desktopPath
	}
	return path.Join(i.EnrollSecret, dir, file)
}

// Exists checks if an installer exists in the S3 bucket
func (i *InstallerStore) Exists(ctx context.Context, installer fleet.Installer) (bool, error) {
	key := keyForInstaller(installer)
	_, err := i.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get retrieves the requested installer from S3
func (i *InstallerStore) Get(ctx context.Context, installer fleet.Installer) (io.ReadCloser, error) {
	key := keyForInstaller(installer)
	req, err := i.s3client.GetObject(&s3.GetObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get installer from storage")
	}
	return req.Body, nil
}
