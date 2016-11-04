package service

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

// getAppConfig is used to return
// current configuration data to the client
type getAppConfigResponse struct {
	Err error `json:"error,omitempty"`
}

func (r getAppConfigResponse) error() error { return r.Err }

func makeGetAppConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.AppConfig(ctx)
		if err != nil {
			return getAppConfigResponse{Err: err}, nil
		}
		response := appConfigPayload(*config)
		return response, nil
	}
}

type modifyAppConfigRequest struct {
	ConfigPayload kolide.AppConfigPayload
}

func makeModifyAppConfigRequest(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyAppConfigRequest)
		config, err := svc.ModifyAppConfig(ctx, req.ConfigPayload)
		if err != nil {
			return getAppConfigResponse{Err: err}, nil
		}
		response := appConfigPayload(*config)
		return response, nil
	}
}

func appConfigPayload(config kolide.AppConfig) kolide.AppConfigPayload {
	orgInfo := func() *kolide.OrgInfo {
		if config.OrgName == "" && config.OrgLogoURL == "" {
			return nil
		}
		return &kolide.OrgInfo{
			OrgName:    nilString(config.OrgName),
			OrgLogoURL: nilString(config.OrgLogoURL),
		}
	}

	serverSettings := func() *kolide.ServerSettings {
		if config.KolideServerURL == "" {
			return nil
		}
		return &kolide.ServerSettings{
			KolideServerURL: nilString(config.KolideServerURL),
		}
	}

	return kolide.AppConfigPayload{
		OrgInfo:        orgInfo(),
		ServerSettings: serverSettings(),
	}
}

func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
