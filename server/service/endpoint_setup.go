package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/endpoint"
	"github.com/pkg/errors"
)

type setupRequest struct {
	Admin        *fleet.UserPayload `json:"admin"`
	OrgInfo      *fleet.OrgInfo     `json:"org_info"`
	ServerURL    *string            `json:"server_url,omitempty"`
	EnrollSecret *string            `json:"osquery_enroll_secret,omitempty"`
}

type setupResponse struct {
	Admin        *fleet.User    `json:"admin,omitempty"`
	OrgInfo      *fleet.OrgInfo `json:"org_info,omitempty"`
	ServerURL    *string        `json:"server_url"`
	EnrollSecret *string        `json:"osquery_enroll_secret"`
	Token        *string        `json:"token,omitempty"`
	Err          error          `json:"error,omitempty"`
}

func (r setupResponse) error() error { return r.Err }

func makeSetupEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			admin         *fleet.User
			config        *fleet.AppConfig
			configPayload fleet.AppConfigPayload
			err           error
		)
		req := request.(setupRequest)
		if req.OrgInfo != nil {
			configPayload.OrgInfo = req.OrgInfo
		}
		configPayload.ServerSettings = &fleet.ServerSettings{}
		if req.ServerURL != nil {
			configPayload.ServerSettings.ServerURL = req.ServerURL
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
		adminPayload.GlobalRole = ptr.String(fleet.RoleAdmin)
		admin, err = svc.CreateInitialUser(ctx, adminPayload)
		if err != nil {
			return setupResponse{Err: err}, nil
		}

		// If everything works to this point, log the user in and return token.  If
		// the login fails for some reason, ignore the error and don't return
		// a token, forcing the user to log in manually
		token := new(string)
		_, *token, err = svc.Login(ctx, *req.Admin.Email, *req.Admin.Password)
		if err != nil {
			token = nil
		}
		return setupResponse{
			Admin: admin,
			OrgInfo: &fleet.OrgInfo{
				OrgName:    &config.OrgName,
				OrgLogoURL: &config.OrgLogoURL,
			},
			ServerURL: &config.ServerURL,
			Token:     token,
		}, nil
	}
}
