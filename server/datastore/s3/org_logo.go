package s3

import (
	"context"
	"errors"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type OrgLogoStore struct {
	*commonFileStore
}

// NewOrgLogoStore reuses the software-installers S3 config (same bucket,
// distinct prefix) — see s3.NewSoftwareTitleIconStore for the precedent.
func NewOrgLogoStore(cfg config.S3Config) (*OrgLogoStore, error) {
	s3store, err := newS3Store(cfg.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &OrgLogoStore{
		&commonFileStore{
			s3store:    s3store,
			pathPrefix: "org-logos",
			fileLabel:  "org logo",
			gcs:        isGCS(cfg.EndpointURL),
		},
	}, nil
}

// Put stores the logo bytes for mode under <bucket>/<prefix>/org-logos/<mode>.
func (s *OrgLogoStore) Put(ctx context.Context, mode fleet.OrgLogoMode, content io.ReadSeeker) error {
	return s.commonFileStore.Put(ctx, string(mode), content)
}

func (s *OrgLogoStore) Get(ctx context.Context, mode fleet.OrgLogoMode) (io.ReadCloser, int64, error) {
	return s.commonFileStore.Get(ctx, string(mode))
}

func (s *OrgLogoStore) Exists(ctx context.Context, mode fleet.OrgLogoMode) (bool, error) {
	return s.commonFileStore.Exists(ctx, string(mode))
}

func (s *OrgLogoStore) Delete(ctx context.Context, mode fleet.OrgLogoMode) error {
	key := path.Join(s.prefix, s.pathPrefix, string(mode))
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		var notFound *types.NotFound
		if errors.As(err, &noSuchKey) || errors.As(err, &notFound) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "deleting org logo from S3 store")
	}
	return nil
}
