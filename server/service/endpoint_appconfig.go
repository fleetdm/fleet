package service

import (
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

type appConfigResponse struct {
	OrgInfo        *kolide.OrgInfo        `json:"org_info,omitemtpy"`
	ServerSettings *kolide.ServerSettings `json:"server_settings,omitempty"`
	SMTPSettings   *kolide.SMTPSettings   `json:"smtp_settings,omitempty"`
	Err            error                  `json:"error,omitempty"`
	// SMTPTestError if present gives reason smtp test failed
	SMTPTestError string `json:"smtp_test_error,omitempty"`
}

func (r appConfigResponse) error() error { return r.Err }

func makeGetAppConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		vc, ok := viewer.FromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("could not fetch user")
		}
		config, err := svc.AppConfig(ctx)
		if err != nil {
			return nil, err
		}
		var smtpSettings *kolide.SMTPSettings
		// only admin can see smtp settings
		if vc.IsAdmin() {
			smtpSettings = smtpSettingsFromAppConfig(config)
		}
		response := appConfigResponse{
			OrgInfo: &kolide.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			ServerSettings: &kolide.ServerSettings{
				KolideServerURL: &config.KolideServerURL,
			},
			SMTPSettings: smtpSettings,
		}
		return response, nil
	}
}

func makeModifyAppConfigRequest(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(kolide.AppConfigPayload)
		config, err := svc.ModifyAppConfig(ctx, req)
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
			},
			SMTPSettings:  smtpSettingsFromAppConfig(config),
			SMTPTestError: config.SMTPLastError,
		}
		return response, nil
	}
}
