package fleet

type InHouseAppPayload struct {
	TeamID          *uint
	Filename        string
	BundleID        string
	StorageID       string
	Platform        string
	ValidatedLabels *LabelIdentsWithScope
	Version         string
	SelfService     bool
}
