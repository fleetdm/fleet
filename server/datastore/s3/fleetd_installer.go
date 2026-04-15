package s3

import (
	"github.com/fleetdm/fleet/v4/server/config"
)

const fleetdInstallersPrefix = "fleetd-installers"

// FleetdInstallerStore implements fleet.FleetdInstallerStore using S3 storage.
type FleetdInstallerStore struct {
	*commonFileStore
}

// NewFleetdInstallerStore creates a new instance with the given S3 config.
// It reuses the software installers bucket configuration with a dedicated path prefix.
func NewFleetdInstallerStore(config config.S3Config) (*FleetdInstallerStore, error) {
	s3store, err := newS3Store(config.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &FleetdInstallerStore{
		&commonFileStore{
			s3store:    s3store,
			pathPrefix: fleetdInstallersPrefix,
			fileLabel:  "fleetd installer",

			gcs: isGCS(config.EndpointURL),
		},
	}, nil
}
