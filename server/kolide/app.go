package kolide

import "context"

// AppConfigStore contains method for saving and retrieving
// application configuration
type AppConfigStore interface {
	NewOrgInfo(info *OrgInfo) (*OrgInfo, error)
	OrgInfo() (*OrgInfo, error)
	SaveOrgInfo(info *OrgInfo) error
}

// AppConfigService provides methods for configuring
// the Kolide application
type AppConfigService interface {
	NewOrgInfo(ctx context.Context, p OrgInfoPayload) (*OrgInfo, error)
	OrgInfo(ctx context.Context) (*OrgInfo, error)
	ModifyOrgInfo(ctx context.Context, p OrgInfoPayload) (*OrgInfo, error)
}

// OrgInfo holds information about the current
// organization using Kolide
type OrgInfo struct {
	ID         uint `gorm:"primary_key"`
	OrgName    string
	OrgLogoURL string
}

// OrgInfoPayload is used to accept
// OrgInfo modifications by a client
type OrgInfoPayload struct {
	OrgName    *string `json:"org_name"`
	OrgLogoURL *string `json:"org_logo_url"`
}
