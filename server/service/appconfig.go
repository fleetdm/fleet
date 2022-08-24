package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/url"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/kolide/kit/version"
)

////////////////////////////////////////////////////////////////////////////////
// Get AppConfig
////////////////////////////////////////////////////////////////////////////////

type appConfigResponse struct {
	fleet.AppConfig
	appConfigResponseFields
}

// appConfigResponseFields are grouped separately to aid with JSON unmarshaling
type appConfigResponseFields struct {
	UpdateInterval  *fleet.UpdateIntervalConfig  `json:"update_interval"`
	Vulnerabilities *fleet.VulnerabilitiesConfig `json:"vulnerabilities"`

	// License is loaded from the service
	License *fleet.LicenseInfo `json:"license,omitempty"`
	// Logging is loaded on the fly rather than from the database.
	Logging *fleet.Logging `json:"logging,omitempty"`
	// SandboxEnabled is true if fleet serve was ran with server.sandbox_enabled=true
	SandboxEnabled bool `json:"sandbox_enabled,omitempty"`

	Err error `json:"error,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface to make sure we serialize
// both AppConfig and appConfigResponseFields properly:
//
// - If this function is not defined, AppConfig.UnmarshalJSON gets promoted and
// will be called instead.
// - If we try to unmarshal everyting in one go, AppConfig.UnmarshalJSON doesn't get
// called.
func (r *appConfigResponse) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &r.AppConfig); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.appConfigResponseFields); err != nil {
		return err
	}
	return nil
}

func (r appConfigResponse) error() error { return r.Err }

func getAppConfigEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("could not fetch user")
	}
	config, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	license, err := svc.License(ctx)
	if err != nil {
		return nil, err
	}
	loggingConfig, err := svc.LoggingConfig(ctx)
	if err != nil {
		return nil, err
	}
	updateIntervalConfig, err := svc.UpdateIntervalConfig(ctx)
	if err != nil {
		return nil, err
	}
	vulnConfig, err := svc.VulnerabilitiesConfig(ctx)
	if err != nil {
		return nil, err
	}

	var smtpSettings fleet.SMTPSettings
	var ssoSettings fleet.SSOSettings
	var hostExpirySettings fleet.HostExpirySettings
	var agentOptions *json.RawMessage
	// only admin can see smtp, sso, and host expiry settings
	if vc.User.GlobalRole != nil && *vc.User.GlobalRole == fleet.RoleAdmin {
		smtpSettings = config.SMTPSettings
		ssoSettings = config.SSOSettings
		hostExpirySettings = config.HostExpirySettings
		agentOptions = config.AgentOptions
	}

	transparencyURL := fleet.DefaultTransparencyURL
	// Fleet Premium license is required for custom transparency url
	if license.IsPremium() && config.FleetDesktop.TransparencyURL != "" {
		transparencyURL = config.FleetDesktop.TransparencyURL
	}
	fleetDesktop := fleet.FleetDesktopSettings{TransparencyURL: transparencyURL}

	hostSettings := config.HostSettings
	response := appConfigResponse{
		AppConfig: fleet.AppConfig{
			OrgInfo:               config.OrgInfo,
			ServerSettings:        config.ServerSettings,
			HostSettings:          hostSettings,
			VulnerabilitySettings: config.VulnerabilitySettings,

			SMTPSettings:       smtpSettings,
			SSOSettings:        ssoSettings,
			HostExpirySettings: hostExpirySettings,
			AgentOptions:       agentOptions,

			FleetDesktop: fleetDesktop,

			WebhookSettings: config.WebhookSettings,
			Integrations:    config.Integrations,
		},
		appConfigResponseFields: appConfigResponseFields{
			UpdateInterval:  updateIntervalConfig,
			Vulnerabilities: vulnConfig,
			License:         license,
			Logging:         loggingConfig,
			SandboxEnabled:  svc.SandboxEnabled(),
		},
	}
	return response, nil
}

func (svc *Service) SandboxEnabled() bool {
	return svc.config.Server.SandboxEnabled
}

func (svc *Service) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	if ac.SMTPSettings.SMTPPassword != "" {
		ac.SMTPSettings.SMTPPassword = fleet.MaskedPassword
	}

	for _, jiraIntegration := range ac.Integrations.Jira {
		jiraIntegration.APIToken = fleet.MaskedPassword
	}

	for _, zdIntegration := range ac.Integrations.Zendesk {
		zdIntegration.APIToken = fleet.MaskedPassword
	}

	return ac, nil
}

////////////////////////////////////////////////////////////////////////////////
// Modify AppConfig
////////////////////////////////////////////////////////////////////////////////

type modifyAppConfigRequest struct {
	json.RawMessage
}

func modifyAppConfigEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*modifyAppConfigRequest)
	config, err := svc.ModifyAppConfig(ctx, req.RawMessage)
	if err != nil {
		return appConfigResponse{appConfigResponseFields: appConfigResponseFields{Err: err}}, nil
	}
	license, err := svc.License(ctx)
	if err != nil {
		return nil, err
	}
	loggingConfig, err := svc.LoggingConfig(ctx)
	if err != nil {
		return nil, err
	}
	response := appConfigResponse{
		AppConfig: *config,
		appConfigResponseFields: appConfigResponseFields{
			License: license,
			Logging: loggingConfig,
		},
	}

	if response.SMTPSettings.SMTPPassword != "" {
		response.SMTPSettings.SMTPPassword = fleet.MaskedPassword
	}

	if license.Tier != "premium" || response.FleetDesktop.TransparencyURL == "" {
		response.FleetDesktop.TransparencyURL = fleet.DefaultTransparencyURL
	}

	return response, nil
}

func (svc *Service) ModifyAppConfig(ctx context.Context, p []byte) (*fleet.AppConfig, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// we need the config from the datastore because API tokens are obfuscated at the service layer
	// we will retrieve the obfuscated config before we return
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	license, err := svc.License(ctx)
	if err != nil {
		return nil, err
	}

	oldSmtpSettings := appConfig.SMTPSettings

	storedJiraByProjectKey, err := fleet.IndexJiraIntegrations(appConfig.Integrations.Jira)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "modify AppConfig")
	}

	storedZendeskByGroupID, err := fleet.IndexZendeskIntegrations(appConfig.Integrations.Zendesk)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "modify AppConfig")
	}

	// TODO(mna): this ports the validations from the old validationMiddleware
	// correctly, but this could be optimized so that we don't unmarshal the
	// incoming bytes twice.
	invalid := &fleet.InvalidArgumentError{}
	var newAppConfig fleet.AppConfig
	if err := json.Unmarshal(p, &newAppConfig); err != nil {
		return nil, ctxerr.Wrap(ctx, &badRequestError{message: err.Error()})
	}

	if newAppConfig.FleetDesktop.TransparencyURL != "" {
		if license.Tier != "premium" {
			invalid.Append("transparency_url", ErrMissingLicense.Error())
			return nil, ctxerr.Wrap(ctx, invalid)
		}
		if _, err := url.Parse(newAppConfig.FleetDesktop.TransparencyURL); err != nil {
			invalid.Append("transparency_url", err.Error())
			return nil, ctxerr.Wrap(ctx, invalid)
		}
	}

	validateSSOSettings(newAppConfig, appConfig, invalid, license)
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}

	// We apply the config that is incoming to the old one
	appConfig.EnableStrictDecoding()
	if err := json.Unmarshal(p, &appConfig); err != nil {
		return nil, ctxerr.Wrap(ctx, &badRequestError{message: err.Error()})
	}

	fleet.ValidateEnabledVulnerabilitiesIntegrations(appConfig.WebhookSettings.VulnerabilitiesWebhook, appConfig.Integrations, invalid)
	fleet.ValidateEnabledFailingPoliciesIntegrations(appConfig.WebhookSettings.FailingPoliciesWebhook, appConfig.Integrations, invalid)
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}

	// ignore the values for SMTPEnabled and SMTPConfigured
	oldSmtpSettings.SMTPEnabled = appConfig.SMTPSettings.SMTPEnabled
	oldSmtpSettings.SMTPConfigured = appConfig.SMTPSettings.SMTPConfigured

	// if we enable SMTP and the settings have changed, then we send a test email
	if appConfig.SMTPSettings.SMTPEnabled {
		if oldSmtpSettings != appConfig.SMTPSettings || !appConfig.SMTPSettings.SMTPConfigured {
			if err = svc.sendTestEmail(ctx, appConfig); err != nil {
				return nil, ctxerr.Wrap(ctx, err)
			}
		}
		appConfig.SMTPSettings.SMTPConfigured = true
	} else {
		appConfig.SMTPSettings.SMTPConfigured = false
	}

	delJira, err := fleet.ValidateJiraIntegrations(ctx, storedJiraByProjectKey, newAppConfig.Integrations.Jira)
	if err != nil {
		if errors.As(err, &fleet.IntegrationTestError{}) {
			return nil, ctxerr.Wrap(ctx, &badRequestError{message: err.Error()})
		}
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("Jira integration", err.Error()))
	}
	appConfig.Integrations.Jira = newAppConfig.Integrations.Jira

	delZendesk, err := fleet.ValidateZendeskIntegrations(ctx, storedZendeskByGroupID, newAppConfig.Integrations.Zendesk)
	if err != nil {
		if errors.As(err, &fleet.IntegrationTestError{}) {
			return nil, ctxerr.Wrap(ctx, &badRequestError{message: err.Error()})
		}
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("Zendesk integration", err.Error()))
	}
	appConfig.Integrations.Zendesk = newAppConfig.Integrations.Zendesk

	// if any integration was deleted, remove it from any team that uses it
	if len(delJira)+len(delZendesk) > 0 {
		if err := svc.ds.DeleteIntegrationsFromTeams(ctx, fleet.Integrations{Jira: delJira, Zendesk: delZendesk}); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete integrations from teams")
		}
	}

	// reset transparency url to empty for downgraded licenses
	if license.Tier != "premium" && appConfig.FleetDesktop.TransparencyURL != "" {
		appConfig.FleetDesktop.TransparencyURL = ""
	}

	if err := svc.ds.SaveAppConfig(ctx, appConfig); err != nil {
		return nil, err
	}

	// retrieve new app config with obfuscated secrets
	obfuscatedConfig, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	return obfuscatedConfig, nil
}

func validateSSOSettings(p fleet.AppConfig, existing *fleet.AppConfig, invalid *fleet.InvalidArgumentError, license *fleet.LicenseInfo) {
	if p.SSOSettings.EnableSSO {
		if p.SSOSettings.Metadata == "" && p.SSOSettings.MetadataURL == "" {
			if existing.SSOSettings.Metadata == "" && existing.SSOSettings.MetadataURL == "" {
				invalid.Append("metadata", "either metadata or metadata_url must be defined")
			}
		}
		if p.SSOSettings.Metadata != "" && p.SSOSettings.MetadataURL != "" {
			invalid.Append("metadata", "both metadata and metadata_url are defined, only one is allowed")
		}
		if p.SSOSettings.EntityID == "" {
			if existing.SSOSettings.EntityID == "" {
				invalid.Append("entity_id", "required")
			}
		} else {
			if len(p.SSOSettings.EntityID) < 5 {
				invalid.Append("entity_id", "must be 5 or more characters")
			}
		}
		if p.SSOSettings.IDPName == "" {
			if existing.SSOSettings.IDPName == "" {
				invalid.Append("idp_name", "required")
			}
		}
		if !license.IsPremium() && p.SSOSettings.EnableJITProvisioning {
			invalid.Append("enable_jit_provisioning", ErrMissingLicense.Error())
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Apply enroll secret spec
////////////////////////////////////////////////////////////////////////////////

type applyEnrollSecretSpecRequest struct {
	Spec *fleet.EnrollSecretSpec `json:"spec"`
}

type applyEnrollSecretSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyEnrollSecretSpecResponse) error() error { return r.Err }

func applyEnrollSecretSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*applyEnrollSecretSpecRequest)
	err := svc.ApplyEnrollSecretSpec(ctx, req.Spec)
	if err != nil {
		return applyEnrollSecretSpecResponse{Err: err}, nil
	}
	return applyEnrollSecretSpecResponse{}, nil
}

func (svc *Service) ApplyEnrollSecretSpec(ctx context.Context, spec *fleet.EnrollSecretSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionWrite); err != nil {
		return err
	}

	for _, s := range spec.Secrets {
		if s.Secret == "" {
			return ctxerr.New(ctx, "enroll secret must not be empty")
		}
	}

	if svc.config.Packaging.GlobalEnrollSecret != "" {
		return ctxerr.New(ctx, "enroll secret cannot be changed when fleet_packaging.global_enroll_secret is set")
	}

	return svc.ds.ApplyEnrollSecrets(ctx, nil, spec.Secrets)
}

////////////////////////////////////////////////////////////////////////////////
// Get enroll secret spec
////////////////////////////////////////////////////////////////////////////////

type getEnrollSecretSpecResponse struct {
	Spec *fleet.EnrollSecretSpec `json:"spec"`
	Err  error                   `json:"error,omitempty"`
}

func (r getEnrollSecretSpecResponse) error() error { return r.Err }

func getEnrollSecretSpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	specs, err := svc.GetEnrollSecretSpec(ctx)
	if err != nil {
		return getEnrollSecretSpecResponse{Err: err}, nil
	}
	return getEnrollSecretSpecResponse{Spec: specs}, nil
}

func (svc *Service) GetEnrollSecretSpec(ctx context.Context) (*fleet.EnrollSecretSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	secrets, err := svc.ds.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &fleet.EnrollSecretSpec{Secrets: secrets}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Version
////////////////////////////////////////////////////////////////////////////////

type versionResponse struct {
	*version.Info
	Err error `json:"error,omitempty"`
}

func (r versionResponse) error() error { return r.Err }

func versionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	info, err := svc.Version(ctx)
	if err != nil {
		return versionResponse{Err: err}, nil
	}
	return versionResponse{Info: info}, nil
}

func (svc *Service) Version(ctx context.Context) (*version.Info, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	info := version.Version()
	return &info, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Certificate Chain
////////////////////////////////////////////////////////////////////////////////

type getCertificateResponse struct {
	CertificateChain []byte `json:"certificate_chain"`
	Err              error  `json:"error,omitempty"`
}

func (r getCertificateResponse) error() error { return r.Err }

func getCertificateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	chain, err := svc.CertificateChain(ctx)
	if err != nil {
		return getCertificateResponse{Err: err}, nil
	}
	return getCertificateResponse{CertificateChain: chain}, nil
}

// Certificate returns the PEM encoded certificate chain for osqueryd TLS termination.
func (svc *Service) CertificateChain(ctx context.Context) ([]byte, error) {
	config, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(config.ServerSettings.ServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing serverURL")
	}

	conn, err := connectTLS(ctx, u)
	if err != nil {
		return nil, err
	}

	return chain(ctx, conn.ConnectionState(), u.Hostname())
}

func connectTLS(ctx context.Context, serverURL *url.URL) (*tls.Conn, error) {
	var hostport string
	if serverURL.Port() == "" {
		hostport = net.JoinHostPort(serverURL.Host, "443")
	} else {
		hostport = serverURL.Host
	}

	// attempt dialing twice, first with a secure conn, and then
	// if that fails, use insecure
	dial := func(insecure bool) (*tls.Conn, error) {
		conn, err := tls.Dial("tcp", hostport, &tls.Config{
			InsecureSkipVerify: insecure,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "dial tls")
		}
		defer conn.Close()
		return conn, nil
	}

	var (
		conn *tls.Conn
		err  error
	)

	conn, err = dial(false)
	if err == nil {
		return conn, nil
	}
	conn, err = dial(true)
	return conn, err
}

// chain builds a PEM encoded certificate chain using the PeerCertificates
// in tls.ConnectionState. chain uses the hostname to omit the Leaf certificate
// from the chain.
func chain(ctx context.Context, cs tls.ConnectionState, hostname string) ([]byte, error) {
	buf := bytes.NewBuffer([]byte(""))

	verifyEncode := func(chain []*x509.Certificate) error {
		for _, cert := range chain {
			if len(chain) > 1 {
				// drop the leaf certificate from the chain. osqueryd does not
				// need it to establish a secure connection
				if err := cert.VerifyHostname(hostname); err == nil {
					continue
				}
			}
			if err := encodePEMCertificate(buf, cert); err != nil {
				return err
			}
		}
		return nil
	}

	// use verified chains if available(which adds the root CA), otherwise
	// use the certificate chain offered by the server (if terminated with
	// self-signed certs)
	if len(cs.VerifiedChains) != 0 {
		for _, chain := range cs.VerifiedChains {
			if err := verifyEncode(chain); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "encode verified chains pem")
			}
		}
	} else {
		if err := verifyEncode(cs.PeerCertificates); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "encode peer certificates pem")
		}
	}
	return buf.Bytes(), nil
}

func encodePEMCertificate(buf io.Writer, cert *x509.Certificate) error {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.Encode(buf, block)
}
