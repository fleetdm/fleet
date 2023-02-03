package macoffice

import "time"

type ProductType int

const (
	OfficeSuite ProductType = iota
	Outlook
	Excel
	PowerPoint
	Word
	OneNote
)

type SecurityUpdate struct {
	Product       ProductType
	Vulnerability string
}

// OfficeRelease contains information about an Office release including security patches.
type OfficeRelease struct {
	Date            time.Time
	Version         string // Ths includes the Build ex: 16.69 (Build 23010700)
	SecurityUpdates []SecurityUpdate
}
