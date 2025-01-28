package s3

import (
	"github.com/fleetdm/fleet/v4/server/config"
)

const softwareInstallersPrefix = "software-installers"

type SoftwareInstallerStore struct {
	*commonFileStore
}

// NewSoftwareInstallerStore creates a new instance with the given S3 config.
func NewSoftwareInstallerStore(config config.S3Config) (*SoftwareInstallerStore, error) {
	s3store, err := newS3store(config.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &SoftwareInstallerStore{
		&commonFileStore{
			s3store:    s3store,
			pathPrefix: softwareInstallersPrefix,
			fileLabel:  "software installer",
		},
	}, nil
}

// NewTestSoftwareInstallerStore is used in tests.
func NewTestSoftwareInstallerStore(conf config.S3Config) (*SoftwareInstallerStore, error) {
	store := &s3store{
		bucket: "test-bucket",
		cloudFrontConfig: &config.S3CloudFrontConfig{
			BaseURL:            conf.SoftwareInstallersCloudFrontURL,
			SigningPublicKeyID: conf.SoftwareInstallersCloudFrontURLSigningPublicKeyID,
			Signer:             conf.SoftwareInstallersCloudFrontSigner,
		},
	}
	return &SoftwareInstallerStore{
		&commonFileStore{
			s3store:    store,
			pathPrefix: softwareInstallersPrefix,
			fileLabel:  "software installer",
		},
	}, nil
}
