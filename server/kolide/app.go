package kolide

import "golang.org/x/net/context"

// AppConfigStore contains method for saving and retrieving
// application configuration
type AppConfigStore interface {
	NewAppConfig(info *AppConfig) (*AppConfig, error)
	AppConfig() (*AppConfig, error)
	SaveAppConfig(info *AppConfig) error
}

// AppConfigService provides methods for configuring
// the Kolide application
type AppConfigService interface {
	NewAppConfig(ctx context.Context, p AppConfigPayload) (info *AppConfig, err error)
	AppConfig(ctx context.Context) (info *AppConfig, err error)
	ModifyAppConfig(ctx context.Context, p AppConfigPayload) (info *AppConfig, err error)
}

// AppConfig holds configuration about the Kolide application.
// AppConfig data can be managed by a Kolide API user.
type AppConfig struct {
	ID              uint `gorm:"primary_key"`
	OrgName         string
	OrgLogoURL      string
	KolideServerURL string
}

// AppConfigPayload contains request and response format of
// the AppConfig struct.
type AppConfigPayload struct {
	OrgInfo        *OrgInfo        `json:"org_info,omitempty"`
	ServerSettings *ServerSettings `json:"server_settings,omitempty"`
}

// OrgInfo contains general info about the organization using Kolide.
type OrgInfo struct {
	OrgName    *string `json:"org_name,omitempty"`
	OrgLogoURL *string `json:"org_logo_url,omitempty"`
}

// ServerSettings contains general settings about the kolide App.
type ServerSettings struct {
	KolideServerURL *string `json:"kolide_server_url,omitempty"`
}

type OrderDirection int

const (
	OrderAscending OrderDirection = iota
	OrderDescending
)

// ListOptions defines options related to paging and ordering to be used when
// listing objects
type ListOptions struct {
	// Which page to return (must be positive integer)
	Page uint
	// How many results per page (must be positive integer, 0 indicates
	// unlimited)
	PerPage uint
	// Key to use for ordering
	OrderKey string
	// Direction of ordering
	OrderDirection OrderDirection
}
