package service

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/kolide"
)

type appConfigRequest struct {
	Payload kolide.AppConfigPayload
}

type appConfigResponse struct {
	OrgInfo        *kolide.OrgInfo             `json:"org_info,omitemtpy"`
	ServerSettings *kolide.ServerSettings      `json:"server_settings,omitempty"`
	SMTPSettings   *kolide.SMTPSettingsPayload `json:"smtp_settings,omitempty"`
	SSOSettings    *kolide.SSOSettingsPayload  `json:"sso_settings,omitempty"`
	Err            error                       `json:"error,omitempty"`
}

func (r appConfigResponse) error() error { return r.Err }

func makeGetAppConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errors.New("could not fetch user")
		}
		config, err := svc.AppConfig(ctx)
		if err != nil {
			return nil, err
		}
		var smtpSettings *kolide.SMTPSettingsPayload
		var ssoSettings *kolide.SSOSettingsPayload
		// only admin can see smtp settings
		if vc.CanPerformAdminActions() {
			smtpSettings = smtpSettingsFromAppConfig(config)
			if smtpSettings.SMTPPassword != nil {
				*smtpSettings.SMTPPassword = "********"
			}
			ssoSettings = &kolide.SSOSettingsPayload{
				EntityID:    &config.EntityID,
				IssuerURI:   &config.IssuerURI,
				IDPImageURL: &config.IDPImageURL,
				Metadata:    &config.Metadata,
				MetadataURL: &config.MetadataURL,
				IDPName:     &config.IDPName,
				EnableSSO:   &config.EnableSSO,
			}
		}
		response := appConfigResponse{
			OrgInfo: &kolide.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			ServerSettings: &kolide.ServerSettings{
				KolideServerURL: &config.KolideServerURL,
				EnrollSecret:    &config.EnrollSecret,
			},
			SMTPSettings: smtpSettings,
			SSOSettings:  ssoSettings,
		}
		return response, nil
	}
}

func makeModifyAppConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(appConfigRequest)
		config, err := svc.ModifyAppConfig(ctx, req.Payload)
		if err != nil {
			return appConfigResponse{Err: err}, nil
		}
		response := appConfigResponse{
			OrgInfo: &kolide.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			ServerSettings: &kolide.ServerSettings{
				KolideServerURL: &config.KolideServerURL,
				EnrollSecret:    &config.EnrollSecret,
			},
			SMTPSettings: smtpSettingsFromAppConfig(config),
			SSOSettings: &kolide.SSOSettingsPayload{
				EntityID:    &config.EntityID,
				IssuerURI:   &config.IssuerURI,
				IDPImageURL: &config.IDPImageURL,
				Metadata:    &config.Metadata,
				MetadataURL: &config.MetadataURL,
				IDPName:     &config.IDPName,
				EnableSSO:   &config.EnableSSO,
			},
		}
		if response.SMTPSettings.SMTPPassword != nil {
			*response.SMTPSettings.SMTPPassword = "********"
		}
		return response, nil
	}
}

func smtpSettingsFromAppConfig(config *kolide.AppConfig) *kolide.SMTPSettingsPayload {
	authType := config.SMTPAuthenticationType.String()
	authMethod := config.SMTPAuthenticationMethod.String()
	return &kolide.SMTPSettingsPayload{
		SMTPEnabled:              &config.SMTPConfigured,
		SMTPConfigured:           &config.SMTPConfigured,
		SMTPSenderAddress:        &config.SMTPSenderAddress,
		SMTPServer:               &config.SMTPServer,
		SMTPPort:                 &config.SMTPPort,
		SMTPAuthenticationType:   &authType,
		SMTPUserName:             &config.SMTPUserName,
		SMTPPassword:             &config.SMTPPassword,
		SMTPEnableTLS:            &config.SMTPEnableTLS,
		SMTPAuthenticationMethod: &authMethod,
		SMTPDomain:               &config.SMTPDomain,
		SMTPVerifySSLCerts:       &config.SMTPVerifySSLCerts,
		SMTPEnableStartTLS:       &config.SMTPEnableStartTLS,
	}
}
