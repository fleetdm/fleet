package s3

import "github.com/fleetdm/fleet/v4/server/config"

const bootstrapPackagePrefix = "bootstrap-packages"

type BootstrapPackageStore struct {
	*commonFileStore
}

// NewBootstrapPackageStore creates a new instance with the given S3 config.
func NewBootstrapPackageStore(config config.S3Config) (*BootstrapPackageStore, error) {
	// bootstrap packages use the same S3 config as software installers
	s3store, err := newS3store(config.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &BootstrapPackageStore{
		&commonFileStore{
			s3store:    s3store,
			pathPrefix: bootstrapPackagePrefix,
			fileLabel:  "bootstrap package",
		},
	}, nil
}
