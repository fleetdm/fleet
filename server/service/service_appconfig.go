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
	newConfig, err := svc.ds.NewAppConfig(appConfigFromAppConfigPayload(p, *config))
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
		return nil, errors.Wrap(err, "failed retrieving existing app config")
	}
	config := appConfigFromAppConfigPayload(p, *oldAppConfig)

	if p.SMTPSettings != nil {
		if err = svc.SendTestEmail(ctx, config); err != nil {
			err = errors.Wrap(err, "test email failed")
			config.SMTPConfigured = false
		} else {
			config.SMTPConfigured = true
		}
	}

	if err := svc.ds.SaveAppConfig(config); err != nil {
		err = errors.Wrap(err, "could not save config")
	}
	return config, err
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
	if p.SMTPSettings != nil {
		if p.SMTPSettings.SMTPAuthenticationMethod == kolide.AuthMethodNameCramMD5 {
			config.SMTPAuthenticationMethod = kolide.AuthMethodCramMD5
		} else {
			config.SMTPAuthenticationMethod = kolide.AuthMethodPlain
		}
		if p.SMTPSettings.SMTPAuthenticationType == kolide.AuthTypeNameUserNamePassword {
			config.SMTPAuthenticationType = kolide.AuthTypeUserNamePassword
		} else {
			config.SMTPAuthenticationType = kolide.AuthTypeNone
		}
		config.SMTPConfigured = p.SMTPSettings.SMTPConfigured
		config.SMTPDomain = p.SMTPSettings.SMTPDomain
		config.SMTPEnableStartTLS = p.SMTPSettings.SMTPEnableStartTLS
		config.SMTPEnableTLS = p.SMTPSettings.SMTPEnableTLS
		config.SMTPPassword = p.SMTPSettings.SMTPPassword
		config.SMTPPort = p.SMTPSettings.SMTPPort
		config.SMTPSenderAddress = p.SMTPSettings.SMTPSenderAddress
		config.SMTPServer = p.SMTPSettings.SMTPServer
		config.SMTPUserName = p.SMTPSettings.SMTPUserName
		config.SMTPVerifySSLCerts = p.SMTPSettings.SMTPVerifySSLCerts
	}
	return &config
}

func smtpSettingsFromAppConfig(config *kolide.AppConfig) *kolide.SMTPSettings {
	return &kolide.SMTPSettings{
		SMTPConfigured:           config.SMTPConfigured,
		SMTPSenderAddress:        config.SMTPSenderAddress,
		SMTPServer:               config.SMTPServer,
		SMTPPort:                 config.SMTPPort,
		SMTPAuthenticationType:   config.SMTPAuthenticationType.String(),
		SMTPUserName:             config.SMTPUserName,
		SMTPPassword:             config.SMTPPassword,
		SMTPEnableTLS:            config.SMTPEnableTLS,
		SMTPAuthenticationMethod: config.SMTPAuthenticationMethod.String(),
		SMTPDomain:               config.SMTPDomain,
		SMTPVerifySSLCerts:       config.SMTPVerifySSLCerts,
		SMTPEnableStartTLS:       config.SMTPEnableStartTLS,
	}
}

func appConfigPayloadFromAppConfig(config *kolide.AppConfig) *kolide.AppConfigPayload {
	return &kolide.AppConfigPayload{
		OrgInfo: &kolide.OrgInfo{
			OrgLogoURL: &config.OrgLogoURL,
			OrgName:    &config.OrgName,
		},
		ServerSettings: &kolide.ServerSettings{
			KolideServerURL: &config.KolideServerURL,
		},
		SMTPSettings: smtpSettingsFromAppConfig(config),
	}
}
