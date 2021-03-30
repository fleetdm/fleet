package service

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mail"
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
		map[string]string{
			"name":   "base",
			"reason": e.message,
		},
	}
}

func (svc service) NewAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	config, err := svc.ds.AppConfig()
	if err != nil {
		return nil, err
	}
	fromPayload := appConfigFromAppConfigPayload(p, *config)

	newConfig, err := svc.ds.NewAppConfig(fromPayload)
	if err != nil {
		return nil, err
	}

	// Set up a default enroll secret
	secret, err := kolide.RandomText(24)
	if err != nil {
		return nil, errors.Wrap(err, "generate enroll secret string")
	}
	spec := &kolide.EnrollSecretSpec{
		Secrets: []kolide.EnrollSecret{
			kolide.EnrollSecret{
				Name:   "default",
				Secret: secret,
				Active: true,
			},
		},
	}
	err = svc.ds.ApplyEnrollSecretSpec(spec)
	if err != nil {
		return nil, errors.Wrap(err, "save enroll secret")
	}

	return newConfig, nil
}

func (svc service) AppConfig(ctx context.Context) (*kolide.AppConfig, error) {
	return svc.ds.AppConfig()
}

func (svc service) SendTestEmail(ctx context.Context, config *kolide.AppConfig) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errNoContext
	}

	testMail := kolide.Email{
		Subject: "Hello from Fleet",
		To:      []string{vc.User.Email},
		Mailer: &mail.SMTPTestMailer{
			BaseURL:  template.URL(config.KolideServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
		Config: config,
	}

	if err := mail.Test(svc.mailService, testMail); err != nil {
		return mailError{message: err.Error()}
	}
	return nil

}

func (svc service) ModifyAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	oldAppConfig, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	config := appConfigFromAppConfigPayload(p, *oldAppConfig)

	if p.SMTPSettings != nil {
		enabled := p.SMTPSettings.SMTPEnabled
		if (enabled == nil && oldAppConfig.SMTPConfigured) || (enabled != nil && *enabled) {
			if err = svc.SendTestEmail(ctx, config); err != nil {
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

func appConfigFromAppConfigPayload(p kolide.AppConfigPayload, config kolide.AppConfig) *kolide.AppConfig {
	if p.OrgInfo != nil && p.OrgInfo.OrgLogoURL != nil {
		config.OrgLogoURL = *p.OrgInfo.OrgLogoURL
	}
	if p.OrgInfo != nil && p.OrgInfo.OrgName != nil {
		config.OrgName = *p.OrgInfo.OrgName
	}
	if p.ServerSettings != nil && p.ServerSettings.KolideServerURL != nil {
		config.KolideServerURL = cleanupURL(*p.ServerSettings.KolideServerURL)
	}
	if p.ServerSettings != nil && p.ServerSettings.LiveQueryDisabled != nil {
		config.LiveQueryDisabled = *p.ServerSettings.LiveQueryDisabled
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
		if settings.AdditionalQueries != nil {
			config.AdditionalQueries = settings.AdditionalQueries
		}
	}

	populateSMTP := func(p *kolide.SMTPSettingsPayload) {
		if p.SMTPAuthenticationMethod != nil {
			switch *p.SMTPAuthenticationMethod {
			case kolide.AuthMethodNameCramMD5:
				config.SMTPAuthenticationMethod = kolide.AuthMethodCramMD5
			case kolide.AuthMethodNamePlain:
				config.SMTPAuthenticationMethod = kolide.AuthMethodPlain
			case kolide.AuthMethodNameLogin:
				config.SMTPAuthenticationMethod = kolide.AuthMethodLogin
			default:
				panic("unknown SMTP AuthMethod: " + *p.SMTPAuthenticationMethod)
			}
		}
		if p.SMTPAuthenticationType != nil {
			switch *p.SMTPAuthenticationType {
			case kolide.AuthTypeNameUserNamePassword:
				config.SMTPAuthenticationType = kolide.AuthTypeUserNamePassword
			case kolide.AuthTypeNameNone:
				config.SMTPAuthenticationType = kolide.AuthTypeNone
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
	return &config
}

func (svc service) ApplyEnrollSecretSpec(ctx context.Context, spec *kolide.EnrollSecretSpec) error {
	for _, s := range spec.Secrets {
		if s.Name == "" {
			return errors.New("enroll secret name must not be empty")
		}
		if s.Secret == "" {
			return errors.New("enroll secret must not be empty")
		}
	}

	return svc.ds.ApplyEnrollSecretSpec(spec)
}

func (svc service) GetEnrollSecretSpec(ctx context.Context) (*kolide.EnrollSecretSpec, error) {
	return svc.ds.GetEnrollSecretSpec()
}

func (svc service) Version(ctx context.Context) (*version.Info, error) {
	info := version.Version()
	return &info, nil
}
