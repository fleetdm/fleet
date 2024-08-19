package fleet

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type VPPAppID struct {
	// AdamID is a unique identifier assigned to each app in
	// the App Store, this value is managed by Apple.
	AdamID   string              `db:"adam_id" json:"app_store_id"`
	Platform AppleDevicePlatform `db:"platform" json:"platform"`
}

// VPPAppTeam contains extra metadata injected by fleet
type VPPAppTeam struct {
	VPPAppID

	SelfService bool `db:"self_service" json:"self_service"`
}

// VPPApp represents a VPP (Volume Purchase Program) application,
// this is used by Apple MDM to manage applications via Apple
// Business Manager.
type VPPApp struct {
	VPPAppTeam
	// BundleIdentifier is the unique bundle identifier of the
	// Application.
	BundleIdentifier string `db:"bundle_identifier" json:"bundle_identifier"`
	// IconURL is the URL of this App icon
	IconURL string `db:"icon_url" json:"icon_url"`
	// Name is the user-facing name of this app.
	Name string `db:"name" json:"name"`
	// LatestVersion is the latest version of this app.
	LatestVersion string `db:"latest_version" json:"latest_version"`
	TeamID        *uint  `db:"-" json:"-"`
	TitleID       uint   `db:"title_id" json:"-"`

	CreatedAt time.Time `db:"created_at" json:"-"`
	UpdatedAt time.Time `db:"updated_at" json:"-"`
}

// AuthzType implements authz.AuthzTyper.
func (v *VPPApp) AuthzType() string {
	return "installable_entity"
}

// VPPAppStoreApp contains the field required by the get software title
// endpoint to represent an App Store app (VPP app).
type VPPAppStoreApp struct {
	VPPAppID
	Name          string               `db:"name" json:"name"`
	LatestVersion string               `db:"latest_version" json:"latest_version"`
	IconURL       *string              `db:"icon_url" json:"icon_url"`
	Status        *VPPAppStatusSummary `db:"-" json:"status"`
	SelfService   bool                 `db:"self_service" json:"self_service"`
}

// VPPAppStatusSummary represents aggregated status metrics for a VPP app.
type VPPAppStatusSummary struct {
	// Installed is the number of hosts that have the VPP app installed.
	Installed uint `json:"installed" db:"installed"`
	// Pending is the number of hosts that have the VPP app pending installation.
	Pending uint `json:"pending" db:"pending"`
	// Failed is the number of hosts that have the VPP app installation failed.
	Failed uint `json:"failed" db:"failed"`
}

type VPPToken struct {
	ID               uint      `db:"id" json:"id"`
	Location         string    `db:"location" json:"location"`
	OrganizationName string    `db:"organization_name" json:"org_name"`
	RenewAt          time.Time `db:"renew_at" json:"renew_date"`
	Token            []byte    `db:"token" json:"-"`
	TeamID           *uint     `db:"team_id" json:"-"`
	NullTeamType     string    `db:"null_team_type" json:"-"`
	TeamName         string    `db:"team_name" json:"-"`

	// the following field is not in the vpp_tokens table, it must be queried
	// by a LEFT JOIN on the corresponding team, coalesced to "All teams" or
	// "no teams" or empty depending on the null_team_type enum.
	//
	// Currently a VPP token can only be linked to a single team, but using a
	// slice so the JSON response will already be able to support many if needed.
	Teams []string `db:"-" json:"teams"`
}

// ExtractToken extracts the metadata from the token as stored in the database,
// and returns the raw token that can be used directly with Apple's VPP API. If
// while extracting the token it notices that the metadata has changed, it will
// update t and return true as second return value, indicating that it changed
// and should be saved.
func (t *VPPToken) ExtractToken() (rawAppleToken string, didUpdateMetadata bool, err error) {
	var vppTokenData VPPTokenData
	if err := json.Unmarshal(t.Token, &vppTokenData); err != nil {
		return "", false, fmt.Errorf("unmarshaling VPP token data: %w", err)
	}

	vppTokenRawBytes, err := base64.StdEncoding.DecodeString(vppTokenData.Token)
	if err != nil {
		return "", false, fmt.Errorf("decoding raw vpp token data: %w", err)
	}

	var vppTokenRaw VPPTokenRaw
	if err := json.Unmarshal(vppTokenRawBytes, &vppTokenRaw); err != nil {
		return "", false, fmt.Errorf("unmarshaling raw vpp token data: %w", err)
	}

	exp, err := time.Parse("2006-01-02T15:04:05Z0700", vppTokenRaw.ExpDate)
	if err != nil {
		return "", false, fmt.Errorf("parsing vpp token expiration date: %w", err)
	}

	if vppTokenData.Location != t.Location {
		t.Location = vppTokenData.Location
		didUpdateMetadata = true
	}
	if vppTokenRaw.OrgName != t.OrganizationName {
		t.OrganizationName = vppTokenRaw.OrgName
		didUpdateMetadata = true
	}
	if !exp.Equal(t.RenewAt) {
		t.RenewAt = exp.UTC()
		didUpdateMetadata = true
	}

	return vppTokenRaw.Token, didUpdateMetadata, nil
}
