package macoffice

import (
	"strings"
	"time"
)

const RelNotesURL = "https://learn.microsoft.com/en-us/officeupdates/release-notes-office-for-mac"

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

// ReleaseNote contains information about an Office release including security patches.
type ReleaseNote struct {
	Date            time.Time
	Version         string // Ths includes the Build ex: 16.69 (Build 23010700)
	SecurityUpdates []SecurityUpdate
}

func (or *ReleaseNote) AddSecurityUpdate(pt ProductType, vuln string) {
	or.SecurityUpdates = append(or.SecurityUpdates, SecurityUpdate{
		Product:       pt,
		Vulnerability: vuln,
	})
}

func GetProductTypeFromBundleId(bundle string) (ProductType, bool) {
	b := strings.ToLower(bundle)
	switch {
	case strings.HasPrefix(b, "com.microsoft.powerpoint"):
		return PowerPoint, true
	case strings.HasPrefix(b, "com.microsoft.word"):
		return Word, true
	case strings.HasPrefix(b, "com.microsoft.excel"):
		return Excel, true
	case strings.HasPrefix(b, "com.microsoft.onenote"):
		return OneNote, true
	case strings.HasPrefix(b, "com.microsoft.outlook"):
		return Outlook, true
	}
	return WholeSuite, false
}
