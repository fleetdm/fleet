package service

import (
	"fmt"

	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/mail"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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
		Subject: "Hello from Kolide",
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
		if err = svc.SendTestEmail(ctx, config); err != nil {
			return nil, err
		}
		config.SMTPConfigured = true
	}

	if err := svc.ds.SaveAppConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func appConfigFromAppConfigPayload(p kolide.AppConfigPayload, config kolide.AppConfig) *kolide.AppConfig {
	if p.OrgInfo != nil && p.OrgInfo.OrgLogoURL != nil {
		config.OrgLogoURL = *p.OrgInfo.OrgLogoURL
	}
	if p.OrgInfo != nil && p.OrgInfo.OrgName != nil {
		config.OrgName = *p.OrgInfo.OrgName
	}
	if p.ServerSettings != nil && p.ServerSettings.KolideServerURL != nil {
		config.KolideServerURL = *p.ServerSettings.KolideServerURL
	}
	if p.ServerSettings != nil && p.ServerSettings.EnrollSecret != nil {
		config.EnrollSecret = *p.ServerSettings.EnrollSecret
	}

	populateSMTP := func(p *kolide.SMTPSettingsPayload) {
		if p.SMTPAuthenticationMethod != nil {
			switch *p.SMTPAuthenticationMethod {
			case kolide.AuthMethodNameCramMD5:
				config.SMTPAuthenticationMethod = kolide.AuthMethodCramMD5
			case kolide.AuthMethodNamePlain:
				config.SMTPAuthenticationMethod = kolide.AuthMethodPlain
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
