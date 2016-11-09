package service

import (
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
)

type setupRequest struct {
	Admin           *kolide.UserPayload `json:"admin"`
	OrgInfo         *kolide.OrgInfo     `json:"org_info"`
	KolideServerURL *string             `json:"kolide_server_url"`
}

type setupResponse struct {
	Admin           *kolide.User    `json:"admin,omitempty"`
	OrgInfo         *kolide.OrgInfo `json:"org_info,omitempty"`
	KolideServerURL *string         `json:"kolide_server_url"`
	Err             error           `json:"error,omitempty"`
}

func (r setupResponse) error() error { return r.Err }

func makeSetupEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			admin         *kolide.User
			config        *kolide.AppConfig
			configPayload kolide.AppConfigPayload
			err           error
		)
		req := request.(setupRequest)
		if req.Admin != nil {
			admin, err = svc.NewAdminCreatedUser(ctx, *req.Admin)
			if err != nil {
				return setupResponse{Err: err}, nil
			}
		}

		if req.OrgInfo != nil {
			configPayload.OrgInfo = req.OrgInfo
		}
		if req.KolideServerURL != nil {
			configPayload.ServerSettings = &kolide.ServerSettings{KolideServerURL: req.KolideServerURL}
		}
		config, err = svc.NewAppConfig(ctx, configPayload)
		if err != nil {
			return setupResponse{Err: err}, nil
		}
		return setupResponse{
			Admin: admin,
			OrgInfo: &kolide.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			KolideServerURL: &config.KolideServerURL,
		}, nil
	}
}
