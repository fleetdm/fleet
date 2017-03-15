package kolide

import (
	"context"
)

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
	SendTestEmail(ctx context.Context, config *AppConfig) error

	// Certificate returns the PEM encoded certificate chain for osqueryd TLS termination.
	// For cases where the connection is self-signed, the server will attempt to
	// connect using the InsecureSkipVerify option in tls.Config.
	CertificateChain(ctx context.Context) (cert []byte, err error)
}

// SMTP settings names returned from API, these map to SMTPAuthType and
// SMTPAuthMethod
const (
	AuthMethodNameCramMD5        = "authmethod_cram_md5"
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
)

func (m SMTPAuthMethod) String() string {
	switch m {
	case AuthMethodPlain:
		return AuthMethodNamePlain
	case AuthMethodCramMD5:
		return AuthMethodNameCramMD5
	default:
		return ""
	}
}

// AppConfig holds configuration about the Kolide application.
// AppConfig data can be managed by a Kolide API user.
type AppConfig struct {
	ID              uint
	OrgName         string `db:"org_name"`
	OrgLogoURL      string `db:"org_logo_url"`
	KolideServerURL string `db:"kolide_server_url"`

	// EnrollSecret is the config value that must be given by osqueryd hosts
	// on enrollment.
	// See https://osquery.readthedocs.io/en/stable/deployment/remote/#remote-authentication
	EnrollSecret string `db:"osquery_enroll_secret"`

	// SMTPConfigured is a flag that indicates if smtp has been successfully
	// tested with the settings provided by an admin user.
	SMTPConfigured bool `db:"smtp_configured"`
	// SMTPSenderAddress is the email address that will appear in emails sent
	// from Kolide
	SMTPSenderAddress string `db:"smtp_sender_address"`
	// SMTPServer is the host name of the SMTP server Kolide will use to send mail
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
}

// ModifyAppConfigRequest contains application configuration information
// sent from front end and used to change app config elements.
type ModifyAppConfigRequest struct {
	// TestSMTP is this is set to true, the SMTP configuration will be tested
	// with the results of the test returned to caller. No config changes
	// will be applied.
	TestSMTP  bool      `json:"test_smtp"`
	AppConfig AppConfig `json:"app_config"`
}

// SMTPSettingsPayload is part of the AppConfigPayload which defines the wire representation
// of the app config endpoints
type SMTPSettingsPayload struct {
	// SMTPConfigured is a flag that indicates if smtp has been successfully
	// tested with the settings provided by an admin user.
	SMTPConfigured *bool `json:"configured"`
	// SMTPSenderAddress is the email address that will appear in emails sent
	// from Kolide
	SMTPSenderAddress *string `json:"sender_address"`
	// SMTPServer is the host name of the SMTP server Kolide will use to send mail
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
	OrgInfo        *OrgInfo             `json:"org_info"`
	ServerSettings *ServerSettings      `json:"server_settings"`
	SMTPSettings   *SMTPSettingsPayload `json:"smtp_settings"`
	// SMTPTest is a flag that if set will cause the server to test email configuration
	SMTPTest *bool `json:"smtp_test,omitempty"`
}

// OrgInfo contains general info about the organization using Kolide.
type OrgInfo struct {
	OrgName    *string `json:"org_name"`
	OrgLogoURL *string `json:"org_logo_url"`
}

// ServerSettings contains general settings about the kolide App.
type ServerSettings struct {
	KolideServerURL *string `json:"kolide_server_url"`
	EnrollSecret    *string `json:"osquery_enroll_secret"`
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
