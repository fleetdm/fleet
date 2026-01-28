package failing

type FailingSoftwareTitleIconStore struct {
	*commonFailingStore
}

func NewFailingSoftwareTitleIconStore() *FailingSoftwareTitleIconStore {
	return &FailingSoftwareTitleIconStore{
		commonFailingStore: &commonFailingStore{
			Entity: "software title icon",
		},
	}
}
