package s3

import (
	"github.com/fleetdm/fleet/v4/server/config"
)

type SoftwareTitleIconStore struct {
	*commonFileStore
}

func NewSoftwareTitleIconStore(config config.S3Config) (*SoftwareTitleIconStore, error) {
	// software title icons use the same S3 config as software installers
	s3store, err := newS3Store(config.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &SoftwareTitleIconStore{
		&commonFileStore{
			s3store:    s3store,
			pathPrefix: "software-title-icons",
			fileLabel:  "software title icon",

			gcs: isGCS(config.EndpointURL),
		},
	}, nil
}
