package macoffice

import "time"

type ProductType int

const (
	WholeSuite ProductType = iota
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

// OfficeReleaseNote contains information about an Office release including security patches.
type OfficeReleaseNote struct {
	Date            time.Time
	Version         string // Ths includes the Build ex: 16.69 (Build 23010700)
	SecurityUpdates []SecurityUpdate
}

func (or *OfficeReleaseNote) AddSecurityUpdate(pt ProductType, vuln string) {
	or.SecurityUpdates = append(or.SecurityUpdates, SecurityUpdate{
		Product:       pt,
		Vulnerability: vuln,
	})
}
