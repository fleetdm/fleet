package fleet

import "time"

// AppleSoftwareUpdateAssetClass identifies the OS family of a GDMF asset row.
type AppleSoftwareUpdateAssetClass string

const (
	AppleSoftwareUpdateAssetClassMacOS AppleSoftwareUpdateAssetClass = "macos"
	AppleSoftwareUpdateAssetClassIOS   AppleSoftwareUpdateAssetClass = "ios"
)

// AppleSoftwareUpdateAsset is a cached row from Apple's GDMF Software Lookup
// Service (https://gdmf.apple.com/v2/pmv).
type AppleSoftwareUpdateAsset struct {
	ID               uint                          `db:"id" json:"id"`
	Class            AppleSoftwareUpdateAssetClass `db:"class" json:"class"`
	ProductVersion   string                        `db:"product_version" json:"product_version"`
	Build            string                        `db:"build" json:"build"`
	PostingDate      *time.Time                    `db:"posting_date" json:"posting_date"`
	ExpirationDate   *time.Time                    `db:"expiration_date" json:"expiration_date"`
	SupportedDevices []byte                        `db:"supported_devices" json:"supported_devices"`
	FirstSeenAt      time.Time                     `db:"first_seen_at" json:"first_seen_at"`
	CreatedAt        time.Time                     `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time                     `db:"updated_at" json:"updated_at"`
}
