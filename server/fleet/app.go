package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
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

type SSOProviderSettings struct {
	// EntityID is a uri that identifies this service provider
	EntityID string `json:"entity_id"`
	// IssuerURI is the uri that identifies the identity provider
	IssuerURI string `json:"issuer_uri"`
	// Metadata contains IDP metadata XML
	Metadata string `json:"metadata"`
	// MetadataURL is a URL provided by the IDP which can be used to download
	// metadata
	MetadataURL string `json:"metadata_url"`
	// IDPName is a human friendly name for the IDP
	IDPName string `json:"idp_name"`
}

func (s SSOProviderSettings) IsEmpty() bool {
	return s == (SSOProviderSettings{})
}

// SSOSettings wire format for SSO settings
type SSOSettings struct {
	SSOProviderSettings

	// IDPImageURL is a link to a logo or other image that is used for UX
	IDPImageURL string `json:"idp_image_url"`
	// EnableSSO flag to determine whether or not to enable SSO
	EnableSSO bool `json:"enable_sso"`
	// EnableSSOIdPLogin flag to determine whether or not to allow IdP-initiated
	// login.
	EnableSSOIdPLogin bool `json:"enable_sso_idp_login"`
	// EnableJITProvisioning allows user accounts to be created the first time
	// users try to log in
	EnableJITProvisioning bool `json:"enable_jit_provisioning"`
	// EnableJITRoleSync is deprecated.
	//
	// EnableJITRoleSync sets whether the roles of existing accounts will be updated
	// every time SSO users log in (does not have effect if EnableJITProvisioning is false).
	EnableJITRoleSync bool `json:"enable_jit_role_sync"`
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

// MDM is part of AppConfig and defines the mdm settings.
type MDM struct {
	AppleBMDefaultTeam string `json:"apple_bm_default_team"`

	// AppleBMEnabledAndConfigured is set to true if Fleet has been
	// configured with the required Apple BM key pair or token. It can't be set
	// manually via the PATCH /config API, it's only set automatically when
	// the server starts.
	AppleBMEnabledAndConfigured bool `json:"apple_bm_enabled_and_configured"`

	// AppleBMTermsExpired is set to true if an Apple Business Manager request
	// failed due to Apple's terms and conditions having changed and need the
	// user to explicitly accept them. It cannot be set manually via the
	// PATCH /config API, it is only set automatically, internally, by detecting
	// the 403 Forbidden error with body T_C_NOT_SIGNED returned by the Apple BM
	// API.
	AppleBMTermsExpired bool `json:"apple_bm_terms_expired"`

	// EnabledAndConfigured is set to true if Fleet has been
	// configured with the required APNS and SCEP certificates. It can't be set
	// manually via the PATCH /config API, it's only set automatically when
	// the server starts.
	//
	// TODO: should ideally be renamed to AppleEnabledAndConfigured, but it
	// implies a lot of changes to existing code across both frontend and
	// backend, should be done only after careful analysis.
	EnabledAndConfigured bool `json:"enabled_and_configured"`

	MacOSUpdates          MacOSUpdates             `json:"macos_updates"`
	MacOSSettings         MacOSSettings            `json:"macos_settings"`
	MacOSSetup            MacOSSetup               `json:"macos_setup"`
	MacOSMigration        MacOSMigration           `json:"macos_migration"`
	EndUserAuthentication MDMEndUserAuthentication `json:"end_user_authentication"`

	// WindowsEnabledAndConfigured indicates if Fleet MDM is enabled for Windows.
	// There is no other configuration required for Windows other than enabling
	// the support, but it is still called "EnabledAndConfigured" for consistency
	// with the similarly named macOS-specific fields.
	WindowsEnabledAndConfigured bool `json:"windows_enabled_and_configured"`

	/////////////////////////////////////////////////////////////////
	// WARNING: If you add to this struct make sure it's taken into
	// account in the AppConfig Clone implementation!
	/////////////////////////////////////////////////////////////////
}

// versionStringRegex is used to validate that a version string is in the x.y.z
// format only (no prerelease or build metadata).
var versionStringRegex = regexp.MustCompile(`^\d+(\.\d+)?(\.\d+)?$`)

// MacOSUpdates is part of AppConfig and defines the macOS update settings.
type MacOSUpdates struct {
	// MinimumVersion is the required minimum operating system version.
	MinimumVersion optjson.String `json:"minimum_version"`
	// Deadline the required installation date for Nudge to enforce the required
	// operating system version.
	Deadline optjson.String `json:"deadline"`
}

// EnabledForHost returns a boolean indicating if updates are enabled for the host
func (m MacOSUpdates) EnabledForHost(h *Host) bool {
	return m.Deadline.Value != "" &&
		m.MinimumVersion.Value != "" &&
		h.IsOsqueryEnrolled() &&
		h.MDMInfo.IsFleetEnrolled()
}

func (m MacOSUpdates) Validate() error {
	// if no settings are provided it's okay to skip further validation
	if m.MinimumVersion.Value == "" && m.Deadline.Value == "" {
		// if one is set and empty, the other must be set and empty too, otherwise
		// it's as if only one was provided.
		if m.MinimumVersion.Set && !m.Deadline.Set {
			return errors.New("deadline is required when minimum_version is provided")
		} else if !m.MinimumVersion.Set && m.Deadline.Set {
			return errors.New("minimum_version is required when deadline is provided")
		}
		return nil
	}

	if m.MinimumVersion.Value != "" && m.Deadline.Value == "" {
		return errors.New("deadline is required when minimum_version is provided")
	}

	if m.Deadline.Value != "" && m.MinimumVersion.Value == "" {
		return errors.New("minimum_version is required when deadline is provided")
	}

	if !versionStringRegex.MatchString(m.MinimumVersion.Value) {
		return errors.New(`minimum_version accepts version numbers only. (E.g., "13.0.1.") NOT "Ventura 13" or "13.0.1 (22A400)"`)
	}

	if _, err := time.Parse("2006-01-02", m.Deadline.Value); err != nil {
		return errors.New(`deadline accepts YYYY-MM-DD format only (E.g., "2023-06-01.")`)
	}

	return nil
}

// MacOSSettings contains settings specific to macOS.
type MacOSSettings struct {
	// CustomSettings is a slice of configuration profile file paths.
	//
	// NOTE: These are only present here for informational purposes.
	// (The source of truth for profiles is in MySQL.)
	CustomSettings []string `json:"custom_settings"`
	// EnableDiskEncryption enables disk encryption on hosts such that the hosts'
	// disk encryption keys will be stored in Fleet.
	EnableDiskEncryption bool `json:"enable_disk_encryption"`

	// NOTE: make sure to update the ToMap/FromMap methods when adding/updating fields.
}

func (s MacOSSettings) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"custom_settings":        s.CustomSettings,
		"enable_disk_encryption": s.EnableDiskEncryption,
	}
}

// FromMap sets the macOS settings from the provided map, which is the map type
// from the ApplyTeams spec struct. It returns a map of fields that were set in
// the map (ie. the key was present even if empty) or an error. If the
// operation updates an existing team, it should be called on the existing
// MacOSSettings so that its fields are replaced only if present in the map.
func (s *MacOSSettings) FromMap(m map[string]interface{}) (map[string]bool, error) {
	set := make(map[string]bool)

	if v, ok := m["custom_settings"]; ok {
		set["custom_settings"] = true

		vals, ok := v.([]interface{})
		if v == nil || ok {
			strs := make([]string, 0, len(vals))
			for _, v := range vals {
				str, ok := v.(string)
				if !ok {
					// error, must be a []string
					return nil, &json.UnmarshalTypeError{
						Value: fmt.Sprintf("%T", v),
						Type:  reflect.TypeOf(s.CustomSettings),
						Field: "macos_settings.custom_settings",
					}
				}
				strs = append(strs, str)
			}
			s.CustomSettings = strs
		}
	}

	if v, ok := m["enable_disk_encryption"]; ok {
		set["enable_disk_encryption"] = true
		b, ok := v.(bool)
		if !ok {
			// error, must be a bool
			return nil, &json.UnmarshalTypeError{
				Value: fmt.Sprintf("%T", v),
				Type:  reflect.TypeOf(s.EnableDiskEncryption),
				Field: "macos_settings.enable_disk_encryption",
			}
		}
		s.EnableDiskEncryption = b
	}

	return set, nil
}

// MacOSSetup contains settings related to the setup of DEP enrolled devices.
type MacOSSetup struct {
	BootstrapPackage            optjson.String `json:"bootstrap_package"`
	EnableEndUserAuthentication bool           `json:"enable_end_user_authentication"`
	MacOSSetupAssistant         optjson.String `json:"macos_setup_assistant"`
}

// MacOSMigration contains settings related to the MDM migration work flow.
type MacOSMigration struct {
	Enable     bool               `json:"enable"`
	Mode       MacOSMigrationMode `json:"mode"`
	WebhookURL string             `json:"webhook_url"`
}

// MacOSMigrationMode defines the possible modes that can be set if a user enables the MDM migration
// work flow in Fleet.
type MacOSMigrationMode string

const (
	MacOSMigrationModeForced    MacOSMigrationMode = "forced"
	MacOSMigrationModeVoluntary MacOSMigrationMode = "voluntary"
)

// IsValid returns true if the mode is one of the valid modes.
func (s MacOSMigrationMode) IsValid() bool {
	switch s {
	case MacOSMigrationModeForced, MacOSMigrationModeVoluntary:
		return true
	default:
		return false
	}
}

// MDMEndUserAuthentication contains settings related to end user authentication
// to gate certain MDM features (eg: enrollment)
type MDMEndUserAuthentication struct {
	// SSOSettings configure the IdP integration. Note that all keys under
	// SSOProviderSettings are top-level keys under this struct, that's why
	// it's embedded.
	SSOProviderSettings
}

// AppConfig holds global server configuration that can be changed via the API.
//
// Note: management of deprecated fields is done on JSON-marshalling and uses
// the legacyConfig struct to list them.
//
// ///////////////////////////////////////////////////////////////
// WARNING: If you add or change fields of this struct make sure
// it's taken into account in the AppConfig Clone implementation!
// ///////////////////////////////////////////////////////////////
type AppConfig struct {
	OrgInfo        OrgInfo        `json:"org_info"`
	ServerSettings ServerSettings `json:"server_settings"`
	// SMTPSettings holds the SMTP integration settings.
	//
	// This field is a pointer to avoid returning this information to non-global-admins.
	SMTPSettings       *SMTPSettings      `json:"smtp_settings,omitempty"`
	HostExpirySettings HostExpirySettings `json:"host_expiry_settings"`
	// Features allows to globally enable or disable features
	Features Features `json:"features"`
	// AgentOptions holds osquery configuration.
	//
	// This field is a pointer to avoid returning this information to non-global-admins.
	AgentOptions *json.RawMessage `json:"agent_options,omitempty"`
	// SMTPTest is a flag that if set will cause the server to test email configuration
	SMTPTest bool `json:"smtp_test,omitempty"`
	// SSOSettings is single sign on integration settings.
	//
	// This field is a pointer to avoid returning this information to non-global-admins.
	SSOSettings *SSOSettings `json:"sso_settings,omitempty"`
	// FleetDesktop holds settings for Fleet Desktop that can be changed via the API.
	FleetDesktop FleetDesktopSettings `json:"fleet_desktop"`

	// VulnerabilitySettings defines how fleet will behave while scanning for vulnerabilities in the host software
	VulnerabilitySettings VulnerabilitySettings `json:"vulnerability_settings"`

	WebhookSettings WebhookSettings `json:"webhook_settings"`
	Integrations    Integrations    `json:"integrations"`

	MDM MDM `json:"mdm"`

	// when true, strictDecoding causes the UnmarshalJSON method to return an
	// error if there are unknown fields in the raw JSON.
	strictDecoding bool
	// this field is set to the list of legacy settings keys during UnmarshalJSON
	// if any legacy settings were set in the raw JSON.
	didUnmarshalLegacySettings []string

	// ///////////////////////////////////////////////////////////////
	// WARNING: If you add or change fields of this struct make sure
	// it's taken into account in the AppConfig Clone implementation!
	// ///////////////////////////////////////////////////////////////
}

// Obfuscate overrides credentials with obfuscated characters.
func (c *AppConfig) Obfuscate() {
	if c.SMTPSettings != nil && c.SMTPSettings.SMTPPassword != "" {
		c.SMTPSettings.SMTPPassword = MaskedPassword
	}
	for _, jiraIntegration := range c.Integrations.Jira {
		jiraIntegration.APIToken = MaskedPassword
	}
	for _, zdIntegration := range c.Integrations.Zendesk {
		zdIntegration.APIToken = MaskedPassword
	}
}

// legacyConfig holds settings that have been replaced, superceded or
// deprecated by other AppConfig settings.
type legacyConfig struct {
	HostSettings *Features `json:"host_settings"`
}

// Clone implements cloner.
func (c *AppConfig) Clone() (interface{}, error) {
	return c.Copy(), nil
}

// Copy returns a copy of the AppConfig.
func (c *AppConfig) Copy() *AppConfig {
	if c == nil {
		return nil
	}

	var clone AppConfig
	clone = *c

	// OrgInfo: nothing needs cloning
	// FleetDesktopSettings: nothing needs cloning

	if c.ServerSettings.DebugHostIDs != nil {
		clone.ServerSettings.DebugHostIDs = make([]uint, len(c.ServerSettings.DebugHostIDs))
		copy(clone.ServerSettings.DebugHostIDs, c.ServerSettings.DebugHostIDs)
	}

	if c.SMTPSettings != nil {
		var smtpSettings SMTPSettings
		smtpSettings = *c.SMTPSettings
		clone.SMTPSettings = &smtpSettings
	}

	// HostExpirySettings: nothing needs cloning

	if c.Features.AdditionalQueries != nil {
		aq := make(json.RawMessage, len(*c.Features.AdditionalQueries))
		copy(aq, *c.Features.AdditionalQueries)
		c.Features.AdditionalQueries = &aq
	}
	if c.AgentOptions != nil {
		ao := make(json.RawMessage, len(*c.AgentOptions))
		copy(ao, *c.AgentOptions)
		clone.AgentOptions = &ao
	}

	if c.SSOSettings != nil {
		var ssoSettings SSOSettings
		ssoSettings = *c.SSOSettings
		clone.SSOSettings = &ssoSettings
	}

	// FleetDesktop: nothing needs cloning
	// VulnerabilitySettings: nothing needs cloning

	if c.WebhookSettings.FailingPoliciesWebhook.PolicyIDs != nil {
		clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs = make([]uint, len(c.WebhookSettings.FailingPoliciesWebhook.PolicyIDs))
		copy(clone.WebhookSettings.FailingPoliciesWebhook.PolicyIDs, c.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	}
	if c.Integrations.Jira != nil {
		clone.Integrations.Jira = make([]*JiraIntegration, len(c.Integrations.Jira))
		for i, j := range c.Integrations.Jira {
			jira := *j
			clone.Integrations.Jira[i] = &jira
		}
	}
	if c.Integrations.Zendesk != nil {
		clone.Integrations.Zendesk = make([]*ZendeskIntegration, len(c.Integrations.Zendesk))
		for i, z := range c.Integrations.Zendesk {
			zd := *z
			clone.Integrations.Zendesk[i] = &zd
		}
	}

	if c.MDM.MacOSSettings.CustomSettings != nil {
		clone.MDM.MacOSSettings.CustomSettings = make([]string, len(c.MDM.MacOSSettings.CustomSettings))
		copy(clone.MDM.MacOSSettings.CustomSettings, c.MDM.MacOSSettings.CustomSettings)
	}

	return &clone
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
	Email           *EmailConfig           `json:"email,omitempty"`
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

	// Add default values for SMTPSettings.
	var smtpSettings SMTPSettings
	smtpSettings.SMTPEnabled = false
	smtpSettings.SMTPPort = 587
	smtpSettings.SMTPEnableStartTLS = true
	smtpSettings.SMTPAuthenticationType = AuthTypeNameUserNamePassword
	smtpSettings.SMTPAuthenticationMethod = AuthMethodNamePlain
	smtpSettings.SMTPVerifySSLCerts = true
	smtpSettings.SMTPEnableTLS = true
	c.SMTPSettings = &smtpSettings

	agentOptions := json.RawMessage(`{"config": {"options": {"pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`)
	c.AgentOptions = &agentOptions

	// Make sure an empty SSOSettings is set.
	var ssoSettings SSOSettings
	c.SSOSettings = &ssoSettings

	c.Features.ApplyDefaultsForNewInstalls()

	c.ApplyDefaults()
}

func (c *AppConfig) ApplyDefaults() {
	c.Features.ApplyDefaults()
	c.WebhookSettings.Interval.Duration = 24 * time.Hour
}

// EnableStrictDecoding enables strict decoding of the AppConfig struct.
func (c *AppConfig) EnableStrictDecoding() { c.strictDecoding = true }

// DidUnmarshalLegacySettings returns the list of legacy settings keys that
// were set in the JSON used to unmarshal this AppConfig.
func (c *AppConfig) DidUnmarshalLegacySettings() []string { return c.didUnmarshalLegacySettings }

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

	c.didUnmarshalLegacySettings = nil
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
		c.didUnmarshalLegacySettings = append(c.didUnmarshalLegacySettings, "host_settings")
		c.Features = *compatConfig.legacyConfig.HostSettings
	}
	sort.Strings(c.didUnmarshalLegacySettings)

	return nil
}

// OrgInfo contains general info about the organization using Fleet.
type OrgInfo struct {
	OrgName                   string `json:"org_name"`
	OrgLogoURL                string `json:"org_logo_url"`
	OrgLogoURLLightBackground string `json:"org_logo_url_light_background"`
	// ContactURL is the URL displayed for users to contact support. By default,
	// https://fleetdm.com/company/contact is used.
	ContactURL string `json:"contact_url"`
}

const DefaultOrgInfoContactURL = "https://fleetdm.com/company/contact"

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
	EnableHostUsers         bool               `json:"enable_host_users"`
	EnableSoftwareInventory bool               `json:"enable_software_inventory"`
	AdditionalQueries       *json.RawMessage   `json:"additional_queries,omitempty"`
	DetailQueryOverrides    map[string]*string `json:"detail_query_overrides,omitempty"`
}

func (f *Features) ApplyDefaultsForNewInstalls() {
	// Software inventory is enabled only for new installs as
	// we didn't want to enable software inventory from one version to the
	// next in already running fleets
	f.EnableSoftwareInventory = true
	f.ApplyDefaults()
}

func (f *Features) ApplyDefaults() {
	f.EnableHostUsers = true
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
	// Used to request the metadata of a query
	IncludeMetadata bool
}

func (l ListOptions) Empty() bool {
	return l == ListOptions{}
}

func (l ListOptions) UsesCursorPagination() bool {
	return l.After != "" && l.OrderKey != ""
}

type ListQueryOptions struct {
	ListOptions

	// TeamID which team the queries belong to. If teamID is nil, then it is assumed the 'global'
	// team.
	TeamID *uint
	// IsScheduled filters queries that are meant to run at a set interval.
	IsScheduled        *bool
	OnlyObserverCanRun bool
}

type ListActivitiesOptions struct {
	ListOptions

	Streamed *bool
}

// ApplySpecOptions are the options available when applying a YAML or JSON spec.
type ApplySpecOptions struct {
	// Force indicates that any validation error in the incoming payload should
	// be ignored and the spec should be applied anyway.
	Force bool
	// DryRun indicates that the spec should not be applied, but the validation
	// errors should be returned.
	DryRun bool
	// TeamForPolicies is the name of the team to set in policy specs.
	TeamForPolicies string
}

// RawQuery returns the ApplySpecOptions url-encoded for use in an URL's
// query string parameters. It only sets the parameters that are not the
// default values.
func (o *ApplySpecOptions) RawQuery() string {
	if o == nil {
		return ""
	}

	query := make(url.Values)
	if o.Force {
		query.Set("force", "true")
	}
	if o.DryRun {
		query.Set("dry_run", "true")
	}
	return query.Encode()
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

// ExtraAuthz implements authz.ExtraAuthzer.
func (e *EnrollSecret) ExtraAuthz() (map[string]interface{}, error) {
	return map[string]interface{}{
		"is_global_secret": e.TeamID == nil,
	}, nil
}

// IsGlobalSecret returns whether the secret is global.
// This method is defined for the Policy Rego code (is_global_secret).
func (e *EnrollSecret) IsGlobalSecret() bool {
	return e.TeamID == nil
}

const (
	EnrollSecretKind          = "enroll_secret"
	EnrollSecretDefaultLength = 24
	// Maximum number of enroll secrets that can be set per team, or globally.
	// Make sure to change the documentation in docs/Contributing/API-for-Contributors.md
	// if you change that value (look for the string `secrets`).
	MaxEnrollSecretsCount = 50
)

// EnrollSecretSpec is the fleetctl spec type for enroll secrets.
type EnrollSecretSpec struct {
	// Secrets is the list of enroll secrets.
	Secrets []*EnrollSecret `json:"secrets"`
}

const (
	// tierBasicDeprecated is for backward compatibility with previous tier names
	tierBasicDeprecated = "basic"

	// TierPremium is Fleet Premium aka the paid license.
	TierPremium = "premium"
	// TierFree is Fleet Free aka the free license.
	TierFree = "free"
	// TierTrial is Fleet Premium but in trial mode
	// this is used to distinguish between Premium, enabling different functionality
	// when the license is expired, like disabling certain features
	TierTrial = "trial"
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
	return l.Tier == TierPremium || l.Tier == tierBasicDeprecated || l.Tier == TierTrial
}

func (l *LicenseInfo) IsExpired() bool {
	return l.Expiration.Before(time.Now())
}

func (l *LicenseInfo) ForceUpgrade() {
	if l.Tier == tierBasicDeprecated {
		l.Tier = TierPremium
	}
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
	Audit  LoggingPlugin `json:"audit"`
}

type EmailConfig struct {
	Backend string      `json:"backend"`
	Config  interface{} `json:"config"`
}

type SESConfig struct {
	Region    string `json:"region"`
	SourceARN string `json:"source_arn"`
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
	DatabasesPath               string        `json:"databases_path"`
	Periodicity                 time.Duration `json:"periodicity"`
	CPEDatabaseURL              string        `json:"cpe_database_url"`
	CPETranslationsURL          string        `json:"cpe_translations_url"`
	CVEFeedPrefixURL            string        `json:"cve_feed_prefix_url"`
	CurrentInstanceChecks       string        `json:"current_instance_checks"`
	DisableDataSync             bool          `json:"disable_data_sync"`
	RecentVulnerabilityMaxAge   time.Duration `json:"recent_vulnerability_max_age"`
	DisableWinOSVulnerabilities bool          `json:"disable_win_os_vulnerabilities"`
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
	AuditStream  string `json:"audit_stream"`
}

// KinesisConfig shadows config.KinesisConfig only exposing a subset of fields
type KinesisConfig struct {
	Region       string `json:"region"`
	StatusStream string `json:"status_stream"`
	ResultStream string `json:"result_stream"`
	AuditStream  string `json:"audit_stream"`
}

// LambdaConfig shadows config.LambdaConfig only exposing a subset of fields
type LambdaConfig struct {
	Region         string `json:"region"`
	StatusFunction string `json:"status_function"`
	ResultFunction string `json:"result_function"`
	AuditFunction  string `json:"audit_function"`
}

// KafkaRESTConfig shadows config.KafkaRESTConfig
type KafkaRESTConfig struct {
	StatusTopic string `json:"status_topic"`
	ResultTopic string `json:"result_topic"`
	AuditTopic  string `json:"audit_topic"`
	ProxyHost   string `json:"proxyhost"`
}

// DeviceGlobalConfig is a subset of AppConfig with information used by the
// device endpoints
type DeviceGlobalConfig struct {
	MDM DeviceGlobalMDMConfig `json:"mdm"`
}

// DeviceGlobalMDMConfig is a subset of AppConfig.MDM with information used by
// the device endpoints
type DeviceGlobalMDMConfig struct {
	EnabledAndConfigured bool `json:"enabled_and_configured"`
}

// Version is the authz type used to check access control to the version endpoint.
type Version struct{}

// AuthzType implements authz.AuthzTyper.
func (v *Version) AuthzType() string {
	return "version"
}
