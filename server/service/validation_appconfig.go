package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (mw validationMiddleware) ModifyAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	invalid := &invalidArgumentError{}

	if p.ServerSettings.KolideServerURL == nil || *p.ServerSettings.KolideServerURL == "" {
		invalid.Append("kolide_server_url", "missing")
	}
	if p.ServerSettings.KolideServerURL != nil && *p.ServerSettings.KolideServerURL != "" {
		if err := validateKolideServerURL(*p.ServerSettings.KolideServerURL); err != nil {
			invalid.Append("kolide_server_url", err.Error())
		}
	}

	if p.SMTPSettings.SMTPEnabled {
		if p.SMTPSettings.SMTPSenderAddress != "" {
			invalid.Append("smtp_sender_address", "required argument")
		}
		if p.SMTPSettings.SMTPServer == "" {
			invalid.Append("smtp_server", "required argument")
		}
		if p.SMTPSettings.SMTPAuthenticationType != kolide.AuthTypeNameUserNamePassword &&
			p.SMTPSettings.SMTPAuthenticationType != kolide.AuthTypeNameNone {
			invalid.Append("smtp_authentication_type", "invalid value")
		}
		if p.SMTPSettings.SMTPAuthenticationType == kolide.AuthTypeNameUserNamePassword {
			if p.SMTPSettings.SMTPAuthenticationMethod != kolide.AuthMethodNameCramMD5 &&
				p.SMTPSettings.SMTPAuthenticationMethod != kolide.AuthMethodNamePlain {
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
