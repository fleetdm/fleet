package service

import (
	"fmt"
	"reflect"

	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/mail"
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

func (svc service) ModifyAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	oldConfig, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	newConfig := appConfigFromAppConfigPayload(p, *oldConfig)
	smtpTest := p.SMTPTest != nil && *p.SMTPTest

	// if the request actually included SMTP settings, let's analyze the request
	// and figure out what is being requested
	if p.SMTPSettings != nil {
		oldSettings := smtpSettingsFromAppConfig(oldConfig)

		// have the setting been updated or is the user trying to send a test email
		if !reflect.DeepEqual(oldSettings, p.SMTPSettings) || smtpTest {
			// make a test email and send it to the user who is changing settings
			vc, ok := viewer.FromContext(ctx)
			if !ok {
				return nil, errNoContext
			}
			testMail := kolide.Email{
				Subject: "Hello from Kolide",
				To:      []string{vc.User.Email},
				Mailer: &kolide.SMTPTestMailer{
					KolideServerURL: newConfig.KolideServerURL,
				},
				Config: newConfig,
			}

			// try sending the test email
			err = mail.Test(svc.mailService, testMail)
			if err != nil {
				// if there is an error, set configured to false
				newConfig.SMTPConfigured = false

				if smtpTest {
					// if we're sending a test email, return: the test failed
					return nil, mailError{
						message: err.Error(),
					}
				}
			} else {
				// if there was not an error sending the test email, set
				// configured to true
				newConfig.SMTPConfigured = true
			}
		}
	}
	// if, this whole time, we've been sending a test email, then we don't want
	// to actually save anything, so we just return the fake object
	if smtpTest {
		return newConfig, nil
	}

	// if this was not a test email, then we save the config to the database and
	// return it
	if err = svc.ds.SaveAppConfig(newConfig); err != nil {
		return nil, err
	}
	return newConfig, nil
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
