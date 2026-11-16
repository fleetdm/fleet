package azure

import (
	"github.com/fleetdm/fleet/v4/server/config"
)

const softwareInstallersPrefix = "software-installers"

type SoftwareInstallerStore struct {
	*commonFileStore
}

// NewSoftwareInstallerStore creates a new instance with the given Azure config.
func NewSoftwareInstallerStore(cfg config.AzureConfig) (*SoftwareInstallerStore, error) {
	store, err := newAzureBlobStore(cfg.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &SoftwareInstallerStore{
		&commonFileStore{
			azureBlobStore: store,
			pathPrefix:     softwareInstallersPrefix,
			fileLabel:      "software installer",
		},
	}, nil
}
