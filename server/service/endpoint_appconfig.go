package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kit/version"
)

type appConfigRequest struct {
	Payload fleet.AppConfig
}

type appConfigResponse struct {
	fleet.AppConfig

	// License is loaded from the service
	License *fleet.LicenseInfo `json:"license,omitempty"`
	// Logging is loaded on the fly rather than from the database.
	Logging *fleet.Logging `json:"logging,omitempty"`
	Err     error          `json:"error,omitempty"`
}

func (r appConfigResponse) error() error { return r.Err }

func makeGetAppConfigEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, errors.New("could not fetch user")
		}
		config, err := svc.AppConfig(ctx)
		if err != nil {
			return nil, err
		}
		license, err := svc.License(ctx)
		if err != nil {
			return nil, err
		}
		loggingConfig, err := svc.LoggingConfig(ctx)
		if err != nil {
			return nil, err
		}

		var smtpSettings *fleet.SMTPSettings
		var ssoSettings *fleet.SSOSettings
		var hostExpirySettings *fleet.HostExpirySettings
		var agentOptions *json.RawMessage
		// only admin can see smtp, sso, and host expiry settings
		if vc.User.GlobalRole != nil && *vc.User.GlobalRole == fleet.RoleAdmin {
			smtpSettings = &fleet.SMTPSettings{
				SMTPEnabled:              config.GetBoolPtr("smtp_settings.enable_smtp"),
				SMTPConfigured:           config.GetBoolPtr("smtp_settings.configured"),
				SMTPSenderAddress:        config.GetStringPtr("smtp_settings.sender_address"),
				SMTPServer:               config.GetStringPtr("smtp_settings.server"),
				SMTPPort:                 config.GetUintPtr("smtp_settings.port"),
				SMTPAuthenticationType:   config.GetStringPtr("smtp_settings.authentication_type"),
				SMTPUserName:             config.GetStringPtr("smtp_settings.user_name"),
				SMTPPassword:             config.GetStringPtr("smtp_settings.password"),
				SMTPEnableTLS:            config.GetBoolPtr("smtp_settings.enable_ssl_tls"),
				SMTPAuthenticationMethod: config.GetStringPtr("smtp_settings.authentication_method"),
				SMTPDomain:               config.GetStringPtr("smtp_settings.domain"),
				SMTPVerifySSLCerts:       config.GetBoolPtr("smtp_settings.verify_ssl_certs"),
				SMTPEnableStartTLS:       config.GetBoolPtr("smtp_settings.enable_start_tls"),
			}
			if smtpSettings.SMTPPassword != nil {
				*smtpSettings.SMTPPassword = "********"
			}
			ssoSettings = &fleet.SSOSettings{
				EntityID:          config.GetStringPtr("sso_settings.entity_id"),
				IssuerURI:         config.GetStringPtr("sso_settings.issuer_uri"),
				IDPImageURL:       config.GetStringPtr("sso_settings.idp_image_url"),
				Metadata:          config.GetStringPtr("sso_settings.metadata"),
				MetadataURL:       config.GetStringPtr("sso_settings.metadata_url"),
				IDPName:           config.GetStringPtr("sso_settings.idp_name"),
				EnableSSO:         config.GetBoolPtr("sso_settings.enable_sso"),
				EnableSSOIdPLogin: config.GetBoolPtr("sso_settings.enable_sso_idp_login"),
			}
			hostExpirySettings = &fleet.HostExpirySettings{
				HostExpiryEnabled: config.GetBoolPtr("host_expiry_settings.host_expiry_enabled"),
				HostExpiryWindow:  config.GetIntPtr("host_expiry_settings.host_expiry_window"),
			}
			agentOptions = config.AgentOptions
		}
		hostSettings := &fleet.HostSettings{
			EnableHostUsers:         config.GetBoolPtr("host_settings.enable_host_users"),
			EnableSoftwareInventory: config.GetBoolPtr("host_settings.enable_software_inventory"),
			AdditionalQueries:       config.GetJSONPtr("host_settings.additional_queries"),
		}
		response := appConfigResponse{
			AppConfig: fleet.AppConfig{
				OrgInfo: &fleet.OrgInfo{
					OrgName:    config.GetStringPtr("org_info.org_name"),
					OrgLogoURL: config.GetStringPtr("org_info.org_logo_url"),
				},
				ServerSettings: &fleet.ServerSettings{
					ServerURL:         config.GetStringPtr("server_settings.server_url"),
					LiveQueryDisabled: config.GetBoolPtr("server_settings.live_query_disabled"),
					EnableAnalytics:   config.GetBoolPtr("server_settings.enable_analytics"),
				},
				HostSettings:          hostSettings,
				VulnerabilitySettings: config.VulnerabilitySettings,

				SMTPSettings:       smtpSettings,
				SSOSettings:        ssoSettings,
				HostExpirySettings: hostExpirySettings,
				AgentOptions:       agentOptions,
			},

			License: license,
			Logging: loggingConfig,
		}
		return response, nil
	}
}

func makeModifyAppConfigEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(appConfigRequest)
		config, err := svc.ModifyAppConfig(ctx, req.Payload)
		if err != nil {
			return appConfigResponse{Err: err}, nil
		}
		license, err := svc.License(ctx)
		if err != nil {
			return nil, err
		}
		loggingConfig, err := svc.LoggingConfig(ctx)
		if err != nil {
			return nil, err
		}
		response := appConfigResponse{
			AppConfig: *config,
			License:   license,
			Logging:   loggingConfig,
		}

		if response.GetString("smtp_settings.smtp_password") != "" {
			*response.SMTPSettings.SMTPPassword = "********"
		}
		return response, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Apply Enroll Secret Spec
////////////////////////////////////////////////////////////////////////////////

type applyEnrollSecretSpecRequest struct {
	Spec *fleet.EnrollSecretSpec `json:"spec"`
}

type applyEnrollSecretSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyEnrollSecretSpecResponse) error() error { return r.Err }

func makeApplyEnrollSecretSpecEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	Spec *fleet.EnrollSecretSpec `json:"spec"`
	Err  error                   `json:"error,omitempty"`
}

func (r getEnrollSecretSpecResponse) error() error { return r.Err }

func makeGetEnrollSecretSpecEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeVersionEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		info, err := svc.Version(ctx)
		if err != nil {
			return versionResponse{Err: err}, nil
		}
		return versionResponse{Info: info}, nil
	}
}
