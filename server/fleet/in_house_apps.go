package fleet

type InHouseAppPayload struct {
	TeamID          *uint
	Title           string // app name
	Filename        string
	BundleID        string
	StorageID       string
	Platform        string
	ValidatedLabels *LabelIdentsWithScope
	CategoryIDs     []uint
	Version         string
	SelfService     bool
}
