package azure

import "github.com/fleetdm/fleet/v4/server/config"

const bootstrapPackagePrefix = "bootstrap-packages"

type BootstrapPackageStore struct {
	*commonFileStore
}

// NewBootstrapPackageStore creates a new instance with the given Azure config.
func NewBootstrapPackageStore(cfg config.AzureConfig) (*BootstrapPackageStore, error) {
	// bootstrap packages use the same Azure config as software installers
	store, err := newAzureBlobStore(cfg.SoftwareInstallersToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &BootstrapPackageStore{
		&commonFileStore{
			azureBlobStore: store,
			pathPrefix:     bootstrapPackagePrefix,
			fileLabel:      "bootstrap package",
		},
	}, nil
}
