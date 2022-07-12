package s3

import (
	"context"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// InstallerParams describes the params used to locate an installer in an
// S3 bucket
type InstallerParams struct {
	secret  string
	ext     string
	desktop bool
}

func (p InstallerParams) buildKey() string {
	var dir string
	if p.desktop {
		dir = "desktop"
	}
	return path.Join(p.secret, dir, "fleet-osquery."+p.ext)
}

// InstallerStore contains methods to retrieve installers from S3
type InstallerStore struct {
	*datastore
}

// NewInstallerStore creates a new instance with the given S3 config
func NewInstallerStore(config config.S3Config) (*InstallerStore, error) {
	s3Store, err := newDatastore(config)
	if err != nil {
		return nil, err
	}

	return &InstallerStore{s3Store}, nil
}

// Exists checks if an installer exists in the S3 bucket
func (s *InstallerStore) Exists(ctx context.Context, params InstallerParams) bool {
	key := params.buildKey()
	_, err := s.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	return err != nil
}

// GetLink returns a pre-signed S3 link that can be used to download the
// installer
func (s *InstallerStore) GetLink(ctx context.Context, params InstallerParams) (string, error) {
	key := params.buildKey()
	req, _ := s.s3client.GetObjectRequest(&s3.GetObjectInput{Bucket: &s.bucket, Key: &key})

	url, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "presign S3 package")
	}
	return url, err
}
