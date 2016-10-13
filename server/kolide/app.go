package kolide

import "golang.org/x/net/context"

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
	NewOrgInfo(ctx context.Context, p OrgInfoPayload) (info *OrgInfo, err error)
	OrgInfo(ctx context.Context) (info *OrgInfo, err error)
	ModifyOrgInfo(ctx context.Context, p OrgInfoPayload) (info *OrgInfo, err error)
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

// ListOptions defines options related to paging and ordering to be used when
// listing objects
type ListOptions struct {
	// Which page to return (must be positive integer)
	Page uint
	// How many results per page (must be positive integer, 0 indicates
	// unlimited)
	PerPage uint
}
