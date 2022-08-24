package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
)

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
	AppConfigKind  = "config"
	MaskedPassword = "********"
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
	// EnableJITProvisioning allows user accounts to be created the first time
	// users try to log in
	EnableJITProvisioning bool `json:"enable_jit_provisioning"`
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

// AppConfig holds server configuration that can be changed via the API.
//
// Note: management of deprecated fields is done on JSON-marshalling and uses
// the legacyConfig struct to list them.
type AppConfig struct {
	OrgInfo            OrgInfo            `json:"org_info"`
	ServerSettings     ServerSettings     `json:"server_settings"`
	SMTPSettings       SMTPSettings       `json:"smtp_settings"`
	HostExpirySettings HostExpirySettings `json:"host_expiry_settings"`
	// Features allows to globally enable or disable features
	Features     Features         `json:"features"`
	AgentOptions *json.RawMessage `json:"agent_options,omitempty"`
	// SMTPTest is a flag that if set will cause the server to test email configuration
	SMTPTest bool `json:"smtp_test,omitempty"`
	// SSOSettings is single sign on settings
	SSOSettings SSOSettings `json:"sso_settings"`
	// FleetDesktop holds settings for Fleet Desktop that can be changed via the API.
	FleetDesktop FleetDesktopSettings `json:"fleet_desktop"`

	// VulnerabilitySettings defines how fleet will behave while scanning for vulnerabilities in the host software
	VulnerabilitySettings VulnerabilitySettings `json:"vulnerability_settings"`

	WebhookSettings WebhookSettings `json:"webhook_settings"`
	Integrations    Integrations    `json:"integrations"`

	strictDecoding bool
}

// legacyConfig holds settings that have been replaced, superceded or
// deprecated by other AppConfig settings.
type legacyConfig struct {
	HostSettings *Features `json:"host_settings"`
}

// EnrichedAppConfig contains the AppConfig along with additional fleet
// instance configuration settings as returned by the
// "GET /api/latest/fleet/config" API endpoint (and fleetctl get config).
type EnrichedAppConfig struct {
	AppConfig
	enrichedAppConfigFields
}

// enrichedAppConfigFields are grouped separately to aid with JSON unmarshaling
type enrichedAppConfigFields struct {
	UpdateInterval  *UpdateIntervalConfig  `json:"update_interval,omitempty"`
	Vulnerabilities *VulnerabilitiesConfig `json:"vulnerabilities,omitempty"`
	License         *LicenseInfo           `json:"license,omitempty"`
	Logging         *Logging               `json:"logging,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface to make sure we serialize
// both AppConfig and enrichedAppConfigFields properly:
//
// - If this function is not defined, AppConfig.UnmarshalJSON gets promoted and
// will be called instead.
// - If we try to unmarshal everything in one go, AppConfig.UnmarshalJSON doesn't get
// called.
func (e *EnrichedAppConfig) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &e.AppConfig); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &e.enrichedAppConfigFields); err != nil {
		return err
	}
	return nil
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d Duration) ValueOr(t time.Duration) time.Duration {
	if d.Duration == 0 {
		return t
	}
	return d.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration type: %T", value)
	}
}

type WebhookSettings struct {
	HostStatusWebhook      HostStatusWebhookSettings      `json:"host_status_webhook"`
	FailingPoliciesWebhook FailingPoliciesWebhookSettings `json:"failing_policies_webhook"`
	VulnerabilitiesWebhook VulnerabilitiesWebhookSettings `json:"vulnerabilities_webhook"`
	// Interval is the interval for running the webhooks.
	//
	// This value currently configures both the host status and failing policies webhooks.
	Interval Duration `json:"interval"`
}

type HostStatusWebhookSettings struct {
	Enable         bool    `json:"enable_host_status_webhook"`
	DestinationURL string  `json:"destination_url"`
	HostPercentage float64 `json:"host_percentage"`
	DaysCount      int     `json:"days_count"`
}

// FailingPoliciesWebhookSettings holds the settings for failing policy webhooks.
type FailingPoliciesWebhookSettings struct {
	// Enable indicates whether the webhook for failing policies is enabled.
	Enable bool `json:"enable_failing_policies_webhook"`
	// DestinationURL is the webhook's URL.
	DestinationURL string `json:"destination_url"`
	// PolicyIDs is a list of policy IDs for which the webhook will be configured.
	PolicyIDs []uint `json:"policy_ids"`
	// HostBatchSize allows sending multiple requests in batches of hosts for each policy.
	// A value of 0 means no batching.
	HostBatchSize int `json:"host_batch_size"`
}

// VulnerabilitiesWebhookSettings holds the settings for vulnerabilities webhooks.
type VulnerabilitiesWebhookSettings struct {
	// Enable indicates whether the webhook for vulnerabilities is enabled.
	Enable bool `json:"enable_vulnerabilities_webhook"`
	// DestinationURL is the webhook's URL.
	DestinationURL string `json:"destination_url"`
	// HostBatchSize allows sending multiple requests in batches of hosts for each vulnerable software found.
	// A value of 0 means no batching.
	HostBatchSize int `json:"host_batch_size"`
}

func (c *AppConfig) ApplyDefaultsForNewInstalls() {
	c.ServerSettings.EnableAnalytics = true

	c.SMTPSettings.SMTPPort = 587
	c.SMTPSettings.SMTPEnableStartTLS = true
	c.SMTPSettings.SMTPAuthenticationType = AuthTypeNameUserNamePassword
	c.SMTPSettings.SMTPAuthenticationMethod = AuthMethodNamePlain
	c.SMTPSettings.SMTPVerifySSLCerts = true
	c.SMTPSettings.SMTPEnableTLS = true

	agentOptions := json.RawMessage(`{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`)
	c.AgentOptions = &agentOptions

	c.Features.EnableSoftwareInventory = true

	c.ApplyDefaults()
}

func (c *AppConfig) ApplyDefaults() {
	c.Features.EnableHostUsers = true
	c.WebhookSettings.Interval.Duration = 24 * time.Hour
}

// EnableStrictDecoding enables strict decoding of the AppConfig struct.
func (c *AppConfig) EnableStrictDecoding() { c.strictDecoding = true }

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *AppConfig) UnmarshalJSON(b []byte) error {
	// Define a new type, this is to prevent infinite recursion when
	// unmarshalling the AppConfig struct.
	type cfgStructUnmarshal AppConfig
	compatConfig := struct {
		*legacyConfig
		*cfgStructUnmarshal
	}{
		&legacyConfig{},
		(*cfgStructUnmarshal)(c),
	}

	decoder := json.NewDecoder(bytes.NewReader(b))
	if c.strictDecoding {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(&compatConfig); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("unexpected extra tokens found in config")
	}

	// Define and assign legacy settings to new fields.
	// This has the drawback of legacy fields taking precedence over new fields
	// if both are defined.
	if compatConfig.legacyConfig.HostSettings != nil {
		c.Features = *compatConfig.legacyConfig.HostSettings
	}

	return nil
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
	DebugHostIDs      []uint `json:"debug_host_ids,omitempty"`
	DeferredSaveHost  bool   `json:"deferred_save_host"`
}

// HostExpirySettings contains settings pertaining to automatic host expiry.
type HostExpirySettings struct {
	HostExpiryEnabled bool `json:"host_expiry_enabled"`
	HostExpiryWindow  int  `json:"host_expiry_window"`
}

type Features struct {
	EnableHostUsers         bool             `json:"enable_host_users"`
	EnableSoftwareInventory bool             `json:"enable_software_inventory"`
	AdditionalQueries       *json.RawMessage `json:"additional_queries,omitempty"`
}

// FleetDesktopSettings contains settings used to configure Fleet Desktop.
type FleetDesktopSettings struct {
	// TransparencyURL is the URL used for the “Transparency” link in the Fleet Desktop menu.
	TransparencyURL string `json:"transparency_url"`
}

// DefaultTransparencyURL is the default URL used for the “Transparency” link in the Fleet Desktop menu.
const DefaultTransparencyURL = "https://fleetdm.com/transparency"

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
	Page uint `query:"page,optional"`
	// How many results per page (must be positive integer, 0 indicates
	// unlimited)
	PerPage uint `query:"per_page,optional"`
	// Key to use for ordering. Can be a comma separated set of items, eg: host_count,id
	OrderKey string `query:"order_key,optional"`
	// Direction of ordering
	OrderDirection OrderDirection `query:"order_direction,optional"`
	// MatchQuery is the query string to match against columns of the entity
	// (varies depending on entity, eg. hostname, IP address for hosts).
	// Handling for this parameter must be implemented separately for each type.
	MatchQuery string `query:"query,optional"`

	// After denotes the row to start from. This is meant to be used in conjunction with OrderKey
	// If OrderKey is "id", it'll assume After is a number and will try to convert it.
	After string `query:"after,optional"`
}

func (l ListOptions) Empty() bool {
	return l == ListOptions{}
}

type ListQueryOptions struct {
	ListOptions

	OnlyObserverCanRun bool
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
	// tierBasic is for backward compatibility with previous tier names
	tierBasic = "basic"

	// TierPremium is Fleet Premium aka the paid license.
	TierPremium = "premium"
	// TierFree is Fleet Free aka the free license.
	TierFree = "free"
)

// LicenseInfo contains information about the Fleet license.
type LicenseInfo struct {
	// Tier is the license tier (currently "free" or "premium")
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

func (l *LicenseInfo) IsPremium() bool {
	return l.Tier == TierPremium || l.Tier == tierBasic
}

func (l *LicenseInfo) IsExpired() bool {
	return l.Expiration.Before(time.Now())
}

const (
	HeaderLicenseKey          = "X-Fleet-License"
	HeaderLicenseValueExpired = "Expired"
)

type Logging struct {
	Debug  bool          `json:"debug"`
	Json   bool          `json:"json"`
	Result LoggingPlugin `json:"result"`
	Status LoggingPlugin `json:"status"`
}

type UpdateIntervalConfig struct {
	OSQueryDetail time.Duration `json:"osquery_detail"`
	OSQueryPolicy time.Duration `json:"osquery_policy"`
}

// VulnerabilitiesConfig contains the vulnerabilities configuration of the
// fleet instance (as configured for the cli, either via flags, env vars or the
// config file), not to be confused with VulnerabilitySettings which is the
// configuration in AppConfig.
type VulnerabilitiesConfig struct {
	DatabasesPath             string        `json:"databases_path"`
	Periodicity               time.Duration `json:"periodicity"`
	CPEDatabaseURL            string        `json:"cpe_database_url"`
	CVEFeedPrefixURL          string        `json:"cve_feed_prefix_url"`
	CurrentInstanceChecks     string        `json:"current_instance_checks"`
	DisableDataSync           bool          `json:"disable_data_sync"`
	RecentVulnerabilityMaxAge time.Duration `json:"recent_vulnerability_max_age"`
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

// KafkaRESTConfig shadows config.KafkaRESTConfig
type KafkaRESTConfig struct {
	StatusTopic string `json:"status_topic"`
	ResultTopic string `json:"result_topic"`
	ProxyHost   string `json:"proxyhost"`
}

// DeviceAPIFeatures specifies a list of features supported
// by the current API version. Each field in the struct is
// meant to be a boolean value.
type DeviceAPIFeatures struct{}
