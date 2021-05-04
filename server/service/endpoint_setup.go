package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
	"github.com/pkg/errors"
)

type setupRequest struct {
	Admin           *kolide.UserPayload `json:"admin"`
	OrgInfo         *kolide.OrgInfo     `json:"org_info"`
	KolideServerURL *string             `json:"kolide_server_url,omitempty"`
	EnrollSecret    *string             `json:"osquery_enroll_secret,omitempty"`
}

type setupResponse struct {
	Admin           *kolide.User    `json:"admin,omitempty"`
	OrgInfo         *kolide.OrgInfo `json:"org_info,omitempty"`
	KolideServerURL *string         `json:"kolide_server_url"`
	EnrollSecret    *string         `json:"osquery_enroll_secret"`
	Token           *string         `json:"token,omitempty"`
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
		if req.OrgInfo != nil {
			configPayload.OrgInfo = req.OrgInfo
		}
		configPayload.ServerSettings = &kolide.ServerSettings{}
		if req.KolideServerURL != nil {
			configPayload.ServerSettings.KolideServerURL = req.KolideServerURL
		}
		config, err = svc.NewAppConfig(ctx, configPayload)
		if err != nil {
			return setupResponse{Err: err}, nil
		}

		if req.Admin == nil {
			return setupResponse{Err: errors.New("setup request must provide admin")}, nil
		}

		// creating the user should be the last action. If there's a user
		// present and other errors occur, the setup endpoint closes.
		adminPayload := *req.Admin
		if adminPayload.Email == nil || *adminPayload.Email == "" {
			err := errors.Errorf("admin email cannot be empty")
			return setupResponse{Err: err}, nil
		}
		if adminPayload.Password == nil || *adminPayload.Password == "" {
			err := errors.Errorf("admin password cannot be empty")
			return setupResponse{Err: err}, nil
		}
		// Make the user an admin
		adminStr := "admin"
		adminPayload.GlobalRole = &adminStr
		admin, err = svc.CreateUser(ctx, adminPayload)
		if err != nil {
			return setupResponse{Err: err}, nil
		}

		// If everything works to this point, log the user in and return token.  If
		// the login fails for some reason, ignore the error and don't return
		// a token, forcing the user to log in manually
		token := new(string)
		_, *token, err = svc.Login(ctx, *req.Admin.Username, *req.Admin.Password)
		if err != nil {
			token = nil
		}
		return setupResponse{
			Admin: admin,
			OrgInfo: &kolide.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			KolideServerURL: &config.KolideServerURL,
			Token:           token,
		}, nil
	}
}
