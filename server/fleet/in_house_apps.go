package fleet

import (
	"time"
)

const InHouseAppSignedURLExpiry = 5 * time.Minute

// InHouseAppInstallTokenTTL must cover the lifetime of a pending
// InstallApplication command (device may be offline). Matches
// BootstrapPackageSignedURLExpiry.
const InHouseAppInstallTokenTTL = 6 * time.Hour

type InHouseAppInstallTokenMetadata struct {
	Token           string    `db:"token"`
	SoftwareTitleID uint      `db:"software_title_id"`
	TeamID          uint      `db:"team_id"`
	HostID          uint      `db:"host_id"`
	ExpiresAt       time.Time `db:"expires_at"`
}

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
	// Configuration is the managed app configuration as raw XML bytes (iOS / iPadOS only).
	Configuration []byte
}
