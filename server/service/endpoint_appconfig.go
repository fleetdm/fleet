package service

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kit/version"
)

type appConfigRequest struct {
	Payload kolide.AppConfigPayload
}

type appConfigResponse struct {
	OrgInfo            *kolide.OrgInfo             `json:"org_info,omitempty"`
	ServerSettings     *kolide.ServerSettings      `json:"server_settings,omitempty"`
	SMTPSettings       *kolide.SMTPSettingsPayload `json:"smtp_settings,omitempty"`
	SSOSettings        *kolide.SSOSettingsPayload  `json:"sso_settings,omitempty"`
	HostExpirySettings *kolide.HostExpirySettings  `json:"host_expiry_settings,omitempty"`
	HostSettings       *kolide.HostSettings        `json:"host_settings,omitempty"`
	Err                error                       `json:"error,omitempty"`
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
		var hostExpirySettings *kolide.HostExpirySettings
		// only admin can see smtp, sso, and host expiry settings
		if vc.CanPerformAdminActions() {
			smtpSettings = smtpSettingsFromAppConfig(config)
			if smtpSettings.SMTPPassword != nil {
				*smtpSettings.SMTPPassword = "********"
			}
			ssoSettings = &kolide.SSOSettingsPayload{
				EntityID:          &config.EntityID,
				IssuerURI:         &config.IssuerURI,
				IDPImageURL:       &config.IDPImageURL,
				Metadata:          &config.Metadata,
				MetadataURL:       &config.MetadataURL,
				IDPName:           &config.IDPName,
				EnableSSO:         &config.EnableSSO,
				EnableSSOIdPLogin: &config.EnableSSOIdPLogin,
			}
			hostExpirySettings = &kolide.HostExpirySettings{
				HostExpiryEnabled: &config.HostExpiryEnabled,
				HostExpiryWindow:  &config.HostExpiryWindow,
			}
		}
		response := appConfigResponse{
			OrgInfo: &kolide.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			ServerSettings: &kolide.ServerSettings{
				KolideServerURL:   &config.KolideServerURL,
				LiveQueryDisabled: &config.LiveQueryDisabled,
			},
			SMTPSettings:       smtpSettings,
			SSOSettings:        ssoSettings,
			HostExpirySettings: hostExpirySettings,
			HostSettings: &kolide.HostSettings{
				AdditionalQueries: config.AdditionalQueries,
			},
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
				KolideServerURL:   &config.KolideServerURL,
				LiveQueryDisabled: &config.LiveQueryDisabled,
			},
			SMTPSettings: smtpSettingsFromAppConfig(config),
			SSOSettings: &kolide.SSOSettingsPayload{
				EntityID:          &config.EntityID,
				IssuerURI:         &config.IssuerURI,
				IDPImageURL:       &config.IDPImageURL,
				Metadata:          &config.Metadata,
				MetadataURL:       &config.MetadataURL,
				IDPName:           &config.IDPName,
				EnableSSO:         &config.EnableSSO,
				EnableSSOIdPLogin: &config.EnableSSOIdPLogin,
			},
			HostExpirySettings: &kolide.HostExpirySettings{
				HostExpiryEnabled: &config.HostExpiryEnabled,
				HostExpiryWindow:  &config.HostExpiryWindow,
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

////////////////////////////////////////////////////////////////////////////////
// Apply Enroll Secret Spec
////////////////////////////////////////////////////////////////////////////////

type applyEnrollSecretSpecRequest struct {
	Spec *kolide.EnrollSecretSpec `json:"spec"`
}

type applyEnrollSecretSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyEnrollSecretSpecResponse) error() error { return r.Err }

func makeApplyEnrollSecretSpecEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyEnrollSecretSpecRequest)
		err := svc.ApplyEnrollSecretSpec(ctx, req.Spec)
		if err != nil {
			return applyEnrollSecretSpecResponse{Err: err}, nil
		}
		return applyEnrollSecretSpecResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Enroll Secret Spec
////////////////////////////////////////////////////////////////////////////////

type getEnrollSecretSpecResponse struct {
	Spec *kolide.EnrollSecretSpec `json:"specs"`
	Err  error                    `json:"error,omitempty"`
}

func (r getEnrollSecretSpecResponse) error() error { return r.Err }

func makeGetEnrollSecretSpecEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		specs, err := svc.GetEnrollSecretSpec(ctx)
		if err != nil {
			return getEnrollSecretSpecResponse{Err: err}, nil
		}
		return getEnrollSecretSpecResponse{Spec: specs}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Version
////////////////////////////////////////////////////////////////////////////////

type versionResponse struct {
	*version.Info
	Err error `json:"error,omitempty"`
}

func (r versionResponse) error() error { return r.Err }

func makeVersionEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		info, err := svc.Version(ctx)
		if err != nil {
			return versionResponse{Err: err}, nil
		}
		return versionResponse{Info: info}, nil
	}
}
