package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mail"
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
	if fromPayload.EnrollSecret == "" {
		// generate a random string if the user hasn't set one in the form.
		rand, err := kolide.RandomText(24)
		if err != nil {
			return nil, errors.Wrap(err, "generate enroll secret string")
		}
		fromPayload.EnrollSecret = rand
	}
	newConfig, err := svc.ds.NewAppConfig(fromPayload)
	if err != nil {
		return nil, err
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
		Mailer: &kolide.SMTPTestMailer{
			KolideServerURL: config.KolideServerURL,
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
	if p.ServerSettings != nil && p.ServerSettings.EnrollSecret != nil {
		config.EnrollSecret = *p.ServerSettings.EnrollSecret
	}

	if p.SSOSettings != nil {
		if p.SSOSettings.EnableSSO != nil {
			config.EnableSSO = *p.SSOSettings.EnableSSO
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

		if p.SMTPPassword != nil {
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
