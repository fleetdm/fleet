package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (mw validationMiddleware) ModifyAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	invalid := &invalidArgumentError{}
	if p.ServerSettings == nil {
		invalid.Append("server_settings", "missing")
	}
	if p.ServerSettings != nil && p.ServerSettings.KolideServerURL == nil {
		invalid.Append("kolide_server_url", "missing")
	}
	if p.ServerSettings != nil && p.ServerSettings.KolideServerURL != nil {
		if err := validateKolideServerURL(*p.ServerSettings.KolideServerURL); err != nil {
			invalid.Append("kolide_server_url", err.Error())
		}
	}
	if p.SMTPSettings == nil {
		invalid.Append("smtp_settings", "missing")
	}
	if p.SMTPSettings != nil && !p.SMTPSettings.SMTPDisabled {
		if p.SMTPSettings.SMTPSenderAddress != "" {
			invalid.Append("smtp_sender_address", "required argument")
		}
		if p.SMTPSettings.SMTPServer == "" {
			invalid.Append("smtp_server", "required argument")
		}
		if p.SMTPSettings.SMTPAuthenticationType != kolide.AuthTypeUserNamePassword &&
			p.SMTPSettings.SMTPAuthenticationType != kolide.AuthTypeNone {
			invalid.Append("smtp_authentication_type", "invalid value")
		}
		if p.SMTPSettings.SMTPAuthenticationType == kolide.AuthTypeUserNamePassword {
			if p.SMTPSettings.SMTPAuthenticationMethod != kolide.AuthMethodCramMD5 &&
				p.SMTPSettings.SMTPAuthenticationMethod != kolide.AuthMethodPlain {
				invalid.Append("smtp_authentication_method", "invalid value")
			}
			if p.SMTPSettings.SMTPUserName == "" {
				invalid.Append("smtp_user_name", "required argument")
			}
			if p.SMTPSettings.SMTPPassword == "" {
				invalid.Append("smtp_password", "required argument")
			}
		}
	}
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.ModifyAppConfig(ctx, p)
}
