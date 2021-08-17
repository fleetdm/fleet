package fleet

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/kolide/kit/version"
)

// AppConfigStore contains method for saving and retrieving
// application configuration
type AppConfigStore interface {
	NewAppConfig(info *AppConfig) (*AppConfig, error)
	AppConfig() (*AppConfig, error)
	SaveAppConfig(info *AppConfig) error

	// VerifyEnrollSecret checks that the provided secret matches an active
	// enroll secret. If it is successfully matched, that secret is returned.
	// Otherwise an error is returned.
	VerifyEnrollSecret(secret string) (*EnrollSecret, error)
	// GetEnrollSecrets gets the enroll secrets for a team (or global if teamID is nil).
	GetEnrollSecrets(teamID *uint) ([]*EnrollSecret, error)
	// ApplyEnrollSecrets replaces the current enroll secrets for a team with the provided secrets.
	ApplyEnrollSecrets(teamID *uint, secrets []*EnrollSecret) error
}

// AppConfigService provides methods for configuring
// the Fleet application
type AppConfigService interface {
	NewAppConfig(ctx context.Context, p AppConfig) (info *AppConfig, err error)
	AppConfig(ctx context.Context) (info *AppConfig, err error)
	ModifyAppConfig(ctx context.Context, p []byte) (info *AppConfig, err error)

	// ApplyEnrollSecretSpec adds and updates the enroll secrets specified in
	// the spec.
	ApplyEnrollSecretSpec(ctx context.Context, spec *EnrollSecretSpec) error
	// GetEnrollSecretSpec gets the spec for the current enroll secrets.
	GetEnrollSecretSpec(ctx context.Context) (*EnrollSecretSpec, error)

	// CertificateChain returns the PEM encoded certificate chain for osqueryd TLS termination.
	// For cases where the connection is self-signed, the server will attempt to
	// connect using the InsecureSkipVerify option in tls.Config.
	CertificateChain(ctx context.Context) (cert []byte, err error)

	// SetupRequired returns whether the app config setup needs to be performed
	// (only when first initializing a Fleet server).
	SetupRequired(ctx context.Context) (bool, error)

	// Version returns version and build information.
	Version(ctx context.Context) (*version.Info, error)

	// License returns the licensing information.
	License(ctx context.Context) (*LicenseInfo, error)

	// LoggingConfig parses config.FleetConfig instance and returns a Logging.
	LoggingConfig(ctx context.Context) (*Logging, error)
}

// SMTP settings names returned from API, these map to SMTPAuthType and
// SMTPAuthMethod
const (
	AuthMethodNameCramMD5        = "authmethod_cram_md5"
	AuthMethodNameLogin          = "authmethod_login"
	AuthMethodNamePlain          = "authmethod_plain"
	AuthTypeNameUserNamePassword = "authtype_username_password"
	AuthTypeNameNone             = "authtype_none"
)

func (c AppConfig) AuthzType() string {
	return "app_config"
}

const (
	AppConfigKind = "config"
)

// ModifyAppConfigRequest contains application configuration information
// sent from front end and used to change app config elements.
type ModifyAppConfigRequest struct {
	// TestSMTP is this is set to true, the SMTP configuration will be tested
	// with the results of the test returned to caller. No config changes
	// will be applied.
	TestSMTP  bool      `json:"test_smtp"`
	AppConfig AppConfig `json:"app_config"`
}

// SSOSettings wire format for SSO settings
type SSOSettings struct {
	// EntityID is a uri that identifies this service provider
	EntityID string `json:"entity_id"`
	// IssuerURI is the uri that identifies the identity provider
	IssuerURI string `json:"issuer_uri"`
	// IDPImageURL is a link to a logo or other image that is used for UX
	IDPImageURL string `json:"idp_image_url"`
	// Metadata contains IDP metadata XML
	Metadata string `json:"metadata"`
	// MetadataURL is a URL provided by the IDP which can be used to download
	// metadata
	MetadataURL string `json:"metadata_url"`
	// IDPName is a human friendly name for the IDP
	IDPName string `json:"idp_name"`
	// EnableSSO flag to determine whether or not to enable SSO
	EnableSSO bool `json:"enable_sso"`
	// EnableSSOIdPLogin flag to determine whether or not to allow IdP-initiated
	// login.
	EnableSSOIdPLogin bool `json:"enable_sso_idp_login"`
}

// SMTPSettings is part of the AppConfig which defines the wire representation
// of the app config endpoints
type SMTPSettings struct {
	// SMTPEnabled indicates whether the user has selected that SMTP is
	// enabled in the UI.
	SMTPEnabled bool `json:"enable_smtp"`
	// SMTPConfigured is a flag that indicates if smtp has been successfully
	// tested with the settings provided by an admin user.
	SMTPConfigured bool `json:"configured"`
	// SMTPSenderAddress is the email address that will appear in emails sent
	// from Fleet
	SMTPSenderAddress string `json:"sender_address"`
	// SMTPServer is the host name of the SMTP server Fleet will use to send mail
	SMTPServer string `json:"server"`
	// SMTPPort port SMTP server will use
	SMTPPort uint `json:"port"`
	// SMTPAuthenticationType type of authentication for SMTP
	SMTPAuthenticationType string `json:"authentication_type"`
	// SMTPUserName must be provided if SMTPAuthenticationType is UserNamePassword
	SMTPUserName string `json:"user_name"`
	// SMTPPassword must be provided if SMTPAuthenticationType is UserNamePassword
	SMTPPassword string `json:"password"`
	// SMTPEnableSSLTLS whether to use SSL/TLS for SMTP
	SMTPEnableTLS bool `json:"enable_ssl_tls"`
	// SMTPAuthenticationMethod authentication method smtp server will use
	SMTPAuthenticationMethod string `json:"authentication_method"`

	// SMTPDomain optional domain for SMTP
	SMTPDomain string `json:"domain"`
	// SMTPVerifySSLCerts defaults to true but can be turned off if self signed
	// SSL certs are used by the SMTP server
	SMTPVerifySSLCerts bool `json:"verify_ssl_certs"`
	// SMTPEnableStartTLS detects of TLS is enabled on mail server and starts to use it (default true)
	SMTPEnableStartTLS bool `json:"enable_start_tls"`
}

// VulnerabilitySettings is part of the AppConfig which defines how fleet will behave
// while scanning for vulnerabilities in the host software
type VulnerabilitySettings struct {
	// DatabasesPath is the directory where fleet will store the different databases
	DatabasesPath string `json:"databases_path"`
}

// AppConfig
type AppConfig struct {
	OrgInfo            OrgInfo            `json:"org_info"`
	ServerSettings     ServerSettings     `json:"server_settings"`
	SMTPSettings       SMTPSettings       `json:"smtp_settings"`
	HostExpirySettings HostExpirySettings `json:"host_expiry_settings"`
	HostSettings       HostSettings       `json:"host_settings"`
	AgentOptions       *json.RawMessage   `json:"agent_options"`
	// SMTPTest is a flag that if set will cause the server to test email configuration
	SMTPTest bool `json:"smtp_test,omitempty"`
	// SSOSettings is single sign on settings
	SSOSettings SSOSettings `json:"sso_settings"`

	// VulnerabilitySettings defines how fleet will behave while scanning for vulnerabilities in the host software
	VulnerabilitySettings VulnerabilitySettings `json:"vulnerability_settings"`
}

func (ac *AppConfig) ApplyDefaultsForNewInstalls() {
	ac.ServerSettings.EnableAnalytics = true
	ac.HostSettings.EnableHostUsers = true
	ac.SMTPSettings.SMTPPort = 587
	ac.SMTPSettings.SMTPEnableStartTLS = true
	ac.SMTPSettings.SMTPAuthenticationType = AuthTypeNameUserNamePassword
	ac.SMTPSettings.SMTPAuthenticationMethod = AuthMethodNamePlain
	ac.SMTPSettings.SMTPVerifySSLCerts = true
	ac.SMTPSettings.SMTPEnableTLS = true
}

// OrgInfo contains general info about the organization using Fleet.
type OrgInfo struct {
	OrgName    string `json:"org_name"`
	OrgLogoURL string `json:"org_logo_url"`
}

// ServerSettings contains general settings about the Fleet application.
type ServerSettings struct {
	ServerURL         string `json:"server_url"`
	LiveQueryDisabled bool   `json:"live_query_disabled"`
	EnableAnalytics   bool   `json:"enable_analytics"`
}

// HostExpirySettings contains settings pertaining to automatic host expiry.
type HostExpirySettings struct {
	HostExpiryEnabled bool `json:"host_expiry_enabled"`
	HostExpiryWindow  int  `json:"host_expiry_window"`
}

type HostSettings struct {
	EnableHostUsers         bool             `json:"enable_host_users"`
	EnableSoftwareInventory bool             `json:"enable_software_inventory"`
	AdditionalQueries       *json.RawMessage `json:"additional_queries,omitempty"`
}

type OrderDirection int

const (
	OrderAscending OrderDirection = iota
	OrderDescending

	// PerPageUnlimited is the value to pass to PerPage when we want
	// "unlimited". If we ever find this limit to be too low, congratulations on
	// incredible growth of the product!
	PerPageUnlimited uint = 9999999
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
	// MatchQuery is the query string to match against columns of the entity
	// (varies depending on entity, eg. hostname, IP address for hosts).
	// Handling for this parameter must be implemented separately for each type.
	MatchQuery string
}

// EnrollSecret contains information about an enroll secret, name, and active
// status. Enroll secrets are used for osquery authentication.
type EnrollSecret struct {
	// Secret is the actual secret key.
	Secret string `json:"secret" db:"secret"`
	// CreatedAt is the time this enroll secret was first added.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// TeamID is the ID for the associated team. If no ID is set, then this is a
	// global enroll secret.
	TeamID *uint `json:"team_id,omitempty" db:"team_id"`
}

func (e *EnrollSecret) AuthzType() string {
	return "enroll_secret"
}

const (
	EnrollSecretKind          = "enroll_secret"
	EnrollSecretDefaultLength = 24
)

// EnrollSecretSpec is the fleetctl spec type for enroll secrets.
type EnrollSecretSpec struct {
	// Secrets is the list of enroll secrets.
	Secrets []*EnrollSecret `json:"secrets"`
}

const (
	// TierBasic is Fleet Basic aka the paid license.
	TierBasic = "basic"
	// TierCore is Fleet Core aka the free license.
	TierCore = "core"
)

// LicenseInfo contains information about the Fleet license.
type LicenseInfo struct {
	// Tier is the license tier (currently "core" or "basic")
	Tier string `json:"tier"`
	// Organization is the name of the licensed organization.
	Organization string `json:"organization,omitempty"`
	// DeviceCount is the number of licensed devices.
	DeviceCount int `json:"device_count,omitempty"`
	// Expiration is when the license expires.
	Expiration time.Time `json:"expiration,omitempty"`
	// Note is any additional terms of license
	Note string `json:"note,omitempty"`
}

type Logging struct {
	Debug  bool          `json:"debug"`
	Json   bool          `json:"json"`
	Result LoggingPlugin `json:"result"`
	Status LoggingPlugin `json:"status"`
}

type LoggingPlugin struct {
	Plugin string      `json:"plugin"`
	Config interface{} `json:"config"`
}

type FilesystemConfig struct {
	config.FilesystemConfig
}

type PubSubConfig struct {
	config.PubSubConfig
}

// FirehoseConfig shadows config.FirehoseConfig only exposing a subset of fields
type FirehoseConfig struct {
	Region       string `json:"region"`
	StatusStream string `json:"status_stream"`
	ResultStream string `json:"result_stream"`
}

// KinesisConfig shadows config.KinesisConfig only exposing a subset of fields
type KinesisConfig struct {
	Region       string `json:"region"`
	StatusStream string `json:"status_stream"`
	ResultStream string `json:"result_stream"`
}

// LambdaConfig shadows config.LambdaConfig only exposing a subset of fields
type LambdaConfig struct {
	Region         string `json:"region"`
	StatusFunction string `json:"status_function"`
	ResultFunction string `json:"result_function"`
}
