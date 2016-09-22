package server

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

// getAppConfig is used to return
// current configuration data to the client
type getAppConfigResponse struct {
	Err error `json:"error,omitempty"`
}

func (r getAppConfigResponse) error() error { return r.Err }

type appConfig map[string]map[string]string

func makeGetAppConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		info, err := svc.OrgInfo(ctx)
		if err != nil {
			return getAppConfigResponse{Err: err}, nil
		}
		config := appConfig{
			"org_info": map[string]string{
				"org_name":     info.OrgName,
				"org_logo_url": info.OrgLogoURL,
			},
		}
		return config, nil
	}
}

type modifyAppConfigRequest struct {
	OrgPayload kolide.OrgInfoPayload `json:"org_info"`
}

func makeModifyAppConfigRequest(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyAppConfigRequest)
		info, err := svc.ModifyOrgInfo(ctx, req.OrgPayload)
		if err != nil {
			return getAppConfigResponse{Err: err}, nil
		}
		config := appConfig{
			"org_info": map[string]string{
				"org_name":     info.OrgName,
				"org_logo_url": info.OrgLogoURL,
			},
		}
		return config, nil
	}
}
