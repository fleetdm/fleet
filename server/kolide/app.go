package kolide

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kolide/kit/version"
)

// AppConfigStore contains method for saving and retrieving
// application configuration
type AppConfigStore interface {
	NewAppConfig(info *AppConfig) (*AppConfig, error)
	AppConfig() (*AppConfig, error)
	SaveAppConfig(info *AppConfig) error

	// VerifyEnrollSecret checks that the provided secret matches an active
	// enroll secret. If it is successfully matched, the name of the secret is
	// returned. Otherwise an error is returned.
	VerifyEnrollSecret(secret string) (string, error)
	// ApplyEnrollSecretSpec adds and updates the enroll secrets specified in
	// the spec.
	ApplyEnrollSecretSpec(spec *EnrollSecretSpec) error
	// GetEnrollSecretSpec gets the spec for the current enroll secrets.
	GetEnrollSecretSpec() (*EnrollSecretSpec, error)
}

// AppConfigService provides methods for configuring
// the Fleet application
type AppConfigService interface {
	NewAppConfig(ctx context.Context, p AppConfigPayload) (info *AppConfig, err error)
	AppConfig(ctx context.Context) (info *AppConfig, err error)
	ModifyAppConfig(ctx context.Context, p AppConfigPayload) (info *AppConfig, err error)
	SendTestEmail(ctx context.Context, config *AppConfig) error

	// ApplyEnrollSecretSpec adds and updates the enroll secrets specified in
	// the spec.
	ApplyEnrollSecretSpec(ctx context.Context, spec *EnrollSecretSpec) error
	// GetEnrollSecretSpec gets the spec for the current enroll secrets.
	GetEnrollSecretSpec(ctx context.Context) (*EnrollSecretSpec, error)

	// Certificate returns the PEM encoded certificate chain for osqueryd TLS termination.
	// For cases where the connection is self-signed, the server will attempt to
	// connect using the InsecureSkipVerify option in tls.Config.
	CertificateChain(ctx context.Context) (cert []byte, err error)

	// Version returns version and build information.
	Version(ctx context.Context) (*version.Info, error)
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

type SMTPAuthType int

const (
	AuthTypeUserNamePassword SMTPAuthType = iota
	AuthTypeNone
)

func (a SMTPAuthType) String() string {
	switch a {
	case AuthTypeUserNamePassword:
		return AuthTypeNameUserNamePassword
	case AuthTypeNone:
		return AuthTypeNameNone
	default:
		return ""
	}
}

type SMTPAuthMethod int

const (
	AuthMethodPlain SMTPAuthMethod = iota
	AuthMethodCramMD5
	AuthMethodLogin
)

func (m SMTPAuthMethod) String() string {
	switch m {
	case AuthMethodPlain:
		return AuthMethodNamePlain
	case AuthMethodCramMD5:
		return AuthMethodNameCramMD5
	case AuthMethodLogin:
		return AuthMethodNameLogin
	default:
		return ""
	}
}

// AppConfig holds configuration about the Fleet application.
// AppConfig data can be managed by a Fleet API user.
type AppConfig struct {
	ID              uint
	OrgName         string `db:"org_name"`
	OrgLogoURL      string `db:"org_logo_url"`
	KolideServerURL string `db:"kolide_server_url"`

	// SMTPConfigured is a flag that indicates if smtp has been successfully
	// tested with the settings provided by an admin user.
	SMTPConfigured bool `db:"smtp_configured"`
	// SMTPSenderAddress is the email address that will appear in emails sent
	// from Fleet
	SMTPSenderAddress string `db:"smtp_sender_address"`
	// SMTPServer is the host name of the SMTP server Fleet will use to send mail
	SMTPServer string `db:"smtp_server"`
	// SMTPPort port SMTP server will use
	SMTPPort uint `db:"smtp_port"`
	// SMTPAuthenticationType type of authentication for SMTP
	SMTPAuthenticationType SMTPAuthType `db:"smtp_authentication_type"`
	// SMTPUserName must be provided if SMTPAuthenticationType is UserNamePassword
	SMTPUserName string `db:"smtp_user_name"`
	// SMTPPassword must be provided if SMTPAuthenticationType is UserNamePassword
	SMTPPassword string `db:"smtp_password"`
	// SMTPEnableSSLTLS whether to use SSL/TLS for SMTP
	SMTPEnableTLS bool `db:"smtp_enable_ssl_tls"`
	// SMTPAuthenticationMethod authentication method smtp server will use
	SMTPAuthenticationMethod SMTPAuthMethod `db:"smtp_authentication_method"`

	// SMTPDomain optional domain for SMTP
	SMTPDomain string `db:"smtp_domain"`
	// SMTPVerifySSLCerts defaults to true but can be turned off if self signed
	// SSL certs are used by the SMTP server
	SMTPVerifySSLCerts bool `db:"smtp_verify_ssl_certs"`
	// SMTPEnableStartTLS detects of TLS is enabled on mail server and starts to use it (default true)
	SMTPEnableStartTLS bool `db:"smtp_enable_start_tls"`
	// EntityID is a uri that identifies this service provider
	EntityID string `db:"entity_id"`
	// IssuerURI is the uri that identifies the identity provider
	IssuerURI string `db:"issuer_uri"`
	// IDPImageURL is a link to a logo or other image that is used for UX
	IDPImageURL string `db:"idp_image_url"`
	// Metadata contains IDP metadata XML
	Metadata string `db:"metadata"`
	// MetadataURL is a URL provided by the IDP which can be used to download
	// metadata
	MetadataURL string `db:"metadata_url"`
	// IDPName is a human friendly name for the IDP
	IDPName string `db:"idp_name"`
	// EnableSSO flag to determine whether or not to enable SSO
	EnableSSO bool `db:"enable_sso"`
	// EnableSSO flag to determine whether or not to enable SSO
	EnableSSOIdPLogin bool `db:"enable_sso_idp_login"`
	// FIMInterval defines the interval when file integrity checks will occur
	FIMInterval int `db:"fim_interval"`
	// FIMFileAccess defines the FIMSections which will be monitored for file access events as a JSON formatted array
	FIMFileAccesses string `db:"fim_file_accesses"`

	// HostExpiryEnabled defines whether automatic host cleanup is enabled.
	HostExpiryEnabled bool `db:"host_expiry_enabled"`
	// HostExpiryWindow defines a number in days after which a host will be removed if it has not communicated with Fleet.
	HostExpiryWindow int `db:"host_expiry_window"`

	// LiveQueryDisabled defines whether live queries are disabled.
	LiveQueryDisabled bool `db:"live_query_disabled"`

	// AdditionalQueries is the set of additional queries that should be run
	// when collecting details from hosts.
	AdditionalQueries *json.RawMessage `db:"additional_queries"`
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

// SSOSettingsPayload wire format for SSO settings
type SSOSettingsPayload struct {
	// EntityID is a uri that identifies this service provider
	EntityID *string `json:"entity_id"`
	// IssuerURI is the uri that identifies the identity provider
	IssuerURI *string `json:"issuer_uri"`
	// IDPImageURL is a link to a logo or other image that is used for UX
	IDPImageURL *string `json:"idp_image_url"`
	// Metadata contains IDP metadata XML
	Metadata *string `json:"metadata"`
	// MetadataURL is a URL provided by the IDP which can be used to download
	// metadata
	MetadataURL *string `json:"metadata_url"`
	// IDPName is a human friendly name for the IDP
	IDPName *string `json:"idp_name"`
	// EnableSSO flag to determine whether or not to enable SSO
	EnableSSO *bool `json:"enable_sso"`
	// EnableSSOIdPLogin flag to determine whether or not to allow IdP-initiated
	// login.
	EnableSSOIdPLogin *bool `json:"enable_sso_idp_login"`
}

// SMTPSettingsPayload is part of the AppConfigPayload which defines the wire representation
// of the app config endpoints
type SMTPSettingsPayload struct {
	// SMTPEnabled indicates whether the user has selected that SMTP is
	// enabled in the UI.
	SMTPEnabled *bool `json:"enable_smtp"`
	// SMTPConfigured is a flag that indicates if smtp has been successfully
	// tested with the settings provided by an admin user.
	SMTPConfigured *bool `json:"configured"`
	// SMTPSenderAddress is the email address that will appear in emails sent
	// from Fleet
	SMTPSenderAddress *string `json:"sender_address"`
	// SMTPServer is the host name of the SMTP server Fleet will use to send mail
	SMTPServer *string `json:"server"`
	// SMTPPort port SMTP server will use
	SMTPPort *uint `json:"port"`
	// SMTPAuthenticationType type of authentication for SMTP
	SMTPAuthenticationType *string `json:"authentication_type"`
	// SMTPUserName must be provided if SMTPAuthenticationType is UserNamePassword
	SMTPUserName *string `json:"user_name"`
	// SMTPPassword must be provided if SMTPAuthenticationType is UserNamePassword
	SMTPPassword *string `json:"password"`
	// SMTPEnableSSLTLS whether to use SSL/TLS for SMTP
	SMTPEnableTLS *bool `json:"enable_ssl_tls"`
	// SMTPAuthenticationMethod authentication method smtp server will use
	SMTPAuthenticationMethod *string `json:"authentication_method"`

	// SMTPDomain optional domain for SMTP
	SMTPDomain *string `json:"domain"`
	// SMTPVerifySSLCerts defaults to true but can be turned off if self signed
	// SSL certs are used by the SMTP server
	SMTPVerifySSLCerts *bool `json:"verify_ssl_certs"`
	// SMTPEnableStartTLS detects of TLS is enabled on mail server and starts to use it (default true)
	SMTPEnableStartTLS *bool `json:"enable_start_tls"`
}

// AppConfigPayload contains request/response format of
// the AppConfig endpoints.
type AppConfigPayload struct {
	OrgInfo            *OrgInfo             `json:"org_info"`
	ServerSettings     *ServerSettings      `json:"server_settings"`
	SMTPSettings       *SMTPSettingsPayload `json:"smtp_settings"`
	HostExpirySettings *HostExpirySettings  `json:"host_expiry_settings"`
	HostSettings       *HostSettings        `json:"host_settings"`
	// SMTPTest is a flag that if set will cause the server to test email configuration
	SMTPTest *bool `json:"smtp_test,omitempty"`
	// SSOSettings is single sign on settings
	SSOSettings *SSOSettingsPayload `json:"sso_settings"`
}

// OrgInfo contains general info about the organization using Fleet.
type OrgInfo struct {
	OrgName    *string `json:"org_name,omitempty"`
	OrgLogoURL *string `json:"org_logo_url,omitempty"`
}

// ServerSettings contains general settings about the kolide App.
type ServerSettings struct {
	KolideServerURL   *string `json:"kolide_server_url,omitempty"`
	LiveQueryDisabled *bool   `json:"live_query_disabled,omitempty"`
}

// HostExpirySettings contains settings pertaining to automatic host expiry.
type HostExpirySettings struct {
	HostExpiryEnabled *bool `json:"host_expiry_enabled,omitempty"`
	HostExpiryWindow  *int  `json:"host_expiry_window,omitempty"`
}

type HostSettings struct {
	AdditionalQueries *json.RawMessage `json:"additional_queries"`
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

// EnrollSecret contains information about an enroll secret, name, and active
// status. Enroll secrets are used for osquery authentication.
type EnrollSecret struct {
	// Name is the name assigned to the secret
	Name string `json:"name" db:"name"`
	// Secret is the actual secret key.
	Secret string `json:"secret" db:"secret"`
	// Active determines whether the secret is currently allowed to be used for
	// authentication.
	Active bool `json:"active" db:"active"`
	// CreatedAt is the time this enroll secret was first added.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

const (
	EnrollSecretKind = "enroll_secret"
)

// EnrollSecretSpec is the fleetctl spec type for enroll secrets.
type EnrollSecretSpec struct {
	// Secrets is the list of enroll secrets.
	Secrets []EnrollSecret `json:"secrets"`
}
