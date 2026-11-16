package azure

import (
	"github.com/fleetdm/fleet/v4/server/config"
)

type SoftwareTitleIconStore struct {
	*commonFileStore
}

// NewSoftwareTitleIconStore creates a new instance with the given Azure config.
func NewSoftwareTitleIconStore(cfg config.AzureConfig) (*SoftwareTitleIconStore, error) {
	// software title icons use the same Azure config as software installers
	store, err := newAzureBlobStore(cfg.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &SoftwareTitleIconStore{
		&commonFileStore{
			azureBlobStore: store,
			pathPrefix:     "software-title-icons",
			fileLabel:      "software title icon",
		},
	}, nil
}
