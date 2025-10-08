package fleet

type InHouseAppPayload struct {
	TeamID          *uint
	Name            string
	BundleID        string
	StorageID       string
	Platform        string
	ValidatedLabels *LabelIdentsWithScope
}
