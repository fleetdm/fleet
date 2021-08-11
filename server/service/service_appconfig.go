package service

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/kolide/kit/version"
	"github.com/pkg/errors"
)

// mailError is set when an error performing mail operations
type mailError struct {
	message string
}

func (e mailError) Error() string {
	return fmt.Sprintf("a mail error occurred: %s", e.message)
}

func (e mailError) MailError() []map[string]string {
	return []map[string]string{
		{
			"name":   "base",
			"reason": e.message,
		},
	}
}

func (svc *Service) NewAppConfig(ctx context.Context, p fleet.AppConfigPayload) (*fleet.AppConfig, error) {
	// skipauth: No user context yet when the app config is first created.
	svc.authz.SkipAuthorization(ctx)

	config, err := svc.ds.AppConfig()
	if err != nil {
		return nil, err
	}
	fromPayload := appConfigFromAppConfigPayload(p, *config)

	// Usage analytics are on by default in new installations.
	fromPayload.EnableAnalytics = true

	newConfig, err := svc.ds.NewAppConfig(fromPayload)
	if err != nil {
		return nil, err
	}

	// Set up a default enroll secret
	secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
	if err != nil {
		return nil, errors.Wrap(err, "generate enroll secret string")
	}
	secrets := []*fleet.EnrollSecret{
		{
			Secret: secret,
		},
	}
	err = svc.ds.ApplyEnrollSecrets(nil, secrets)
	if err != nil {
		return nil, errors.Wrap(err, "save enroll secret")
	}

	return newConfig, nil
}

func (svc *Service) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.AppConfig()
}

func (svc *Service) sendTestEmail(ctx context.Context, config *fleet.AppConfig) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	testMail := fleet.Email{
		Subject: "Hello from Fleet",
		To:      []string{vc.User.Email},
		Mailer: &mail.SMTPTestMailer{
			BaseURL:  template.URL(config.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
		Config: config,
	}

	if err := mail.Test(svc.mailService, testMail); err != nil {
		return mailError{message: err.Error()}
	}
	return nil

}

func (svc *Service) ModifyAppConfig(ctx context.Context, p fleet.AppConfigPayload) (*fleet.AppConfig, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	oldAppConfig, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	config := appConfigFromAppConfigPayload(p, *oldAppConfig)

	if p.SMTPSettings != nil {
		enabled := p.SMTPSettings.SMTPEnabled
		if (enabled == nil && oldAppConfig.SMTPConfigured) || (enabled != nil && *enabled) {
			if err = svc.sendTestEmail(ctx, config); err != nil {
				return nil, err
			}
			config.SMTPConfigured = true
		} else if enabled != nil && !*enabled {
			config.SMTPConfigured = false
		}
	}

	if err := svc.ds.SaveAppConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func cleanupURL(url string) string {
	return strings.TrimRight(strings.Trim(url, " \t\n"), "/")
}

func appConfigFromAppConfigPayload(p fleet.AppConfigPayload, config fleet.AppConfig) *fleet.AppConfig {
	if p.OrgInfo != nil && p.OrgInfo.OrgLogoURL != nil {
		config.OrgLogoURL = *p.OrgInfo.OrgLogoURL
	}
	if p.OrgInfo != nil && p.OrgInfo.OrgName != nil {
		config.OrgName = *p.OrgInfo.OrgName
	}
	if p.ServerSettings != nil {
		if p.ServerSettings.ServerURL != nil {
			config.ServerURL = cleanupURL(*p.ServerSettings.ServerURL)
		}
		if p.ServerSettings.LiveQueryDisabled != nil {
			config.LiveQueryDisabled = *p.ServerSettings.LiveQueryDisabled
		}
		if p.ServerSettings.EnableAnalytics != nil {
			config.EnableAnalytics = *p.ServerSettings.EnableAnalytics
		}
	}

	if p.SSOSettings != nil {
		if p.SSOSettings.EnableSSO != nil {
			config.EnableSSO = *p.SSOSettings.EnableSSO
		}
		if p.SSOSettings.EnableSSOIdPLogin != nil {
			config.EnableSSOIdPLogin = *p.SSOSettings.EnableSSOIdPLogin
		}
		if p.SSOSettings.EntityID != nil {
			config.EntityID = *p.SSOSettings.EntityID
		}
		if p.SSOSettings.IDPImageURL != nil {
			config.IDPImageURL = *p.SSOSettings.IDPImageURL
		}
		if p.SSOSettings.IDPName != nil {
			config.IDPName = *p.SSOSettings.IDPName
		}
		if p.SSOSettings.IssuerURI != nil {
			config.IssuerURI = *p.SSOSettings.IssuerURI
		}
		if p.SSOSettings.Metadata != nil {
			config.Metadata = *p.SSOSettings.Metadata
		}
		if p.SSOSettings.MetadataURL != nil {
			config.MetadataURL = *p.SSOSettings.MetadataURL
		}
	}

	if p.HostExpirySettings != nil {
		if p.HostExpirySettings.HostExpiryEnabled != nil {
			config.HostExpiryEnabled = *p.HostExpirySettings.HostExpiryEnabled
		}
		if p.HostExpirySettings.HostExpiryWindow != nil {
			config.HostExpiryWindow = *p.HostExpirySettings.HostExpiryWindow
		}
	}

	if settings := p.HostSettings; settings != nil {
		config.AdditionalQueries = settings.AdditionalQueries
		if settings.EnableHostUsers != nil {
			config.EnableHostUsers = *settings.EnableHostUsers
		}
		if settings.EnableSoftwareInventory != nil {
			config.EnableSoftwareInventory = *settings.EnableSoftwareInventory
		}
	} else {
		config.AdditionalQueries = nil
	}

	if p.AgentOptions != nil {
		config.AgentOptions = p.AgentOptions
	}

	populateSMTP := func(p *fleet.SMTPSettingsPayload) {
		if p.SMTPAuthenticationMethod != nil {
			switch *p.SMTPAuthenticationMethod {
			case fleet.AuthMethodNameCramMD5:
				config.SMTPAuthenticationMethod = fleet.AuthMethodCramMD5
			case fleet.AuthMethodNamePlain:
				config.SMTPAuthenticationMethod = fleet.AuthMethodPlain
			case fleet.AuthMethodNameLogin:
				config.SMTPAuthenticationMethod = fleet.AuthMethodLogin
			default:
				panic("unknown SMTP AuthMethod: " + *p.SMTPAuthenticationMethod)
			}
		}
		if p.SMTPAuthenticationType != nil {
			switch *p.SMTPAuthenticationType {
			case fleet.AuthTypeNameUserNamePassword:
				config.SMTPAuthenticationType = fleet.AuthTypeUserNamePassword
			case fleet.AuthTypeNameNone:
				config.SMTPAuthenticationType = fleet.AuthTypeNone
			default:
				panic("unknown SMTP AuthType: " + *p.SMTPAuthenticationType)
			}
		}

		if p.SMTPDomain != nil {
			config.SMTPDomain = *p.SMTPDomain
		}

		if p.SMTPEnableStartTLS != nil {
			config.SMTPEnableStartTLS = *p.SMTPEnableStartTLS
		}

		if p.SMTPEnableTLS != nil {
			config.SMTPEnableTLS = *p.SMTPEnableTLS
		}

		if p.SMTPPassword != nil && *p.SMTPPassword != "********" {
			config.SMTPPassword = *p.SMTPPassword
		}

		if p.SMTPPort != nil {
			config.SMTPPort = *p.SMTPPort
		}

		if p.SMTPSenderAddress != nil {
			config.SMTPSenderAddress = *p.SMTPSenderAddress
		}

		if p.SMTPServer != nil {
			config.SMTPServer = *p.SMTPServer
		}

		if p.SMTPUserName != nil {
			config.SMTPUserName = *p.SMTPUserName
		}

		if p.SMTPVerifySSLCerts != nil {
			config.SMTPVerifySSLCerts = *p.SMTPVerifySSLCerts
		}
	}

	if p.SMTPSettings != nil {
		populateSMTP(p.SMTPSettings)
	}

	if p.VulnerabilitySettings != nil {
		config.VulnerabilityDatabasesPath = ptr.String(p.VulnerabilitySettings.DatabasesPath)
	} else {
		config.VulnerabilityDatabasesPath = nil
	}

	return &config
}

func (svc *Service) ApplyEnrollSecretSpec(ctx context.Context, spec *fleet.EnrollSecretSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionWrite); err != nil {
		return err
	}

	for _, s := range spec.Secrets {
		if s.Secret == "" {
			return errors.New("enroll secret must not be empty")
		}
	}

	return svc.ds.ApplyEnrollSecrets(nil, spec.Secrets)
}

func (svc *Service) GetEnrollSecretSpec(ctx context.Context) (*fleet.EnrollSecretSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	secrets, err := svc.ds.GetEnrollSecrets(nil)
	if err != nil {
		return nil, err
	}
	return &fleet.EnrollSecretSpec{Secrets: secrets}, nil
}

func (svc *Service) Version(ctx context.Context) (*version.Info, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	info := version.Version()
	return &info, nil
}

func (svc *Service) License(ctx context.Context) (*fleet.LicenseInfo, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return &svc.license, nil
}

func (svc *Service) SetupRequired(ctx context.Context) (bool, error) {
	users, err := svc.ds.ListUsers(fleet.UserListOptions{ListOptions: fleet.ListOptions{Page: 0, PerPage: 1}})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return true, nil
	}
	return false, nil
}

func (svc *Service) LoggingConfig(ctx context.Context) (*fleet.Logging, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	conf := svc.config
	logging := &fleet.Logging{
		Debug: conf.Logging.Debug,
		Json:  conf.Logging.JSON,
	}

	switch conf.Osquery.StatusLogPlugin {
	case "", "filesystem":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "filesystem",
			Config: fleet.FilesystemConfig{FilesystemConfig: conf.Filesystem},
		}
	case "kinesis":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "kinesis",
			Config: fleet.KinesisConfig{
				Region:       conf.Kinesis.Region,
				StatusStream: conf.Kinesis.StatusStream,
				ResultStream: conf.Kinesis.ResultStream,
			},
		}
	case "firehose":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "firehose",
			Config: fleet.FirehoseConfig{
				Region:       conf.Firehose.Region,
				StatusStream: conf.Firehose.StatusStream,
				ResultStream: conf.Firehose.ResultStream,
			},
		}
	case "lambda":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "lambda",
			Config: fleet.LambdaConfig{
				Region:         conf.Lambda.Region,
				StatusFunction: conf.Lambda.StatusFunction,
				ResultFunction: conf.Lambda.ResultFunction,
			},
		}
	case "pubsub":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "pubsub",
			Config: fleet.PubSubConfig{PubSubConfig: conf.PubSub},
		}
	case "stdout":
		logging.Status = fleet.LoggingPlugin{Plugin: "stdout"}
	default:
		return nil, errors.Errorf("unrecognized logging plugin: %s", conf.Osquery.StatusLogPlugin)
	}

	switch conf.Osquery.ResultLogPlugin {
	case "", "filesystem":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "filesystem",
			Config: fleet.FilesystemConfig{FilesystemConfig: conf.Filesystem},
		}
	case "kinesis":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "kinesis",
			Config: fleet.KinesisConfig{
				Region:       conf.Kinesis.Region,
				StatusStream: conf.Kinesis.StatusStream,
				ResultStream: conf.Kinesis.ResultStream,
			},
		}
	case "firehose":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "firehose",
			Config: fleet.FirehoseConfig{
				Region:       conf.Firehose.Region,
				StatusStream: conf.Firehose.StatusStream,
				ResultStream: conf.Firehose.ResultStream,
			},
		}
	case "lambda":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "lambda",
			Config: fleet.LambdaConfig{
				Region:         conf.Lambda.Region,
				StatusFunction: conf.Lambda.StatusFunction,
				ResultFunction: conf.Lambda.ResultFunction,
			},
		}
	case "pubsub":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "pubsub",
			Config: fleet.PubSubConfig{PubSubConfig: conf.PubSub},
		}
	case "stdout":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "stdout",
		}
	default:
		return nil, errors.Errorf("unrecognized logging plugin: %s", conf.Osquery.ResultLogPlugin)

	}
	return logging, nil
}
