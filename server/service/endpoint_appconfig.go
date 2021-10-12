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
	Payload json.RawMessage
}

type appConfigResponse struct {
	fleet.AppConfig

	UpdateInterval  *fleet.UpdateIntervalConfig  `json:"update_interval"`
	Vulnerabilities *fleet.VulnerabilitiesConfig `json:"vulnerabilities"`

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
		updateIntervalConfig, err := svc.UpdateIntervalConfig(ctx)
		if err != nil {
			return nil, err
		}
		vulnConfig, err := svc.VulnerabilitiesConfig(ctx)
		if err != nil {
			return nil, err
		}

		var smtpSettings fleet.SMTPSettings
		var ssoSettings fleet.SSOSettings
		var hostExpirySettings fleet.HostExpirySettings
		var agentOptions *json.RawMessage
		// only admin can see smtp, sso, and host expiry settings
		if vc.User.GlobalRole != nil && *vc.User.GlobalRole == fleet.RoleAdmin {
			smtpSettings = config.SMTPSettings
			if smtpSettings.SMTPPassword != "" {
				smtpSettings.SMTPPassword = "********"
			}
			ssoSettings = config.SSOSettings
			hostExpirySettings = config.HostExpirySettings
			agentOptions = config.AgentOptions
		}
		hostSettings := config.HostSettings
		response := appConfigResponse{
			AppConfig: fleet.AppConfig{
				OrgInfo:               config.OrgInfo,
				ServerSettings:        config.ServerSettings,
				HostSettings:          hostSettings,
				VulnerabilitySettings: config.VulnerabilitySettings,

				SMTPSettings:       smtpSettings,
				SSOSettings:        ssoSettings,
				HostExpirySettings: hostExpirySettings,
				AgentOptions:       agentOptions,

				WebhookSettings: config.WebhookSettings,
			},
			UpdateInterval:  updateIntervalConfig,
			Vulnerabilities: vulnConfig,
			License:         license,
			Logging:         loggingConfig,
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

		if response.SMTPSettings.SMTPPassword != "" {
			response.SMTPSettings.SMTPPassword = "********"
		}
		return response, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Apply enroll secret spec
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
// Get enroll secret spec
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
