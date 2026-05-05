package failing

type FailingSoftwareInstallerStore struct {
	*commonFailingStore
}

func NewFailingSoftwareInstallerStore() *FailingSoftwareInstallerStore {
	return &FailingSoftwareInstallerStore{
		commonFailingStore: &commonFailingStore{
			Entity: "software installer",
		},
	}
}
