package failing

// FailingFleetdInstallerStore is an implementation of FleetdInstallerStore
// that fails all operations. It is used when S3 is not configured and the
// local filesystem store could not be set up.
type FailingFleetdInstallerStore struct {
	*commonFailingStore
}

func NewFailingFleetdInstallerStore() *FailingFleetdInstallerStore {
	return &FailingFleetdInstallerStore{
		commonFailingStore: &commonFailingStore{
			Entity: "fleetd installer",
		},
	}
}
